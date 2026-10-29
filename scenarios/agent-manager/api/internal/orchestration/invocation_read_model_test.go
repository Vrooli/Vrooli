package orchestration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

type recordingInvocationReadModel struct {
	mu             sync.Mutex
	facts          []invocationreadmodel.Fact
	watermark      *invocationreadmodel.Watermark
	err            error
	findingMetrics invocationreadmodel.FindingMetrics
	findingErr     error
	calls          chan struct{}
}

func (s *recordingInvocationReadModel) Replace(_ context.Context, facts []invocationreadmodel.Fact, watermark invocationreadmodel.Watermark) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls != nil {
		s.calls <- struct{}{}
	}
	if s.err != nil {
		return s.err
	}
	s.facts = append([]invocationreadmodel.Fact(nil), facts...)
	w := watermark
	s.watermark = &w
	return nil
}

func (s *recordingInvocationReadModel) Facts(_ context.Context, _ string) ([]invocationreadmodel.Fact, error) {
	return s.facts, nil
}

func (s *recordingInvocationReadModel) Watermark(_ context.Context, _ string) (*invocationreadmodel.Watermark, error) {
	return s.watermark, nil
}

func (s *recordingInvocationReadModel) Cohort(context.Context, invocationreadmodel.Filter, int) (invocationreadmodel.Cohort, error) {
	return invocationreadmodel.Cohort{}, nil
}

func (s *recordingInvocationReadModel) Metrics(context.Context, invocationreadmodel.Filter) (invocationreadmodel.Metrics, error) {
	return invocationreadmodel.Metrics{}, nil
}

func (s *recordingInvocationReadModel) RunMetrics(context.Context, invocationreadmodel.Filter) (invocationreadmodel.RunMetrics, error) {
	return invocationreadmodel.RunMetrics{}, nil
}

func (s *recordingInvocationReadModel) WindowedUsage(context.Context, invocationreadmodel.Filter) (invocationreadmodel.UsageTotals, error) {
	return invocationreadmodel.UsageTotals{}, nil
}

func (s *recordingInvocationReadModel) FallbackInsights(context.Context, invocationreadmodel.Filter) (invocationreadmodel.FallbackInsights, error) {
	return invocationreadmodel.FallbackInsights{}, nil
}

func (s *recordingInvocationReadModel) RunDurationStatistics(context.Context, invocationreadmodel.Filter) (invocationreadmodel.RunDurationStatistics, error) {
	return invocationreadmodel.RunDurationStatistics{}, nil
}

func (s *recordingInvocationReadModel) RunStatusCounts(context.Context, invocationreadmodel.Filter) ([]invocationreadmodel.RunStatusCount, error) {
	return nil, nil
}

func (s *recordingInvocationReadModel) RunBreakdown(context.Context, invocationreadmodel.Filter, string, int) ([]invocationreadmodel.RunBreakdownRow, error) {
	return nil, nil
}

func (s *recordingInvocationReadModel) RunTimeSeries(context.Context, invocationreadmodel.Filter, time.Duration) ([]invocationreadmodel.RunTimeSeriesBucket, error) {
	return nil, nil
}

func (s *recordingInvocationReadModel) ToolUsage(context.Context, invocationreadmodel.Filter, int) ([]invocationreadmodel.ToolUsageRow, error) {
	return nil, nil
}

func (s *recordingInvocationReadModel) ErrorPatterns(context.Context, invocationreadmodel.Filter, int) ([]invocationreadmodel.ErrorPattern, error) {
	return nil, nil
}

func (s *recordingInvocationReadModel) FindingMetrics(context.Context, invocationreadmodel.Filter) (invocationreadmodel.FindingMetrics, error) {
	return s.findingMetrics, s.findingErr
}

// A terminal writer leaves a durable mark; the reconcile sweep projects it
// once, and a second sweep finds nothing left to do.
func TestTerminalProjectionIsDurableMarkProjectedBySweep(t *testing.T) {
	repos, events, cleanup := testutil.SetupTestRepos(t)
	defer cleanup()
	ctx := context.Background()
	o := New(nil, nil, repos.Runs, WithEvents(events), WithInvocationReadModel(repos.InvocationReadModel), WithClock(func() time.Time { return time.Date(2026, 7, 29, 18, 30, 0, 0, time.UTC) }))
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	call := domain.NewToolCallEvent(run.ID, "shell", "call-1", map[string]any{"command": "vrooli help"})
	call.Timestamp = time.Date(2026, 7, 29, 18, 29, 0, 0, time.UTC)
	if err := events.Append(ctx, run.ID, call); err != nil {
		t.Fatal(err)
	}
	o.projectTerminalInvocationReadModel(run)
	o.projectTerminalInvocationReadModel(run)
	if watermark, err := repos.InvocationReadModel.Watermark(ctx, run.ID.String()); err != nil || watermark != nil {
		t.Fatalf("marking projected synchronously: %+v %v", watermark, err)
	}
	if err := o.ProjectPendingInvocationReadModels(ctx); err != nil {
		t.Fatal(err)
	}
	watermark, err := repos.InvocationReadModel.Watermark(ctx, run.ID.String())
	if err != nil || watermark == nil || watermark.LastEventID != call.ID.String() || watermark.ClassifierVersion != runsignal.InvocationFactVersion {
		t.Fatalf("watermark=%+v err=%v", watermark, err)
	}
	queue := repos.InvocationReadModel.(invocationreadmodel.ProjectionQueue)
	if due, err := queue.DueProjections(ctx, time.Now().Add(time.Hour), 10); err != nil || len(due) != 0 {
		t.Fatalf("projected mark not cleared: %+v %v", due, err)
	}
	if err := o.ProjectPendingInvocationReadModels(ctx); err != nil {
		t.Fatalf("idempotent sweep: %v", err)
	}
	if due, err := queue.DueProjections(ctx, time.Now().Add(time.Hour), 10); err != nil || len(due) != 0 {
		t.Fatalf("a current watermark was re-enqueued: %+v %v", due, err)
	}

	// An episode classifier change (F-W5-4) re-projects ended history like a
	// fact classifier change does.
	ended := time.Date(2026, 7, 29, 18, 29, 30, 0, time.UTC)
	run.EndedAt, run.Status = &ended, domain.RunStatusComplete
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	if added, err := queue.EnqueueStaleProjections(ctx, runsignal.InvocationFactVersion, runsignal.EpisodeClassifierVersion, time.Now(), 10); err != nil || added != 0 {
		t.Fatalf("a current watermark was enqueued: %d %v", added, err)
	}
	stale := *watermark
	stale.EpisodeClassifierVersion = "friction-episode.v9"
	if err := repos.InvocationReadModel.Replace(ctx, nil, stale); err != nil {
		t.Fatal(err)
	}
	if added, err := queue.EnqueueStaleProjections(ctx, runsignal.InvocationFactVersion, runsignal.EpisodeClassifierVersion, time.Now(), 10); err != nil || added != 1 {
		t.Fatalf("stale episode classifier enqueued %d runs (%v), want 1", added, err)
	}
}

type failingReadModel struct {
	invocationreadmodel.Store
	invocationreadmodel.ProjectionQueue
}

func (failingReadModel) Replace(context.Context, []invocationreadmodel.Fact, invocationreadmodel.Watermark) error {
	return errors.New("projection unavailable")
}

// A failed projection keeps its mark with backoff and never touches the run.
func TestProjectionSweepDefersFailureWithoutMutatingTerminalRun(t *testing.T) {
	repos, events, cleanup := testutil.SetupTestRepos(t)
	defer cleanup()
	ctx := context.Background()
	queue := repos.InvocationReadModel.(invocationreadmodel.ProjectionQueue)
	o := New(nil, nil, repos.Runs, WithEvents(events), WithInvocationReadModel(failingReadModel{Store: repos.InvocationReadModel, ProjectionQueue: queue}))
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	o.projectTerminalInvocationReadModel(run)
	if err := o.ProjectPendingInvocationReadModels(ctx); err != nil {
		t.Fatal(err)
	}
	if due, err := queue.DueProjections(ctx, time.Now(), 10); err != nil || len(due) != 0 {
		t.Fatalf("failed projection retried immediately: %+v %v", due, err)
	}
	due, err := queue.DueProjections(ctx, time.Now().Add(24*time.Hour), 10)
	if err != nil || len(due) != 1 || due[0].RunID != run.ID.String() || due[0].Attempts != 1 {
		t.Fatalf("failed projection lost its durable mark: %+v %v", due, err)
	}
	stored, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || stored.Status != run.Status {
		t.Fatalf("projection changed the run: %+v %v", stored, err)
	}
}

func TestReplayInvocationFacts_RebuildsRetainedAndPreservesPruned(t *testing.T) {
	repos, events, cleanup := testutil.SetupTestRepos(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Date(2026, 7, 29, 19, 0, 0, 0, time.UTC)
	task := &domain.Task{ID: uuid.New(), Title: "fixture", ScopePath: "fixture", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	makeRun := func() *domain.Run {
		return &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, StartedAt: &now, EndedAt: &now, Tag: "fixture", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex}}
	}
	retained, pruned, empty := makeRun(), makeRun(), makeRun()
	if err := repos.Runs.Create(ctx, retained); err != nil {
		t.Fatal(err)
	}
	if err := repos.Runs.Create(ctx, pruned); err != nil {
		t.Fatal(err)
	}
	if err := repos.Runs.Create(ctx, empty); err != nil {
		t.Fatal(err)
	}
	call := domain.NewToolCallEvent(retained.ID, "shell", "call-1", map[string]any{"command": "vrooli help"})
	call.Timestamp = now
	if err := events.Append(ctx, retained.ID, call); err != nil {
		t.Fatal(err)
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithInvocationReadModel(repos.InvocationReadModel), WithClock(func() time.Time { return now }))
	replayed, err := o.ReplayInvocationFacts(ctx, retained.ID)
	if err != nil || replayed.Status != "replayed" || replayed.FactCount != 1 {
		t.Fatalf("retained replay=%+v err=%v", replayed, err)
	}
	emptyResult, err := o.ReplayInvocationFacts(ctx, empty.ID)
	if err != nil || emptyResult.Status != "unreplayable" || emptyResult.ClassifierVersion != "unprojected" {
		t.Fatalf("empty replay=%+v err=%v", emptyResult, err)
	}
	legacy := invocationreadmodel.Fact{InvocationFact: runsignal.InvocationFact{Version: "legacy.v1", CallEventID: "gone", ToolName: "shell", Ownership: "external", Outcome: "success", Fingerprint: "fp", Availability: "available"}, RunID: pruned.ID.String(), OccurredAt: now, TimeBasis: "run_end_derived", ProfileID: "unknown", RunnerType: "unknown", Model: "unknown", Tag: "fixture", RunStatus: "complete"}
	if err := repos.InvocationReadModel.Replace(ctx, []invocationreadmodel.Fact{legacy}, invocationreadmodel.Watermark{RunID: pruned.ID.String(), LastEventID: "gone", LastEventAt: now, ClassifierVersion: "legacy.v1", ProjectedAt: now}); err != nil {
		t.Fatal(err)
	}
	unreplayable, err := o.ReplayInvocationFacts(ctx, pruned.ID)
	if err != nil || unreplayable.Status != "unreplayable" || unreplayable.ClassifierVersion != "legacy.v1" {
		t.Fatalf("pruned replay=%+v err=%v", unreplayable, err)
	}
	facts, err := repos.InvocationReadModel.Facts(ctx, pruned.ID.String())
	if err != nil || len(facts) != 1 || facts[0].Version != "legacy.v1" {
		t.Fatalf("pruned facts=%+v err=%v", facts, err)
	}

	// `usage reindex <run-id>` selects one run of the corpus: a refresh of an
	// unchanged run skips it, a full reindex rebuilds it, and no other run is
	// touched (D-11).
	refreshed, err := o.ReplayInvocationCorpus(ctx, ReplayFilter{RunID: &retained.ID}, true)
	if err != nil || refreshed.Skipped != 1 || refreshed.Replayed+refreshed.Refreshed+refreshed.Unreplayable != 0 {
		t.Fatalf("refresh of one unchanged run=%+v err=%v", refreshed, err)
	}
	full, err := o.ReplayInvocationCorpus(ctx, ReplayFilter{RunID: &retained.ID}, false)
	if err != nil || full.Replayed != 1 || full.Skipped+full.Unreplayable != 0 {
		t.Fatalf("full reindex of one run=%+v err=%v", full, err)
	}
}

// TestReplayInvocationFacts_ProjectsRunsLongerThanOneEventPage guards the
// defect where projection loaded a single capped page of retained events. The
// run total was derived from the same truncated slice, so the conservation
// ledger still reconciled — against a partial run — and the longest, most
// expensive runs were the least accurately measured.
func TestReplayInvocationFacts_ProjectsRunsLongerThanOneEventPage(t *testing.T) {
	repos, events, cleanup := testutil.SetupTestRepos(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Date(2026, 8, 9, 4, 0, 0, 0, time.UTC)
	task := &domain.Task{ID: uuid.New(), Title: "long-run", ScopePath: "fixture", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, StartedAt: &now, EndedAt: &now, Tag: "fixture", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex}}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}

	appendEvent := func(evt *domain.RunEvent) {
		t.Helper()
		evt.Timestamp = now
		if err := events.Append(ctx, run.ID, evt); err != nil {
			t.Fatal(err)
		}
	}
	// One call inside the first page and one past it. A truncating load keeps
	// the leading call and silently drops the trailing one.
	appendEvent(domain.NewToolCallEvent(run.ID, "shell", "call-first", map[string]any{"command": "vrooli help"}))
	for i := 0; i < projectionEventPage; i++ {
		appendEvent(domain.NewLogEvent(run.ID, "info", "filler"))
	}
	appendEvent(domain.NewToolCallEvent(run.ID, "shell", "call-last", map[string]any{"command": "vrooli status"}))

	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithInvocationReadModel(repos.InvocationReadModel), WithClock(func() time.Time { return now }))
	result, err := o.ReplayInvocationFacts(ctx, run.ID)
	if err != nil {
		t.Fatalf("replay err=%v", err)
	}
	if result.FactCount != 2 {
		t.Fatalf("FactCount=%d, want 2: the call past the first page of %d events was dropped", result.FactCount, projectionEventPage)
	}
	facts, err := repos.InvocationReadModel.Facts(ctx, run.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, fact := range facts {
		seen[fact.ToolCallID] = true
	}
	if !seen["call-first"] || !seen["call-last"] {
		t.Fatalf("projected tool calls=%v, want both the leading and trailing call", seen)
	}
}
