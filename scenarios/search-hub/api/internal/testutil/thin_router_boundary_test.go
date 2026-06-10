package testutil_test

// OT-P0-006 architectural tests: "Thin-router boundary held."
//
// These tests assert two invariants that keep search-hub a pure federation
// router rather than a data-accreting monolith:
//
//  1. No qdrant dependency in the module graph — the router holds no vectors
//     and must never pull in a qdrant client, even transitively via go.mod.
//
//  2. No corpus-content tables in the SQLite schema — the router stores only
//     the provider registry + query telemetry (and the eval suite/run tables
//     that are also registry-and-telemetry in character). Corpus data lives in
//     provider scenarios; the router never copies it here.
//
// Implementation notes:
//   - The qdrant check reads go.sum (the complete resolved dependency graph,
//     including indirect deps) and fails if any line mentions "qdrant".
//     go.sum is the right target: go.mod can omit indirect deps that are still
//     pulled in, but go.sum always lists every resolved module version.
//   - The schema check reads every *.sql file under api/ and asserts that no
//     CREATE TABLE statement introduces a table name outside the known-good
//     allow-list. The allow-list is the source of truth for which tables the
//     router owns; adding a table triggers a deliberate, reviewed update here.

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// knownTables is the exhaustive allow-list of tables the thin router is
// permitted to own. Any CREATE TABLE in the SQLite schema files that names a
// table NOT on this list is a boundary violation — it may indicate corpus
// content being stored inside the router.
//
// To add a legitimate table: extend this list in the same PR that adds the
// schema, and explain in the commit message why the new table is still
// consistent with the thin-router invariant (registry/telemetry only).
var knownTables = map[string]bool{
	// Provider registry (internal/registry/schema.sql)
	"providers": true,
	// Query telemetry (internal/metrics/schema.sql)
	"query_telemetry":          true,
	"query_telemetry_provider": true,
	// Eval suite registry + run history (internal/eval/schema.sql)
	// These are telemetry/registry in character: eval_suites are provider
	// descriptors; eval_runs are immutable run records.
	"eval_suites": true,
	"eval_runs":   true,
}

// createTableRE matches: CREATE TABLE [IF NOT EXISTS] <name>
// capturing the table name in group 1. It is only applied to non-comment
// lines (lines not starting with "--") to avoid false positives from SQL
// comments like "-- Use CREATE TABLE IF NOT EXISTS so re-runs are no-ops."
var createTableRE = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)`)

// TestThinRouterBoundary_NoQdrantDependency asserts that the resolved module
// graph (go.sum) contains no entry mentioning "qdrant". A qdrant client
// import would mean the router is pulling vectors in-process, violating the
// "no qdrant import" invariant of OT-P0-006.
func TestThinRouterBoundary_NoQdrantDependency(t *testing.T) {
	// go.sum is two levels above internal/testutil/ → api/go.sum
	sumPath := filepath.Join("..", "..", "go.sum")
	f, err := os.Open(sumPath)
	if err != nil {
		t.Fatalf("open go.sum: %v", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	var violations []string
	for sc.Scan() {
		line := sc.Text()
		if strings.Contains(strings.ToLower(line), "qdrant") {
			violations = append(violations, line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan go.sum: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("OT-P0-006 violated: go.sum contains qdrant dependency lines (router must hold no vectors):")
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}

// TestThinRouterBoundary_NoCorpusContentTables asserts that every SQL file
// under the api/ tree creates only tables in the knownTables allow-list.
// Corpus-content tables (embeddings, document chunks, raw text, etc.) must
// never appear here — they belong in the provider scenarios.
func TestThinRouterBoundary_NoCorpusContentTables(t *testing.T) {
	// Walk from api/ root (two levels above internal/testutil/)
	root := filepath.Join("..", "..")
	var sqlFiles []string
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".sql") {
			sqlFiles = append(sqlFiles, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	if len(sqlFiles) == 0 {
		t.Fatal("no .sql files found under api/ — expected at least the registry/metrics/eval schemas")
	}

	var violations []string
	for _, sqlFile := range sqlFiles {
		data, err := os.ReadFile(sqlFile)
		if err != nil {
			t.Errorf("read %s: %v", sqlFile, err)
			continue
		}
		// Strip SQL line comments before scanning so patterns like
		// "-- Use CREATE TABLE IF NOT EXISTS so re-runs …" in documentation
		// comments do not trigger false positives.
		var nonCommentLines []string
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "--") {
				continue
			}
			nonCommentLines = append(nonCommentLines, line)
		}
		stripped := strings.Join(nonCommentLines, "\n")
		matches := createTableRE.FindAllStringSubmatch(stripped, -1)
		for _, m := range matches {
			tableName := strings.ToLower(m[1])
			if !knownTables[tableName] {
				rel, _ := filepath.Rel(root, sqlFile)
				violations = append(violations, rel+": CREATE TABLE "+tableName+" (not in allow-list)")
			}
		}
	}

	if len(violations) > 0 {
		t.Errorf("OT-P0-006 violated: SQL schema declares table(s) outside the thin-router allow-list.")
		t.Errorf("The allow-list (knownTables) admits only registry + telemetry tables.")
		t.Errorf("Corpus-content tables must live in provider scenarios, not the router.")
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}
