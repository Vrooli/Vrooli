package cliinstall

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// installationInventorySchemaDigest pins the stored shape of format 1: the
// install record, the per-binary metadata sidecars and retained uninstall
// plans. The digest is derived from the types, so a stored field change moves
// it. A deliberate change must bump installRecordVersion/planVersion and
// re-pin.
const installationInventorySchemaDigest = "d88455499ffe1949b4d1400aff6b3ec1f9877023a54f8c322c02f2db699a74e2"

// The install record and retained plans live in fixed directories; the
// per-binary sidecars sit beside whatever the manager installed, so they are
// declared as families under the runtime bin directory.
const (
	installationRecordPath   = "state/install-record.json"
	installationPlansPattern = "state/uninstall-plans/*.json"
	installationMetadataGlob = "bin/*.build.meta"
	installationManifestGlob = "bin/*.manifest.json"
)

// StateFormat describes the installation inventory as a versioned store.
//
// The stored vocabulary is the install kind and uninstall entry-kind sets: an
// older reader that does not know one of them cannot interpret a record a
// candidate writes.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[InstallRecord](),
		reflect.TypeFor[InstallMetadata](),
		reflect.TypeFor[UninstallPlan](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	values := []string{string(KindScenario), string(KindResource), managerGoModule, installationGoModule}
	for _, kind := range []InstallEntryKind{
		EntryFile, EntryDirectory, EntryBinary, EntryService, EntryPackage,
		EntryImage, EntryContainer, EntryVolume, EntryNetwork,
	} {
		values = append(values, string(kind))
	}
	return stateformat.Store{
		Name:       "installation-inventory",
		Write:      stateformat.Version(installRecordVersion),
		Reads:      []stateformat.Version{stateformat.Version(installRecordVersion)},
		Schemas:    map[stateformat.Version]string{stateformat.Version(installRecordVersion): digest},
		Vocabulary: stateformat.CanonicalVocabulary(values...),
		StatePaths: []string{
			installationRecordPath,
			installationPlansPattern,
			installationMetadataGlob,
			installationManifestGlob,
		},
	}, nil
}
