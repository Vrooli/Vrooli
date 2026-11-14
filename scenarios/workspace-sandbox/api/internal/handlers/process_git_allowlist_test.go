// Tests for the protected-mode git allowlist enforcement on the /exec
// handler. See execute/protected-sandbox-git-and-network-guardrails.

package handlers

import (
	"strings"
	"testing"

	"workspace-sandbox/internal/runtime"
	"workspace-sandbox/internal/types"
)

// noMsgs is the zero-value GitDenyMessages — fall back to hardcoded defaults.
var noMsgs = runtime.GitDenyMessages{}

func TestEvaluateProtectedGitAllowlist_EmptyAllowlistSkipsEnforcement(t *testing.T) {
	cfg := types.ProtectedConfig{}
	if reason := runtime.EvaluateProtectedGitAllowlist(cfg, noMsgs, "git", []string{"commit", "-m", "x"}); reason != "" {
		t.Fatalf("expected no enforcement when allowlist is empty, got %q", reason)
	}
}

func TestEvaluateProtectedGitAllowlist_WildcardSkipsEnforcement(t *testing.T) {
	cfg := types.ProtectedConfig{GitAllowlist: []string{"*"}}
	if reason := runtime.EvaluateProtectedGitAllowlist(cfg, noMsgs, "git", []string{"commit"}); reason != "" {
		t.Fatalf("expected wildcard to allow any verb, got %q", reason)
	}
}

func TestEvaluateProtectedGitAllowlist_AllowsListedVerb(t *testing.T) {
	cfg := types.ProtectedConfig{GitAllowlist: types.DefaultProtectedGitAllowlist()}
	for _, verb := range []string{"status", "diff", "log", "show", "rev-parse"} {
		if reason := runtime.EvaluateProtectedGitAllowlist(cfg, noMsgs, "git", []string{verb}); reason != "" {
			t.Errorf("expected %q to be allowed, got denial %q", verb, reason)
		}
	}
}

func TestEvaluateProtectedGitAllowlist_BlocksMutatingVerbs(t *testing.T) {
	cfg := types.ProtectedConfig{GitAllowlist: types.DefaultProtectedGitAllowlist()}
	for _, verb := range []string{"commit", "branch", "checkout", "reset", "rebase", "merge", "push", "pull", "clean"} {
		reason := runtime.EvaluateProtectedGitAllowlist(cfg, noMsgs, "git", []string{verb, "--force"})
		if reason == "" {
			t.Errorf("expected %q to be blocked under default allowlist, got allowed", verb)
		}
	}
}

func TestEvaluateProtectedGitAllowlist_OnlyAppliesToGit(t *testing.T) {
	cfg := types.ProtectedConfig{GitAllowlist: []string{"status"}}
	if reason := runtime.EvaluateProtectedGitAllowlist(cfg, noMsgs, "ls", []string{"-la"}); reason != "" {
		t.Fatalf("expected non-git command to bypass allowlist, got %q", reason)
	}
	if reason := runtime.EvaluateProtectedGitAllowlist(cfg, noMsgs, "/usr/bin/python", []string{"-c", "print('hi')"}); reason != "" {
		t.Fatalf("expected non-git command to bypass allowlist, got %q", reason)
	}
}

func TestEvaluateProtectedGitAllowlist_HonorsAbsolutePathToGit(t *testing.T) {
	// /usr/bin/git status should be treated identically to `git status`.
	cfg := types.ProtectedConfig{GitAllowlist: []string{"status"}}
	if reason := runtime.EvaluateProtectedGitAllowlist(cfg, noMsgs, "/usr/bin/git", []string{"status"}); reason != "" {
		t.Fatalf("expected /usr/bin/git status to be allowed, got %q", reason)
	}
	if reason := runtime.EvaluateProtectedGitAllowlist(cfg, noMsgs, "/usr/bin/git", []string{"commit"}); reason == "" {
		t.Fatal("expected /usr/bin/git commit to be blocked under allowlist=[status]")
	}
}

func TestEvaluateProtectedGitAllowlist_RejectsBareGit(t *testing.T) {
	// `git` with no verb is meaningful only as a CLI usage prompt; deny so
	// agents don't probe the boundary by invoking interactive git.
	cfg := types.ProtectedConfig{GitAllowlist: types.DefaultProtectedGitAllowlist()}
	if reason := runtime.EvaluateProtectedGitAllowlist(cfg, noMsgs, "git", nil); reason == "" {
		t.Fatal("expected bare `git` to be denied when allowlist is configured")
	}
}

func TestEvaluateProtectedGitAllowlist_UsesCustomBlockedTemplate(t *testing.T) {
	cfg := types.ProtectedConfig{GitAllowlist: []string{"status"}}
	msgs := runtime.GitDenyMessages{
		Blocked: "custom: verb={verb} allowlist={allowlist}",
	}
	got := runtime.EvaluateProtectedGitAllowlist(cfg, msgs, "git", []string{"commit"})
	want := "custom: verb=commit allowlist=status"
	if got != want {
		t.Fatalf("custom template: got %q, want %q", got, want)
	}
}

func TestEvaluateProtectedGitAllowlist_UsesCustomNoVerbTemplate(t *testing.T) {
	cfg := types.ProtectedConfig{GitAllowlist: []string{"status", "log"}}
	msgs := runtime.GitDenyMessages{
		NoVerb: "no-verb-custom allowlist={allowlist}",
	}
	got := runtime.EvaluateProtectedGitAllowlist(cfg, msgs, "git", nil)
	want := "no-verb-custom allowlist=status, log"
	if got != want {
		t.Fatalf("custom no-verb template: got %q, want %q", got, want)
	}
}

func TestEvaluateProtectedGitAllowlist_DefaultBlockedTemplateMentionsAllowlist(t *testing.T) {
	cfg := types.ProtectedConfig{GitAllowlist: types.DefaultProtectedGitAllowlist()}
	got := runtime.EvaluateProtectedGitAllowlist(cfg, noMsgs, "git", []string{"reset", "--hard"})
	if !strings.Contains(got, "status, diff, log, show, rev-parse") {
		t.Fatalf("default blocked template must mention the allowlist; got %q", got)
	}
	if !strings.Contains(got, "\"reset\"") {
		t.Fatalf("default blocked template must mention the rejected verb; got %q", got)
	}
}
