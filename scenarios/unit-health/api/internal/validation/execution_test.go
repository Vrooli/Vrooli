package validation

import (
	"context"
	"path/filepath"
	"testing"

	"unit-health/internal/discovery"
	"unit-health/internal/executor"
)

// fakeExecutor returns a canned result keyed by workspace id.
type fakeExecutor struct {
	byWorkspace map[string]executor.Result
}

func (f fakeExecutor) Run(_ context.Context, cmd executor.Command) executor.Result {
	r, ok := f.byWorkspace[cmd.WorkspaceID]
	if !ok {
		return executor.Result{WorkspaceID: cmd.WorkspaceID, Name: cmd.Name, Status: executor.StatusPassed}
	}
	r.WorkspaceID = cmd.WorkspaceID
	r.Name = cmd.Name
	return r
}

func goSurfaceInventory(t *testing.T) discovery.Inventory {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "api", "go.mod"), "module x\n")
	return discovery.Inventory{
		Scenario: "demo", TargetKind: "scenario", RootPath: root,
		Surfaces: []discovery.Surface{{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api"), Status: "known"}},
	}
}

func TestExecuteSkippedWithoutIncludeExecution(t *testing.T) {
	spec := loadSpec(t)
	svc := newService(fakeDiscoverer{inv: goSurfaceInventory(t)}, spec)
	svc.Executor = fakeExecutor{byWorkspace: map[string]executor.Result{"api": {Status: executor.StatusFailed, FailureClass: executor.ClassTestFailure, ExitCode: 1}}}

	resp, err := svc.Validate(context.Background(), Request{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(resp.CommandResults) != 0 {
		t.Errorf("expected no command results without IncludeExecution, got %d", len(resp.CommandResults))
	}
	if resp.Status != "passed" {
		t.Errorf("status = %q, want passed (dry run)", resp.Status)
	}
}

func TestExecuteTestFailureProducesFindingAndFailedStatus(t *testing.T) {
	spec := loadSpec(t)
	svc := newService(fakeDiscoverer{inv: goSurfaceInventory(t)}, spec)
	svc.Executor = fakeExecutor{byWorkspace: map[string]executor.Result{
		"api": {Status: executor.StatusFailed, FailureClass: executor.ClassTestFailure, ExitCode: 1, Stderr: "--- FAIL: TestFoo"},
	}}

	resp, err := svc.Validate(context.Background(), Request{Scenario: "demo", IncludeExecution: true})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(resp.CommandResults) != 1 || resp.CommandResults[0].Status != "failed" {
		t.Fatalf("command results = %+v", resp.CommandResults)
	}
	if !hasFinding(resp.Findings, codeTestExecutionFailure) {
		t.Errorf("expected %s finding, got %v", codeTestExecutionFailure, codes(resp.Findings))
	}
	if resp.Status != "failed" {
		t.Errorf("status = %q, want failed", resp.Status)
	}
}

func TestExecuteTimeoutMapsToHangFinding(t *testing.T) {
	spec := loadSpec(t)
	svc := newService(fakeDiscoverer{inv: goSurfaceInventory(t)}, spec)
	svc.Executor = fakeExecutor{byWorkspace: map[string]executor.Result{
		"api": {Status: executor.StatusTimeout, FailureClass: executor.ClassTimeoutHang, FailureReason: "exceeded timeout"},
	}}

	resp, err := svc.Validate(context.Background(), Request{Scenario: "demo", IncludeExecution: true})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !hasFinding(resp.Findings, codeTestTimeoutHang) {
		t.Errorf("expected %s finding, got %v", codeTestTimeoutHang, codes(resp.Findings))
	}
}

func TestExecutePassProducesNoFinding(t *testing.T) {
	spec := loadSpec(t)
	svc := newService(fakeDiscoverer{inv: goSurfaceInventory(t)}, spec)
	svc.Executor = fakeExecutor{byWorkspace: map[string]executor.Result{"api": {Status: executor.StatusPassed}}}

	resp, err := svc.Validate(context.Background(), Request{Scenario: "demo", IncludeExecution: true})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(resp.CommandResults) != 1 || resp.CommandResults[0].Status != "passed" {
		t.Fatalf("command results = %+v", resp.CommandResults)
	}
	if resp.Status != "passed" {
		t.Errorf("status = %q, want passed", resp.Status)
	}
}
