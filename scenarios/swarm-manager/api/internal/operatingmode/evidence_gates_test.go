package operatingmode

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/evidence"

	_ "modernc.org/sqlite"
)

// [REQ:REQ-P1-011-EVIDENCE-GATES]
func TestEvidenceRequirementStaysPendingThenRecoversFromLateFact(t *testing.T) {
	def := syntheticHarnessDefinition(t)
	assess := def.PhaseGraph.Phases["assess"]
	assess.EvidenceRequirements = []EvidenceRequirement{{
		SubjectKind: "plan", Action: "created", ProducerID: "plan-manager",
		MinConfidence: evidence.ConfidenceAuthoritative,
	}}
	def.PhaseGraph.Phases["assess"] = assess
	withSyntheticModeRegistry(t, def)

	root := t.TempDir()
	agent := &fakeAgent{states: map[string]agentmanager.RunState{}}
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		agent: agent,
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-synthetic": {Name: "init-synthetic", Title: "Synthetic", Description: "evidence gate test", Mode: string(modeSyntheticHarness), Items: []string{"execute/synthetic"}, AcceptanceCriteria: []string{"works"}},
		}},
	})
	evidenceStore := newOperatingModeEvidenceStore(t)
	svc.SetEvidenceService(evidence.NewService(evidenceStore, evidence.RunOwnerResolver{}))

	round := startSyntheticPhase(t, svc, "init-synthetic", "assess")
	completeSyntheticRun(t, agent, round.RunID, `{"operating_mode_result":{"handoff":{"summary":"done"}}}`)
	pending, err := svc.RefreshRound(context.Background(), "init-synthetic", modeSyntheticHarness, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound pending: %v", err)
	}
	if pending.Status != RoundStatusPendingEvidence || !strings.Contains(pending.Error, "producer coverage is incomplete") {
		t.Fatalf("pending round = status %q error %q, want pending evidence with coverage diagnostic", pending.Status, pending.Error)
	}

	if err := evidenceStore.SaveWatermark(context.Background(), evidence.Watermark{ProducerID: "plan-manager", RunID: round.RunID, FactKind: "plan", Coverage: "all plan mutations"}); err != nil {
		t.Fatalf("save terminal watermark: %v", err)
	}
	missing, err := svc.RefreshRound(context.Background(), "init-synthetic", modeSyntheticHarness, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound missing: %v", err)
	}
	if missing.Status != RoundStatusNeedsAttention || !strings.Contains(missing.Error, "terminal coverage") {
		t.Fatalf("missing round = status %q error %q, want definitive missing evidence", missing.Status, missing.Error)
	}

	owner := evidence.Owner{Kind: evidence.OwnerOperatingModeExecution, ID: round.ExecutionID, Round: round.Round}
	if _, err := svc.evidenceService.IngestForOwner(context.Background(), owner, evidence.Observation{
		SourceSystem: "plan-manager", SourceEventID: "plan-created-1", RunID: round.RunID,
		Subject: evidence.Subject{Kind: "plan", ID: "plan-1"}, Action: "created",
		Confidence: evidence.ConfidenceAuthoritative, Verification: evidence.VerificationVerified,
		ObservedAt: time.Date(2026, 7, 12, 20, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("ingest late plan fact: %v", err)
	}
	completed, err := svc.RefreshRound(context.Background(), "init-synthetic", modeSyntheticHarness, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound late fact: %v", err)
	}
	if completed.Status != RoundStatusCompleted || completed.Error != "" {
		t.Fatalf("completed round = status %q error %q, want completed after late fact", completed.Status, completed.Error)
	}
}

func newOperatingModeEvidenceStore(t *testing.T) *evidence.Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open evidence sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := evidence.NewStore(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("init evidence schema: %v", err)
	}
	return store
}
