package retainedruntime

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// retainedRuntimeSchemaDigest pins the stored shape of this store's format. The digest is
// derived from the types, so a stored field change moves it. A deliberate change
// must bump the store's write version and re-pin.
const retainedRuntimeSchemaDigest = "c8b08627e00006e5870db7280424726761f4187acfb13fc9f942d7e426f61ad1"

// StateFormat describes the retained-runtime manifest and its entries. The store root is chosen by the lifecycle caller, so this store declares the contract only.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[manifest](),
		reflect.TypeFor[entry](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "retained-runtime",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		ContractOnly: true,
	}, nil
}
