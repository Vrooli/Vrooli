package operatorcapability

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// operatorInputSchemaDigest pins the stored shape of format 1: the durable
// operator-input queue and the capability inventory and evidence references it
// carries. The digest is derived from the types, so a stored field change moves
// it. A deliberate change must bump the version and re-pin.
const operatorInputSchemaDigest = "18fbc324535cc1cabfeb4d8e8070184edc0c2c5922d77a25563593baddc0cbce"

// operatorInputStatePath is the queue document, relative to the runtime home. It
// must match QueuePath's state/operator-input.json resolution.
const operatorInputStatePath = "state/operator-input.json"

// StateFormat describes the operator-input queue as a versioned store.
//
// The stored vocabulary is the closed set of candidate statuses, evidence
// effects and operator decisions: a prior reader that does not know one of
// them cannot interpret a queue a candidate writes.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[Pending](),
		reflect.TypeFor[Inventory](),
		reflect.TypeFor[EvidenceReference](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:       "operator-capability",
		Write:      1,
		Reads:      []stateformat.Version{1},
		Schemas:    map[stateformat.Version]string{1: digest},
		Vocabulary: stateformat.CanonicalVocabulary(CandidateStatusReady, EvidenceEffectReadOnly, EvidenceEffectBoundedWrite, DecisionDeclined, DecisionPending),
		StatePaths: []string{operatorInputStatePath},
	}, nil
}
