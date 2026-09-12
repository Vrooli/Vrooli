package ramp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/releasesvc"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

// ArtifactKind is the spine artifact kind for a cloud release.
const ArtifactKind = "cloud-release"

// ArtifactRef is the immutable reference spelling for a release digest.
func ArtifactRef(releaseDigest string) string {
	return "release:" + strings.TrimPrefix(strings.ToLower(strings.TrimSpace(releaseDigest)), "sha256:")
}

// ReleaseDigestFromRef reverses ArtifactRef.
func ReleaseDigestFromRef(ref string) string {
	return strings.TrimPrefix(strings.TrimSpace(ref), "release:")
}

// Builder wraps the release builder. SourceRef is a deployment id whose
// stored manifest is built; Parameters may override goos/goarch, closure
// digest and trust mode. The artifact is the immutable release artifact set
// keyed by its release digest.
type Builder struct {
	Deployments Deployments
	Releases    ReleaseBuilder
}

// Build implements deliveryramp.Builder.
func (b Builder) Build(ctx context.Context, request deliveryramp.BuildRequest) (deliveryramp.Artifact, error) {
	if b.Deployments == nil || b.Releases == nil {
		return deliveryramp.Artifact{}, fmt.Errorf("cloud builder is missing its deployment source or release builder")
	}
	deploymentID := strings.TrimSpace(request.SourceRef)
	if deploymentID == "" {
		return deliveryramp.Artifact{}, fmt.Errorf("cloud build source reference (deployment id) is required")
	}
	dep, err := b.Deployments.GetDeployment(ctx, deploymentID)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("load deployment %s: %w", deploymentID, err)
	}
	if dep == nil {
		return deliveryramp.Artifact{}, fmt.Errorf("deployment %s was not found", deploymentID)
	}
	var manifest domain.CloudManifest
	if err := json.Unmarshal(dep.Manifest, &manifest); err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("decode deployment manifest: %w", err)
	}
	params := request.Parameters
	goos, goarch := strings.TrimSpace(params["goos"]), strings.TrimSpace(params["goarch"])
	if goos == "" {
		goos = "linux"
	}
	if goarch == "" {
		goarch = "amd64"
	}
	rel, err := b.Releases.Build(ctx, releasesvc.BuildRequest{Manifest: manifest, ClosureDigest: strings.TrimSpace(params["closure_digest"]), GOOS: goos, GOARCH: goarch, TrustMode: strings.TrimSpace(params["trust_mode"])})
	if err != nil {
		return deliveryramp.Artifact{}, err
	}
	if !rel.Complete || strings.TrimSpace(rel.Digest) == "" {
		return deliveryramp.Artifact{}, fmt.Errorf("release builder returned an incomplete release")
	}
	artifact := deliveryramp.Artifact{
		ImmutableRef: ArtifactRef(rel.Digest),
		Kind:         ArtifactKind,
		Checksum:     "sha256:" + strings.TrimPrefix(rel.Digest, "sha256:"),
		CreatedAt:    time.Now().UTC(),
		Metadata: map[string]string{
			"release_digest":       rel.Digest,
			"bundle_sha256":        rel.Manifest.BundleSHA256,
			"native_cli":           rel.Manifest.NativeCLI.SHA256,
			"native_cli_platform":  rel.Manifest.NativeCLI.GOOS + "/" + rel.Manifest.NativeCLI.GOARCH,
			"closure_digest":       rel.Manifest.ClosureDigest,
			"configuration_digest": rel.Manifest.ConfigurationDigest,
			"deployment_id":        dep.ID,
		},
	}
	if built, err := time.Parse(time.RFC3339Nano, rel.Inputs.BuiltAt); err == nil {
		artifact.CreatedAt = built.UTC()
	}
	if rel.Dir != "" {
		artifact.LocalPath = rel.BundlePath()
		if info, err := os.Stat(artifact.LocalPath); err == nil {
			artifact.SizeBytes = info.Size()
		}
	}
	return artifact, nil
}

var _ deliveryramp.Builder = Builder{}
