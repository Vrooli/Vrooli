// Package fixtures provides domain-object factories for tests.
//
// Each NewX returns a sane default instance plus a slice of functional
// options for the unusual fields. Default values are picked so that
// the most common test path is `fixtures.NewSandbox(t)` — no opts.
package fixtures

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// SandboxOpt mutates a sandbox during construction. Compose multiple
// opts to build non-default sandboxes.
type SandboxOpt func(*types.Sandbox)

// NewSandbox returns a sandbox in StatusActive with /tmp paths and a
// fresh UUID. The `t` parameter is accepted (rather than ignored) so
// the factory can call t.Helper() and report friendlier errors when
// (e.g.) an option panics.
func NewSandbox(t *testing.T, opts ...SandboxOpt) *types.Sandbox {
	t.Helper()
	now := time.Now()
	s := &types.Sandbox{
		ID:            uuid.New(),
		ScopePath:     "/tmp/project/src",
		ProjectRoot:   "/tmp/project",
		Owner:         "test-user",
		OwnerType:     types.OwnerTypeUser,
		Status:        types.StatusActive,
		DriverID:      "mock",
		DriverVersion: "1.0.0",
		LowerDir:      "/tmp/lower",
		UpperDir:      "/tmp/upper",
		WorkDir:       "/tmp/work",
		MergedDir:     "/tmp/merged",
		CreatedAt:     now,
		LastUsedAt:    now,
		UpdatedAt:     now,
		Version:       1,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// WithSandboxID sets a deterministic ID. Use when a test needs a stable
// fingerprint (e.g., audit-log assertions keyed by SandboxID).
func WithSandboxID(id uuid.UUID) SandboxOpt {
	return func(s *types.Sandbox) { s.ID = id }
}

// WithSandboxStatus overrides the default StatusActive.
func WithSandboxStatus(st types.Status) SandboxOpt {
	return func(s *types.Sandbox) { s.Status = st }
}

// WithSandboxOwner sets Owner and OwnerType.
func WithSandboxOwner(owner string, ot types.OwnerType) SandboxOpt {
	return func(s *types.Sandbox) {
		s.Owner = owner
		s.OwnerType = ot
	}
}

// WithSandboxScope replaces ScopePath and ProjectRoot.
func WithSandboxScope(scopePath, projectRoot string) SandboxOpt {
	return func(s *types.Sandbox) {
		s.ScopePath = scopePath
		s.ProjectRoot = projectRoot
	}
}

// WithSandboxLastUsed overrides LastUsedAt. Test helper for idle/TTL
// reconciler scenarios.
func WithSandboxLastUsed(ts time.Time) SandboxOpt {
	return func(s *types.Sandbox) { s.LastUsedAt = ts }
}

// WithSandboxBehavior installs the given behavior config.
func WithSandboxBehavior(b types.SandboxBehavior) SandboxOpt {
	return func(s *types.Sandbox) { s.Behavior = b }
}

// WithSandboxHomeOverlayState sets the per-sandbox home overlay state.
// Used by handler tests gating exec on HomeOverlayPresent.
func WithSandboxHomeOverlayState(state types.HomeOverlayState) SandboxOpt {
	return func(s *types.Sandbox) { s.HomeOverlayState = state }
}

// WithSandboxIdempotencyKey sets the idempotency key.
func WithSandboxIdempotencyKey(key string) SandboxOpt {
	return func(s *types.Sandbox) { s.IdempotencyKey = key }
}

// WithSandboxBaseCommit sets BaseCommitHash. Conflict-detection tests
// rely on this.
func WithSandboxBaseCommit(hash string) SandboxOpt {
	return func(s *types.Sandbox) { s.BaseCommitHash = hash }
}
