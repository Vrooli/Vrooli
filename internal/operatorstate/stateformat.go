package operatorstate

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// operatorStateSchemaDigest pins the stored shape of this store's format. The digest is
// derived from the types, so a stored field change moves it. A deliberate change
// must bump the store's write version and re-pin.
const operatorStateSchemaDigest = "6295a983713c000669282d15e72f755edb8e60d3f9cc2aa22020d5f2f8078927"

// StateFormat describes the operator-state document. Its root is chosen from the platform storage roots (~/.local/state by default), outside the runtime home, so this store declares the contract only.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[Document](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "operator-state",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		Vocabulary: stateformat.CanonicalVocabulary(
			ResourceSourceManaged,
			ResourceSourceHost,
			AccelPreferenceAuto,
			AccelPreferenceForceCPU,
			AccelPreferencePreferGPU,
			SchemaPath,
			StateFile,
			capacityPostureBalanced,
			capacityPostureMinimal,
			capacityPostureResponsive,
			capacityPostureThroughput,
			documentVersion,
			hostWorkloadPostureVrooliOnly,
			HostWorkloadPostureWholeHost,
			profileSessionGuided,
			profileSessionManual,
			trustPostureHosted,
			trustPosturePersonal,
			trustPostureShared,
			updateControlGuard,
			updateControlObserve,
			updateControlOwn,
		),
		ContractOnly: true,
	}, nil
}
