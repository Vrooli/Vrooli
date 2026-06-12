package measures

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	gomeasures "github.com/vrooli/measures-go"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	smmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/measures"

	"swarm-manager/internal/eventlog"
)

// fakeCounter records the last call and returns canned values keyed by event type.
type fakeCounter struct {
	events      map[eventlog.EventType]int
	transitions map[eventlog.EventType]int
	lastFrom    time.Time
	lastTo      time.Time
}

func (f *fakeCounter) CountEventsInRange(_ context.Context, et eventlog.EventType, from, to time.Time) (int, error) {
	f.lastFrom, f.lastTo = from, to
	return f.events[et], nil
}

func (f *fakeCounter) CountStatusTransitionsInRange(_ context.Context, et eventlog.EventType, _ string, from, to time.Time) (int, error) {
	f.lastFrom, f.lastTo = from, to
	return f.transitions[et], nil
}

// fixedNow is a Tuesday so this_week (Monday 00:00) is a non-trivial range.
func fixedNow() time.Time { return time.Date(2026, 6, 9, 15, 0, 0, 0, time.UTC) }

func TestDeclarations_AllValidFullTier(t *testing.T) {
	decls := Declarations()
	if len(decls) != 5 {
		t.Fatalf("want 5 declarations, got %d", len(decls))
	}
	for _, d := range decls {
		if err := d.Validate(); err != nil {
			t.Errorf("declaration %s invalid: %v", d.Name, err)
		}
		p, ok := d.Params["window"]
		if !ok || !p.IsCanonical() {
			t.Errorf("declaration %s: window param is not canonical time_window", d.Name)
		}
		if d.Effect != gomeasures.EffectRead || !d.RunEligible {
			t.Errorf("declaration %s must be read + run-eligible", d.Name)
		}
	}
}

func TestRegistry_ExecuteBacklogCompleted(t *testing.T) {
	fc := &fakeCounter{transitions: map[eventlog.EventType]int{eventlog.EventBacklogStatusChanged: 7}}
	reg, err := NewRegistry(fc, fixedNow)
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
	reg, _ := NewRegistry(fc, fixedNow)
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
			eventlog.EventInitiativeCreated:   2,
			eventlog.EventAgentSessionCreated: 9,
		},
	}
	h := NewHandler(fc, fixedNow)
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
}

func TestConnectHandler_ExplicitWindowResolves(t *testing.T) {
	fc := &fakeCounter{events: map[eventlog.EventType]int{eventlog.EventInitiativeCreated: 1}}
	h := NewHandler(fc, fixedNow)
	_, err := h.CountInitiativesCreated(context.Background(), connect.NewRequest(&smmeasuresv1.CountInitiativesCreatedRequest{
		Window: &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_30D}},
	}))
	if err != nil {
		t.Fatalf("CountInitiativesCreated: %v", err)
	}
	// last_30d → from is 30 days before now.
	wantFrom := fixedNow().AddDate(0, 0, -30)
	if fc.lastFrom.Sub(wantFrom).Abs() > time.Second {
		t.Fatalf("last_30d from = %v, want ~%v", fc.lastFrom, wantFrom)
	}
}
