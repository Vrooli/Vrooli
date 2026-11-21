package triposrsource

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// triposrSourceSchemaDigest pins the stored shape of this store's format. The digest is
// derived from the types, so a stored field change moves it. A deliberate change
// must bump the store's write version and re-pin.
const triposrSourceSchemaDigest = "cb07088914c52fa5b28aadca1e1a8c5d14aa79e2aa05e6fff3d08698d6e361e6"

// StateFormat describes the TripoSR source receipt and profile identities. The profile is written inside an adapted artifact tree, so this store declares the contract only.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[Receipt](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "triposr-source",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		Vocabulary: stateformat.CanonicalVocabulary(
			AdapterVersion,
			ArchiveSHA256,
			Commit,
			ToolName,
			sourceProfileFile,
			sourceProfileLicense,
		),
		ContractOnly: true,
	}, nil
}
