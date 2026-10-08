package cliutil

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type fakeContainer struct {
	runs      []string
	selfCalls []string
	fail      bool
}

func (f *fakeContainer) Run(_ context.Context, scope string, c SessionContainment, p SessionProcess) (ContainedSession, error) {
	f.runs = append(f.runs, scope+" "+c.Slice)
	if f.fail {
		return ContainedSession{Method: ContainmentMethodNone}, &UncontainedError{Err: errors.New("no user manager")}
	}
	if p.Stdout != nil {
		_, _ = io.WriteString(p.Stdout, "0::/user.slice/user-1000.slice/user@1000.service/vrooli-agents.slice/"+scope+".scope\n")
	}
	return ContainedSession{Scope: "cgroup:/user.slice/user-1000.slice/user@1000.service/vrooli-agents.slice/" + scope + ".scope", Method: "systemd-run"}, nil
}

func (f *fakeContainer) ContainSelf(scope string, c SessionContainment) (ContainedSession, error) {
	f.selfCalls = append(f.selfCalls, scope)
	return ContainedSession{Scope: "cgroup:/x/" + scope + ".scope", Method: "transient-unit"}, nil
}

func testContainment() (SessionContainment, string) {
	return SessionContainment{CPUWeight: 50, TasksMax: 64, MemoryMax: "200M", Slice: AgentSlice}, ContainmentSourceDefaults
}

func withFakeContainer(t *testing.T, fail bool) *fakeContainer {
	t.Helper()
	previous, previousFn := DefaultSessionContainer, agentContainmentFn
	f := &fakeContainer{fail: fail}
	DefaultSessionContainer = f
	agentContainmentFn = testContainment
	t.Cleanup(func() { DefaultSessionContainer, agentContainmentFn = previous, previousFn })
	return f
}

// The spawn branch hands the child to the registered container with the
// session scope and reports the scope and method it got back.
func TestLaunchSpawnBranchUsesRegisteredContainer(t *testing.T) {
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	f := withFakeContainer(t, false)
	var stdout bytes.Buffer
	result, err := LaunchCodingAgentResult(context.Background(), AgentLaunchRequest{
		Agent: "codex", APIBase: "http://127.0.0.1:1", Args: []string{"-c", "true"},
		LookPath: func(string) (string, error) { return "/bin/sh", nil },
		Stdout:   &stdout, AttachTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("LaunchCodingAgentResult() error = %v", err)
	}
	if len(f.runs) != 1 || !strings.HasPrefix(f.runs[0], "vrooli-agent-") || !strings.HasSuffix(f.runs[0], " "+AgentSlice) {
		t.Fatalf("container runs = %v", f.runs)
	}
	if result.ContainmentMethod != "systemd-run" || !strings.Contains(result.Scope, "vrooli-agents.slice/vrooli-agent-") || result.ContainmentSource != ContainmentSourceDefaults || result.ContainmentFailure != "" {
		t.Fatalf("result = %+v", result)
	}
}

// A container that cannot set the ceiling up never blocks the launch: the
// child runs uncontained and the result says why.
func TestLaunchRunsUncontainedWhenTheContainerCannot(t *testing.T) {
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	withFakeContainer(t, true)
	var stdout bytes.Buffer
	result, err := LaunchCodingAgentResult(context.Background(), AgentLaunchRequest{
		Agent: "codex", APIBase: "http://127.0.0.1:1", Args: []string{"-c", "echo ran"},
		LookPath: func(string) (string, error) { return "/bin/sh", nil },
		Stdout:   &stdout, AttachTimeout: 50 * time.Millisecond,
	})
	if err != nil || strings.TrimSpace(stdout.String()) != "ran" {
		t.Fatalf("child did not run: %v %q", err, stdout.String())
	}
	if result.ContainmentMethod != ContainmentMethodNone || !strings.Contains(result.ContainmentFailure, "no user manager") {
		t.Fatalf("result = %+v", result)
	}
	DefaultSessionContainer = nil
	result, err = LaunchCodingAgentResult(context.Background(), AgentLaunchRequest{
		Agent: "codex", APIBase: "http://127.0.0.1:1", Args: []string{"-c", "true"},
		LookPath: func(string) (string, error) { return "/bin/sh", nil },
		Stdout:   &stdout, AttachTimeout: 50 * time.Millisecond,
	})
	if err != nil || !strings.Contains(result.ContainmentFailure, "no session container registered") {
		t.Fatalf("unregistered result = %+v, %v", result, err)
	}
}

func TestAgentScopeNameFoldsToUnitCharacters(t *testing.T) {
	if got := agentScopeName("run/1 x", ""); got != "vrooli-agent-run-1-x" {
		t.Fatalf("scope = %q", got)
	}
	if got := agentScopeName("", "sess"); got != "vrooli-agent-sess" {
		t.Fatalf("scope = %q", got)
	}
	if got := agentScopeName("", ""); !strings.HasPrefix(got, "vrooli-agent-") || len(got) <= len("vrooli-agent-") {
		t.Fatalf("minted scope = %q", got)
	}
}
