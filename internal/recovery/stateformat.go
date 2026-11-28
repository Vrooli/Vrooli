package recovery

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// recoveryRecordSchemaDigest pins the stored shape of this store's format. The digest is
// derived from the types, so a stored field change moves it. A deliberate change
// must bump the store's write version and re-pin.
const recoveryRecordSchemaDigest = "8c41b896bf31d0773ad42bc98fbe2572cf686aebff762587646cd49ea4cd6e90"

// StateFormat describes the coding-agent recovery records. Their directory is resolved by the recovery namespace, so this store declares the contract only.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[Record](),
		reflect.TypeFor[CodingAgentRecord](),
		reflect.TypeFor[CodingAgentExecution](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "coding-agent-recovery",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		Vocabulary: stateformat.CanonicalVocabulary(
			RecoveryTierOne,
			RecoveryTierTwo,
			RecoveryTierThree,
			RecoveryTierFour,
		),
		ContractOnly: true,
	}, nil
}
