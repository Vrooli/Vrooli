package runs

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"data-backup-manager/internal/clock"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/failures"
	"data-backup-manager/internal/preflight"
	"data-backup-manager/internal/sources"
	"github.com/vrooli/api-core/storage"
)

// Service is the application surface the runs handlers and the scheduler
// depend on. TriggerRun is the backup-run orchestration (FLOWS.md "Scheduled
// backup run"): per target × destination, capture → cap-check → snapshot →
// retention → record outcome, with partial-failure isolation and a
// backup-outcome event on close. Execution is asynchronous — TriggerRun
// enqueues onto a background worker and returns a non-terminal run immediately;
// callers poll GetRun/ListRuns for progress and terminal status.
type Service interface {
	// TriggerRun creates a run for plan planID, enqueues it for background
	// execution, and returns the freshly-created (pending) run immediately. The
	// run executes on the service's executor under a non-request context.
	TriggerRun(ctx context.Context, planID string, trigger TriggerSource) (Run, error)
	Preflight(ctx context.Context, planID string) (preflight.Result, error)
	GetRun(ctx context.Context, id string) (Run, error)
	ListRuns(ctx context.Context, planID string, limit int) ([]Run, error)
	// ListTargetStatus returns the last-success/last-run rollup. targetIDs
	// empty means "current catalog targets" when ActiveTargets is wired,
	// otherwise "all known from run history" for tests and legacy composition.
	ListTargetStatus(ctx context.Context, targetIDs []string) ([]TargetStatus, error)
	// BrowseSnapshot lists entries within a snapshot in a destination.
	BrowseSnapshot(ctx context.Context, destinationID, snapshotID, path string) ([]engine.SnapshotEntry, error)
	// GetRunStats aggregates run performance over the recent history window,
	// optionally scoped to a plan.
	GetRunStats(ctx context.Context, planID string) (RunStats, error)
	// Reconcile closes runs left non-terminal by a crash/restart/disconnect,
	// marking them failed. Called once at startup before serving traffic so no
	// run can wedge in a non-terminal state across a restart.
	Reconcile(ctx context.Context) error
	// Shutdown drains the background executor, bounded by ctx.
	Shutdown(ctx context.Context) error
}

// Deps bundles the seams the run service orchestrates.
type Deps struct {
	Repo          Repository
	Plans         PlanLookup
	Targets       TargetLookup
	ActiveTargets ActiveTargetLookup
	Destinations  DestinationLookup
	Engine        engine.KopiaEngine
	// NextSchedule resolves each target's next scheduled backup time. Optional —
	// nil omits next_scheduled_at from ListTargetStatus.
	NextSchedule NextScheduleSource
	Sources      *sources.Registry
	Events       EventSink
	Clock        clock.Clock
	Logger       *log.Logger
	// StagingRoot is the base directory capture artifacts are staged under
	// before snapshotting. Empty uses the OS temp dir. Each run gets a
	// subdirectory that is removed when the run closes.
	StagingRoot string
	// FileRoots routes scenario-owned staging files through the lifecycle's
	// test-isolation seam when present. StagingRoot remains an explicit test
	// override.
	RoutedRoots FileRootPicker
	// BaseContext is the server-lifetime context the background executor binds
	// its workers to. Nil uses context.Background(). It must outlive requests —
	// a client disconnect must not cancel an in-flight backup.
	BaseContext context.Context
	// Workers is the number of background run workers (concurrent runs). < 1
	// clamps to 1. Within-run target concurrency is a separate knob.
	Workers int
	// TargetConcurrency bounds how many target×destination units a single run
	// executes in parallel. < 1 uses defaultTargetConcurrency. Partial-failure
	// isolation and cap-block-before-write are invariant under concurrency.
	TargetConcurrency int
	// OverdueAfter is the age past which a target's last success is overdue.
	// <= 0 disables the age component (a target is then overdue only when its
	// last run failed/partial or it has never succeeded). ListTargetStatus uses
	// it to compute per-target freshness.
	OverdueAfter time.Duration
	// Executor overrides the background executor. Nil wires the production
	// AsyncExecutor. Tests inject a synchronous executor for deterministic
	// completion.
	Executor Executor
	// PreflightSourcePaths enables read-only existence checks for filesystem and
	// SQLite locators. Production enables it; unit tests with synthetic locators
	// can leave it false.
	PreflightSourcePaths bool
}

type FileRootPicker interface {
	Pick(context.Context, storage.Class) (string, error)
	RecordWrite(context.Context)
}

const (
	defaultRunListLimit      = 100
	defaultTargetConcurrency = 4
	// statsWindow caps how many recent runs GetRunStats aggregates. Run history
	// is bounded catalog data; the cap keeps the aggregation cheap and is
	// surfaced as RunStats.Window so the figure is never silently partial.
	statsWindow = 1000
)

type service struct {
	deps     Deps
	executor Executor
}

// NewService constructs the production run service and starts its background
// executor. The executor is bound to s.executeRun so enqueued jobs run the
// full capture→snapshot→persist lifecycle off the request path.
func NewService(d Deps) Service {
	s := &service{deps: d}
	exec := d.Executor
	if exec == nil {
		exec = NewAsyncExecutor(d.Workers)
	}
	s.executor = exec
	s.executor.Bind(d.BaseContext, s.executeRun)
	return s
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) logf(format string, args ...any) {
	if s.deps.Logger != nil {
		s.deps.Logger.Printf("runs: "+format, args...)
	}
}

func (s *service) TriggerRun(ctx context.Context, planID string, trigger TriggerSource) (Run, error) {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return Run{}, ErrInvalidRun{Field: "plan_id", Reason: "required"}
	}
	// Validate the plan exists up-front so the caller gets a synchronous 404
	// instead of a run that fails in the background.
	if _, err := s.deps.Plans.PlanForRun(ctx, planID); err != nil {
		return Run{}, err
	}
	if active, err := s.deps.Repo.ListNonTerminalRuns(ctx); err != nil {
		return Run{}, fmt.Errorf("check active runs: %w", err)
	} else {
		for _, existing := range active {
			if existing.PlanID == planID {
				return Run{}, ErrRunAlreadyActive{PlanID: planID, RunID: existing.ID}
			}
		}
	}

	now := s.deps.Clock.Now().UTC()
	run, err := s.deps.Repo.CreateRun(ctx, Run{
		PlanID:    planID,
		Trigger:   trigger,
		Status:    RunPending,
		StartedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return Run{}, fmt.Errorf("create run: %w", err)
	}

	s.executor.Submit(RunJob{RunID: run.ID, PlanID: planID, Trigger: trigger})
	return run, nil
}

// Preflight performs the same read-only checks a run uses, without creating a
// run, staging source data, or writing to a repository.
func (s *service) Preflight(ctx context.Context, planID string) (preflight.Result, error) {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return preflight.Result{}, ErrInvalidRun{Field: "plan_id", Reason: "required"}
	}
	plan, err := s.deps.Plans.PlanForRun(ctx, planID)
	if err != nil {
		return preflight.Result{}, err
	}
	return s.runPreflight(ctx, plan), nil
}

// executeRun is the background worker body: it drives one created run through
// its persisted lifecycle (capturing → snapshotting → terminal), writing each
// target outcome as it lands so progress is durable before the run closes. It
// runs under the executor's base context, so a client disconnect cannot cancel
// it. It never returns an error — failures are persisted onto the run/outcomes.
func (s *service) executeRun(ctx context.Context, job RunJob) {
	plan, err := s.deps.Plans.PlanForRun(ctx, job.PlanID)
	if err != nil {
		s.finishRun(ctx, job, RunFailed, fmt.Sprintf("resolve plan: %v", err), failures.Unknown, failures.CategoryExecution, "inspect the plan catalog and retry", 0, 0, 0, 0)
		return
	}
	preflightResult := s.runPreflight(ctx, plan)
	if len(preflightResult.Incidents) > 0 {
		first := preflightResult.Incidents[0]
		var evidenceErrs []string
		if updater, ok := s.deps.Repo.(FailureUpdater); ok {
			if err := updater.UpdateRunFailure(ctx, job.RunID, first.Code, first.Category, first.NextAction); err != nil {
				s.logf("run %s persist preflight failure: %v", job.RunID, err)
				evidenceErrs = append(evidenceErrs, fmt.Sprintf("run failure: %v", err))
			}
		}
		if updater, ok := s.deps.Repo.(IncidentUpdater); ok {
			if err := updater.SavePreflightIncidents(ctx, job.RunID, preflightResult.Incidents); err != nil {
				s.logf("run %s persist preflight incidents: %v", job.RunID, err)
				evidenceErrs = append(evidenceErrs, fmt.Sprintf("preflight incidents: %v", err))
			}
		}
		// A shared prerequisite blocks the plan before staging, repository-size
		// probes, or target fan-out. Keep one outcome per planned unit for
		// precise impact while retaining the grouped incident as the root cause.
		blocked := 0
		for _, targetID := range plan.TargetIDs {
			for _, destinationID := range plan.DestinationIDs {
				cause, targetBlocked := preflightResult.BlockedTargets[targetID]
				if !targetBlocked {
					cause = preflightResult.BlockedDestinations[destinationID]
				}
				if cause.Code == "" {
					cause = first
				}
				o := TargetOutcome{
					TargetID: targetID, DestinationID: destinationID, Status: OutcomeBlocked,
					Error:       cause.Message + "; " + cause.NextAction,
					FailureCode: cause.Code, FailureCategory: cause.Category,
					StartedAt: preflightResult.CheckedAt, FinishedAt: s.deps.Clock.Now().UTC(),
				}
				if err := s.deps.Repo.SaveOutcome(ctx, job.RunID, o); err != nil {
					s.logf("run %s save preflight outcome %s/%s: %v", job.RunID, targetID, destinationID, err)
					evidenceErrs = append(evidenceErrs, fmt.Sprintf("outcome %s/%s: %v", targetID, destinationID, err))
				}
				blocked++
			}
		}
		errMsg := preflightResult.Summary()
		if len(evidenceErrs) > 0 {
			errMsg += "; evidence_persistence_failed: " + strings.Join(evidenceErrs, " | ")
		}
		s.finishRun(ctx, job, RunFailed, errMsg, first.Code, first.Category, first.NextAction, 0, 0, blocked, 0)
		return
	}

	if err := s.deps.Repo.UpdateRunStatus(ctx, job.RunID, RunCapturing); err != nil {
		s.logf("run %s -> capturing: %v", job.RunID, err)
		s.finishRun(ctx, job, RunFailed, fmt.Sprintf("could not persist capturing transition: %v", err), failures.Unknown, failures.CategoryExecution, "repair catalog storage and retry the backup", 0, 0, 0, 0)
		return
	}

	stageBase, cleanup, err := s.stagingDir(ctx, job.RunID)
	if err != nil {
		s.finishRun(ctx, job, RunFailed, err.Error(), failures.Unknown, failures.CategoryExecution, "inspect staging storage and retry", 0, 0, 0, 0)
		return
	}

	// Physical-bytes baseline: snapshot each destination repo's on-disk size
	// before fan-out, so the post-run delta is the deduped growth this run
	// caused. Best-effort observability — a measurement failure just omits that
	// destination from the metric and never blocks the backup.
	preSizes := s.measureRepoSizes(ctx, plan.DestinationIDs)

	// The run transitions to snapshotting once the first target begins its
	// snapshot — guarded so the persisted transition happens exactly once even
	// under concurrent target fan-out.
	var (
		once          sync.Once
		lifecycleMu   sync.Mutex
		lifecycleErrs []string
	)
	onSnapshotStart := func() {
		once.Do(func() {
			if err := s.deps.Repo.UpdateRunStatus(ctx, job.RunID, RunSnapshotting); err != nil {
				s.logf("run %s -> snapshotting: %v", job.RunID, err)
				lifecycleMu.Lock()
				lifecycleErrs = append(lifecycleErrs, fmt.Sprintf("snapshotting transition: %v", err))
				lifecycleMu.Unlock()
			}
		})
	}

	// Fan out target×destination units with bounded concurrency. Each runOne is
	// independent; outcomes persist as each completes (SaveOutcome). The tally
	// and the run-level snapshotting transition are guarded so partial-failure
	// isolation and cap-block-before-write hold unchanged under concurrency.
	type unit struct{ targetID, destID string }
	units := make([]unit, 0, len(plan.TargetIDs)*len(plan.DestinationIDs))
	for _, targetID := range plan.TargetIDs {
		for _, destID := range plan.DestinationIDs {
			units = append(units, unit{targetID, destID})
		}
	}

	var (
		mu                                   sync.Mutex
		succeeded, failed, blocked, warnings int
		evidenceErrs                         []string
		wg                                   sync.WaitGroup
	)
	sem := make(chan struct{}, s.targetConcurrency())
	for _, u := range units {
		wg.Add(1)
		sem <- struct{}{}
		go func(u unit) {
			defer wg.Done()
			defer func() { <-sem }()
			o := s.runOne(ctx, plan, job.RunID, u.targetID, u.destID, stageBase, onSnapshotStart, preflightResult)
			if err := s.deps.Repo.SaveOutcome(ctx, job.RunID, o); err != nil {
				s.logf("run %s save outcome %s/%s: %v", job.RunID, u.targetID, u.destID, err)
				mu.Lock()
				evidenceErrs = append(evidenceErrs, fmt.Sprintf("outcome %s/%s: %v", u.targetID, u.destID, err))
				mu.Unlock()
			}
			mu.Lock()
			switch o.Status {
			case OutcomeSucceeded:
				succeeded++
			case OutcomeFailed:
				failed++
			case OutcomeBlocked:
				blocked++
			}
			if o.Warning != "" {
				warnings++
			}
			mu.Unlock()
		}(u)
	}
	wg.Wait()
	cleanupErr := cleanup()

	// Measure again and attribute the deduped repo growth to this run. Only
	// destinations with a baseline count (a missing baseline would make the
	// whole repo look like this run's delta); negative deltas (maintenance or a
	// concurrent run's compaction) are clamped to 0.
	physicalBytes := repoGrowth(preSizes, s.measureRepoSizes(ctx, plan.DestinationIDs))
	lifecycleMu.Lock()
	evidenceErrs = append(evidenceErrs, lifecycleErrs...)
	lifecycleMu.Unlock()

	status := RunCompleted // empty plan: vacuously complete
	if succeeded+failed+blocked > 0 {
		status = classifyTerminal(succeeded, failed, blocked)
	}
	if warnings > 0 && status == RunCompleted {
		status = RunPartialFailed
	}
	errMsg := preflightResult.Summary()
	if errMsg == "preflight ready" {
		errMsg = ""
	}
	if warnings > 0 {
		if errMsg != "" {
			errMsg += "; "
		}
		errMsg += fmt.Sprintf("retention_failed: %d outcome(s) require retention maintenance", warnings)
	}
	if len(evidenceErrs) > 0 {
		if errMsg != "" {
			errMsg += "; "
		}
		errMsg += "evidence_persistence_failed: " + strings.Join(evidenceErrs, " | ")
		if status == RunCompleted {
			status = RunPartialFailed
		}
	}
	if cleanupErr != nil {
		if errMsg != "" {
			errMsg += "; "
		}
		errMsg += "cleanup_failed: staging cleanup could not be completed"
		failed++
		if status == RunCompleted {
			status = RunPartialFailed
		}
	}
	code, category, next := failures.Code(""), failures.Category(""), ""
	if len(preflightResult.Incidents) > 0 {
		code, category, next = preflightResult.Incidents[0].Code, preflightResult.Incidents[0].Category, preflightResult.Incidents[0].NextAction
	}
	s.finishRun(ctx, job, status, errMsg, code, category, next, succeeded, failed, blocked, physicalBytes)
}

// measureRepoSizes returns the current physical (on-disk, deduped) size of each
// destination repo keyed by destination id. It is best-effort: a resolve or
// repo-stats failure logs and omits that destination rather than failing the
// run, because the physical-bytes metric is observability, never a gate.
func (s *service) measureRepoSizes(ctx context.Context, destIDs []string) map[string]int64 {
	sizes := make(map[string]int64, len(destIDs))
	for _, destID := range destIDs {
		dest, err := s.deps.Destinations.DestinationForRun(ctx, destID)
		if err != nil {
			s.logf("physical-bytes: resolve destination %s: %v", destID, err)
			continue
		}
		stats, err := s.deps.Engine.RepoStats(ctx, dest.Name)
		if err != nil {
			s.logf("physical-bytes: repo stats %s: %v", dest.Name, err)
			continue
		}
		sizes[destID] = stats.SizeBytes
	}
	return sizes
}

// repoGrowth sums the post-minus-pre repo-size delta across destinations that
// were measured both before and after the run, clamping negative deltas to 0.
func repoGrowth(pre, post map[string]int64) int64 {
	var total int64
	for destID, after := range post {
		before, ok := pre[destID]
		if !ok {
			continue // no baseline — don't mistake the whole repo for this run
		}
		if d := after - before; d > 0 {
			total += d
		}
	}
	return total
}

// targetConcurrency bounds in-run target×destination parallelism (>= 1).
func (s *service) targetConcurrency() int {
	if s.deps.TargetConcurrency < 1 {
		return defaultTargetConcurrency
	}
	return s.deps.TargetConcurrency
}

// finishRun persists the terminal state and emits the backup-outcome event.
func (s *service) finishRun(ctx context.Context, job RunJob, status RunStatus, errMsg string, code failures.Code, category failures.Category, nextAction string, succeeded, failed, blocked int, physicalBytes int64) {
	if err := s.deps.Repo.FinishRun(ctx, job.RunID, status, errMsg, s.deps.Clock.Now().UTC(), physicalBytes); err != nil {
		s.logf("run %s finish (%s): %v", job.RunID, status, err)
	}
	if updater, ok := s.deps.Repo.(FailureUpdater); ok && code != "" {
		if err := updater.UpdateRunFailure(ctx, job.RunID, code, category, nextAction); err != nil {
			s.logf("run %s finish failure evidence: %v", job.RunID, err)
		}
	}
	if s.deps.Events != nil {
		s.deps.Events.BackupOutcome(ctx, RunOutcomeEvent{
			RunID: job.RunID, PlanID: job.PlanID, Status: status,
			Succeeded: succeeded, Failed: failed, Blocked: blocked,
		})
	}
}

// Reconcile closes any run left in a non-terminal state by a crash/restart or a
// client disconnect that killed an in-flight backup. v1 policy is fail-not-
// resume: each orphan is marked failed with a reconciliation reason. Resume is
// a deliberate future option (see docs/internal/PROBLEMS.md).
func (s *service) Reconcile(ctx context.Context) error {
	orphans, err := s.deps.Repo.ListNonTerminalRuns(ctx)
	if err != nil {
		return fmt.Errorf("reconcile: list non-terminal runs: %w", err)
	}
	const reason = "reconciled: process restarted while run was in-flight"
	for _, r := range orphans {
		if err := s.deps.Repo.FinishRun(ctx, r.ID, RunFailed, reason, s.deps.Clock.Now().UTC(), 0); err != nil {
			s.logf("reconcile run %s: %v", r.ID, err)
			continue
		}
		if updater, ok := s.deps.Repo.(FailureUpdater); ok {
			if err := updater.UpdateRunFailure(ctx, r.ID, failures.ProcessInterrupted, failures.CategoryExecution, "inspect the interrupted run and retry after the service is healthy"); err != nil {
				s.logf("reconcile run %s failure evidence: %v", r.ID, err)
			}
		}
	}
	if len(orphans) > 0 {
		s.logf("reconciled %d orphaned non-terminal run(s) to failed", len(orphans))
	}
	return nil
}

// Shutdown drains the background executor, bounded by ctx.
func (s *service) Shutdown(ctx context.Context) error {
	return s.executor.Shutdown(ctx)
}

// runOne executes a single target × destination and returns its outcome. A
// failure here never aborts the run — the caller aggregates. onSnapshotStart is
// invoked once the capture succeeds and the snapshot is about to begin, so the
// run can record its capturing→snapshotting transition.
func (s *service) runOne(ctx context.Context, plan PlanForRun, runID, targetID, destID, stageBase string, onSnapshotStart func(), pf preflight.Result) TargetOutcome {
	out := TargetOutcome{TargetID: targetID, DestinationID: destID, StartedAt: s.deps.Clock.Now().UTC()}
	finish := func(status OutcomeStatus, errMsg string) TargetOutcome {
		out.Status = status
		out.Error = errMsg
		out.FinishedAt = s.deps.Clock.Now().UTC()
		return out
	}
	if cause, ok := pf.BlockedDestinations[destID]; ok {
		out.FailureCode, out.FailureCategory = cause.Code, cause.Category
		return finish(OutcomeBlocked, cause.Message+"; "+cause.NextAction)
	}
	if cause, ok := pf.BlockedTargets[targetID]; ok {
		out.FailureCode, out.FailureCategory = cause.Code, cause.Category
		return finish(OutcomeBlocked, cause.Message+"; "+cause.NextAction)
	}

	target, err := s.deps.Targets.TargetForRun(ctx, targetID)
	if err != nil {
		cause := failures.Classify(err)
		out.FailureCode, out.FailureCategory = cause.Code, cause.Category
		return finish(OutcomeFailed, cause.Message)
	}
	dest, err := s.deps.Destinations.DestinationForRun(ctx, destID)
	if err != nil {
		cause := failures.Classify(err)
		out.FailureCode, out.FailureCategory = cause.Code, cause.Category
		return finish(OutcomeFailed, cause.Message)
	}
	capturer, err := s.deps.Sources.Capturer(target.Kind)
	if err != nil {
		cause := failures.Classify(err)
		out.FailureCode, out.FailureCategory = cause.Code, cause.Category
		return finish(OutcomeFailed, cause.Message)
	}

	stageDir := filepath.Join(stageBase, sanitize(targetID)+"_"+sanitize(destID))
	if mkErr := os.MkdirAll(stageDir, 0o755); mkErr != nil {
		out.FailureCode, out.FailureCategory = failures.Unknown, failures.CategoryExecution
		return finish(OutcomeFailed, "staging directory could not be created")
	}

	artifact, err := capturer.Capture(ctx, sources.CaptureSpec{Locator: target.Locator, StageDir: stageDir})
	if err != nil {
		cause := failures.Classify(err)
		out.FailureCode, out.FailureCategory = cause.Code, cause.Category
		return finish(OutcomeFailed, cause.Message)
	}
	out.Bytes = artifact.Bytes

	// Storage-limit block: check BEFORE writing. Never evicts.
	blocked, reason, err := s.deps.Destinations.WouldBlock(ctx, destID, artifact.Bytes)
	if err != nil {
		out.FailureCode, out.FailureCategory = failures.DestinationCapacity, failures.CategoryCapacity
		return finish(OutcomeFailed, "destination capacity could not be checked")
	}
	if blocked {
		out.FailureCode, out.FailureCategory = failures.DestinationCapacity, failures.CategoryCapacity
		return finish(OutcomeBlocked, "storage cap reached: "+reason)
	}

	if onSnapshotStart != nil {
		onSnapshotStart()
	}
	snap, err := s.deps.Engine.SnapshotCreate(ctx, dest.Name, artifact.Path,
		snapshotMetadata(runID, destID, target))
	if err != nil {
		cause := failures.Classify(err)
		out.FailureCode, out.FailureCategory = cause.Code, cause.Category
		return finish(OutcomeFailed, cause.Message)
	}
	out.SnapshotID = snap.ID

	// Retention is best-effort: a policy failure does not undo a good snapshot.
	if plan.KeepLatest > 0 {
		if retentionErr := s.deps.Engine.PolicySet(ctx, dest.Name, artifact.Path, plan.KeepLatest); retentionErr != nil {
			out.Warning = "retention_failed: retention policy could not be applied"
		}
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
	if len(targetIDs) == 0 && s.deps.ActiveTargets != nil {
		activeIDs, err := s.deps.ActiveTargets.ActiveTargetIDs(ctx)
		if err != nil {
			return nil, err
		}
		if len(activeIDs) == 0 {
			return []TargetStatus{}, nil
		}
		targetIDs = activeIDs
	}
	statuses, err := s.deps.Repo.TargetStatuses(ctx, targetIDs)
	if err != nil {
		return nil, err
	}
	now := s.deps.Clock.Now().UTC()
	// Next scheduled fire per target (best-effort: a failure here must not break
	// the freshness rollup, which is load-bearing for /health and the UI).
	var nextByTarget map[string]time.Time
	if s.deps.NextSchedule != nil {
		if m, err := s.deps.NextSchedule.NextScheduledByTarget(ctx); err != nil {
			s.logf("next-scheduled lookup: %v", err)
		} else {
			nextByTarget = m
		}
	}
	for i := range statuses {
		statuses[i].Overdue = isOverdue(statuses[i], now, s.deps.OverdueAfter)
		if !statuses[i].LastSuccessAt.IsZero() {
			if age := now.Sub(statuses[i].LastSuccessAt); age > 0 {
				statuses[i].LastSuccessAgeSeconds = int64(age.Seconds())
			}
		}
		if next, ok := nextByTarget[statuses[i].TargetID]; ok {
			statuses[i].NextScheduledAt = next.UTC()
		}
	}
	return statuses, nil
}

// isOverdue is the single freshness rule shared by every surface (CLI, /health,
// UI): a target is overdue when its last run failed or partial-failed, it has
// never succeeded, or its last success is older than the overdue threshold.
// A non-positive threshold disables only the age component.
func isOverdue(s TargetStatus, now time.Time, after time.Duration) bool {
	switch s.LastRunStatus {
	case RunFailed, RunPartialFailed:
		return true
	}
	if s.LastSuccessAt.IsZero() {
		return true
	}
	return after > 0 && now.Sub(s.LastSuccessAt) > after
}

func (s *service) GetRunStats(ctx context.Context, planID string) (RunStats, error) {
	runs, err := s.deps.Repo.ListRuns(ctx, strings.TrimSpace(planID), statsWindow)
	if err != nil {
		return RunStats{}, err
	}
	return computeRunStats(runs), nil
}

func (s *service) BrowseSnapshot(ctx context.Context, destinationID, snapshotID, path string) ([]engine.SnapshotEntry, error) {
	dest, err := s.deps.Destinations.DestinationForRun(ctx, destinationID)
	if err != nil {
		return nil, err
	}
	return s.deps.Engine.BrowseSnapshot(ctx, dest.Name, snapshotID, path)
}

// stagingDir creates a per-run staging directory and returns a cleanup func.
func (s *service) stagingDir(ctx context.Context, runID string) (string, func() error, error) {
	base := s.deps.StagingRoot
	if base == "" && s.deps.RoutedRoots != nil {
		dataRoot, err := s.deps.RoutedRoots.Pick(ctx, storage.ClassData)
		if err != nil {
			return "", func() error { return nil }, fmt.Errorf("staging root: %w", err)
		}
		base = filepath.Join(dataRoot, "staging")
	}
	if base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "dbm-run-"+sanitize(runID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", func() error { return nil }, fmt.Errorf("staging dir: %w", err)
	}
	if s.deps.RoutedRoots != nil && s.deps.StagingRoot == "" {
		s.deps.RoutedRoots.RecordWrite(ctx)
	}
	return dir, func() error { return os.RemoveAll(dir) }, nil
}

// snapshotMetadata builds the self-identifying kopia metadata stamped on each
// snapshot so a standalone `kopia snapshot list --json` can attribute it to a
// DBM owner/name/kind/run/destination without DBM running. It deliberately
// omits the target locator (potentially sensitive) and never carries a secret.
func snapshotMetadata(runID, destID string, target TargetForRun) engine.SnapshotMetadata {
	owner := strings.TrimSpace(target.Owner)
	name := strings.TrimSpace(target.Name)
	ref := name
	if owner != "" {
		ref = owner + "/" + name
	}
	tags := []string{
		"dbm:true",
		"dbm.target_id:" + target.ID,
		"dbm.kind:" + string(target.Kind),
		"dbm.run_id:" + runID,
		"dbm.destination_id:" + destID,
	}
	if owner != "" {
		tags = append(tags, "dbm.owner:"+owner)
	}
	if name != "" {
		tags = append(tags, "dbm.name:"+name)
	}
	return engine.SnapshotMetadata{
		Description:    fmt.Sprintf("Data Backup Manager target %s run %s", ref, runID),
		OverrideSource: "dbm://" + ref,
		Tags:           tags,
	}
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
