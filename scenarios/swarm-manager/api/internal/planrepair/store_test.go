package planrepair

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
)

type fakeWorkflow struct {
	starts     int
	completion agentmanager.PlanRepairWorkflowCompletion
}

func (f *fakeWorkflow) StartPlanRepair(_ context.Context, _ agentmanager.PlanRepairSnapshot, _ string) (agentmanager.WorkflowStart, error) {
	f.starts++
	return agentmanager.WorkflowStart{ExecutionID: "workflow-1", DefinitionDigest: "sha256:workflow"}, nil
}

func (f *fakeWorkflow) CollectPlanRepair(_ context.Context, _ string) (agentmanager.PlanRepairWorkflowCompletion, error) {
	return f.completion, nil
}

func validRecord() Record {
	return Record{ID: "repair-1", EntityKind: "fix", EntityName: "item", EntityVersion: "v1", PlanReference: "plan-1", FrontierDigest: "sha256:frontier", WorkflowExecution: "workflow-1", WorkflowDigest: "sha256:workflow", ApplyState: ApplyPending}
}

func TestServiceStartIsIdempotentForImmutableFrontier(t *testing.T) {
	workflow := &fakeWorkflow{}
	service := NewService(NewStore(filepath.Join(t.TempDir(), "repairs.json")), workflow)
	req := StartRequest{EntityKind: "fix", EntityName: "item", EntityVersion: "v1", PlanReference: "plan-1", PlanContent: "# plan", FrontierDigest: "sha256:frontier", ValidationFindings: []any{map[string]any{"code": "missing"}}, CheckedAt: "2026-07-17T00:00:00Z", MaxRepairAttempts: 1}
	first, err := service.Start(context.Background(), req)
	if err != nil {
		t.Fatalf("first Start: %v", err)
	}
	second, err := service.Start(context.Background(), req)
	if err != nil {
		t.Fatalf("second Start: %v", err)
	}
	if first.ID != second.ID || workflow.starts != 1 {
		t.Fatalf("starts=%d records=%q/%q", workflow.starts, first.ID, second.ID)
	}
}

func TestStoreRoundTripAndApplyInvariant(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "state", "repairs.json"))
	record := validRecord()
	if err := store.Save(record); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load(record.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != record {
		t.Fatalf("Load = %#v, want %#v", got, record)
	}
	record.ApplyState = ApplyComplete
	if err := store.Save(record); err == nil {
		t.Fatal("Save completed without plan id succeeded")
	}
	if _, err := store.Load("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing error = %v", err)
	}
}

func TestServiceCollectRejectsWrongWorkflowRevision(t *testing.T) {
	workflow := &fakeWorkflow{completion: agentmanager.PlanRepairWorkflowCompletion{
		ExecutionID: "workflow-1", DefinitionDigest: "sha256:other",
		Result: []byte(`{"outcome":"ready","candidatePlan":"# repaired"}`),
	}}
	service := NewService(NewStore(filepath.Join(t.TempDir(), "repairs.json")), workflow)
	req := StartRequest{EntityKind: "fix", EntityName: "item", EntityVersion: "v1", PlanReference: "plan-1", PlanContent: "# plan", FrontierDigest: "sha256:frontier", ValidationFindings: []any{map[string]any{"code": "missing"}}, CheckedAt: "now", MaxRepairAttempts: 1}
	record, err := service.Start(context.Background(), req)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, _, err := service.Collect(context.Background(), record.ID); err == nil {
		t.Fatal("Collect accepted wrong revision")
	}
}
