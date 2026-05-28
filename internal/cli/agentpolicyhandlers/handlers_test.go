package agentpolicyhandlers

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/agentpolicycli"
)

type testCtx struct{}

func newDeps(installed map[string]bool, run func(binary string, args []string) (string, error)) (HandlerDeps[*testCtx], *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	return HandlerDeps[*testCtx]{
		Stdout: func(*testCtx) io.Writer { return stdout },
		Stderr: func(*testCtx) io.Writer { return stdout },
		ResolveBinary: func(name string) (string, bool) {
			if installed[name] {
				return "/fake/" + name, true
			}
			return "", false
		},
		RunCommand: run,
		Agents:     agentpolicycli.SupportedAgents,
	}, stdout
}

func TestFanOutAllInstalledSuccess(t *testing.T) {
	calls := map[string][]string{}
	deps, stdout := newDeps(
		map[string]bool{"resource-claude-code": true, "resource-codex": true, "resource-opencode": true},
		func(binary string, args []string) (string, error) {
			calls[binary] = args
			return "ok\n", nil
		},
	)
	h := RootHandler(deps)
	ctx := &testCtx{}
	if err := h(ctx, []string{"deny", "git stash *"}); err != nil {
		t.Fatalf("Handler: %v", err)
	}
	if len(calls) != 3 {
		t.Errorf("expected 3 fan-out calls, got %d (%v)", len(calls), calls)
	}
	for binary, args := range calls {
		if len(args) < 3 || args[0] != "permissions" || args[1] != "deny" || args[2] != "git stash *" {
			t.Errorf("%s called with wrong args: %v", binary, args)
		}
	}
	if !strings.Contains(stdout.String(), "==> resource-claude-code") {
		t.Errorf("stdout missing per-agent header: %s", stdout.String())
	}
}

func TestFanOutSkipsUninstalled(t *testing.T) {
	deps, stdout := newDeps(
		map[string]bool{"resource-claude-code": true},
		func(binary string, args []string) (string, error) { return "ok", nil },
	)
	h := RootHandler(deps)
	if err := h(&testCtx{}, []string{"deny", "x"}); err != nil {
		t.Fatalf("Handler: %v", err)
	}
	if !strings.Contains(stdout.String(), "not-installed") {
		t.Errorf("expected not-installed marker for missing agents: %s", stdout.String())
	}
}

func TestFanOutNoneInstalledErrors(t *testing.T) {
	deps, _ := newDeps(map[string]bool{}, func(binary string, args []string) (string, error) { return "", nil })
	h := RootHandler(deps)
	err := h(&testCtx{}, []string{"deny", "x"})
	if err == nil || !strings.Contains(err.Error(), "no coding-agent resources installed") {
		t.Fatalf("expected no-resources error, got %v", err)
	}
}

func TestFanOutReportsFailures(t *testing.T) {
	deps, _ := newDeps(
		map[string]bool{"resource-claude-code": true, "resource-codex": true},
		func(binary string, args []string) (string, error) {
			if strings.Contains(binary, "codex") {
				return "boom", errors.New("exit status 1")
			}
			return "ok", nil
		},
	)
	h := RootHandler(deps)
	err := h(&testCtx{}, []string{"deny", "x"})
	if err == nil || !strings.Contains(err.Error(), "1 of 2 coding-agent resources reported failure") {
		t.Fatalf("expected failure aggregation, got %v", err)
	}
}

func TestPassesOverrideFlagThrough(t *testing.T) {
	var seenArgs []string
	deps, _ := newDeps(
		map[string]bool{"resource-claude-code": true},
		func(binary string, args []string) (string, error) {
			seenArgs = args
			return "", nil
		},
	)
	h := RootHandler(deps)
	if err := h(&testCtx{}, []string{"deny", "--i-was-explicitly-authorized", "Bash(git stash*)"}); err != nil {
		t.Fatalf("Handler: %v", err)
	}
	joined := strings.Join(seenArgs, " ")
	if !strings.Contains(joined, "--i-was-explicitly-authorized") {
		t.Errorf("override flag should be forwarded verbatim: %v", seenArgs)
	}
	if !strings.Contains(joined, "Bash(git stash*)") {
		t.Errorf("pattern should be forwarded verbatim: %v", seenArgs)
	}
}

func TestUnknownVerbHelp(t *testing.T) {
	deps, stdout := newDeps(map[string]bool{}, func(binary string, args []string) (string, error) { return "", nil })
	h := RootHandler(deps)
	err := h(&testCtx{}, []string{})
	// No args → help/usage path, no error.
	if err != nil {
		t.Fatalf("expected help path, got error: %v", err)
	}
	if !strings.Contains(stdout.String(), "agent-policy") {
		t.Errorf("expected help text in stdout: %s", stdout.String())
	}
}
