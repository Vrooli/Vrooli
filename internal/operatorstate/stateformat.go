package operatorstate

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// operatorStateSchemaDigest pins the stored shape of this store's format. The digest is
// derived from the types, so a stored field change moves it. A deliberate change
// must bump the store's write version and re-pin.
const operatorStateSchemaDigestV1 = "b286678103fd8235423c3ccb556e37e958c1a25b59ec708eea36922d3d19ac36"
const operatorStateSchemaDigestV2 = "c88aec15a907d9a8d3f3ec34de1a45d555bc584c970b4770fa6fcddfad69df66"
const operatorStateSchemaDigest = "e3f14df0618238b87878f9217602760decb6424a31afaeef67ca2656547e8b24"

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
		Write: 3,
		Reads: []stateformat.Version{1, 2, 3},
		Schemas: map[stateformat.Version]string{
			1: operatorStateSchemaDigestV1,
			2: operatorStateSchemaDigestV2,
			3: digest,
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
			permissionReceiptField,
			permissionReceiptOutcomeUpdated,
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
