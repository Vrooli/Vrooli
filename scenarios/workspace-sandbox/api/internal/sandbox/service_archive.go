package sandbox

// service_archive.go: snapshot wiring for terminal status transitions.
//
// The single entry point is snapshotAndTransition. Approve/Reject/Delete
// call it BEFORE running their downstream side effects (audit, lifecycle,
// agent-manager notification). The function:
//
//   1. Generates the diff via Service.GetDiff (or skips when captured=false).
//   2. Writes per-file content blobs and the unified-diff blob to disk.
//   3. Opens a SQL transaction, inserts the archive row, updates the
//      sandbox status, commits.
//   4. On any error: rolls back the transaction and best-effort removes
//      the per-sandbox blob directory so disk debris cannot accumulate.
//
// The transition is atomic by construction: the sandbox row's terminal
// status and the archive row's existence flip together. A terminal-
// status sandbox without an archive row is impossible, and an archive
// row without a corresponding terminal sandbox is impossible (PK +
// foreign key). See docs/internal/ARCHIVE_DESIGN.md for the full
// contract.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"

	"workspace-sandbox/internal/blobstore"
	"workspace-sandbox/internal/types"
)

// snapshotAndTransition writes a diff archive (or a not-captured marker)
// and atomically flips the sandbox's status to terminalStatus.
//
// terminalStatus must be one of StatusApproved, StatusRejected, or
// StatusDeleted; any other value returns an error before touching state.
//
// captured controls whether to materialize content. When true, the
// caller asserts the sandbox's overlay paths are still mountable
// (CanGenerateDiff(sandbox.Status) returns nil). When false, the
// archive row is inserted with archive_state="not_captured", no blobs
// are written, and Files/UnifiedDiffSHA256/TotalBlobBytes are zeroed.
// Pass captured=false for paths that cannot generate a diff (Error
// sandboxes, missing overlay).
//
// mutator is called inside the transaction with the sandbox struct
// (status already set to terminalStatus); use it for status-specific
// stamps like ApprovedAt. mutator may be nil. mutator MUST NOT issue
// SQL — the surrounding transaction owns that.
//
// On success the sandbox row in the database is at terminalStatus; on
// failure no row was changed and the caller should propagate the error
// without running downstream side effects.
//
// When the archive seam is not configured (s.archiveRepo or s.blobs
// nil), snapshotAndTransition only updates the sandbox status and
// returns nil — the test paths that don't wire the archive still get a
// working terminal transition.
func (s *Service) snapshotAndTransition(
	ctx context.Context,
	sandbox *types.Sandbox,
	terminalStatus types.Status,
	captured bool,
	mutator func(*types.Sandbox),
) error {
	if sandbox == nil {
		return errors.New("sandbox.snapshotAndTransition: sandbox is nil")
	}
	switch terminalStatus {
	case types.StatusApproved, types.StatusRejected, types.StatusDeleted:
		// ok
	default:
		return fmt.Errorf("sandbox.snapshotAndTransition: invalid terminal status %q", terminalStatus)
	}

	// Tests that don't wire the archive seam fall back to a plain status
	// flip. Production always wires both via WithArchive in main.go.
	if s.archiveRepo == nil || s.blobs == nil {
		sandbox.Status = terminalStatus
		if mutator != nil {
			mutator(sandbox)
		}
		if err := s.repo.Update(ctx, sandbox); err != nil {
			return fmt.Errorf("update sandbox to %s: %w", terminalStatus, err)
		}
		return nil
	}

	archive := &types.DiffArchive{
		SandboxID:         sandbox.ID,
		ArchiveState:      types.ArchiveStateNotCaptured,
		SandboxStatus:     terminalStatus,
		ProjectRoot:       sandbox.ProjectRoot,
		Owner:             sandbox.Owner,
		AgentManagerRunID: metadataString(sandbox.Metadata, metadataAgentManagerRunID),
	}

	// Track whether blobs were written so we can clean up on rollback.
	// Cleanup is best-effort: a leaked blob directory is annoying but
	// not a correctness failure (retention will eventually evict it).
	blobsWritten := false

	if captured {
		written, err := s.captureBlobs(ctx, sandbox, archive)
		if err != nil {
			// Defensive cleanup: captureBlobs may have written some
			// blobs before failing.
			if written {
				_ = s.blobs.DeleteSandbox(ctx, uuidText(sandbox.ID))
			}
			return fmt.Errorf("capture blobs: %w", err)
		}
		blobsWritten = written
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		if blobsWritten {
			_ = s.blobs.DeleteSandbox(ctx, uuidText(sandbox.ID))
		}
		return fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
			if blobsWritten {
				_ = s.blobs.DeleteSandbox(ctx, uuidText(sandbox.ID))
			}
		}
	}()

	if err := s.archiveRepo.Insert(ctx, tx.Tx(), archive); err != nil {
		return fmt.Errorf("insert archive: %w", err)
	}

	sandbox.Status = terminalStatus
	if mutator != nil {
		mutator(sandbox)
	}
	if err := tx.Update(ctx, sandbox); err != nil {
		return fmt.Errorf("update sandbox to %s: %w", terminalStatus, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	committed = true
	return nil
}

// captureBlobs reads each file's content from the upper/lower dir,
// writes content blobs to the blobstore, and populates archive.Files
// + archive.UnifiedDiffSHA256 + archive.TotalBlobBytes + archive.Stats.
//
// Returns blobsWritten=true if any Put succeeded (so the caller can
// schedule cleanup on rollback). Returns blobsWritten=false only when
// no Put was attempted — i.e. zero changes.
func (s *Service) captureBlobs(ctx context.Context, sandbox *types.Sandbox, archive *types.DiffArchive) (blobsWritten bool, _ error) {
	diffResult, err := s.GetDiff(ctx, sandbox.ID)
	if err != nil {
		return false, fmt.Errorf("get diff: %w", err)
	}

	archive.ArchiveState = types.ArchiveStateComplete
	archive.Stats = diffResult.Stats

	// Best-effort sandbox-id text for blobstore. uuidText guarantees
	// canonical form; the blobstore validator rejects anything else.
	sandboxIDText := uuidText(sandbox.ID)

	// Write the unified-diff blob. Always non-empty when captured=true
	// even if zero changes (we still want a row that says "snapshot
	// taken, no changes" — the empty diff is meaningful content).
	udResult, err := s.blobs.Put(ctx, sandboxIDText, []byte(diffResult.UnifiedDiff))
	if err != nil {
		return false, fmt.Errorf("put unified diff: %w", err)
	}
	archive.UnifiedDiffSHA256 = udResult.SHA256Hex
	archive.TotalBlobBytes += udResult.SizeOnDisk
	blobsWritten = true

	archive.Files = make([]types.ArchivedFileEntry, 0, len(diffResult.Files))
	for _, change := range diffResult.Files {
		entry := types.ArchivedFileEntry{
			Path:           change.FilePath,
			ChangeType:     change.ChangeType,
			Size:           change.FileSize,
			FileMode:       change.FileMode,
			ApprovalStatus: change.ApprovalStatus,
		}

		content, readErr := readFileForArchive(sandbox, change)
		if readErr != nil {
			// Read failure is fatal — we cannot produce a faithful
			// archive with missing per-file content. The transaction
			// will not commit; downstream cleanup runs.
			return blobsWritten, fmt.Errorf("read %s: %w", change.FilePath, readErr)
		}
		// content == nil for unreadable files (directories, sockets,
		// non-existent on the appropriate side). Record the entry with
		// no blob reference so the archive index stays complete.
		if content == nil {
			archive.Files = append(archive.Files, entry)
			continue
		}

		put, err := s.blobs.Put(ctx, sandboxIDText, content)
		if err != nil {
			return blobsWritten, fmt.Errorf("put %s: %w", change.FilePath, err)
		}
		entry.BlobSHA256 = put.SHA256Hex
		archive.TotalBlobBytes += put.SizeOnDisk
		archive.Files = append(archive.Files, entry)
	}

	return blobsWritten, nil
}

// readFileForArchive reads a file's content from upper (for added/
// modified) or lower (for deleted). Returns (nil, nil) for files that
// have no usable content (directory, special file, missing on the
// expected side); the archive entry is still recorded but with no
// BlobSHA256.
//
// Unlike diff.GetFileContent, this does NOT skip binary files: the
// archive should preserve full content even for binaries so future
// callers (e.g. asset diff viewers) have the raw bytes available. The
// display layer is free to skip rendering binary content; the storage
// layer should not silently drop it.
func readFileForArchive(sandbox *types.Sandbox, change *types.FileChange) ([]byte, error) {
	var targetPath string
	switch change.ChangeType {
	case types.ChangeTypeDeleted:
		targetPath = filepath.Join(sandbox.LowerDir, change.FilePath)
	default:
		targetPath = filepath.Join(sandbox.UpperDir, change.FilePath)
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, nil
	}
	if !info.Mode().IsRegular() {
		// Sockets, devices, FIFOs — no content to capture.
		return nil, nil
	}

	content, err := os.ReadFile(targetPath) // #nosec G304 -- joined under sandbox-owned roots
	if err != nil {
		return nil, err
	}
	return content, nil
}

// GetArchive returns the durable diff archive for sandboxID, or
// (nil, nil) when none exists.
//
// The returned *types.DiffResult carries ArchiveState set from the
// archive row so callers can render an explicit "no diff captured"
// state for not_captured archives. Per-file blob content is NOT eagerly
// loaded; callers fetch each file's content via FetchArchiveFile (Phase 3).
//
// Returns nil DiffResult and nil error when:
//   - sandboxID has no archive row (sandbox never reached a terminal
//     status), or
//   - the archive seam is not configured (test path).
func (s *Service) GetArchive(ctx context.Context, sandboxID uuid.UUID) (*types.DiffResult, error) {
	if s.archiveRepo == nil {
		return nil, nil
	}
	archive, err := s.archiveRepo.Get(ctx, sandboxID)
	if err != nil {
		return nil, fmt.Errorf("get archive: %w", err)
	}
	if archive == nil {
		return nil, nil
	}

	files := make([]*types.FileChange, 0, len(archive.Files))
	for _, e := range archive.Files {
		files = append(files, &types.FileChange{
			SandboxID:      sandboxID,
			FilePath:       e.Path,
			ChangeType:     e.ChangeType,
			FileSize:       e.Size,
			FileMode:       e.FileMode,
			ApprovalStatus: e.ApprovalStatus,
		})
	}

	unified := ""
	if archive.ArchiveState == types.ArchiveStateComplete && archive.UnifiedDiffSHA256 != "" && s.blobs != nil {
		raw, getErr := s.blobs.Get(ctx, uuidText(sandboxID), archive.UnifiedDiffSHA256)
		if getErr != nil && !errors.Is(getErr, blobstore.ErrNotFound) {
			return nil, fmt.Errorf("read unified diff blob: %w", getErr)
		}
		if raw != nil {
			unified = string(raw)
		}
	}

	return &types.DiffResult{
		SandboxID:    sandboxID,
		Files:        files,
		UnifiedDiff:  unified,
		Generated:    archive.SnapshotAt,
		Stats:        archive.Stats,
		ArchiveState: archive.ArchiveState,
	}, nil
}

// FetchArchiveFile returns the raw content of one file in an archive,
// addressed by its path within the archive's Files index. Returns
// (nil, blobstore.ErrNotFound) when no entry matches path or the entry
// has no blob content (directory, binary that wasn't captured, etc.).
func (s *Service) FetchArchiveFile(ctx context.Context, sandboxID uuid.UUID, path string) ([]byte, error) {
	if s.archiveRepo == nil || s.blobs == nil {
		return nil, blobstore.ErrNotFound
	}
	archive, err := s.archiveRepo.Get(ctx, sandboxID)
	if err != nil {
		return nil, fmt.Errorf("get archive: %w", err)
	}
	if archive == nil {
		return nil, blobstore.ErrNotFound
	}
	for _, e := range archive.Files {
		if e.Path == path {
			if e.BlobSHA256 == "" {
				return nil, blobstore.ErrNotFound
			}
			return s.blobs.Get(ctx, uuidText(sandboxID), e.BlobSHA256)
		}
	}
	return nil, blobstore.ErrNotFound
}

// ListHistory returns archive metadata rows matching filter. Used by
// the History tab in the workspace-sandbox UI and by Phase 3's
// /sandboxes/history endpoint.
func (s *Service) ListHistory(ctx context.Context, filter types.ArchiveListFilter) ([]*types.DiffArchive, int, error) {
	if s.archiveRepo == nil {
		return nil, 0, nil
	}
	return s.archiveRepo.List(ctx, filter)
}

// RetentionPolicy is the snapshot of retention levers a single
// reconciler pass enforces. Decoupled from config.RetentionConfig so
// the sandbox package never imports config (test-only convenience
// types are kept package-local).
//
// Zero-valued levers disable that lever. The combined eviction rule is
// "any archive matching ANY active lever's eviction predicate is
// evicted" — the levers compose, they don't override each other.
type RetentionPolicy struct {
	// MaxArchiveAgeDays evicts archives older than this many days.
	// 0 disables age-based eviction.
	MaxArchiveAgeDays int

	// MaxArchiveSizeBytes is the global byte budget across all
	// archive blobs. When the sum exceeds this, evict oldest-first
	// until the running total falls below the budget. 0 disables.
	MaxArchiveSizeBytes int64

	// MaxArchivesPerProject caps the number of archives per
	// project_root. Excess archives within a project are evicted
	// oldest-first. 0 disables the cap.
	MaxArchivesPerProject int
}

// ArchiveRetentionReport summarizes one reconciler pass.
type ArchiveRetentionReport struct {
	// Scanned is the total number of archive rows examined.
	Scanned int

	// EvictedAge is the number of archives evicted by the age lever.
	EvictedAge int

	// EvictedSize is the number of archives evicted by the size budget.
	EvictedSize int

	// EvictedPerProject is the number of archives evicted by the
	// per-project cap.
	EvictedPerProject int

	// BlobFailures is the number of archives whose blob deletion
	// failed; their rows were left in place to retry on the next
	// pass. Counted per failing archive (not per blob).
	BlobFailures int

	// LastError is the most recent non-fatal error encountered.
	// The pass continues past per-archive failures; LastError is the
	// signal for the operator to investigate.
	LastError string

	// Duration is how long the pass took.
	Duration time.Duration
}

// TotalEvicted is the sum of the three eviction counters. An archive
// can be selected by more than one lever in the same pass; we count
// the lever that triggered first (age → per-project → size), so a
// double-counted eviction is impossible by construction.
func (r ArchiveRetentionReport) TotalEvicted() int {
	return r.EvictedAge + r.EvictedSize + r.EvictedPerProject
}

// ReconcileArchiveRetention enforces the retention levers exactly once.
// The pass is fully idempotent: running it twice with no new archives
// in between evicts the same set the first time and an empty set the
// second.
//
// Eviction order within a single pass:
//
//  1. Age-based: archives older than now - MaxArchiveAgeDays days.
//  2. Per-project cap: for each project, keep the N newest, evict the
//     rest oldest-first.
//  3. Global size budget: while the sum of remaining archives exceeds
//     MaxArchiveSizeBytes, evict oldest-first.
//
// For each evicted archive the blobstore directory is removed first,
// THEN the archive row. If the blob removal fails, the row is left in
// place and BlobFailures is incremented; the next pass will retry. This
// prevents the dangerous shape of "row says archive is gone, but the
// disk still has the blobs" (silent disk leak).
//
// When the archive seam is not configured (s.archiveRepo or s.blobs
// nil) the function returns an empty report — same shape as the other
// reconcilers' tests-only fallbacks.
func (s *Service) ReconcileArchiveRetention(ctx context.Context, policy RetentionPolicy) ArchiveRetentionReport {
	start := s.clock.Now()
	report := ArchiveRetentionReport{}
	if s.archiveRepo == nil || s.blobs == nil {
		report.Duration = schedule.Since(start)
		return report
	}

	all, err := s.archiveRepo.AllOrdered(ctx)
	if err != nil {
		report.LastError = fmt.Sprintf("list archives: %v", err)
		report.Duration = schedule.Since(start)
		return report
	}
	report.Scanned = len(all)
	if len(all) == 0 {
		report.Duration = schedule.Since(start)
		return report
	}

	// evicted maps sandbox_id → which lever fired first. The lever
	// is recorded in the per-lever counters (no double counting).
	evicted := make(map[uuid.UUID]struct{}, len(all))

	// 1. Age-based eviction.
	if policy.MaxArchiveAgeDays > 0 {
		cutoff := s.clock.Now().Add(-time.Duration(policy.MaxArchiveAgeDays) * 24 * time.Hour)
		for _, a := range all {
			if a.SnapshotAt.Before(cutoff) {
				if s.evictArchive(ctx, a, &report) {
					evicted[a.SandboxID] = struct{}{}
					report.EvictedAge++
				}
			}
		}
	}

	// 2. Per-project cap eviction.
	if policy.MaxArchivesPerProject > 0 {
		// Group survivors by project_root, preserving oldest-first
		// order within each group (AllOrdered guarantees that).
		byProject := make(map[string][]*types.DiffArchive)
		for _, a := range all {
			if _, gone := evicted[a.SandboxID]; gone {
				continue
			}
			byProject[a.ProjectRoot] = append(byProject[a.ProjectRoot], a)
		}
		for _, group := range byProject {
			if len(group) <= policy.MaxArchivesPerProject {
				continue
			}
			// Evict the oldest (group is already oldest-first).
			toEvict := len(group) - policy.MaxArchivesPerProject
			for i := 0; i < toEvict; i++ {
				a := group[i]
				if s.evictArchive(ctx, a, &report) {
					evicted[a.SandboxID] = struct{}{}
					report.EvictedPerProject++
				}
			}
		}
	}

	// 3. Global size-budget eviction. Re-sum remaining archives;
	// evict oldest-first until the running total falls under budget.
	if policy.MaxArchiveSizeBytes > 0 {
		var remaining []*types.DiffArchive
		var total int64
		for _, a := range all {
			if _, gone := evicted[a.SandboxID]; gone {
				continue
			}
			remaining = append(remaining, a)
			total += a.TotalBlobBytes
		}
		// remaining is already oldest-first. Walk forward, evicting
		// until total <= budget OR we've evicted everything.
		for _, a := range remaining {
			if total <= policy.MaxArchiveSizeBytes {
				break
			}
			if s.evictArchive(ctx, a, &report) {
				evicted[a.SandboxID] = struct{}{}
				report.EvictedSize++
				total -= a.TotalBlobBytes
			} else {
				// Blob delete failed — counted in BlobFailures by
				// evictArchive. Move on; we cannot count its bytes
				// as evicted because the row stays.
				continue
			}
		}
	}

	report.Duration = schedule.Since(start)
	return report
}

// evictArchive removes the blobs first, then the archive row. Returns
// true on success, false on any failure (blob or row); on failure the
// report is annotated and the caller treats the archive as not-evicted.
func (s *Service) evictArchive(ctx context.Context, a *types.DiffArchive, report *ArchiveRetentionReport) bool {
	if a == nil {
		return false
	}
	// Blobs first: a row deletion before blob removal would leak
	// disk space invisibly. Order matters more than performance.
	if err := s.blobs.DeleteSandbox(ctx, uuidText(a.SandboxID)); err != nil {
		report.BlobFailures++
		report.LastError = fmt.Sprintf("delete blobs %s: %v", a.SandboxID, err)
		return false
	}
	if err := s.archiveRepo.Delete(ctx, a.SandboxID); err != nil {
		report.LastError = fmt.Sprintf("delete archive row %s: %v", a.SandboxID, err)
		// The blobs are gone and the row is still present — next
		// pass will retry the row delete. The blob delete is
		// idempotent so re-running it is harmless.
		return false
	}
	return true
}

// uuidText is the canonical lowercase 8-4-4-4-12 form of a UUID, the
// shape the blobstore and archive repository both validate against.
// Kept package-local because every caller already has a uuid.UUID; the
// repository's parse/format helpers stay private to repository.
func uuidText(id uuid.UUID) string { return id.String() }
