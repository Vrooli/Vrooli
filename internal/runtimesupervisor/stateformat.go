package runtimesupervisor

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// supervisorOwnerSchemaDigest pins the stored shape of this store's format. The digest is
// derived from the types, so a stored field change moves it. A deliberate change
// must bump the store's write version and re-pin.
const supervisorOwnerSchemaDigest = "100dd83fb2b1030bdd8ecfe0f44ed0ebbb07f9c0968f58d45c0725c9cba9ede3"

// StateFormat describes the supervisor owner identity retained in the runtime registry database, which the runtime-registry store already snapshots. This store declares the identity shape only.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[supervisorOwnerIdentity](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "supervisor-owner",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		Vocabulary: stateformat.CanonicalVocabulary(
			LogFileName,
		),
		ContractOnly: true,
	}, nil
}
