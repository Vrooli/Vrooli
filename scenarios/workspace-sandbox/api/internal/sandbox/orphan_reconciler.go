package sandbox

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/types"
)

// OrphanReport summarizes the work done by a single orphan-reconciler
// pass. Returned to callers (startup logger, periodic loop) so the
// outcome is observable without trawling audit logs.
type OrphanReport struct {
	// FilesystemDirs is the total count of UUID-named dirs the driver
	// found under BaseDir at the start of the pass.
	FilesystemDirs int

	// OrphansCleaned is the number of orphan dirs the reconciler
	// successfully released (unmounted + removed).
	OrphansCleaned int

	// OrphansFailed is the number of orphan dirs that survived a
	// cleanup attempt. These usually have a process holding the
	// mount (a still-running fuse-overlayfs or a scanner inside the
	// merged dir); they will be retried on the next pass.
	OrphansFailed int

	// FailedIDs lists the IDs that failed cleanup, paired with the
	// last error message. Bounded at len()<=20 so a pathological run
	// doesn't bloat the report.
	FailedIDs []OrphanCleanupFailure

	// Duration is how long the pass took.
	Duration time.Duration
}

// OrphanCleanupFailure describes one cleanup that didn't complete.
type OrphanCleanupFailure struct {
	ID    uuid.UUID
	Error string
}

// ReconcileFilesystemOrphans walks the driver's BaseDir, cross-references
// each UUID-named dir against the repository, and releases any dir the
// repository does not know about (or which it has marked deleted).
//
// Why this exists (2026-04-28 incident):
//
// Before the agent-manager finalize() seam landed, ~280 fuse-overlayfs
// mounts accumulated under ~/.local/share/workspace-sandbox/ that the
// repository had no record of. agent-manager had Delete()d them at the
// API, but workspace-sandbox lost track of the sandboxes (process
// restarts, partial completion, or the API never registering the
// sandbox in the first place during the pre-resolveSandboxConfig era).
// The existing TTL GC in lifecycle.go is repo-driven: it cannot see
// dirs the repo doesn't know about.
//
// This reconciler closes that gap. It is the second-line defense, run
// once on startup (synchronously, so the boot log surfaces the orphan
// count) and on every Runner tick (so any future drift is
// caught within minutes).
//
// Safety properties:
//
//  1. Only acts on dirs whose UUID is NOT in the repo, OR is in the
//     repo with Status=Deleted. Active/Stopped/Approved/Rejected
//     sandboxes are NEVER touched here — they belong to the existing
//     repo-driven lifecycle.
//  2. Cleanup is best-effort and idempotent: a failed unmount on one
//     pass will be retried on the next.
//  3. Audit events are emitted for every orphan cleaned, so operators
//     have a permanent trail of system-initiated cleanups.
func (s *Service) ReconcileFilesystemOrphans(ctx context.Context) OrphanReport {
	report := OrphanReport{}

	if s == nil || s.driver == nil {
		return report
	}
	start := s.clock.Now()

	dirs, err := s.driver.ListSandboxDirs(ctx)
	if err != nil {
		log.Printf("orphan-reconciler: failed to list sandbox dirs: %v", err)
		return report
	}
	report.FilesystemDirs = len(dirs)

	for _, id := range dirs {
		if !s.isOrphan(ctx, id) {
			continue
		}
		if err := s.driver.CleanupOrphan(ctx, id); err != nil {
			report.OrphansFailed++
			if len(report.FailedIDs) < 20 {
				report.FailedIDs = append(report.FailedIDs, OrphanCleanupFailure{
					ID:    id,
					Error: err.Error(),
				})
			}
			s.logOrphanAuditEvent(ctx, id, "sandbox.orphan-cleanup-failed", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}
		report.OrphansCleaned++
		s.logOrphanAuditEvent(ctx, id, "sandbox.orphan-cleaned", map[string]interface{}{
			"reason": "filesystem dir not in repository",
		})
	}

	report.Duration = s.clock.Since(start)
	return report
}

// isOrphan returns true if the sandbox dir at id has no matching
// non-deleted record in the repository. A repo error other than
// "not found" returns false (defensive: don't delete a dir we
// can't confirm is orphaned — rather wait for the next pass when
// the repo is healthy again).
func (s *Service) isOrphan(ctx context.Context, id uuid.UUID) bool {
	if s.repo == nil {
		return false
	}
	sandbox, err := s.repo.Get(ctx, id)
	if err == nil {
		// Repo's contract: (nil, nil) means the row doesn't exist —
		// treat as orphan. Otherwise act only on Status=Deleted (the
		// API has finished bookkeeping but the FS still has the dir).
		if sandbox == nil {
			return true
		}
		return sandbox.Status == types.StatusDeleted
	}
	// Repo error path: only act on the typed NotFoundError. Anything
	// else means the repo is unhealthy → defer to the next pass.
	var notFound *types.NotFoundError
	return errors.As(err, &notFound)
}

// logOrphanAuditEvent records an audit event for an orphan we acted on.
// Routes through the audit.Emitter seam (rather than Service.logAuditEvent)
// because we don't have a *types.Sandbox to feed the snapshot helper —
// the sandbox isn't in the repo by definition.
func (s *Service) logOrphanAuditEvent(ctx context.Context, id uuid.UUID, eventType string, details map[string]interface{}) {
	if s.audit == nil {
		return
	}
	idCopy := id
	if details == nil {
		details = map[string]interface{}{}
	}
	if err := s.audit.Emit(ctx, audit.Event{
		EventType: eventType,
		SandboxID: &idCopy,
		Actor:     "system",
		ActorType: "system",
		Details:   details,
	}); err != nil {
		// Audit logging is best-effort; the operational outcome (orphan
		// cleaned) is more important than the audit trail.
		log.Printf("orphan-reconciler: audit log failed for %s: %v", id, err)
	}
}

// FormatOrphanReport renders an OrphanReport for log output. Kept as a
// free function so callers (main.go startup, lifecycle ticker) can use
// the same wording.
func FormatOrphanReport(r OrphanReport) string {
	if r.FilesystemDirs == 0 {
		return "orphan-reconciler: 0 sandbox dirs on disk"
	}
	return fmt.Sprintf(
		"orphan-reconciler: %d dirs on disk, %d orphans cleaned, %d failed (%v)",
		r.FilesystemDirs, r.OrphansCleaned, r.OrphansFailed, r.Duration.Round(time.Millisecond),
	)
}
