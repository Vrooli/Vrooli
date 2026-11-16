package triposrsource

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the source profile used to select and verify the
// managed upstream and adapted trees across CLI activations.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:  "triposr-source",
		Roots: []reflect.Type{reflect.TypeFor[Receipt]()},
		Facts: map[string]any{
			"sourceProfileFile":    sourceProfileFile,
			"sourceProfileLicense": sourceProfileLicense,
			"sourceProfileSchema":  sourceProfileSchema,
			"adapterProfileSchema": adapterProfileSchema,
			"adapterVersion":       AdapterVersion,
			"commit":               Commit,
			"archiveSHA256":        ArchiveSHA256,
			"archiveBytes":         ArchiveBytes,
		},
	}
}
