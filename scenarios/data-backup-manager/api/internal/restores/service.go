package restores

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"data-backup-manager/internal/clock"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/sources"
)

// Service is the application surface the restores handler depends on.
type Service interface {
	// RestoreTarget restores a snapshot to the caller-chosen location, then
	// applies the source-kind restore step. Returns the closed Restore record.
	RestoreTarget(ctx context.Context, targetID, destinationID, snapshotID, location string) (Restore, error)
	// VerifyTarget test-restores a snapshot to a scratch directory, checksums
	// the result, and records last_verified_at on success. The scratch dir is
	// ALWAYS cleaned up (even on failure). Returns the closed Restore record.
	VerifyTarget(ctx context.Context, targetID, destinationID, snapshotID string) (Restore, error)
	// GetRestore returns a single restore record by id.
	GetRestore(ctx context.Context, id string) (Restore, error)
	// ListRestores returns records newest-first, optionally filtered by target.
	ListRestores(ctx context.Context, targetID string, limit int) ([]Restore, error)
	// LastVerifiedByTarget returns the latest successful verify per target
	// (the proven-restorable rollup), optionally filtered to targetIDs.
	LastVerifiedByTarget(ctx context.Context, targetIDs []string) ([]VerifiedStatus, error)
}

// Deps bundles the seams the restores service orchestrates.
type Deps struct {
	Repo         Repository
	Targets      TargetLookup
	Destinations DestinationLookup
	Engine       engine.KopiaEngine
	Sources      *sources.Registry
	Clock        clock.Clock
	// ScratchRoot is the base directory scratch verify dirs are created under.
	// Empty uses the OS temp dir.
	ScratchRoot string
}

const defaultRestoreListLimit = 100

type service struct {
	deps Deps
}

// NewService constructs the production restore service.
func NewService(d Deps) Service { return &service{deps: d} }

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) RestoreTarget(ctx context.Context, targetID, destinationID, snapshotID, location string) (Restore, error) {
	targetID = strings.TrimSpace(targetID)
	destinationID = strings.TrimSpace(destinationID)
	snapshotID = strings.TrimSpace(snapshotID)
	location = strings.TrimSpace(location)
	if targetID == "" {
		return Restore{}, ErrInvalidRestore{Field: "target_id", Reason: "required"}
	}
	if destinationID == "" {
		return Restore{}, ErrInvalidRestore{Field: "destination_id", Reason: "required"}
	}
	if snapshotID == "" {
		return Restore{}, ErrInvalidRestore{Field: "snapshot_id", Reason: "required"}
	}
	if location == "" {
		return Restore{}, ErrInvalidRestore{Field: "location", Reason: "required"}
	}
	// Fail-closed safety (Contract Decision): refuse to restore into an existing
	// non-empty directory. v1 has no overwrite flag, so a restore must target an
	// empty or not-yet-existing path — this prevents clobbering live data.
	if nonEmpty, err := isNonEmptyDir(location); err != nil {
		return Restore{}, ErrInvalidRestore{Field: "location", Reason: fmt.Sprintf("cannot inspect restore target: %v", err)}
	} else if nonEmpty {
		return Restore{}, ErrInvalidRestore{
			Field:  "location",
			Reason: "restore target already exists and is not empty; choose an empty or new directory (overwriting existing data is not supported)",
		}
	}

	now := s.deps.Clock.Now().UTC()
	rec := Restore{
		TargetID:      targetID,
		DestinationID: destinationID,
		SnapshotID:    snapshotID,
		Mode:          ModeRestore,
		Status:        RestoreRequested,
		Location:      location,
		RequestedAt:   now,
	}

	target, err := s.deps.Targets.TargetForRestore(ctx, targetID)
	if err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("resolve target: %v", err))
	}
	dest, err := s.deps.Destinations.DestinationForRestore(ctx, destinationID)
	if err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("resolve destination: %v", err))
	}

	scratchBase := s.deps.ScratchRoot
	if scratchBase == "" {
		scratchBase = os.TempDir()
	}
	if err := os.MkdirAll(scratchBase, 0o755); err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("scratch root: %v", err))
	}
	artifactDir, err := os.MkdirTemp(scratchBase, "dbm-restore-"+sanitize(snapshotID)+"-"+sanitize(targetID)+"-")
	if err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("scratch dir: %v", err))
	}
	defer func() { _ = os.RemoveAll(artifactDir) }()

	if err := s.deps.Engine.SnapshotRestore(ctx, dest.Name, snapshotID, artifactDir); err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("snapshot restore: %v", err))
	}

	capturer, err := s.deps.Sources.Capturer(target.Kind)
	if err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("source capturer: %v", err))
	}
	if err := capturer.Restore(ctx, sources.RestoreSpec{
		Locator:      target.Locator,
		ArtifactPath: artifactDir,
		Target:       location,
	}); err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("source restore: %v", err))
	}

	rec.Status = RestoreRestored
	rec.FinishedAt = s.deps.Clock.Now().UTC()
	return s.deps.Repo.CreateRestore(ctx, rec)
}

func (s *service) VerifyTarget(ctx context.Context, targetID, destinationID, snapshotID string) (Restore, error) {
	targetID = strings.TrimSpace(targetID)
	destinationID = strings.TrimSpace(destinationID)
	snapshotID = strings.TrimSpace(snapshotID)
	if targetID == "" {
		return Restore{}, ErrInvalidRestore{Field: "target_id", Reason: "required"}
	}
	if destinationID == "" {
		return Restore{}, ErrInvalidRestore{Field: "destination_id", Reason: "required"}
	}
	if snapshotID == "" {
		return Restore{}, ErrInvalidRestore{Field: "snapshot_id", Reason: "required"}
	}

	now := s.deps.Clock.Now().UTC()
	rec := Restore{
		TargetID:      targetID,
		DestinationID: destinationID,
		SnapshotID:    snapshotID,
		Mode:          ModeVerify,
		Status:        RestoreRequested,
		RequestedAt:   now,
	}

	dest, err := s.deps.Destinations.DestinationForRestore(ctx, destinationID)
	if err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("resolve destination: %v", err))
	}

	// Create a scratch directory for the test-restore.
	scratchBase := s.deps.ScratchRoot
	if scratchBase == "" {
		scratchBase = os.TempDir()
	}
	scratchDir := filepath.Join(scratchBase, "dbm-verify-"+sanitize(snapshotID)+"-"+sanitize(targetID))
	if err := os.MkdirAll(scratchDir, 0o755); err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("scratch dir: %v", err))
	}
	// ALWAYS clean up the scratch dir, even on failure.
	defer func() { _ = os.RemoveAll(scratchDir) }()

	// Restore snapshot to scratch.
	if err := s.deps.Engine.SnapshotRestore(ctx, dest.Name, snapshotID, scratchDir); err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("snapshot restore: %v", err))
	}

	// Verify the snapshot's restorability (full 100% byte-verify).
	if err := s.deps.Engine.SnapshotVerify(ctx, dest.Name, snapshotID, 100); err != nil {
		// CRITICAL INVARIANT: a verify failure must NEVER set status=verified or
		// last_verified_at. This is OT-P0-006.
		return s.failRestore(ctx, rec, fmt.Sprintf("snapshot verify: %v", err))
	}

	// Compute a checksum of the scratch tree as evidence.
	checksum, err := checksumDir(scratchDir)
	if err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("checksum: %v", err))
	}

	rec.Status = RestoreVerified
	rec.Checksum = checksum
	rec.LastVerifiedAt = s.deps.Clock.Now().UTC()
	rec.FinishedAt = rec.LastVerifiedAt
	return s.deps.Repo.CreateRestore(ctx, rec)
}

func (s *service) GetRestore(ctx context.Context, id string) (Restore, error) {
	return s.deps.Repo.GetRestore(ctx, id)
}

func (s *service) ListRestores(ctx context.Context, targetID string, limit int) ([]Restore, error) {
	if limit <= 0 {
		limit = defaultRestoreListLimit
	}
	return s.deps.Repo.ListRestores(ctx, strings.TrimSpace(targetID), limit)
}

func (s *service) LastVerifiedByTarget(ctx context.Context, targetIDs []string) ([]VerifiedStatus, error) {
	return s.deps.Repo.LastVerifiedByTarget(ctx, targetIDs)
}

// failRestore records a failed restore and persists it. If persistence fails,
// returns the persistence error (the caller still sees a failure).
func (s *service) failRestore(ctx context.Context, rec Restore, errMsg string) (Restore, error) {
	rec.Status = RestoreFailed
	rec.Error = errMsg
	rec.FinishedAt = s.deps.Clock.Now().UTC()
	saved, persistErr := s.deps.Repo.CreateRestore(ctx, rec)
	if persistErr != nil {
		return Restore{}, fmt.Errorf("persist failed restore: %w (original: %s)", persistErr, errMsg)
	}
	return saved, nil
}

// isNonEmptyDir reports whether path exists, is a directory, and contains at
// least one entry. A non-existent path is not "non-empty" (the source restore
// step creates it). A path that exists but is not a directory is treated as
// non-empty (it would be clobbered), so the restore fails closed.
func isNonEmptyDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return true, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

// checksumDir computes a deterministic sha256 over the contents of all files
// under dir, sorted by path, as verify evidence. Symlinks are hashed by their
// link target (never followed) — a restored tree can contain links to
// directories (e.g. ~/.vrooli/state's codex session trees), which os.Open would
// choke on, and following them would reach outside the snapshot entirely.
func checksumDir(dir string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Include the relative path in the hash for determinism.
		rel, _ := filepath.Rel(dir, path)
		_, _ = fmt.Fprintf(h, "%s\n", rel)
		if d.Type()&fs.ModeSymlink != 0 {
			link, linkErr := os.Readlink(path)
			if linkErr != nil {
				return linkErr
			}
			_, _ = fmt.Fprintf(h, "symlink:%s\n", link)
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// sanitize keeps path fragments filesystem-safe.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, s)
}
