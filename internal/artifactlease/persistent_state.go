package artifactlease

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares artifact leases. Installers and resource drivers
// write them and reclamation reads them in other processes; a lease a
// restored prior image cannot parse would leave an in-use artifact
// unprotected.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:  "artifact-leases",
		Roots: []reflect.Type{reflect.TypeFor[Lease]()},
		Facts: map[string]any{"Schema": Schema, "Suffix": Suffix},
	}
}
