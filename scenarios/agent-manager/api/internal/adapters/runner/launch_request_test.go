package runner

import (
	"io"
	"testing"
)

func TestBuildEnvWrappedLaunchRequestPreservesAgentInvocationContract(t *testing.T) {
	req := BuildEnvWrappedLaunchRequest(
		"CODEX_AGENT_TAG", "/opt/bin/codex", []string{"exec", "--json"},
		"run-123", "solve this", []string{"PATH=/bin", "LANG=C"}, "/work",
	)
	if req.Command != "env" || req.WorkingDir != "/work" || req.IdleTimeout != DefaultStreamIdleTimeout {
		t.Fatalf("request = %#v", req)
	}
	wantArgs := []string{"CODEX_AGENT_TAG=run-123", "/opt/bin/codex", "exec", "--json"}
	if len(req.Args) != len(wantArgs) {
		t.Fatalf("args = %#v, want %#v", req.Args, wantArgs)
	}
	for i := range wantArgs {
		if req.Args[i] != wantArgs[i] {
			t.Fatalf("args = %#v, want %#v", req.Args, wantArgs)
		}
	}
	got, err := io.ReadAll(req.Stdin)
	if err != nil || string(got) != "solve this" {
		t.Fatalf("stdin = %q, err=%v", got, err)
	}
}

func TestBuildEnvWrappedLaunchRequestClosesEmptyPromptImmediately(t *testing.T) {
	req := BuildEnvWrappedLaunchRequest("TAG", "agent", nil, "id", "", nil, "")
	if req.Stdin != nil {
		t.Fatal("empty prompt must leave stdin nil so the launcher closes it")
	}
	if req.IdleTimeout != DefaultStreamIdleTimeout {
		t.Fatalf("idle timeout = %s", req.IdleTimeout)
	}
}
