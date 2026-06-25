package db

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ExportFormat selects the serialization used by table exports.
type ExportFormat string

const (
	FormatSQL  ExportFormat = "sql"
	FormatCSV  ExportFormat = "csv"
	FormatJSON ExportFormat = "json"
)

// ExportTable serializes a whole table in the requested format. For SQL it
// emits the CREATE statement followed by INSERTs.
func ExportTable(ctx context.Context, db *sql.DB, schema, table string, format ExportFormat) (string, string, error) {
	rs, err := Exec(ctx, db, "", "SELECT * FROM "+QuoteIdent(schema)+"."+QuoteIdent(table))
	if err != nil {
		return "", "", err
	}

	switch format {
	case FormatCSV:
		return exportCSV(rs), "text/csv", nil
	case FormatJSON:
		return exportJSON(rs), "application/json", nil
	default:
		ddl, err := CreateTableSQL(ctx, db, schema, table)
		if err != nil {
			return "", "", err
		}
		return exportSQL(schema, table, ddl, rs), "application/sql", nil
	}
}

// ExportDatabase produces a SQL dump of every base table in a schema.
func ExportDatabase(ctx context.Context, db *sql.DB, schema string) (string, error) {
	tables, err := Tables(ctx, db, schema)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "-- GoTypeMyAdmin dump\n-- Database: %s\n-- Generated: %s\n\n",
		schema, time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "CREATE DATABASE IF NOT EXISTS %s;\nUSE %s;\n\n", QuoteIdent(schema), QuoteIdent(schema))

	for _, t := range tables {
		if t.Type == "VIEW" {
			continue
		}
		ddl, err := CreateTableSQL(ctx, db, schema, t.Name)
		if err != nil {
			return "", err
		}
		rs, err := Exec(ctx, db, "", "SELECT * FROM "+QuoteIdent(schema)+"."+QuoteIdent(t.Name))
		if err != nil {
			return "", err
		}
		b.WriteString(exportSQL(schema, t.Name, ddl, rs))
		b.WriteString("\n")
	}
	return b.String(), nil
}

func exportSQL(schema, table, ddl string, rs *ResultSet) string {
	var b strings.Builder
	fmt.Fprintf(&b, "-- Table: %s\n", table)
	fmt.Fprintf(&b, "DROP TABLE IF EXISTS %s;\n", QuoteIdent(table))
	b.WriteString(ddl)
	b.WriteString(";\n\n")

	if len(rs.Rows) == 0 {
		return b.String()
	}

	cols := make([]string, len(rs.Columns))
	for i, c := range rs.Columns {
		cols[i] = QuoteIdent(c)
	}
	fmt.Fprintf(&b, "INSERT INTO %s (%s) VALUES\n", QuoteIdent(table), strings.Join(cols, ", "))
	for i, row := range rs.Rows {
		b.WriteString("  (")
		for j, cell := range row {
			if j > 0 {
				b.WriteString(", ")
			}
			b.WriteString(sqlLiteral(cell))
		}
		if i == len(rs.Rows)-1 {
			b.WriteString(");\n")
		} else {
			b.WriteString("),\n")
		}
	}
	b.WriteString("\n")
	return b.String()
}

func exportCSV(rs *ResultSet) string {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(rs.Columns)
	for _, row := range rs.Rows {
		rec := make([]string, len(row))
		for i, cell := range row {
			if cell == nil {
				rec[i] = ""
			} else {
				rec[i] = *cell
			}
		}
		_ = w.Write(rec)
	}
	w.Flush()
	return buf.String()
}

func exportJSON(rs *ResultSet) string {
	out := make([]map[string]any, 0, len(rs.Rows))
	for _, row := range rs.Rows {
		obj := make(map[string]any, len(rs.Columns))
		for i, col := range rs.Columns {
			if row[i] == nil {
				obj[col] = nil
			} else {
				obj[col] = *row[i]
			}
		}
		out = append(out, obj)
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return string(b)
}

// sqlLiteral renders a cell as a SQL literal (NULL or single-quoted, escaped).
func sqlLiteral(cell *string) string {
	if cell == nil {
		return "NULL"
	}
	s := *cell
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "''")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return "'" + s + "'"
}
