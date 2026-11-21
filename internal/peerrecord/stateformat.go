package peerrecord

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// peerSchemaDigest pins the stored shape of the peer envelope. The digest is
// derived from the type, so a stored field change moves it. A deliberate change
// must bump SchemaVersion and re-pin.
const peerSchemaDigest = "dc80fcbd39b1abc0d08251632198262d5deb8b285e38fd6fe474d3758a04b5b7"

// peerStatePattern is the peer envelope family. identity.go derives the
// directory from the runtime home's config entry and the file name from the
// peer's address, so the member names exist only at runtime.
const peerStatePattern = "config/peers/*.json"

// StateFormat describes peer records as a versioned store.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(reflect.TypeFor[Record]())
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:       "peer-records",
		Write:      stateformat.Version(SchemaVersion),
		Reads:      []stateformat.Version{stateformat.Version(SchemaVersion)},
		Schemas:    map[stateformat.Version]string{stateformat.Version(SchemaVersion): digest},
		StatePaths: []string{peerStatePattern},
	}, nil
}
