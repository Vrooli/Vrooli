// Tests for the protected-mode git allowlist enforcement on the /exec
// handler. See execute/protected-sandbox-git-and-network-guardrails.

package handlers

import (
	"testing"

	"workspace-sandbox/internal/types"
)

func TestEvaluateProtectedGitAllowlist_EmptyAllowlistSkipsEnforcement(t *testing.T) {
	cfg := types.ProtectedConfig{}
	if reason := evaluateProtectedGitAllowlist(cfg, "git", []string{"commit", "-m", "x"}); reason != "" {
		t.Fatalf("expected no enforcement when allowlist is empty, got %q", reason)
	}
}

func TestEvaluateProtectedGitAllowlist_WildcardSkipsEnforcement(t *testing.T) {
	cfg := types.ProtectedConfig{GitAllowlist: []string{"*"}}
	if reason := evaluateProtectedGitAllowlist(cfg, "git", []string{"commit"}); reason != "" {
		t.Fatalf("expected wildcard to allow any verb, got %q", reason)
	}
}

func TestEvaluateProtectedGitAllowlist_AllowsListedVerb(t *testing.T) {
	cfg := types.ProtectedConfig{GitAllowlist: types.DefaultProtectedGitAllowlist()}
	for _, verb := range []string{"status", "diff", "log", "show", "rev-parse"} {
		if reason := evaluateProtectedGitAllowlist(cfg, "git", []string{verb}); reason != "" {
			t.Errorf("expected %q to be allowed, got denial %q", verb, reason)
		}
	}
}

func TestEvaluateProtectedGitAllowlist_BlocksMutatingVerbs(t *testing.T) {
	cfg := types.ProtectedConfig{GitAllowlist: types.DefaultProtectedGitAllowlist()}
	for _, verb := range []string{"commit", "branch", "checkout", "reset", "rebase", "merge", "push", "pull", "clean"} {
		reason := evaluateProtectedGitAllowlist(cfg, "git", []string{verb, "--force"})
		if reason == "" {
			t.Errorf("expected %q to be blocked under default allowlist, got allowed", verb)
		}
	}
}

func TestEvaluateProtectedGitAllowlist_OnlyAppliesToGit(t *testing.T) {
	cfg := types.ProtectedConfig{GitAllowlist: []string{"status"}}
	if reason := evaluateProtectedGitAllowlist(cfg, "ls", []string{"-la"}); reason != "" {
		t.Fatalf("expected non-git command to bypass allowlist, got %q", reason)
	}
	if reason := evaluateProtectedGitAllowlist(cfg, "/usr/bin/python", []string{"-c", "print('hi')"}); reason != "" {
		t.Fatalf("expected non-git command to bypass allowlist, got %q", reason)
	}
}

func TestEvaluateProtectedGitAllowlist_HonorsAbsolutePathToGit(t *testing.T) {
	// /usr/bin/git status should be treated identically to `git status`.
	cfg := types.ProtectedConfig{GitAllowlist: []string{"status"}}
	if reason := evaluateProtectedGitAllowlist(cfg, "/usr/bin/git", []string{"status"}); reason != "" {
		t.Fatalf("expected /usr/bin/git status to be allowed, got %q", reason)
	}
	if reason := evaluateProtectedGitAllowlist(cfg, "/usr/bin/git", []string{"commit"}); reason == "" {
		t.Fatal("expected /usr/bin/git commit to be blocked under allowlist=[status]")
	}
}

func TestEvaluateProtectedGitAllowlist_RejectsBareGit(t *testing.T) {
	// `git` with no verb is meaningful only as a CLI usage prompt; deny so
	// agents don't probe the boundary by invoking interactive git.
	cfg := types.ProtectedConfig{GitAllowlist: types.DefaultProtectedGitAllowlist()}
	if reason := evaluateProtectedGitAllowlist(cfg, "git", nil); reason == "" {
		t.Fatal("expected bare `git` to be denied when allowlist is configured")
	}
}
