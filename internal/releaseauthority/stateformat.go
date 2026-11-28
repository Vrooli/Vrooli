package releaseauthority

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// releaseAuthoritySchemaDigest pins the stored shape of this store's format. The digest is
// derived from the types, so a stored field change moves it. A deliberate change
// must bump the store's write version and re-pin.
const releaseAuthoritySchemaDigest = "9611be30abcb0e8de5c7edc0809816fb31e8ba02d13612ae80ce2e764f0f0896"

// StateFormat describes the release authorization status and authority a signed release manifest carries. The files belong to a release bundle staged elsewhere, so this store declares the contract only.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[Status](),
		reflect.TypeFor[Authority](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "release-authority",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		ContractOnly: true,
	}, nil
}
