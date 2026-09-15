package cloudtarget

import (
	"errors"
	"os"
	"path/filepath"
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
		report.Deleted = append(report.Deleted, digest)
		report.ReclaimedBytes += facts.SizeBytes
	}
	return report, nil
}
