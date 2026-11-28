package runtimeapp

import (
	"reflect"

	"github.com/vrooli/cli-core/guardmodes"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// guardModeSchemaDigest pins the stored shapes of the guard-mode state and its
// audit entries. A deliberate change must bump the store's write version and
// re-pin.
const guardModeSchemaDigest = "47c38ccf64df495f42f7739791a504d68605c1267a7416aea98364c69938f80e"

// guardModesStateFormat describes the guard-mode store.
//
// The state file and audit log live under the user config directory
// (~/.config/vrooli/agent-policy), outside the runtime home, so this store
// declares the contract only. Its types live in the shared cli-core module, so
// the descriptor is owned here with the rest of the root persistence
// declarations.
func guardModesStateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[guardmodes.GuardMode](),
		reflect.TypeFor[guardmodes.GuardModeAudit](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:         "guard-modes",
		Write:        1,
		Reads:        []stateformat.Version{1},
		Schemas:      map[stateformat.Version]string{1: digest},
		ContractOnly: true,
	}, nil
}
