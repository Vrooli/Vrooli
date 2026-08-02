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

func (s *recordingInvocationReadModel) Aggregate(context.Context, invocationreadmodel.Filter, string, int) ([]invocationreadmodel.AggregateRow, error) {
	return nil, nil
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

func TestFindingRecurrenceRateUsesDurableMeasureAndPreservesUnavailable(t *testing.T) {
	store := &recordingInvocationReadModel{findingMetrics: invocationreadmodel.FindingMetrics{TotalFindings: 4, RecurrenceRate: 0.5}}
	o := New(nil, nil, nil, WithInvocationReadModel(store))
	value, err := o.findingRecurrenceRate(context.Background())
	if err != nil || value == nil || *value != 0.5 {
		t.Fatalf("value=%v err=%v", value, err)
	}
	empty := New(nil, nil, nil, WithInvocationReadModel(&recordingInvocationReadModel{}))
	value, err = empty.findingRecurrenceRate(context.Background())
	if err != nil || value != nil {
		t.Fatalf("empty value=%v err=%v", value, err)
	}
}

func TestProjectTerminalInvocationReadModel_WritesFactsAndWatermarkIdempotently(t *testing.T) {
	repos, events, cleanup := testutil.SetupTestRepos(t)
	defer cleanup()
	store := &recordingInvocationReadModel{calls: make(chan struct{}, 2)}
	o := New(nil, nil, repos.Runs, WithEvents(events), WithInvocationReadModel(store), WithClock(func() time.Time { return time.Date(2026, 7, 29, 18, 30, 0, 0, time.UTC) }))
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex}}
	call := domain.NewToolCallEvent(run.ID, "shell", "call-1", map[string]any{"command": "vrooli help"})
	call.Timestamp = time.Date(2026, 7, 29, 18, 29, 0, 0, time.UTC)
	if err := events.Append(context.Background(), run.ID, call); err != nil {
		t.Fatal(err)
	}
	o.projectTerminalInvocationReadModel(run)
	o.projectTerminalInvocationReadModel(run)
	for range 2 {
		select {
		case <-store.calls:
		case <-time.After(time.Second):
			t.Fatal("projection did not complete")
		}
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.facts) != 1 || store.facts[0].CallEventID != call.ID.String() {
		t.Fatalf("facts=%+v", store.facts)
	}
	if store.watermark == nil || store.watermark.LastEventID != call.ID.String() || store.watermark.ClassifierVersion != runsignal.InvocationFactVersion {
		t.Fatalf("watermark=%+v", store.watermark)
	}
}

func TestProjectTerminalInvocationReadModel_FailureDoesNotMutateTerminalRun(t *testing.T) {
	repos, events, cleanup := testutil.SetupTestRepos(t)
	defer cleanup()
	store := &recordingInvocationReadModel{err: errors.New("projection unavailable"), calls: make(chan struct{}, 1)}
	o := New(nil, nil, repos.Runs, WithEvents(events), WithInvocationReadModel(store))
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted}
	o.projectTerminalInvocationReadModel(run)
	select {
	case <-store.calls:
	case <-time.After(time.Second):
		t.Fatal("projection did not attempt write")
	}
	if run.Status != domain.RunStatusComplete || run.Phase != domain.RunPhaseCompleted {
		t.Fatalf("projection changed terminal run: %+v", run)
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
}
