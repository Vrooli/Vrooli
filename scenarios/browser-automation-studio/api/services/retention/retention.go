// Package retention implements execution-artifact retention for Browser
// Automation Studio. It selects terminal executions (completed, failed) that
// are eligible for removal and, in apply mode, deletes their artifact
// directories and database index rows together.
//
// The package owns the retention business logic; transport handlers (Connect
// RPC) and the CLI are thin wrappers that convert request/response shapes and
// call Sweep. Filesystem access goes through the FileSystem seam so tests can
// run without touching real disk and so deletion stays constrained to the
// configured recordings root.
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
	"github.com/sirupsen/logrus"

	"github.com/vrooli/browser-automation-studio/database"
)

// Skip/annotation reasons reported per execution.
const (
	ReasonNonTerminal     = "non-terminal execution"
	ReasonKeepLatest      = "protected by keep_latest"
	ReasonTooNew          = "newer than max_age_days"
	ReasonUnsafePath      = "artifact directory outside recordings root"
	ReasonMissingDir      = "artifact directory already absent"
	ReasonDeleteDirFailed = "failed to delete artifact directory"
	ReasonDeleteRowFailed = "failed to delete execution row"
)

// ErrRecordingsRootNotConfigured is returned when no recordings root is set, in
// which case retention cannot resolve or safely delete artifact directories.
var ErrRecordingsRootNotConfigured = errors.New("recordings root not configured")

// FileSystem is the dependency-injected filesystem seam used by retention.
type FileSystem interface {
	// DirSize returns the total bytes under dir and whether dir exists. A missing
	// directory returns (0, false, nil) rather than an error.
	DirSize(dir string) (sizeBytes int64, exists bool, err error)
	// RemoveAll removes dir and everything beneath it. Removing a missing dir is
	// a no-op (no error), mirroring os.RemoveAll.
	RemoveAll(dir string) error
}

// ExecutionStore is the minimal repository surface retention needs.
type ExecutionStore interface {
	ListExecutions(ctx context.Context, workflowID *uuid.UUID, projectID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error)
	ListExecutionsByStatus(ctx context.Context, status string, limit, offset int) ([]*database.ExecutionIndex, error)
	DeleteExecution(ctx context.Context, id uuid.UUID) error
}

// Service performs retention sweeps.
type Service struct {
	store          ExecutionStore
	fs             FileSystem
	recordingsRoot string
	now            func() time.Time
	log            *logrus.Logger
}

// NewService constructs a retention Service. When now is nil, time.Now is used.
func NewService(store ExecutionStore, fs FileSystem, recordingsRoot string, log *logrus.Logger) *Service {
	return &Service{
		store:          store,
		fs:             fs,
		recordingsRoot: strings.TrimSpace(recordingsRoot),
		now:            time.Now,
		log:            log,
	}
}

// WithClock overrides the time source (used in tests).
func (s *Service) WithClock(now func() time.Time) *Service {
	if now != nil {
		s.now = now
	}
	return s
}

// Options describe a retention sweep.
type Options struct {
	// MaxAgeDays removes executions older than this many days (by completed_at
	// when present, otherwise started_at). 0 disables the age filter.
	MaxAgeDays int
	// KeepLatest spares this many most-recent terminal executions per workflow.
	KeepLatest int
	// WorkflowID, ProjectID optionally narrow the candidate set.
	WorkflowID *uuid.UUID
	ProjectID  *uuid.UUID
	// Status optionally restricts to a single terminal status; empty means both
	// completed and failed are eligible.
	Status string
	// Apply performs deletion. When false, the sweep is a pure dry-run.
	Apply bool
}

// Item describes one execution considered by a sweep.
type Item struct {
	ExecutionID    uuid.UUID
	Status         string
	WorkflowID     uuid.UUID
	StartedAt      time.Time
	CompletedAt    *time.Time
	ResultPath     string
	ArtifactDir    string
	EstimatedBytes int64
	Reason         string
}

// Report is the structured result of a sweep.
type Report struct {
	DryRun          bool
	Removed         []Item
	Skipped         []Item
	EstimatedBytes  int64
	RemovedCount    int
	SkippedCount    int
	ErrorCount      int
	RemovedByStatus map[string]int
}

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

	protected := s.computeProtected(candidates, opts.KeepLatest)

	cleanRoot := filepath.Clean(s.recordingsRoot)
	var cutoff time.Time
	if opts.MaxAgeDays > 0 {
		cutoff = s.now().Add(-time.Duration(opts.MaxAgeDays) * 24 * time.Hour)
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
		if opts.MaxAgeDays > 0 && effectiveTime(exec).After(cutoff) {
			report.skip(item, ReasonTooNew)
			continue
		}

		artifactDir, ok := resolveArtifactDir(cleanRoot, exec)
		if !ok {
			report.skip(item, ReasonUnsafePath)
			continue
		}
		item.ArtifactDir = artifactDir

		size, exists, sizeErr := s.fs.DirSize(artifactDir)
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
			if err := s.fs.RemoveAll(artifactDir); err != nil {
				if s.log != nil {
					s.log.WithError(err).WithField("artifact_dir", artifactDir).Error("retention: failed to remove artifact directory")
				}
				report.errored(item, fmt.Sprintf("%s: %v", ReasonDeleteDirFailed, err))
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

func (s *Service) gatherCandidates(ctx context.Context, opts Options, status string) ([]*database.ExecutionIndex, error) {
	targetStatuses := []string{database.ExecutionStatusCompleted, database.ExecutionStatusFailed}
	if status != "" {
		targetStatuses = []string{status}
	}

	// When a workflow/project filter is present the repository can only filter by
	// those dimensions, so we list and filter to terminal statuses in memory.
	if opts.WorkflowID != nil || opts.ProjectID != nil {
		all, err := s.store.ListExecutions(ctx, opts.WorkflowID, opts.ProjectID, 0, 0)
		if err != nil {
			return nil, fmt.Errorf("list executions: %w", err)
		}
		want := map[string]bool{}
		for _, st := range targetStatuses {
			want[st] = true
		}
		out := make([]*database.ExecutionIndex, 0, len(all))
		for _, e := range all {
			if e != nil && want[e.Status] {
				out = append(out, e)
			}
		}
		return out, nil
	}

	var out []*database.ExecutionIndex
	for _, st := range targetStatuses {
		list, err := s.store.ListExecutionsByStatus(ctx, st, 0, 0)
		if err != nil {
			return nil, fmt.Errorf("list executions by status %q: %w", st, err)
		}
		out = append(out, list...)
	}
	return out, nil
}

func (s *Service) computeProtected(candidates []*database.ExecutionIndex, keepLatest int) map[uuid.UUID]bool {
	protected := map[uuid.UUID]bool{}
	if keepLatest <= 0 {
		return protected
	}
	byWorkflow := map[uuid.UUID][]*database.ExecutionIndex{}
	for _, e := range candidates {
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
