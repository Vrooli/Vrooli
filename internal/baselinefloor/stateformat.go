package baselinefloor

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// baselineFloorSchemaDigest pins the stored shape of this store's format. The digest is
// derived from the types, so a stored field change moves it. A deliberate change
// must bump the store's write version and re-pin.
const baselineFloorSchemaDigest = "c6d8764dcf20edb4ca9b6fe61716d5459b517029f38e5c2b02aca58712c00f9a"

// StateFormat describes the baseline-floor engagement manifest and script. Engagements live under a caller-chosen root, so this store declares the contract only.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[Manifest](),
		reflect.TypeFor[Script](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "baseline-floor",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		Vocabulary: stateformat.CanonicalVocabulary(
			string(EngineSQLite),
			string(ModeLive),
			string(ModeShadow),
			string(SharedSourcesFrozen),
			string(SharedSourcesUnfrozen),
		),
		ContractOnly: true,
	}, nil
}
