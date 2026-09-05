package credentials

import (
	"encoding/json"
	"testing"

	"github.com/vrooli/api-core/trustposture"
	"github.com/vrooli/vrooli/internal/artifactledger"
	"github.com/vrooli/vrooli/internal/credentialauthority"
	credentialrepair "github.com/vrooli/vrooli/internal/credentials"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

func jsonContractValue(t *testing.T, value any) map[string]json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON contract: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("decode JSON contract: %v", err)
	}
	return raw
}

// TestCredentialJSONContracts pins the shape emitted by every credentials
// command that offers --format json. Values are deliberately zero or minimal:
// the contract is the field set, while host state and secret-bearing values
// must remain variable.
func TestCredentialJSONContracts(t *testing.T) {
	assertJSONKeys(t, "listCredentials", jsonContractValue(t, credentialListReport{}), "inventory_basis", "managed_instances_included", "credential_count", "declaration_site_count", "uncovered", "required_absent", "credentials")
	assertJSONKeys(t, "credentialStatus", jsonContractValue(t, credentialauthority.Status{}), "identity", "field", "configured", "provider", "provider_state", "checked_at")
	assertJSONKeys(t, "exportCredentialRecovery", jsonContractValue(t, credentialRecoveryExportReport{}), "written", "skipped")
	assertJSONKeys(t, "verifyCredentialRecovery", jsonContractValue(t, credentialauthority.RecoveryManifest{}), "version", "entries")
	assertJSONKeys(t, "credentialsDoctor", jsonContractValue(t, credentialDoctorReport{}), "provider", "credentials", "credential_count", "declaration_site_count", "inventory_basis", "managed_instances_included", "recovery")
	assertJSONKeys(t, "credentialsDoctor.recovery", jsonContractValue(t, recoveryStatus{}), "receipt_exists", "exported_at", "entry_count", "uncovered", "required_absent", "required_absent_details", "root_copy", "basis", "managed_instances_included")
	assertJSONKeys(t, "credentialsStoreCopy", jsonContractValue(t, securestore.CopyStatus{}), "path", "sink", "copied_at", "generation", "verified_at")
	assertJSONKeys(t, "credentialsStoreCopyConfigure", jsonContractValue(t, securestore.CopyConfig{}), "enabled", "sink", "interval")
	assertJSONKeys(t, "credentialsStoreStatus", jsonContractValue(t, securestore.StoreStatus{}), "path", "initialized", "unlocked", "wraps", "entries", "active", "unattended")
	assertJSONKeys(t, "credentialsStoreEntries", jsonContractValue(t, credentialStoreEntriesReport{}), "basis", "entries")
	assertJSONKeys(t, "credentialsStoreDeleteEntry", jsonContractValue(t, credentialStoreDeleteEntryReport{}), "service", "key", "deleted")
	assertJSONKeys(t, "credentialsStoreInit", jsonContractValue(t, securestore.StoreStatus{}), "path", "initialized", "unlocked", "wraps", "entries", "active", "unattended")
	assertJSONKeys(t, "credentialsStoreRewrap", jsonContractValue(t, securestore.StoreStatus{}), "path", "initialized", "unlocked", "wraps", "entries", "active", "unattended")
	assertJSONKeys(t, "credentialsStoreReselect", jsonContractValue(t, artifactledger.Receipt{}), "schema", "id", "outcome", "path", "component", "predicate", "identity", "recorded_at")
	assertJSONKeys(t, "credentialsKeyringStatusCommand", jsonContractValue(t, credentialKeyringStatus{}), "state", "supported")
	assertJSONKeys(t, "credentialsKeyringFile", jsonContractValue(t, securestore.KeyringReport{}), "path", "assessed", "loadable", "repaired", "staleDaemon")
	assertJSONKeys(t, "credentialsKeyringRepair", jsonContractValue(t, credentialrepair.RepairReport{}), "platform", "stateBefore", "stateAfter", "rungs", "resolved")
	assertJSONKeys(t, "renderBreakGlassStatus", jsonContractValue(t, trustposture.KeyStatus{}), "complete", "metadata", "public", "wrapped_private")
	assertJSONKeys(t, "renderBreakGlassCredential", jsonContractValue(t, breakGlassCredentialOutput{}), "path", "expires_at")
}
