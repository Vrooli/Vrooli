package vrooli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

func TestRuntimeSupervisorCheckNonLinuxIsNotApplicable(t *testing.T) {
	originalOS := checkOS
	checkOS = "darwin"
	t.Cleanup(func() { checkOS = originalOS })

	result := NewRuntimeSupervisorCheck().Run(context.Background())
	if result.Status != checks.StatusNotApplicable {
		t.Fatalf("status = %v, want %v", result.Status, checks.StatusNotApplicable)
	}
}

func TestRuntimeSupervisorCheckReportsRunning(t *testing.T) {
	exec := testutil.NewMockExecutor()
	exec.Responses["vrooli runtime supervisor status --json"] = testutil.MockResponse{
		Output: []byte(`{"status":"running","supervisor_id":"sup-live","pid":1234}`),
	}
	check := NewRuntimeSupervisorCheck(WithRuntimeSupervisorExecutor(exec))

	result := check.Run(context.Background())
	if result.Status != checks.StatusOK {
		t.Fatalf("Status = %s, want ok; result=%#v", result.Status, result)
	}
	if got := result.Details["status"]; got != "running" {
		t.Fatalf("details status = %#v, want running", got)
	}
}

func TestRuntimeSupervisorCheckReportsDeadAsCritical(t *testing.T) {
	exec := testutil.NewMockExecutor()
	exec.Responses["vrooli runtime supervisor status --json"] = testutil.MockResponse{
		Output: []byte(`{"status":"dead","status_reason":"recorded PID 11980 is not running","supervisor_id":"sup-dead","pid":11980}`),
	}
	check := NewRuntimeSupervisorCheck(WithRuntimeSupervisorExecutor(exec))

	result := check.Run(context.Background())
	if result.Status != checks.StatusCritical {
		t.Fatalf("Status = %s, want critical; result=%#v", result.Status, result)
	}
	if !containsText(result.Message, "recorded PID") {
		t.Fatalf("Message = %q, want status reason", result.Message)
	}

	actions := check.RecoveryActions(&result)
	if len(actions) == 0 || actions[0].ID != "install-start" || !actions[0].Available || actions[0].Dangerous {
		t.Fatalf("actions = %#v, want available safe install-start first", actions)
	}
}

func TestRuntimeSupervisorCheckInstallStartAction(t *testing.T) {
	exec := testutil.NewMockExecutor()
	exec.Responses["vrooli runtime supervisor install --user"] = testutil.MockResponse{
		Output: []byte("Installed runtime supervisor service\n"),
	}
	check := NewRuntimeSupervisorCheck(WithRuntimeSupervisorExecutor(exec))

	result := check.ExecuteAction(context.Background(), "install-start")
	if !result.Success {
		t.Fatalf("Success = false, want true; result=%#v", result)
	}
	if len(exec.Calls) != 1 || exec.Calls[0].Name != "vrooli" || strings.Join(exec.Calls[0].Args, " ") != "runtime supervisor install --user" {
		t.Fatalf("calls = %#v, want install command", exec.Calls)
	}
}

func TestRuntimeSupervisorCheckStatusCommandFailure(t *testing.T) {
	exec := testutil.NewMockExecutor()
	exec.Responses["vrooli runtime supervisor status --json"] = testutil.MockResponse{
		Output: []byte("boom"),
		Error:  errors.New("exit status 1"),
	}
	check := NewRuntimeSupervisorCheck(WithRuntimeSupervisorExecutor(exec))

	result := check.Run(context.Background())
	if result.Status != checks.StatusCritical {
		t.Fatalf("Status = %s, want critical; result=%#v", result.Status, result)
	}
	if result.Details["error"] == "" {
		t.Fatalf("Details error missing: %#v", result.Details)
	}
}
