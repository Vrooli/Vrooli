package execution

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

type stubSpecSyncArchiver struct{ calls int }

func (s *stubSpecSyncArchiver) ArchiveScenario(context.Context, ArchiveContext) error {
	s.calls++
	return nil
}

func TestScenarioSpecSyncWorkflow_QueuesAndAppliesExactlyOnce(t *testing.T) {
	root := t.TempDir()
	scenarioPath := filepath.Join(root, "scenario-to-archive")
	if err := os.MkdirAll(scenarioPath, 0o755); err != nil {
		t.Fatal(err)
	}
	workflow := &stubConclusionWorkflow{start: agentmanager.WorkflowStart{ExecutionID: "spec-sync-1", RunID: "run-spec", DefinitionDigest: "sha256:spec"}}
	archiver := &stubSpecSyncArchiver{}
	service := NewService(ServiceConfig{DataRoot: root, StorePath: filepath.Join(root, "runs.json"), AgentService: &stubAgentService{}, SpecSyncWorkflow: workflow, Archiver: archiver})
	started, err := service.QueueSpecSyncArchive(context.Background(), ArchiveContext{ScenarioName: "scenario-to-archive", ScenarioPath: scenarioPath})
	if err != nil {
		t.Fatal(err)
	}
	if started.AgentWorkflowKey != "swarm-manager/scenario-spec-sync" || started.OpExecutionID != "" {
		t.Fatalf("expected workflow-backed spec sync, got %#v", started)
	}
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "complete", "summary": "synced"}})
	if err != nil {
		t.Fatal(err)
	}
	workflow.completion = agentmanager.InvocationCompletion{ExecutionID: started.AgentWorkflowExecutionID, DefinitionDigest: started.AgentWorkflowDefinition, Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Input: workflow.invocation.Input, Output: output}
	first, err := service.ApplySpecSyncWorkflow(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Idempotent || first.Record.Status != StatusCompleted || first.Record.AgentWorkflowApplyState != workflowApplyComplete {
		t.Fatalf("unexpected first apply: %#v", first)
	}
	if archiver.calls != 1 {
		t.Fatalf("archive calls = %d, want 1", archiver.calls)
	}
	if _, err := os.Stat(scenarioPath); !os.IsNotExist(err) {
		t.Fatalf("scenario directory should be removed after archive, stat err=%v", err)
	}
	second, err := service.ApplySpecSyncWorkflow(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Idempotent || workflow.collectCalls != 1 || archiver.calls != 1 {
		t.Fatalf("expected idempotent replay, got %#v collects=%d archives=%d", second, workflow.collectCalls, archiver.calls)
	}
}
