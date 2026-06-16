package executor

import (
	"context"
	"testing"
	"time"
)

func TestBoundedRunPasses(t *testing.T) {
	res := Bounded{}.Run(context.Background(), Command{
		WorkspaceID: "w", Name: "echo", Argv: []string{"sh", "-c", "echo hello"}, TimeoutSeconds: 10,
	})
	if res.Status != StatusPassed {
		t.Fatalf("status = %q (%s), want passed", res.Status, res.FailureReason)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit = %d, want 0", res.ExitCode)
	}
	if got := res.Stdout; got == "" {
		t.Errorf("expected captured stdout, got empty")
	}
}

func TestBoundedRunNonzeroIsFailed(t *testing.T) {
	res := Bounded{}.Run(context.Background(), Command{
		WorkspaceID: "w", Argv: []string{"sh", "-c", "echo boom >&2; exit 3"}, TimeoutSeconds: 10,
	})
	if res.Status != StatusFailed {
		t.Fatalf("status = %q, want failed", res.Status)
	}
	if res.ExitCode != 3 {
		t.Errorf("exit = %d, want 3", res.ExitCode)
	}
	if res.FailureClass != ClassTestFailure {
		t.Errorf("class = %q, want test_failure", res.FailureClass)
	}
}

func TestBoundedRunTimeoutHang(t *testing.T) {
	res := Bounded{}.Run(context.Background(), Command{
		WorkspaceID: "w", Argv: []string{"sh", "-c", "sleep 30"}, TimeoutSeconds: 1,
	})
	if res.Status != StatusTimeout {
		t.Fatalf("status = %q (%s), want timeout", res.Status, res.FailureReason)
	}
	if res.FailureClass != ClassTimeoutHang {
		t.Errorf("class = %q, want timeout_hang", res.FailureClass)
	}
}

func TestBoundedRunNoOutputStall(t *testing.T) {
	res := Bounded{NoOutputTimeout: 500 * time.Millisecond}.Run(context.Background(), Command{
		WorkspaceID: "w", Argv: []string{"sh", "-c", "sleep 30"}, TimeoutSeconds: 60,
	})
	if res.Status != StatusTimeout {
		t.Fatalf("status = %q, want timeout", res.Status)
	}
	if res.FailureClass != ClassNoOutputStall {
		t.Errorf("class = %q, want no_output_stall", res.FailureClass)
	}
}

func TestBoundedRunMissingCommand(t *testing.T) {
	res := Bounded{}.Run(context.Background(), Command{
		WorkspaceID: "w", Argv: []string{"definitely-not-a-real-binary-xyz"}, TimeoutSeconds: 5,
	})
	if res.Status != StatusError || res.FailureClass != ClassMissingDependency {
		t.Fatalf("got status=%q class=%q, want error/missing_dependency", res.Status, res.FailureClass)
	}
}

func TestRunAllPreservesOrderAndConcurrency(t *testing.T) {
	cmds := []Command{
		{WorkspaceID: "a", Argv: []string{"sh", "-c", "echo a"}, TimeoutSeconds: 10},
		{WorkspaceID: "b", Argv: []string{"sh", "-c", "exit 1"}, TimeoutSeconds: 10},
		{WorkspaceID: "c", Argv: []string{"sh", "-c", "echo c"}, TimeoutSeconds: 10},
	}
	results := RunAll(context.Background(), Bounded{}, cmds, 2)
	if len(results) != 3 {
		t.Fatalf("want 3 results, got %d", len(results))
	}
	if results[0].WorkspaceID != "a" || results[1].WorkspaceID != "b" || results[2].WorkspaceID != "c" {
		t.Errorf("order not preserved: %+v", results)
	}
	if results[1].Status != StatusFailed {
		t.Errorf("b should fail, got %q", results[1].Status)
	}
}
