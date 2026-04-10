package store

import (
	"context"
	"encoding/json"
	"testing"
)

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
