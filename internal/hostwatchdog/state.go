// Package hostwatchdog owns the host-watchdog's durable state and its declared
// format. The type lives here rather than in the CLI package so the root
// control plane can declare the store without importing the CLI, which would
// form an import cycle.
package hostwatchdog

import (
	"reflect"
	"time"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// SchemaDigest pins the stored shape of the watchdog state. A deliberate change
// must bump the store's write version and re-pin.
const SchemaDigest = "90b7dc7184ffda818b8b0e55f7bace4a31085675b6a8a459678611878b7a5d43"

// StatePath is the watchdog's durable state, relative to the runtime home.
const StatePath = "state/host-watchdog/state.json"

// State is the host-watchdog's durable record: when the watched mount was
// first observed below its floor. It carries no vocabulary; any reader can
// consume any timestamp this writer produces.
type State struct {
	FirstBelow time.Time `json:"first_below"`
}

// StateFormat describes the host-watchdog state as a versioned store.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(reflect.TypeFor[State]())
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:       "host-watchdog-state",
		Write:      1,
		Reads:      []stateformat.Version{1},
		Schemas:    map[stateformat.Version]string{1: digest},
		StatePaths: []string{StatePath},
	}, nil
}
