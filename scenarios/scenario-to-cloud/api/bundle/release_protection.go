package bundle

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/vrooli/vrooli/packages/artifactlease"
	"github.com/vrooli/vrooli/packages/cloudrelease"
)

// ReleasesDirName is the directory beneath the bundle store that holds
// release artifact sets (releases/<release_digest>/…).
const ReleasesDirName = "releases"

// ReleaseProtectedBundleSHA256s returns the bundle digests of every release
// under releasesDir that holds an unexpired artifact lease. Garbage collection
// treats those digests as protected: a release the cloud side still owns
// (active, rollback predecessor) is never reclaimed for convenience.
//
// Leases live beside the release directory at
// releases/<digest>.lease.json, the shape packages/artifactlease defines. A
// release without a lease, or with an expired one, is not protected by this
// rule (the retention count still applies).
func ReleaseProtectedBundleSHA256s(releasesDir string, now time.Time) ([]string, error) {
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var protected []string
	for _, entry := range entries {
		if !entry.IsDir() || !cloudrelease.IsDigest(entry.Name()) {
			continue
		}
		releaseDir := filepath.Join(releasesDir, entry.Name())
		lease, found, err := artifactlease.Load(releaseDir)
		if err != nil || !found {
			continue
		}
		expires, err := time.Parse(time.RFC3339Nano, lease.ExpiresAt)
		if err != nil || !expires.After(now) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(releaseDir, cloudrelease.ManifestFileName))
		if err != nil {
			continue
		}
		manifest, err := cloudrelease.ParseManifest(raw)
		if err != nil {
			continue
		}
		protected = append(protected, manifest.BundleSHA256)
	}
	sort.Strings(protected)
	return protected, nil
}

// releaseProtectionFn resolves the lease-protected bundle digests for GC. The
// default reads the local bundle store; tests inject a fixture directory.
var releaseProtectionFn = func(now time.Time) ([]string, error) {
	dir, err := GetLocalBundlesDir()
	if err != nil {
		return nil, err
	}
	return ReleaseProtectedBundleSHA256s(filepath.Join(dir, ReleasesDirName), now)
}
