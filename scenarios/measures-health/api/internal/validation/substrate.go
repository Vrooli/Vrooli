package validation

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// DetectedEntity is a persisted countable entity discovered from evidence in a
// scenario's source — independent of whether anyone declared a measure for it.
// It is the positive signal behind the anti-under-declaration cross-check: a
// detected entity that is neither covered, waived, nor a known stateless domain
// raises measures.undeclared-substrate.
type DetectedEntity struct {
	Name     string // normalized domain-style name (the table name, normalized)
	Evidence string // human-readable evidence (table + file) for the finding
}

// SubstrateDetector independently detects a scenario's persisted countable
// entities from evidence (NOT from the measure declarations). Production scans
// the scenario's SQL (FilesystemSubstrateDetector); tests inject fixtures.
// Keeping it a seam lets Classify stay pure. A nil SubstrateDetector skips the
// cross-check entirely.
type SubstrateDetector interface {
	DetectedEntities(scenario string) ([]DetectedEntity, error)
}

// substrateInfraTables are table names that, by strong convention, are storage
// substrate (event logs, migration bookkeeping) rather than first-party countable
// business entities — even when they carry a created_at column. They back other
// domains (e.g. a CQRS `events` table backs backlog/execution), so flagging them
// as their own undeclared entity is a false positive.
var substrateInfraTables = map[string]struct{}{
	"events":            {},
	"event_log":         {},
	"eventlog":          {},
	"event_store":       {},
	"outbox":            {},
	"schema_migrations": {},
	"migrations":        {},
	"goose_db_version":  {},
}

var (
	// createTableHeadRe matches the head of a CREATE TABLE statement up to the
	// opening paren, capturing the (optionally quoted) table name. The column
	// body is then scanned with balanced parens (createTableBody) so nested
	// CHECK(...)/DEFAULT(...) parens don't truncate it.
	createTableHeadRe = regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?["` + "`" + `\[]?([a-zA-Z_][a-zA-Z0-9_]*)["` + "`" + `\]]?\s*\(`)
	// createdAtColRe matches a created_at (or <prefix>_created_at) column name as
	// a whole word — the canonical "rows accumulate over time" signal. Mutation
	// stamps (updated_at, last_*_at, last_seen) deliberately do NOT qualify.
	createdAtColRe = regexp.MustCompile(`(?i)\bcreated_at\b`)
	// singletonRe matches a CHECK (id = 1) constraint — a singleton state row, not
	// a countable entity.
	singletonRe = regexp.MustCompile(`(?is)CHECK\s*\(\s*id\s*=\s*1\s*\)`)
)

// FilesystemSubstrateDetector scans a scenario's api/ tree for SQL CREATE TABLE
// statements (in .sql files and Go-embedded SQL) and surfaces tables that hold a
// countable accumulating entity: a non-singleton, non-infrastructure table with
// a created_at column.
type FilesystemSubstrateDetector struct {
	RepoRoot string
}

// DetectedEntities walks scenarios/<scenario>/api and returns one DetectedEntity
// per qualifying table, de-duplicated by normalized name and sorted.
func (d FilesystemSubstrateDetector) DetectedEntities(scenario string) ([]DetectedEntity, error) {
	root := filepath.Join(d.RepoRoot, "scenarios", scenario, "api")
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	byName := map[string]string{} // name -> evidence (first win)
	err := filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if e.IsDir() {
			name := e.Name()
			switch name {
			case "node_modules", "vendor", ".git", "dist", "build", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		name := e.Name()
		isSQL := strings.HasSuffix(name, ".sql")
		isGo := strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
		if !isSQL && !isGo {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		rel, _ := filepath.Rel(d.RepoRoot, path)
		for _, ent := range scanCreateTables(string(b)) {
			norm := normalizeDomain(ent.table)
			if norm == "" {
				continue
			}
			if _, infra := substrateInfraTables[norm]; infra {
				continue
			}
			if strings.HasPrefix(norm, "sqlite_") {
				continue
			}
			if !createdAtColRe.MatchString(ent.body) {
				continue
			}
			if singletonRe.MatchString(ent.body) {
				continue
			}
			if _, seen := byName[norm]; !seen {
				byName[norm] = "table `" + ent.table + "` (created_at) in " + rel
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]DetectedEntity, 0, len(byName))
	for n, ev := range byName {
		out = append(out, DetectedEntity{Name: n, Evidence: ev})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// singularize strips a single trailing "s" so a plural SQL table name (runs,
// chats) matches its singular domain (run, chat) in the cross-check. Deliberately
// minimal — conservative matching favours false-negatives over false-positives.
func singularize(s string) string {
	if len(s) > 1 && strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}

type rawTable struct {
	table string
	body  string // the column-definition body between the outermost parens
}

// scanCreateTables finds every CREATE TABLE statement in src and returns its
// table name + balanced-paren column body.
func scanCreateTables(src string) []rawTable {
	var out []rawTable
	locs := createTableHeadRe.FindAllStringSubmatchIndex(src, -1)
	for _, loc := range locs {
		table := src[loc[2]:loc[3]]
		// loc[1] is the index just past the opening "(".
		open := loc[1] - 1
		depth := 0
		end := -1
		for i := open; i < len(src); i++ {
			switch src[i] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					end = i
				}
			}
			if end >= 0 {
				break
			}
		}
		if end < 0 {
			continue
		}
		out = append(out, rawTable{table: table, body: src[open+1 : end]})
	}
	return out
}
