// Package sandbox provides tests that document and verify key system assumptions.
//
// # Purpose
//
// These tests serve as executable documentation of the assumptions made
// throughout the workspace-sandbox system. When an assumption is violated,
// these tests should fail with a clear message explaining what was expected.
//
// # Categories
//
//   - State invariants: What must be true for each status
//   - Precondition guards: What inputs are validated and how
//   - Data shape requirements: What fields must be set at each lifecycle stage
package sandbox

import (
	"testing"

	"workspace-sandbox/internal/types"
)

// --- Status State Machine Assumptions ---

// TestAssumption_OnlyActiveSandboxesHaveMountedOverlays documents the assumption
// that MergedDir is only valid for Active sandboxes.
//
// ASSUMPTION: Only Active sandboxes have a mounted overlay filesystem.
// WHY: The overlay is unmounted when a sandbox is stopped to free resources.
// FAILURE MODE: If MergedDir is accessed for non-active sandbox, agents will
// get errors or work on stale/missing data.
func TestAssumption_OnlyActiveSandboxesHaveMountedOverlays(t *testing.T) {
	activeStatuses := []types.Status{types.StatusCreating, types.StatusActive}
	for _, status := range activeStatuses {
		if !status.IsActive() {
			t.Errorf("Expected %s to be considered active for overlay purposes", status)
		}
	}

	inactiveStatuses := []types.Status{
		types.StatusStopped, types.StatusApproved,
		types.StatusRejected, types.StatusDeleted, types.StatusError,
	}
	for _, status := range inactiveStatuses {
		if status.IsActive() {
			t.Errorf("Expected %s to NOT be considered active", status)
		}
	}
}

// TestAssumption_TerminalStatusesCannotTransition documents that terminal
// statuses are truly final.
//
// ASSUMPTION: Once a sandbox reaches a terminal status, it cannot change.
// WHY: Terminal statuses indicate the sandbox has been finalized (approved,
// rejected, or deleted). Allowing transitions would break audit guarantees.
func TestAssumption_TerminalStatusesCannotTransition(t *testing.T) {
	terminalStatuses := []types.Status{
		types.StatusApproved, types.StatusRejected, types.StatusDeleted,
	}

	for _, status := range terminalStatuses {
		if !status.IsTerminal() {
			t.Errorf("Expected %s to be terminal", status)
		}

		// Verify no valid next transitions exist (except to Deleted)
		for _, nextStatus := range []types.Status{
			types.StatusCreating, types.StatusActive, types.StatusStopped,
		} {
			if types.CanTransitionTo(status, nextStatus) {
				t.Errorf("%s should not be able to transition to %s", status, nextStatus)
			}
		}
	}
}

// TestAssumption_DeleteIsAlwaysAllowed documents that any sandbox can be deleted.
//
// ASSUMPTION: Deletion is always allowed regardless of current status.
// WHY: Cleanup must always be possible to prevent resource leaks.
func TestAssumption_DeleteIsAlwaysAllowed(t *testing.T) {
	allStatuses := []types.Status{
		types.StatusCreating, types.StatusActive, types.StatusStopped,
		types.StatusApproved, types.StatusRejected, types.StatusError,
	}

	for _, status := range allStatuses {
		if !types.CanTransitionTo(status, types.StatusDeleted) {
			t.Errorf("Expected %s to allow transition to Deleted", status)
		}
	}

	// Special case: Deleted cannot transition to Deleted again
	if types.CanTransitionTo(types.StatusDeleted, types.StatusDeleted) {
		t.Error("Deleted should not transition to Deleted (idempotent delete)")
	}
}

// --- Path Overlap Assumptions ---

// TestAssumption_MutualExclusionPreventsOverlap documents the scope overlap rule.
//
// ASSUMPTION: Two sandboxes cannot have scopes where one is ancestor of the other.
// WHY: If sandbox A has scope /project and sandbox B has scope /project/src,
// changes in A could affect B's working area (and vice versa).
func TestAssumption_MutualExclusionPreventsOverlap(t *testing.T) {
	cases := []struct {
		existing    string
		new         string
		shouldBlock bool
		reason      string
	}{
		{"/project", "/project/src", true, "parent scope blocks child scope"},
		{"/project/src", "/project", true, "child scope blocks parent scope"},
		{"/project/src", "/project/src", true, "exact match is blocked"},
		{"/project/src", "/project/tests", false, "sibling scopes are allowed"},
		{"/project1", "/project2", false, "unrelated paths are allowed"},
	}

	for _, tc := range cases {
		overlap := types.CheckPathOverlap(tc.existing, tc.new)
		hasConflict := overlap != ""

		if hasConflict != tc.shouldBlock {
			t.Errorf("%s: existing=%q new=%q: expected blocked=%v but got blocked=%v",
				tc.reason, tc.existing, tc.new, tc.shouldBlock, hasConflict)
		}
	}
}

// TestAssumption_PathNormalizationPreventsAliases documents that paths are
// normalized to prevent aliasing.
//
// ASSUMPTION: Paths are normalized before comparison to prevent bypass via
// different representations of the same path (e.g., /project/../project/src).
// WHY: Without normalization, an attacker or buggy agent could create
// overlapping sandboxes by using path aliases.
func TestAssumption_PathNormalizationPreventsAliases(t *testing.T) {
	// These should all be detected as the same path after normalization
	pathVariants := []struct {
		path1 string
		path2 string
	}{
		{"/project/src", "/project/src"},
		{"/project/./src", "/project/src"},
		{"/project/foo/../src", "/project/src"},
	}

	for _, tc := range pathVariants {
		conflict := types.CheckPathOverlap(tc.path1, tc.path2)
		if conflict != types.ConflictTypeExact {
			t.Errorf("Expected %q and %q to be detected as exact match after normalization, got: %q",
				tc.path1, tc.path2, conflict)
		}
	}
}

// --- Approval Flow Assumptions ---

// TestAssumption_ApprovalRequiresStoppedOrActive documents when approval is valid.
//
// ASSUMPTION: Changes can only be approved from Active or Stopped status.
// WHY: The sandbox must be in a stable state for diff generation and approval.
// Creating sandboxes are still mounting, and terminal sandboxes are finalized.
func TestAssumption_ApprovalRequiresStoppedOrActive(t *testing.T) {
	approvableStatuses := []types.Status{types.StatusActive, types.StatusStopped}
	for _, status := range approvableStatuses {
		if !types.CanTransitionTo(status, types.StatusApproved) {
			t.Errorf("Expected %s to allow transition to Approved", status)
		}
	}

	nonApprovableStatuses := []types.Status{
		types.StatusCreating, types.StatusApproved,
		types.StatusRejected, types.StatusDeleted, types.StatusError,
	}
	for _, status := range nonApprovableStatuses {
		if types.CanTransitionTo(status, types.StatusApproved) {
			t.Errorf("Expected %s to NOT allow transition to Approved", status)
		}
	}
}

// --- Error Type Assumptions ---

// TestAssumption_ErrorTypesHaveHints documents that all error types provide hints.
//
// ASSUMPTION: All custom error types include actionable hints for resolution.
// WHY: Users and agents need to know how to fix errors, not just what went wrong.
func TestAssumption_ErrorTypesHaveHints(t *testing.T) {
	// Test that NotFoundError has a hint method that returns non-empty string
	notFound := types.NewNotFoundError("test-id")
	if notFound.Hint() == "" {
		t.Error("NotFoundError should have a hint")
	}

	// Test that ValidationError can have a hint via GetHint method
	validation := &types.ValidationError{
		Field:   "scopePath",
		Message: "test error",
		Hint:    "test hint",
	}
	if validation.GetHint() == "" {
		t.Error("ValidationError should support hints")
	}

	// Test that ScopeConflictError has a hint method
	scopeConflict := &types.ScopeConflictError{
		Conflicts: []types.PathConflict{{
			ExistingID:    "abc",
			ExistingScope: "/project",
			NewScope:      "/project/src",
			ConflictType:  types.ConflictTypeExistingContainsNew,
		}},
	}
	if scopeConflict.Hint() == "" {
		t.Error("ScopeConflictError should have a hint")
	}
}

// --- Data Shape Assumptions ---

// TestAssumption_SandboxIDsAreStable documents the ID stability guarantee.
//
// ASSUMPTION: Sandbox IDs (UUIDs) remain constant throughout the lifecycle.
// WHY: External systems reference sandboxes by ID. Changing IDs would break
// these references and make audit trails unreliable.
func TestAssumption_SandboxIDsAreStable(t *testing.T) {
	// This is a documentation test - UUIDs by design don't change
	// The test documents the assumption for future maintainers
	t.Log("ASSUMPTION: Once assigned, a sandbox's UUID never changes")
	t.Log("RATIONALE: External systems (agents, logs, APIs) reference by ID")
	t.Log("VIOLATION IMPACT: Would break audit trails, API consumers, and agent state")
}
