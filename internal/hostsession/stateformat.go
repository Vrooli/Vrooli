package hostsession

import (
	"github.com/vrooli/vrooli/internal/stateformat"
)

// hostSessionSchemaDigest pins the stored shape of format 1. The fallback token
// is an opaque random string in one file, so the shape is the file itself: a
// single trimmed token. This constant exists so a future change to the token
// encoding is a review event rather than a silent compatibility change.
const hostSessionSchemaDigest = "7e950de1bbf3cc9273ebd4af84b5d29e90b5f3839a2eac93e0b84a08be692bcf"

// hostSessionStatePath is the fallback token file, relative to the runtime home.
const hostSessionStatePath = "state/" + sessionFile

// StateFormat describes the host-session fallback token as a versioned store.
//
// The token has no fields to misinterpret, so no vocabulary and no migration:
// any reader can consume any token this writer produces. What an older image
// must still find is the file, and that is the declared state path.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestSchema(struct {
		Kind    string `json:"kind"`
		Version int    `json:"version"`
	}{"trimmed-token", 1})
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:       "host-session",
		Write:      1,
		Reads:      []stateformat.Version{1},
		Schemas:    map[stateformat.Version]string{1: digest},
		StatePaths: []string{hostSessionStatePath},
	}, nil
}
