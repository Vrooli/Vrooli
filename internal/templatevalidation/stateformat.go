package templatevalidation

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// templateMarkerSchemaDigest pins the stored shape of the run marker. A
// deliberate change must bump MarkerVersion and re-pin.
const templateMarkerSchemaDigest = "ccb742910310dca7fa00ac3e89e4d2b1b3212b75dc0bcb98b3f6aac1643b5e6c"

// StateFormat describes the template-validation run marker as a versioned
// store. The marker lives inside the repository under validation, so this store
// declares the contract only: there is no runtime-home file to snapshot.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(reflect.TypeFor[RunMarker]())
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:         "template-validation-markers",
		Write:        1,
		Reads:        []stateformat.Version{1},
		Schemas:      map[stateformat.Version]string{1: digest},
		ContractOnly: true,
	}, nil
}
