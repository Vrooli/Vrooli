package process

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// processRecordSchemaDigest pins the stored shape of format 1: the per-step
// record written under each scenario's process directory. The digest is derived
// from the type, so adding, removing, renaming or retyping a stored field moves
// it. A deliberate change must bump the version and re-pin; nothing else should.
const processRecordSchemaDigest = "cf4537cb9c1b5e6642f4202dd2e1286e8b333e34efac84ad300eddcaad61d1b1"

// processRecordStatePaths name the record families. The member names are one
// directory per scenario and one file per step, neither of which is known when
// the binary is compiled, so they are declared as glob families. The lock files
// that serialise writers are transient and deliberately not snapshotted.
const (
	processRecordPathPattern = "processes/scenarios/*/*.json"
	processPIDSidecarPattern = "processes/scenarios/*/*.pid"
)

// StateFormat describes the per-step process records as a versioned store.
//
// The stored values are the record statuses and launch-reference prefixes, so
// the vocabulary is built from those constants rather than the whole package:
// a new status is a value an older reader must already understand.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(reflect.TypeFor[Record]())
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:       "process-records",
		Write:      1,
		Reads:      []stateformat.Version{1},
		Schemas:    map[stateformat.Version]string{1: digest},
		Vocabulary: stateformat.CanonicalVocabulary(RecordStatusRunning, RecordStatusExitedVerified, launchRefPrefix),
		StatePaths: []string{processRecordPathPattern, processPIDSidecarPattern},
	}, nil
}
