package runtimehandlers

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

// operatorPaneCgroup is a live web-console operator pane's membership
// (captured 2026-11-07); agentscope's caller tests cover the classification.
const operatorPaneCgroup = "0::/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/tmux-spawn-a21c3737-0e07-4132-83a7-a22e5fff677b.scope\n"

func withCaller(t *testing.T, kind cliutil.CallerKind, cgroup string) {
	t.Helper()
	oldKind, oldCgroup := callerKindFn, callerCgroupFn
	t.Cleanup(func() { callerKindFn, callerCgroupFn = oldKind, oldCgroup })
	callerKindFn = func() cliutil.CallerKind { return kind }
	callerCgroupFn = func() string { return cgroup }
}

func TestRebaselineRefusesAgentSessionsBeforeAnythingElse(t *testing.T) {
	withCaller(t, cliutil.CallerKindExternalAgent, "")
	resolved := false
	var stdout, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), []string{"runtime", "update", "rebaseline", "/candidate"}, os.Stdin, &stdout, &stderr, func() (string, error) {
		resolved = true
		return "/source", nil
	})
	if code != 1 || resolved || !strings.Contains(stderr.String(), "operator decision") || !strings.Contains(stderr.String(), "Ask the operator") {
		t.Fatalf("an agent session reached the transition: code=%d resolved=%t %q", code, resolved, stderr.String())
	}
}

func TestRebaselineNeedsAnInteractiveTerminal(t *testing.T) {
	withCaller(t, cliutil.CallerKindUnknown, operatorPaneCgroup)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	_, _ = writer.WriteString("y\n")
	_ = writer.Close()
	resolved := false
	var stdout, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), []string{"runtime", "update", "rebaseline", "/candidate"}, reader, &stdout, &stderr, func() (string, error) {
		resolved = true
		return "/source", nil
	})
	if code != 1 || resolved || stdout.Len() != 0 || !strings.Contains(stderr.String(), "interactive terminal") {
		t.Fatalf("piped confirmation was accepted: code=%d resolved=%t stdout=%q stderr=%q", code, resolved, stdout.String(), stderr.String())
	}
	stderr.Reset()
	if code := runRootRebaseline(context.Background(), []string{"runtime", "update", "rebaseline"}, reader, &stdout, &stderr, nil); code != 2 || !strings.Contains(stderr.String(), "rebaseline CANDIDATE") {
		t.Fatalf("missing candidate was not a usage error: %d %q", code, stderr.String())
	}
}

func TestRebaselineIsHandledInProcessNotByTheInstalledCoordinator(t *testing.T) {
	var stdout, stderr bytes.Buffer
	handled, code := RunRootUpdateEffectCommand(context.Background(), []string{"runtime", "update", "rebaseline"}, &stdout, &stderr, nil)
	if !handled || code != 2 || !strings.Contains(stderr.String(), "rebaseline CANDIDATE") {
		t.Fatalf("rebaseline did not reach its own handler: %t %d %q", handled, code, stderr.String())
	}
}

func TestRebaselineSourceIsPinnedToTheCheckoutItRunsFrom(t *testing.T) {
	checkout, other := t.TempDir(), t.TempDir()
	old := repoRootFromWorkingDirectory
	t.Cleanup(func() { repoRootFromWorkingDirectory = old })
	repoRootFromWorkingDirectory = func() (string, error) { return checkout, nil }
	candidate := checkout + "/.vrooli/build/vrooli"
	if got, err := pinnedRebaselineSource(func() (string, error) { return checkout, nil }, candidate); err != nil || got != canonicalPath(checkout) {
		t.Fatalf("matching checkout = %q, %v", got, err)
	}
	if _, err := pinnedRebaselineSource(func() (string, error) { return other, nil }, candidate); err == nil || !strings.Contains(err.Error(), "override") {
		t.Fatalf("a disagreeing source root override was accepted: %v", err)
	}
	if _, err := pinnedRebaselineSource(func() (string, error) { return checkout, nil }, other+"/vrooli"); err == nil || !strings.Contains(err.Error(), "not built in the checkout") {
		t.Fatalf("a candidate from another checkout was accepted: %v", err)
	}
}

func TestEnrollmentAndRetryRefuseAgentSessionsBeforeReachingTheWatchdog(t *testing.T) {
	withCaller(t, cliutil.CallerKindUnknown, "0::/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/vrooli-agent-claude-code-946a880f99ff35ffbb4a0f3e77e3b25c.scope\n")
	for _, args := range [][]string{
		{"runtime", "update", "enroll"},
		{"runtime", "update", "disable", "--json"},
		{"runtime", "update", "retry", "/candidate"},
	} {
		resolved := false
		var stdout, stderr bytes.Buffer
		handled, code := RunRootUpdateEffectCommand(context.Background(), args, &stdout, &stderr, func() (string, error) {
			resolved = true
			return "/source", nil
		})
		if !handled || code != 1 || resolved || !strings.Contains(stderr.String(), "operator decision") || !strings.Contains(stderr.String(), "vrooli-agent-claude-code") {
			t.Fatalf("%v from an agent session was not refused first: handled=%t code=%d resolved=%t %q", args, handled, code, resolved, stderr.String())
		}
	}
}
