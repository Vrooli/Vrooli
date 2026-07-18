package integrationstatus

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"swarm-manager/internal/transitions"
)

type fakeChecker struct {
	status Status
	err    error
}

func (f fakeChecker) Check(context.Context) (Status, error) { return f.status, f.err }

func TestPreflightUsesSameStatusTruthAsProjection(t *testing.T) {
	now := time.Now().UTC()
	provider := NewWithClock(map[string]Checker{
		"agent-manager": fakeChecker{status: Status{Configured: true, Availability: Available, CheckedAt: now, FreshUntil: now.Add(time.Minute), DegradedBehavior: "new workflow starts are blocked"}},
		"plan-manager":  fakeChecker{status: Status{Configured: true, Availability: Stale, CheckedAt: now, FreshUntil: now.Add(time.Minute), DegradedBehavior: "plan-gated work is parked", Diagnostic: "last validation is stale"}},
	}, func() time.Time { return now })
	statuses, err := provider.Statuses(context.Background())
	if err != nil {
		t.Fatalf("Statuses: %v", err)
	}
	if len(statuses) != 2 || statuses[1].ID != "plan-manager" || statuses[1].Availability != Stale {
		t.Fatalf("Statuses = %#v", statuses)
	}
	definition := transitions.Definition{Key: "plan.author", Requires: []string{"agent-manager", "plan-manager"}}
	err = provider.Preflight(context.Background(), definition)
	if err == nil || !strings.Contains(err.Error(), "last validation is stale") {
		t.Fatalf("Preflight error = %v, want stale plan-manager diagnosis", err)
	}
}

func TestStatusesDeriveAffectedTransitionsFromRegistry(t *testing.T) {
	registry, err := transitions.LoadFS(fstest.MapFS{
		"transitions.json": &fstest.MapFile{Data: []byte(`[{"schemaVersion":"swarm-transition/v1","key":"plan.author","subject":"backlog item","kind":"workflow","workflow":{"owner":"swarm-manager","key":"swarm-manager/plan-author"},"requires":["agent-manager","plan-manager"],"inputContract":"plan-author/v1","terminalOutcomes":["ready"],"applyAction":"bind-plan"},{"schemaVersion":"swarm-transition/v1","key":"plan.repair","subject":"backlog item","kind":"workflow","workflow":{"owner":"swarm-manager","key":"swarm-manager/plan-repair"},"requires":["agent-manager","plan-manager"],"inputContract":"plan-repair/v1","terminalOutcomes":["ready"],"applyAction":"repair-plan"}]`)},
	}, ".")
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	now := time.Now().UTC()
	provider := NewWithClock(map[string]Checker{
		"agent-manager": fakeChecker{status: Status{Configured: true, Availability: Available, DegradedBehavior: "block", CheckedAt: now, FreshUntil: now.Add(time.Minute)}},
		"plan-manager":  fakeChecker{status: Status{Configured: true, Availability: Available, DegradedBehavior: "block", CheckedAt: now, FreshUntil: now.Add(time.Minute)}},
	}, func() time.Time { return now })
	provider.SetTransitionRegistry(registry)
	statuses, err := provider.Statuses(context.Background())
	if err != nil {
		t.Fatalf("Statuses: %v", err)
	}
	if got, want := statuses[0].AffectedTransitions, []string{"plan.author", "plan.repair"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("affected transitions = %#v, want %#v", got, want)
	}
}

func TestPreflightRejectsUnknownRequirement(t *testing.T) {
	provider := New(nil)
	err := provider.Preflight(context.Background(), transitions.Definition{Key: "x", Requires: []string{"missing"}})
	if err == nil || !strings.Contains(err.Error(), "unknown integration") {
		t.Fatalf("Preflight error = %v", err)
	}
}

func TestPreflightRejectsExpiredFreshness(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	provider := NewWithClock(map[string]Checker{
		"test-genie": fakeChecker{status: Status{Configured: true, Availability: Available, CheckedAt: now.Add(-time.Hour), FreshUntil: now.Add(-time.Second), DegradedBehavior: "validation-dependent completion is parked"}},
	}, func() time.Time { return now })
	err := provider.Preflight(context.Background(), transitions.Definition{Key: "work.fix_and_revalidate", Requires: []string{"test-genie"}})
	if err == nil || !strings.Contains(err.Error(), "available") {
		t.Fatalf("Preflight error = %v, want expired-freshness block", err)
	}
}
