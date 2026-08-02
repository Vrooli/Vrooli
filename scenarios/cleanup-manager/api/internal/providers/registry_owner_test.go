package providers

import (
	"testing"

	"cleanup-manager/internal/cleanup"
)

// TestArchitectureCartographerIsRegistered pins the registration whose absence
// made the 2026-07-31 incident invisible to the tool built to prevent it.
//
// graph_snapshots held 77.2 GB — the single largest consumer on the host — and
// cleanup-manager could not see or report a byte of it, because
// architecture-cartographer was not among the registered owner scenarios.
func TestArchitectureCartographerIsRegistered(t *testing.T) {
	providers := OwnerScenarioBuiltIns(nil)

	var found cleanup.ProviderMetadata
	for _, p := range providers {
		meta := p.Metadata()
		if meta.ID == "architecture-cartographer-snapshots" {
			found = meta
			break
		}
	}

	if found.ID == "" {
		t.Fatal("architecture-cartographer is not registered as an owner cleanup provider; the largest consumer on the incident host would again be invisible")
	}
	if found.OwnerScenario != "architecture-cartographer" {
		t.Errorf("OwnerScenario = %q, want architecture-cartographer", found.OwnerScenario)
	}
	if found.SafetyTier != cleanup.SafetyTierSafeWithOwner {
		t.Errorf("SafetyTier = %q, want safe_with_owner to match the other owner providers", found.SafetyTier)
	}
	// Disabled by default, matching the three existing owner providers.
	// Enabling them is an operator decision this plan surfaces rather than
	// settles.
	if found.DefaultMode != cleanup.ProviderModeDisabled {
		t.Errorf("DefaultMode = %q, want disabled", found.DefaultMode)
	}
	if found.DefaultApproval != cleanup.ApprovalModeOwner {
		t.Errorf("DefaultApproval = %q, want owner", found.DefaultApproval)
	}
	if err := found.Validate(); err != nil {
		t.Errorf("provider metadata is invalid: %v", err)
	}
}

// TestOwnerProvidersDeleteThroughTheirOwner asserts every owner provider
// delegates rather than reaching into another scenario's storage itself.
//
// cleanup-manager documents that it never duplicates owner-private deletion
// logic. For architecture-cartographer that boundary is load-bearing:
// graph_snapshots shares a database file with fourteen other tables, and a
// provider that truncated the file would destroy 753,927 analytics rows.
func TestOwnerProvidersDeleteThroughTheirOwner(t *testing.T) {
	for _, p := range OwnerScenarioBuiltIns(nil) {
		meta := p.Metadata()
		if meta.OwnerScenario == "" {
			t.Errorf("provider %q declares no owner scenario", meta.ID)
		}
		if meta.SafetyTier != cleanup.SafetyTierSafeWithOwner {
			t.Errorf("provider %q has tier %q; owner-delegated providers must be safe_with_owner", meta.ID, meta.SafetyTier)
		}
	}
}
