package cloudtarget

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/cloudrelease"
)

// ReleaseFacts are the size, freshness and content identities of one staged
// release, read from its directory. They let the cloud side plan retention
// (newest per scenario, lease protection by bundle digest) without a shell.
type ReleaseFacts struct {
	SizeBytes    int64  `json:"size_bytes"`
	ModTime      string `json:"mod_time,omitempty"`
	BundleSHA256 string `json:"bundle_sha256,omitempty"`
}

func releaseFacts(dir string) ReleaseFacts {
	facts := ReleaseFacts{}
	if info, err := os.Stat(dir); err == nil {
		facts.ModTime = info.ModTime().UTC().Format(time.RFC3339)
	}
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, ierr := d.Info(); ierr == nil {
			facts.SizeBytes += info.Size()
		}
		return nil
	})
	if raw, err := os.ReadFile(filepath.Join(dir, releaseManifestFile)); err == nil { //nolint:gosec // owner-written path
		facts.SizeBytes = int64(len(raw))
		if manifest, perr := cloudrelease.ParseManifest(raw); perr == nil {
			facts.BundleSHA256 = manifest.BundleSHA256
		}
	}
	return facts
}

// releaseFactsSummary is used by release list, which is a hot status path.
// Walking an entire staged release there makes a large control-plane bundle
// block workload startup. Retention planning still uses releaseFacts when it
// needs the exact byte count.
func releaseFactsSummary(dir string) ReleaseFacts {
	facts := ReleaseFacts{}
	if info, err := os.Stat(dir); err == nil {
		facts.ModTime = info.ModTime().UTC().Format(time.RFC3339)
	}
	if raw, err := os.ReadFile(filepath.Join(dir, releaseManifestFile)); err == nil { //nolint:gosec // owner-written path
		facts.SizeBytes = int64(len(raw))
		if manifest, perr := cloudrelease.ParseManifest(raw); perr == nil {
			facts.BundleSHA256 = manifest.BundleSHA256
		}
	}
	return facts
}

// PruneRequest names the complete releases the cloud side wants removed.
// The owner refuses the active and previous releases and anything that is
// still staging, whatever the request says: retention is planned by the
// cloud, safety is decided here.
type PruneRequest struct {
	DeploymentID string
	Releases     []string
}

// PruneReport is the owner's answer: what was removed, what was refused and
// why, and the bytes reclaimed.
type PruneReport struct {
	DeploymentID   string            `json:"deployment_id"`
	Deleted        []string          `json:"deleted"`
	Refused        map[string]string `json:"refused,omitempty"`
	ReclaimedBytes int64             `json:"reclaimed_bytes"`
}

// PruneReleases removes the named complete releases that are neither active
// nor previous. It is idempotent: a digest that is already absent is
// reported as deleted with zero bytes.
func (s *Store) PruneReleases(req PruneRequest) (PruneReport, error) {
	report := PruneReport{DeploymentID: req.DeploymentID, Deleted: []string{}, Refused: map[string]string{}}
	if len(req.Releases) == 0 {
		return report, fail(CodeInvalidArgument, "prune names no releases")
	}
	active, err := s.ReadActive(req.DeploymentID)
	if err != nil {
		return report, err
	}
	intent, err := s.readIntent(req.DeploymentID)
	if err != nil {
		return report, err
	}
	for _, digest := range req.Releases {
		dir, err := s.ReleaseDir(req.DeploymentID, digest)
		if err != nil {
			return report, err
		}
		switch {
		case active != nil && digest == active.ActiveRelease:
			report.Refused[digest] = "active"
			continue
		case active != nil && digest == active.PreviousRelease:
			report.Refused[digest] = "previous"
			continue
		case intent != nil && (digest == intent.Candidate || digest == intent.Previous):
			report.Refused[digest] = "interrupted_activation"
			continue
		}
		if _, err := os.Stat(dir); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				report.Deleted = append(report.Deleted, digest)
				continue
			}
			return report, fail(CodeStoreIO, "stat release %s: %v", digest, err)
		}
		if releaseState(dir) == ReleaseStateStaging {
			report.Refused[digest] = "staging"
			continue
		}
		facts := releaseFacts(dir)
		if err := os.RemoveAll(dir); err != nil {
			return report, fail(CodeStoreIO, "remove release %s: %v", digest, err)
		}
		if err := s.removeUnreferencedBundle(facts.BundleSHA256); err != nil {
			return report, err
		}
		report.Deleted = append(report.Deleted, digest)
		report.ReclaimedBytes += facts.SizeBytes
	}
	if err := s.sweepUnreferencedBundles(); err != nil {
		return report, err
	}
	return report, nil
}

// removeUnreferencedBundle removes the delivery cache for a pruned release
// only when no release in any deployment still references its bundle digest.
// Release activation copies the bundle contents into the immutable release
// tree, so the delivery cache is no longer needed after pruning. The scan is
// deliberately owner-local and conservative across deployments.
func (s *Store) removeUnreferencedBundle(bundleSHA256 string) error {
	bundleSHA256 = strings.TrimSpace(bundleSHA256)
	if bundleSHA256 == "" {
		return nil
	}
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fail(CodeStoreIO, "list deployments for bundle cleanup: %v", err)
	}
	for _, deployment := range entries {
		if !deployment.IsDir() {
			continue
		}
		releases := filepath.Join(s.Root, deployment.Name(), releasesDirName)
		walkErr := filepath.WalkDir(releases, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || filepath.Base(path) != releaseManifestFile {
				return nil
			}
			raw, readErr := os.ReadFile(path) //nolint:gosec // path is owner-derived
			if readErr != nil {
				return nil
			}
			manifest, parseErr := cloudrelease.ParseManifest(raw)
			if parseErr == nil && manifest.BundleSHA256 == bundleSHA256 {
				return errBundleStillReferenced
			}
			return nil
		})
		if errors.Is(walkErr, errBundleStillReferenced) {
			return nil
		}
		if walkErr != nil && !errors.Is(walkErr, os.ErrNotExist) {
			return fail(CodeStoreIO, "scan releases for bundle cleanup: %v", walkErr)
		}
	}
	cacheDir := filepath.Join(s.bundleCacheDir(), "sha256-"+bundleSHA256)
	if err := os.RemoveAll(cacheDir); err != nil {
		return fail(CodeStoreIO, "remove bundle cache %s: %v", bundleSHA256, err)
	}
	return nil
}

var errBundleStillReferenced = errors.New("bundle still referenced")

// sweepUnreferencedBundles removes delivery caches left behind by older
// releases whose release directories were already retired. It runs only
// through the target owner's store and protects references across every
// deployment on the host.
func (s *Store) sweepUnreferencedBundles() error {
	referenced := map[string]bool{}
	deployments, err := os.ReadDir(s.Root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fail(CodeStoreIO, "list deployments for bundle sweep: %v", err)
	}
	for _, deployment := range deployments {
		if !deployment.IsDir() {
			continue
		}
		releases := filepath.Join(s.Root, deployment.Name(), releasesDirName)
		walkErr := filepath.WalkDir(releases, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || filepath.Base(path) != releaseManifestFile {
				return nil
			}
			raw, readErr := os.ReadFile(path) //nolint:gosec // path is owner-derived
			if readErr != nil {
				return nil
			}
			manifest, parseErr := cloudrelease.ParseManifest(raw)
			if parseErr == nil && manifest.BundleSHA256 != "" {
				referenced[manifest.BundleSHA256] = true
			}
			return nil
		})
		if walkErr != nil && !errors.Is(walkErr, os.ErrNotExist) {
			return fail(CodeStoreIO, "scan releases for bundle sweep: %v", walkErr)
		}
	}
	bundlesDir := s.bundleCacheDir()
	bundles, err := os.ReadDir(bundlesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fail(CodeStoreIO, "list bundle cache: %v", err)
	}
	for _, entry := range bundles {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "sha256-") {
			continue
		}
		sha := strings.TrimPrefix(entry.Name(), "sha256-")
		if !digestPattern.MatchString(sha) || referenced[sha] {
			continue
		}
		if err := os.RemoveAll(filepath.Join(bundlesDir, entry.Name())); err != nil {
			return fail(CodeStoreIO, "remove unreferenced bundle cache %s: %v", sha, err)
		}
	}
	return nil
}

// bundleCacheDir is rooted at the delivered repository workdir. The runtime
// home owns deployment state, while the workdir owns the temporary delivery
// cache used to stage release archives; they are intentionally separate.
func (s *Store) bundleCacheDir() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return filepath.Join(root, ".vrooli", "cloud", "bundles")
	}
	return filepath.Join(filepath.Dir(s.Root), "bundles")
}
