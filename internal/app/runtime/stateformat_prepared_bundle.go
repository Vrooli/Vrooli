package runtimeapp

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/stateformat"
)

// preparedBundleSchemaDigest pins the stored shape of the ordinary-bundle
// intent record. A deliberate change must bump its Version and re-pin.
const preparedBundleSchemaDigest = "cc134bbbc38037c277b802963356f8a149c6ad9a84f66730fc5817752d3d93af"

// preparedBundleIntentPath is the durable intent record. The bundle directory
// beside it holds the recovery artefacts themselves (the candidate CLI and the
// atomic installer), which are retained deliberately rather than snapshotted:
// they are the means of recovery, not rollback state.
const preparedBundleIntentPath = "state/root-rebaseline/ordinary-bundle/intent.json"

// preparedRootBundleStateFormat describes the ordinary-bundle intent record.
//
// The record is read by the bundle's own CLI on resume, so its compatibility
// question is bounded: a newer candidate writes and reads its own intent. What
// still must not change silently is the record's shape, which this digest pins.
func preparedRootBundleStateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(reflect.TypeFor[buildinfo.PreparedBundleIntent]())
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:       "prepared-root-bundle",
		Write:      1,
		Reads:      []stateformat.Version{1},
		Schemas:    map[stateformat.Version]string{1: digest},
		StatePaths: []string{preparedBundleIntentPath},
	}, nil
}
