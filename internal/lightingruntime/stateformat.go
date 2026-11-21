package lightingruntime

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// lightingIntentSchemaDigest pins the stored shape of the durable intent. The
// digest is derived from the types, so a stored field change moves it. A
// deliberate change must bump lightingStateVersion and re-pin.
const lightingIntentSchemaDigest = "88bfe47f92713dd16bf9f6ba23a8e3e6246c8f4cc48c7b9140a9158bd131c5c3"

// lightingIntentStatePath is the intent document, relative to the runtime home.
// It must match NewController's state/host-lighting/intent.json resolution.
const lightingIntentStatePath = "state/" + lightingStateDir + "/" + lightingStateFile

// StateFormat describes the headless-lighting intent as a versioned store.
//
// The stored vocabulary is the phase set plus the preparation sources: a prior
// reader that does not know a phase cannot interpret an intent a candidate
// writes, so adding one is a transition.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(reflect.TypeFor[Intent]())
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:       "host-lighting-intent",
		Write:      stateformat.Version(lightingStateVersion),
		Reads:      []stateformat.Version{stateformat.Version(lightingStateVersion)},
		Schemas:    map[stateformat.Version]string{stateformat.Version(lightingStateVersion): digest},
		Vocabulary: stateformat.CanonicalVocabulary(string(Disabled), string(Enabled), string(Disabling), PreparationManaged, PreparationExisting),
		StatePaths: []string{lightingIntentStatePath},
	}, nil
}
