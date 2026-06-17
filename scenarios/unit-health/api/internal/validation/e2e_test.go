package validation

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"unit-health/internal/discovery"
	"unit-health/internal/runhistory"
	uhdb "unit-health/internal/testutil/db"
)

// TestEndToEndGoWorkspaceThroughValidate is a true pipeline test: it builds a
// real, self-contained Go module with a passing covered test, then drives the
// full Validate() path with the REAL bounded executor (running `go test`), the
// real coverage parser, the real analyzers, and real SQLite-backed history —
// not just /health. Discovery is injected (Code Facts is not reachable in a unit
// test) but everything downstream of it is live.
func TestEndToEndGoWorkspaceThroughValidate(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}

	// A single-package module keeps coverage merging off the covdata tool.
	root := t.TempDir()
	wsDir := filepath.Join(root, "api")
	writeFile(t, filepath.Join(wsDir, "go.mod"), "module e2efix\n\ngo 1.25\n")
	writeFile(t, filepath.Join(wsDir, "calc.go"), "package e2efix\n\nfunc Add(a, b int) int { return a + b }\n")
	writeFile(t, filepath.Join(wsDir, "calc_test.go"), `package e2efix

import "testing"

func TestAddHandlesErrorAndBoundary(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("add wrong")
	}
	if Add(0, 0) != 0 {
		t.Fatal("zero boundary wrong")
	}
}
`)

	inv := discovery.Inventory{
		Scenario: "e2e", TargetKind: "scenario", RootPath: root,
		Surfaces: []discovery.Surface{{ID: "api", Kind: "api", Language: "go", RootPath: wsDir, Status: "known"}},
	}

	db := uhdb.NewSQLite(t)
	if _, err := db.Exec(runhistory.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	repo := runhistory.NewRepository(db)

	svc := newService(fakeDiscoverer{inv: inv}, loadSpec(t))
	svc.History = repo // real bounded executor (Executor left nil)

	resp, err := svc.Validate(context.Background(), Request{Scenario: "e2e", IncludeExecution: true})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	// The test command actually ran.
	if len(resp.CommandResults) == 0 {
		t.Fatalf("expected at least one command result from real execution")
	}
	var ranGoTest bool
	for _, cr := range resp.CommandResults {
		if cr.Status == "passed" {
			ranGoTest = true
		}
	}
	if !ranGoTest {
		t.Fatalf("expected the go test command to pass; results=%+v", resp.CommandResults)
	}
	if resp.Status != "passed" {
		t.Errorf("overall status = %q, want passed; findings=%v", resp.Status, codes(resp.Findings))
	}

	// Coverage was parsed from the real coverage.out artifact.
	if len(resp.Coverage) == 0 {
		t.Errorf("expected coverage targets parsed from coverage.out")
	}

	// The run was persisted to history.
	hist, err := repo.CommandHistory(context.Background(), "e2e", 10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(hist) == 0 {
		t.Errorf("expected the executed run to be persisted to history")
	}

	// A maturity rung was assessed.
	if resp.Maturity.Label == "" {
		t.Errorf("expected a maturity label, got empty")
	}

	// Run artifacts were derived end-to-end: the run id, the target, the
	// executed command's working directory, and the coverage location.
	kinds := map[string]bool{}
	for _, a := range resp.Artifacts {
		kinds[a.Kind] = true
	}
	for _, want := range []string{"run", "target", "command", "coverage"} {
		if !kinds[want] {
			t.Errorf("expected a %q artifact; got %+v", want, resp.Artifacts)
		}
	}
}
