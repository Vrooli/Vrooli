// Tests for the auditability-contract domain primitives added by
// execute/agent-manager-sandbox-auto-apply-defaults Phase 1, then updated
// by execute/protected-sandbox-agent-launch (the runner-fork) to allow
// protected mode at the validation layer.
//
//   - SandboxMode validation (tracking + protected accepted; unknown rejected)
//   - SandboxConfig defaults + AutoApply / ApplyOnFailure resolution
//   - RunOutcome.ToContract mapping (D5)
//   - ResolveConversationID precedence (D7)
//
// See scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md for the
// canonical contract these tests pin to behavior.

package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestDefaultSandboxConfig_LockedDefaults(t *testing.T) {
	cfg := DefaultSandboxConfig()

	// Default flipped to protected by Slice 4 of
	// execute/protected-sandbox-agent-launch — once all three runners
	// (claude_code, codex, opencode) and both Execute and Continue
	// paths route through the launcher seam, the agent process tree
	// itself can safely run inside workspace-sandbox by default.
	if cfg.Mode != SandboxModeProtected {
		t.Errorf("Mode = %q, want %q", cfg.Mode, SandboxModeProtected)
	}
	if cfg.ManualReview {
		t.Error("ManualReview = true, want false (operator-opt-in only)")
	}
	if !cfg.GetAutoApply() {
		t.Error("GetAutoApply() = false, want true (contract default)")
	}
	if !cfg.GetApplyOnFailure() {
		t.Error("GetApplyOnFailure() = false, want true (contract default)")
	}
	if cfg.NetworkMode != NetworkAccessLocalhost {
		t.Errorf("NetworkMode = %q, want %q", cfg.NetworkMode, NetworkAccessLocalhost)
	}
	if !cfg.NoLock {
		t.Error("NoLock = false, want true (lock=false in contract terms)")
	}
}

func TestSandboxConfig_GetAutoApply_NilSafe(t *testing.T) {
	var cfg *SandboxConfig
	if !cfg.GetAutoApply() {
		t.Error("nil receiver GetAutoApply() should default to true (contract)")
	}
	if !cfg.GetApplyOnFailure() {
		t.Error("nil receiver GetApplyOnFailure() should default to true (contract)")
	}
}

func TestSandboxConfig_GetAutoApply_ExplicitFalse(t *testing.T) {
	off := false
	cfg := &SandboxConfig{AutoApply: &off, ApplyOnFailure: &off}
	if cfg.GetAutoApply() {
		t.Error("explicit AutoApply=false should resolve to false")
	}
	if cfg.GetApplyOnFailure() {
		t.Error("explicit ApplyOnFailure=false should resolve to false")
	}
}

func TestSandboxMode_Effective(t *testing.T) {
	if got := SandboxModeUnspecified.Effective(); got != SandboxModeProtected {
		t.Errorf("unspecified.Effective() = %q, want protected", got)
	}
	if got := SandboxModeTracking.Effective(); got != SandboxModeTracking {
		t.Errorf("tracking.Effective() = %q, want tracking", got)
	}
	if got := SandboxModeProtected.Effective(); got != SandboxModeProtected {
		t.Errorf("protected.Effective() = %q, want protected", got)
	}
}

// TestValidateSandboxConfig_ProtectedModeAccepted verifies that protected
// mode is now a valid SandboxMode. The validation gate flipped from
// "reserved" to "allowed" with execute/protected-sandbox-agent-launch —
// the runner-fork that introduced the SandboxLauncher seam. Whether a
// protected-mode run actually launches in the sandbox or falls back to
// the host depends on the runner's SandboxLauncherFactory wiring at
// runtime; that fallback is intentionally NOT enforced at the validation
// layer (validation is about request shape, not deployment topology).
func TestValidateSandboxConfig_ProtectedModeAccepted(t *testing.T) {
	cfg := &SandboxConfig{Mode: SandboxModeProtected}
	if err := validateSandboxConfig(cfg); err != nil {
		t.Fatalf("validateSandboxConfig(mode=protected) returned error %v; want nil", err)
	}
}

func TestValidateSandboxConfig_UnknownMode(t *testing.T) {
	cfg := &SandboxConfig{Mode: SandboxMode("bogus")}
	err := validateSandboxConfig(cfg)
	if err == nil {
		t.Fatal("validateSandboxConfig(mode=bogus) returned nil; want validation error")
	}
}

func TestValidateSandboxConfig_TrackingMode(t *testing.T) {
	cfg := DefaultSandboxConfig()
	if err := validateSandboxConfig(cfg); err != nil {
		t.Fatalf("default config rejected by validation: %v", err)
	}
}

func TestValidateSandboxConfig_InvalidNetworkMode(t *testing.T) {
	cfg := &SandboxConfig{NetworkMode: NetworkAccess("over-9000")}
	err := validateSandboxConfig(cfg)
	if err == nil {
		t.Fatal("validateSandboxConfig(networkMode=invalid) returned nil")
	}
}

func TestRunOutcome_ToContract_MappingTable(t *testing.T) {
	cases := []struct {
		domain RunOutcome
		want   ContractRunOutcome
	}{
		{RunOutcomeSuccess, ContractRunOutcomeSuccess},
		{RunOutcomeCancelled, ContractRunOutcomeCancelled},
		{RunOutcomeTimeout, ContractRunOutcomeTimeout},
		{RunOutcomeExitError, ContractRunOutcomeFailure},
		{RunOutcomeException, ContractRunOutcomeFailure},
		{RunOutcomeSandboxFail, ContractRunOutcomeFailure},
		{RunOutcomeRunnerFail, ContractRunOutcomeFailure},
		{RunOutcome("unknown"), ContractRunOutcomeFailure},
	}
	for _, c := range cases {
		t.Run(string(c.domain), func(t *testing.T) {
			if got := c.domain.ToContract(); got != c.want {
				t.Errorf("RunOutcome(%q).ToContract() = %q, want %q", c.domain, got, c.want)
			}
		})
	}
}

func TestResolveConversationID_SpawnerProvidedWins(t *testing.T) {
	parentID := uuid.New()
	run := &Run{
		ConversationID: "spawner-supplied",
		ParentRunID:    &parentID,
	}
	parentLookup := func(id uuid.UUID) (string, bool) {
		t.Errorf("parentLookup unexpectedly called when spawner supplied a value")
		return "should-not-be-used", true
	}
	if got := ResolveConversationID(run, parentLookup); got != "spawner-supplied" {
		t.Errorf("ResolveConversationID = %q, want spawner-supplied", got)
	}
}

func TestResolveConversationID_InheritsFromParent(t *testing.T) {
	parentID := uuid.New()
	run := &Run{ParentRunID: &parentID}
	parentLookup := func(id uuid.UUID) (string, bool) {
		if id != parentID {
			t.Errorf("parentLookup got id=%v, want %v", id, parentID)
		}
		return "parent-conversation", true
	}
	if got := ResolveConversationID(run, parentLookup); got != "parent-conversation" {
		t.Errorf("ResolveConversationID = %q, want parent-conversation", got)
	}
}

func TestResolveConversationID_ParentNotFoundFallsBackToFreshUUID(t *testing.T) {
	parentID := uuid.New()
	run := &Run{ParentRunID: &parentID}
	parentLookup := func(id uuid.UUID) (string, bool) {
		return "", false
	}
	got := ResolveConversationID(run, parentLookup)
	if got == "" {
		t.Error("ResolveConversationID returned empty when fallback should have generated UUID")
	}
	if _, err := uuid.Parse(got); err != nil {
		t.Errorf("fresh value %q is not a valid UUID: %v", got, err)
	}
}

func TestResolveConversationID_NoParentGeneratesFreshUUID(t *testing.T) {
	run := &Run{}
	got := ResolveConversationID(run, nil)
	if got == "" {
		t.Error("ResolveConversationID returned empty for standalone run")
	}
	if _, err := uuid.Parse(got); err != nil {
		t.Errorf("fresh value %q is not a valid UUID: %v", got, err)
	}
}

func TestResolveConversationID_ParentEmptyFallsBackToFreshUUID(t *testing.T) {
	parentID := uuid.New()
	run := &Run{ParentRunID: &parentID}
	parentLookup := func(id uuid.UUID) (string, bool) {
		return "", true // found, but empty — treat as missing
	}
	got := ResolveConversationID(run, parentLookup)
	if _, err := uuid.Parse(got); err != nil {
		t.Errorf("fresh value %q is not a valid UUID: %v", got, err)
	}
}
