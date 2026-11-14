package autohealrecoveryprivileges

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	"testing"
)

func TestInspectRequiresTheInstalledBrokerRecoveryCapability(t *testing.T) {
	old := inspectBroker
	t.Cleanup(func() { inspectBroker = old })
	h := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_recovery_privileges"})
	for _, available := range []bool{false, true} {
		inspectBroker = func() privilegebroker.SetupStatus {
			caps := []string{privilegebroker.ActionCapabilitiesInspect}
			if available {
				caps = append(caps, privilegebroker.ActionServiceRecover)
			}
			return privilegebroker.SetupStatus{Available: true, Capabilities: caps}
		}
		status := h.Inspect(hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{})
		if status.Applied != available {
			t.Fatalf("available=%t status=%+v", available, status)
		}
		if !available && status.ExecutionState != hostreqkit.ExecutionManualActionRequired {
			t.Fatalf("missing capability claimed ready: %+v", status)
		}
	}
}
func TestApplyNeverCreatesAPrivateRuntimeGrant(t *testing.T) {
	h := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_recovery_privileges"})
	status, err := h.Apply(hostreqkit.Host{OS: "linux"}, hostreqkit.ItemStatus{}, hostreqkit.EnsureOptions{})
	if err != nil || status.Applied || status.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("status=%+v err=%v", status, err)
	}
}
func TestOtherPlatformsAreExplicitlyNotApplicable(t *testing.T) {
	h := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_recovery_privileges"})
	for _, platform := range []string{"darwin", "windows"} {
		status := h.Inspect(hostreqkit.Host{OS: platform}, hostreqspec.ResolvedRequirement{})
		if status.SupportClass != hostreqkit.SupportNotApplicable {
			t.Fatalf("platform=%s status=%+v", platform, status)
		}
	}
}
