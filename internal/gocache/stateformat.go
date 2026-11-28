package gocache

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// goCacheSchemaDigest pins the stored shape of this store's format. The digest is
// derived from the types, so a stored field change moves it. A deliberate change
// must bump the store's write version and re-pin.
const goCacheSchemaDigest = "073abb646b4004e160dcbc2b1bdfef46480f290265819d56c35f13ad90d24e09"

// StateFormat describes the Go build-cache reclaim plan, cursor, receipt and pin journals. They live inside the Go build cache, outside the runtime home, so this store declares the contract only.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[savedPlan](),
		reflect.TypeFor[sweepCursor](),
		reflect.TypeFor[Receipt](),
		reflect.TypeFor[pinJournal](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "go-cache",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		Vocabulary: stateformat.CanonicalVocabulary(
			PlanSchema,
			ReceiptSchema,
			StatusSchema,
			StopBlocked,
			StopBusy,
			StopComplete,
			StopDeadline,
			StopLockTimeout,
			StopMaxBytes,
		),
		ContractOnly: true,
	}, nil
}
