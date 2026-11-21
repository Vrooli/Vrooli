package depsapproved

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestDiscoverBuildableModulesSkipsDocsEvidence(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("module example.com/test\n\ngo 1.26.0\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod")
	write("resources/go.mod")
	write("docs/go.mod")
	write("docs/evidence/preimage/go.mod")
	write("scenarios/demo/api/go.mod")
	write("scenarios/demo/docs/go.mod")
	write("scenarios/demo/api/internal/testdata/fixture/go.mod")
	write("scenarios/demo/bas/fixtures/go/go.mod")
	write("templates/scenarios/basic/api/go.mod")

	modules, err := discoverBuildableModulesAt(root)
	if err != nil {
		t.Fatalf("discoverBuildableModulesAt() error = %v", err)
	}
	for _, active := range []string{"go.mod", "resources/go.mod", "scenarios/demo/api/go.mod"} {
		if _, ok := modules[active]; !ok {
			t.Errorf("active module %q was not discovered", active)
		}
	}
	for _, excluded := range []string{"docs/go.mod", "docs/evidence/preimage/go.mod", "scenarios/demo/docs/go.mod", "scenarios/demo/api/internal/testdata/fixture/go.mod", "scenarios/demo/bas/fixtures/go/go.mod", "templates/scenarios/basic/api/go.mod"} {
		if _, ok := modules[excluded]; ok {
			t.Errorf("docs evidence module %q was discovered", excluded)
		}
	}
}

func TestReconcileReportSeparatesChangesRefusalsAndFailures(t *testing.T) {
	targets := []reconcileTarget{{label: "demo"}, {label: "other"}}
	responses := []*scenariovalidationv1.FixResponse{
		{Candidates: []*scenariovalidationv1.FixCandidate{
			{RuleId: "dependency.go.tidy", FilePath: "scenarios/demo/api/go.mod", Description: "Converge go.mod: add m v1 // indirect", Before: "a", After: "b"},
			{RuleId: "dependency.go.tidy", FilePath: "scenarios/demo/api/go.sum", Description: "sync", Before: "big", After: "bigger"},
		}},
		{Messages: []string{
			"refused scenarios/other/api/go.mod: go/example.com/new@v1.0.0 (unrecorded): Record it first",
			"failed scenarios/other/cli/go.mod: go mod tidy: boom",
			"All Go surfaces are converged on their import closure.",
		}},
	}
	report := buildReconcileReport(targets, responses, false, false)
	if len(report.Changes) != 2 || report.Changes[0].After != "b" || report.Changes[1].After != "" {
		t.Fatalf("changes = %#v, want go.mod content and summarized go.sum", report.Changes)
	}
	if len(report.Refused) != 1 || !strings.Contains(report.Refused[0], "example.com/new") || len(report.Failed) != 1 {
		t.Fatalf("refused=%q failed=%q", report.Refused, report.Failed)
	}
	if report.err() == nil {
		t.Fatal("refusals and failures must exit non-zero")
	}
}

func TestReconcileReportCheckFailsOnlyWhenNotConverged(t *testing.T) {
	clean := buildReconcileReport([]reconcileTarget{{label: "demo"}}, []*scenariovalidationv1.FixResponse{{}}, false, true)
	if clean.Mode != "check" || clean.err() != nil {
		t.Fatalf("a converged check must pass: %#v %v", clean, clean.err())
	}
	pending := buildReconcileReport([]reconcileTarget{{label: "demo"}}, []*scenariovalidationv1.FixResponse{{Candidates: []*scenariovalidationv1.FixCandidate{{FilePath: "x/go.mod"}}}}, false, true)
	if pending.err() == nil {
		t.Fatal("--check must fail when a file would change")
	}
	preview := buildReconcileReport([]reconcileTarget{{label: "demo"}}, []*scenariovalidationv1.FixResponse{{Candidates: []*scenariovalidationv1.FixCandidate{{FilePath: "x/go.mod"}}}}, false, false)
	if preview.err() != nil {
		t.Fatal("a plain dry run reports changes without failing")
	}
}
