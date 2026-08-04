package resources

import (
	"testing"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// Recovery must be able to name what no manifest can. An instance ID is
// generated at runtime, so without this source `recovery export --all` walks
// declarations only and silently omits every unseal key — the one piece of
// material whose loss cannot be undone.
func TestVaultUnsealKeyEntriesNamesEveryLiveInstance(t *testing.T) {
	broker := &Broker{instances: map[string]ManagedInstance{
		"beta":  {ID: "beta", Resource: "vault", Provider: resourcedeployment.ProviderManagedPrivate},
		"alpha": {ID: "alpha", Resource: "vault", Provider: resourcedeployment.ProviderManagedPrivate},
		// A different resource must not appear: only Vault keeps unseal keys.
		"pg": {ID: "pg", Resource: "postgres"},
	}}

	entries := VaultUnsealKeyEntries(broker)
	if len(entries) != 2 {
		t.Fatalf("entries = %+v, want the two vault instances only", entries)
	}
	// Sorted, so a recovery bundle's contents do not depend on map order.
	if entries[0].InstanceID != "alpha" || entries[1].InstanceID != "beta" {
		t.Fatalf("entries are not deterministic: %+v", entries)
	}
	if entries[0].LogicalID != "vrooli/vault/alpha" || entries[0].Field != "unseal-key" {
		t.Fatalf("entry address = %s:%s, want vrooli/vault/alpha:unseal-key",
			entries[0].LogicalID, entries[0].Field)
	}
}

func TestVaultUnsealKeyEntriesToleratesNoBroker(t *testing.T) {
	if entries := VaultUnsealKeyEntries(nil); len(entries) != 0 {
		t.Fatalf("entries = %+v, want none without a broker", entries)
	}
	if entries := VaultUnsealKeyEntries(&Broker{}); len(entries) != 0 {
		t.Fatalf("entries = %+v, want none from an empty broker", entries)
	}
}
