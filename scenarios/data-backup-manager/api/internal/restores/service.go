package restores

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/platformreceipts"
	"data-backup-manager/internal/sources"

	"github.com/vrooli/api-core/schedule"
)

// Service is the application surface the restores handler depends on.
//
// RestoreTarget and VerifyTarget are ASYNCHRONOUS: they validate synchronously
// (so a bad request still gets an immediate error), persist a non-terminal
// record, schedule the heavy kopia work on a background worker bound to the
// server-lifetime context, and return the record in its current state. The
// request RPC therefore returns in milliseconds — a client/proxy disconnect can
// no longer sever an in-flight restore (the regression that forced a 6h server
// WriteTimeout). Callers poll GetRestore for the terminal status.
type Service interface {
	// RestoreTarget schedules a restore of a snapshot to the caller-chosen
	// location and returns the record (non-terminal in production). The worker
	// restores the kopia snapshot then applies the source-kind restore step.
	RestoreTarget(ctx context.Context, targetID, destinationID, snapshotID, location string) (Restore, error)
	// VerifyTarget schedules a test-restore of a snapshot to a scratch directory,
	// a 100% byte-verify, and a checksum; it records last_verified_at on success
	// and ALWAYS cleans up the scratch dir. Returns the record (non-terminal in
	// production). A verify failure NEVER yields status=verified (OT-P0-006).
	VerifyTarget(ctx context.Context, targetID, destinationID, snapshotID string) (Restore, error)
	// GetRestore returns a single restore record by id.
	GetRestore(ctx context.Context, id string) (Restore, error)
	// ListRestores returns records newest-first, optionally filtered by target.
	ListRestores(ctx context.Context, targetID string, limit int) ([]Restore, error)
	// LastVerifiedByTarget returns the latest successful verify per target
	// (the proven-restorable rollup), optionally filtered to targetIDs.
	LastVerifiedByTarget(ctx context.Context, targetIDs []string) ([]VerifiedStatus, error)
	LastVerifiedByTargetDestination(ctx context.Context, targetIDs []string) ([]DestinationVerifiedStatus, error)
	// Reconcile closes any restore left non-terminal by a crash/restart as failed
	// (fail-not-resume, mirroring runs). Called once at startup.
	Reconcile(ctx context.Context) error
	// Shutdown drains the background executor, bounded by ctx.
	Shutdown(ctx context.Context) error
}

// Deps bundles the seams the restores service orchestrates.
type Deps struct {
	Repo         Repository
	Targets      TargetLookup
	Destinations DestinationLookup
	Engine       engine.KopiaEngine
	Sources      *sources.Registry
	Clock        schedule.Clock
	// ScratchRoot is the base directory scratch verify dirs are created under.
	// Empty uses the OS temp dir.
	ScratchRoot string
	// BaseContext is the server-lifetime context the background workers bind to.
	// Nil uses context.Background(). It must outlive requests — a client
	// disconnect must not cancel an in-flight restore.
	BaseContext context.Context
	// Workers is the number of background restore workers. < 1 clamps to 1.
	Workers int
	// Executor overrides the background executor. Nil wires the production
	// AsyncExecutor. Tests inject SyncExecutor for deterministic completion.
	Executor Executor
	// Logger receives background-worker diagnostics. Optional.
	Logger *log.Logger
	// RemoveAll is the cleanup seam for restore scratch directories. It is
	// injectable so cleanup failures remain testable evidence rather than an
	// ignored filesystem side effect.
	RemoveAll func(string) error
	// PlatformReceipts records successful scratch verifies as native-runtime
	// evidence. Nil preserves isolation for narrow tests and fixtures.
	PlatformReceipts platformreceipts.Sink
}

const defaultRestoreListLimit = 100

type service struct {
	deps     Deps
	executor Executor
}

// NewService constructs the production restore service and starts its background
// executor, bound to s.executeJob so scheduled jobs run the full
// restore/verify+persist lifecycle off the request path.
func NewService(d Deps) Service {
	if d.RemoveAll == nil {
		d.RemoveAll = removeScratchTree
	}
	s := &service{deps: d}
	exec := d.Executor
	if exec == nil {
		exec = NewAsyncExecutor(d.Workers)
	}
	s.executor = exec
	s.executor.Bind(d.BaseContext, s.executeJob)
	return s
}

// removeScratchTree normalizes owner permissions before removing a restored
// scratch tree. Kopia preserves source permissions, so read-only directories
// can otherwise make an otherwise verified drill fail during cleanup and
// leave restored data in the temp directory. This helper never follows
// symlinks and only operates on an exact directory created by DBM's scratch
// restore path.
func removeScratchTree(root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("scratch path is required")
	}
	base := filepath.Base(root)
	if !strings.HasPrefix(base, "dbm-restore-") && !strings.HasPrefix(base, "dbm-verify-") && !strings.HasPrefix(base, "dbm-materialized-") {
		return fmt.Errorf("scratch cleanup root %q is outside the DBM scratch naming contract", base)
	}
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("scratch cleanup root must be a real directory")
	}
	if err := os.Chmod(root, info.Mode().Perm()|0o700); err != nil {
		return fmt.Errorf("make scratch root removable: %w", err)
	}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		mode := entryInfo.Mode().Perm()
		if entry.IsDir() {
			mode |= 0o700
		} else {
			// Also clear read-only file attributes on platforms where those
			// attributes prevent unlinking a restored file.
			mode |= 0o600
		}
		if err := os.Chmod(path, mode); err != nil {
			return fmt.Errorf("make scratch entry removable %q: %w", path, err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("prepare scratch tree for removal: %w", err)
	}
	return os.RemoveAll(root)
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) logf(format string, args ...any) {
	if s.deps.Logger != nil {
		s.deps.Logger.Printf("restores: "+format, args...)
	}
}

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
		UpdatedAt:     now,
	}

	// Resolve synchronously so a bad target/destination is reported immediately
	// (recorded as a failed record) rather than failing in the background.
	target, err := s.deps.Targets.TargetForRestore(ctx, targetID)
	if err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("resolve target: %v", err))
	}
	dest, err := s.deps.Destinations.DestinationForRestore(ctx, destinationID)
	if err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("resolve destination: %v", err))
	}

	created, err := s.deps.Repo.CreateRestore(ctx, rec)
	if err != nil {
		return Restore{}, fmt.Errorf("create restore: %w", err)
	}
	s.executor.Submit(RestoreJob{Restore: created, Target: target, DestinationID: dest.ID, DestName: dest.Name})
	return s.deps.Repo.GetRestore(ctx, created.ID)
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
		UpdatedAt:     now,
	}

	dest, err := s.deps.Destinations.DestinationForRestore(ctx, destinationID)
	if err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("resolve destination: %v", err))
	}

	if s.deps.Targets == nil {
		return s.failRestore(ctx, rec, "target lookup is unavailable")
	}
	target, err := s.deps.Targets.TargetForRestore(ctx, targetID)
	if err != nil {
		return s.failRestore(ctx, rec, fmt.Sprintf("resolve target: %v", err))
	}
	created, err := s.deps.Repo.CreateRestore(ctx, rec)
	if err != nil {
		return Restore{}, fmt.Errorf("create restore: %w", err)
	}
	s.executor.Submit(RestoreJob{Restore: created, Target: target, DestinationID: dest.ID, DestName: dest.Name})
	return s.deps.Repo.GetRestore(ctx, created.ID)
}

// executeJob is the background worker body: it drives one requested restore or
// verify to its terminal state, persisting the transition and the result so a
// crashed job is reconcilable rather than stranded. It runs under the executor's
// server-lifetime context, so a client disconnect cannot cancel it, and it
// never returns an error — failures are persisted onto the record.
func (s *service) executeJob(ctx context.Context, job RestoreJob) {
	// Resolve again in the worker immediately before repository use. Requests
	// may sit in the queue while a removable volume is unplugged or remounted.
	dest, err := s.deps.Destinations.DestinationForRestore(ctx, job.DestinationID)
	if err != nil {
		s.finishFailed(ctx, job.Restore.ID, fmt.Sprintf("re-resolve destination before repository use: %v", err))
		return
	}
	if dest.Name != job.DestName {
		s.finishFailed(ctx, job.Restore.ID, "destination identity changed while restore was queued")
		return
	}
	switch job.Restore.Mode {
	case ModeVerify:
		s.runVerify(ctx, job)
	default:
		s.runRestore(ctx, job)
	}
}

func (s *service) runRestore(ctx context.Context, job RestoreJob) {
	id := job.Restore.ID
	if err := s.deps.Repo.UpdateRestoreStatus(ctx, id, EventStartRestore); err != nil {
		s.logf("restore %s -> restoring: %v", id, err)
		s.finishFailed(ctx, id, fmt.Sprintf("could not persist restoring transition: %v", err))
		return
	}

	scratchBase := s.scratchBase()
	if err := os.MkdirAll(scratchBase, 0o755); err != nil {
		s.finishFailed(ctx, id, fmt.Sprintf("scratch root: %v", err))
		return
	}
	artifactDir, err := os.MkdirTemp(scratchBase, "dbm-restore-"+sanitize(job.Restore.SnapshotID)+"-"+sanitize(job.Restore.TargetID)+"-")
	if err != nil {
		s.finishFailed(ctx, id, fmt.Sprintf("scratch dir: %v", err))
		return
	}
	workErr := func() error {
		if err := s.withResolvedDestination(ctx, job, func(name string) error {
			return s.deps.Engine.SnapshotRestore(ctx, name, job.Restore.SnapshotID, artifactDir)
		}); err != nil {
			return fmt.Errorf("snapshot restore: %w", err)
		}

		capturer, err := s.deps.Sources.Capturer(job.Target.Kind)
		if err != nil {
			return fmt.Errorf("source capturer: %w", err)
		}
		if err := capturer.Restore(ctx, sources.RestoreSpec{
			Locator:      job.Target.Locator,
			ArtifactPath: artifactDir,
			Target:       job.Restore.Location,
		}); err != nil {
			return fmt.Errorf("source restore: %w", err)
		}
		return nil
	}()
	cleanupErr := s.deps.RemoveAll(artifactDir)
	if workErr != nil {
		if cleanupErr != nil {
			workErr = fmt.Errorf("%w; scratch cleanup: %v", workErr, cleanupErr)
		}
		s.finishFailed(ctx, id, workErr.Error())
		return
	}
	if cleanupErr != nil {
		s.finishFailed(ctx, id, fmt.Sprintf("scratch cleanup: %v", cleanupErr))
		return
	}

	if err := s.deps.Repo.FinishRestore(ctx, id, EventRestored, "", time.Time{}, s.deps.Clock.Now().UTC(), ""); err != nil {
		s.logf("finish restore %s: %v", id, err)
	}
}

func (s *service) runVerify(ctx context.Context, job RestoreJob) {
	id := job.Restore.ID
	if err := s.deps.Repo.UpdateRestoreStatus(ctx, id, EventStartVerify); err != nil {
		s.logf("restore %s -> verifying: %v", id, err)
		s.finishFailed(ctx, id, fmt.Sprintf("could not persist verifying transition: %v", err))
		return
	}

	scratchBase := s.scratchBase()
	if err := os.MkdirAll(scratchBase, 0o755); err != nil {
		s.finishFailed(ctx, id, fmt.Sprintf("scratch root: %v", err))
		return
	}
	scratchDir, err := os.MkdirTemp(scratchBase, "dbm-verify-"+sanitize(job.Restore.SnapshotID)+"-"+sanitize(job.Restore.TargetID)+"-")
	if err != nil {
		s.finishFailed(ctx, id, fmt.Sprintf("scratch dir: %v", err))
		return
	}
	workErr, checksum := func() (error, string) {
		if err := s.withResolvedDestination(ctx, job, func(name string) error {
			if err := s.deps.Engine.SnapshotRestore(ctx, name, job.Restore.SnapshotID, scratchDir); err != nil {
				return fmt.Errorf("snapshot restore: %w", err)
			}
			// Full 100% byte-verify. CRITICAL INVARIANT (OT-P0-006): a verify
			// failure must NEVER set status=verified or last_verified_at.
			return s.deps.Engine.SnapshotVerify(ctx, name, job.Restore.SnapshotID, 100)
		}); err != nil {
			return fmt.Errorf("repository use: %w", err), ""
		}

		// Workspace verification exercises the final source adapter, not just
		// engine extraction. Its capture manifest covers bytes AND metadata.
		if job.Target.Kind == sources.KindWorkspace {
			capturer, err := s.deps.Sources.Capturer(job.Target.Kind)
			if err != nil {
				return err, ""
			}
			materialized, err := os.MkdirTemp(scratchBase, "dbm-materialized-")
			if err != nil {
				return err, ""
			}
			restoreErr := capturer.Restore(ctx, sources.RestoreSpec{ArtifactPath: scratchDir, Target: materialized})
			cleanupErr := s.deps.RemoveAll(materialized)
			if restoreErr != nil {
				return fmt.Errorf("workspace materialization: %w; cleanup: %v", restoreErr, cleanupErr), ""
			}
			if cleanupErr != nil {
				return fmt.Errorf("workspace cleanup: %w", cleanupErr), ""
			}
		}

		checksum, err := checksumDir(scratchDir)
		if err != nil {
			return fmt.Errorf("checksum: %w", err), ""
		}
		return nil, checksum
	}()
	cleanupErr := s.deps.RemoveAll(scratchDir)
	if workErr != nil {
		if cleanupErr != nil {
			workErr = fmt.Errorf("%w; scratch cleanup: %v", workErr, cleanupErr)
		}
		s.finishFailed(ctx, id, workErr.Error())
		return
	}
	if cleanupErr != nil {
		s.finishFailed(ctx, id, fmt.Sprintf("scratch cleanup: %v", cleanupErr))
		return
	}

	now := s.deps.Clock.Now().UTC()
	if err := s.deps.Repo.FinishRestore(ctx, id, EventVerified, checksum, now, now, ""); err != nil {
		s.logf("finish verify %s: %v", id, err)
	} else if s.deps.PlatformReceipts != nil {
		if err := s.deps.PlatformReceipts.Record(ctx, platformreceipts.Input{
			Capability: "scratch_restore", Outcome: "passed", Operation: job.Restore.ID,
			TargetID: job.Restore.TargetID, DestinationID: job.Restore.DestinationID,
			Proof: checksum, ObservedAt: now,
		}); err != nil {
			s.logf("verify %s platform receipt: %v", id, err)
		}
	}
}

// withResolvedDestination makes the repository locator current immediately
// before a Kopia read and holds DBM's destination lock through that operation.
func (s *service) withResolvedDestination(ctx context.Context, job RestoreJob, use func(string) error) error {
	return s.deps.Destinations.WithResolvedDestinationForRead(ctx, job.DestinationID, func(current DestinationForRestore) error {
		if current.ID != job.DestinationID || current.Name != job.DestName {
			return fmt.Errorf("destination identity changed before repository use")
		}
		return use(current.Name)
	})
}

// finishFailed persists a terminal failed state, leaving checksum and
// last_verified_at unset (the OT-P0-006 invariant for a failed verify).
func (s *service) finishFailed(ctx context.Context, id, errMsg string) {
	now := s.deps.Clock.Now().UTC()
	if err := s.deps.Repo.FinishRestore(ctx, id, EventFail, "", time.Time{}, now, errMsg); err != nil {
		s.logf("finish failed restore %s: %v", id, err)
		return
	}
	if s.deps.PlatformReceipts != nil {
		restore, err := s.deps.Repo.GetRestore(ctx, id)
		if err != nil {
			s.logf("failed restore %s platform receipt read: %v", id, err)
			return
		}
		if restore.Mode == ModeVerify {
			if err := s.deps.PlatformReceipts.Record(ctx, platformreceipts.Input{
				Capability: "scratch_restore", Outcome: "failed", Operation: restore.ID,
				TargetID: restore.TargetID, DestinationID: restore.DestinationID,
				Proof: errMsg, ObservedAt: now,
			}); err != nil {
				s.logf("failed verify %s platform receipt: %v", id, err)
			}
		}
	}
}

func (s *service) scratchBase() string {
	if s.deps.ScratchRoot != "" {
		return s.deps.ScratchRoot
	}
	return os.TempDir()
}

// Reconcile closes any restore left in a non-terminal state by a crash/restart
// or a client disconnect that killed an in-flight job. Policy is fail-not-
// resume (matching runs): each orphan is marked failed with a reconciliation
// reason — never silently resumed and never falsely "verified".
func (s *service) Reconcile(ctx context.Context) error {
	orphans, err := s.deps.Repo.ListNonTerminalRestores(ctx)
	if err != nil {
		return fmt.Errorf("reconcile: list non-terminal restores: %w", err)
	}
	const reason = "reconciled: process restarted while restore was in-flight"
	now := s.deps.Clock.Now().UTC()
	for _, r := range orphans {
		if err := s.deps.Repo.FinishRestore(ctx, r.ID, EventFail, "", time.Time{}, now, reason); err != nil {
			s.logf("reconcile restore %s: %v", r.ID, err)
		}
	}
	if len(orphans) > 0 {
		s.logf("reconciled %d orphaned non-terminal restore(s) to failed", len(orphans))
	}
	return nil
}

// Shutdown drains the background executor, bounded by ctx.
func (s *service) Shutdown(ctx context.Context) error {
	return s.executor.Shutdown(ctx)
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

func (s *service) LastVerifiedByTargetDestination(ctx context.Context, targetIDs []string) ([]DestinationVerifiedStatus, error) {
	return s.deps.Repo.LastVerifiedByTargetDestination(ctx, targetIDs)
}

// failRestore records a failed restore and persists it. If persistence fails,
// returns the persistence error (the caller still sees a failure).
func (s *service) failRestore(ctx context.Context, rec Restore, errMsg string) (Restore, error) {
	rec.Status = RestoreRequested
	if err := rec.Transition(EventFail); err != nil {
		return Restore{}, fmt.Errorf("transition new restore to failed: %w", err)
	}
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

// ChecksumDirectory returns the same deterministic content digest persisted by
// DBM's terminal verify lifecycle. Callers use it only on bounded private
// materializations whose extraction limits were checked before restore.
func ChecksumDirectory(dir string) (string, error) { return checksumDir(dir) }

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
