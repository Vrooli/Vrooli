package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
)

func newRoutedExperimentStore(t *testing.T) *SQLiteExperimentStore {
	t.Helper()
	db, err := database.Open(context.Background(), database.Config{
		Driver: database.DriverSQLite,
		DSN:    filepath.Join(t.TempDir(), "experiments.sqlite"),
	})
	if err != nil {
		t.Fatalf("open routed experiment database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.EnsureSchemas(context.Background(), db.Primary(), database.SchemaProviderFunc(ExperimentSchema)); err != nil {
		t.Fatalf("initialize experiment schema: %v", err)
	}
	return NewSQLiteExperimentStore(db)
}

func TestExperimentStore_CreateAndGet(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	exp := &Experiment{
		ID:         "exp-1",
		SkillID:    "test-skill",
		Name:       "Concise vs Detailed",
		Hypothesis: "Concise prompts produce equal quality with less tokens",
		Arms: []ExperimentArm{
			{VariantID: ControlVariantID, Weight: 0.5},
			{VariantID: "concise-v1", Weight: 0.5},
		},
	}

	if err := es.Create(ctx, exp); err != nil {
		t.Fatalf("create experiment: %v", err)
	}

	got, err := es.Get(ctx, "exp-1")
	if err != nil {
		t.Fatalf("get experiment: %v", err)
	}

	if got.Name != "Concise vs Detailed" {
		t.Errorf("expected name %q, got %q", "Concise vs Detailed", got.Name)
	}
	if got.Kind != KindExperiment {
		t.Errorf("expected kind %q, got %q", KindExperiment, got.Kind)
	}
	if got.Status != ExperimentStatusDraft {
		t.Errorf("expected status %q, got %q", ExperimentStatusDraft, got.Status)
	}
	if len(got.Arms) != 2 {
		t.Errorf("expected 2 arms, got %d", len(got.Arms))
	}
	if got.Revision != 1 {
		t.Errorf("expected revision 1, got %d", got.Revision)
	}
}

func TestExperimentStore_List(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	// Initially empty
	exps, err := es.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(exps) != 0 {
		t.Errorf("expected 0, got %d", len(exps))
	}

	// Create two
	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "E1"}); err != nil {
		t.Fatal(err)
	}
	if err := es.Create(ctx, &Experiment{ID: "exp-2", SkillID: "s2", Name: "E2"}); err != nil {
		t.Fatal(err)
	}

	exps, err = es.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(exps) != 2 {
		t.Errorf("expected 2, got %d", len(exps))
	}
}

func TestExperimentStore_ListBySkill(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "skill-a", Name: "E1"}); err != nil {
		t.Fatal(err)
	}
	if err := es.Create(ctx, &Experiment{ID: "exp-2", SkillID: "skill-b", Name: "E2"}); err != nil {
		t.Fatal(err)
	}
	if err := es.Create(ctx, &Experiment{ID: "exp-3", SkillID: "skill-a", Name: "E3"}); err != nil {
		t.Fatal(err)
	}

	filtered, err := es.ListBySkill(ctx, "skill-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2, got %d", len(filtered))
	}
}

func TestExperimentStore_Update(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "Old"}); err != nil {
		t.Fatal(err)
	}

	updated := &Experiment{
		SkillID: "s1",
		Name:    "New Name",
		Status:  ExperimentStatusRunning,
	}
	if err := es.Update(ctx, "exp-1", updated); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := es.Get(ctx, "exp-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "New Name" {
		t.Errorf("expected %q, got %q", "New Name", got.Name)
	}
	if got.Status != ExperimentStatusRunning {
		t.Errorf("expected status %q, got %q", ExperimentStatusRunning, got.Status)
	}
	if got.Revision != 2 {
		t.Errorf("expected revision 2, got %d", got.Revision)
	}
}

func TestExperimentStore_Delete(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "E1"}); err != nil {
		t.Fatal(err)
	}

	if err := es.Delete(ctx, "exp-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := es.Get(ctx, "exp-1"); err == nil {
		t.Error("expected error getting deleted experiment")
	}
}

func TestExperimentStore_RecordAndListOutcomes(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "E1"}); err != nil {
		t.Fatal(err)
	}

	// Record outcomes
	outcomes := []ExperimentOutcome{
		{VariantID: "control", Source: "swarm-manager", SchemaVersion: 1, RecordedAt: "2026-04-06T14:00:00Z", Data: json.RawMessage(`{"classification":"ready"}`)},
		{VariantID: "concise-v1", Source: "swarm-manager", SchemaVersion: 1, RecordedAt: "2026-04-06T14:01:00Z", Data: json.RawMessage(`{"classification":"needs_work"}`)},
		{VariantID: "control", Source: "swarm-manager", SchemaVersion: 1, RecordedAt: "2026-04-06T14:02:00Z", Data: json.RawMessage(`{"classification":"ready"}`)},
	}
	for _, o := range outcomes {
		if err := es.RecordOutcome(ctx, "exp-1", o); err != nil {
			t.Fatalf("record outcome: %v", err)
		}
	}

	// List outcomes
	got, err := es.ListOutcomes(ctx, "exp-1")
	if err != nil {
		t.Fatalf("list outcomes: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 outcomes, got %d", len(got))
	}
	if got[0].VariantID != "control" {
		t.Errorf("expected first outcome variant %q, got %q", "control", got[0].VariantID)
	}
}

func TestExperimentStore_CountOutcomesByVariant(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "E1"}); err != nil {
		t.Fatal(err)
	}

	outcomes := []ExperimentOutcome{
		{VariantID: "control", Source: "sm", SchemaVersion: 1, RecordedAt: "t1", Data: json.RawMessage(`{}`)},
		{VariantID: "v1", Source: "sm", SchemaVersion: 1, RecordedAt: "t2", Data: json.RawMessage(`{}`)},
		{VariantID: "control", Source: "sm", SchemaVersion: 1, RecordedAt: "t3", Data: json.RawMessage(`{}`)},
		{VariantID: "v1", Source: "sm", SchemaVersion: 1, RecordedAt: "t4", Data: json.RawMessage(`{}`)},
		{VariantID: "v1", Source: "sm", SchemaVersion: 1, RecordedAt: "t5", Data: json.RawMessage(`{}`)},
	}
	for _, o := range outcomes {
		if err := es.RecordOutcome(ctx, "exp-1", o); err != nil {
			t.Fatal(err)
		}
	}

	counts, err := es.CountOutcomesByVariant(ctx, "exp-1")
	if err != nil {
		t.Fatalf("count outcomes: %v", err)
	}
	if counts["control"] != 2 {
		t.Errorf("expected control=2, got %d", counts["control"])
	}
	if counts["v1"] != 3 {
		t.Errorf("expected v1=3, got %d", counts["v1"])
	}
}

func TestExperimentStore_CountOutcomesEmpty(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "E1"}); err != nil {
		t.Fatal(err)
	}

	counts, err := es.CountOutcomesByVariant(ctx, "exp-1")
	if err != nil {
		t.Fatalf("count outcomes: %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("expected empty counts, got %v", counts)
	}
}

func TestExperimentStore_RecordAndListServes(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "E1"}); err != nil {
		t.Fatal(err)
	}

	serves := []ExperimentServe{
		{ExperimentID: "exp-1", SkillID: "s1", VariantID: "control", Source: "agent-manager", ServedAt: "2026-04-06T14:00:00Z"},
		{ExperimentID: "exp-1", SkillID: "s1", VariantID: "concise-v1", Source: "agent-manager"},
		{ExperimentID: "exp-1", SkillID: "s1", VariantID: "control", Source: "agent-manager", ServedAt: "2026-04-06T14:02:00Z"},
	}
	for _, sv := range serves {
		if err := es.RecordServe(ctx, sv); err != nil {
			t.Fatalf("record serve: %v", err)
		}
	}

	got, err := es.ListServes(ctx, "exp-1")
	if err != nil {
		t.Fatalf("list serves: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 serves, got %d", len(got))
	}
	if got[0].VariantID != "control" {
		t.Errorf("expected first serve variant %q, got %q", "control", got[0].VariantID)
	}
	if got[1].ServedAt == "" {
		t.Error("expected RecordServe to backfill servedAt")
	}
}

func TestExperimentStore_ListServesSkipsMalformed(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "E1"}); err != nil {
		t.Fatal(err)
	}

	if err := es.RecordServe(ctx, ExperimentServe{ExperimentID: "exp-1", SkillID: "s1", VariantID: "control"}); err != nil {
		t.Fatal(err)
	}

	// Corrupt the log with a malformed line and a blank line
	path := filepath.Join(storeDir, "experiments", "exp-1", "serve.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{not-json\n\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if err := es.RecordServe(ctx, ExperimentServe{ExperimentID: "exp-1", SkillID: "s1", VariantID: "v1"}); err != nil {
		t.Fatal(err)
	}

	got, err := es.ListServes(ctx, "exp-1")
	if err != nil {
		t.Fatalf("list serves: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 serves (malformed skipped), got %d", len(got))
	}

	counts, err := es.CountServesByVariant(ctx, "exp-1")
	if err != nil {
		t.Fatalf("count serves: %v", err)
	}
	if counts["control"] != 1 || counts["v1"] != 1 {
		t.Errorf("unexpected counts: %v", counts)
	}
}

func TestExperimentStore_CountServesByVariant(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "E1"}); err != nil {
		t.Fatal(err)
	}

	for _, vid := range []string{"control", "v1", "control", "v1", "v1"} {
		if err := es.RecordServe(ctx, ExperimentServe{ExperimentID: "exp-1", SkillID: "s1", VariantID: vid}); err != nil {
			t.Fatal(err)
		}
	}

	counts, err := es.CountServesByVariant(ctx, "exp-1")
	if err != nil {
		t.Fatalf("count serves: %v", err)
	}
	if counts["control"] != 2 {
		t.Errorf("expected control=2, got %d", counts["control"])
	}
	if counts["v1"] != 3 {
		t.Errorf("expected v1=3, got %d", counts["v1"])
	}
}

func TestExperimentStore_ListServesEmpty(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "E1"}); err != nil {
		t.Fatal(err)
	}

	serves, err := es.ListServes(ctx, "exp-1")
	if err != nil {
		t.Fatalf("list serves: %v", err)
	}
	if len(serves) != 0 {
		t.Errorf("expected 0 serves, got %d", len(serves))
	}

	counts, err := es.CountServesByVariant(ctx, "exp-1")
	if err != nil {
		t.Fatalf("count serves: %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("expected empty counts, got %v", counts)
	}
}

func TestExperimentStore_ListServesNotFound(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if _, err := es.ListServes(ctx, "nonexistent"); err == nil {
		t.Error("expected error listing serves for nonexistent experiment")
	}
}

func TestExperimentStore_DuplicateCreate(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "E1"}); err != nil {
		t.Fatal(err)
	}

	err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "E1 Dup"})
	if err == nil {
		t.Error("expected error creating duplicate experiment")
	}
}

func TestExperimentStore_NotFound(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	if _, err := es.Get(ctx, "nonexistent"); err == nil {
		t.Error("expected error for nonexistent experiment")
	}
}

func TestExperimentStore_RecordOutcomeNotFound(t *testing.T) {
	storeDir := t.TempDir()
	es := NewFileExperimentStore(storeDir)
	ctx := context.Background()

	err := es.RecordOutcome(ctx, "nonexistent", ExperimentOutcome{VariantID: "v1", Source: "test", SchemaVersion: 1, RecordedAt: "t1", Data: json.RawMessage(`{}`)})
	if err == nil {
		t.Error("expected error recording outcome for nonexistent experiment")
	}
}

func TestSQLiteExperimentStore_DeduplicatesEvidenceWrites(t *testing.T) {
	ctx := context.Background()
	es := newRoutedExperimentStore(t)
	if err := es.Create(ctx, &Experiment{ID: "exp-1", SkillID: "s1", Name: "test"}); err != nil {
		t.Fatal(err)
	}
	serve := ExperimentServe{ExperimentID: "exp-1", SkillID: "s1", VariantID: "control", Source: "workflow", ServedAt: "2026-01-01T00:00:00Z"}
	if err := es.RecordServe(ctx, serve); err != nil {
		t.Fatal(err)
	}
	if err := es.RecordServe(ctx, serve); err != nil {
		t.Fatal(err)
	}
	serves, err := es.ListServes(ctx, "exp-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(serves) != 1 {
		t.Fatalf("expected one idempotent serve, got %d", len(serves))
	}

	outcome := ExperimentOutcome{VariantID: "control", Source: "evaluator", SchemaVersion: 1, RecordedAt: "2026-01-01T00:01:00Z", Data: json.RawMessage(`{"verdict":"pass"}`)}
	success := true
	outcome.Controlled = &ControlledExperimentOutcome{AssignmentID: "attempt-1", ExecutionID: "execution-1", EvaluatorAttemptID: "attempt-evaluator", EvaluatorRunID: "run-evaluator", Verdict: "pass", Success: &success, OutcomeStatus: "complete", RubricHash: "sha256:rubric", EvaluatorPromptHash: "sha256:prompt", StructuredSchemaHash: "sha256:schema"}
	if err := es.RecordOutcome(ctx, "exp-1", outcome); err != nil {
		t.Fatal(err)
	}
	if err := es.RecordOutcome(ctx, "exp-1", outcome); err != nil {
		t.Fatal(err)
	}
	outcomes, err := es.ListOutcomes(ctx, "exp-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("expected one idempotent outcome, got %d", len(outcomes))
	}
	if outcomes[0].Controlled == nil || outcomes[0].Controlled.AssignmentID != "attempt-1" || outcomes[0].Controlled.Success == nil || !*outcomes[0].Controlled.Success {
		t.Fatalf("controlled outcome projection missing: %#v", outcomes[0])
	}
}

func TestSQLiteExperimentStore_AssignmentSnapshotIsDurableAndIdempotent(t *testing.T) {
	ctx := context.Background()
	es := newRoutedExperimentStore(t)
	assignment := ExperimentAssignment{ExperimentID: "exp-1", SkillID: "s1", VariantID: "treatment", ExecutionID: "execution-1", NodeID: "node-1", AttemptKey: "1", IdempotencyKey: "workflow/execution-1/node/node-1/attempt/1", Content: "immutable prompt", ContentHash: "sha256:one"}
	if err := es.CreateAssignment(ctx, assignment); err != nil {
		t.Fatal(err)
	}
	if err := es.CreateAssignment(ctx, assignment); err == nil {
		t.Fatal("expected duplicate assignment to violate idempotency key")
	}
	got, err := es.GetAssignment(ctx, assignment.ExperimentID, assignment.IdempotencyKey)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != assignment.Content || got.ContentHash != assignment.ContentHash || got.VariantID != assignment.VariantID || got.AssignedAt == "" {
		t.Fatalf("unexpected immutable assignment: %#v", got)
	}
}

func TestSQLiteExperimentStore_PersistsAuditReceipt(t *testing.T) {
	ctx := context.Background()
	es := newRoutedExperimentStore(t)
	receipt := ExperimentAuditReceipt{ExperimentID: "exp-1", ProtocolHash: "sha256:protocol", SampledAssignmentIDs: []string{"a-1"}, FindingsHash: "sha256:findings", ChallengeState: "clear", CompletedAt: "2026-01-01T00:00:00Z", SignatureEnvelope: json.RawMessage(`{"version":"vrooli.receipt-signature.v1","keyId":"key:v1"}`), IdempotencyKey: "audit/exp-1"}
	if err := es.RecordAuditReceipt(ctx, receipt); err != nil {
		t.Fatal(err)
	}
	got, err := es.GetAuditReceipt(ctx, "exp-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ProtocolHash != receipt.ProtocolHash || string(got.SignatureEnvelope) != string(receipt.SignatureEnvelope) || len(got.SampledAssignmentIDs) != 1 {
		t.Fatalf("unexpected audit receipt: %#v", got)
	}
}

func TestSQLiteExperimentStore_RoutesTestModeWritesToLeasedPool(t *testing.T) {
	ctx := context.Background()
	primaryPath := filepath.Join(t.TempDir(), "primary.sqlite")
	testPath := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: primaryPath})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.EnsureSchemas(ctx, db.Primary(), database.SchemaProviderFunc(ExperimentSchema)); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return database.EnsureSchemas(ctx, pool, database.SchemaProviderFunc(ExperimentSchema))
	})
	if err := db.InstallTestPool(ctx, testPath, "experiment-isolation", time.Minute); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.ClearTestPool("experiment-isolation") })

	es := NewSQLiteExperimentStore(db)
	testCtx := database.WithTestMode(ctx)
	if err := es.Create(testCtx, &Experiment{ID: "test-only", SkillID: "skill", Name: "isolated"}); err != nil {
		t.Fatal(err)
	}
	if _, err := es.Get(ctx, "test-only"); err == nil {
		t.Fatal("test-mode experiment write leaked into the primary database")
	}
	if _, err := es.Get(testCtx, "test-only"); err != nil {
		t.Fatalf("test-mode experiment missing from leased pool: %v", err)
	}
}
