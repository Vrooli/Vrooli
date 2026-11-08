package securestore

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the credential store. A restored prior image must
// still open the sealed file, follow the backend selection and use the unlock
// cache the candidate wrote, or unattended credential access stops after a
// rollback. The sealing, key-wrapping and unlock sources (and the shared
// credentialpolicy envelope) are whole-source because their cryptographic
// framing lives in function bodies; the rest of the package is precise.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:        "secure-store",
		WholeSource: true,
		Roots: []reflect.Type{
			reflect.TypeFor[sealedFile](),
			reflect.TypeFor[passphraseParams](),
			reflect.TypeFor[BackendSelection](),
			reflect.TypeFor[CopyConfig](),
		},
		Facts: map[string]any{
			"sealedFormatVersion":     sealedFormatVersion,
			"backendSelectionVersion": backendSelectionVersion,
		},
	}
}
