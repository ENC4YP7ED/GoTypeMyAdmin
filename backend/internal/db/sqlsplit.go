package db

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// SplitStatements splits a SQL script into individual statements on `;`,
// respecting single/double-quoted strings, backtick identifiers, and `--`,
// `#` and `/* */` comments. It does not handle DELIMITER changes (stored
// routines) — those remain a single-statement concern.
func SplitStatements(script string) []string {
	var statements []string
	var cur strings.Builder
	runes := []rune(script)
	n := len(runes)

	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			statements = append(statements, s)
		}
		cur.Reset()
	}

	for i := 0; i < n; i++ {
		c := runes[i]
		switch c {
		case '\'', '"', '`':
			// Consume the quoted span verbatim.
			quote := c
			cur.WriteRune(c)
			for i++; i < n; i++ {
				cur.WriteRune(runes[i])
				if runes[i] == '\\' && quote != '`' && i+1 < n {
					i++
					cur.WriteRune(runes[i])
					continue
				}
				if runes[i] == quote {
					break
				}
			}
		case '-':
			if i+1 < n && runes[i+1] == '-' {
				for i < n && runes[i] != '\n' {
					i++
				}
				cur.WriteRune('\n')
			} else {
				cur.WriteRune(c)
			}
		case '#':
			for i < n && runes[i] != '\n' {
				i++
			}
			cur.WriteRune('\n')
		case '/':
			if i+1 < n && runes[i+1] == '*' {
				i += 2
				for i+1 < n && !(runes[i] == '*' && runes[i+1] == '/') {
					i++
				}
				i++ // skip the closing '/'
			} else {
				cur.WriteRune(c)
			}
		case ';':
			flush()
		default:
			cur.WriteRune(c)
		}
	}
	flush()
	return statements
}

// ImportResult summarises a script run.
type ImportResult struct {
	Statements int     `json:"statements"`
	Executed   int     `json:"executed"`
	Affected   int64   `json:"affected"`
	DurationMS float64 `json:"durationMs"`
	FailedAt   int     `json:"failedAt"`   // 1-based index of the failing statement, 0 if none
	Error      string  `json:"error"`
}

// RunScript executes each statement of a script sequentially, stopping at the
// first error and reporting where it failed.
func RunScript(ctx context.Context, db *sql.DB, schema, script string) (*ImportResult, error) {
	stmts := SplitStatements(script)
	res := &ImportResult{Statements: len(stmts)}
	start := time.Now()

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if schema != "" {
		if _, err := conn.ExecContext(ctx, "USE "+QuoteIdent(schema)); err != nil {
			return nil, err
		}
	}

	for i, stmt := range stmts {
		r, err := conn.ExecContext(ctx, stmt)
		if err != nil {
			res.FailedAt = i + 1
			res.Error = err.Error()
			res.DurationMS = float64(time.Since(start).Microseconds()) / 1000.0
			return res, nil
		}
		if n, e := r.RowsAffected(); e == nil {
			res.Affected += n
		}
		res.Executed++
	}
	res.DurationMS = float64(time.Since(start).Microseconds()) / 1000.0
	return res, nil
}
