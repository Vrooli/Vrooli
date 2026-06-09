package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// SchemaProvider is implemented by code units that own SQL they want
// applied to a scenario's database at boot. Domain packages embed
// schema.sql via go:embed and return its contents from a Schema()
// function; SchemaProviderFunc adapts that function to the interface
// so the slice in main.go can list providers without each domain
// declaring a wrapper type.
//
// The pattern is: each internal/<dom>/ ships schema.sql (forward-only
// declarative — IF NOT EXISTS / DO blocks for idempotency) and a
// Schema() function. Cross-cutting infrastructure (postgres extensions,
// custom types, cross-domain views) lives in a "system" home (e.g.,
// internal/database/system.sql) which exposes its own SystemSchema().
// EnsureSchemas applies them in registration order; empty schemas skip.
type SchemaProvider interface {
	Schema() string
}

// SchemaProviderFunc adapts a bare func() string to SchemaProvider so
// domain packages can stay free of declaration boilerplate.
type SchemaProviderFunc func() string

func (f SchemaProviderFunc) Schema() string { return f() }

// SchemaExecer is the minimal database surface EnsureSchemas needs.
// *sql.DB satisfies it directly; tests can supply a fake without
// pulling in a real driver.
type SchemaExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// SchemaQuerier is the optional read surface EnsureSchemas uses for its
// post-apply drift check (see EnsureSchemas). *sql.DB satisfies it; when
// db does not (e.g. a write-only test fake), the drift check is skipped and
// EnsureSchemas is apply-only, exactly as before.
type SchemaQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Compile-time guarantee that *sql.DB satisfies both surfaces.
var (
	_ SchemaExecer  = (*sql.DB)(nil)
	_ SchemaQuerier = (*sql.DB)(nil)
)

// EnsureSchemas applies each provider's schema to db in the order given.
// Schemas must be idempotent (CREATE TABLE IF NOT EXISTS, etc.) — each
// boot calls EnsureSchemas and a successful apply must be a no-op on
// the next call.
//
// Empty schemas (Schema() returns "") are skipped silently so a system
// home that has no cross-cutting bits doesn't need a placeholder
// statement.
//
// Errors include the provider's index (1-based) and the underlying
// error. Provider type names are not included because providers are
// often SchemaProviderFunc-wrapped functions whose type is uninformative.
//
// Drift check (SQLite only). After applying, if db also implements
// SchemaQuerier, EnsureSchemas compares each declared table's columns
// against the live table (via PRAGMA table_info) and returns an error if a
// pre-existing table is missing a declared column. This catches the silent
// trap where a new column is added to a CREATE TABLE IF NOT EXISTS block:
// the statement is a no-op on a DB that already has the table, so the column
// never lands and queries fail later at runtime instead of loudly at boot.
// The fix is a one-shot migration (storage-steer §5), not recreating the DB.
// The check is dialect-scoped by construction: PRAGMA table_info errors on
// non-SQLite engines (e.g. Postgres, where ADD COLUMN IF NOT EXISTS works and
// this trap does not exist), and that error is treated as "cannot verify" and
// skipped rather than surfaced. The parser is deliberately conservative —
// anything it cannot confidently read is skipped, so the check never blocks
// boot on a false positive.
func EnsureSchemas(ctx context.Context, db SchemaExecer, providers ...SchemaProvider) error {
	for i, p := range providers {
		sqlText := p.Schema()
		if sqlText == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, sqlText); err != nil {
			return fmt.Errorf("apply schema provider %d: %w", i+1, err)
		}
	}
	if q, ok := db.(SchemaQuerier); ok {
		return verifyDeclaredColumns(ctx, q, providers)
	}
	return nil
}

// verifyDeclaredColumns enforces the EnsureSchemas drift check. Returns nil
// (skips) for any engine where PRAGMA table_info is unavailable.
func verifyDeclaredColumns(ctx context.Context, q SchemaQuerier, providers []SchemaProvider) error {
	declared := map[string][]string{}
	for _, p := range providers {
		for tbl, cols := range DeclaredColumns(p.Schema()) {
			declared[tbl] = append(declared[tbl], cols...)
		}
	}
	if len(declared) == 0 {
		return nil
	}

	var drifts []string
	for tbl, cols := range declared {
		actual, err := sqliteTableColumns(ctx, q, tbl)
		if err != nil {
			// PRAGMA table_info unsupported (non-SQLite) or unreadable —
			// the silent-no-op trap is SQLite-specific, so we cannot and
			// need not verify here. Skip the whole check.
			return nil
		}
		if len(actual) == 0 {
			// Table absent (should not happen post-apply) — nothing to compare.
			continue
		}
		for _, c := range cols {
			if !actual[c] {
				drifts = append(drifts, tbl+"."+c)
			}
		}
	}
	if len(drifts) == 0 {
		return nil
	}
	sort.Strings(drifts)
	return fmt.Errorf("schema drift detected: pre-existing table(s) missing declared column(s) %v — "+
		"under SQLite, adding a column to a CREATE TABLE IF NOT EXISTS block is a silent no-op on a DB that "+
		"already has the table, so the column never lands. Apply a one-shot migration to bring the existing "+
		"DB to the declared shape (storage-steer §5: write /tmp/<scenario>/migrate-*.sql with the ALTER TABLE "+
		"... ADD COLUMN, run it with the scenario stopped, then delete it). Do not recreate the DB", drifts)
}

// sqliteTableColumns returns the set of column names on table via PRAGMA
// table_info. A query error (e.g. PRAGMA unsupported on a non-SQLite engine)
// is propagated so the caller can skip verification; a missing table yields an
// empty set with no error.
func sqliteTableColumns(ctx context.Context, q SchemaQuerier, table string) (map[string]bool, error) {
	rows, err := q.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := map[string]bool{}
	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cols, nil
}

var tableHeaderRe = regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(` +
	`[A-Za-z_][A-Za-z0-9_]*` + `|` + // bare identifier
	`"[^"]+"` + `|` + // "quoted"
	"`[^`]+`" + `|` + // `backtick`
	`\[[^\]]+\]` + // [bracketed]
	`)\s*\(`)

var constraintKeywords = map[string]bool{
	"PRIMARY":    true,
	"FOREIGN":    true,
	"UNIQUE":     true,
	"CHECK":      true,
	"CONSTRAINT": true,
	"KEY":        true,
	"EXCLUDE":    true,
}

// DeclaredColumns parses CREATE TABLE statements out of schema SQL and returns
// table name -> declared column names. It is deliberately conservative: any
// construct it cannot confidently parse is skipped rather than guessed, so a
// missing entry never causes a false-positive drift error. Table-level
// constraints (PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK, CONSTRAINT, ...) are
// not columns and are excluded. Identifier quoting (", `, []) is stripped so
// the names compare equal to PRAGMA table_info output.
func DeclaredColumns(schema string) map[string][]string {
	schema = stripSQLComments(schema)
	out := map[string][]string{}
	for _, loc := range tableHeaderRe.FindAllStringSubmatchIndex(schema, -1) {
		name := unquoteIdent(schema[loc[2]:loc[3]])
		open := loc[1] - 1 // the regex ends at the opening '('
		body, ok := extractParenBody(schema, open)
		if !ok {
			continue
		}
		if cols := parseColumnNames(body); len(cols) > 0 {
			out[name] = append(out[name], cols...)
		}
	}
	return out
}

// extractParenBody returns the text between the '(' at index open and its
// matching ')', tracking single-quoted string literals so a ')' inside a
// default value doesn't end the body early.
func extractParenBody(s string, open int) (string, bool) {
	depth := 0
	inStr := false
	for i := open; i < len(s); i++ {
		c := s[i]
		if inStr {
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' { // '' escape stays in string
					i++
					continue
				}
				inStr = false
			}
			continue
		}
		switch c {
		case '\'':
			inStr = true
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[open+1 : i], true
			}
		}
	}
	return "", false
}

// parseColumnNames splits a CREATE TABLE body at top-level commas and returns
// the leading identifier of each definition that is a column (not a
// table-level constraint).
func parseColumnNames(body string) []string {
	var cols []string
	for _, part := range splitTopLevel(body) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		first := firstToken(part)
		if first == "" || constraintKeywords[strings.ToUpper(unquoteIdent(first))] {
			continue
		}
		cols = append(cols, unquoteIdent(first))
	}
	return cols
}

// splitTopLevel splits s on commas that sit at paren depth 0 and outside
// single-quoted strings.
func splitTopLevel(s string) []string {
	var parts []string
	depth := 0
	inStr := false
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					i++
					continue
				}
				inStr = false
			}
			continue
		}
		switch c {
		case '\'':
			inStr = true
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// firstToken returns the leading token of a column/constraint definition,
// treating a quoted identifier ("x", `x`, [x]) as a single token.
func firstToken(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	switch s[0] {
	case '"':
		if j := strings.IndexByte(s[1:], '"'); j >= 0 {
			return s[:j+2]
		}
	case '`':
		if j := strings.IndexByte(s[1:], '`'); j >= 0 {
			return s[:j+2]
		}
	case '[':
		if j := strings.IndexByte(s[1:], ']'); j >= 0 {
			return s[:j+2]
		}
	}
	if i := strings.IndexAny(s, " \t\r\n("); i >= 0 {
		return s[:i]
	}
	return s
}

func unquoteIdent(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return s
	}
	switch {
	case s[0] == '"' && s[len(s)-1] == '"',
		s[0] == '`' && s[len(s)-1] == '`',
		s[0] == '[' && s[len(s)-1] == ']':
		return s[1 : len(s)-1]
	}
	return s
}

// stripSQLComments removes -- line comments and /* */ block comments so they
// can't confuse column parsing. String literals are not specially handled here
// because schema DDL does not embed comment markers inside strings in practice;
// the paren/comma walkers do track strings.
func stripSQLComments(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if i+1 < len(s) && s[i] == '-' && s[i+1] == '-' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			if i < len(s) {
				b.WriteByte(s[i])
			}
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			i++ // skip the '/' (loop's i++ skips the '*')
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
