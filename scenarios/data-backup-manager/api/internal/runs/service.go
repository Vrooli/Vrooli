package runs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"data-backup-manager/internal/clock"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/sources"
)

// Service is the application surface the runs handlers and the scheduler
// depend on. TriggerRun is the backup-run orchestration (FLOWS.md "Scheduled
// backup run"): per target × destination, capture → cap-check → snapshot →
// retention → record outcome, with partial-failure isolation and a
// backup-outcome event on close.
type Service interface {
	// TriggerRun executes plan planID now and returns the closed run.
	TriggerRun(ctx context.Context, planID string, trigger TriggerSource) (Run, error)
	GetRun(ctx context.Context, id string) (Run, error)
	ListRuns(ctx context.Context, planID string, limit int) ([]Run, error)
	// ListTargetStatus returns the last-success/last-run rollup. targetIDs
	// empty means "all known from run history".
	ListTargetStatus(ctx context.Context, targetIDs []string) ([]TargetStatus, error)
	// BrowseSnapshot lists entries within a snapshot in a destination.
	BrowseSnapshot(ctx context.Context, destinationID, snapshotID, path string) ([]engine.SnapshotEntry, error)
}

// Deps bundles the seams the run service orchestrates.
type Deps struct {
	Repo         Repository
	Plans        PlanLookup
	Targets      TargetLookup
	Destinations DestinationLookup
	Engine       engine.KopiaEngine
	Sources      *sources.Registry
	Events       EventSink
	Clock        clock.Clock
	// StagingRoot is the base directory capture artifacts are staged under
	// before snapshotting. Empty uses the OS temp dir. Each run gets a
	// subdirectory that is removed when the run closes.
	StagingRoot string
}

const defaultRunListLimit = 100

type service struct {
	deps Deps
}

// NewService constructs the production run service.
func NewService(d Deps) Service { return &service{deps: d} }

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) TriggerRun(ctx context.Context, planID string, trigger TriggerSource) (Run, error) {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return Run{}, ErrInvalidRun{Field: "plan_id", Reason: "required"}
	}
	plan, err := s.deps.Plans.PlanForRun(ctx, planID)
	if err != nil {
		return Run{}, err
	}

	now := s.deps.Clock.Now().UTC()
	run, err := s.deps.Repo.CreateRun(ctx, Run{
		PlanID:    planID,
		Trigger:   trigger,
		Status:    RunPending,
		StartedAt: now,
	})
	if err != nil {
		return Run{}, fmt.Errorf("create run: %w", err)
	}

	stageBase, cleanup, err := s.stagingDir(run.ID)
	if err != nil {
		return Run{}, err
	}
	defer cleanup()

	run.Status = RunCapturing // pending -> capturing
	outcomes := make([]TargetOutcome, 0, len(plan.TargetIDs)*len(plan.DestinationIDs))
	for _, targetID := range plan.TargetIDs {
		for _, destID := range plan.DestinationIDs {
			outcomes = append(outcomes, s.runOne(ctx, plan, targetID, destID, stageBase))
		}
	}

	var succeeded, failed, blocked int
	for _, o := range outcomes {
		switch o.Status {
		case OutcomeSucceeded:
			succeeded++
		case OutcomeFailed:
			failed++
		case OutcomeBlocked:
			blocked++
		}
	}
	run.Outcomes = outcomes
	run.FinishedAt = s.deps.Clock.Now().UTC()
	if len(outcomes) == 0 {
		run.Status = RunCompleted // empty plan: vacuously complete
	} else {
		run.Status = classifyTerminal(succeeded, failed, blocked)
	}

	saved, err := s.deps.Repo.SaveRun(ctx, run)
	if err != nil {
		return Run{}, fmt.Errorf("save run %q: %w", run.ID, err)
	}
	if s.deps.Events != nil {
		s.deps.Events.BackupOutcome(ctx, RunOutcomeEvent{
			RunID: saved.ID, PlanID: planID, Status: saved.Status,
			Succeeded: succeeded, Failed: failed, Blocked: blocked,
		})
	}
	return saved, nil
}

// runOne executes a single target × destination and returns its outcome. A
// failure here never aborts the run — the caller aggregates.
func (s *service) runOne(ctx context.Context, plan PlanForRun, targetID, destID, stageBase string) TargetOutcome {
	out := TargetOutcome{TargetID: targetID, DestinationID: destID, StartedAt: s.deps.Clock.Now().UTC()}
	finish := func(status OutcomeStatus, errMsg string) TargetOutcome {
		out.Status = status
		out.Error = errMsg
		out.FinishedAt = s.deps.Clock.Now().UTC()
		return out
	}

	target, err := s.deps.Targets.TargetForRun(ctx, targetID)
	if err != nil {
		return finish(OutcomeFailed, fmt.Sprintf("resolve target: %v", err))
	}
	dest, err := s.deps.Destinations.DestinationForRun(ctx, destID)
	if err != nil {
		return finish(OutcomeFailed, fmt.Sprintf("resolve destination: %v", err))
	}
	capturer, err := s.deps.Sources.Capturer(target.Kind)
	if err != nil {
		return finish(OutcomeFailed, fmt.Sprintf("source capturer: %v", err))
	}

	stageDir := filepath.Join(stageBase, sanitize(targetID)+"_"+sanitize(destID))
	if mkErr := os.MkdirAll(stageDir, 0o755); mkErr != nil {
		return finish(OutcomeFailed, fmt.Sprintf("stage dir: %v", mkErr))
	}

	artifact, err := capturer.Capture(ctx, sources.CaptureSpec{Locator: target.Locator, StageDir: stageDir})
	if err != nil {
		return finish(OutcomeFailed, fmt.Sprintf("capture: %v", err))
	}
	out.Bytes = artifact.Bytes

	// Storage-limit block: check BEFORE writing. Never evicts.
	blocked, reason, err := s.deps.Destinations.WouldBlock(ctx, destID, artifact.Bytes)
	if err != nil {
		return finish(OutcomeFailed, fmt.Sprintf("cap check: %v", err))
	}
	if blocked {
		return finish(OutcomeBlocked, "storage cap reached: "+reason)
	}

	snap, err := s.deps.Engine.SnapshotCreate(ctx, dest.Name, artifact.Path)
	if err != nil {
		return finish(OutcomeFailed, fmt.Sprintf("snapshot: %v", err))
	}
	out.SnapshotID = snap.ID

	// Retention is best-effort: a policy failure does not undo a good snapshot.
	if plan.KeepLatest > 0 {
		_ = s.deps.Engine.PolicySet(ctx, dest.Name, artifact.Path, plan.KeepLatest)
	}
	return finish(OutcomeSucceeded, "")
}

func (s *service) GetRun(ctx context.Context, id string) (Run, error) {
	return s.deps.Repo.GetRun(ctx, id)
}

func (s *service) ListRuns(ctx context.Context, planID string, limit int) ([]Run, error) {
	if limit <= 0 {
		limit = defaultRunListLimit
	}
	return s.deps.Repo.ListRuns(ctx, strings.TrimSpace(planID), limit)
}

func (s *service) ListTargetStatus(ctx context.Context, targetIDs []string) ([]TargetStatus, error) {
	return s.deps.Repo.TargetStatuses(ctx, targetIDs)
}

func (s *service) BrowseSnapshot(ctx context.Context, destinationID, snapshotID, path string) ([]engine.SnapshotEntry, error) {
	dest, err := s.deps.Destinations.DestinationForRun(ctx, destinationID)
	if err != nil {
		return nil, err
	}
	return s.deps.Engine.BrowseSnapshot(ctx, dest.Name, snapshotID, path)
}

// stagingDir creates a per-run staging directory and returns a cleanup func.
func (s *service) stagingDir(runID string) (string, func(), error) {
	base := s.deps.StagingRoot
	if base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "dbm-run-"+sanitize(runID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", func() {}, fmt.Errorf("staging dir: %w", err)
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
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
