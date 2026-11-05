package retention

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/vrooli/browser-automation-studio/database"
)

// Sweep selects eligible terminal executions and, when opts.Apply is true,
// deletes their artifact directories and DB rows. A dry-run (Apply=false)
// performs no filesystem or database mutation.
func (s *Service) Sweep(ctx context.Context, opts Options) (*Report, error) {
	if s == nil {
		return nil, errors.New("retention service not configured")
	}
	if s.recordingsRoot == "" {
		return nil, ErrRecordingsRootNotConfigured
	}
	status := strings.ToLower(strings.TrimSpace(opts.Status))
	if status != "" && !database.IsTerminalStatus(status) {
		return nil, fmt.Errorf("status filter %q is not a terminal status", status)
	}

	candidates, err := s.gatherCandidates(ctx, opts, status)
	if err != nil {
		return nil, err
	}

	// Deterministic order: newest first.
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].StartedAt.After(candidates[j].StartedAt)
	})

	if opts.MaxItems > 0 && len(candidates) > opts.MaxItems {
		// candidates are newest-first; retain the oldest slice so a bounded tick
		// makes steady progress through expired evidence. Protection is resolved
		// separately against the full per-workflow history below.
		candidates = candidates[len(candidates)-opts.MaxItems:]
	}
	protected, err := s.computeProtected(ctx, candidates, opts, status)
	if err != nil {
		return nil, err
	}
	selectedBySize := s.selectByMaxBytes(candidates, protected, opts)

	cleanRoot := filepath.Clean(s.recordingsRoot)
	var cutoff time.Time
	if opts.MaxAgeDays > 0 {
		cutoff = s.now().Add(-time.Duration(opts.MaxAgeDays) * 24 * time.Hour)
	}
	if opts.MaxAgeSeconds > 0 {
		cutoff = s.now().Add(-time.Duration(opts.MaxAgeSeconds) * time.Second)
	}

	report := &Report{DryRun: !opts.Apply, RemovedByStatus: map[string]int{}}

	for _, exec := range candidates {
		item := Item{
			ExecutionID: exec.ID,
			Status:      exec.Status,
			WorkflowID:  exec.WorkflowID,
			StartedAt:   exec.StartedAt,
			CompletedAt: exec.CompletedAt,
			ResultPath:  exec.ResultPath,
		}

		// Defensive: never touch non-terminal executions.
		if !database.IsTerminalStatus(exec.Status) {
			report.skip(item, ReasonNonTerminal)
			continue
		}
		if protected[exec.ID] {
			report.skip(item, ReasonKeepLatest)
			continue
		}
		if opts.MaxBytes > 0 && !selectedBySize[exec.ID] {
			report.skip(item, ReasonMaxBytes)
			continue
		}
		if !cutoff.IsZero() && effectiveTime(exec).After(cutoff) {
			report.skip(item, ReasonTooNew)
			continue
		}

		artifactDir, ok := resolveArtifactDir(cleanRoot, exec)
		if !ok {
			report.skip(item, ReasonUnsafePath)
			continue
		}
		item.ArtifactDir = artifactDir

		size, exists, sizeErr := int64(0), false, error(nil)
		if estimated, ok := opts.EstimatedBytes[exec.ID]; ok && estimated >= 0 {
			size = estimated
			exists = true
			if checker, ok := s.fs.(pathExistence); ok {
				exists, sizeErr = checker.Exists(artifactDir)
			}
		} else {
			size, exists, sizeErr = s.fs.DirSize(artifactDir)
		}
		if sizeErr != nil {
			if s.log != nil {
				s.log.WithError(sizeErr).WithField("artifact_dir", artifactDir).Warn("retention: failed to size artifact directory")
			}
			// Treat a sizing error as zero bytes; deletion below still attempts cleanup.
			size = 0
		}
		item.EstimatedBytes = size

		if !opts.Apply {
			if !exists {
				item.Reason = ReasonMissingDir
			}
			report.remove(item)
			continue
		}

		// Apply mode: delete files first, then the DB row.
		if exists {
			var deleteErr error
			if deleter, ok := s.fs.(containedDeleter); ok {
				deleteErr = deleter.DeleteContained(ctx, cleanRoot, artifactDir)
			} else {
				deleteErr = s.fs.RemoveAll(artifactDir)
			}
			if deleteErr != nil {
				if s.log != nil {
					s.log.WithError(deleteErr).WithField("artifact_dir", artifactDir).Error("retention: failed to remove artifact directory")
				}
				report.errored(item, fmt.Sprintf("%s: %v", ReasonDeleteDirFailed, deleteErr))
				continue
			}
		} else {
			item.Reason = ReasonMissingDir
		}

		if err := s.store.DeleteExecution(ctx, exec.ID); err != nil {
			if s.log != nil {
				s.log.WithError(err).WithField("execution_id", exec.ID).Error("retention: failed to delete execution row")
			}
			report.errored(item, fmt.Sprintf("%s: %v", ReasonDeleteRowFailed, err))
			continue
		}

		report.remove(item)
	}

	return report, nil
}

func (s *Service) selectByMaxBytes(candidates []*database.ExecutionIndex, protected map[uuid.UUID]bool, opts Options) map[uuid.UUID]bool {
	selected := make(map[uuid.UUID]bool)
	if opts.MaxBytes <= 0 {
		for _, candidate := range candidates {
			selected[candidate.ID] = true
		}
		return selected
	}
	var used int64
	cleanRoot := filepath.Clean(s.recordingsRoot)
	cutoff := time.Time{}
	if opts.MaxAgeDays > 0 {
		cutoff = s.now().Add(-time.Duration(opts.MaxAgeDays) * 24 * time.Hour)
	}
	if opts.MaxAgeSeconds > 0 {
		cutoff = s.now().Add(-time.Duration(opts.MaxAgeSeconds) * time.Second)
	}
	// Candidates are newest-first. Select from the oldest end so a batch cap
	// preserves the newest expired evidence when the cap is smaller than the
	// eligible set.
	for i := len(candidates) - 1; i >= 0; i-- {
		exec := candidates[i]
		if protected[exec.ID] || !database.IsTerminalStatus(exec.Status) || (!cutoff.IsZero() && effectiveTime(exec).After(cutoff)) {
			continue
		}
		artifactDir, ok := resolveArtifactDir(cleanRoot, exec)
		if !ok {
			continue
		}
		size, _, err := s.fs.DirSize(artifactDir)
		if err != nil || size <= 0 || used+size > opts.MaxBytes {
			continue
		}
		selected[exec.ID] = true
		used += size
	}
	return selected
}

func (s *Service) gatherCandidates(ctx context.Context, opts Options, status string) ([]*database.ExecutionIndex, error) {
	targetStatuses := []string{database.ExecutionStatusCompleted, database.ExecutionStatusFailed, database.ExecutionStatusCancelled}
	if status != "" {
		targetStatuses = []string{status}
	}
	// Recovery applies carry an exact preview ID set. Fetch only those rows;
	// listing every terminal execution would turn each bounded owner batch into
	// a full-index scan and can saturate BAS on large evidence stores.
	if len(opts.ExecutionIDs) > 0 {
		out := make([]*database.ExecutionIndex, 0, len(opts.ExecutionIDs))
		for _, id := range opts.ExecutionIDs {
			entry, err := s.store.GetExecution(ctx, id)
			if err != nil {
				if errors.Is(err, database.ErrNotFound) {
					continue
				}
				return nil, fmt.Errorf("get execution %q: %w", id, err)
			}
			if entry == nil || !database.IsTerminalStatus(entry.Status) {
				continue
			}
			if status != "" && entry.Status != status {
				continue
			}
			out = append(out, entry)
		}
		return out, nil
	}

	var out []*database.ExecutionIndex
	for _, st := range targetStatuses {
		limit := 0
		// Preserve the unbounded filtered sweep and the existing per-status
		// preview budget. Both use the canonical repository query.
		if opts.WorkflowID == nil && opts.ProjectID == nil && opts.MaxItems > 0 {
			limit = opts.MaxItems
		}
		list, _, err := s.store.ListExecutions(ctx, database.ExecutionQuery{
			WorkflowID: opts.WorkflowID, ProjectID: opts.ProjectID, Status: st,
			Limit: limit, OldestFirst: limit > 0,
		})
		if err != nil {
			return nil, fmt.Errorf("list executions by status %q: %w", st, err)
		}
		out = append(out, list...)
	}
	return out, nil
}

func (s *Service) computeProtected(ctx context.Context, candidates []*database.ExecutionIndex, opts Options, status string) (map[uuid.UUID]bool, error) {
	keepLatest := opts.KeepLatest
	protected := map[uuid.UUID]bool{}
	if keepLatest <= 0 {
		return protected, nil
	}

	// A bounded/preview candidate batch is only a removal window, not the
	// population from which keep_latest is defined. Query the newest terminal
	// rows per workflow separately so an old row cannot protect itself merely
	// because newer rows were omitted from the batch.
	if opts.MaxItems > 0 || len(opts.ExecutionIDs) > 0 {
		workflows := make(map[uuid.UUID]struct{}, len(candidates))
		for _, candidate := range candidates {
			if candidate != nil {
				workflows[candidate.WorkflowID] = struct{}{}
			}
		}
		statuses := []string{database.ExecutionStatusCompleted, database.ExecutionStatusFailed, database.ExecutionStatusCancelled}
		if status != "" {
			statuses = []string{status}
		}
		workflowIDs := make([]uuid.UUID, 0, len(workflows))
		for workflowID := range workflows {
			workflowIDs = append(workflowIDs, workflowID)
		}
		var latest []*database.ExecutionIndex
		if reader, ok := s.store.(latestTerminalExecutionsPerWorkflow); ok {
			rows, err := reader.ListLatestTerminalExecutionsPerWorkflow(ctx, workflowIDs, opts.ProjectID, statuses, keepLatest)
			if err != nil {
				return nil, err
			}
			latest = rows
		} else {
			latest = make([]*database.ExecutionIndex, 0, len(workflows)*keepLatest)
			for _, workflowID := range workflowIDs {
				for _, candidateStatus := range statuses {
					rows, _, err := s.store.ListExecutions(ctx, database.ExecutionQuery{
						WorkflowID: &workflowID,
						ProjectID:  opts.ProjectID,
						Status:     candidateStatus,
						Limit:      keepLatest,
					})
					if err != nil {
						return nil, fmt.Errorf("list latest executions for workflow %s status %q: %w", workflowID, candidateStatus, err)
					}
					latest = append(latest, rows...)
				}
			}
		}
		return computeProtectedFromCandidates(latest, keepLatest), nil
	}
	return computeProtectedFromCandidates(candidates, keepLatest), nil
}

func computeProtectedFromCandidates(candidates []*database.ExecutionIndex, keepLatest int) map[uuid.UUID]bool {
	protected := map[uuid.UUID]bool{}
	byWorkflow := map[uuid.UUID][]*database.ExecutionIndex{}
	for _, e := range candidates {
		if e == nil {
			continue
		}
		byWorkflow[e.WorkflowID] = append(byWorkflow[e.WorkflowID], e)
	}
	for _, list := range byWorkflow {
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].StartedAt.After(list[j].StartedAt)
		})
		for i := 0; i < len(list) && i < keepLatest; i++ {
			protected[list[i].ID] = true
		}
	}
	return protected
}

func effectiveTime(exec *database.ExecutionIndex) time.Time {
	if exec.CompletedAt != nil && !exec.CompletedAt.IsZero() {
		return *exec.CompletedAt
	}
	return exec.StartedAt
}

// resolveArtifactDir returns the artifact directory for an execution and whether
// it is safe to operate on (strictly under cleanRoot, not cleanRoot itself).
// The canonical layout is <recordingsRoot>/<execution-id>; a recorded result
// path that resolves elsewhere is rejected.
func resolveArtifactDir(cleanRoot string, exec *database.ExecutionIndex) (string, bool) {
	dir := filepath.Join(cleanRoot, exec.ID.String())
	if rp := strings.TrimSpace(exec.ResultPath); rp != "" {
		derived := filepath.Dir(filepath.Clean(rp))
		if !underRoot(cleanRoot, derived) {
			return "", false
		}
		dir = derived
	}
	if !underRoot(cleanRoot, dir) {
		return "", false
	}
	return dir, true
}

// underRoot reports whether target is strictly contained within root (and is not
// root itself). It rejects path traversal and absolute escapes.
func underRoot(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." || rel == ".." {
		return false
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	if filepath.IsAbs(rel) {
		return false
	}
	return true
}

func (r *Report) remove(item Item) {
	r.Removed = append(r.Removed, item)
	r.RemovedCount++
	r.EstimatedBytes += item.EstimatedBytes
	r.RemovedByStatus[item.Status]++
}

func (r *Report) skip(item Item, reason string) {
	item.Reason = reason
	r.Skipped = append(r.Skipped, item)
	r.SkippedCount++
}

func (r *Report) errored(item Item, reason string) {
	item.Reason = reason
	r.Skipped = append(r.Skipped, item)
	r.SkippedCount++
	r.ErrorCount++
}
