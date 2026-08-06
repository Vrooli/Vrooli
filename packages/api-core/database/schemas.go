package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
	if err := ApplySchemas(ctx, db, providers...); err != nil {
		return err
	}
	return ReconcileDeclaredColumns(ctx, db, providers...)
}

// ApplySchemas is EnsureSchemas without the drift check.
//
// Use it when the caller owns forward-only migrations, which have to run
// BETWEEN the apply and the check: the schema declares the new column, the
// CREATE TABLE IF NOT EXISTS is a no-op on an existing DB, and the migration
// is what actually adds it. Running the check first would fail boot on the
// very drift the migration exists to repair — the check would be reporting a
// problem that the next few lines were about to fix.
//
// Pair it with VerifyDeclaredColumns once the migrations have run; a caller
// with no migrations should keep using EnsureSchemas, which does both.
func ApplySchemas(ctx context.Context, db SchemaExecer, providers ...SchemaProvider) error {
	for i, p := range providers {
		sqlText := p.Schema()
		if sqlText == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, sqlText); err != nil {
			return fmt.Errorf("apply schema provider %d: %w", i+1, err)
		}
	}
	return nil
}

// ReconcileDeclaredColumns brings a live database up to the declared shape and
// then verifies it, and is what EnsureSchemas uses.
//
// For every declared column missing from a pre-existing table it replays the
// column's own definition through `ALTER TABLE ... ADD COLUMN`. That is the
// exact repair the drift error used to ask an operator to perform by hand, and
// it is why adding a column to a `CREATE TABLE IF NOT EXISTS` block now simply
// works on a database that already has the table.
//
// Repair is strictly additive — it only ever ADDs a column the schema already
// declares. Nothing is dropped, renamed, retyped, or reordered, so it cannot
// destroy data. Columns SQLite refuses to add this way (see additiveAddRisk)
// are not attempted, an attempt that fails is not retried or worked around,
// and either case falls through to the same hard error as before with the
// reason attached. Every repair is logged.
//
// Use VerifyDeclaredColumns instead when a caller must not have its database
// written to.
func ReconcileDeclaredColumns(ctx context.Context, db SchemaExecer, providers ...SchemaProvider) error {
	return checkDeclaredColumns(ctx, db, providers, true)
}

// VerifyDeclaredColumns reports declared-column drift without touching the
// database. Read-only counterpart to ReconcileDeclaredColumns.
func VerifyDeclaredColumns(ctx context.Context, db SchemaExecer, providers ...SchemaProvider) error {
	return checkDeclaredColumns(ctx, db, providers, false)
}

// checkDeclaredColumns implements both entry points. Returns nil (skips) for
// any engine where PRAGMA table_xinfo is unavailable, and when db offers no
// read surface at all.
func checkDeclaredColumns(ctx context.Context, db SchemaExecer, providers []SchemaProvider, repair bool) error {
	q, ok := db.(SchemaQuerier)
	if !ok {
		return nil
	}

	declared := map[string][]ColumnDecl{}
	for _, p := range providers {
		for tbl, cols := range DeclaredColumnDefs(p.Schema()) {
			declared[tbl] = append(declared[tbl], cols...)
		}
	}
	if len(declared) == 0 {
		return nil
	}

	// Deterministic order so repairs and error text are stable across runs.
	tables := make([]string, 0, len(declared))
	for tbl := range declared {
		tables = append(tables, tbl)
	}
	sort.Strings(tables)

	var drifts []string
	for _, tbl := range tables {
		actual, err := sqliteTableColumns(ctx, q, tbl)
		if err != nil {
			// PRAGMA table_xinfo unsupported (non-SQLite) or unreadable —
			// the silent-no-op trap is SQLite-specific, so we cannot and
			// need not verify here. Skip the whole check.
			return nil
		}
		if len(actual) == 0 {
			// Table absent (should not happen post-apply) — nothing to compare.
			continue
		}
		for _, col := range declared[tbl] {
			if actual[col.Name] {
				continue
			}
			if !repair {
				drifts = append(drifts, tbl+"."+col.Name)
				continue
			}
			if risk := additiveAddRisk(col.Definition); risk != "" {
				drifts = append(drifts, fmt.Sprintf("%s.%s (not auto-addable: %s)", tbl, col.Name, risk))
				continue
			}
			stmt := fmt.Sprintf("ALTER TABLE %q ADD COLUMN %s", tbl, col.Definition)
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				drifts = append(drifts, fmt.Sprintf("%s.%s (ADD COLUMN failed: %v)", tbl, col.Name, err))
				continue
			}
			log.Printf("database: reconciled schema drift by adding %s.%s", tbl, col.Name)
		}
	}
	if len(drifts) == 0 {
		return nil
	}
	sort.Strings(drifts)
	return fmt.Errorf("schema drift detected: pre-existing table(s) missing declared column(s) %v — "+
		"under SQLite, adding a column to a CREATE TABLE IF NOT EXISTS block is a silent no-op on a DB that "+
		"already has the table, so the column never lands. Additive columns are repaired automatically; these "+
		"could not be. Apply a one-shot migration to bring the existing DB to the declared shape "+
		"(storage-steer §5: write /tmp/<scenario>/migrate-*.sql with the ALTER TABLE ... ADD COLUMN, run it "+
		"with the scenario stopped, then delete it). Do not recreate the DB", drifts)
}

// additiveAddRisk returns a non-empty reason when SQLite will not accept a
// column definition in ALTER TABLE ... ADD COLUMN, so the caller can report it
// precisely instead of issuing a statement that is known to fail.
//
// The rules are SQLite's (see "ALTER TABLE ADD COLUMN" in its docs):
// the column may not be PRIMARY KEY or UNIQUE; NOT NULL requires a default;
// a REFERENCES clause requires a NULL default; STORED generated columns are
// rejected (VIRTUAL ones are fine); and a non-constant default such as
// CURRENT_TIMESTAMP is not allowed.
//
// This is a pre-screen for a better message, not the safety boundary — a
// definition that slips through and is rejected by SQLite still lands in the
// same drift error. String literals are blanked first so a default value like
// 'PRIMARY KEY' cannot be mistaken for a constraint.
func additiveAddRisk(def string) string {
	upper := strings.ToUpper(blankStringLiterals(def))
	hasDefault := strings.Contains(upper, "DEFAULT")
	switch {
	case strings.Contains(upper, "PRIMARY KEY"):
		return "PRIMARY KEY columns cannot be added to an existing table"
	case strings.Contains(upper, "UNIQUE"):
		return "UNIQUE columns cannot be added to an existing table"
	case strings.Contains(upper, "REFERENCES"):
		return "foreign-key columns need a NULL default; add it by hand"
	case strings.Contains(upper, "STORED"):
		return "STORED generated columns cannot be added to an existing table"
	case strings.Contains(upper, "NOT NULL") && !hasDefault:
		return "NOT NULL without a DEFAULT leaves existing rows invalid"
	case strings.Contains(upper, "CURRENT_TIME"), strings.Contains(upper, "CURRENT_DATE"),
		strings.Contains(upper, "CURRENT_TIMESTAMP"):
		return "a non-constant DEFAULT is not allowed when adding a column"
	}
	return ""
}

// blankStringLiterals replaces the contents of single-quoted literals with
// spaces, preserving length and structure so keyword scanning cannot be fooled
// by a default value that happens to contain SQL keywords.
func blankStringLiterals(s string) string {
	out := []byte(s)
	inStr := false
	for i := 0; i < len(out); i++ {
		c := out[i]
		if inStr {
			if c == '\'' {
				if i+1 < len(out) && out[i+1] == '\'' { // '' escape stays in string
					out[i+1] = ' '
					i++
					continue
				}
				inStr = false
				continue
			}
			out[i] = ' '
			continue
		}
		if c == '\'' {
			inStr = true
		}
	}
	return string(out)
}

// sqliteTableColumns returns the set of column names on table via PRAGMA
// table_xinfo. A query error (e.g. PRAGMA unsupported on a non-SQLite engine)
// is propagated so the caller can skip verification; a missing table yields an
// empty set with no error.
//
// table_xinfo rather than table_info: table_info omits generated columns
// (VIRTUAL and STORED alike), so a schema declaring one would be reported as
// permanently drifted, and the error text would send the operator off to write
// an ADD COLUMN migration that SQLite rejects for generated columns. table_xinfo
// returns the same rows plus the hidden ones, so it is a strict superset and can
// only remove false positives.
func sqliteTableColumns(ctx context.Context, q SchemaQuerier, table string) (map[string]bool, error) {
	rows, err := q.QueryContext(ctx, fmt.Sprintf("PRAGMA table_xinfo(%q)", table))
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
			// hidden: 0 ordinary, 1 virtual table hidden, 2 VIRTUAL generated,
			// 3 STORED generated. All of them are columns the schema declared.
			hidden int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk, &hidden); err != nil {
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
	out := map[string][]string{}
	for tbl, decls := range DeclaredColumnDefs(schema) {
		names := make([]string, 0, len(decls))
		for _, d := range decls {
			names = append(names, d.Name)
		}
		out[tbl] = names
	}
	return out
}

// ColumnDecl is one column as the schema declares it: the bare name (compared
// against PRAGMA output) plus the definition text as written, which is what an
// additive repair replays into ALTER TABLE ... ADD COLUMN.
type ColumnDecl struct {
	Name       string
	Definition string
}

// DeclaredColumnDefs is DeclaredColumns with each column's definition text
// retained. Same conservative parsing: anything it cannot confidently read is
// skipped rather than guessed.
func DeclaredColumnDefs(schema string) map[string][]ColumnDecl {
	schema = stripSQLComments(schema)
	out := map[string][]ColumnDecl{}
	for _, loc := range tableHeaderRe.FindAllStringSubmatchIndex(schema, -1) {
		name := unquoteIdent(schema[loc[2]:loc[3]])
		open := loc[1] - 1 // the regex ends at the opening '('
		body, ok := extractParenBody(schema, open)
		if !ok {
			continue
		}
		if cols := parseColumnDecls(body); len(cols) > 0 {
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

// parseColumnDecls splits a CREATE TABLE body at top-level commas and returns
// one entry per definition that is a column (not a table-level constraint),
// carrying both the bare name and the definition text exactly as written.
//
// The definition is kept because SQLite accepts the same column-definition
// grammar in `ALTER TABLE ... ADD COLUMN` as in `CREATE TABLE`, so a declared
// column that is missing from a live table can usually be added verbatim
// rather than reported as unfixable drift. See reconcileDeclaredColumns.
func parseColumnDecls(body string) []ColumnDecl {
	var cols []ColumnDecl
	for _, part := range splitTopLevel(body) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		first := firstToken(part)
		if first == "" || constraintKeywords[strings.ToUpper(unquoteIdent(first))] {
			continue
		}
		cols = append(cols, ColumnDecl{Name: unquoteIdent(first), Definition: part})
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
