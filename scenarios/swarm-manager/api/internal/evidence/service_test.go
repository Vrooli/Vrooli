package evidence

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"swarm-manager/internal/planclient"

	_ "modernc.org/sqlite"
)

type stubOwnerIndex struct {
	owners []Owner
	err    error
	calls  int
}

type fakePlanAuditReader struct {
	facts []planclient.PlanAuditFact
	err   error
}

func (f fakePlanAuditReader) ListAuditFacts(context.Context, string) ([]planclient.PlanAuditFact, error) {
	return f.facts, f.err
}

func (s *stubOwnerIndex) LookupOwners(_ context.Context, _ string) ([]Owner, error) {
	s.calls++
	return s.owners, s.err
}

func newEvidenceService(t *testing.T, sessions, modes *stubOwnerIndex) (*Service, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	return NewService(store, RunOwnerResolver{Sessions: sessions, OperatingModes: modes}), db
}

func verifiedObservation() Observation {
	return Observation{
		SourceSystem: "plan-manager", SourceEventID: "plan-42-created", RunID: "run-42",
		Subject: Subject{Kind: "plan", ID: "plan-42"}, Action: "created",
		Confidence: ConfidenceAuthoritative, Verification: VerificationVerified,
		ContentDigest: "sha256:content", Metadata: map[string]string{"plan_id": "plan-42"},
		ObservedAt: time.Date(2026, 7, 12, 19, 0, 0, 0, time.UTC),
	}
}

// [REQ:REQ-P1-011-CANONICAL-LEDGER]
func TestIngestLinksResolvedEvidenceIdempotently(t *testing.T) {
	sessions := &stubOwnerIndex{owners: []Owner{{Kind: OwnerAgentSession, ID: "session-1"}}}
	modes := &stubOwnerIndex{}
	service, db := newEvidenceService(t, sessions, modes)

	first, err := service.Ingest(context.Background(), verifiedObservation())
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	second, err := service.Ingest(context.Background(), verifiedObservation())
	if err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	if first.Duplicate || !second.Duplicate || first.Owner == nil || second.Owner == nil {
		t.Fatalf("ingest results: first=%+v second=%+v", first, second)
	}
	if sessions.calls != 2 || modes.calls != 2 {
		t.Fatalf("every ingest must query both owner indexes: sessions=%d modes=%d", sessions.calls, modes.calls)
	}
	var observations, links int
	if err := db.QueryRow(`SELECT COUNT(*) FROM evidence_observations`).Scan(&observations); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM evidence_links`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if observations != 1 || links != 1 {
		t.Fatalf("stored observations=%d links=%d", observations, links)
	}
	records, err := service.ListByOwner(context.Background(), Owner{Kind: OwnerAgentSession, ID: "session-1"})
	if err != nil {
		t.Fatalf("list by owner: %v", err)
	}
	if len(records) != 1 || records[0].Observation.Action != "created" || records[0].Observation.Metadata["plan_id"] != "plan-42" {
		t.Fatalf("records = %+v", records)
	}
}

func TestEvidenceQueriesByRunAndEntity(t *testing.T) {
	sessions := &stubOwnerIndex{owners: []Owner{{Kind: OwnerAgentSession, ID: "session-1"}}}
	service, _ := newEvidenceService(t, sessions, &stubOwnerIndex{})
	if _, err := service.Ingest(context.Background(), verifiedObservation()); err != nil {
		t.Fatalf("ingest evidence: %v", err)
	}
	byRun, err := service.ListByRun(context.Background(), "run-42")
	if err != nil || len(byRun) != 1 || byRun[0].Owner.ID != "session-1" {
		t.Fatalf("ListByRun = %+v, %v; want one session record", byRun, err)
	}
	byEntity, err := service.ListByEntity(context.Background(), Subject{Kind: "plan", ID: "plan-42"})
	if err != nil || len(byEntity) != 1 || byEntity[0].Observation.Action != "created" {
		t.Fatalf("ListByEntity = %+v, %v; want plan creation", byEntity, err)
	}
}

// [REQ:REQ-P1-011-OWNER-RECONCILIATION]
func TestIngestLeavesUnresolvedAndAmbiguousOwnershipRetryable(t *testing.T) {
	tests := []struct {
		name     string
		sessions []Owner
		modes    []Owner
		want     OwnershipStatus
	}{
		{name: "unresolved", want: OwnershipUnresolved},
		{name: "ambiguous", sessions: []Owner{{Kind: OwnerAgentSession, ID: "session-1"}}, modes: []Owner{{Kind: OwnerOperatingModeExecution, ID: "execution-1", Round: 2}}, want: OwnershipAmbiguous},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, db := newEvidenceService(t, &stubOwnerIndex{owners: tt.sessions}, &stubOwnerIndex{owners: tt.modes})
			result, err := service.Ingest(context.Background(), verifiedObservation())
			if err != nil {
				t.Fatalf("ingest: %v", err)
			}
			if result.OwnershipStatus != tt.want || result.Owner != nil {
				t.Fatalf("result = %+v", result)
			}
			var links int
			if err := db.QueryRow(`SELECT COUNT(*) FROM evidence_links`).Scan(&links); err != nil {
				t.Fatal(err)
			}
			if links != 0 {
				t.Fatalf("unresolved evidence must not have links: %d", links)
			}
		})
	}
}

func TestIngestRejectsChangedReplayAndStoresProducerProgress(t *testing.T) {
	service, db := newEvidenceService(t, &stubOwnerIndex{}, &stubOwnerIndex{})
	if _, err := service.Ingest(context.Background(), verifiedObservation()); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	changed := verifiedObservation()
	changed.ContentDigest = "sha256:changed"
	if _, err := service.Ingest(context.Background(), changed); !errors.Is(err, ErrSourceConflict) {
		t.Fatalf("changed replay error = %v", err)
	}
	store := NewStore(db)
	if err := store.SaveCheckpoint(context.Background(), Checkpoint{ProducerID: "plan-manager", RunID: "run-42", FactKind: "plan", Cursor: "42"}); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	if err := store.SaveWatermark(context.Background(), Watermark{ProducerID: "plan-manager", RunID: "run-42", FactKind: "plan", Coverage: "all plan mutations through cursor 42"}); err != nil {
		t.Fatalf("save watermark: %v", err)
	}
	var checkpoints, watermarks int
	if err := db.QueryRow(`SELECT COUNT(*) FROM evidence_checkpoints`).Scan(&checkpoints); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM evidence_watermarks`).Scan(&watermarks); err != nil {
		t.Fatal(err)
	}
	if checkpoints != 1 || watermarks != 1 {
		t.Fatalf("checkpoints=%d watermarks=%d", checkpoints, watermarks)
	}
}

func TestIngestForOwnerBatchIsAtomicOnValidationFailure(t *testing.T) {
	service, db := newEvidenceService(t, &stubOwnerIndex{}, &stubOwnerIndex{})
	owner := Owner{Kind: OwnerAgentSession, ID: "session-1"}
	valid := verifiedObservation()
	valid.SourceEventID = "valid"
	invalid := verifiedObservation()
	invalid.SourceEventID = "invalid"
	invalid.Subject.ID = ""
	if _, err := service.IngestForOwnerBatch(context.Background(), owner, []Observation{valid, invalid}); err == nil {
		t.Fatal("IngestForOwnerBatch accepted an invalid observation")
	}
	var observations, links int
	if err := db.QueryRow(`SELECT COUNT(*) FROM evidence_observations`).Scan(&observations); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM evidence_links`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if observations != 0 || links != 0 {
		t.Fatalf("partial batch persisted observations=%d links=%d", observations, links)
	}
}

func TestReconcilePlanManagerCreatesAuthoritativeLinkedEvidence(t *testing.T) {
	service, db := newEvidenceService(t, &stubOwnerIndex{owners: []Owner{{Kind: OwnerOperatingModeExecution, ID: "execution-1", Round: 2}}}, &stubOwnerIndex{})
	reader := fakePlanAuditReader{facts: []planclient.PlanAuditFact{{
		EventID: "plan-1:plan.created:1", RunID: "run-42", TaskID: "task-42", Action: "plan.created", PlanID: "plan-1", ContentDigest: "sha256:plan", OccurredAt: time.Date(2026, 7, 12, 20, 0, 0, 0, time.UTC),
	}}}
	results, err := service.ReconcilePlanManager(context.Background(), reader, "run-42")
	if err != nil {
		t.Fatalf("reconcile plan manager: %v", err)
	}
	if len(results) != 1 || results[0].Owner == nil || results[0].Owner.ID != "execution-1" {
		t.Fatalf("results = %+v", results)
	}
	var confidence, checkpoint string
	if err := db.QueryRow(`SELECT confidence FROM evidence_observations`).Scan(&confidence); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT cursor FROM evidence_checkpoints WHERE producer_id='plan-manager'`).Scan(&checkpoint); err != nil {
		t.Fatal(err)
	}
	if confidence != string(ConfidenceAuthoritative) || checkpoint != "plan-1:plan.created:1" {
		t.Fatalf("confidence=%q checkpoint=%q", confidence, checkpoint)
	}
	if complete, err := NewStore(db).HasTerminalWatermark(context.Background(), "plan-manager", "run-42", "plan"); err != nil || !complete {
		t.Fatalf("plan-manager terminal watermark = %v, %v; want true after committed reconciliation", complete, err)
	}
}

// [REQ:REQ-P1-011-EVIDENCE-GATES]
func TestEvaluateRequirementDefersAbsenceUntilProducerWatermark(t *testing.T) {
	service, db := newEvidenceService(t, &stubOwnerIndex{owners: []Owner{{Kind: OwnerAgentSession, ID: "session-1"}}}, &stubOwnerIndex{})
	requirement := Requirement{SubjectKind: "plan", Action: "plan.created", ProducerID: "plan-manager", MinConfidence: ConfidenceAuthoritative}
	pending, err := service.EvaluateRequirement(context.Background(), Owner{Kind: OwnerAgentSession, ID: "session-1"}, "run-42", requirement)
	if err != nil || pending.State != RequirementPending {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}
	if err := NewStore(db).SaveWatermark(context.Background(), Watermark{ProducerID: "plan-manager", RunID: "run-42", FactKind: "plan", Coverage: "terminal"}); err != nil {
		t.Fatal(err)
	}
	missing, err := service.EvaluateRequirement(context.Background(), Owner{Kind: OwnerAgentSession, ID: "session-1"}, "run-42", requirement)
	if err != nil || missing.State != RequirementMissing {
		t.Fatalf("missing=%+v err=%v", missing, err)
	}
}

func TestEvaluateRequirementDoesNotAcceptUnverifiedObservation(t *testing.T) {
	service, _ := newEvidenceService(t, &stubOwnerIndex{}, &stubOwnerIndex{})
	owner := Owner{Kind: OwnerAgentSession, ID: "session-1"}
	observation := verifiedObservation()
	observation.Verification = VerificationUnverified
	if _, err := service.IngestForOwner(context.Background(), owner, observation); err != nil {
		t.Fatalf("ingest unverified observation: %v", err)
	}
	result, err := service.EvaluateRequirement(context.Background(), owner, "run-42", Requirement{SubjectKind: "plan", Action: "created", ProducerID: "plan-manager", MinConfidence: ConfidenceAuthoritative})
	if err != nil || result.State != RequirementPending {
		t.Fatalf("result=%+v err=%v, want pending instead of satisfying unverified evidence", result, err)
	}
}

func TestRecordOperatorVerifiedRequiresAttributionAndAppendsEvidence(t *testing.T) {
	service, _ := newEvidenceService(t, &stubOwnerIndex{}, &stubOwnerIndex{})
	owner := Owner{Kind: OwnerAgentSession, ID: "session-1"}
	if _, err := service.RecordOperatorVerified(context.Background(), owner, "operator-1", "run-42", Subject{Kind: "plan", ID: "plan-42"}, "created", "", "checked audit trail", nil); err == nil {
		t.Fatal("RecordOperatorVerified accepted a missing actor")
	}
	result, err := service.RecordOperatorVerified(context.Background(), owner, "operator-1", "run-42", Subject{Kind: "plan", ID: "plan-42"}, "created", "matt", "checked audit trail", map[string]string{"plan_id": "plan-42"})
	if err != nil || result.Owner == nil || result.Owner.ID != "session-1" {
		t.Fatalf("RecordOperatorVerified = %+v, %v", result, err)
	}
	records, err := service.ListByOwner(context.Background(), owner)
	if err != nil || len(records) != 1 {
		t.Fatalf("ListByOwner = %+v, %v", records, err)
	}
	observation := records[0].Observation
	if observation.Confidence != ConfidenceOperator || observation.Verification != VerificationVerified || observation.Metadata["operator_actor"] != "matt" || observation.Metadata["operator_reason"] != "checked audit trail" {
		t.Fatalf("operator observation = %+v", observation)
	}
}
