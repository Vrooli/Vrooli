package release

import (
	"encoding/json"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"time"

	"github.com/vrooli/vrooli/packages/artifactlease"
	"github.com/vrooli/vrooli/packages/cloudrelease"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
)

// DefaultLeaseTTL is how long a release stays protected from garbage
// collection after it was built or last renewed. Activation renews it.
const DefaultLeaseTTL = 30 * 24 * time.Hour

// Verification failure reasons carried in details.reason of a
// release_verification_failed error.
const (
	ReasonNotFound   = "release_not_found"
	ReasonIncomplete = "release_incomplete"
)

// Store addresses releases beneath a bundle store directory.
type Store struct {
	// Dir is the bundle store directory (bundle.GetLocalBundlesDir()).
	Dir string
}

// Release is one stored release artifact set.
type Release struct {
	Digest   string                `json:"release_digest"`
	Dir      string                `json:"dir"`
	Complete bool                  `json:"complete"`
	Manifest cloudrelease.Manifest `json:"manifest"`
	Inputs   domain.ReleaseInputs  `json:"inputs"`
}

// BundlePath is the bundle archive inside the release.
func (r Release) BundlePath() string { return filepath.Join(r.Dir, cloudrelease.BundleFileName) }

// NativeCLIPath is the control-plane binary inside the release.
func (r Release) NativeCLIPath() string {
	return filepath.Join(r.Dir, cloudrelease.NativeCLIFileName(r.Manifest.NativeCLI.GOOS, r.Manifest.NativeCLI.GOARCH))
}

// ManifestPath is the release manifest inside the release.
func (r Release) ManifestPath() string { return filepath.Join(r.Dir, cloudrelease.ManifestFileName) }

// Summary projects the release for API responses.
func (r Release) Summary() domain.ReleaseSummary {
	return domain.ReleaseSummary{
		Digest: r.Digest, Dir: r.Dir, Complete: r.Complete,
		BundleSHA256: r.Manifest.BundleSHA256, NativeCLI: r.Inputs.NativeCLI,
		ClosureDigest: r.Manifest.ClosureDigest, ConfigurationDigest: r.Manifest.ConfigurationDigest,
		Provenance: r.Inputs.Provenance, BuiltAt: r.Inputs.BuiltAt,
	}
}

// ReleasesDir is where release directories live.
func (s Store) ReleasesDir() string { return filepath.Join(s.Dir, bundle.ReleasesDirName) }

// Path returns the directory for a digest, refusing anything that is not a
// lowercase sha256 hex string (so a digest can never traverse).
func (s Store) Path(digest string) (string, error) {
	if !cloudrelease.IsDigest(digest) {
		return "", apierrors.Newf(apierrors.CodeInvalidRequest, "release digest %q is not a lowercase sha256 hex string", digest)
	}
	return filepath.Join(s.ReleasesDir(), digest), nil
}

// Complete reports whether the release directory carries the completion
// marker. A directory without it is not activatable.
func (s Store) Complete(digest string) bool {
	dir, err := s.Path(digest)
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, cloudrelease.CompleteMarker))
	return err == nil
}

// Load reads a complete release. A missing or incomplete directory is a
// typed release_verification_failed with details.reason naming which.
func (s Store) Load(digest string) (Release, error) {
	dir, err := s.Path(digest)
	if err != nil {
		return Release{}, err
	}
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Release{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "release %s is not in the store", digest).WithDetail("reason", ReasonNotFound).WithDetail("release_digest", digest)
		}
		return Release{}, apierrors.Internal("stat release", err)
	}
	if !s.Complete(digest) {
		return Release{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "release %s is incomplete and cannot be used", digest).WithDetail("reason", ReasonIncomplete).WithDetail("release_digest", digest)
	}
	return loadReleaseDir(dir, true)
}

// LoadDir reads a release directory outside a store (for example the parent
// of a bundle path). It reports completeness rather than refusing an
// incomplete tree; callers that need a usable release check Complete.
func LoadDir(dir string) (Release, error) {
	_, err := os.Stat(filepath.Join(dir, cloudrelease.CompleteMarker))
	return loadReleaseDir(dir, err == nil)
}

func loadReleaseDir(dir string, complete bool) (Release, error) {
	raw, err := os.ReadFile(filepath.Join(dir, cloudrelease.ManifestFileName))
	if err != nil {
		return Release{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "read release manifest: %v", err).WithDetail("reason", "release_manifest_invalid")
	}
	manifest, err := cloudrelease.ParseManifest(raw)
	if err != nil {
		return Release{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "%v", err).WithDetail("reason", "release_manifest_invalid")
	}
	if filepath.Base(dir) != manifest.ReleaseDigest {
		return Release{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "release directory %s does not match manifest release_digest %s", filepath.Base(dir), manifest.ReleaseDigest).WithDetail("reason", "release_digest_mismatch")
	}
	var inputs domain.ReleaseInputs
	if rawInputs, err := os.ReadFile(filepath.Join(dir, cloudrelease.InputsFileName)); err == nil {
		if err := json.Unmarshal(rawInputs, &inputs); err != nil {
			return Release{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "parse %s: %v", cloudrelease.InputsFileName, err).WithDetail("reason", "release_inputs_invalid")
		}
	}
	return Release{Digest: manifest.ReleaseDigest, Dir: dir, Complete: complete, Manifest: manifest, Inputs: inputs}, nil
}

// List returns every release directory, complete or not, newest first by
// built_at (incomplete ones sort last).
func (s Store) List() ([]Release, error) {
	entries, err := os.ReadDir(s.ReleasesDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Release{}, nil
		}
		return nil, err
	}
	releases := []Release{}
	for _, entry := range entries {
		if !entry.IsDir() || !cloudrelease.IsDigest(entry.Name()) {
			continue
		}
		rel, err := loadReleaseDir(filepath.Join(s.ReleasesDir(), entry.Name()), s.Complete(entry.Name()))
		if err != nil {
			continue
		}
		releases = append(releases, rel)
	}
	sort.Slice(releases, func(i, j int) bool {
		if releases[i].Complete != releases[j].Complete {
			return releases[i].Complete
		}
		return releases[i].Inputs.BuiltAt > releases[j].Inputs.BuiltAt
	})
	return releases, nil
}

// Protect claims (or renews) the artifact lease of a release so garbage
// collection leaves it alone. ownerModule names the deployment or caller that
// depends on it.
func (s Store) Protect(digest, ownerModule string, ttl time.Duration, now time.Time) (artifactlease.Lease, error) {
	dir, err := s.Path(digest)
	if err != nil {
		return artifactlease.Lease{}, err
	}
	if ttl <= 0 {
		ttl = DefaultLeaseTTL
	}
	return artifactlease.Renew(dir, currentOwner(), ownerModule, ttl, now)
}

// ProtectedBundleSHA256s lists the bundle digests of every leased release.
func (s Store) ProtectedBundleSHA256s(now time.Time) ([]string, error) {
	return bundle.ReleaseProtectedBundleSHA256s(s.ReleasesDir(), now)
}

func currentOwner() artifactlease.Owner {
	owner := artifactlease.Owner{}
	if host, err := os.Hostname(); err == nil {
		owner.Node = host
	}
	if current, err := user.Current(); err == nil {
		owner.User = current.Username
	}
	return owner
}
