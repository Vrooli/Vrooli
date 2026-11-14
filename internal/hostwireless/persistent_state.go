package hostwireless

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
	mt7927source "github.com/vrooli/vrooli/internal/tools/mt7927-source"
)

const preparedArtifactState = "prepared"
const localFirmwareArtifactState = "staged_local"
const signedBundleSuppliedMethod = "supplied-signed-copies"
const signedBundleMOKMethod = "installed-exact-abi-sign-file-existing-mok"

// PersistentState declares every published artifact, protected recovery record
// and native file/radio journal. Whole-source coverage is restricted to their
// codecs, identity recomputation and rollback machinery (see WholeSources).
// Read-only preflight/reporting, host inventory, build selection and CLI output
// are not whole-source stamped. Embedded source/recipe identities and the
// actual artifact vocabulary participate as explicit facts.
func PersistentState() statecontract.Owner {
	sourceDescriptors := map[string]string{}
	for _, name := range []string{mt7927source.Name, mt7927source.NobleName, mt7927source.ASUSBluetoothName} {
		profile, _ := mt7927source.Lookup(name) // Fixed owner-declared identities.
		sourceDescriptors[name+"@"+profile.Version()] = profile.DescriptorSHA256()
	}
	roots := []reflect.Type{reflect.TypeFor[Receipt](), reflect.TypeFor[SignedBundleReceipt](), reflect.TypeFor[BluetoothFirmwareQualification](), reflect.TypeFor[RecoveryProof]()}
	roots = append(roots, wirelessJournalPersistentTypes()...)
	return statecontract.Owner{Name: "host-wireless", WholeSource: true, Roots: roots, Facts: map[string]any{
		"artifact_schema_version": 1, "signed_schema_version": 1, "recovery_schema_version": 1, "journal_schema_version": 1, "radio_schema_version": 1,
		"prepared_state": preparedArtifactState, "local_firmware_state": localFirmwareArtifactState, "signed_supplied_method": signedBundleSuppliedMethod, "signed_mok_method": signedBundleMOKMethod,
		"wifi_profile": NobleProfile, "bluetooth_profile": NobleBluetoothProfile, "reviewed_abi": nobleABI,
		"wifi_patch_sha256": digest(nobleDelta), "wifi_recipe_sha256": digest(nobleBuild), "bluetooth_patch_sha256": digest(bluetoothDelta), "bluetooth_recipe_sha256": digest(bluetoothBuild),
		"governed_source_descriptors_sha256": sourceDescriptors,
	}}
}
