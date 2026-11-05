package sessioncontain

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	platform "github.com/vrooli/platform-go"
)

func testContainment() cliutil.SessionContainment {
	return cliutil.SessionContainment{CPUWeight: 50, TasksMax: 64, MemoryMax: "200M", Slice: cliutil.AgentSlice}
}

// [REQ:STORM-002] The spawn branch starts the agent in a scope under the
// agent slice and reports how.
func TestLaunchSpawnBranchLandsInAgentSlice(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("scopes are a systemd primitive")
	}
	if _, err := exec.LookPath("systemd-run"); err != nil {
		t.Skip("systemd-run is not on PATH")
	}
	t.Setenv(cliutil.AgentManagerIdentityTokenEnv, "")
	Register()
	var stdout bytes.Buffer
	session, err := Container{}.Run(context.Background(), "vrooli-agent-test-spawn", testContainment(), cliutil.SessionProcess{
		Path: "/bin/sh", Args: []string{"-c", "cat /proc/self/cgroup"}, Env: os.Environ(), Stdout: &stdout, Stderr: os.Stderr,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "vrooli-agents.slice/vrooli-agent-test-spawn.scope") {
		t.Fatalf("child cgroup = %q, want the scope under vrooli-agents.slice", stdout.String())
	}
	if session.Method != platform.MethodSystemdRun || !strings.HasPrefix(session.Scope, "cgroup:/") {
		t.Fatalf("session = %+v", session)
	}
	// Through the launcher itself, the same child lands in a minted scope.
	stdout.Reset()
	result, err := cliutil.LaunchCodingAgentResult(context.Background(), cliutil.AgentLaunchRequest{
		Agent: "codex", APIBase: "http://127.0.0.1:1", Args: []string{"-c", "cat /proc/self/cgroup"},
		LookPath: func(string) (string, error) { return "/bin/sh", nil },
		Stdout:   &stdout, Stderr: os.Stderr, AttachTimeout: 50 * time.Millisecond,
	})
	if err != nil || !strings.Contains(stdout.String(), "vrooli-agents.slice/vrooli-agent-") || result.ContainmentMethod != platform.MethodSystemdRun {
		t.Fatalf("launcher: err=%v out=%q result=%+v", err, stdout.String(), result)
	}
}

// The exec-replace branch moves the launcher into the scope before the exec,
// so the process that replaces it is already inside the ceiling.
func TestLaunchExecBranchMovesSelfBeforeExec(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("scopes are a systemd primitive")
	}
	if _, err := exec.LookPath("busctl"); err != nil {
		t.Skip("busctl is not on PATH")
	}
	cmd := exec.Command(os.Args[0], "-test.run", "^TestHelperExecBranch$")
	cmd.Env = append(os.Environ(), "VROOLI_TEST_EXEC_BRANCH=1", cliutil.AgentManagerIdentityTokenEnv+"=")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("helper: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "vrooli-agents.slice/vrooli-agent-") {
		t.Fatalf("exec'd process cgroup = %q, want a scope under vrooli-agents.slice", output)
	}
}

// TestHelperExecBranch is the subprocess of TestLaunchExecBranchMovesSelfBeforeExec:
// it takes the exec-replace branch with inherited stdio and becomes `cat
// /proc/self/cgroup`, so its stdout is the cgroup of the exec'd image.
func TestHelperExecBranch(t *testing.T) {
	if os.Getenv("VROOLI_TEST_EXEC_BRANCH") != "1" {
		t.Skip("helper only")
	}
	Register()
	_, err := cliutil.LaunchCodingAgentResult(context.Background(), cliutil.AgentLaunchRequest{
		Agent: "codex", APIBase: "http://127.0.0.1:1", Args: []string{"/proc/self/cgroup"},
		LookPath: func(string) (string, error) { return "/bin/cat", nil }, AttachTimeout: 50 * time.Millisecond,
	})
	t.Fatalf("exec did not replace the process: %v", err)
}
