package execution

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/handoff"
	"swarm-manager/internal/planclient"
	"swarm-manager/internal/promptmanager"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	"google.golang.org/protobuf/types/known/structpb"
)

// stubPolicyProvider implements PolicyProvider for tests.
type stubPolicyProvider struct {
	policy Policy
}

func (s *stubPolicyProvider) LoadPolicy() (Policy, error) {
	return s.policy, nil
}

type stubAgentService struct {
	spawnCalls int
	spawnErr   error
}

type stubPhasedPlanWorkflow struct {
	start        agentmanager.WorkflowStart
	startErr     error
	invocation   agentmanager.Invocation
	startCalls   int
	completion   agentmanager.InvocationCompletion
	collectErr   error
	collectCalls int
	approveCalls int
	cancelCalls  int
	progress     agentmanager.WorkflowProgress
	progressErr  error
}

type stubConclusionWorkflow struct {
	start        agentmanager.WorkflowStart
	startErr     error
	invocation   agentmanager.Invocation
	startCalls   int
	completion   agentmanager.InvocationCompletion
	collectErr   error
	collectCalls int
}

func (s *stubConclusionWorkflow) StartWorkflow(_ context.Context, invocation agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	s.startCalls++
	s.invocation = invocation
	if s.startErr != nil {
		return agentmanager.WorkflowStart{}, s.startErr
	}
	if s.start.ExecutionID == "" {
		return agentmanager.WorkflowStart{ExecutionID: "conclusion-1", RunID: "run-c", DefinitionDigest: "sha256:conclusion"}, nil
	}
	return s.start, nil
}

func (s *stubConclusionWorkflow) CollectWorkflow(context.Context, string) (agentmanager.InvocationCompletion, error) {
	s.collectCalls++
	return s.completion, s.collectErr
}

func (s *stubPhasedPlanWorkflow) StartWorkflow(_ context.Context, invocation agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	s.startCalls++
	s.invocation = invocation
	if s.startErr != nil {
		return agentmanager.WorkflowStart{}, s.startErr
	}
	if s.start.ExecutionID == "" && s.start.DefinitionDigest == "" && s.start.RunID == "" {
		return agentmanager.WorkflowStart{ExecutionID: "wfx-1", RunID: "run-1", DefinitionDigest: "sha256:def"}, nil
	}
	return s.start, nil
}

func (s *stubPhasedPlanWorkflow) CollectWorkflow(_ context.Context, _ string) (agentmanager.InvocationCompletion, error) {
	s.collectCalls++
	return s.completion, s.collectErr
}

func (s *stubPhasedPlanWorkflow) GetWorkflowProgress(_ context.Context, _ string) (agentmanager.WorkflowProgress, error) {
	return s.progress, s.progressErr
}

func TestWorkflowProgressReadsAgentManagerTraceWithoutPersistingIt(t *testing.T) {
	root := t.TempDir()
	workflow := &stubPhasedPlanWorkflow{progress: agentmanager.WorkflowProgress{CurrentNode: "slice", SliceCount: 2, Turns: 7, CostUSD: 0.42, EdgeTraversals: map[string]int32{"slice->review": 2}, UpdatedAt: "2026-07-22T12:00:00Z"}}
	service := NewService(ServiceConfig{DataRoot: root, StorePath: filepath.Join(root, "executions.json"), PlanRenderer: testPlanRenderer(), PhasedPlanWorkflow: workflow})
	if err := service.store.Save([]Record{{ExecutionID: "execution-1", AgentWorkflowExecutionID: "workflow-1", BacklogKind: "execute", BacklogName: "item-a", Status: StatusRunning, Mode: ModeYOLO}}); err != nil {
		t.Fatal(err)
	}
	progress, err := service.WorkflowProgress(context.Background(), "execution-1")
	if err != nil {
		t.Fatal(err)
	}
	if progress.CurrentNode != "slice" || progress.SliceCount != 2 || progress.Turns != 7 {
		t.Fatalf("progress=%+v", progress)
	}
	stored, err := service.Get(context.Background(), "execution-1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.AgentWorkflowExecutionID != "workflow-1" {
		t.Fatalf("live progress mutated execution correlation: %+v", stored)
	}
}

func (s *stubPhasedPlanWorkflow) SignalWorkflow(context.Context, string, string, *structpb.Value, string) error {
	s.approveCalls++
	return nil
}

func (s *stubPhasedPlanWorkflow) CancelWorkflow(context.Context, string, string, string) error {
	s.cancelCalls++
	return nil
}

func (s *stubAgentService) IsEnabled() bool { return true }

// stubOperationStarter fakes the historical operation reaper seam.
type stubOperationStarter struct {
	cancelCalls int
	cancelReq   OperationCancelRequest
}

func (s *stubOperationStarter) CancelOperation(_ context.Context, req OperationCancelRequest) error {
	s.cancelCalls++
	s.cancelReq = req
	return nil
}

type snapshotAgentService struct {
	stubAgentService
	runStateCalls int
}

func testPlanRenderer() *fakeMarkdownRenderer {
	return &fakeMarkdownRenderer{result: planclient.RenderMarkdownResult{
		Markdown: "# Rendered implementation plan\n\nTest plan content.", QualityStatus: "pass",
		Plan: &sharedv1.Plan{ContentHash: "sha256:test-plan"},
	}}
}

func (s *snapshotAgentService) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	s.runStateCalls++
	return agentmanager.RunState{Status: "completed", FinishedAt: "2026-05-14T00:00:00Z"}, nil
}

func TestListSnapshotDoesNotProcessActiveExecutions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	agent := &snapshotAgentService{}
	svc := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		AgentService: agent,
	})
	if err := svc.store.Save([]Record{{
		ExecutionID: "exec-1",
		BacklogKind: "execute",
		BacklogName: "slow-graph",
		Status:      StatusRunning,
		RunID:       "run-1",
		CreatedAt:   "2026-05-14T00:00:00Z",
	}}); err != nil {
		t.Fatalf("save executions: %v", err)
	}

	records, err := svc.ListSnapshot(context.Background(), ListFilters{})
	if err != nil {
		t.Fatalf("ListSnapshot: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListSnapshot returned %d records, want 1", len(records))
	}
	if records[0].Status != StatusRunning {
		t.Fatalf("snapshot status = %q, want persisted running", records[0].Status)
	}
	if agent.runStateCalls != 0 {
		t.Fatalf("ListSnapshot called GetRunState %d times, want 0", agent.runStateCalls)
	}
}

func TestQueueAndStartManualExecution(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "test-idea", map[string]any{
		"name":        "test-idea",
		"title":       "Test",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "test-idea")

	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})
	workflow := &stubPhasedPlanWorkflow{}
	service.SetPhasedPlanWorkflow(workflow)

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "test-idea",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	if record.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", record.Status)
	}
	if strings.TrimSpace(record.QueuedAt) == "" {
		t.Fatal("expected QueuedAt to be set on enqueue")
	}
	queuedAt := record.QueuedAt

	storedItem := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "test-idea", "spec.json"))
	if storedItem["status"] != "queued" {
		t.Fatalf("expected backlog status queued, got %#v", storedItem["status"])
	}

	started, err := service.Start(context.Background(), record.ExecutionID)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if started.Status != StatusStarting {
		t.Fatalf("expected starting status, got %s", started.Status)
	}
	if started.QueuedAt != queuedAt {
		t.Fatalf("expected QueuedAt preserved through start: got %q want %q", started.QueuedAt, queuedAt)
	}
	// A plan-backed item starts the bounded data-defined workflow. The consumer
	// keeps the workflow execution and immutable frontier correlations.
	if started.RunID != "run-1" {
		t.Fatalf("expected run id from operation start, got %q", started.RunID)
	}
	if started.AgentWorkflowExecutionID != "wfx-1" || started.TaskID != "wfx-1" {
		t.Fatalf("expected workflow execution refs on record, got workflow=%q task=%q", started.AgentWorkflowExecutionID, started.TaskID)
	}
	if agent.spawnCalls != 0 {
		t.Fatalf("expected no direct spawn for a plan-backed item, got %d", agent.spawnCalls)
	}
	if workflow.startCalls != 1 {
		t.Fatalf("expected 1 workflow start, got %d", workflow.startCalls)
	}
	input, _ := workflow.invocation.Input.AsInterface().(map[string]any)
	plan, _ := input["plan"].(map[string]any)
	consumer, _ := input["consumer"].(map[string]any)
	if plan["reference"] != "test-plan-test-idea" {
		t.Fatalf("expected plan handle from the item plan_ref, got %#v", plan["reference"])
	}
	if plan["frontierDigest"] == "" || consumer["entityVersion"] == "" {
		t.Fatal("expected immutable workflow frontier correlations")
	}
}

// TestQueueAndStartManualExecution_ResearchRequiresPlan proves a
func TestQueueAndStartManualExecution_ResearchRequiresCanonicalPlan(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "research", "test-research", map[string]any{
		"name":        "test-research",
		"title":       "Research Test",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		AgentService: &stubAgentService{},
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})
	_, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "research",
		BacklogName: "test-research",
		Mode:        ModeManual,
	})
	if err == nil || !strings.Contains(err.Error(), "plan_ref") {
		t.Fatalf("QueueBacklog error = %v, want canonical plan guard", err)
	}
}

func TestQueueBacklog_UsesPolicyDefaultsWhenModeMissing(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "policy-idea", map[string]any{
		"name":        "policy-idea",
		"title":       "Policy Idea",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "policy-idea")

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		PolicyProvider: &stubPolicyProvider{policy: Policy{
			DefaultMode: ModeManual,
		}},
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "policy-idea",
		Mode:        "",
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	if record.Mode != ModeManual {
		t.Fatalf("expected manual mode from policy, got %s", record.Mode)
	}
	if record.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", record.Status)
	}
}

func TestQueueBacklog_AllowsArchivedIdeas(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "archived-idea", map[string]any{
		"name":        "archived-idea",
		"title":       "Archived Idea",
		"description": "desc",
		"status":      "ready",
		"priority":    3,
		"tags":        []string{},
		"archived_at": "2025-01-01T00:00:00Z",
	})
	mustWriteDeliverableFile(t, root, "idea", "archived-idea")

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "archived-idea",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	if record.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", record.Status)
	}
}

func TestQueueBacklog_YOLORollsBackWhenSpawnFails(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "rollback-idea", map[string]any{
		"name":        "rollback-idea",
		"title":       "Rollback Idea",
		"description": "desc",
		"status":      "ready",
		"priority":    3,
		"tags":        []string{},
		"archived_at": "2025-01-01T00:00:00Z",
	})
	mustWriteDeliverableFile(t, root, "idea", "rollback-idea")

	workflow := &stubPhasedPlanWorkflow{startErr: errors.New("workflow start failed")}
	service := NewService(ServiceConfig{
		DataRoot:           root,
		StorePath:          filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer:       testPlanRenderer(),
		PhasedPlanWorkflow: workflow,
		PromptClient:       &promptmanager.MockClient{Result: "test prompt"},
	})

	_, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "rollback-idea",
		Mode:        ModeYOLO,
	})
	if err == nil {
		t.Fatal("expected queue error when spawn fails")
	}

	storedItem := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "rollback-idea", "spec.json"))
	if storedItem["status"] != "ready" {
		t.Fatalf("expected ready status restored, got %#v", storedItem["status"])
	}

	records := mustLoadRecords(t, filepath.Join(root, ".vrooli", "execution-runs.json"))
	if len(records) != 0 {
		t.Fatalf("expected rollback to remove execution record, got %d", len(records))
	}
}

func TestCancel_RestoresArchivedIdeaStatus(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "archived-cancel", map[string]any{
		"name":          "archived-cancel",
		"title":         "Archived Cancel",
		"description":   "desc",
		"status":        "ready",
		"priority":      3,
		"tags":          []string{},
		"archived_at":   "2025-01-01T00:00:00Z",
		"archiveReason": "scenario deleted with archive=true",
	})
	mustWriteDeliverableFile(t, root, "idea", "archived-cancel")

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "archived-cancel",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}

	_, err = service.Cancel(context.Background(), record.ExecutionID)
	if err != nil {
		t.Fatalf("Cancel error: %v", err)
	}

	storedItem := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "archived-cancel", "spec.json"))
	if storedItem["status"] != "ready" {
		t.Fatalf("expected ready status after cancel, got %#v", storedItem["status"])
	}
	if storedItem["archived_at"] != "2025-01-01T00:00:00Z" {
		t.Fatalf("expected archived_at preserved, got %#v", storedItem["archived_at"])
	}
	if storedItem["archiveReason"] != "scenario deleted with archive=true" {
		t.Fatalf("expected archive metadata preserved, got %#v", storedItem["archiveReason"])
	}
}

func TestCancel_RestoresArchivedStatusAfterForcedQueue(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "archived-cancel-forced", map[string]any{
		"name":          "archived-cancel-forced",
		"title":         "Archived Cancel Forced",
		"description":   "desc",
		"status":        "ready",
		"priority":      3,
		"tags":          []string{},
		"archived_at":   "2025-01-01T00:00:00Z",
		"archiveReason": "scenario deleted with archive=true",
	})
	mustWriteDeliverableFile(t, root, "idea", "archived-cancel-forced")

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "archived-cancel-forced",
		Mode:        ModeManual,
		Force:       true,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}

	_, err = service.Cancel(context.Background(), record.ExecutionID)
	if err != nil {
		t.Fatalf("Cancel error: %v", err)
	}

	storedItem := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "archived-cancel-forced", "spec.json"))
	if storedItem["status"] != "ready" {
		t.Fatalf("expected ready status after cancel, got %#v", storedItem["status"])
	}
	if storedItem["archived_at"] != "2025-01-01T00:00:00Z" {
		t.Fatalf("expected archived_at preserved after cancel, got %#v", storedItem["archived_at"])
	}
}

func TestCancel_ReturnsErrorWhenRestoreFails(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "cancel-restore-error", map[string]any{
		"name":        "cancel-restore-error",
		"title":       "Cancel Restore Error",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "cancel-restore-error")

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "cancel-restore-error",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}

	specPath := filepath.Join(root, "ideas", "cancel-restore-error", "spec.json")
	if err := os.Remove(specPath); err != nil {
		t.Fatalf("remove spec for restore failure simulation: %v", err)
	}

	_, err = service.Cancel(context.Background(), record.ExecutionID)
	if err == nil {
		t.Fatal("expected cancel restore error")
	}
	if !strings.Contains(err.Error(), "backlog status restore failed") {
		t.Fatalf("expected restore error, got %v", err)
	}
}

func mustWriteBacklogItem(t *testing.T, root, kind, name string, payload map[string]any) {
	t.Helper()
	kindDir := "ideas"
	switch kind {
	case "research":
		kindDir = "research"
	case "fix":
		kindDir = "fix"
	case "execute":
		kindDir = "execute"
	case "chore":
		kindDir = "chore"
	}
	dir := filepath.Join(root, kindDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir backlog item: %v", err)
	}
	if kind != "research" {
		if _, ok := payload["plan_ref"]; !ok {
			payload["plan_ref"] = map[string]any{
				"provider": "plan-manager",
				"plan_id":  "test-plan-" + name,
				"slug":     "test-plan-" + name,
				"role":     "execution_spec",
			}
		}
		if _, ok := payload["plan_acceptance"]; !ok {
			raw, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal acceptance fixture: %v", err)
			}
			var item backlogItem
			if err := json.Unmarshal(raw, &item); err != nil {
				t.Fatalf("decode acceptance fixture: %v", err)
			}
			item.Name = name
			item.Kind = kind
			payload["plan_acceptance"] = map[string]any{
				"actor": "test", "accepted_at": "2026-01-01T00:00:00Z",
				"plan_content_hash": "sha256:test-plan",
				"subject_version":   executionPlanAcceptanceSubjectVersion(item),
			}
		}
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.json"), bytes, 0o644); err != nil {
		t.Fatalf("write spec.json: %v", err)
	}
}

// mustWriteDeliverableFile is retained as a call-site fixture name. Readiness
// is now uniformly driven by plan_ref, including for research items.
func mustWriteDeliverableFile(t *testing.T, root, kind, name string) {
	t.Helper()
	_ = root
	_ = kind
	_ = name
}

func mustLoadBacklogItem(t *testing.T, path string) map[string]any {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value map[string]any
	if err := json.Unmarshal(bytes, &value); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return value
}

type stubInspector struct {
	state agentmanager.RunState
	err   error
}

func (s *stubInspector) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	if s.err != nil {
		return agentmanager.RunState{}, s.err
	}
	return s.state, nil
}

type stubStopper struct {
	stopCalls int
	err       error
}

func (s *stubStopper) StopRun(_ context.Context, _ string) error {
	s.stopCalls++
	return s.err
}

func TestCancel_StartingExecution(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "starting-cancel", map[string]any{
		"name":        "starting-cancel",
		"title":       "Starting Cancel",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "starting-cancel")

	stopper := &stubStopper{}
	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})
	service.stopper = stopper
	workflow := &stubPhasedPlanWorkflow{}
	service.SetPhasedPlanWorkflow(workflow)

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "starting-cancel",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	started, err := service.Start(context.Background(), record.ExecutionID)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if started.Status != StatusStarting {
		t.Fatalf("expected starting, got %s", started.Status)
	}

	canceled, err := service.Cancel(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatalf("Cancel error: %v", err)
	}
	if canceled.Status != StatusCanceled {
		t.Fatalf("expected canceled, got %s", canceled.Status)
	}
	if canceled.AgentWorkflowApplyState != workflowApplyComplete {
		t.Fatalf("workflow cancellation must close the consumer apply boundary, got %q", canceled.AgentWorkflowApplyState)
	}
	if stopper.stopCalls != 0 {
		t.Fatalf("workflow-owned cancellation must not stop a child run directly, got %d", stopper.stopCalls)
	}
	if workflow.cancelCalls != 1 {
		t.Fatalf("expected 1 workflow cancellation, got %d", workflow.cancelCalls)
	}
}

func TestStart_EmptyWorkflowExecutionIDFailsCleanly(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "norunid-item", map[string]any{
		"name": "norunid-item", "title": "No Run", "description": "d",
		"status": "backlog", "priority": 3, "tags": []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "norunid-item")

	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "p"},
	})
	workflow := &stubPhasedPlanWorkflow{start: agentmanager.WorkflowStart{DefinitionDigest: "sha256:def"}}
	service.SetPhasedPlanWorkflow(workflow)

	rec, err := service.QueueBacklog(context.Background(), CreateRequest{BacklogKind: "idea", BacklogName: "norunid-item", Mode: ModeManual})
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if _, startErr := service.Start(context.Background(), rec.ExecutionID); startErr == nil {
		t.Fatal("expected a start error when the workflow returns no execution id")
	}
	records, err := service.store.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var got *Record
	for i := range records {
		if records[i].ExecutionID == rec.ExecutionID {
			got = &records[i]
		}
	}
	if got == nil {
		t.Fatal("record disappeared")
	}
	if got.Status != StatusPending {
		t.Fatalf("record must stay pending/retryable, got %s", got.Status)
	}
}

func TestCancel_NeedsReviewExecution(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "review-cancel", map[string]any{
		"name":        "review-cancel",
		"title":       "Review Cancel",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "review-cancel")

	stopper := &stubStopper{}
	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})
	service.stopper = stopper
	workflow := &stubPhasedPlanWorkflow{}
	service.SetPhasedPlanWorkflow(workflow)

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "review-cancel",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	started, err := service.Start(context.Background(), record.ExecutionID)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	// Manually set to needs_review to simulate agent-manager transition
	records, idx, _ := service.loadRecordLocked(started.ExecutionID)
	records[idx].Status = StatusNeedsReview
	_ = service.store.Save(records)

	canceled, err := service.Cancel(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatalf("Cancel error: %v", err)
	}
	if canceled.Status != StatusCanceled {
		t.Fatalf("expected canceled, got %s", canceled.Status)
	}
}

func TestMigrateRecords_OrphanedRunning(t *testing.T) {
	records := []Record{
		{ExecutionID: "ok", Status: StatusRunning, RunID: "run-1"},
		{ExecutionID: "orphan", Status: StatusRunning, RunID: ""},
		{ExecutionID: "done", Status: StatusCompleted},
	}
	migrated := migrateRecords(records)
	if migrated[0].Status != StatusRunning {
		t.Fatalf("expected running with RunID to stay running, got %s", migrated[0].Status)
	}
	if migrated[1].Status != StatusFailed {
		t.Fatalf("expected orphaned running to become failed, got %s", migrated[1].Status)
	}
	if migrated[1].FailureReason != "orphaned execution: no run ID" {
		t.Fatalf("expected orphan failure reason, got %q", migrated[1].FailureReason)
	}
	if migrated[2].Status != StatusCompleted {
		t.Fatalf("expected completed to stay completed, got %s", migrated[2].Status)
	}
}

func TestIsFinalizationEligible(t *testing.T) {
	tests := []struct {
		name     string
		record   Record
		expected bool
	}{
		{name: "default process run", record: Record{}, expected: true},
		{name: "fixup run", record: Record{PromptTrace: &PromptTrace{Purpose: "fixup"}}, expected: true},
		{name: "followup run", record: Record{PromptTrace: &PromptTrace{Purpose: "followup"}}, expected: true},
		{name: "custom run", record: Record{PromptTrace: &PromptTrace{Purpose: "custom"}}, expected: true},
		{name: "research run excluded", record: Record{PromptTrace: &PromptTrace{Purpose: "research"}}, expected: false},
		{name: "archive run excluded", record: Record{ArchiveContext: &ArchiveContext{ScenarioName: "web-console"}}, expected: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if actual := isFinalizationEligible(tc.record); actual != tc.expected {
				t.Fatalf("expected eligible=%t, got %t", tc.expected, actual)
			}
		})
	}
}

func mustLoadRecords(t *testing.T, path string) []map[string]any {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read records: %v", err)
	}
	var records []map[string]any
	if err := json.Unmarshal(bytes, &records); err != nil {
		t.Fatalf("unmarshal records: %v", err)
	}
	return records
}

func TestRecordToProto_MapsFinalization(t *testing.T) {
	record := Record{
		ExecutionID:       "exec-review-1",
		BacklogKind:       "idea",
		BacklogName:       "reviewed-idea",
		Status:            StatusNeedsFixup,
		Mode:              ModeYOLO,
		ParentExecutionID: "parent-exec-0",
		FixupAttempt:      2,
		Finalization: &Finalization{
			Eligible:                true,
			Status:                  FinalizationStatusCompleted,
			Phase:                   FinalizationPhaseCompleted,
			ScopeSource:             FinalizationScopeSandboxDiff,
			StartedAt:               "2026-03-24T10:00:00Z",
			CompletedAt:             "2026-03-24T12:00:00Z",
			AggregateClassification: FinalizationAggregateNeedsWork,
			AggregateSummary:        "Tests failing",
			AffectedScenarios:       []string{"web-console"},
			Warnings: []FinalizationWarning{{
				Code:      "health_retry",
				Message:   "restarted twice",
				Retryable: true,
				CreatedAt: "2026-03-24T11:00:00Z",
			}},
			Scenarios: []ScenarioFinalization{
				{
					ScenarioName: "web-console",
					ChangedPaths: []string{"scenarios/web-console/ui/src/App.tsx"},
					Restart: RestartResult{
						Status:     FinalizationStatusCompleted,
						Attempts:   2,
						StartedAt:  "2026-03-24T10:00:00Z",
						FinishedAt: "2026-03-24T10:01:00Z",
					},
					Health: HealthCheckResult{
						Status:         FinalizationStatusCompleted,
						ScenarioStatus: "running",
						HealthStatus:   "healthy",
						SchemaValid:    true,
						Details:        "scenario is healthy",
						CheckedAt:      "2026-03-24T10:02:00Z",
					},
					Review: ScenarioReviewStep{
						Status: FinalizationStatusCompleted,
						JobID:  "review-job-1",
						Result: &ReviewResult{
							JobID:          "review-job-1",
							Classification: "needs_work",
							Summary:        "Tests failing",
							ReviewedAt:     "2026-03-24T12:00:00Z",
							Dimensions: []ReviewDimension{
								{Name: "tests", Status: "red", Details: "3 tests failing"},
								{Name: "lint", Status: "green"},
							},
						},
					},
				},
			},
		},
		CreatedAt: "2026-03-24T00:00:00Z",
		UpdatedAt: "2026-03-24T01:00:00Z",
	}
	pb := recordToProto(record)

	if pb.ParentExecutionId == nil || *pb.ParentExecutionId != "parent-exec-0" {
		t.Fatalf("expected parent_execution_id parent-exec-0, got %v", pb.ParentExecutionId)
	}
	if pb.FixupAttempt != 2 {
		t.Fatalf("expected fixup_attempt 2, got %d", pb.FixupAttempt)
	}
	if pb.Finalization == nil {
		t.Fatal("expected finalization to be set")
	}
	if pb.Finalization.AggregateClassification != "needs_work" {
		t.Fatalf("expected aggregate classification needs_work, got %s", pb.Finalization.AggregateClassification)
	}
	if pb.Finalization.AggregateSummary == nil || *pb.Finalization.AggregateSummary != "Tests failing" {
		t.Fatalf("expected aggregate summary 'Tests failing', got %v", pb.Finalization.AggregateSummary)
	}
	if len(pb.Finalization.Scenarios) != 1 {
		t.Fatalf("expected 1 scenario finalization, got %d", len(pb.Finalization.Scenarios))
	}
	if len(pb.Finalization.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(pb.Finalization.Warnings))
	}
	scenario := pb.Finalization.Scenarios[0]
	if scenario.Review == nil || scenario.Review.Result == nil {
		t.Fatal("expected scenario review result to be set")
	}
	dim0 := scenario.Review.Result.Dimensions[0]
	if dim0.Name != "tests" || dim0.Status != "red" {
		t.Fatalf("expected tests/red, got %s/%s", dim0.Name, dim0.Status)
	}
	if dim0.Details == nil || *dim0.Details != "3 tests failing" {
		t.Fatalf("expected details '3 tests failing', got %v", dim0.Details)
	}
	dim1 := scenario.Review.Result.Dimensions[1]
	if dim1.Name != "lint" || dim1.Status != "green" {
		t.Fatalf("expected lint/green, got %s/%s", dim1.Name, dim1.Status)
	}
}

// --- stubReviewClient for testing ---

type stubReviewClient struct {
	triggerJobID string
	triggerErr   error
	pollResult   *ReviewResult
	pollDone     bool
	pollErr      error
	pingErr      error
}

func (s *stubReviewClient) TriggerReview(_ context.Context, _ ReviewRequest) (string, error) {
	return s.triggerJobID, s.triggerErr
}

func (s *stubReviewClient) PollReview(_ context.Context, _ string) (*ReviewResult, bool, error) {
	return s.pollResult, s.pollDone, s.pollErr
}

func (s *stubReviewClient) Ping(_ context.Context) error {
	return s.pingErr
}

func TestTriggerReview_CompletedExecution(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "exec.json"))
	records := []Record{{
		ExecutionID: "exec-tr-1",
		BacklogKind: "execute",
		BacklogName: "my-feature",
		Status:      StatusCompleted,
		Mode:        ModeYOLO,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}}
	if err := store.Save(records); err != nil {
		t.Fatal(err)
	}

	// Create a backlog spec with acceptance_allow so scenario extraction works.
	specDir := filepath.Join(dir, "execute", "my-feature")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := `{"name":"my-feature","acceptance_allow":["scenarios/web-console/**"]}`
	if err := os.WriteFile(filepath.Join(specDir, "spec.json"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &Service{
		dataRoot:       dir,
		repoRoot:       dir,
		store:          store,
		reviewClient:   &stubReviewClient{triggerJobID: "job-new"},
		policyProvider: &defaultPolicyProvider{},
	}

	result, err := svc.TriggerReview(context.Background(), "exec-tr-1")
	if err != nil {
		t.Fatalf("TriggerReview failed: %v", err)
	}
	if result.Status != StatusValidating {
		t.Fatalf("expected status validating, got %s", result.Status)
	}
	if result.Finalization == nil {
		t.Fatal("expected finalization to be initialized")
	}
	if result.Finalization.Status != FinalizationStatusPending {
		t.Fatalf("expected pending finalization status, got %s", result.Finalization.Status)
	}
	if result.Finalization.Phase != FinalizationPhaseScopeDetection {
		t.Fatalf("expected scope_detection phase, got %s", result.Finalization.Phase)
	}
}

func TestTriggerReview_WrongStatus(t *testing.T) {
	for _, status := range []Status{StatusRunning, StatusPending, StatusStarting, StatusValidating} {
		t.Run(string(status), func(t *testing.T) {
			dir := t.TempDir()
			store := NewStore(filepath.Join(dir, "exec.json"))
			records := []Record{{
				ExecutionID: "exec-tr-2",
				BacklogKind: "execute",
				BacklogName: "test",
				Status:      status,
				RunID:       "run-placeholder", // Prevent migrateRecords from changing running→failed
				Mode:        ModeYOLO,
				CreatedAt:   nowRFC3339(),
				UpdatedAt:   nowRFC3339(),
			}}
			if err := store.Save(records); err != nil {
				t.Fatal(err)
			}

			svc := &Service{
				dataRoot:       dir,
				repoRoot:       dir,
				store:          store,
				reviewClient:   &stubReviewClient{triggerJobID: "job-x"},
				policyProvider: &defaultPolicyProvider{},
			}

			_, err := svc.TriggerReview(context.Background(), "exec-tr-2")
			if err == nil {
				t.Fatal("expected error for non-terminal execution")
			}
			if !strings.Contains(err.Error(), "cannot trigger post-run checks") {
				t.Fatalf("expected status validation error, got: %v", err)
			}
		})
	}
}

func TestTriggerReview_MissingExecution(t *testing.T) {
	svc := &Service{
		store:          NewStore(filepath.Join(t.TempDir(), "exec.json")),
		policyProvider: &defaultPolicyProvider{},
	}
	_, err := svc.TriggerReview(context.Background(), "exec-x")
	if err == nil {
		t.Fatal("expected error when execution is missing")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %v", err)
	}
}

func TestRecordToProto_OmitsEmptyFinalization(t *testing.T) {
	record := Record{
		ExecutionID: "exec-empty-fields",
		BacklogKind: "execute",
		BacklogName: "test",
		Status:      StatusCompleted,
		Mode:        ModeYOLO,
		CreatedAt:   "2026-03-24T00:00:00Z",
		UpdatedAt:   "2026-03-24T01:00:00Z",
	}
	pb := recordToProto(record)

	if pb.Finalization != nil {
		t.Fatalf("expected nil finalization for empty record, got %v", pb.Finalization)
	}
}

// --- buildExecutionPrompt tests ---

func TestBuildExecutionPrompt_ProcessRun(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "idea",
		Name:               "video-studio",
		Title:              "Video Studio",
		ItemFolder:         "/path/to/ideas/video-studio",
		RunType:            "process",
		DeliverablePath:    "plan-manager:video-studio",
		DeliverableContent: "# Plan\nBuild a video editor.",
		IdeaHandoff: &handoff.Package{
			Dir:             "/path/to/ideas/video-studio/handoff",
			BriefPath:       "/path/to/ideas/video-studio/handoff/brief.md",
			ManifestPath:    "/path/to/ideas/video-studio/handoff/manifest.json",
			SourceIndexPath: "/path/to/ideas/video-studio/handoff/source-index.json",
			BriefMarkdown:   "# Idea Execution Handoff\n",
		},
	})

	// Execution context tag present with metadata.
	if !strings.Contains(prompt, "<execution-context>") || !strings.Contains(prompt, "</execution-context>") {
		t.Error("missing execution-context tags")
	}
	if !strings.Contains(prompt, "Backlog item: idea/video-studio") {
		t.Error("missing backlog item line")
	}
	if !strings.Contains(prompt, "Title: Video Studio") {
		t.Error("missing title line")
	}
	if !strings.Contains(prompt, "Item folder: /path/to/ideas/video-studio") {
		t.Error("missing item folder line")
	}
	if !strings.Contains(prompt, "Run type: process") {
		t.Error("missing run type line")
	}

	// Plan tag present with content.
	if !strings.Contains(prompt, "<implementation-plan path=\"plan-manager:video-studio\">") || !strings.Contains(prompt, "</implementation-plan>") {
		t.Error("missing implementation-plan tags")
	}
	if !strings.Contains(prompt, "Build a video editor.") {
		t.Error("missing plan content")
	}
	if !strings.Contains(prompt, "<idea-handoff>") || !strings.Contains(prompt, "<idea-handoff-brief path=\"/path/to/ideas/video-studio/handoff/brief.md\">") {
		t.Error("missing idea handoff tags")
	}
	if !strings.Contains(prompt, "Execute the next bounded plan slice through the declared swarm-manager workflow") {
		t.Error("missing downstream handoff instruction")
	}

	// No review or follow-up tags for a process run.
	if strings.Contains(prompt, "<review-feedback>") {
		t.Error("process run should not have review-feedback tag")
	}
	if strings.Contains(prompt, "<follow-up-context>") {
		t.Error("process run should not have follow-up-context tag")
	}
}

func TestBuildExecutionPrompt_FixupRun(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "fix",
		Name:               "login-crash",
		Title:              "Fix Login Crash",
		ItemFolder:         "/path/to/fix/login-crash",
		RunType:            "fixup",
		DeliverablePath:    "plan-manager:login-crash",
		DeliverableContent: "# Plan\nFix the nil pointer.",
		ReviewFeedback:     "Tests still failing.\n- test_coverage (red): Missing edge case test",
	})

	if !strings.Contains(prompt, "Run type: fixup") {
		t.Error("missing fixup run type")
	}

	// Review feedback tag present.
	if !strings.Contains(prompt, "<review-feedback>") || !strings.Contains(prompt, "</review-feedback>") {
		t.Error("missing review-feedback tags")
	}
	if !strings.Contains(prompt, "Tests still failing.") {
		t.Error("missing review summary in prompt")
	}
	if !strings.Contains(prompt, "Missing edge case test") {
		t.Error("missing review dimension detail")
	}

	// Plan still included.
	if !strings.Contains(prompt, "<implementation-plan path=\"plan-manager:login-crash\">") {
		t.Error("fixup run should still include implementation plan")
	}
	if !strings.Contains(prompt, "Fix the nil pointer.") {
		t.Error("missing plan content in fixup prompt")
	}
}

func TestBuildExecutionPrompt_FollowUpRun(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "execute",
		Name:               "dependency-update",
		Title:              "Update Dependencies",
		ItemFolder:         "/path/to/execute/dependency-update",
		RunType:            "followup",
		DeliverablePath:    "plan-manager:dependency-update",
		DeliverableContent: "# Plan\nUpdate all Go deps.",
		FollowUpNote:       "Focus on the swarm-manager scenario only.",
	})

	if !strings.Contains(prompt, "Run type: followup") {
		t.Error("missing followup run type")
	}

	// Follow-up context tag present.
	if !strings.Contains(prompt, "<follow-up-context>") || !strings.Contains(prompt, "</follow-up-context>") {
		t.Error("missing follow-up-context tags")
	}
	if !strings.Contains(prompt, "Focus on the swarm-manager scenario only.") {
		t.Error("missing follow-up note content")
	}

	// Plan still included.
	if !strings.Contains(prompt, "Update all Go deps.") {
		t.Error("missing plan content")
	}
}

func TestBuildExecutionPrompt_NoPlan(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:       "chore",
		Name:       "cleanup",
		Title:      "Clean Up",
		ItemFolder: "/path/to/chore/cleanup",
		RunType:    "process",
	})

	if !strings.Contains(prompt, "<execution-context>") {
		t.Error("should still have execution context")
	}
	if strings.Contains(prompt, "<implementation-plan>") {
		t.Error("should not have implementation-plan tag when plan is empty")
	}
}

func TestBuildExecutionPrompt_EmptyOptionalSections(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "idea",
		Name:               "test",
		ItemFolder:         "/tmp/test",
		RunType:            "process",
		DeliverablePath:    "plan-manager:test",
		DeliverableContent: "plan content",
		ReviewFeedback:     "",
		FollowUpNote:       "   ",
	})

	if strings.Contains(prompt, "<review-feedback>") {
		t.Error("empty review feedback should not produce tag")
	}
	if strings.Contains(prompt, "<follow-up-context>") {
		t.Error("whitespace-only follow-up note should not produce tag")
	}
}

func TestBuildExecutionPrompt_NoTitle(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "fix",
		Name:               "bug",
		ItemFolder:         "/tmp/fix/bug",
		RunType:            "process",
		DeliverablePath:    "plan-manager:bug",
		DeliverableContent: "fix it",
	})

	if strings.Contains(prompt, "Title:") {
		t.Error("should not include Title line when title is empty")
	}
}

func TestBuildExecutionPrompt_SuggestedSkills(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "execute",
		Name:               "refactor-api",
		Title:              "Refactor API",
		ItemFolder:         "/tmp/execute/refactor-api",
		RunType:            "process",
		DeliverablePath:    "plan-manager:refactor-api",
		DeliverableContent: "# Plan\nRefactor.",
		SuggestedSkills:    []string{"refactor", "screaming-architecture-audit"},
	})

	if !strings.Contains(prompt, "<suggested-skills>") || !strings.Contains(prompt, "</suggested-skills>") {
		t.Error("missing suggested-skills tags")
	}
	if !strings.Contains(prompt, "prompt-manager skill read refactor") {
		t.Error("missing refactor skill in suggested-skills")
	}
	if !strings.Contains(prompt, "prompt-manager skill read screaming-architecture-audit") {
		t.Error("missing screaming-architecture-audit skill in suggested-skills")
	}
}

func TestBuildExecutionPrompt_NoSuggestedSkills(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "fix",
		Name:               "bug-fix",
		ItemFolder:         "/tmp/fix/bug-fix",
		RunType:            "process",
		DeliverablePath:    "plan-manager:bug-fix",
		DeliverableContent: "fix it",
	})

	if strings.Contains(prompt, "<suggested-skills>") {
		t.Error("should not include suggested-skills when none provided")
	}
}

func TestBuildFinalizationFeedback_NilResult(t *testing.T) {
	if got := buildFinalizationFeedback(nil); got != "" {
		t.Errorf("expected empty string for nil result, got %q", got)
	}
}

func TestBuildFinalizationFeedback_WithDimensions(t *testing.T) {
	result := &Finalization{
		AggregateSummary: "Needs work.",
		Warnings: []FinalizationWarning{{
			Code:      "health_retry",
			Message:   "restarted twice",
			Retryable: true,
			CreatedAt: "2026-03-24T01:00:00Z",
		}},
		Scenarios: []ScenarioFinalization{{
			ScenarioName: "web-console",
			Review: ScenarioReviewStep{
				Result: &ReviewResult{
					Summary: "Needs work.",
					Dimensions: []ReviewDimension{
						{Name: "tests", Status: "red", Details: "3 tests failing"},
						{Name: "docs", Status: "green", Details: "OK"},
						{Name: "lint", Status: "yellow", Details: "2 warnings"},
					},
				},
			},
		}},
	}
	got := buildFinalizationFeedback(result)

	if !strings.Contains(got, "Needs work.") {
		t.Error("missing summary")
	}
	if !strings.Contains(got, "web-console tests (red): 3 tests failing") {
		t.Error("missing red dimension")
	}
	if strings.Contains(got, "docs (green)") {
		t.Error("green dimensions should be excluded")
	}
	if !strings.Contains(got, "warning [health_retry]: restarted twice") {
		t.Error("missing warning")
	}
	if !strings.Contains(got, "web-console lint (yellow): 2 warnings") {
		t.Error("missing yellow dimension")
	}
}

// --- checkReviewAgentEnabled tests ---

type errPolicyProvider struct {
	err error
}

func (p *errPolicyProvider) LoadPolicy() (Policy, error) {
	return Policy{}, p.err
}

func TestCheckReviewAgentEnabled_Enabled(t *testing.T) {
	svc := &Service{
		policyProvider: &stubPolicyProvider{policy: Policy{ReviewAgentEnabled: true}},
	}
	enabled, reason := svc.checkReviewAgentEnabled()
	if !enabled {
		t.Fatal("expected enabled=true")
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got %q", reason)
	}
}

func TestCheckReviewAgentEnabled_Disabled(t *testing.T) {
	svc := &Service{
		policyProvider: &stubPolicyProvider{policy: Policy{ReviewAgentEnabled: false}},
	}
	enabled, reason := svc.checkReviewAgentEnabled()
	if enabled {
		t.Fatal("expected enabled=false")
	}
	if reason != finalizationWarningEvidenceSkippedDisabled {
		t.Fatalf("expected %q, got %q", finalizationWarningEvidenceSkippedDisabled, reason)
	}
}

func TestCheckReviewAgentEnabled_PolicyLoadError(t *testing.T) {
	svc := &Service{
		policyProvider: &errPolicyProvider{err: errors.New("disk full")},
	}
	enabled, reason := svc.checkReviewAgentEnabled()
	if enabled {
		t.Fatal("expected enabled=false on policy load error")
	}
	if reason != finalizationWarningEvidenceSkippedPolicyErr {
		t.Fatalf("expected %q, got %q", finalizationWarningEvidenceSkippedPolicyErr, reason)
	}
}

func TestEvidenceSkipMessage(t *testing.T) {
	svc := &Service{}
	msg := svc.evidenceSkipMessage(finalizationWarningEvidenceSkippedDisabled)
	if !strings.Contains(msg, "disabled in settings") {
		t.Fatalf("expected settings hint, got %q", msg)
	}
	msg = svc.evidenceSkipMessage(finalizationWarningEvidenceSkippedPolicyErr)
	if !strings.Contains(msg, "Could not load settings") {
		t.Fatalf("expected policy error hint, got %q", msg)
	}
	msg = svc.evidenceSkipMessage("unknown_code")
	if msg != "Evidence gathering was skipped." {
		t.Fatalf("expected fallback message, got %q", msg)
	}
}
