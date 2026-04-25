package agentmanager

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"

	"swarm-manager/internal/projectroot"
)

func setupRepoRootEnv(t *testing.T) string {
	t.Helper()
	root, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromCWD: %v", err)
	}
	t.Setenv("VROOLI_ROOT", root)
	return root
}

func TestResolveScopeAndRoot_BothEmptyUsesResolver(t *testing.T) {
	repoRoot := setupRepoRootEnv(t)

	scope, root, err := resolveScopeAndRoot("", "", []string{
		"scenarios/swarm-manager/api/**",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != repoRoot {
		t.Errorf("root = %q, want %q", root, repoRoot)
	}
	if scope != "scenarios/swarm-manager" {
		t.Errorf("scope = %q, want %q", scope, "scenarios/swarm-manager")
	}
}

func TestResolveScopeAndRoot_DotsTreatedAsEmpty(t *testing.T) {
	repoRoot := setupRepoRootEnv(t)

	scope, root, err := resolveScopeAndRoot(".", ".", []string{
		"scenarios/swarm-manager/api/**",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != repoRoot {
		t.Errorf("root = %q, want %q (repo root, not literal dot)", root, repoRoot)
	}
	if scope == "." {
		t.Errorf("scope unexpectedly stayed as dot; resolver should have replaced it")
	}
}

func TestResolveScopeAndRoot_ExplicitOverridesPreserved(t *testing.T) {
	setupRepoRootEnv(t)

	scope, root, err := resolveScopeAndRoot("custom/scope/path", "/custom/abs/root", []string{
		"scenarios/swarm-manager/api/**",
	})
	if err == nil {
		// Validation should fire because /custom/abs/root is absolute but
		// scenarios/swarm-manager will not exist under it. Either we get
		// the override values back (root = /custom/abs/root) and an error
		// from validation, or no error and the override values. Tolerate
		// both shapes; this test only cares that overrides aren't ignored.
		if root != "/custom/abs/root" {
			t.Errorf("root = %q, want override %q", root, "/custom/abs/root")
		}
		if scope != "custom/scope/path" {
			t.Errorf("scope = %q, want override %q", scope, "custom/scope/path")
		}
		return
	}
	// Error path: must be acceptance mismatch (validation fired against the
	// bogus override root). Anything else means the override was discarded.
	if !errors.Is(err, projectroot.ErrAcceptanceMismatch) {
		t.Errorf("expected ErrAcceptanceMismatch from validation, got %v", err)
	}
}

func TestResolveScopeAndRoot_PartialOverride(t *testing.T) {
	repoRoot := setupRepoRootEnv(t)

	// Caller fixes scope but lets root be derived.
	scope, root, err := resolveScopeAndRoot("custom/scope", "", []string{
		"scenarios/swarm-manager/api/**",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope != "custom/scope" {
		t.Errorf("scope = %q, want explicit override %q", scope, "custom/scope")
	}
	if root != repoRoot {
		t.Errorf("root = %q, want resolved %q", root, repoRoot)
	}
}

func TestResolveScopeAndRoot_ValidationRejectsMissingPath(t *testing.T) {
	setupRepoRootEnv(t)

	_, _, err := resolveScopeAndRoot("", "", []string{
		"scenarios/this-scenario-does-not-exist-anywhere/api/**",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !errors.Is(err, projectroot.ErrAcceptanceMismatch) {
		t.Errorf("expected ErrAcceptanceMismatch, got %v", err)
	}
}

func TestResolveScopeAndRoot_AmbiguousScenariosErrors(t *testing.T) {
	setupRepoRootEnv(t)

	_, _, err := resolveScopeAndRoot("", "", []string{
		"scenarios/foo/cli/**",
		"scenarios/bar/api/**",
	})
	if err == nil {
		t.Fatal("expected ambiguous-scenarios error, got nil")
	}
	if !errors.Is(err, projectroot.ErrAmbiguousScenarios) {
		t.Errorf("expected ErrAmbiguousScenarios, got %v", err)
	}
}

func TestResolveScopeAndRoot_NoAcceptanceFallsBackToWideScope(t *testing.T) {
	repoRoot := setupRepoRootEnv(t)

	scope, root, err := resolveScopeAndRoot("", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != repoRoot {
		t.Errorf("root = %q, want %q", root, repoRoot)
	}
	if scope != "." {
		t.Errorf("scope = %q, want \".\" for wide-scope fallback", scope)
	}
}

func TestResolveScopeAndRoot_AbsoluteOverrideValidated(t *testing.T) {
	repoRoot := setupRepoRootEnv(t)

	// Use the real repo root as the override; acceptance globs should
	// validate cleanly against it.
	_, _, err := resolveScopeAndRoot("scenarios/swarm-manager", repoRoot, []string{
		"scenarios/swarm-manager/api/**",
	})
	if err != nil {
		t.Errorf("unexpected error for valid override: %v", err)
	}
}

func TestIsEmptyOrDot(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", true},
		{" ", true},
		{".", true},
		{" . ", true},
		{"./", false},
		{"foo", false},
		{"/abs", false},
		{"scenarios/foo", false},
	}
	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.in, " ", "_"), func(t *testing.T) {
			got := isEmptyOrDot(tt.in)
			if got != tt.want {
				t.Errorf("isEmptyOrDot(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveScopeAndRoot_RelativeOverrideRootSkipsValidation(t *testing.T) {
	setupRepoRootEnv(t)

	// A relative ProjectRoot override (rare) should not trigger acceptance
	// validation, since we cannot stat paths under a relative root reliably
	// without context. We just pass it through.
	scope, root, err := resolveScopeAndRoot("scope", "relative/root", []string{
		"scenarios/nonexistent-anywhere/**",
	})
	if err != nil {
		t.Errorf("relative override should skip validation, got error: %v", err)
	}
	if !filepath.IsAbs(root) && root != "relative/root" {
		t.Errorf("root = %q, want override preserved", root)
	}
	if scope != "scope" {
		t.Errorf("scope = %q, want override preserved", scope)
	}
}
