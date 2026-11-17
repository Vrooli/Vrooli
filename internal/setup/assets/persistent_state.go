package assets

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the image manifest and host provisioning receipt
// that subsequent setup inspections read to determine readiness.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:  "setup-assets",
		Roots: []reflect.Type{reflect.TypeFor[ImageManifest](), reflect.TypeFor[Receipt]()},
		Facts: map[string]any{"assetPlanVersion": assetPlanVersion, "aptPackageManager": aptPackageManager, "imageManifestSchemaVersion": imageManifestSchemaVersion, "imageReceiptVersion": imageReceiptVersion},
	}
}
