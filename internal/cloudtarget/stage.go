package cloudtarget

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/binaryfetch"
)

// StageRequest stages one verified release into the deployment's release
// tree. Verification always runs first; nothing is written on a refusal.
type StageRequest struct {
	Effect EffectRequest
	Verify VerifyRequest
}

// ReleaseState is the durable state of one release directory.
type ReleaseState string

const (
	ReleaseStateComplete   ReleaseState = "complete"
	ReleaseStateIncomplete ReleaseState = "incomplete"
	ReleaseStateStaging    ReleaseState = "staging"
	ReleaseStateMissing    ReleaseState = "missing"
)

// releaseState inspects a release directory. Only a directory that carries
// the manifest and no incomplete marker is complete.
func releaseState(dir string) ReleaseState {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return ReleaseStateMissing
	}
	if _, err := os.Stat(filepath.Join(dir, incompleteMarker)); err == nil {
		return ReleaseStateIncomplete
	}
	if _, err := os.Stat(filepath.Join(dir, releaseManifestFile)); err != nil {
		return ReleaseStateIncomplete
	}
	return ReleaseStateComplete
}

// Stage verifies then extracts a release into an owned inactive staging
// directory, promoting it to releases/<digest>/ only on full success.
func (s *Store) Stage(ctx context.Context, req StageRequest) (EffectResult, error) {
	req.Effect.Verb = "release.stage"
	manifest, err := LoadManifest(req.Verify.ManifestPath)
	if err != nil {
		return EffectResult{}, err
	}
	req.Effect.Input = map[string]any{"release_digest": manifest.ReleaseDigest, "bundle_sha256": manifest.BundleSHA256}
	return s.RunEffect(ctx, req.Effect, func(ctx context.Context) (map[string]any, Outcome, error) {
		report, manifest, err := Verify(ctx, req.Verify)
		details := report.details()
		if err != nil {
			return details, OutcomeFailed, err
		}
		releaseDir, err := s.ReleaseDir(req.Effect.DeploymentID, manifest.ReleaseDigest)
		if err != nil {
			return details, OutcomeFailed, err
		}
		details["release_dir"] = releaseDir
		switch releaseState(releaseDir) {
		case ReleaseStateComplete:
			details["state"] = string(ReleaseStateComplete)
			return details, OutcomeUnchanged, nil
		case ReleaseStateIncomplete:
			return details, OutcomeFailed, refuse(CodeReleaseIncomplete, "release directory %s exists but is incomplete; it is not owned by this stage", releaseDir)
		}
		staging, err := s.stagingDir(req.Effect.DeploymentID, manifest.ReleaseDigest, req.Effect.OperationID)
		if err != nil {
			return details, OutcomeFailed, err
		}
		if err := os.RemoveAll(staging); err != nil {
			return details, OutcomeFailed, fail(CodeStageFailed, "clear stale staging directory: %v", err)
		}
		cleanup := func() { _ = os.RemoveAll(staging) }
		if err := os.MkdirAll(staging, dirMode); err != nil {
			return details, OutcomeFailed, fail(CodeStageFailed, "create staging directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(staging, incompleteMarker), []byte(req.Effect.OperationID+"\n"), fileMode); err != nil {
			cleanup()
			return details, OutcomeFailed, fail(CodeStageFailed, "mark staging directory: %v", err)
		}
		budget := req.Verify.TimeBudget
		if budget <= 0 {
			budget = DefaultVerifyBudget
		}
		summary, err := binaryfetch.ExtractArchiveBounded(req.Verify.ArchivePath, ReleaseArchiveFormat, staging, manifest.ExtractOptions(time.Now().Add(budget)))
		details["extracted"] = summary
		if err != nil {
			cleanup()
			return details, OutcomeFailed, archiveError(err)
		}
		if err := s.fault("stage:after_extract"); err != nil {
			cleanup()
			return details, OutcomeFailed, fail(CodeStageFailed, "%v", err)
		}
		if err := copyFile(req.Verify.ManifestPath, filepath.Join(staging, releaseManifestFile)); err != nil {
			cleanup()
			return details, OutcomeFailed, fail(CodeStageFailed, "record release manifest: %v", err)
		}
		if err := os.Remove(filepath.Join(staging, incompleteMarker)); err != nil {
			cleanup()
			return details, OutcomeFailed, fail(CodeStageFailed, "clear incomplete marker: %v", err)
		}
		if err := os.Rename(staging, releaseDir); err != nil {
			if releaseState(releaseDir) == ReleaseStateComplete {
				// A concurrent stage of the same digest won the rename. The
				// content is identical by digest, so this stage is unchanged.
				cleanup()
				details["state"] = string(ReleaseStateComplete)
				return details, OutcomeUnchanged, nil
			}
			cleanup()
			return details, OutcomeFailed, fail(CodeStageFailed, "promote staging directory: %v", err)
		}
		details["state"] = string(ReleaseStateComplete)
		return details, OutcomeSucceeded, nil
	})
}

func copyFile(src, dst string) error {
	raw, err := os.ReadFile(src) //nolint:gosec // operator-provided manifest path
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, fileMode)
}

// ReleaseEntry is one row of `release list`.
type ReleaseEntry struct {
	Digest string       `json:"digest"`
	State  ReleaseState `json:"state"`
	Role   string       `json:"role"`
	Path   string       `json:"path"`
	ReleaseFacts
}

// ReleaseListing is the durable release view for one deployment.
type ReleaseListing struct {
	DeploymentID          string            `json:"deployment_id"`
	Active                *ActiveRelease    `json:"active,omitempty"`
	InterruptedActivation *ActivationIntent `json:"interrupted_activation,omitempty"`
	Fence                 Fence             `json:"fence"`
	Releases              []ReleaseEntry    `json:"releases"`
}

// ListReleases reports staged, active and previous releases plus any
// activation that was interrupted between the runtime switch and the pointer
// commit. Restart reconciliation reads this to learn the actual active release.
func (s *Store) ListReleases(deploymentID string) (ReleaseListing, error) {
	releases, err := s.releasesDir(deploymentID)
	if err != nil {
		return ReleaseListing{}, err
	}
	listing := ReleaseListing{DeploymentID: deploymentID, Releases: []ReleaseEntry{}}
	if listing.Fence, err = s.ReadFence(deploymentID); err != nil {
		return ReleaseListing{}, err
	}
	active, err := s.ReadActive(deploymentID)
	if err != nil {
		return ReleaseListing{}, err
	}
	listing.Active = active
	intent, err := s.readIntent(deploymentID)
	if err != nil {
		return ReleaseListing{}, err
	}
	listing.InterruptedActivation = intent
	entries, err := os.ReadDir(releases)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return listing, nil
		}
		return ReleaseListing{}, fail(CodeStoreIO, "list releases: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(releases, entry.Name())
		row := ReleaseEntry{Digest: entry.Name(), Path: path, Role: "staged"}
		if idx := indexOf(entry.Name(), stagingInfix); idx > 0 {
			row.Digest = entry.Name()[:idx]
			row.State = ReleaseStateStaging
			row.Role = "staging"
		} else {
			row.State = releaseState(path)
			row.ReleaseFacts = releaseFacts(path)
		}
		if active != nil && row.State == ReleaseStateComplete {
			switch row.Digest {
			case active.ActiveRelease:
				row.Role = "active"
			case active.PreviousRelease:
				row.Role = "previous"
			}
		}
		listing.Releases = append(listing.Releases, row)
	}
	return listing, nil
}

func indexOf(value, infix string) int {
	for i := 0; i+len(infix) <= len(value); i++ {
		if value[i:i+len(infix)] == infix {
			return i
		}
	}
	return -1
}
