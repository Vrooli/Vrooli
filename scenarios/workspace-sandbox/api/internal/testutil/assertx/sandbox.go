// Package assertx contains domain-aware assertions that produce
// useful diff output when they fail.
package assertx

import (
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// AssertStatus fails the test if got.Status != want.
func AssertStatus(t *testing.T, got *types.Sandbox, want types.Status) {
	t.Helper()
	if got == nil {
		t.Fatalf("AssertStatus: sandbox is nil (want %v)", want)
	}
	if got.Status != want {
		t.Errorf("AssertStatus: got %v, want %v (sandbox %s)", got.Status, want, got.ID)
	}
}

// AssertHomeOverlayState fails if the sandbox's home-overlay state
// does not match `want`.
func AssertHomeOverlayState(t *testing.T, got *types.Sandbox, want types.HomeOverlayState) {
	t.Helper()
	if got == nil {
		t.Fatalf("AssertHomeOverlayState: sandbox is nil (want %v)", want)
	}
	if got.HomeOverlayState != want {
		t.Errorf("AssertHomeOverlayState: got %v, want %v (sandbox %s)", got.HomeOverlayState, want, got.ID)
	}
}

// AssertSandboxID fails if the sandbox's ID is not the expected one.
func AssertSandboxID(t *testing.T, got *types.Sandbox, want uuid.UUID) {
	t.Helper()
	if got == nil {
		t.Fatalf("AssertSandboxID: sandbox is nil (want %s)", want)
	}
	if got.ID != want {
		t.Errorf("AssertSandboxID: got %s, want %s", got.ID, want)
	}
}
