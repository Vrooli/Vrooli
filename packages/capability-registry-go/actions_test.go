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
