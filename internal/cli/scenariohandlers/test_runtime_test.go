package scenariohandlers

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"

	scenario "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/scenarioexec"
)

type capturedSubprocess struct {
	calls  []scenarioexec.SubprocessSpec
	stdout string
	err    error
	onRun  func(scenarioexec.SubprocessSpec) error
}

func (c *capturedSubprocess) Run(_ struct{}, spec scenarioexec.SubprocessSpec) error {
	c.calls = append(c.calls, spec)
	if c.onRun != nil {
		if err := c.onRun(spec); err != nil {
			return err
		}
	}
	if c.stdout != "" && spec.Stdout != nil {
		_, _ = io.WriteString(spec.Stdout, c.stdout)
	}
	return c.err
}

func newScenarioTestDeps(
	root string,
	globals rootcli.GlobalOptions,
	stdout io.Writer,
	stderr io.Writer,
	capture *capturedSubprocess,
) HandlerDeps[struct{}] {
	return HandlerDeps[struct{}]{
		Stdout: func(struct{}) io.Writer { return stdout },
		Stderr: func(struct{}) io.Writer { return stderr },
		Root:   func(struct{}) string { return root },
		Globals: func(struct{}) rootcli.GlobalOptions {
			return globals
		},
		OutputFormat: func(struct{}) (cliout.Format, error) { return cliout.FormatHuman, nil },
		LifecycleRunner: func(struct{}) (scenario.PhaseRunner, error) {
			return nil, errors.New("lifecycle runner must not be used by scenario test")
		},
		LocateTestGenieCLI: func(struct{}) (string, error) { return "/bin/test-genie", nil },
		RunSubprocess:      capture.Run,
		CommandEnv:         func(struct{}) []string { return []string{"A=B"} },
	}
}

func TestScenarioTestDelegatesDirectlyToTestGenieExecute(t *testing.T) {
	var stdout, stderr bytes.Buffer
	capture := &capturedSubprocess{}
	deps := newScenarioTestDeps("/repo", rootcli.GlobalOptions{}, &stdout, &stderr, capture)

	if err := TestHandler(deps)(struct{}{}, []string{"plan-manager", "--preset", "comprehensive", "--wait"}); err != nil {
		t.Fatalf("TestHandler returned error: %v", err)
	}
	if len(capture.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(capture.calls))
	}
	call := capture.calls[0]
	if call.Name != "/bin/test-genie" {
		t.Fatalf("name = %q", call.Name)
	}
	wantArgs := []string{"--auto-start", "execute", "plan-manager", "--preset", "comprehensive", "--wait"}
	if !reflect.DeepEqual(call.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", call.Args, wantArgs)
	}
	if call.Dir != "/repo" {
		t.Fatalf("dir = %q, want /repo", call.Dir)
	}
	if !reflect.DeepEqual(call.Env, []string{"A=B"}) {
		t.Fatalf("env = %#v", call.Env)
	}
	if call.Stdout != &stdout || call.Stderr != &stderr {
		t.Fatal("subprocess must receive wrapper stdout/stderr for direct passthrough")
	}
}

func TestScenarioTestGlobalJSONForwardsToTestGenie(t *testing.T) {
	capture := &capturedSubprocess{}
	deps := newScenarioTestDeps("/repo", rootcli.GlobalOptions{JSON: true}, io.Discard, io.Discard, capture)

	if err := TestHandler(deps)(struct{}{}, []string{"plan-manager", "--wait"}); err != nil {
		t.Fatalf("TestHandler returned error: %v", err)
	}
	wantArgs := []string{"--auto-start", "execute", "plan-manager", "--wait", "--json"}
	if !reflect.DeepEqual(capture.calls[0].Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", capture.calls[0].Args, wantArgs)
	}
}

func TestScenarioTestDoesNotDuplicateExplicitMachineFormat(t *testing.T) {
	for _, explicit := range []string{"--json", "--jsonl"} {
		capture := &capturedSubprocess{}
		deps := newScenarioTestDeps("/repo", rootcli.GlobalOptions{JSON: true}, io.Discard, io.Discard, capture)

		if err := TestHandler(deps)(struct{}{}, []string{"plan-manager", explicit}); err != nil {
			t.Fatalf("TestHandler(%s) returned error: %v", explicit, err)
		}
		wantArgs := []string{"--auto-start", "execute", "plan-manager", explicit}
		if !reflect.DeepEqual(capture.calls[0].Args, wantArgs) {
			t.Fatalf("args = %#v, want %#v", capture.calls[0].Args, wantArgs)
		}
	}
}

func TestScenarioTestDoesNotUseLifecycleRunner(t *testing.T) {
	capture := &capturedSubprocess{}
	deps := newScenarioTestDeps("/repo", rootcli.GlobalOptions{}, io.Discard, io.Discard, capture)

	if err := TestHandler(deps)(struct{}{}, []string{"plan-manager"}); err != nil {
		t.Fatalf("TestHandler returned error: %v", err)
	}
}
