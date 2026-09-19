package stats

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"swarm-manager/internal/eventlog"
)

func emitWorkshop(t *testing.T, repo *eventlog.SQLiteRepository, entityID string, p eventlog.WorkshopRoundPayload) {
	t.Helper()
	appendEvent(t, repo, time.Now().UTC(), eventlog.EntityBacklogItem, entityID, eventlog.EventWorkshopRoundCompleted, p)
}

func approxEq(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestEngine_RecommendationAcceptanceAggregation(t *testing.T) {
	engine, repo := setupEngine(t)

	// idea/foo: 4 answered, 3 recommended chosen, 0 freeform
	emitWorkshop(t, repo, "idea/foo", eventlog.WorkshopRoundPayload{
		RoundNumber: 1, Kind: "idea",
		ItemsTotal: 4, ItemsAnswered: 4, ItemsRecommendedChosen: 3,
	})
	// fix/bar: 2 answered, 1 recommended, 1 freeform
	emitWorkshop(t, repo, "fix/bar", eventlog.WorkshopRoundPayload{
		RoundNumber: 1, Kind: "fix",
		ItemsTotal: 2, ItemsAnswered: 2, ItemsRecommendedChosen: 1, ItemsFreeformChosen: 1,
	})

	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	a := engine.GetStats().Agent

	// Global: 4+2 = 6 answered, 3+1 = 4 recommended, 1 freeform.
	if a.RecommendationAcceptanceSampleSize != 6 {
		t.Errorf("sample: got %d want 6", a.RecommendationAcceptanceSampleSize)
	}
	if !approxEq(a.RecommendationAcceptanceRate, 4.0/6.0) {
		t.Errorf("acceptance rate: got %v want %v", a.RecommendationAcceptanceRate, 4.0/6.0)
	}
	if !approxEq(a.FreeformOverrideRate, 1.0/6.0) {
		t.Errorf("freeform rate: got %v want %v", a.FreeformOverrideRate, 1.0/6.0)
	}
	idea := a.RecommendationAcceptanceByKind["idea"]
	if idea.SampleSize != 4 || !approxEq(idea.Rate, 0.75) {
		t.Errorf("idea kind: got %+v", idea)
	}
	fix := a.RecommendationAcceptanceByKind["fix"]
	if fix.SampleSize != 2 || !approxEq(fix.Rate, 0.5) {
		t.Errorf("fix kind: got %+v", fix)
	}
}

func TestEngine_LegacyPayloadIgnored(t *testing.T) {
	engine, repo := setupEngine(t)

	// Legacy: only RoundNumber populated. Should not contribute to
	// recommendation counters but workshopRounds should still update.
	emitWorkshop(t, repo, "idea/legacy", eventlog.WorkshopRoundPayload{RoundNumber: 3})

	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	a := engine.GetStats().Agent
	if a.RecommendationAcceptanceSampleSize != 0 {
		t.Errorf("expected zero sample for legacy, got %d", a.RecommendationAcceptanceSampleSize)
	}
	if a.WorkshopRoundsSampleSize != 1 {
		t.Errorf("expected workshop sample 1, got %d", a.WorkshopRoundsSampleSize)
	}
	if a.AvgWorkshopRounds != 3 {
		t.Errorf("expected avg rounds 3, got %v", a.AvgWorkshopRounds)
	}
}

func TestEngine_RecommendationAcceptanceExcludesUnattributedEvents(t *testing.T) {
	engine, repo := setupEngine(t)
	payload, err := json.Marshal(eventlog.WorkshopRoundPayload{RoundNumber: 1, Kind: "idea", ItemsTotal: 4, ItemsAnswered: 4, ItemsRecommendedChosen: 4})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Append(context.Background(), eventlog.Event{Timestamp: time.Now().UTC(), EntityType: eventlog.EntityBacklogItem, EntityID: "idea/unattributed", EventType: eventlog.EventWorkshopRoundCompleted, Metadata: payload}); err != nil {
		t.Fatal(err)
	}
	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := engine.GetStats().Agent.RecommendationAcceptanceSampleSize; got != 0 {
		t.Fatalf("unattributed sample size = %d, want 0", got)
	}
}

func TestEngine_FreeformCountsAsRejection(t *testing.T) {
	engine, repo := setupEngine(t)

	// All freeform: 0% acceptance, 100% override.
	emitWorkshop(t, repo, "execute/foo", eventlog.WorkshopRoundPayload{
		RoundNumber: 1, Kind: "execute",
		ItemsTotal: 3, ItemsAnswered: 3, ItemsFreeformChosen: 3,
	})

	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	a := engine.GetStats().Agent
	if a.RecommendationAcceptanceSampleSize != 3 {
		t.Errorf("sample: got %d want 3", a.RecommendationAcceptanceSampleSize)
	}
	if a.RecommendationAcceptanceRate != 0 {
		t.Errorf("rate: got %v want 0", a.RecommendationAcceptanceRate)
	}
	if a.FreeformOverrideRate != 1 {
		t.Errorf("freeform rate: got %v want 1", a.FreeformOverrideRate)
	}
}

func TestEngine_KindFromEntityIDFallback(t *testing.T) {
	engine, repo := setupEngine(t)
	// Kind field empty — engine should derive from entity ID prefix.
	emitWorkshop(t, repo, "research/x", eventlog.WorkshopRoundPayload{
		RoundNumber: 1,
		ItemsTotal:  2, ItemsAnswered: 2, ItemsRecommendedChosen: 2,
	})

	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	a := engine.GetStats().Agent
	research := a.RecommendationAcceptanceByKind["research"]
	if research.SampleSize != 2 || research.Rate != 1 {
		t.Errorf("research bucket: got %+v", research)
	}
}

func TestEngine_RecommendationAcceptanceByGateUsesAttributedEvidence(t *testing.T) {
	engine, repo := setupEngine(t)
	emitWorkshop(t, repo, "idea/gated", eventlog.WorkshopRoundPayload{
		RoundNumber: 1, GateID: "capture-to-suggested", Kind: "idea",
		ItemsTotal: 4, ItemsAnswered: 4, ItemsRecommendedChosen: 3,
	})
	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := engine.GetStats().Agent.RecommendationAcceptanceByGate["capture-to-suggested"]
	if got.SampleSize != 4 || !approxEq(got.Rate, 0.75) {
		t.Fatalf("gate evidence = %+v, want sample 4 and rate .75", got)
	}
}
