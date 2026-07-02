package capabilities

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type recordedCommand struct {
	name string
	args []string
}

type fakeCommandRunner struct {
	results []CommandResult
	errs    []error
	calls   []recordedCommand
}

func (f *fakeCommandRunner) Run(_ context.Context, name string, args ...string) (CommandResult, error) {
	f.calls = append(f.calls, recordedCommand{name: name, args: append([]string(nil), args...)})
	i := len(f.calls) - 1
	if i < len(f.results) {
		err := error(nil)
		if i < len(f.errs) {
			err = f.errs[i]
		}
		return f.results[i], err
	}
	return CommandResult{}, nil
}

func lifecycleActionTestService(runner *fakeCommandRunner) LifecycleActionService {
	return LifecycleActionService{
		Defs: []Def{
			{
				ID:             "audio-tools",
				DependencyKind: DependencyScenario,
				DependencySlug: "audio-tools",
			},
			{
				ID:             "ollama",
				DependencyKind: DependencyResource,
				DependencySlug: "ollama",
			},
		},
		Runner:  runner,
		CLIPath: "vrooli",
		Timeout: time.Second,
	}
}

func TestLifecycleActionServiceRunsStartThenSingleWait(t *testing.T) {
	runner := &fakeCommandRunner{results: []CommandResult{
		{Stdout: []byte(`{"success":true}`), ExitCode: 0},
		{Stdout: []byte(`{"success":true,"verdict":"healthy","exit_code":0}`), ExitCode: 0},
	}}
	svc := lifecycleActionTestService(runner)

	got, err := svc.Run(context.Background(), LifecycleActionRequest{CapabilityID: "audio-tools", ActionKind: ActionKindScenarioStart})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !got.Success || got.Status != "healthy" {
		t.Fatalf("result = %+v, want healthy success", got)
	}
	want := []recordedCommand{
		{name: "vrooli", args: []string{"scenario", "start", "audio-tools", "--json", "--timeout", "1"}},
		{name: "vrooli", args: []string{"scenario", "wait", "audio-tools", "--json", "--timeout", "1"}},
	}
	if !reflect.DeepEqual(runner.calls, want) {
		t.Fatalf("commands = %#v, want %#v", runner.calls, want)
	}
}

func TestLifecycleActionServiceRunsRestart(t *testing.T) {
	runner := &fakeCommandRunner{results: []CommandResult{
		{Stdout: []byte(`{"success":true}`), ExitCode: 0},
		{Stdout: []byte(`{"success":true,"verdict":"healthy","exit_code":0}`), ExitCode: 0},
	}}
	svc := lifecycleActionTestService(runner)

	got, err := svc.Run(context.Background(), LifecycleActionRequest{CapabilityID: "audio-tools", ActionKind: ActionKindScenarioRestart})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !got.Success {
		t.Fatalf("result = %+v, want success", got)
	}
	if runner.calls[0].args[1] != "restart" {
		t.Fatalf("first command = %#v, want restart", runner.calls[0].args)
	}
}

func TestLifecycleActionServiceReportsStartFailureWithoutWait(t *testing.T) {
	runner := &fakeCommandRunner{
		results: []CommandResult{{Stderr: []byte("build failed"), ExitCode: 1}},
		errs:    []error{errors.New("exit status 1")},
	}
	svc := lifecycleActionTestService(runner)

	got, err := svc.Run(context.Background(), LifecycleActionRequest{CapabilityID: "audio-tools", ActionKind: ActionKindScenarioStart})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got.Success || got.Status != "failed" {
		t.Fatalf("result = %+v, want failed", got)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(runner.calls))
	}
}

func TestLifecycleActionServiceReportsWaitFailure(t *testing.T) {
	runner := &fakeCommandRunner{
		results: []CommandResult{
			{Stdout: []byte(`{"success":true}`), ExitCode: 0},
			{Stderr: []byte("timeout"), ExitCode: 124},
		},
		errs: []error{nil, errors.New("exit status 124")},
	}
	svc := lifecycleActionTestService(runner)

	got, err := svc.Run(context.Background(), LifecycleActionRequest{CapabilityID: "audio-tools", ActionKind: ActionKindScenarioStart})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got.Success || got.Status != "failed" || got.Message == "" {
		t.Fatalf("result = %+v, want wait failure", got)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(runner.calls))
	}
}

func TestLifecycleActionServiceRejectsUndeclaredAndResourceCapabilities(t *testing.T) {
	runner := &fakeCommandRunner{}
	svc := lifecycleActionTestService(runner)

	for _, capabilityID := range []string{"missing", "ollama"} {
		if _, err := svc.Run(context.Background(), LifecycleActionRequest{CapabilityID: capabilityID, ActionKind: ActionKindScenarioStart}); err == nil {
			t.Fatalf("Run(%q) error = nil, want rejection", capabilityID)
		}
	}
	if len(runner.calls) != 0 {
		t.Fatalf("calls = %d, want 0", len(runner.calls))
	}
}

func TestLifecycleActionServiceRejectsUnsupportedAction(t *testing.T) {
	runner := &fakeCommandRunner{}
	svc := lifecycleActionTestService(runner)

	if _, err := svc.Run(context.Background(), LifecycleActionRequest{CapabilityID: "audio-tools", ActionKind: ActionKindOperatorCommand}); err == nil {
		t.Fatal("Run error = nil, want unsupported action rejection")
	}
	if len(runner.calls) != 0 {
		t.Fatalf("calls = %d, want 0", len(runner.calls))
	}
}

func TestLifecycleActionServicePassesSlugAsSingleArg(t *testing.T) {
	runner := &fakeCommandRunner{results: []CommandResult{
		{Stdout: []byte(`{"success":true}`), ExitCode: 0},
		{Stdout: []byte(`{"success":true,"verdict":"healthy","exit_code":0}`), ExitCode: 0},
	}}
	svc := LifecycleActionService{
		Defs: []Def{{
			ID:             "scenario-x",
			DependencyKind: DependencyScenario,
			DependencySlug: "scenario-x; rm -rf /",
		}},
		Runner:  runner,
		CLIPath: "vrooli",
		Timeout: time.Second,
	}

	if _, err := svc.Run(context.Background(), LifecycleActionRequest{CapabilityID: "scenario-x", ActionKind: ActionKindScenarioStart}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got := runner.calls[0].args[2]; got != "scenario-x; rm -rf /" {
		t.Fatalf("slug arg = %q, want single original argument", got)
	}
}
