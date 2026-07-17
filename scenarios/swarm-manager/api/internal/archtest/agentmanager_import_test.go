package archtest

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// agentManagerImportPath is the Agent Manager client package. Direct use of it
// couples a package to the agent substrate; the declarative-operations
// architecture funnels all LAUNCH through the operation runner, so only the
// packages below may import the client at all. The spawn-boundary AST test
// (spawn_boundary_test.go) enforces the no-launch rule call-by-call; this test
// enforces the coarser import boundary so a NEW package cannot quietly acquire
// direct agent-manager coupling without an explicit, reviewed allowlist edit.
const agentManagerImportPath = "swarm-manager/internal/agentmanager"

// allowedAgentManagerImporters is the closed allowlist of internal packages
// (dir names under internal/) that may import the client, each with the reason
// it is infrastructure rather than a target-bound domain consumer. The test
// fails when a package outside this list imports the client (a boundary
// regression) AND when a listed package no longer imports it (a stale entry
// that must be deleted so the allowlist only ever shrinks or is consciously
// re-expanded).
var allowedAgentManagerImporters = map[string]string{
	"agentactivity":    "the spawn chokepoint: the ONLY launch path (operation runner -> engine -> here)",
	"agentsessions":    "the interactive human-in-the-loop session boundary",
	"agentmanager":     "the client package itself",
	"backlog":          "availability signaling (agentmanager.ErrNotAvailable) on the operation reroute paths",
	"captures":         "classify spawn kept by recorded class-(d) scope decision (AGENT-CUTOVER-LEDGER closeout)",
	"evidence":         "agent-manager-backed evidence collection (run transcripts as evidence sources)",
	"execution":        "typed workflow command/result adapter plus legacy run management; domain code never creates or continues runs",
	"initiativereview": "run management for review rounds (RunInspector wiring) — never launch",
	"operatingmode":    "the engine: builds spawn requests, launched via the agentactivity chokepoint",
	"review":           "run management for gathering rounds (RunInspector wiring) — never launch",
	"testutil":         "shared test fakes for the client seam",
}

// TestAgentManagerImportBoundary walks every package under internal/ and
// asserts the import allowlist above. Red-proof:
// TestAgentManagerImportDetectorFiresOnViolation runs the same detection
// primitive against a synthetic importer.
// [REQ:REQ-P0-009-OPERATION-SPAWN-BOUNDARY]
func TestAgentManagerImportBoundary(t *testing.T) {
	internalRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	importers := map[string][]string{} // package dir name -> files importing the client
	walkErr := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		imports, parseErr := fileImports(path)
		if parseErr != nil {
			return parseErr
		}
		if imports[agentManagerImportPath] {
			pkg := filepath.Base(filepath.Dir(path))
			importers[pkg] = append(importers[pkg], filepath.Base(path))
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk internal/: %v", walkErr)
	}

	var violations, stale []string
	for pkg, files := range importers {
		if _, ok := allowedAgentManagerImporters[pkg]; !ok {
			sort.Strings(files)
			violations = append(violations, pkg+" ("+strings.Join(files, ", ")+")")
		}
	}
	for pkg, reason := range allowedAgentManagerImporters {
		if pkg == "testutil" || pkg == "agentmanager" {
			continue // testutil imports it only from shared fakes; the client is itself
		}
		if len(importers[pkg]) == 0 {
			stale = append(stale, pkg+" (no longer imports the client — delete this allowlist entry; was: "+reason+")")
		}
	}
	sort.Strings(violations)
	sort.Strings(stale)
	if len(violations) > 0 {
		t.Errorf("packages outside the allowlist import %s — work must be expressed as an operation (opsrunner), not by talking to Agent Manager directly:\n  %s",
			agentManagerImportPath, strings.Join(violations, "\n  "))
	}
	if len(stale) > 0 {
		t.Errorf("stale agent-manager import allowlist entries:\n  %s", strings.Join(stale, "\n  "))
	}
}

// fileImports parses only the import clause of a Go file and returns the
// imported paths. It is the detection primitive the boundary test and its
// red-proof share.
func fileImports(path string) (map[string]bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(file.Imports))
	for _, imp := range file.Imports {
		out[strings.Trim(imp.Path.Value, `"`)] = true
	}
	return out, nil
}

// TestAgentManagerImportDetectorFiresOnViolation red-proofs the import
// detector: a synthetic file importing the client is detected; one that does
// not is not.
func TestAgentManagerImportDetectorFiresOnViolation(t *testing.T) {
	dir := t.TempDir()
	violating := filepath.Join(dir, "violating.go")
	if err := os.WriteFile(violating, []byte("package sample\n\nimport _ \""+agentManagerImportPath+"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	clean := filepath.Join(dir, "clean.go")
	if err := os.WriteFile(clean, []byte("package sample\n\nimport _ \"fmt\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := fileImports(violating)
	if err != nil || !got[agentManagerImportPath] {
		t.Fatalf("detector must flag the synthetic import (err=%v, imports=%v)", err, got)
	}
	got, err = fileImports(clean)
	if err != nil || got[agentManagerImportPath] {
		t.Fatalf("detector must not flag a clean file (err=%v, imports=%v)", err, got)
	}
}
