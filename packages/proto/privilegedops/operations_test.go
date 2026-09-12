package privilegedops

import (
	"testing"
)

func TestOnboardingCapabilitiesAreStableAndCopied(t *testing.T) {
	capabilities := OnboardingCapabilities()
	if len(capabilities) != 7 {
		t.Fatalf("capability count = %d", len(capabilities))
	}
	capabilities[0].Name = "mutated"
	if OnboardingCapabilities()[0].Name == "mutated" {
		t.Fatal("capability vocabulary leaked a mutable slice")
	}
	want := []string{CapabilityAgentPresence, CapabilityRuntime, CapabilityProvisioning, CapabilitySSHManagement, CapabilityCleanupPlanning, CapabilityCleanupApplication, CapabilityTargetBoundBreakGlass}
	got := CapabilityNames()
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("capability[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
