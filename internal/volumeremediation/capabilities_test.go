package volumeremediation

import "testing"

func TestAdvertisedCapabilitiesAreStableAndSafe(t *testing.T) {
	a := AdvertisedCapabilities()
	b := AdvertisedCapabilities()
	if a.Contract != ContractVersion || len(a.Actions) != 5 {
		t.Fatalf("capabilities = %+v", a)
	}
	a.Actions[0] = "tampered"
	if b.Actions[0] == "tampered" {
		t.Fatal("capability calls share mutable action storage")
	}
}
