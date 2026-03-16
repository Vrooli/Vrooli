package policy

import (
	"context"
	"strings"
	"testing"
	"time"

	"workspace-sandbox/internal/types"

	"github.com/google/uuid"
)

// =============================================================================
// TEARDOWN POLICY TESTS
//
// These tests verify that the teardown policy correctly runs pre-teardown hooks
// before sandbox unmount/delete operations. The teardown policy enables external
// systems (e.g., the Vrooli scenario lifecycle) to evacuate processes from the
// sandbox's merged directory before the filesystem disappears.
//
// Key invariant: teardown hooks are ALWAYS best-effort. Failures must be
// recorded in results but must never block teardown.
// =============================================================================

func testSandboxWithDir(t *testing.T) *types.Sandbox {
	t.Helper()
	dir := t.TempDir()
	return &types.Sandbox{
		ID:          uuid.New(),
		ScopePath:   "/project/scenarios/test-scenario",
		ProjectRoot: dir,
		UpperDir:    dir + "/upper",
		MergedDir:   dir, // Use the temp dir itself as merged so cmd.Dir works
		Status:      types.StatusActive,
	}
}

// TestNoOpTeardownPolicy_ReturnsEmpty verifies that the no-op policy
// (used when no hooks are configured) returns nil without doing anything.
func TestNoOpTeardownPolicy_ReturnsEmpty(t *testing.T) {
	p := NewNoOpTeardownPolicy()
	results := p.RunPreTeardownHooks(context.Background(), testSandboxWithDir(t), "delete")
	if results != nil {
		t.Errorf("expected nil results from no-op policy, got %v", results)
	}
}

// TestHookTeardownPolicy_SuccessfulHook verifies that a hook returning exit
// code 0 is recorded as successful in the results.
func TestHookTeardownPolicy_SuccessfulHook(t *testing.T) {
	hooks := []TeardownHook{{
		Name:    "success-hook",
		Command: "/bin/true",
	}}
	p := NewHookTeardownPolicy(hooks)

	results := p.RunPreTeardownHooks(context.Background(), testSandboxWithDir(t), "delete")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Errorf("expected success, got failure: %v", results[0].Error)
	}
	if results[0].HookName != "success-hook" {
		t.Errorf("expected hook name 'success-hook', got %q", results[0].HookName)
	}
}

// TestHookTeardownPolicy_FailingHook_DoesNotBlock verifies that a failing hook
// is recorded as failed but does NOT cause RunPreTeardownHooks to return an error.
// This is the critical invariant: teardown must never be blocked by hook failures.
func TestHookTeardownPolicy_FailingHook_DoesNotBlock(t *testing.T) {
	hooks := []TeardownHook{{
		Name:    "failing-hook",
		Command: "/bin/false",
	}}
	p := NewHookTeardownPolicy(hooks)

	results := p.RunPreTeardownHooks(context.Background(), testSandboxWithDir(t), "stop")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("expected failure result for /bin/false hook")
	}
	if results[0].Error == nil {
		t.Error("expected non-nil error for failed hook")
	}
}

// TestHookTeardownPolicy_Timeout verifies that hooks respect the global timeout.
// A hook that would run forever is cancelled and recorded as failed.
func TestHookTeardownPolicy_Timeout(t *testing.T) {
	hooks := []TeardownHook{{
		Name:    "slow-hook",
		Command: "sleep",
		Args:    []string{"60"},
	}}
	p := NewHookTeardownPolicy(hooks,
		WithTeardownGlobalTimeout(500*time.Millisecond),
	)

	start := time.Now()
	results := p.RunPreTeardownHooks(context.Background(), testSandboxWithDir(t), "delete")
	elapsed := time.Since(start)

	if elapsed > 5*time.Second {
		t.Errorf("hook took %v, expected timeout around 500ms", elapsed)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("expected timeout failure")
	}
}

// TestHookTeardownPolicy_EnvVars verifies that hooks receive the correct
// environment variables including SANDBOX_MERGED_DIR and SANDBOX_TEARDOWN_REASON.
func TestHookTeardownPolicy_EnvVars(t *testing.T) {
	hooks := []TeardownHook{{
		Name:    "env-check",
		Command: "env",
	}}
	p := NewHookTeardownPolicy(hooks)

	sbx := testSandboxWithDir(t)
	results := p.RunPreTeardownHooks(context.Background(), sbx, "delete")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Fatalf("env hook failed: %v", results[0].Error)
	}

	output := results[0].Output
	if !strings.Contains(output, "SANDBOX_MERGED_DIR="+sbx.MergedDir) {
		t.Errorf("missing SANDBOX_MERGED_DIR in env output:\n%s", output)
	}
	if !strings.Contains(output, "SANDBOX_TEARDOWN_REASON=delete") {
		t.Errorf("missing SANDBOX_TEARDOWN_REASON in env output:\n%s", output)
	}
	if !strings.Contains(output, "SANDBOX_ID="+sbx.ID.String()) {
		t.Errorf("missing SANDBOX_ID in env output:\n%s", output)
	}
}

// TestHookTeardownPolicy_MultipleHooks verifies that all hooks execute even
// when one fails. Order and independence are preserved.
func TestHookTeardownPolicy_MultipleHooks(t *testing.T) {
	hooks := []TeardownHook{
		{Name: "first-succeeds", Command: "/bin/true"},
		{Name: "second-fails", Command: "/bin/false"},
		{Name: "third-succeeds", Command: "/bin/true"},
	}
	p := NewHookTeardownPolicy(hooks)

	results := p.RunPreTeardownHooks(context.Background(), testSandboxWithDir(t), "delete")
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if !results[0].Success {
		t.Error("first hook should succeed")
	}
	if results[1].Success {
		t.Error("second hook should fail")
	}
	if !results[2].Success {
		t.Error("third hook should succeed (not blocked by second's failure)")
	}
}
