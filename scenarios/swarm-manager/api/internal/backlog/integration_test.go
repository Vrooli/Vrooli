package backlog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/execution"
	"swarm-manager/internal/testutil"
)

// mockExecutionQueuer implements ExecutionQueuer for testing the confirm:true path.
type mockExecutionQueuer struct {
	preflightResult execution.ProcessPreflight
	preflightErr    error
	queueResult     execution.Record
	queueErr        error

	// Manual-accept controls.
	manuallyAcceptedCalls []manuallyAcceptedCall
	manuallyAcceptedID    string
	manuallyAcceptedOK    bool
	manuallyAcceptedErr   error

	// Track calls for assertion.
	preflightCalls []string // "kind/name" of each ProcessPreflight call
	queueCalls     []execution.CreateRequest

	// Retry-latest controls.
	retryLatestCalls    []retryLatestCall
	retryLatestRecord   execution.Record
	retryLatestHasPrior bool
	retryLatestErr      error
}

type manuallyAcceptedCall struct {
	Kind, Name, Acceptor, Reason string
}

func (m *mockExecutionQueuer) ProcessPreflight(_ context.Context, backlogKind, backlogName string) (execution.ProcessPreflight, error) {
	m.preflightCalls = append(m.preflightCalls, backlogKind+"/"+backlogName)
	if m.preflightErr != nil {
		return execution.ProcessPreflight{}, m.preflightErr
	}
	return m.preflightResult, nil
}

func (m *mockExecutionQueuer) ProcessPreflightForSpec(_ context.Context, spec execution.PreflightSpec) execution.ProcessPreflight {
	m.preflightCalls = append(m.preflightCalls, spec.Kind+"/"+spec.Name)
	return m.preflightResult
}

func (m *mockExecutionQueuer) QueueBacklog(_ context.Context, req execution.CreateRequest) (execution.Record, error) {
	m.queueCalls = append(m.queueCalls, req)
	if m.queueErr != nil {
		return execution.Record{}, m.queueErr
	}
	// Return a unique execution ID per item for traceability.
	result := m.queueResult
	result.ExecutionID = fmt.Sprintf("exec-%s-%s", req.BacklogKind, req.BacklogName)
	result.BacklogKind = req.BacklogKind
	result.BacklogName = req.BacklogName
	return result, nil
}

func (m *mockExecutionQueuer) ManuallyAcceptLatestForBacklog(_ context.Context, backlogKind, backlogName, acceptor, reason string) (string, bool, error) {
	m.manuallyAcceptedCalls = append(m.manuallyAcceptedCalls, manuallyAcceptedCall{
		Kind: backlogKind, Name: backlogName, Acceptor: acceptor, Reason: reason,
	})
	return m.manuallyAcceptedID, m.manuallyAcceptedOK, m.manuallyAcceptedErr
}

// retryLatestCall captures one RetryLatestForBacklog invocation.
type retryLatestCall struct {
	Kind, Name, Note string
}

// Retry-latest controls. Add fields to the mock so tests can wire
// success/failure paths without touching unrelated mocks.
func (m *mockExecutionQueuer) RetryLatestForBacklog(_ context.Context, backlogKind, backlogName, note string) (execution.Record, bool, error) {
	m.retryLatestCalls = append(m.retryLatestCalls, retryLatestCall{Kind: backlogKind, Name: backlogName, Note: note})
	if m.retryLatestErr != nil {
		return execution.Record{}, m.retryLatestHasPrior, m.retryLatestErr
	}
	return m.retryLatestRecord, m.retryLatestHasPrior, nil
}

// TestGoldenPath_BatchCreateMilestoneQueue exercises the full pipeline:
// batch-create items with dependencies and an milestone, then batch-queue
// them with confirm:true, verifying topological ordering and execution IDs.
func TestGoldenPath_BatchCreateMilestoneQueue(t *testing.T) {
	h, rootDir, ia := setupBatchTestHandler(t)

	eq := &mockExecutionQueuer{
		preflightResult: execution.ProcessPreflight{Ready: true},
	}
	h.SetExecutionQueuer(eq)

	// Phase 1: Batch-create 3 items in a dependency chain with an milestone.
	p1, p2, p3 := int32(1), int32(2), int32(3)
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-a", Title: "Item A", Kind: "idea", Priority: &p1, Milestone: "test-milestone", PlanRef: testPlanRef("item-a")},
			{Name: "item-b", Title: "Item B", Kind: "idea", Priority: &p2, DependsOn: []string{"idea/item-a"}, Milestone: "test-milestone", PlanRef: testPlanRef("item-b")},
			{Name: "item-c", Title: "Item C", Kind: "idea", Priority: &p3, DependsOn: []string{"idea/item-b"}, Milestone: "test-milestone", PlanRef: testPlanRef("item-c")},
		},
		Milestones: []batchCreateMilestone{
			{Name: "test-milestone", Title: "Test Milestone"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	// Verify items on disk.
	for _, name := range []string{"item-a", "item-b", "item-c"} {
		specPath := filepath.Join(rootDir, "ideas", name, "spec.json")
		if _, err := os.Stat(specPath); err != nil {
			t.Errorf("expected %s to exist on disk: %v", specPath, err)
		}
	}

	// Verify milestone was ensured and items were added.
	if _, ok := ia.snapshots["test-milestone"]; !ok {
		t.Error("expected milestone 'test-milestone' to be created")
	}
	if len(ia.addedItems["test-milestone"]) != 3 {
		t.Errorf("expected 3 items added to milestone, got %d", len(ia.addedItems["test-milestone"]))
	}

	// Phase 2: Make items queueable by setting status to "ready" and preserving plan_ref.
	store := NewFileStore(rootDir)
	for _, name := range []string{"item-a", "item-b", "item-c"} {
		item, err := store.LoadItem(KindIdea, name)
		if err != nil {
			t.Fatalf("failed to load %s: %v", name, err)
		}
		item.Status = StatusReady
		if err := store.SaveItem(item); err != nil {
			t.Fatalf("failed to save %s: %v", name, err)
		}
	}

	// Phase 3: Preview mode — all items should be ready.
	previewPayload := batchQueueRequest{
		Items:   []string{"idea/item-c", "idea/item-b", "idea/item-a"},
		Confirm: false,
	}
	w = doBatchQueue(t, h, previewPayload)
	testutil.AssertStatusOK(t, w)

	var previewResp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&previewResp); err != nil {
		t.Fatalf("failed to decode preview response: %v", err)
	}
	if len(previewResp.Results) != 3 {
		t.Fatalf("expected 3 preview results, got %d", len(previewResp.Results))
	}
	for _, r := range previewResp.Results {
		if r.Queued {
			t.Errorf("preview mode should not queue items, but %q was queued", r.Item)
		}
	}

	// Phase 4: Confirm mode — all items should be queued with execution IDs.
	confirmPayload := batchQueueRequest{
		Items:   []string{"idea/item-c", "idea/item-b", "idea/item-a"},
		Confirm: true,
	}
	w = doBatchQueue(t, h, confirmPayload)
	testutil.AssertStatusOK(t, w)

	var confirmResp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&confirmResp); err != nil {
		t.Fatalf("failed to decode confirm response: %v", err)
	}

	if len(confirmResp.Results) != 3 {
		t.Fatalf("expected 3 confirm results, got %d", len(confirmResp.Results))
	}

	// All items should be queued with execution IDs.
	for _, r := range confirmResp.Results {
		if !r.Queued {
			t.Errorf("expected %q to be queued, got message: %s", r.Item, r.Message)
		}
		if r.ExecutionID == "" {
			t.Errorf("expected %q to have an execution ID", r.Item)
		}
	}

	// Verify topological order: A before B before C.
	order := confirmResp.ExecutionOrder
	aIdx, bIdx, cIdx := -1, -1, -1
	for i, ref := range order {
		switch ref {
		case "idea/item-a":
			aIdx = i
		case "idea/item-b":
			bIdx = i
		case "idea/item-c":
			cIdx = i
		}
	}
	if aIdx == -1 || bIdx == -1 || cIdx == -1 {
		t.Fatalf("expected all 3 items in execution order, got %v", order)
	}
	if aIdx >= bIdx || bIdx >= cIdx {
		t.Errorf("expected topological order A < B < C, got A=%d, B=%d, C=%d", aIdx, bIdx, cIdx)
	}

	// Verify the mock received 3 queue calls.
	if len(eq.queueCalls) != 3 {
		t.Errorf("expected 3 QueueBacklog calls, got %d", len(eq.queueCalls))
	}
}

func testPlanRef(name string) *PlanRef {
	return &PlanRef{
		Provider: PlanRefProviderPlanManager,
		PlanID:   "test-plan-" + name,
		Slug:     "test-plan-" + name,
		Role:     PlanRefRoleExecutionSpec,
	}
}
