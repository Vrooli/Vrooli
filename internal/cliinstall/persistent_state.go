package cliinstall

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the installation-ownership inventory, the install
// metadata sidecars and retained uninstall plans.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name: "installation-inventory",
		Roots: []reflect.Type{
			reflect.TypeFor[InstallRecord](),
			reflect.TypeFor[InstallMetadata](),
			reflect.TypeFor[UninstallPlan](),
		},
		Facts: map[string]any{
			"installRecordVersion": installRecordVersion,
			"planVersion":          planVersion,
		},
	}
}
