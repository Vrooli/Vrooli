package measures

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	gomeasures "github.com/vrooli/measures-go"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	smmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/measures"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eventlog"
)

// fakeCounter records the last call and returns canned values keyed by event type.
type fakeCounter struct {
	events      map[eventlog.EventType]int
	transitions map[eventlog.EventType]int
	lastFrom    time.Time
	lastTo      time.Time
	all         []eventlog.Event
}

type fakePlanRefStore struct {
	items []backlog.BacklogItem
	err   error
}

func (f *fakePlanRefStore) LoadAll(_ []backlog.BacklogKind) ([]backlog.BacklogItem, error) {
	return f.items, f.err
}

func (f *fakeCounter) CountEventsInRange(_ context.Context, et eventlog.EventType, from, to time.Time) (int, error) {
	f.lastFrom, f.lastTo = from, to
	return f.events[et], nil
}

func (f *fakeCounter) CountStatusTransitionsInRange(_ context.Context, et eventlog.EventType, _ string, from, to time.Time) (int, error) {
	f.lastFrom, f.lastTo = from, to
	return f.transitions[et], nil
}

func (f *fakeCounter) All(_ context.Context) ([]eventlog.Event, error) { return f.all, nil }

// fixedNow is a Tuesday so this_week (Monday 00:00) is a non-trivial range.
func fixedNow() time.Time { return time.Date(2026, 6, 9, 15, 0, 0, 0, time.UTC) }

func TestDeclarations_AllValidFullTier(t *testing.T) {
	decls := Declarations()
	if len(decls) != 15 {
		t.Fatalf("want 15 declarations, got %d", len(decls))
	}
	for _, d := range decls {
		if err := d.Validate(); err != nil {
			t.Errorf("declaration %s invalid: %v", d.Name, err)
		}
		if d.Name != MeasurePlanRefCount && d.Name != MeasureBacklogOpen && d.Name != MeasureBacklogBlocked && d.Name != MeasureGoalMilestoneHealth {
			p, ok := d.Params["window"]
			if !ok || !p.IsCanonical() {
				t.Errorf("declaration %s: window param is not canonical time_window", d.Name)
			}
		}
		if d.Effect != gomeasures.EffectRead || !d.RunEligible {
			t.Errorf("declaration %s must be read + run-eligible", d.Name)
		}
	}
}

func TestMetadataMeasuresShareRegistryComputation(t *testing.T) {
	createdAt := fixedNow().Add(-48 * time.Hour)
	completedAt := fixedNow().Add(-time.Hour)
	completionMetadata, _ := json.Marshal(eventlog.ExecutionCompletedPayload{DurationSeconds: 180})
	statusMetadata, _ := json.Marshal(eventlog.StatusChangePayload{To: string(backlog.StatusCompleted)})
	fc := &fakeCounter{all: []eventlog.Event{
		{EventType: eventlog.EventBacklogCreated, EntityID: "fix/item", Timestamp: createdAt},
		{EventType: eventlog.EventBacklogStatusChanged, EntityID: "fix/item", Timestamp: completedAt, Metadata: statusMetadata},
		{EventType: eventlog.EventExecutionCompleted, EntityID: "execution-1", Timestamp: completedAt, Metadata: completionMetadata},
		{EventType: eventlog.EventExecutionFailed, EntityID: "execution-2", Timestamp: completedAt},
		{EventType: eventlog.EventReviewRoundCompleted, EntityID: "execution-1", Timestamp: completedAt},
	}}
	reg, err := NewRegistry(fc, &fakePlanRefStore{}, fixedNow)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	lead, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{Measure: MeasureBacklogLeadTime, Params: map[string]string{"window": "this_week"}})
	if err != nil || lead.Value != "47" {
		t.Fatalf("lead = %+v, %v", lead, err)
	}
	duration, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{Measure: MeasureExecutionDuration, Params: map[string]string{"window": "this_week"}})
	if err != nil || duration.Value != "3" {
		t.Fatalf("duration = %+v, %v", duration, err)
	}
	reviewRate, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{Measure: MeasureExecutionReviewRate, Params: map[string]string{"window": "this_week"}})
	if err != nil || reviewRate.Value != "0.5" {
		t.Fatalf("review rate = %+v, %v", reviewRate, err)
	}

	h := NewHandler(fc, &fakePlanRefStore{}, fixedNow)
	leadRPC, err := h.BacklogLeadTime(context.Background(), connect.NewRequest(&smmeasuresv1.BacklogLeadTimeRequest{}))
	if err != nil || leadRPC.Msg.GetAverageHours() != 47 || leadRPC.Msg.GetMedianHours() != 47 || leadRPC.Msg.GetSampleSize() != 1 {
		t.Fatalf("lead rpc = %+v, %v", leadRPC, err)
	}
	durationRPC, err := h.ExecutionDuration(context.Background(), connect.NewRequest(&smmeasuresv1.ExecutionDurationRequest{}))
	if err != nil || durationRPC.Msg.GetAverageMinutes() != 3 || durationRPC.Msg.GetMedianMinutes() != 3 || durationRPC.Msg.GetSampleSize() != 1 {
		t.Fatalf("duration rpc = %+v, %v", durationRPC, err)
	}
	reviewRPC, err := h.ExecutionReviewRate(context.Background(), connect.NewRequest(&smmeasuresv1.ExecutionReviewRateRequest{}))
	if err != nil || reviewRPC.Msg.GetRate() != 0.5 || reviewRPC.Msg.GetSampleSize() != 2 {
		t.Fatalf("review rpc = %+v, %v", reviewRPC, err)
	}
}

func TestRegistry_ExecuteBacklogCompleted(t *testing.T) {
	fc := &fakeCounter{transitions: map[eventlog.EventType]int{eventlog.EventBacklogStatusChanged: 7}}
	reg, err := NewRegistry(fc, &fakePlanRefStore{}, fixedNow)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	res, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{
		Measure: MeasureBacklogCompleted,
		Params:  map[string]string{"window": "this_week"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Value != "7" {
		t.Fatalf("want value 7, got %q", res.Value)
	}
	if strings.TrimSpace(res.Provenance.ExecutedQuery) == "" {
		t.Fatal("provenance.executed_query must be set")
	}
	if res.Provenance.ComputedAt.IsZero() {
		t.Fatal("provenance.computed_at must be stamped")
	}
	// this_week on a Tuesday → from is Monday 00:00, before now.
	if !fc.lastFrom.Before(fc.lastTo) || !fc.lastTo.Equal(fixedNow()) {
		t.Fatalf("this_week range looks wrong: [%v, %v)", fc.lastFrom, fc.lastTo)
	}
}

func TestRegistry_DefaultWindowWhenUnset(t *testing.T) {
	fc := &fakeCounter{events: map[eventlog.EventType]int{eventlog.EventBacklogCreated: 3}}
	reg, _ := NewRegistry(fc, &fakePlanRefStore{}, fixedNow)
	res, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{Measure: MeasureBacklogCreated})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Value != "3" {
		t.Fatalf("want 3, got %q", res.Value)
	}
	// Default this_week applied (Monday 00:00 .. now).
	if !fc.lastTo.Equal(fixedNow()) {
		t.Fatalf("default window not applied: to=%v", fc.lastTo)
	}
}

func TestConnectHandler_SharesComputePath(t *testing.T) {
	fc := &fakeCounter{
		transitions: map[eventlog.EventType]int{eventlog.EventBacklogStatusChanged: 11},
		events: map[eventlog.EventType]int{
			eventlog.EventBacklogCreated:      4,
			eventlog.EventExecutionCompleted:  5,
			eventlog.EventGoalCreated:         2,
			eventlog.EventAgentSessionCreated: 9,
		},
	}
	h := NewHandler(fc, &fakePlanRefStore{}, fixedNow)
	ctx := context.Background()

	bc, err := h.CountBacklogCompleted(ctx, connect.NewRequest(&smmeasuresv1.CountBacklogCompletedRequest{}))
	if err != nil {
		t.Fatalf("CountBacklogCompleted: %v", err)
	}
	if bc.Msg.GetCount() != 11 {
		t.Fatalf("want 11, got %d", bc.Msg.GetCount())
	}

	ex, err := h.CountExecutionsCompleted(ctx, connect.NewRequest(&smmeasuresv1.CountExecutionsCompletedRequest{}))
	if err != nil {
		t.Fatalf("CountExecutionsCompleted: %v", err)
	}
	if ex.Msg.GetCount() != 5 {
		t.Fatalf("want 5, got %d", ex.Msg.GetCount())
	}

	as, err := h.CountAgentSessionsCreated(ctx, connect.NewRequest(&smmeasuresv1.CountAgentSessionsCreatedRequest{}))
	if err != nil {
		t.Fatalf("CountAgentSessionsCreated: %v", err)
	}
	if as.Msg.GetCount() != 9 {
		t.Fatalf("want 9, got %d", as.Msg.GetCount())
	}

	net, err := h.CountBacklogNetDelta(ctx, connect.NewRequest(&smmeasuresv1.CountBacklogNetDeltaRequest{}))
	if err != nil {
		t.Fatalf("CountBacklogNetDelta: %v", err)
	}
	if net.Msg.GetCount() != -7 { // 4 created - 11 completed
		t.Fatalf("want -7, got %d", net.Msg.GetCount())
	}
}

func TestConnectHandler_ExplicitWindowResolves(t *testing.T) {
	fc := &fakeCounter{events: map[eventlog.EventType]int{eventlog.EventGoalCreated: 1}}
	h := NewHandler(fc, &fakePlanRefStore{}, fixedNow)
	_, err := h.CountGoalsCreated(context.Background(), connect.NewRequest(&smmeasuresv1.CountGoalsCreatedRequest{
		Window: &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_30D}},
	}))
	if err != nil {
		t.Fatalf("CountGoalsCreated: %v", err)
	}
	// last_30d → from is 30 days before now.
	wantFrom := fixedNow().AddDate(0, 0, -30)
	if fc.lastFrom.Sub(wantFrom).Abs() > time.Second {
		t.Fatalf("last_30d from = %v, want ~%v", fc.lastFrom, wantFrom)
	}
}

func TestAgentSessionProposalRateSharesRegistryComputation(t *testing.T) {
	fc := &fakeCounter{events: map[eventlog.EventType]int{eventlog.EventAgentSessionProposalCreated: 4, eventlog.EventAgentSessionProposalApplied: 3}}
	reg, err := NewRegistry(fc, &fakePlanRefStore{}, fixedNow)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	result, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{Measure: MeasureAgentSessionProposalRate, Params: map[string]string{"window": "this_week"}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Value != "0.75" {
		t.Fatalf("want 0.75, got %q", result.Value)
	}
	h := NewHandler(fc, &fakePlanRefStore{}, fixedNow)
	response, err := h.AgentSessionProposalRate(context.Background(), connect.NewRequest(&smmeasuresv1.AgentSessionProposalRateRequest{}))
	if err != nil {
		t.Fatalf("rpc: %v", err)
	}
	if response.Msg.GetRate() != 0.75 || response.Msg.GetSampleSize() != 4 {
		t.Fatalf("response = %+v", response.Msg)
	}
}

func TestExecutionSuccessRateSharesRegistryComputation(t *testing.T) {
	fc := &fakeCounter{events: map[eventlog.EventType]int{eventlog.EventExecutionCompleted: 6, eventlog.EventExecutionFailed: 2}}
	reg, err := NewRegistry(fc, &fakePlanRefStore{}, fixedNow)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	result, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{Measure: MeasureExecutionSuccessRate, Params: map[string]string{"window": "this_week"}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Value != "0.75" {
		t.Fatalf("want 0.75, got %q", result.Value)
	}
	h := NewHandler(fc, &fakePlanRefStore{}, fixedNow)
	response, err := h.ExecutionSuccessRate(context.Background(), connect.NewRequest(&smmeasuresv1.ExecutionSuccessRateRequest{}))
	if err != nil {
		t.Fatalf("rpc: %v", err)
	}
	if response.Msg.GetRate() != 0.75 || response.Msg.GetSampleSize() != 8 {
		t.Fatalf("response = %+v", response.Msg)
	}
}

func TestPlanRefMeasure_UsesDurableBacklogProjection(t *testing.T) {
	plans := &fakePlanRefStore{items: []backlog.BacklogItem{
		{Name: "without-plan"},
		{Name: "with-plan-a", PlanRef: &backlog.PlanRef{PlanID: "plan-a"}},
		{Name: "with-plan-b", PlanRef: &backlog.PlanRef{PlanID: "plan-b"}},
	}}
	reg, err := NewRegistry(&fakeCounter{}, plans, fixedNow)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	res, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{Measure: MeasurePlanRefCount})
	if err != nil {
		t.Fatalf("execute plan_ref.count: %v", err)
	}
	if res.Value != "2" {
		t.Fatalf("want 2 plan refs, got %q", res.Value)
	}

	h := NewHandler(&fakeCounter{}, plans, fixedNow)
	resp, err := h.CountPlanRefs(context.Background(), connect.NewRequest(&smmeasuresv1.CountPlanRefsRequest{}))
	if err != nil {
		t.Fatalf("CountPlanRefs: %v", err)
	}
	if resp.Msg.GetCount() != 2 {
		t.Fatalf("want 2 plan refs from RPC, got %d", resp.Msg.GetCount())
	}
}

func TestBacklogOpenMeasure_UsesDurableActionableState(t *testing.T) {
	archivedAt := "2026-01-01T00:00:00Z"
	plans := &fakePlanRefStore{items: []backlog.BacklogItem{
		{Name: "open", Status: backlog.StatusReady},
		{Name: "completed", Status: backlog.StatusCompleted},
		{Name: "dropped", Status: backlog.StatusDropped},
		{Name: "archived", Status: backlog.StatusReady, ArchivedAt: &archivedAt},
	}}
	reg, err := NewRegistry(&fakeCounter{}, plans, fixedNow)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	result, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{Measure: MeasureBacklogOpen})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Value != "1" {
		t.Fatalf("want 1, got %q", result.Value)
	}
}

func TestBacklogBlockedMeasure_UsesCanonicalDependencyGraph(t *testing.T) {
	plans := &fakePlanRefStore{items: []backlog.BacklogItem{
		{Name: "dependency", Kind: backlog.KindIdea, Status: backlog.StatusBacklog},
		{Name: "blocked", Kind: backlog.KindFix, Status: backlog.StatusReady, DependsOn: []string{"idea/dependency"}},
		{Name: "ready", Kind: backlog.KindFix, Status: backlog.StatusReady},
	}}
	reg, err := NewRegistry(&fakeCounter{}, plans, fixedNow)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	result, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{Measure: MeasureBacklogBlocked})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Value != "1" {
		t.Fatalf("want 1, got %q", result.Value)
	}
}

func TestGoalMilestoneHealthUsesDurableBacklogProjection(t *testing.T) {
	plans := &fakePlanRefStore{items: []backlog.BacklogItem{
		{Name: "complete", Kind: backlog.KindIdea, Milestone: "m-1", Status: backlog.StatusCompleted},
		{Name: "dependency", Kind: backlog.KindFix, Milestone: "m-1", Status: backlog.StatusBacklog},
		{Name: "blocked", Kind: backlog.KindExecute, Milestone: "m-1", Status: backlog.StatusReady, DependsOn: []string{"fix/dependency"}},
		{Name: "in-flight", Kind: backlog.KindExecute, Milestone: "m-2", Status: backlog.StatusInProgress},
	}}
	reg, err := NewRegistry(&fakeCounter{}, plans, fixedNow)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	result, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{Measure: MeasureGoalMilestoneHealth})
	if err != nil || len(result.Fields) != 2 {
		t.Fatalf("result = %+v, %v", result, err)
	}
	if got := result.Fields[0]; got["milestone"] != "m-1" || got["total"] != "3" || got["completed"] != "1" || got["blocked"] != "1" {
		t.Fatalf("m-1 fields = %#v", got)
	}
	h := NewHandler(&fakeCounter{}, plans, fixedNow)
	response, err := h.GoalMilestoneHealth(context.Background(), connect.NewRequest(&smmeasuresv1.GoalMilestoneHealthRequest{}))
	if err != nil || len(response.Msg.GetMilestones()) != 2 {
		t.Fatalf("rpc = %+v, %v", response, err)
	}
	if got := response.Msg.GetMilestones()[1]; got.GetMilestone() != "m-2" || got.GetInProgress() != 1 {
		t.Fatalf("m-2 rpc = %+v", got)
	}
}
