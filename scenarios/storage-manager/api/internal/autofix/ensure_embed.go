package autofix

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// previewEnsureEmbed remediates the unambiguous //go:embed sub-case of
// ENSURE_SCHEMAS_NOT_WIRED: a per-domain schema.sql (under api/internal/<domain>/)
// that declares tables but has no sibling Go file embedding it, so its bytes
// never reach a SchemaProvider. The fix scaffolds a schema.go in the same
// directory: a //go:embed of schema.sql plus a Schema() accessor.
//
// It deliberately does NOT attempt the other ENSURE_SCHEMAS_NOT_WIRED sub-case
// (database.EnsureSchemas never called anywhere): the correct insertion point in
// main.go and the modules.AllSchemas() registration are not mechanically
// determinable without risking a wrong edit, so that case is left to the
// analyzer to report and an author to wire.
func previewEnsureEmbed(root string) ([]Candidate, error) {
	var out []Candidate
	for _, dir := range domainSchemaDirsNeedingEmbed(root) {
		path := filepath.Join(dir.abs, "schema.go")
		out = append(out, Candidate{
			RuleID:      RuleEnsureSchemasUnwire,
			FilePath:    path,
			Description: "Scaffold schema.go embedding the domain schema.sql so EnsureSchemas can apply it.",
			Before:      "",
			After:       schemaGoScaffold(dir.pkg),
		})
	}
	return out, nil
}

// canFixEnsureEmbed reports whether any domain schema.sql under root still needs
// an embed sibling scaffolded.
func canFixEnsureEmbed(root, _ string) bool {
	return len(domainSchemaDirsNeedingEmbed(root)) > 0
}

// schemaDir pairs a domain schema directory's absolute path with its Go package
// name (the directory's base name, sanitized).
type schemaDir struct {
	abs string
	pkg string
}

// domainSchemaDirsNeedingEmbed returns each api/internal/<domain>/ directory that
// holds a schema.sql declaring at least one table but has no .go file embedding a
// .sql via //go:embed. The result is deterministic (sorted by path) and stays
// empty once every domain schema is embedded — the idempotency guarantee.
func domainSchemaDirsNeedingEmbed(root string) []schemaDir {
	apiDir := filepath.Join(root, "api")
	internalDir := filepath.Join(apiDir, "internal")
	info, err := os.Stat(internalDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	var out []schemaDir
	seen := map[string]bool{}
	_ = filepath.WalkDir(internalDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || filepath.Base(path) != "schema.sql" {
			return nil
		}
		dir := filepath.Dir(path)
		if seen[dir] {
			return nil
		}
		seen[dir] = true

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil || isExemptGoPath(filepath.ToSlash(rel)) {
			return nil
		}
		// Only per-domain schema.sql under api/internal/<domain>/ — skip the
		// cross-cutting system home (internal/database/system.sql).
		if domainOfSchemaPath(internalDir, path) == "" {
			return nil
		}
		if !sqlDeclaresTable(path) {
			return nil
		}
		if dirHasGoEmbedSQL(dir) {
			return nil
		}
		out = append(out, schemaDir{abs: dir, pkg: sanitizePkgName(filepath.Base(dir))})
		return nil
	})

	sortSchemaDirs(out)
	return out
}

// domainOfSchemaPath mirrors scan.go's domainOf: the first segment under
// api/internal/, excluding cross-cutting infra dirs (database, server).
func domainOfSchemaPath(internalDir, path string) string {
	rel, err := filepath.Rel(internalDir, path)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "..") {
		return ""
	}
	seg := strings.SplitN(rel, "/", 2)
	domain := seg[0]
	switch domain {
	case "database", "server":
		return ""
	}
	return domain
}

var createTableRE = regexp.MustCompile(`(?is)create\s+(temp(orary)?\s+)?table`)

// sqlDeclaresTable reports whether the SQL file CREATEs at least one table.
func sqlDeclaresTable(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return createTableRE.Match(data)
}

// dirHasGoEmbedSQL reports whether any .go file in dir contains a //go:embed
// directive targeting a .sql file (so the schema already reaches a provider).
func dirHasGoEmbedSQL(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "//go:embed") && strings.Contains(t, ".sql") {
				return true
			}
		}
	}
	return false
}

// schemaGoScaffold renders the schema.go a domain needs to embed its schema.sql.
func schemaGoScaffold(pkg string) string {
	return "package " + pkg + `

import _ "embed"

// schemaSQL is the embedded per-domain schema applied by EnsureSchemas at boot.
//
//go:embed schema.sql
var schemaSQL string

// Schema returns the embedded schema DDL for this domain. Register it in
// modules.AllSchemas() so database.EnsureSchemas applies it on startup.
func Schema() string { return schemaSQL }
`
}

// sanitizePkgName lowercases and strips non-identifier characters so a directory
// name maps to a valid Go package name.
func sanitizePkgName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "schema"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "d" + out
	}
	return out
}

func sortSchemaDirs(dirs []schemaDir) {
	for i := 1; i < len(dirs); i++ {
		for j := i; j > 0 && dirs[j].abs < dirs[j-1].abs; j-- {
			dirs[j], dirs[j-1] = dirs[j-1], dirs[j]
		}
	}
}
