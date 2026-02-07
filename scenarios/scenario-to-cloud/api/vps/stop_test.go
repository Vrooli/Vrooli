package vps

import (
	"context"
	"strings"
	"testing"

	"scenario-to-cloud/ssh"
)

// testSSHRunner implements ssh.Runner for vps package tests.
type testSSHRunner struct {
	responses map[string]ssh.Result
	errs      map[string]error
	calls     []string
}

func (r *testSSHRunner) Run(_ context.Context, _ ssh.Config, command string, _ ssh.RunOptions) (ssh.Result, error) {
	r.calls = append(r.calls, command)

	if err, ok := r.errs[command]; ok {
		return ssh.Result{ExitCode: 255}, err
	}
	// Try prefix matching for commands with dynamic content
	for pattern, res := range r.responses {
		if strings.Contains(command, pattern) {
			return res, nil
		}
	}
	if res, ok := r.responses[command]; ok {
		return res, nil
	}
	return ssh.Result{ExitCode: 0}, nil
}

func TestStopExistingScenario_Success(t *testing.T) {
	t.Parallel()

	runner := &testSSHRunner{
		responses: map[string]ssh.Result{
			"which vrooli":  {Stdout: "/usr/local/bin/vrooli", ExitCode: 0},
			"scenario stop": {ExitCode: 0},
			"pkill":         {ExitCode: 0},
		},
	}

	cfg := ssh.Config{Host: "test", User: "root", Port: 22}
	result := StopExistingScenario(context.Background(), runner, cfg, "/opt/vrooli", "my-app", []int{35000})

	if !result.OK {
		t.Errorf("expected OK, got error: %s", result.Error)
	}
	if !result.ScenarioStop {
		t.Error("expected ScenarioStop = true")
	}
}

func TestStopExistingScenario_StopFails_StillKills(t *testing.T) {
	t.Parallel()

	runner := &testSSHRunner{
		responses: map[string]ssh.Result{
			// vrooli CLI exists but stop fails
			"which vrooli":  {Stdout: "/usr/local/bin/vrooli", ExitCode: 0},
			"scenario stop": {ExitCode: 1, Stderr: "scenario not running"},
			"pkill":         {ExitCode: 0},
		},
	}

	cfg := ssh.Config{Host: "test", User: "root", Port: 22}
	result := StopExistingScenario(context.Background(), runner, cfg, "/opt/vrooli", "my-app", nil)

	if !result.OK {
		t.Errorf("expected OK even when stop fails, got error: %s", result.Error)
	}
	// ScenarioStop should be false since stop returned non-zero
	if result.ScenarioStop {
		t.Error("expected ScenarioStop = false when stop command fails")
	}
}
