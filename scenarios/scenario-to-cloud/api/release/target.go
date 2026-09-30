package release

import (
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/packages/cloudrelease"

	"scenario-to-cloud/apierrors"
)

// NativeCLIArtifact is the release-bound control-plane binary delivered to a
// target. It is only ever resolved from a complete release whose manifest
// binds the binary's sha256 and platform.
type NativeCLIArtifact struct {
	ReleaseDigest string `json:"release_digest"`
	Path          string `json:"path"`
	FileName      string `json:"file_name"`
	SHA256        string `json:"sha256"`
	GOOS          string `json:"goos"`
	GOARCH        string `json:"goarch"`
}

// NativeCLIForTarget resolves the native control plane of the release that
// contains bundlePath and checks it against the target platform before any
// delivery. Refusals are typed: a bundle outside a release or an incomplete
// release is release_verification_failed; a platform mismatch is
// unsupported_capability naming the artifact; a binary or bundle whose
// bytes differ from the manifest is release_verification_failed with the
// mismatching artifact in details.
func NativeCLIForTarget(bundlePath string, platform Platform) (NativeCLIArtifact, error) {
	dir := filepath.Dir(bundlePath)
	if filepath.Base(bundlePath) != cloudrelease.BundleFileName {
		return NativeCLIArtifact{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "bundle %s is not a release artifact; build a release first (releases/<digest>/%s)", bundlePath, cloudrelease.BundleFileName).
			WithDetail("reason", ReasonNotFound)
	}
	if _, err := os.Stat(filepath.Join(dir, cloudrelease.CompleteMarker)); err != nil {
		return NativeCLIArtifact{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "release at %s is incomplete; it cannot be delivered", dir).WithDetail("reason", ReasonIncomplete)
	}
	rel, err := loadReleaseDir(dir, true)
	if err != nil {
		return NativeCLIArtifact{}, err
	}
	manifest := rel.Manifest
	if manifest.NativeCLI.GOOS != platform.GOOS || manifest.NativeCLI.GOARCH != platform.GOARCH {
		return NativeCLIArtifact{}, apierrors.Newf(apierrors.CodeUnsupportedCapability, "release %s carries a %s/%s control plane; the target is %s", rel.Digest, manifest.NativeCLI.GOOS, manifest.NativeCLI.GOARCH, platform).
			WithDetail("artifact", cloudrelease.NativeCLIFileName(manifest.NativeCLI.GOOS, manifest.NativeCLI.GOARCH)).WithDetail("declared", manifest.NativeCLI.GOOS+"/"+manifest.NativeCLI.GOARCH).WithDetail("required", platform.String()).WithDetail("release_digest", rel.Digest)
	}
	bundleSum, _, err := fileSHA256(bundlePath)
	if err != nil {
		return NativeCLIArtifact{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "read bundle: %v", err).WithDetail("reason", "archive_unreadable")
	}
	if bundleSum != manifest.BundleSHA256 {
		return NativeCLIArtifact{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "bundle sha256 does not match release %s", rel.Digest).
			WithDetail("reason", "bundle_sha256_mismatch").WithDetail("artifact", cloudrelease.BundleFileName).WithDetail("declared", manifest.BundleSHA256).WithDetail("observed", bundleSum)
	}
	name := cloudrelease.NativeCLIFileName(manifest.NativeCLI.GOOS, manifest.NativeCLI.GOARCH)
	path := filepath.Join(dir, name)
	sum, _, err := fileSHA256(path)
	if err != nil {
		return NativeCLIArtifact{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "read %s: %v", name, err).WithDetail("reason", "native_cli_mismatch").WithDetail("artifact", name)
	}
	if sum != manifest.NativeCLI.SHA256 {
		return NativeCLIArtifact{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "%s sha256 does not match release %s", name, rel.Digest).
			WithDetail("reason", "native_cli_mismatch").WithDetail("artifact", name).WithDetail("declared", manifest.NativeCLI.SHA256).WithDetail("observed", sum)
	}
	return NativeCLIArtifact{ReleaseDigest: rel.Digest, Path: path, FileName: name, SHA256: sum, GOOS: manifest.NativeCLI.GOOS, GOARCH: manifest.NativeCLI.GOARCH}, nil
}
