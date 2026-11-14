package resourcedeployment

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the managed-service artifact cache manifest, its
// name and version, the key derivation that locates cache entries, and the
// release trust format.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name: "resource-artifact-cache",
		Roots: []reflect.Type{
			reflect.TypeFor[artifactCacheManifest](),
			reflect.TypeFor[ArtifactCacheKey](),
		},
		Facts: map[string]any{
			"artifactCacheSchemaVersion": artifactCacheSchemaVersion,
			"artifactCacheManifestName":  artifactCacheManifestName,
			"releaseTrustSchemaVersion":  releaseTrustSchemaVersion,
			"artifact_cache_dir_sample":  artifactCacheDirectorySample(),
		},
	}
}

// sampleArch is the derived-identity sample's architecture input.
const sampleArch = "amd64"

func artifactCacheDirectorySample() string {
	dir, err := artifactCacheDirectory("/cache", ArtifactCacheKey{Resource: "sample", Version: "1", SourceIdentity: "source", Platform: Platform{OS: deploymentOSLinux, Arch: sampleArch}})
	if err != nil {
		return "error: " + err.Error()
	}
	return dir
}
