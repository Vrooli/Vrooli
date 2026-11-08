package sessioncontain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	platform "github.com/vrooli/platform-go"
)

func TestCancellationStopsContainedDescendants(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("native scope qualification requires Linux")
	}
	if _, err := exec.LookPath("systemd-run"); err != nil {
		t.Skip("systemd-run unavailable")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	scope := fmt.Sprintf("vrooli-agent-test-cancel-%d", time.Now().UnixNano())
	type result struct {
		session cliutil.ContainedSession
		err     error
	}
	done := make(chan result, 1)
	go func() {
		session, err := (Container{}).Run(ctx, scope, testContainment(), cliutil.SessionProcess{
			Path: "/bin/sh", Args: []string{"-c", `sleep 300 & echo $! > "$1"; wait`, "fixture", pidFile}, Env: os.Environ(),
		})
		done <- result{session, err}
	}()
	var childPID int
	deadline := time.After(5 * time.Second)
	for childPID == 0 {
		if data, err := os.ReadFile(pidFile); err == nil {
			childPID, _ = strconv.Atoi(strings.TrimSpace(string(data)))
		}
		if childPID != 0 {
			break
		}
		select {
		case got := <-done:
			t.Fatalf("fixture did not start: %v", got.err)
		case <-deadline:
			t.Fatal("fixture startup timed out")
		case <-time.After(time.Millisecond):
		}
	}
	path, err := platform.ProcessScope(childPID)
	if err != nil || !strings.HasSuffix(path, "/"+scope+".scope") {
		t.Fatalf("fixture scope=%q error=%v", path, err)
	}
	ref := platform.ScopeRef{Kind: platform.ScopeKindCgroup, Path: path, Name: scope}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = platform.StopScopeContext(cleanup, ref)
	})
	cancel()
	select {
	case got := <-done:
		if !errors.Is(got.err, context.Canceled) {
			t.Fatalf("cancellation error=%v session=%+v", got.err, got.session)
		}
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		pids, err := platform.ScopeProcessesContext(cleanup, ref)
		if err != nil || len(pids) != 0 {
			t.Fatalf("cancellation left descendants: %v error=%v runner_error=%v session=%+v", pids, err, got.err, got.session)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("contained runner ignored caller cancellation")
	}
}

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
	t.Setenv(cliutil.AgentSessionEnv, "")
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
	t.Setenv(cliutil.AgentSessionEnv, "")
	Register()
	_, err := cliutil.LaunchCodingAgentResult(context.Background(), cliutil.AgentLaunchRequest{
		Agent: "codex", APIBase: "http://127.0.0.1:1", Args: []string{"/proc/self/cgroup"},
		LookPath: func(string) (string, error) { return "/bin/cat", nil }, AttachTimeout: 50 * time.Millisecond,
	})
	t.Fatalf("exec did not replace the process: %v", err)
}
