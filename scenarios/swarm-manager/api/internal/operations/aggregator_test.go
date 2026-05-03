package operations

import (
	"context"
	"errors"
	"testing"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/operatingmode"
)

// fakeActivityLister returns a fixed slice and records the filters it
// receives so tests can pin window-pushdown semantics.
type fakeActivityLister struct {
	records []agentactivity.Record
	last    agentactivity.ListFilters
	err     error
}

func (f *fakeActivityLister) List(_ context.Context, filters agentactivity.ListFilters) ([]agentactivity.Record, error) {
	f.last = filters
	if f.err != nil {
		return nil, f.err
	}
	out := make([]agentactivity.Record, 0, len(f.records))
	for _, rec := range f.records {
		// Reproduce the polling.matchesFilters time-window pushdown so
		// tests exercise the same shape the real store uses.
		if !filters.ActiveOrFinishedSince.IsZero() {
			if !recordWithinWindowForTest(rec, filters.ActiveOrFinishedSince) {
				continue
			}
		}
		out = append(out, rec)
	}
	return out, nil
}

func recordWithinWindowForTest(rec agentactivity.Record, since time.Time) bool {
	if IsActiveStatus(string(rec.Status)) {
		return true
	}
	if rec.FinishedAt == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, rec.FinishedAt)
	if err != nil {
		return true
	}
	return !t.Before(since)
}

type fakeRoundProjection struct {
	rounds map[string]operatingmode.ActiveRoundSummary
	err    error
}

func (f *fakeRoundProjection) ActiveRoundsByInitiative(_ context.Context) (map[string]operatingmode.ActiveRoundSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[string]operatingmode.ActiveRoundSummary, len(f.rounds))
	for k, v := range f.rounds {
		out[k] = v
	}
	return out, nil
}

type fakeGovernance struct {
	resp *execution.GovernanceStatusResponse
	err  error
}

func (f *fakeGovernance) GovernanceStatus() (*execution.GovernanceStatusResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func defaultGovernance() *execution.GovernanceStatusResponse {
	return &execution.GovernanceStatusResponse{
		Lanes: []execution.LaneStatus{
			{Lane: "investigate", Capacity: 6},
			{Lane: "execute", Capacity: 3, Queue: 2},
			{Lane: "review", Capacity: 8},
			{Lane: "reconcile", Capacity: 2},
		},
		QueueDepth:    2,
		MaxQueueDepth: 50,
	}
}

func fixedNow() func() time.Time {
	t, _ := time.Parse(time.RFC3339, "2026-05-02T15:00:00Z")
	return func() time.Time { return t }
}

func mustAgg(t *testing.T, cfg AggregatorConfig) *Aggregator {
	t.Helper()
	if cfg.Now == nil {
		cfg.Now = fixedNow()
	}
	a, err := NewAggregator(cfg)
	if err != nil {
		t.Fatalf("NewAggregator: %v", err)
	}
	return a
}

func TestAggregate_TimeWindow(t *testing.T) {
	now := fixedNow()()
	withinWindow := now.Add(-1 * time.Hour).Format(time.RFC3339)
	outsideWindow := now.Add(-5 * time.Hour).Format(time.RFC3339)

	records := []agentactivity.Record{
		{
			ActivityID:  "a-active",
			OwnerType:   agentactivity.OwnerBacklog,
			OwnerKind:   "feature",
			OwnerName:   "active-item",
			Purpose:     agentactivity.PurposeProcess,
			Status:      agentactivity.StatusRunning,
			RequestedAt: now.Add(-30 * time.Minute).Format(time.RFC3339),
		},
		{
			ActivityID:  "a-recent",
			OwnerType:   agentactivity.OwnerBacklog,
			OwnerKind:   "feature",
			OwnerName:   "recent",
			Purpose:     agentactivity.PurposeProcess,
			Status:      agentactivity.StatusComplete,
			RequestedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
			FinishedAt:  withinWindow,
		},
		{
			ActivityID:  "a-old",
			OwnerType:   agentactivity.OwnerBacklog,
			OwnerKind:   "feature",
			OwnerName:   "old",
			Purpose:     agentactivity.PurposeProcess,
			Status:      agentactivity.StatusComplete,
			RequestedAt: now.Add(-10 * time.Hour).Format(time.RFC3339),
			FinishedAt:  outsideWindow,
		},
	}

	lister := &fakeActivityLister{records: records}
	agg := mustAgg(t, AggregatorConfig{
		Activities: lister,
		Governance: &fakeGovernance{resp: defaultGovernance()},
	})

	view, err := agg.Aggregate(context.Background(), Filters{Window: 3 * time.Hour})
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}

	since := lister.last.ActiveOrFinishedSince
	wantSince := now.Add(-3 * time.Hour)
	if !since.Equal(wantSince) {
		t.Fatalf("ActiveOrFinishedSince = %v, want %v", since, wantSince)
	}

	if len(view.Activities) != 1 || view.Activities[0].ActivityID != "a-active" {
		t.Fatalf("Activities = %+v, want one [a-active]", view.Activities)
	}
	if len(view.RecentlyFinished) != 1 || view.RecentlyFinished[0].ActivityID != "a-recent" {
		t.Fatalf("RecentlyFinished = %+v, want one [a-recent]", view.RecentlyFinished)
	}
	if view.WindowSeconds != int((3 * time.Hour).Seconds()) {
		t.Fatalf("WindowSeconds = %d, want %d", view.WindowSeconds, int((3 * time.Hour).Seconds()))
	}
	if view.GeneratedAt.IsZero() {
		t.Fatal("GeneratedAt is zero")
	}
}

func TestAggregate_LaneMath(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		// Two investigate-lane runs (workshop default purpose lane).
		{
			ActivityID: "i1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "i1", Purpose: agentactivity.PurposeWorkshop, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
		{
			ActivityID: "i2", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "i2", Purpose: agentactivity.PurposeClarify, Status: agentactivity.StatusStarting,
			RequestedAt: now.Format(time.RFC3339),
		},
		// One execute-lane run.
		{
			ActivityID: "e1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "e1", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
		// One reconcile-lane run via PhaseKind override.
		{
			ActivityID: "r1", OwnerType: agentactivity.OwnerInitiative,
			OwnerName: "auth-rewrite", Purpose: agentactivity.Purpose("holistic_loop_reconcile"),
			PhaseKind: "reconcile", Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
		// Terminal record — should not contribute to live counts.
		{
			ActivityID: "done1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "done1", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusComplete,
			RequestedAt: now.Add(-30 * time.Minute).Format(time.RFC3339),
			FinishedAt:  now.Add(-10 * time.Minute).Format(time.RFC3339),
		},
	}

	agg := mustAgg(t, AggregatorConfig{
		Activities: &fakeActivityLister{records: records},
		Governance: &fakeGovernance{resp: defaultGovernance()},
	})

	view, err := agg.Aggregate(context.Background(), Filters{})
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}

	if got := len(view.Lanes); got != 4 {
		t.Fatalf("Lanes len = %d, want 4", got)
	}
	wantActive := map[string]int{
		"investigate": 2,
		"execute":     1,
		"review":      0,
		"reconcile":   1,
	}
	for _, lane := range view.Lanes {
		if lane.Active != wantActive[lane.Lane] {
			t.Errorf("lane %q active = %d, want %d", lane.Lane, lane.Active, wantActive[lane.Lane])
		}
	}
	// Capacity comes from governance.
	wantCapacity := map[string]int{
		"investigate": 6,
		"execute":     3,
		"review":      8,
		"reconcile":   2,
	}
	for _, lane := range view.Lanes {
		if lane.Capacity != wantCapacity[lane.Lane] {
			t.Errorf("lane %q capacity = %d, want %d", lane.Lane, lane.Capacity, wantCapacity[lane.Lane])
		}
	}
}

func TestAggregate_QueueCounting(t *testing.T) {
	now := fixedNow()()
	gov := &execution.GovernanceStatusResponse{
		Lanes: []execution.LaneStatus{
			{Lane: "investigate", Capacity: 6},
			{Lane: "execute", Capacity: 3, Queue: 5},
			{Lane: "review", Capacity: 8},
			{Lane: "reconcile", Capacity: 2},
		},
		QueueDepth:    5,
		MaxQueueDepth: 50,
	}

	agg := mustAgg(t, AggregatorConfig{
		Activities: &fakeActivityLister{records: []agentactivity.Record{
			{
				ActivityID: "x", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
				OwnerName: "x", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
				RequestedAt: now.Format(time.RFC3339),
			},
		}},
		Governance: &fakeGovernance{resp: gov},
	})

	view, err := agg.Aggregate(context.Background(), Filters{})
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	if view.Queue.Depth != 5 || view.Queue.MaxDepth != 50 {
		t.Fatalf("Queue = %+v, want {5,50}", view.Queue)
	}
	for _, lane := range view.Lanes {
		if lane.Lane == "execute" && lane.Queue != 5 {
			t.Errorf("execute lane queue = %d, want 5", lane.Queue)
		}
		if lane.Lane != "execute" && lane.Queue != 0 {
			t.Errorf("lane %q queue = %d, want 0", lane.Lane, lane.Queue)
		}
	}
}

func TestAggregate_IncludesRecentlyFinished(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "f1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "f1", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusComplete,
			RequestedAt: now.Add(-90 * time.Minute).Format(time.RFC3339),
			FinishedAt:  now.Add(-60 * time.Minute).Format(time.RFC3339),
		},
		{
			ActivityID: "f2", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "f2", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusFailed,
			RequestedAt:   now.Add(-2 * time.Hour).Format(time.RFC3339),
			FinishedAt:    now.Add(-90 * time.Minute).Format(time.RFC3339),
			FailureReason: "boom",
		},
	}

	agg := mustAgg(t, AggregatorConfig{
		Activities: &fakeActivityLister{records: records},
		Governance: &fakeGovernance{resp: defaultGovernance()},
	})

	view, err := agg.Aggregate(context.Background(), Filters{})
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	if len(view.Activities) != 0 {
		t.Errorf("Activities len = %d, want 0", len(view.Activities))
	}
	if len(view.RecentlyFinished) != 2 {
		t.Fatalf("RecentlyFinished len = %d, want 2", len(view.RecentlyFinished))
	}
	// Newest first by FinishedAt.
	if view.RecentlyFinished[0].ActivityID != "f1" {
		t.Errorf("first finished = %q, want f1", view.RecentlyFinished[0].ActivityID)
	}
	if view.RecentlyFinished[1].FailureReason != "boom" {
		t.Errorf("failure reason not propagated: %+v", view.RecentlyFinished[1])
	}
}

func TestAggregate_JoinsRoundsByRunID(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "rnd-1", OwnerType: agentactivity.OwnerInitiative,
			OwnerName: "auth-rewrite", Purpose: agentactivity.Purpose("holistic_loop_review"),
			PhaseKind:   "review",
			Status:      agentactivity.StatusRunning,
			RunID:       "run-abc",
			RequestedAt: now.Format(time.RFC3339),
		},
		{
			ActivityID: "rnd-orphan", OwnerType: agentactivity.OwnerInitiative,
			OwnerName: "mobile-polish", Purpose: agentactivity.Purpose("phased_plan_drain_execute_next"),
			PhaseKind:   "execute",
			Status:      agentactivity.StatusRunning,
			RunID:       "run-orphan",
			RequestedAt: now.Format(time.RFC3339),
		},
	}

	rounds := map[string]operatingmode.ActiveRoundSummary{
		"auth-rewrite": {Mode: "holistic-loop", Phase: "review", Round: 4, Status: "running", RunID: "run-abc"},
	}

	agg := mustAgg(t, AggregatorConfig{
		Activities: &fakeActivityLister{records: records},
		Rounds:     &fakeRoundProjection{rounds: rounds},
		Governance: &fakeGovernance{resp: defaultGovernance()},
	})

	view, err := agg.Aggregate(context.Background(), Filters{})
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	byID := map[string]ActivityRow{}
	for _, row := range view.Activities {
		byID[row.ActivityID] = row
	}
	got := byID["rnd-1"]
	if got.Mode != "holistic-loop" || got.Phase != "review" || got.Round != 4 {
		t.Errorf("rnd-1 round join wrong: %+v", got)
	}
	if got.InitiativeName != "auth-rewrite" {
		t.Errorf("rnd-1 initiative = %q, want auth-rewrite", got.InitiativeName)
	}
	orphan := byID["rnd-orphan"]
	if orphan.Mode != "" || orphan.Round != 0 {
		t.Errorf("rnd-orphan should not carry round join: %+v", orphan)
	}
	if orphan.InitiativeName != "mobile-polish" {
		// Owner-derived initiative still set.
		t.Errorf("rnd-orphan initiative = %q, want mobile-polish", orphan.InitiativeName)
	}
}

func TestAggregate_FilterByLane(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "i1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "i1", Purpose: agentactivity.PurposeWorkshop, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
		{
			ActivityID: "e1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "e1", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
	}
	agg := mustAgg(t, AggregatorConfig{
		Activities: &fakeActivityLister{records: records},
		Governance: &fakeGovernance{resp: defaultGovernance()},
	})

	view, err := agg.Aggregate(context.Background(), Filters{Lanes: []string{"execute"}})
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	if len(view.Activities) != 1 || view.Activities[0].ActivityID != "e1" {
		t.Fatalf("filter-by-lane Activities = %+v, want [e1]", view.Activities)
	}
}

func TestAggregate_FilterByStatusAndQ(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "s1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "alpha-thing", OwnerTitle: "Alpha thing", Purpose: agentactivity.PurposeProcess,
			Status: agentactivity.StatusRunning, RequestedAt: now.Format(time.RFC3339),
		},
		{
			ActivityID: "s2", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "beta-thing", OwnerTitle: "Beta thing", Purpose: agentactivity.PurposeProcess,
			Status: agentactivity.StatusNeedsReview, RequestedAt: now.Format(time.RFC3339),
		},
	}
	agg := mustAgg(t, AggregatorConfig{
		Activities: &fakeActivityLister{records: records},
		Governance: &fakeGovernance{resp: defaultGovernance()},
	})

	view, err := agg.Aggregate(context.Background(), Filters{
		Statuses: []string{"needs_review"},
		Q:        "BETA",
	})
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	if len(view.Activities) != 1 || view.Activities[0].ActivityID != "s2" {
		t.Fatalf("status+q filter Activities = %+v, want [s2]", view.Activities)
	}
}

func TestAggregate_ClampsWindowAboveMax(t *testing.T) {
	now := fixedNow()()
	lister := &fakeActivityLister{records: []agentactivity.Record{}}
	agg := mustAgg(t, AggregatorConfig{
		Activities: lister,
		Governance: &fakeGovernance{resp: defaultGovernance()},
	})
	if _, err := agg.Aggregate(context.Background(), Filters{Window: 100 * time.Hour}); err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	wantSince := now.Add(-MaxWindow)
	if !lister.last.ActiveOrFinishedSince.Equal(wantSince) {
		t.Fatalf("clamp Since = %v, want %v", lister.last.ActiveOrFinishedSince, wantSince)
	}
}

func TestAggregate_ComputesRuntimeForActiveAndFinished(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "active", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "active", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
			RequestedAt: now.Add(-90 * time.Second).Format(time.RFC3339),
			StartedAt:   now.Add(-60 * time.Second).Format(time.RFC3339),
		},
		{
			ActivityID: "done", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "done", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusComplete,
			RequestedAt: now.Add(-60 * time.Minute).Format(time.RFC3339),
			StartedAt:   now.Add(-50 * time.Minute).Format(time.RFC3339),
			FinishedAt:  now.Add(-30 * time.Minute).Format(time.RFC3339),
		},
	}
	agg := mustAgg(t, AggregatorConfig{
		Activities: &fakeActivityLister{records: records},
		Governance: &fakeGovernance{resp: defaultGovernance()},
	})

	view, err := agg.Aggregate(context.Background(), Filters{})
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}

	for _, row := range view.Activities {
		if row.ActivityID == "active" && row.RuntimeSeconds != 60 {
			t.Errorf("active RuntimeSeconds = %d, want 60", row.RuntimeSeconds)
		}
	}
	for _, row := range view.RecentlyFinished {
		if row.ActivityID == "done" && row.RuntimeSeconds != 20*60 {
			t.Errorf("done RuntimeSeconds = %d, want %d", row.RuntimeSeconds, 20*60)
		}
	}
}

func TestAggregate_RuntimePinsToFinishedAtForNeedsReview(t *testing.T) {
	// needs_review is an active status (the agent is waiting on a human),
	// but a record that already carries FinishedAt should pin runtime to
	// that span — otherwise the dashboard counts wall-clock days for
	// records sitting in needs_review.
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "review", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "review", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusNeedsReview,
			RequestedAt: now.Add(-3 * 24 * time.Hour).Format(time.RFC3339),
			StartedAt:   now.Add(-3 * 24 * time.Hour).Format(time.RFC3339),
			FinishedAt:  now.Add(-3*24*time.Hour + 5*time.Second).Format(time.RFC3339),
		},
	}
	agg := mustAgg(t, AggregatorConfig{
		Activities: &fakeActivityLister{records: records},
		Governance: &fakeGovernance{resp: defaultGovernance()},
	})
	view, err := agg.Aggregate(context.Background(), Filters{Window: MaxWindow})
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	for _, row := range view.Activities {
		if row.ActivityID == "review" && row.RuntimeSeconds != 5 {
			t.Errorf("needs_review RuntimeSeconds = %d, want 5 (FinishedAt should win over now)", row.RuntimeSeconds)
		}
	}
}

func TestAggregate_PropagatesActivityListerError(t *testing.T) {
	agg := mustAgg(t, AggregatorConfig{
		Activities: &fakeActivityLister{err: errors.New("boom")},
		Governance: &fakeGovernance{resp: defaultGovernance()},
	})
	if _, err := agg.Aggregate(context.Background(), Filters{}); err == nil {
		t.Fatal("expected error from activity lister")
	}
}

func TestNewAggregator_RequiresActivitiesAndGovernance(t *testing.T) {
	if _, err := NewAggregator(AggregatorConfig{Governance: &fakeGovernance{resp: defaultGovernance()}}); err == nil {
		t.Fatal("expected error when Activities is nil")
	}
	if _, err := NewAggregator(AggregatorConfig{Activities: &fakeActivityLister{}}); err == nil {
		t.Fatal("expected error when Governance is nil")
	}
}
