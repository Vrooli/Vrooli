package resourcedeployment

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// artifactCacheSchemaDigest pins the stored shape of this store's format. The digest is
// derived from the types, so a stored field change moves it. A deliberate change
// must bump the store's write version and re-pin.
const artifactCacheSchemaDigest = "d99bda994dce1a533fd469747df205be741a8779bf4fd16ee9a45bfb81efce91"

// StateFormat describes the resource artifact cache manifest and key. Cache roots are chosen by the caller, so this store declares the contract only.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[artifactCacheManifest](),
		reflect.TypeFor[ArtifactCacheKey](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "resource-artifact-cache",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		Vocabulary: stateformat.CanonicalVocabulary(
			OSLinux,
			OSMacOS,
			OSWindows,
			ArchAMD64,
			ArchARM64,
			ActivationStateActivated,
			ExternalManagementForbidden,
			ServiceShutdownInterrupt,
			ServiceShutdownTerminate,
			UpdateStrategyExclusiveStopStart,
			UpdateStrategyRollingStateless,
			activationCurrentEntry,
			activationGenerations,
			activationStagingDir,
			artifactCacheManifestName,
			artifactCacheSchemaVersion,
			dataAcquisitionEager,
			dataAcquisitionOnDemand,
			deploymentLayoutDir,
			deploymentLayoutFile,
			TargetUnsupported,
			deploymentVerificationChecksum,
			deploymentVerificationHostTool,
			releaseTrustSchemaVersion,
			string(AccessReadOnly),
			string(AccessReadWrite),
			string(ArtifactTrustDevelopmentLocal),
			string(ArtifactTrustProduction),
			string(ProviderAttachOnly),
			string(ProviderManagedDiscovered),
			string(ProviderManagedPrivate),
			string(ProviderManagedShared),
			string(ProviderRemoteVrooli),
			string(ProviderTargetControlPlane),
			string(ProviderTargetDesktopBundle),
		),
		ContractOnly: true,
	}, nil
}
