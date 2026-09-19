package validationbroker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"test-genie/internal/testsqlite"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/types/known/durationpb"
)

func validIntent(caller, idempotency string) *validationv1.ValidationIntent {
	return &validationv1.ValidationIntent{
		SchemaVersion:     1,
		IdempotencyKey:    idempotency,
		CallerScenario:    caller,
		CallerExecutionId: "execution-1",
		PlanId:            "plan-1",
		PhaseId:           "phase-1",
		Targets:           []*commonv1.ValidationTarget{{Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, Id: "demo", Root: "scenarios/demo"}},
		Purpose:           validationv1.ValidationPurpose_VALIDATION_PURPOSE_PHASE,
		RequiredStrength:  validationv1.ValidationStrength_VALIDATION_STRENGTH_TARGETED,
		ReusePolicy:       &validationv1.ReusePolicy{Mode: validationv1.ReuseMode_REUSE_MODE_ATTACH_OR_TERMINAL, MaximumAge: durationpb.New(time.Hour)},
		ConcurrencyPolicy: &validationv1.ConcurrencyPolicy{Mode: validationv1.ConcurrencyMode_CONCURRENCY_MODE_SHARED_COMPATIBLE, MaximumParallelism: 2},
		ExpectedIdentity:  &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:abc", Roots: []*validationv1.ContentRootIdentity{{Name: "scenario", Identity: "ri:v1:def"}}},
		ContentInputs:     []*validationv1.ContentInputRoot{{Name: "scenario", Root: "scenarios/demo", Selections: []*validationv1.InputSelection{{Glob: "**", Required: true}}}},
		EvidencePolicy:    &validationv1.EvidencePolicy{RequiredEvidenceKinds: []string{"test-genie-run"}},
		DeadlinePolicy:    &validationv1.DeadlinePolicy{QueueBudget: durationpb.New(time.Minute), ExecutionBudget: durationpb.New(time.Hour), MaximumAttempts: 2},
	}
}

func TestAdmissionIsIdempotentAndRejectsKeyReuseWithDifferentIntent(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	repo := NewRepository(testsqllite(t))
	ctx := context.Background()
	first, err := repo.Admit(ctx, validIntent("plan-manager", "key-1"))
	if err != nil || first.Kind != AdmissionNew {
		t.Fatalf("first admission = %#v err=%v", first, err)
	}
	repeated, err := repo.Admit(ctx, validIntent("plan-manager", "key-1"))
	if err != nil || repeated.Kind != AdmissionIdempotent || repeated.Receipt.GetReceiptId() != first.Receipt.GetReceiptId() {
		t.Fatalf("repeated admission = %#v err=%v", repeated, err)
	}
	changed := validIntent("plan-manager", "key-1")
	changed.RequiredStrength = validationv1.ValidationStrength_VALIDATION_STRENGTH_CERTIFICATION
	if _, err := repo.Admit(ctx, changed); !errors.Is(err, ErrIdempotencyKey) {
		t.Fatalf("changed intent error = %v", err)
	}
}

func TestReceiptCompatibilitySeparatesEvidenceFromGitAttribution(t *testing.T) {
	repo := NewRepository(testsqllite(t))
	ctx := context.Background()
	first := validIntent("caller-a", "first")
	first.BehavioralPrior = "prior-a"
	first.ExpectedIdentity.Commit = "before-commit"
	one, err := repo.Admit(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	first.ExpectedIdentity.Commit = "after-commit"
	first.ExpectedIdentity.Branch = "other-branch"
	first.ExpectedIdentity.Dirty = true
	replay, err := repo.Admit(ctx, first)
	if err != nil || replay.Kind != AdmissionIdempotent {
		t.Fatalf("commit-only replay: %#v %v", replay, err)
	}
	first.CallerScenario, first.IdempotencyKey = "caller-b", "second"
	attached, err := repo.Admit(ctx, first)
	if err != nil || attached.Receipt.GetLineageId() != one.Receipt.GetLineageId() {
		t.Fatalf("commit-only attachment: %#v %v", attached, err)
	}
	first.IdempotencyKey, first.BehavioralPrior = "third", "prior-b"
	different, err := repo.Admit(ctx, first)
	if err != nil || different.Kind != AdmissionNew {
		t.Fatalf("different prior reused: %#v %v", different, err)
	}
}

func TestHistoricalPriorNormalizesToTypedSelection(t *testing.T) {
	intent := validIntent("caller", "legacy")
	intent.CallerAttributes = map[string]string{"baseline_name": "prior-a"}
	normalized, err := normalizeIntent(intent)
	if err != nil || normalized.GetBehavioralPrior() != "prior-a" || normalized.GetCallerAttributes()["baseline_name"] != "" {
		t.Fatalf("normalize: %v %v", normalized, err)
	}
	intent.BehavioralPrior = "prior-b"
	if _, err := normalizeIntent(intent); !errors.Is(err, ErrInvalidIntent) {
		t.Fatalf("conflicting prior: %v", err)
	}
	intent = validIntent("caller", "missing-policy")
	intent.EvidencePolicy = nil
	if _, err := normalizeIntent(intent); !errors.Is(err, ErrInvalidIntent) {
		t.Fatalf("missing policy: %v", err)
	}
}

func TestCompatibleConcurrentIntentsShareOneProducerAndLineage(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	repo := NewRepository(testsqllite(t))
	ctx := context.Background()
	const callers = 12
	results := make(chan Admission, callers)
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			admission, err := repo.Admit(ctx, validIntent("caller-"+string(rune('a'+i)), "key"))
			results <- admission
			errs <- err
		}(i)
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	newCount := 0
	lineage := ""
	for result := range results {
		if result.Kind == AdmissionNew {
			newCount++
		}
		if lineage == "" {
			lineage = result.Receipt.GetLineageId()
		}
		if result.Receipt.GetLineageId() != lineage {
			t.Fatalf("lineage = %q, want %q", result.Receipt.GetLineageId(), lineage)
		}
	}
	if newCount != 1 {
		t.Fatalf("producer admissions = %d, want 1", newCount)
	}
}

func TestNeverReuseIntentsReceiveDistinctProducerKeys(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	repo := NewRepository(testsqllite(t))
	first := validIntent("caller-a", "first")
	first.ReusePolicy.Mode = validationv1.ReuseMode_REUSE_MODE_NEVER
	second := validIntent("caller-b", "second")
	second.ReusePolicy.Mode = validationv1.ReuseMode_REUSE_MODE_NEVER
	one, err := repo.Admit(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	two, err := repo.Admit(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	if one.Kind != AdmissionNew || two.Kind != AdmissionNew || one.Receipt.GetLineageId() == two.Receipt.GetLineageId() {
		t.Fatalf("never-reuse admissions = %#v %#v", one, two)
	}
}

func TestInvalidIntentIsRejectedBeforeDurableAdmission(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	db := testsqllite(t)
	repo := NewRepository(db)
	intent := validIntent("plan-manager", "bad")
	intent.ExpectedIdentity = nil
	if _, err := repo.Admit(context.Background(), intent); !errors.Is(err, ErrInvalidIntent) {
		t.Fatalf("invalid intent error = %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM validation_receipts`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("durable invalid receipts = %d err=%v", count, err)
	}
}

func TestReceiptTransitionsAreLegalDurableAndIdempotent(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	repo := NewRepository(testsqllite(t))
	created, err := repo.Admit(context.Background(), validIntent("plan-manager", "transitions"))
	if err != nil {
		t.Fatal(err)
	}
	queued, err := repo.Transition(context.Background(), created.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_QUEUED, nil)
	if err != nil || queued.GetRevision() != 2 {
		t.Fatalf("queued = %#v err=%v", queued, err)
	}
	running, err := repo.Transition(context.Background(), queued.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, nil)
	if err != nil {
		t.Fatal(err)
	}
	succeeded, err := repo.Transition(context.Background(), running.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, func(receipt *validationv1.ValidationReceipt) error {
		receipt.AchievedStrength = validationv1.ValidationStrength_VALIDATION_STRENGTH_TARGETED
		receipt.Evidence = []*validationv1.EvidenceReference{{EvidenceId: "run-1", Kind: "test-run", Owner: "test-genie"}}
		return nil
	})
	if err != nil || succeeded.GetTerminalAt() == nil {
		t.Fatalf("succeeded = %#v err=%v", succeeded, err)
	}
	repeated, err := repo.Transition(context.Background(), succeeded.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, nil)
	if err != nil || repeated.GetRevision() != succeeded.GetRevision() {
		t.Fatalf("idempotent terminal = %#v err=%v", repeated, err)
	}
	loaded, err := repo.Get(context.Background(), succeeded.GetReceiptId())
	if err != nil || loaded.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || len(loaded.GetEvidence()) != 1 {
		t.Fatalf("loaded = %#v err=%v", loaded, err)
	}
	if _, err := repo.Transition(context.Background(), succeeded.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, nil); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("terminal reversal error = %v", err)
	}
	history, err := repo.History(context.Background(), succeeded.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 4 || history[0].From != validationv1.ReceiptState_RECEIPT_STATE_UNSPECIFIED || history[0].To != validationv1.ReceiptState_RECEIPT_STATE_ADMITTED || history[3].To != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
		t.Fatalf("transition history = %#v", history)
	}
}

func TestTerminalReuseCopiesEvidenceIntoSameLineage(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	repo := NewRepository(testsqllite(t))
	created, err := repo.Admit(context.Background(), validIntent("caller-a", "first"))
	if err != nil {
		t.Fatal(err)
	}
	running, err := repo.Transition(context.Background(), created.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, nil)
	if err != nil {
		t.Fatal(err)
	}
	terminalReceipt, err := repo.Transition(context.Background(), running.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, func(receipt *validationv1.ValidationReceipt) error {
		receipt.AchievedStrength = validationv1.ValidationStrength_VALIDATION_STRENGTH_TARGETED
		receipt.Evidence = []*validationv1.EvidenceReference{{EvidenceId: "run-1", Kind: "test-run", Owner: "test-genie"}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	reused, err := repo.Admit(context.Background(), validIntent("caller-b", "second"))
	if err != nil {
		t.Fatal(err)
	}
	if reused.Kind != AdmissionReusedTerminal || reused.Receipt.GetLineageId() != terminalReceipt.GetLineageId() || len(reused.Receipt.GetEvidence()) != 1 {
		t.Fatalf("terminal reuse = %#v", reused)
	}
}

func testsqllite(t testing.TB) *sql.DB {
	t.Helper()
	return testsqlite.Open(t)
}

func BenchmarkRepositoryAdmissionAndIndexedList(b *testing.B) {
	repo := NewRepository(testsqllite(b))
	ctx := context.Background()
	b.Run("admit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			intent := validIntent("benchmark", fmt.Sprintf("key-%d", i))
			intent.ReusePolicy.Mode = validationv1.ReuseMode_REUSE_MODE_NEVER
			if _, err := repo.Admit(ctx, intent); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("indexed-list", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, _, err := repo.List(ctx, ListFilter{CallerScenario: "benchmark", PlanID: "plan-1", Limit: 50}); err != nil {
				b.Fatal(err)
			}
		}
	})
}
