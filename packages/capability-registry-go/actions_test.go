package capabilityregistry

import (
	"context"
	"errors"
	"testing"
	"time"
)

type actionRunner struct {
	calls [][]string
	err   error
}

func (r *actionRunner) Run(_ context.Context, name string, args ...string) (CommandResult, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if r.err != nil {
		return CommandResult{Stderr: []byte("runner failed")}, r.err
	}
	return CommandResult{Stdout: []byte(`{"success":true,"verdict":"ready"}`)}, nil
}

func TestLifecycleActionServiceRunsOneWaitForDeclaredScenario(t *testing.T) {
	runner := &actionRunner{}
	service := LifecycleActionService{Defs: []Def{{ID: "swarm", DependencyKind: DependencyScenario, DependencySlug: "swarm-manager", ActionKind: ActionKindScenarioStart}}, Runner: runner, CLIPath: "vrooli", Timeout: time.Second}
	got, err := service.Run(context.Background(), LifecycleActionRequest{IntegrationID: "swarm", ActionKind: ActionKindScenarioStart})
	if err != nil || !got.Success {
		t.Fatalf("result=%+v err=%v", got, err)
	}
	if len(runner.calls) != 2 || runner.calls[1][1] != "scenario" || runner.calls[1][2] != "wait" {
		t.Fatalf("calls=%v, want start then one wait", runner.calls)
	}
}

func TestLifecycleActionServiceRejectsUndeclaredAndNonScenarioTargets(t *testing.T) {
	service := LifecycleActionService{Defs: []Def{{ID: "resource", DependencyKind: DependencyResource, DependencySlug: "redis"}}}
	for _, req := range []LifecycleActionRequest{{IntegrationID: "missing", ActionKind: ActionKindScenarioStart}, {IntegrationID: "resource", ActionKind: ActionKindScenarioStart}} {
		if _, err := service.Run(context.Background(), req); err == nil {
			t.Fatalf("request %+v unexpectedly accepted", req)
		}
	}
}

func TestLifecycleActionServiceDoesNotWaitAfterStartFailure(t *testing.T) {
	runner := &actionRunner{err: errors.New("no cli")}
	service := LifecycleActionService{Defs: []Def{{ID: "swarm", DependencyKind: DependencyScenario, DependencySlug: "swarm-manager"}}, Runner: runner}
	got, err := service.Run(context.Background(), LifecycleActionRequest{IntegrationID: "swarm", ActionKind: ActionKindScenarioRestart})
	if err != nil || got.Success || len(runner.calls) != 1 {
		t.Fatalf("result=%+v err=%v calls=%v", got, err, runner.calls)
	}
}

func TestLifecycleActionServiceRunsDeclaredOperatorActionWithoutShellText(t *testing.T) {
	runner := &actionRunner{}
	service := LifecycleActionService{
		Defs:    []Def{{ID: "codex", DependencyKind: DependencyResource, DependencySlug: "codex", ActionKind: ActionKindOperatorCommand, OperatorCommand: "vrooli resource install codex --json"}},
		Runner:  runner,
		CLIPath: "/usr/local/bin/vrooli",
		Timeout: time.Second,
	}
	got, err := service.Run(context.Background(), LifecycleActionRequest{IntegrationID: "codex", ActionKind: ActionKindOperatorCommand})
	if err != nil || !got.Success {
		t.Fatalf("result=%+v err=%v", got, err)
	}
	if len(runner.calls) != 1 || runner.calls[0][0] != "/usr/local/bin/vrooli" {
		t.Fatalf("calls=%v, want one fixed CLI invocation", runner.calls)
	}
	want := []string{"/usr/local/bin/vrooli", "resource", "install", "codex", "--json"}
	for i, value := range want {
		if runner.calls[0][i] != value {
			t.Fatalf("argv=%v, want %v", runner.calls[0], want)
		}
	}
}

func TestLifecycleActionServiceRejectsDuplicateInFlightAction(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	runner := &blockingActionRunner{started: started, release: release}
	tracker := NewActionTracker()
	service := LifecycleActionService{
		Defs:    []Def{{ID: "swarm", DependencyKind: DependencyScenario, DependencySlug: "swarm-manager"}},
		Runner:  runner,
		Tracker: tracker,
	}
	firstDone := make(chan error, 1)
	go func() {
		_, err := service.Run(context.Background(), LifecycleActionRequest{IntegrationID: "swarm", ActionKind: ActionKindScenarioRestart})
		firstDone <- err
	}()
	<-started
	if _, err := service.Run(context.Background(), LifecycleActionRequest{IntegrationID: "swarm", ActionKind: ActionKindScenarioStart}); !errors.Is(err, ErrActionInFlight) {
		t.Fatalf("duplicate action error = %v, want ErrActionInFlight", err)
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first action error = %v", err)
	}
}

type blockingActionRunner struct {
	started chan<- struct{}
	release <-chan struct{}
}

func (r *blockingActionRunner) Run(ctx context.Context, _ string, args ...string) (CommandResult, error) {
	if len(args) > 1 && args[1] == "restart" {
		select {
		case r.started <- struct{}{}:
		default:
		}
		select {
		case <-r.release:
		case <-ctx.Done():
			return CommandResult{}, ctx.Err()
		}
	}
	return CommandResult{Stdout: []byte(`{"success":true,"verdict":"ready"}`)}, nil
}
