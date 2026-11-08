package volumeremediation

import (
	"slices"
	"testing"
)

// TestAdvertisedCapabilitiesAreStableAndSafe pins the exact host-volume-v1
// verb registry. Every verb is either read-only (inspect, check, smart),
// attachment-only (unmount, mount_read_write), or the single guarded
// destructive verb (repair, which requires an explicit data-loss
// acknowledgement). Adding a verb is a contract change that must be reviewed
// here; verbs that destroy data rather than recover it (format, wipe, erase)
// must never appear.
func TestAdvertisedCapabilitiesAreStableAndSafe(t *testing.T) {
	a := AdvertisedCapabilities()
	b := AdvertisedCapabilities()
	want := []string{
		string(ActionInspect),
		string(ActionCheck),
		CapabilitySMARTInspect,
		string(ActionRepair),
		string(ActionUnmount),
		string(ActionMountReadWrite),
	}
	if a.Contract != ContractVersion || !slices.Equal(a.Actions, want) {
		t.Fatalf("capabilities = %+v, want contract %q actions %v", a, ContractVersion, want)
	}
	destructive := 0
	for _, name := range a.Actions {
		if Action(name).Destructive() {
			destructive++
		}
	}
	if destructive != 1 {
		t.Fatalf("advertised %d destructive verbs, want only repair", destructive)
	}
	a.Actions[0] = "tampered"
	if b.Actions[0] == "tampered" {
		t.Fatal("capability calls share mutable action storage")
	}
}
