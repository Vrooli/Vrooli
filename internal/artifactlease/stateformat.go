package artifactlease

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// artifactLeaseSchemaDigest pins the stored shape of this store's format. The digest is
// derived from the types, so a stored field change moves it. A deliberate change
// must bump the store's write version and re-pin.
const artifactLeaseSchemaDigest = "d2bcb5e9498089d0cc634b58068870c80e7cca96c62272e510d2ddc1525cffc1"

// StateFormat describes artifact leases. A lease file sits beside the artifact it protects, so the family is declared under the runtime bin directory where installers publish.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[Lease](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "artifact-leases",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		Vocabulary: stateformat.CanonicalVocabulary(
			Schema,
			Suffix,
		),
		StatePaths: []string{
			"bin/*.lease.json",
		},
	}, nil
}
