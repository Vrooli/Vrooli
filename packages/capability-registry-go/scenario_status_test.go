package capabilityregistry

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type commandRunnerFunc func(context.Context, string, ...string) (CommandResult, error)

func (f commandRunnerFunc) Run(ctx context.Context, name string, args ...string) (CommandResult, error) {
	return f(ctx, name, args...)
}

func TestScenarioStatusCheckerUsesBoundedControlPlaneCommand(t *testing.T) {
	checker := ScenarioStatusChecker{
		Scenario: " audio-tools ",
		CLIPath:  "/test/vrooli",
		Timeout:  time.Second,
		Runner: commandRunnerFunc(func(ctx context.Context, name string, args ...string) (CommandResult, error) {
			if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > time.Second {
				t.Fatalf("command context has no bounded deadline")
			}
			if name != "/test/vrooli" || !reflect.DeepEqual(args, []string{"scenario", "status", "audio-tools", "--json"}) {
				t.Fatalf("command = %q %v", name, args)
			}
			return CommandResult{Stdout: []byte(`{"scenario":{"status":"HEALTHY"}}`)}, nil
		}),
	}
	status, message := checker.Check(context.Background())
	if status != StatusAvailable || message != "audio-tools is healthy" {
		t.Fatalf("check = %q, %q", status, message)
	}
}

func TestScenarioStatusCheckerDefaultDeadlineMatchesLivenessBudget(t *testing.T) {
	checker := ScenarioStatusChecker{Scenario: "audio-tools", Runner: commandRunnerFunc(func(ctx context.Context, _ string, _ ...string) (CommandResult, error) {
		deadline, ok := ctx.Deadline()
		remaining := time.Until(deadline)
		if !ok || remaining < livenessCheckerBudget-100*time.Millisecond || remaining > livenessCheckerBudget {
			t.Fatalf("default timeout remaining = %s, want near %s", remaining, livenessCheckerBudget)
		}
		return CommandResult{Stdout: []byte(`{"scenario":{"status":"running"}}`)}, nil
	})}
	if status, _ := checker.Check(context.Background()); status != StatusAvailable {
		t.Fatalf("status = %q, want available", status)
	}
}

func TestScenarioStatusCheckerInheritsCallerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	checker := ScenarioStatusChecker{Scenario: "audio-tools", Runner: commandRunnerFunc(func(ctx context.Context, _ string, _ ...string) (CommandResult, error) {
		called = true
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatalf("runner context error = %v, want canceled", ctx.Err())
		}
		return CommandResult{}, ctx.Err()
	})}
	status, _ := checker.Check(ctx)
	if status != StatusUnavailable || !called {
		t.Fatalf("status = %q, runner called = %v", status, called)
	}
}

func TestScenarioStatusCheckerHandlesStoppedAndInvalidStatus(t *testing.T) {
	for _, tc := range []struct {
		name, stdout, wantMessage string
		wantStatus                Status
	}{
		{name: "running", stdout: `{"scenario":{"status":"RUNNING"}}`, wantMessage: "audio-tools is healthy", wantStatus: StatusAvailable},
		{name: "stopped", stdout: `{"scenario":{"status":"stopped"}}`, wantMessage: "audio-tools is not running; start it with the operator action", wantStatus: StatusUnavailable},
		{name: "malformed", stdout: `{`, wantMessage: "audio-tools status was not valid JSON", wantStatus: StatusUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checker := ScenarioStatusChecker{Scenario: "audio-tools", Runner: commandRunnerFunc(func(context.Context, string, ...string) (CommandResult, error) {
				return CommandResult{Stdout: []byte(tc.stdout)}, nil
			})}
			status, message := checker.Check(context.Background())
			if status != tc.wantStatus || message != tc.wantMessage {
				t.Fatalf("check = %q, %q", status, message)
			}
		})
	}
}

func TestScenarioStatusCheckerBoundsRunnerAndReportsUnavailable(t *testing.T) {
	checker := ScenarioStatusChecker{
		Scenario: "audio-tools",
		Timeout:  time.Millisecond,
		Runner: commandRunnerFunc(func(ctx context.Context, _ string, _ ...string) (CommandResult, error) {
			<-ctx.Done()
			return CommandResult{}, ctx.Err()
		}),
	}
	status, message := checker.Check(context.Background())
	if status != StatusUnavailable || message != "audio-tools is unavailable; start it with the operator action" {
		t.Fatalf("check = %q, %q", status, message)
	}
}

func TestScenarioStatusCheckerUsesDefaultCLIAndRejectsEmptyTarget(t *testing.T) {
	called := false
	checker := ScenarioStatusChecker{Scenario: "audio-tools", Runner: commandRunnerFunc(func(_ context.Context, name string, _ ...string) (CommandResult, error) {
		called = true
		if name != "vrooli" {
			t.Fatalf("CLI path = %q, want default vrooli", name)
		}
		return CommandResult{}, errors.New("missing")
	})}
	status, _ := checker.Check(context.Background())
	if status != StatusUnavailable || !called {
		t.Fatalf("check status = %q, runner called = %v", status, called)
	}
	called = false
	status, _ = (ScenarioStatusChecker{}).Check(context.Background())
	if status != StatusUnavailable || called {
		t.Fatalf("empty target status = %q, runner called = %v", status, called)
	}
}
