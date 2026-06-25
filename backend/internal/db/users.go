package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// User is one MySQL account plus a coarse privilege summary.
type User struct {
	User      string `json:"user"`
	Host      string `json:"host"`
	SuperUser bool   `json:"superUser"`
	Locked    bool   `json:"locked"`
}

// Users lists accounts from mysql.user.
func Users(ctx context.Context, db *sql.DB) ([]User, error) {
	// Column availability differs across MySQL/MariaDB versions; select the
	// portable subset and probe Super_priv which both expose.
	rows, err := db.QueryContext(ctx, "SELECT User, Host, Super_priv FROM mysql.user ORDER BY User, Host")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		var super string
		if err := rows.Scan(&u.User, &u.Host, &super); err != nil {
			return nil, err
		}
		u.SuperUser = strings.EqualFold(super, "Y")
		out = append(out, u)
	}
	return out, rows.Err()
}

// Grants returns the SHOW GRANTS output for an account.
func Grants(ctx context.Context, db *sql.DB, user, host string) ([]string, error) {
	q := fmt.Sprintf("SHOW GRANTS FOR %s@%s", quoteString(user), quoteString(host))
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grants []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		grants = append(grants, g)
	}
	return grants, rows.Err()
}

// CreateUser creates an account, optionally with a password.
func CreateUser(ctx context.Context, db *sql.DB, user, host, password string) error {
	q := fmt.Sprintf("CREATE USER %s@%s", quoteString(user), quoteString(host))
	if password != "" {
		q += " IDENTIFIED BY " + quoteString(password)
	}
	_, err := db.ExecContext(ctx, q)
	return err
}

// DropUser removes an account.
func DropUser(ctx context.Context, db *sql.DB, user, host string) error {
	q := fmt.Sprintf("DROP USER %s@%s", quoteString(user), quoteString(host))
	_, err := db.ExecContext(ctx, q)
	return err
}

// GrantPrivileges grants a privilege set on a scope (e.g. "*.*" or "`db`.*").
func GrantPrivileges(ctx context.Context, db *sql.DB, user, host, privileges, scope string) error {
	if privileges == "" {
		privileges = "ALL PRIVILEGES"
	}
	if scope == "" {
		scope = "*.*"
	}
	q := fmt.Sprintf("GRANT %s ON %s TO %s@%s", privileges, scope, quoteString(user), quoteString(host))
	if _, err := db.ExecContext(ctx, q); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, "FLUSH PRIVILEGES")
	return err
}

// quoteString single-quotes and escapes a string literal (for identifiers that
// are values, like user/host names, which cannot be parameterized in DDL).
func quoteString(s string) string {
	return "'" + strings.ReplaceAll(strings.ReplaceAll(s, "\\", "\\\\"), "'", "''") + "'"
}
