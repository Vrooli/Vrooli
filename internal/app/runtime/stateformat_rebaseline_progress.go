package runtimeapp

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/rebaselineprogress"
	"github.com/vrooli/vrooli/internal/stateformat"
)

// rebaselineProgressSchemaDigest pins the stored shape of the transition
// progress record. The digest is derived from the type, so a stored field
// change moves it. A deliberate change must bump Progress.Version and re-pin.
const rebaselineProgressSchemaDigest = "0905d4fda6ba64baca6c359b0b2792934ddee4a718ff2b298e716996772eb9b2"

// rebaselineProgressStateFormat describes the durable transition progress and
// pause marker as a versioned store. The paths match
// preparedRebaselineStatePathsForHome, which is the only writer.
func rebaselineProgressStateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(reflect.TypeFor[rebaselineprogress.Progress]())
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "root-rebaseline-progress",
		Write: stateformat.Version(rebaselineprogress.Version),
		Reads: []stateformat.Version{stateformat.Version(rebaselineprogress.Version)},
		Schemas: map[stateformat.Version]string{
			stateformat.Version(rebaselineprogress.Version): digest,
		},
		StatePaths: []string{
			"state/root-rebaseline/progress.json",
			"state/root-rebaseline/pause.json",
		},
	}, nil
}
