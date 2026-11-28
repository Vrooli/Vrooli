package resourcedeployment

// sampleArch is the derived-identity sample's architecture input.
const sampleArch = "amd64"

func artifactCacheDirectorySample() string {
	dir, err := artifactCacheDirectory("/cache", ArtifactCacheKey{Resource: "sample", Version: "1", SourceIdentity: "source", Platform: Platform{OS: OSLinux, Arch: ArchAMD64}})
	if err != nil {
		return "error: " + err.Error()
	}
	return dir
}
