package resources

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/artifactlock"
)

// The exact retained envelope measured in the isolated release qualification.
// This bound belongs to the same frozen release profile as its existing digest.
const openRGBAppImageDownloadBytes int64 = 37528768

func managedServiceExactDownloadBytes(manifest ResourceManifest, target binaryfetch.AcquisitionTarget) (int64, bool) {
	if manifest.Name != openRGBResourceName {
		return 0, false
	}
	profile, err := openRGBReleaseProfileForCurrentHost()
	if err != nil || profile.validate() != nil {
		return 0, false
	}
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil || manifest.ManagedService.Artifact.Version != openRGBAppImageVersion || target.SHA256 != profile.EnvelopeSHA256 || target.Layout != "file" || target.Archive != "" && target.Archive != "none" || manifest.ManagedService.Acquisition.EffectiveKind(target) != "url" {
		return 0, false
	}
	return profile.DownloadBytes, true
}

// ManagedArtifactStagingPlan enumerates writes for the capacity owner. The
// executor separately revalidates the admitted manifest snapshot before writes.
type ManagedArtifactStagingPlan struct {
	Path                string
	DownloadBytes       int64
	MetadataDirectories []string
}

// PlanManagedServiceArtifactStage resolves every declared staging write path
// without acquiring bytes, creating directories or invoking a service.
func (c *Controller) PlanManagedServiceArtifactStage(ctx context.Context, name, version, manifestSHA256 string) (ManagedArtifactStagingPlan, error) {
	if len(manifestSHA256) != 64 {
		return ManagedArtifactStagingPlan{}, errors.New("admitted resource manifest SHA-256 is required")
	}
	manifest, err := c.managedServiceArtifactManifestSnapshot(name, manifestSHA256)
	if err != nil {
		return ManagedArtifactStagingPlan{}, err
	}
	if manifest.ManagedService.Artifact.Version != version {
		return ManagedArtifactStagingPlan{}, errors.New("admitted resource release changed")
	}
	target, err := manifest.ManagedService.Acquisition.Resolve(binaryfetch.Facts{"os": runtime.GOOS, "arch": runtime.GOARCH})
	if err != nil {
		return ManagedArtifactStagingPlan{}, err
	}
	bytes, bounded := managedServiceExactDownloadBytes(manifest, target)
	if !bounded || target.Auth != nil || !binaryfetch.UsesOnlyBuildTimeFacts(target) || len(manifest.ManagedService.DataArtifacts) != 0 || manifest.ManagedService.Acquisition.Provenance.Kind != "none" {
		return ManagedArtifactStagingPlan{}, errors.New("resource has no complete bounded raw-file acquisition profile")
	}
	artifact, err := managedServiceArtifactForTarget(manifest, target)
	if err != nil {
		return ManagedArtifactStagingPlan{}, err
	}
	if artifact.SHA256 != target.SHA256 {
		return ManagedArtifactStagingPlan{}, errors.New("bounded resource target and artifact digest differ")
	}
	path, err := managedServiceArtifactStorePath(c, manifest)
	if err != nil {
		return ManagedArtifactStagingPlan{}, err
	}
	paths, err := resourceStoragePaths(name)
	if err != nil {
		return ManagedArtifactStagingPlan{}, err
	}
	sourceState, err := c.operatorStateService().Path(ctx)
	if err != nil {
		return ManagedArtifactStagingPlan{}, err
	}
	lock, err := artifactlock.LockPath("resource:" + name)
	if err != nil {
		return ManagedArtifactStagingPlan{}, err
	}
	return ManagedArtifactStagingPlan{Path: path, DownloadBytes: bytes, MetadataDirectories: []string{paths.ConfigDir, paths.DataDir, paths.CacheDir, paths.LogsDir, paths.StateDir, filepath.Dir(lock), filepath.Dir(sourceState)}}, nil
}
