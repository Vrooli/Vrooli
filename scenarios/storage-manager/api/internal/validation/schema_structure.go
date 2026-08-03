package validation

import (
	"context"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// This file holds the Tier-1 schema-structure analyzers: the static checks that
// keep a scenario's embedded SQL schema idempotent, per-domain, and free of the
// shapes that crash boot or rot deletability (ALTER-in-schema, a centralized
// schema home, a non-empty system home, cross-domain hard FKs, unwired
// EnsureSchemas). Every analyzer here is Go-only and operates purely on the
// .sql/.go text the shared scan helpers collect — no SQL-parser dependency.
//
// All unexported helpers/types in this file are prefixed with `schema` so the
// tier composes cleanly with the isolation/hygiene tiers developed in parallel
// in this same package.

func init() {
	register(&schemaHasAlter{})
	register(&schemaNotIdempotent{})
	register(&schemaCentralized{})
	register(&schemaNotPerDomain{})
	register(&schemaEnsureNotWired{})
	register(&schemaSystemNotEmpty{})
	register(&schemaCrossDomainFK{})
}

// schemaCommentRE matches a `--` line comment to its end of line. SQL string
// literals never legitimately span a `--`-to-EOL inside our schema files, so a
// blunt strip is safe and keeps the scanner dependency-free.
var schemaCommentRE = regexp.MustCompile(`--[^\n]*`)

// schemaStripComments removes `--` line comments and lowercases for
// case-insensitive matching, preserving newlines so per-statement reasoning and
// crude line attribution still work.
func schemaStripComments(sql string) string {
	return schemaCommentRE.ReplaceAllString(sql, "")
}

// schemaStatements splits comment-stripped SQL into individual statements on
// `;`. Blank statements are dropped. Good enough for the structural checks here
// (we never need full expression parsing, only statement-leading keywords).
func schemaStatements(sql string) []string {
	stripped := schemaStripComments(sql)
	var out []string
	for _, raw := range strings.Split(stripped, ";") {
		s := strings.TrimSpace(raw)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// schemaLineOf returns the 1-based line number of the first match of needle
// (case-insensitive) in the original (un-stripped) text, or 0 when absent — for
// a `Location` of the form `file.sql:NN`.
func schemaLineOf(original, needle string) int {
	lower := strings.ToLower(original)
	idx := strings.Index(lower, strings.ToLower(needle))
	if idx < 0 {
		return 0
	}
	return strings.Count(original[:idx], "\n") + 1
}

// schemaLoc renders a finding location, appending :line when line > 0.
func schemaLoc(relPath string, line int) string {
	if line > 0 {
		return relPath + ":" + strconv.Itoa(line)
	}
	return relPath
}

// schemaCreateTableRE / schemaCreateIndexRE detect a CREATE TABLE / CREATE INDEX
// statement leader, capturing the (optional) IF NOT EXISTS clause so the
// idempotency analyzer can tell a guarded create from a bare one. `(?is)` =
// case-insensitive, dot-matches-newline (so multi-line headers match).
var (
	schemaCreateTableRE = regexp.MustCompile(`(?is)^\s*create\s+(temp(orary)?\s+)?table\s+(if\s+not\s+exists\s+)?`)
	schemaCreateIndexRE = regexp.MustCompile(`(?is)^\s*create\s+(unique\s+)?index\s+(if\s+not\s+exists\s+)?`)
	// schemaAlterRE matches an ALTER (TABLE/INDEX/etc.) statement leader.
	schemaAlterRE = regexp.MustCompile(`(?is)^\s*alter\s+`)
	// schemaCreateTableNameRE captures the table name from a CREATE TABLE header,
	// tolerating IF NOT EXISTS and an optional schema/quote wrapper.
	schemaCreateTableNameRE = regexp.MustCompile(`(?is)^\s*create\s+(?:temp(?:orary)?\s+)?table\s+(?:if\s+not\s+exists\s+)?["` + "`" + `\[]?([a-zA-Z0-9_."]+)["` + "`" + `\]]?`)
	// schemaReferencesRE captures the referenced table name from a REFERENCES
	// clause (inline FK or FOREIGN KEY ... REFERENCES).
	schemaReferencesRE = regexp.MustCompile(`(?is)references\s+["` + "`" + `\[]?([a-zA-Z0-9_."]+)["` + "`" + `\]]?`)
)

// schemaUnquote strips surrounding quotes/brackets and any schema-qualifier
// prefix from an identifier, lowercasing for comparison.
func schemaUnquote(ident string) string {
	ident = strings.Trim(ident, "\"`[]")
	if i := strings.LastIndex(ident, "."); i >= 0 {
		ident = ident[i+1:]
	}
	return strings.ToLower(strings.Trim(ident, "\"`[]"))
}

// -----------------------------------------------------------------------------
// 1. SCHEMA_HAS_ALTER
// -----------------------------------------------------------------------------

// schemaHasAlter flags any ALTER statement living in an embedded schema .sql
// file. ALTER belongs in a migration; in schema.sql it either crashes boot
// (SQLite execs the file as one blob) or silently no-ops.
type schemaHasAlter struct{}

func (schemaHasAlter) Name() string { return "schema-has-alter" }

func (schemaHasAlter) Applies(ac AnalyzerContext) bool { return ac.IsGo() }

func (a schemaHasAlter) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var findings []Finding
	for _, f := range CollectSQLFiles(ac) {
		text := ReadFile(f.AbsPath)
		for _, stmt := range schemaStatements(text) {
			if schemaAlterRE.MatchString(stmt) {
				findings = append(findings, Finding{
					Code:     "SCHEMA_HAS_ALTER",
					Severity: SeverityError,
					Title:    "ALTER statement in embedded schema",
					Message: "An ALTER statement was found in the embedded schema file " + f.RelPath + ". " +
						"EnsureSchemas execs schema.sql as a single blob on every boot with no per-statement tolerance: " +
						"`ALTER TABLE … ADD COLUMN` is a syntax error in SQLite (our default engine) and a bare ADD COLUMN " +
						"errors with \"duplicate column name\" on the second boot — either way it crashes startup. Any change " +
						"to an existing table's columns is a migration, not a declarative edit.",
					Location:    schemaLoc(f.RelPath, schemaLineOf(text, "alter")),
					Remediation: "Move the column change to a versioned migration (migrations/NNNN_*.sql); keep schema.sql to idempotent CREATE TABLE/INDEX IF NOT EXISTS only.",
					Analyzer:    a.Name(),
				})
				break // one finding per file is enough to direct the fix
			}
		}
	}
	return findings, nil
}

// -----------------------------------------------------------------------------
// 2. SCHEMA_NOT_IDEMPOTENT
// -----------------------------------------------------------------------------

// schemaNotIdempotent flags a CREATE TABLE or CREATE INDEX that lacks IF NOT
// EXISTS in an embedded schema file. EnsureSchemas re-runs every boot; a bare
// CREATE errors "table already exists" on the second start.
type schemaNotIdempotent struct{}

func (schemaNotIdempotent) Name() string { return "schema-not-idempotent" }

func (schemaNotIdempotent) Applies(ac AnalyzerContext) bool { return ac.IsGo() }

func (a schemaNotIdempotent) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var findings []Finding
	for _, f := range CollectSQLFiles(ac) {
		text := ReadFile(f.AbsPath)
		for _, stmt := range schemaStatements(text) {
			var kind string
			switch {
			case schemaCreateTableRE.MatchString(stmt):
				kind = "table"
			case schemaCreateIndexRE.MatchString(stmt):
				kind = "index"
			default:
				continue
			}
			if schemaStmtHasIfNotExists(stmt) {
				continue
			}
			findings = append(findings, Finding{
				Code:     "SCHEMA_NOT_IDEMPOTENT",
				Severity: SeverityError,
				Title:    "Non-idempotent CREATE in embedded schema",
				Message: "A CREATE " + strings.ToUpper(kind) + " without IF NOT EXISTS was found in " + f.RelPath + ". " +
					"EnsureSchemas re-runs schema.sql on every boot, so a bare CREATE fails with \"already exists\" on the " +
					"second start and crashes the API. Embedded schema must be forward-only and idempotent.",
				Location:    schemaLoc(f.RelPath, schemaStmtLine(text, stmt)),
				Remediation: "Add IF NOT EXISTS to the CREATE " + kind + " (CREATE " + strings.ToUpper(kind) + " IF NOT EXISTS …).",
				Analyzer:    a.Name(),
			})
		}
	}
	return findings, nil
}

// schemaStmtHasIfNotExists reports whether a CREATE statement carries an
// IF NOT EXISTS guard (case/space-insensitive).
func schemaStmtHasIfNotExists(stmt string) bool {
	return regexp.MustCompile(`(?is)if\s+not\s+exists`).MatchString(stmt)
}

// schemaStmtLine finds the 1-based line of a statement's leading token in the
// original text, for a precise location. Falls back to the file when not found.
func schemaStmtLine(original, stmt string) int {
	// Use the first ~24 chars of the (trimmed) statement as a stable anchor.
	anchor := stmt
	if len(anchor) > 24 {
		anchor = anchor[:24]
	}
	anchor = strings.Join(strings.Fields(anchor), " ")
	// Try the leading keyword as a cheap, reliable anchor.
	if fields := strings.Fields(stmt); len(fields) > 0 {
		return schemaLineOf(original, fields[0])
	}
	return schemaLineOf(original, anchor)
}

// -----------------------------------------------------------------------------
// 3. SCHEMA_CENTRALIZED
// -----------------------------------------------------------------------------

// schemaCentralized flags a project-level/central schema file that holds domain
// tables instead of per-domain internal/<domain>/schema.sql files. A scenario-
// root schema.sql or api/schema.sql, or any single .sql outside a domain dir
// that declares multiple domains' tables, is a deletability bug.
type schemaCentralized struct{}

func (schemaCentralized) Name() string { return "schema-centralized" }

func (schemaCentralized) Applies(ac AnalyzerContext) bool {
	return ac.IsGo() && ac.HasRelationalStore()
}

func (a schemaCentralized) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var findings []Finding
	for _, f := range CollectSQLFiles(ac) {
		if f.IsSystem {
			continue // the system home is the sanctioned cross-cutting file
		}
		if f.Domain != "" {
			continue // a per-domain schema.sql is exactly right
		}
		text := ReadFile(f.AbsPath)
		tables := schemaCreatedTables(text)
		if len(tables) == 0 {
			continue // a non-domain .sql with no tables isn't a centralized home
		}
		// Anything here is a non-system, non-domain .sql under api/ that declares
		// tables: a scenario-root/api-root schema.sql or a loose .sql that owns
		// tables but lives nowhere a domain can own. That is a centralized home.
		findings = append(findings, Finding{
			Code:     "SCHEMA_CENTRALIZED",
			Severity: SeverityError,
			Title:    "Centralized schema file holds domain tables",
			Message: "The file " + f.RelPath + " is a project-level/central schema home declaring " + strconv.Itoa(len(tables)) +
				" table(s) (" + strings.Join(tables, ", ") + ") that belong to individual domains. A domain whose tables " +
				"live in a central schema can never be deleted by removing its folder — the definition stays, becomes " +
				"orphaned, and is recreated on every boot. This is shotgun surgery for adds and a deletability bug for removes.",
			Location:    f.RelPath,
			Remediation: "Split the file into per-domain internal/<domain>/schema.sql files (each embedded via //go:embed and registered in modules.AllSchemas()); keep only cross-cutting infra in internal/database/system.sql.",
			Analyzer:    a.Name(),
		})
	}
	return findings, nil
}

// schemaCreatedTables returns the lowercased names of every table a schema text
// CREATEs, in declaration order (deduplicated).
func schemaCreatedTables(text string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, stmt := range schemaStatements(text) {
		m := schemaCreateTableNameRE.FindStringSubmatch(stmt)
		if m == nil {
			continue
		}
		name := schemaUnquote(m[1])
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

// -----------------------------------------------------------------------------
// 4. SCHEMA_NOT_PER_DOMAIN
// -----------------------------------------------------------------------------

// schemaNotPerDomain flags schema that is present but not organized per-domain,
// only when the scenario has >= 2 domains (the rule is meaningless otherwise).
// Either a non-domain file owns tables, or no domain owns any schema while the
// scenario clearly partitions into multiple domains.
type schemaNotPerDomain struct{}

func (schemaNotPerDomain) Name() string { return "schema-not-per-domain" }

func (schemaNotPerDomain) Applies(ac AnalyzerContext) bool {
	return ac.IsGo() && ac.HasRelationalStore() && len(ac.Domains) >= 2
}

func (a schemaNotPerDomain) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	files := CollectSQLFiles(ac)

	var domainSchemaFiles, looseTableFiles int
	for _, f := range files {
		text := ReadFile(f.AbsPath)
		tables := schemaCreatedTables(text)
		if len(tables) == 0 {
			continue
		}
		if f.Domain != "" {
			domainSchemaFiles++
			continue
		}
		if f.IsSystem {
			continue // system home is allowed to hold cross-cutting views/types
		}
		looseTableFiles++
	}

	if looseTableFiles == 0 && domainSchemaFiles > 0 {
		return nil, nil // schema is organized per-domain — clean
	}
	if domainSchemaFiles == 0 && looseTableFiles == 0 {
		return nil, nil // no domain tables present anywhere — nothing to organize
	}

	return []Finding{{
		Code:     "SCHEMA_NOT_PER_DOMAIN",
		Severity: SeverityWarning,
		Title:    "Schema not organized per-domain",
		Message: "This scenario partitions into " + strconv.Itoa(len(ac.Domains)) + " domains (" + strings.Join(ac.Domains, ", ") +
			") but its schema is not organized per-domain: table definitions live outside internal/<domain>/schema.sql. " +
			"Per-domain schema is what makes a domain add a one-folder change and a domain delete an `rm -rf` instead of a " +
			"central-file archaeology dig.",
		Location:    "api/internal/",
		Remediation: "Move each table's CREATE into the internal/<domain>/schema.sql of the domain that owns it; embed each via //go:embed and register in modules.AllSchemas().",
		Analyzer:    a.Name(),
	}}, nil
}

// -----------------------------------------------------------------------------
// 5. ENSURE_SCHEMAS_NOT_WIRED
// -----------------------------------------------------------------------------

// schemaEnsureNotWired flags embedded schema that is present but never applied:
// either database.EnsureSchemas( is called nowhere in the api Go sources, or a
// domain schema.sql has no sibling //go:embed in a .go file (so its bytes never
// reach a SchemaProvider).
type schemaEnsureNotWired struct{}

func (schemaEnsureNotWired) Name() string { return "schema-ensure-not-wired" }

func (schemaEnsureNotWired) Applies(ac AnalyzerContext) bool {
	return ac.IsGo() && ac.HasRelationalStore()
}

func (a schemaEnsureNotWired) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	sqlFiles := CollectSQLFiles(ac)

	// Only domain/loose schema files carry tables we expect to be applied; an
	// empty system home does not require wiring.
	var schemaWithTables []SQLFile
	for _, f := range sqlFiles {
		if len(schemaCreatedTables(ReadFile(f.AbsPath))) > 0 {
			schemaWithTables = append(schemaWithTables, f)
		}
	}
	if len(schemaWithTables) == 0 {
		return nil, nil // no embedded tables → nothing to wire
	}

	goFiles := CollectGoFiles(ac)
	ensureCalled := false
	embedded := map[string]bool{} // dir(relPath) → some .go in that dir has //go:embed *.sql
	for _, g := range goFiles {
		src := ReadFile(g.AbsPath)
		if schemaApplicationCallPresent(src) {
			ensureCalled = true
		}
		if schemaGoEmbedsSQL(src) {
			embedded[filepath.ToSlash(filepath.Dir(g.RelPath))] = true
		}
	}

	var findings []Finding
	if !ensureCalled {
		findings = append(findings, Finding{
			Code:     "ENSURE_SCHEMAS_NOT_WIRED",
			Severity: SeverityWarning,
			Title:    "EnsureSchemas never called",
			Message: "Embedded schema files declare tables but database.EnsureSchemas( is not called anywhere in the api Go " +
				"sources, so the schema is never applied at boot — the tables won't exist at runtime.",
			Location:    "api/main.go",
			Remediation: "Call database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...) during API startup (after the DB handle is opened).",
			Analyzer:    a.Name(),
		})
	}
	for _, f := range schemaWithTables {
		if f.Domain == "" {
			continue // only domain schema.sql is expected to have a sibling embed
		}
		dir := filepath.ToSlash(filepath.Dir(f.RelPath))
		if embedded[dir] {
			continue
		}
		findings = append(findings, Finding{
			Code:     "ENSURE_SCHEMAS_NOT_WIRED",
			Severity: SeverityWarning,
			Title:    "Domain schema has no //go:embed",
			Message: "The domain schema file " + f.RelPath + " has no sibling //go:embed in a .go file in the same directory, " +
				"so its bytes never reach a SchemaProvider and EnsureSchemas cannot apply it.",
			Location:    f.RelPath,
			Remediation: "Add a schema.go in the same directory: //go:embed schema.sql\\nvar schemaSQL string; func Schema() string { return schemaSQL }, and register it in modules.AllSchemas().",
			Analyzer:    a.Name(),
		})
	}
	return findings, nil
}

// schemaApplicationCallPresent accepts the api-core EnsureSchemas seam and a
// deliberately narrow equivalent used by applications that own a richer
// schema bootstrap (for example, one that applies migrations and optional
// seed data in the same startup transaction). The equivalent must be an
// actual qualified ApplySchema call; merely defining a helper with that name
// is not evidence that startup invokes it.
func schemaApplicationCallPresent(src string) bool {
	if strings.Contains(src, "EnsureSchemas(") {
		return true
	}
	return strings.Contains(src, ".ApplySchema(")
}

// schemaGoEmbedsSQL reports whether a Go source contains a //go:embed directive
// targeting a .sql file.
func schemaGoEmbedsSQL(src string) bool {
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "//go:embed") && strings.Contains(t, ".sql") {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------------
// 6. SYSTEM_SQL_NOT_EMPTY
// -----------------------------------------------------------------------------

// schemaSystemNotEmpty flags a CREATE TABLE in internal/database/system.sql. The
// system home holds only cross-cutting infra (extensions, custom types, views);
// a domain table there means a domain that doesn't exist yet.
type schemaSystemNotEmpty struct{}

func (schemaSystemNotEmpty) Name() string { return "schema-system-not-empty" }

func (schemaSystemNotEmpty) Applies(ac AnalyzerContext) bool { return ac.IsGo() }

func (a schemaSystemNotEmpty) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var findings []Finding
	for _, f := range CollectSQLFiles(ac) {
		if !f.IsSystem {
			continue
		}
		text := ReadFile(f.AbsPath)
		tables := schemaCreatedTables(text)
		if len(tables) == 0 {
			continue
		}
		findings = append(findings, Finding{
			Code:     "SYSTEM_SQL_NOT_EMPTY",
			Severity: SeverityWarning,
			Title:    "Domain table in system schema home",
			Message: "The system schema home " + f.RelPath + " declares " + strconv.Itoa(len(tables)) + " table(s) (" +
				strings.Join(tables, ", ") + "). system.sql is reserved for cross-cutting infrastructure only — extensions, " +
				"custom types, cross-domain views. A domain table here is a tripwire that the table belongs to a domain that " +
				"doesn't exist yet.",
			Location:    schemaLoc(f.RelPath, schemaLineOf(text, "create table")),
			Remediation: "Re-home each table to the internal/<domain>/schema.sql of the domain that owns it; create the domain first if it doesn't exist.",
			Analyzer:    a.Name(),
		})
	}
	return findings, nil
}

// -----------------------------------------------------------------------------
// 7. CROSS_DOMAIN_HARD_FK
// -----------------------------------------------------------------------------

// schemaCrossDomainFK flags a REFERENCES to a table owned by a DIFFERENT domain.
// Cross-domain coupling should be a soft ID, not a hard FK that ties two
// bounded contexts' lifecycles together.
type schemaCrossDomainFK struct{}

func (schemaCrossDomainFK) Name() string { return "schema-cross-domain-fk" }

func (schemaCrossDomainFK) Applies(ac AnalyzerContext) bool {
	return ac.IsGo() && ac.HasRelationalStore()
}

func (a schemaCrossDomainFK) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	files := CollectSQLFiles(ac)

	// Map table → owning domain by which domain's schema.sql CREATEs it. Only
	// domain-owned files contribute ownership (system/loose files are ignored as
	// owners — a hard FK heuristic needs a clear domain home on both ends).
	owner := map[string]string{}
	for _, f := range files {
		if f.Domain == "" {
			continue
		}
		for _, tbl := range schemaCreatedTables(ReadFile(f.AbsPath)) {
			if _, exists := owner[tbl]; !exists {
				owner[tbl] = f.Domain
			}
		}
	}

	var findings []Finding
	for _, f := range files {
		if f.Domain == "" {
			continue
		}
		text := ReadFile(f.AbsPath)
		reported := map[string]struct{}{}
		for _, stmt := range schemaStatements(text) {
			if !strings.Contains(strings.ToLower(stmt), "references") {
				continue
			}
			for _, m := range schemaReferencesRE.FindAllStringSubmatch(stmt, -1) {
				ref := schemaUnquote(m[1])
				ownerDomain, known := owner[ref]
				if !known || ownerDomain == f.Domain {
					continue // same domain, or referenced table not domain-owned
				}
				key := ref + "→" + f.Domain
				if _, dup := reported[key]; dup {
					continue
				}
				reported[key] = struct{}{}
				findings = append(findings, Finding{
					Code:     "CROSS_DOMAIN_HARD_FK",
					Severity: SeverityWarning,
					Title:    "Hard FK across domain boundary",
					Message: "The schema for domain \"" + f.Domain + "\" (" + f.RelPath + ") declares a hard foreign key " +
						"REFERENCES " + ref + ", a table owned by domain \"" + ownerDomain + "\". A hard cross-domain FK ties " +
						"two bounded contexts' lifecycles together: you can no longer delete or migrate one domain without the " +
						"other, defeating per-domain ownership.",
					Location:    schemaLoc(f.RelPath, schemaLineOf(text, "references "+ref)),
					Remediation: "Replace the hard FK with a soft ID column (store the referenced id without a REFERENCES constraint); enforce referential integrity in the application layer if needed.",
					Analyzer:    a.Name(),
				})
			}
		}
	}
	return findings, nil
}
