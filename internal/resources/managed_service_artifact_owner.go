package resources

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/binaryfetch"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
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
	return c.stageManagedServiceArtifact(ctx, manifest, expectedVersion)
}

func (c *Controller) stageManagedServiceArtifact(ctx context.Context, manifest ResourceManifest, expectedVersion string) (ManagedServiceArtifact, error) {
	name := manifest.Name
	if expectedVersion == "" || manifest.ManagedService.Artifact.Version != expectedVersion {
		return ManagedServiceArtifact{}, fmt.Errorf("resource %s release changed before staging (expected %q, declared %q)", name, expectedVersion, manifest.ManagedService.Artifact.Version)
	}
	if strings.EqualFold(manifest.ManagedService.Artifact.Verification, "host-tool") {
		return ManagedServiceArtifact{}, fmt.Errorf("resource %s adopts a host tool; it has no resource-owned pinned acquisition", name)
	}
	if err := c.requireManagedArtifactSource(ctx, manifest); err != nil {
		return ManagedServiceArtifact{}, err
	}
	if err := ensureManagedServiceArtifactForStaging(ctx, c, manifest); err != nil {
		return ManagedServiceArtifact{}, err
	}
	if err := c.requireManagedArtifactSource(ctx, manifest); err != nil {
		return ManagedServiceArtifact{}, err
	}
	path, err := managedServiceArtifactStorePath(c, manifest)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	target, err := managedServiceAcquisitionTarget(ctx, manifest)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	artifact, err := managedServiceArtifactForTarget(manifest, target)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	if err := artifact.VerifyFileContext(ctx, path); err != nil {
		return ManagedServiceArtifact{}, err
	}
	return ManagedServiceArtifact{Resource: name, Path: path, Version: artifact.Version, SHA256: artifact.SHA256}, nil
}

func (c *Controller) requireManagedArtifactSource(ctx context.Context, manifest ResourceManifest) error {
	choice, selected, err := c.ManagedServiceSource(ctx, manifest.Name)
	if err != nil {
		return err
	}
	if manifest.ManagedService.SourceSelectionRequired && !selected {
		return fmt.Errorf("resource %s requires an explicit managed source choice before staging", manifest.Name)
	}
	if selected && choice.Mode == "host" {
		return fmt.Errorf("resource %s uses an adopted host executable. Its host package manager owns updates", manifest.Name)
	}
	if selected && choice.Mode != "managed" {
		return fmt.Errorf("resource %s source changed to %q; managed staging cannot evaluate or update an adopted host installation", manifest.Name, choice.Mode)
	}
	return nil
}

func (c *Controller) managedServiceArtifactManifest(name string) (ResourceManifest, error) {
	return c.managedServiceArtifactManifestSnapshot(name, "")
}

func (c *Controller) managedServiceArtifactManifestSnapshot(name, expectedManifestSHA256 string) (ResourceManifest, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return ResourceManifest{}, fmt.Errorf("invalid resource name")
	}
	path := filepath.Join(c.Root, "resources", name, "resource.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return ResourceManifest{}, err
	}
	if expectedManifestSHA256 != "" && fmt.Sprintf("%x", sha256.Sum256(raw)) != expectedManifestSHA256 {
		return ResourceManifest{}, fmt.Errorf("resource %s manifest changed after acquisition admission", name)
	}
	manifest, err := manifestpkg.Parse(path, raw)
	if err != nil {
		return ResourceManifest{}, err
	}
	if manifest.Name != name || manifest.Driver != "managed-service" || manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return ResourceManifest{}, fmt.Errorf("resource %s has no managed-service acquisition authority", name)
	}
	return manifest, nil
}

// StageManagedServiceArtifactSnapshot retains the exact admitted source
// declaration through acquisition. It performs no lifecycle operation.
func (c *Controller) StageManagedServiceArtifactSnapshot(ctx context.Context, name, expectedVersion, manifestSHA256 string) (ManagedServiceArtifact, error) {
	if err := ctx.Err(); err != nil {
		return ManagedServiceArtifact{}, err
	}
	if len(manifestSHA256) != 64 {
		return ManagedServiceArtifact{}, fmt.Errorf("an admitted resource manifest SHA-256 is required")
	}
	manifest, err := c.managedServiceArtifactManifestSnapshot(name, manifestSHA256)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	return c.stageManagedServiceArtifact(ctx, manifest, expectedVersion)
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
	artifact, err := managedServiceArtifactForLaunch(ctx, c, manifest, path)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	if strings.EqualFold(artifact.Verification, "host-tool") {
		if verifiedRunning != nil && path == verifiedRunning.ArtifactPath {
			artifact.SHA256 = verifiedRunning.ArtifactSHA256
		} else {
			choice, selected, sourceErr := c.ManagedServiceSource(ctx, manifest.Name)
			if sourceErr != nil || !selected || choice.Mode != "host" || path != choice.Executable {
				return ManagedServiceArtifact{}, fmt.Errorf("resource %s has no verified host-tool adoption record", manifest.Name)
			}
			artifact.SHA256 = choice.SHA256
		}
	}
	info := ManagedServiceArtifact{Resource: manifest.Name, Path: path, Version: artifact.Version, SHA256: artifact.SHA256}
	if strings.EqualFold(artifact.Verification, "host-tool") {
		if manifest.Name == openRGBResourceName {
			actual, err := inspectOpenRGBAdoptedSource(ctx, path)
			if err != nil || actual != artifact.SHA256 {
				return info, errors.Join(errors.New("OpenRGB adopted native profile or dependency closure changed after selection"), err)
			}
			return info, nil
		}
		actual, hashErr := managedFileSHA256(path)
		if hashErr != nil {
			return info, hashErr
		}
		if actual != artifact.SHA256 {
			return info, fmt.Errorf("resource %s adopted executable changed after source selection", manifest.Name)
		}
		return info, nil
	}
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
