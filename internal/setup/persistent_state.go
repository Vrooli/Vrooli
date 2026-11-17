package setup

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState covers the setup result and in-progress record. Bootstrap
// and configuration writers remain whole-source until their map-based formats
// and handoff semantics have explicit, independently verified contracts.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name: "setup-state", WholeSource: true,
		Roots: []reflect.Type{reflect.TypeFor[SetupResult](), reflect.TypeFor[activeSetupState]()},
		Facts: map[string]any{"SetupResultVersion": SetupResultVersion, "ownershipMigrationVersion": ownershipMigrationVersion, "ownershipMigrationClasses": ownershipMigrationClasses},
	}
}
