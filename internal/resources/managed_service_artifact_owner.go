package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/binaryfetch"
)

// ManagedServiceArtifact is a verified staged image, not a serving-process claim.
// It carries no credential material or launch authority.
type ManagedServiceArtifact struct {
	Resource string `json:"resource"`
	Path     string `json:"path"`
	Version  string `json:"version"`
	SHA256   string `json:"sha256"`
}

// InspectManagedServiceArtifact reads the resource-owned acquisition authority.
// It does not adopt PATH executables, acquire bytes, or change service lifetime.
func (c *Controller) InspectManagedServiceArtifact(ctx context.Context, name string) (ManagedServiceArtifact, error) {
	manifest, err := c.managedServiceArtifactManifest(name)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	return c.inspectManagedServiceArtifact(ctx, manifest)
}

// StageManagedServiceArtifact converges the manifest-selected artifact through
// the existing verified acquisition owner. Callers must admit dependency policy
// before invoking this write; staging never starts/stops a serving process.
func (c *Controller) StageManagedServiceArtifact(ctx context.Context, name, expectedVersion string) (ManagedServiceArtifact, error) {
	if err := ctx.Err(); err != nil {
		return ManagedServiceArtifact{}, err
	}
	manifest, err := c.managedServiceArtifactManifest(name)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	if expectedVersion == "" || manifest.ManagedService.Artifact.Version != expectedVersion {
		return ManagedServiceArtifact{}, fmt.Errorf("resource %s release changed before staging (expected %q, declared %q)", name, expectedVersion, manifest.ManagedService.Artifact.Version)
	}
	if strings.EqualFold(manifest.ManagedService.Artifact.Verification, "host-tool") {
		return ManagedServiceArtifact{}, fmt.Errorf("resource %s adopts a host tool; it has no resource-owned pinned acquisition", name)
	}
	if err := ensureManagedServiceArtifact(ctx, c, manifest); err != nil {
		return ManagedServiceArtifact{}, err
	}
	return c.inspectManagedServiceArtifact(ctx, manifest)
}

func (c *Controller) managedServiceArtifactManifest(name string) (ResourceManifest, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return ResourceManifest{}, fmt.Errorf("invalid resource name")
	}
	manifest, err := c.LoadManifest(filepath.Join(c.Root, "resources", name, "resource.json"))
	if err != nil {
		return ResourceManifest{}, err
	}
	if manifest.Name != name || manifest.Driver != "managed-service" || manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return ResourceManifest{}, fmt.Errorf("resource %s has no managed-service acquisition authority", name)
	}
	return manifest, nil
}

func (c *Controller) inspectManagedServiceArtifact(ctx context.Context, manifest ResourceManifest) (ManagedServiceArtifact, error) {
	return c.inspectManagedServiceArtifactWithRunning(ctx, manifest, nil)
}

// verifiedRunning is supplied only after native ownership and retained digest verification.
// Reuse that proof for identical bytes instead of hashing a large service tree twice.
func (c *Controller) inspectManagedServiceArtifactWithRunning(ctx context.Context, manifest ResourceManifest, verifiedRunning *ManagedServiceState) (ManagedServiceArtifact, error) {
	if err := ctx.Err(); err != nil {
		return ManagedServiceArtifact{}, err
	}
	path, err := managedServiceArtifactPath(c, manifest)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	artifact, err := managedServiceArtifactForLaunch(ctx, manifest, path)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	if strings.EqualFold(artifact.Verification, "host-tool") {
		if verifiedRunning == nil || path != verifiedRunning.ArtifactPath {
			return ManagedServiceArtifact{}, fmt.Errorf("resource %s has no checksum-pinned artifact", manifest.Name)
		}
		artifact.SHA256 = verifiedRunning.ArtifactSHA256
	}
	info := ManagedServiceArtifact{Resource: manifest.Name, Path: path, Version: artifact.Version, SHA256: artifact.SHA256}
	if verifiedRunning == nil || path != verifiedRunning.ArtifactPath || artifact.SHA256 != verifiedRunning.ArtifactSHA256 {
		if err := artifact.VerifyFileContext(ctx, path); err != nil {
			return info, err
		}
	}
	return info, nil
}

// Verify the retained image against the identity witnessed at launch, independent
// of a newer desired release. Process ownership has already been checked by Status.
func verifyManagedServiceRunningArtifact(ctx context.Context, state ManagedServiceState) error {
	if state.ArtifactSHA256 == "" {
		return fmt.Errorf("running artifact has no recorded checksum")
	}
	info, err := os.Stat(state.ArtifactPath)
	if err != nil {
		return err
	}
	layout := "file"
	if info.IsDir() {
		layout = "dir"
	}
	return binaryfetch.VerifyArtifactContext(ctx, state.ArtifactPath, layout, state.ArtifactSHA256)
}
