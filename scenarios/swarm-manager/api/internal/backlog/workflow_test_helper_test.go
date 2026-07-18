package backlog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/testutil"
)

// workflowStartFake is the workflow-only test seam used by backlog tests. It
// replaces the retired operating-mode phase engine and operation runner.
type workflowStartFake struct {
	runID   string
	started bool
	err     error
}

func (f *workflowStartFake) StartWorkshopRound(_ context.Context, _ agentmanager.BacklogWorkshopSnapshot, _ string) (agentmanager.WorkflowStart, error) {
	f.started = true
	if f.err != nil {
		return agentmanager.WorkflowStart{}, f.err
	}
	return agentmanager.WorkflowStart{ExecutionID: "workflow-test", RunID: f.runID, DefinitionDigest: "sha256:test"}, nil
}

func (f *workflowStartFake) CollectWorkshopRound(context.Context, string) (agentmanager.WorkshopWorkflowCompletion, error) {
	return agentmanager.WorkshopWorkflowCompletion{}, nil
}

func (f *workflowStartFake) StartWorkflow(_ context.Context, _ agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	f.started = true
	if f.err != nil {
		return agentmanager.WorkflowStart{}, f.err
	}
	return agentmanager.WorkflowStart{ExecutionID: "workflow-test", RunID: f.runID, DefinitionDigest: "sha256:test"}, nil
}

func (f *workflowStartFake) CollectWorkflow(context.Context, string) (agentmanager.InvocationCompletion, error) {
	return agentmanager.InvocationCompletion{}, nil
}

// setupTestHandlerWithRunner retains the established test setup name while
// returning a workflow start seam; there is no runner in the cutover design.
func setupTestHandlerWithRunner(t *testing.T, runID string) (*Handler, string, *workflowStartFake, struct{}) {
	t.Helper()
	// Creation and auto-advance intentionally start workflows detached from the
	// request. Give those bounded test fakes a chance to persist their pending
	// correlation before TempDir cleanup begins.
	t.Cleanup(func() { time.Sleep(25 * time.Millisecond) })
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	disableAutoWorkshopSettings(t, rootDir)
	h := NewHandlerWithClients(rootDir, pathutil.ResolveScenarioRoot("swarm-manager"), &mockAgentService{}, &promptmanager.MockClient{Result: "test prompt"})
	scopeExecutionQueuerForTest(t, h, rootDir, &mockAgentService{})
	workflow := &workflowStartFake{runID: runID}
	h.SetWorkshopWorkflow(workflow)
	h.SetClarificationWorkflow(workflow)
	return h, rootDir, workflow, struct{}{}
}
