package audits

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"data-backup-manager/internal/clock"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/sources"
)

// Service is the application surface the audits handler depends on.
//
// RunSnapshotAudit is ASYNCHRONOUS (mirroring restores): it validates
// synchronously (so a bad request still gets an immediate error), persists a
// non-terminal record, schedules the heavy restore+capture+walk on a background
// worker bound to the server-lifetime context, and returns the record in its
// current state. The request RPC therefore returns in milliseconds — a client
// disconnect cannot sever an in-flight audit. Callers poll GetAudit for the
// terminal status.
type Service interface {
	// RunSnapshotAudit schedules a generic audit of a snapshot: restore to
	// scratch, capture live to scratch (read-only on live), walk both, and
	// compare by generic signals only. Returns the record (non-terminal in
	// production). Never modifies live data.
	RunSnapshotAudit(ctx context.Context, targetID, destinationID, snapshotID string, includeContentHash, includeSQLiteCheck bool) (Audit, error)
	// GetAudit returns a single audit record by id.
	GetAudit(ctx context.Context, id string) (Audit, error)
	// ListAudits returns records newest-first, optionally filtered by target.
	ListAudits(ctx context.Context, targetID string, limit int) ([]Audit, error)
	// Reconcile closes any audit left non-terminal by a crash/restart as failed
	// (fail-not-resume, mirroring restores). Called once at startup.
	Reconcile(ctx context.Context) error
	// Shutdown drains the background executor, bounded by ctx.
	Shutdown(ctx context.Context) error
}

// Deps bundles the seams the audit service orchestrates.
type Deps struct {
	Repo         Repository
	Targets      TargetLookup
	Destinations DestinationLookup
	Engine       engine.KopiaEngine
	Sources      *sources.Registry
	SQLite       SQLiteChecker
	Clock        clock.Clock
	// ScratchRoot is the base directory scratch restore/capture dirs are created
	// under. Empty uses the OS temp dir.
	ScratchRoot string
	// BaseContext is the server-lifetime context the background workers bind to.
	// Nil uses context.Background(). It must outlive requests.
	BaseContext context.Context
	// Workers is the number of background audit workers. < 1 clamps to 1.
	Workers int
	// Executor overrides the background executor. Nil wires the production
	// AsyncExecutor. Tests inject SyncExecutor for deterministic completion.
	Executor Executor
	// Logger receives background-worker diagnostics. Optional.
	Logger *log.Logger
}

const defaultAuditListLimit = 100

type service struct {
	deps     Deps
	executor Executor
}

// NewService constructs the production audit service and starts its background
// executor, bound to s.executeJob so scheduled jobs run the full
// restore+capture+walk+persist lifecycle off the request path.
func NewService(d Deps) Service {
	if d.SQLite == nil {
		d.SQLite = NewSQLiteChecker()
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

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) logf(format string, args ...any) {
	if s.deps.Logger != nil {
		s.deps.Logger.Printf("audits: "+format, args...)
	}
}

func (s *service) RunSnapshotAudit(ctx context.Context, targetID, destinationID, snapshotID string, includeContentHash, includeSQLiteCheck bool) (Audit, error) {
	targetID = strings.TrimSpace(targetID)
	destinationID = strings.TrimSpace(destinationID)
	snapshotID = strings.TrimSpace(snapshotID)
	if targetID == "" {
		return Audit{}, ErrInvalidAudit{Field: "target_id", Reason: "required"}
	}
	if destinationID == "" {
		return Audit{}, ErrInvalidAudit{Field: "destination_id", Reason: "required"}
	}
	if snapshotID == "" {
		return Audit{}, ErrInvalidAudit{Field: "snapshot_id", Reason: "required"}
	}

	now := s.deps.Clock.Now().UTC()
	rec := Audit{
		TargetID:           targetID,
		DestinationID:      destinationID,
		SnapshotID:         snapshotID,
		Status:             AuditRequested,
		IncludeContentHash: includeContentHash,
		IncludeSQLiteCheck: includeSQLiteCheck,
		RequestedAt:        now,
		UpdatedAt:          now,
	}

	// Resolve synchronously so a bad target/destination is reported immediately
	// (recorded as a failed record) rather than failing in the background.
	target, err := s.deps.Targets.TargetForAudit(ctx, targetID)
	if err != nil {
		return s.failSync(ctx, rec, fmt.Sprintf("resolve target: %v", err))
	}
	dest, err := s.deps.Destinations.DestinationForAudit(ctx, destinationID)
	if err != nil {
		return s.failSync(ctx, rec, fmt.Sprintf("resolve destination: %v", err))
	}

	created, err := s.deps.Repo.CreateAudit(ctx, rec)
	if err != nil {
		return Audit{}, fmt.Errorf("create audit: %w", err)
	}
	s.executor.Submit(AuditJob{Audit: created, Target: target, DestName: dest.Name})
	return s.deps.Repo.GetAudit(ctx, created.ID)
}

// executeJob is the background worker body: it drives one requested audit to its
// terminal state. It runs under the executor's server-lifetime context, never
// returns an error (failures are persisted onto the record), and ALWAYS cleans
// up its scratch directories.
func (s *service) executeJob(ctx context.Context, job AuditJob) {
	id := job.Audit.ID
	rec := job.Audit
	if err := s.deps.Repo.UpdateAuditStatus(ctx, id, AuditRunning); err != nil {
		s.logf("audit %s -> running: %v", id, err)
	}

	scratchBase := s.scratchBase()
	if err := os.MkdirAll(scratchBase, 0o755); err != nil {
		s.finishFailed(ctx, rec, fmt.Sprintf("scratch root: %v", err))
		return
	}
	restoreDir, err := os.MkdirTemp(scratchBase, "dbm-audit-restore-"+sanitize(job.Audit.SnapshotID)+"-")
	if err != nil {
		s.finishFailed(ctx, rec, fmt.Sprintf("scratch restore dir: %v", err))
		return
	}
	defer func() { _ = os.RemoveAll(restoreDir) }()
	captureDir, err := os.MkdirTemp(scratchBase, "dbm-audit-live-"+sanitize(job.Audit.TargetID)+"-")
	if err != nil {
		s.finishFailed(ctx, rec, fmt.Sprintf("scratch capture dir: %v", err))
		return
	}
	defer func() { _ = os.RemoveAll(captureDir) }()

	// 1. Restore the snapshot to scratch — this proves recoverability.
	if err := s.deps.Engine.SnapshotRestore(ctx, job.DestName, job.Audit.SnapshotID, restoreDir); err != nil {
		s.finishFailed(ctx, rec, fmt.Sprintf("snapshot restore: %v", err))
		return
	}
	rec.Restorable = true
	rec.SnapshotTime = s.snapshotTime(ctx, job.DestName, job.Audit.SnapshotID)

	opts := walkOptions{includeContentHash: rec.IncludeContentHash, detectSQLite: rec.IncludeSQLiteCheck}
	now := s.deps.Clock.Now().UTC()

	// 2. Walk the restored snapshot artifact.
	snapWalk, err := walkTree(restoreDir, opts, now)
	if err != nil {
		s.finishFailed(ctx, rec, fmt.Sprintf("walk snapshot: %v", err))
		return
	}
	snapInv := snapWalk.summary
	if rec.IncludeSQLiteCheck {
		snapInv.SQLite = s.checkSQLite(ctx, snapWalk.sqlite)
	}
	rec.Snapshot = &snapInv

	// 3. Capture the live target to scratch (read-only on live).
	capturer, err := s.deps.Sources.Capturer(job.Target.Kind)
	if err != nil {
		s.finishFailed(ctx, rec, fmt.Sprintf("source capturer: %v", err))
		return
	}
	artifact, err := capturer.Capture(ctx, sources.CaptureSpec{Locator: job.Target.Locator, StageDir: captureDir})
	if err != nil {
		s.finishFailed(ctx, rec, fmt.Sprintf("capture live: %v", err))
		return
	}

	// 4. Walk the freshly captured live artifact.
	liveWalk, err := walkTree(artifact.Path, opts, s.deps.Clock.Now().UTC())
	if err != nil {
		s.finishFailed(ctx, rec, fmt.Sprintf("walk live: %v", err))
		return
	}
	liveInv := liveWalk.summary
	if rec.IncludeSQLiteCheck {
		liveInv.SQLite = s.checkSQLite(ctx, liveWalk.sqlite)
	}
	rec.Live = &liveInv

	// 5. Compare by generic signals only and persist the completed audit.
	comparison := compareInventories(liveInv, snapInv, rec.SnapshotTime)
	rec.Comparison = &comparison
	rec.Status = AuditCompleted
	rec.FinishedAt = s.deps.Clock.Now().UTC()
	if err := s.deps.Repo.FinishAudit(ctx, rec); err != nil {
		s.logf("finish audit %s: %v", id, err)
	}
}

// checkSQLite runs the read-only SQLite checker over each discovered candidate.
func (s *service) checkSQLite(ctx context.Context, candidates []sqliteCandidate) []SqliteInventory {
	if len(candidates) == 0 {
		return nil
	}
	out := make([]SqliteInventory, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, s.deps.SQLite.Check(ctx, c.Abs, c.Rel))
	}
	return out
}

// snapshotTime resolves a snapshot's start time from the engine's snapshot list
// (best-effort; zero when unknown). It powers the drift interpretation.
func (s *service) snapshotTime(ctx context.Context, repo, snapshotID string) time.Time {
	snaps, err := s.deps.Engine.SnapshotList(ctx, repo, "")
	if err != nil {
		return time.Time{}
	}
	for _, sn := range snaps {
		if sn.ID == snapshotID {
			if t, err := time.Parse(time.RFC3339, sn.StartTime); err == nil {
				return t.UTC()
			}
			if t, err := time.Parse(time.RFC3339Nano, sn.StartTime); err == nil {
				return t.UTC()
			}
			return time.Time{}
		}
	}
	return time.Time{}
}

// finishFailed persists a terminal failed audit, keeping whatever partial
// evidence (restorable, snapshot inventory) was already gathered.
func (s *service) finishFailed(ctx context.Context, rec Audit, errMsg string) {
	rec.Status = AuditFailed
	rec.Error = errMsg
	rec.FinishedAt = s.deps.Clock.Now().UTC()
	if err := s.deps.Repo.FinishAudit(ctx, rec); err != nil {
		s.logf("finish failed audit %s: %v", rec.ID, err)
	}
}

// failSync records a synchronously-failed audit (bad target/destination) and
// persists it. If persistence fails, returns the persistence error.
func (s *service) failSync(ctx context.Context, rec Audit, errMsg string) (Audit, error) {
	rec.Status = AuditFailed
	rec.Error = errMsg
	rec.FinishedAt = s.deps.Clock.Now().UTC()
	saved, persistErr := s.deps.Repo.CreateAudit(ctx, rec)
	if persistErr != nil {
		return Audit{}, fmt.Errorf("persist failed audit: %w (original: %s)", persistErr, errMsg)
	}
	return saved, nil
}

func (s *service) scratchBase() string {
	if s.deps.ScratchRoot != "" {
		return s.deps.ScratchRoot
	}
	return os.TempDir()
}

// Reconcile closes any audit left non-terminal by a crash/restart, fail-not-
// resume (matching restores): each orphan is marked failed with a
// reconciliation reason — never silently resumed.
func (s *service) Reconcile(ctx context.Context) error {
	orphans, err := s.deps.Repo.ListNonTerminalAudits(ctx)
	if err != nil {
		return fmt.Errorf("reconcile: list non-terminal audits: %w", err)
	}
	const reason = "reconciled: process restarted while audit was in-flight"
	for _, a := range orphans {
		a.Status = AuditFailed
		a.Error = reason
		a.FinishedAt = s.deps.Clock.Now().UTC()
		if err := s.deps.Repo.FinishAudit(ctx, a); err != nil {
			s.logf("reconcile audit %s: %v", a.ID, err)
		}
	}
	if len(orphans) > 0 {
		s.logf("reconciled %d orphaned non-terminal audit(s) to failed", len(orphans))
	}
	return nil
}

// Shutdown drains the background executor, bounded by ctx.
func (s *service) Shutdown(ctx context.Context) error {
	return s.executor.Shutdown(ctx)
}

func (s *service) GetAudit(ctx context.Context, id string) (Audit, error) {
	return s.deps.Repo.GetAudit(ctx, id)
}

func (s *service) ListAudits(ctx context.Context, targetID string, limit int) ([]Audit, error) {
	if limit <= 0 {
		limit = defaultAuditListLimit
	}
	return s.deps.Repo.ListAudits(ctx, strings.TrimSpace(targetID), limit)
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
