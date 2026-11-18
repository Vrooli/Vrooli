package coreactivation

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// coreActivationSchemaDigest pins the compiled stored shape.
//
// The digest is derived from the types, so it moves on its own whenever a
// stored field is added, removed, renamed or retyped. This constant makes that
// move a review event instead of a silent compatibility change: a deliberate
// shape change must bump the store's format version (Write and Reads), add the
// downlevel fixture that proves the new version is readable by the version it
// claims to support, and re-pin this digest. Update it for no other reason.
const coreActivationSchemaDigest = "2912891f8063589bccff7bcf61100920f0167a0a13d47060b7fd70ea089fe301"

// StateFormat describes the activation store as a versioned persistent store.
//
// The stored bytes are one operation record and one enrollment policy. Their
// contract is the JSON shape this package compiles, not the source text that
// produced it: a comment, a refactor or a toolchain change cannot move it, while
// adding, removing, renaming or retyping a stored field does. A shape change is
// breaking until a version is declared for it and a downlevel fixture proves the
// older image can still read what this one writes.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[Record](),
		reflect.TypeFor[Enrollment](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "core-activation",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		StatePaths: []string{
			"state/core-activation/current.json",
			"state/core-activation/initialized",
			"state/core-activation/enrollment.json",
			"state/core-activation/enrollment-initialized",
		},
	}, nil
}
