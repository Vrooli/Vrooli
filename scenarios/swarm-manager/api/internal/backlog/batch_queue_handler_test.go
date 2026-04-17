package backlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/testutil"
	"testing"
)

func doBatchQueue(t *testing.T, h *Handler, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/v1/backlog/batch/queue", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.BatchQueue(w, req)
	return w
}

func TestBatchQueue_PreviewMode(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Create items with a dependency chain: b depends on a.
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:     "item-a",
		Title:    "Item A",
		Status:   StatusReady,
		Priority: 3,
		Tags:     []string{},
	})
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:      "item-b",
		Title:     "Item B",
		Status:    StatusReady,
		Priority:  3,
		Tags:      []string{},
		DependsOn: []string{"idea/item-a"},
	})

	payload := batchQueueRequest{
		Items:   []string{"idea/item-a", "idea/item-b"},
		Confirm: false, // preview mode
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}

	// In preview mode, no items should be marked as queued.
	for _, r := range resp.Results {
		if r.Queued {
			t.Errorf("expected item %q not to be queued in preview mode", r.Item)
		}
	}

	// Verify execution order has both items.
	if len(resp.ExecutionOrder) != 2 {
		t.Errorf("expected 2 items in execution order, got %d", len(resp.ExecutionOrder))
	}
}

func TestBatchQueue_UnmetDependencies(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// item-a is NOT completed, item-b depends on item-a.
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:     "item-a",
		Title:    "Item A",
		Status:   StatusBacklog, // not completed
		Priority: 3,
		Tags:     []string{},
	})
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:      "item-b",
		Title:     "Item B",
		Status:    StatusReady,
		Priority:  3,
		Tags:      []string{},
		DependsOn: []string{"idea/item-a"},
	})

	// Queue only item-b (item-a is not in the batch and not completed).
	payload := batchQueueRequest{
		Items:   []string{"idea/item-b"},
		Confirm: false,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}

	result := resp.Results[0]
	if result.Queued {
		t.Error("expected item-b not to be queued due to unmet dependencies")
	}
	if len(result.UnmetDependencies) == 0 {
		t.Error("expected unmet dependencies to be reported")
	}
}

func TestBatchQueue_DependencyOrder(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Create items: c depends on b, b depends on a.
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:     "item-a",
		Title:    "Item A",
		Status:   StatusReady,
		Priority: 3,
		Tags:     []string{},
	})
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:      "item-b",
		Title:     "Item B",
		Status:    StatusReady,
		Priority:  3,
		Tags:      []string{},
		DependsOn: []string{"idea/item-a"},
	})
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:      "item-c",
		Title:     "Item C",
		Status:    StatusReady,
		Priority:  3,
		Tags:      []string{},
		DependsOn: []string{"idea/item-b"},
	})

	// Request all three in reverse order.
	payload := batchQueueRequest{
		Items:   []string{"idea/item-c", "idea/item-b", "idea/item-a"},
		Confirm: false,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Topological sort should put a before b before c.
	if len(resp.ExecutionOrder) != 3 {
		t.Fatalf("expected 3 items in execution order, got %d", len(resp.ExecutionOrder))
	}

	order := resp.ExecutionOrder
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

	if aIdx >= bIdx || bIdx >= cIdx {
		t.Errorf("expected order a < b < c, got a=%d, b=%d, c=%d", aIdx, bIdx, cIdx)
	}
}

func TestBatchQueue_EmptyBatch(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := batchQueueRequest{
		Items:   []string{},
		Confirm: false,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
}

func TestBatchQueue_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := batchQueueRequest{
		Items:   []string{"idea/nonexistent"},
		Confirm: false,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusNotFound(t, w)
}

func TestBatchQueue_ResearchItemSkipped(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindResearch, BacklogItem{
		Name:     "research-item",
		Title:    "Research Item",
		Status:   StatusReady,
		Priority: 3,
		Tags:     []string{},
	})

	payload := batchQueueRequest{
		Items:   []string{"research/research-item"},
		Confirm: false,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].Queued {
		t.Error("research items should not be queued")
	}
}

func TestBatchQueue_CompletedDependencyAllowsQueuing(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// item-a is completed; item-b depends on it and should be queueable.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:     "dep-done",
		Title:    "Done Dep",
		Status:   StatusCompleted,
		Priority: 3,
		Tags:     []string{},
	})
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:      "needs-dep",
		Title:     "Needs Dep",
		Status:    StatusReady,
		Priority:  3,
		Tags:      []string{},
		DependsOn: []string{"idea/dep-done"},
	})

	// Write plan.md for dep-done too so directory exists properly.
	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "dep-done", "plan.md"),
		"# Plan\nCompleted plan.")

	payload := batchQueueRequest{
		Items:   []string{"idea/needs-dep"},
		Confirm: false, // preview
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}

	// Should be ready to queue (no unmet deps).
	if len(resp.Results[0].UnmetDependencies) > 0 {
		t.Errorf("expected no unmet dependencies, got %v", resp.Results[0].UnmetDependencies)
	}
}

func TestBatchQueue_DuplicateItemsDeduped(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "dup-item", Title: "Dup Item", Status: StatusReady, Priority: 3, Tags: []string{},
	})

	// Submit the same item twice.
	payload := batchQueueRequest{
		Items:   []string{"idea/dup-item", "idea/dup-item"},
		Confirm: false,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should be deduplicated to 1 result, not 2.
	if len(resp.Results) != 1 {
		t.Errorf("expected 1 result after dedup, got %d", len(resp.Results))
	}
}

// setupConfirmTestHandler creates a handler with a mock ExecutionQueuer for testing
// the confirm:true code path.
func setupConfirmTestHandler(t *testing.T, eq *mockExecutionQueuer) (*Handler, string) {
	t.Helper()
	h, rootDir := setupTestHandler(t)
	h.SetExecutionQueuer(eq)
	return h, rootDir
}

func TestBatchQueue_ConfirmTrue_QueuesAll(t *testing.T) {
	eq := &mockExecutionQueuer{
		preflightResult: execution.ProcessPreflight{Ready: true},
	}
	h, rootDir := setupConfirmTestHandler(t, eq)

	// Create two ready items with no dependencies.
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "alpha", Title: "Alpha", Status: StatusReady, Priority: 3, Tags: []string{},
	})
	createReadyTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "beta", Title: "Beta", Status: StatusReady, Priority: 2, Tags: []string{},
	})

	payload := batchQueueRequest{
		Items:   []string{"idea/alpha", "fix/beta"},
		Confirm: true,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}

	for _, r := range resp.Results {
		if !r.Queued {
			t.Errorf("expected %q to be queued, got message: %s", r.Item, r.Message)
		}
		if r.ExecutionID == "" {
			t.Errorf("expected %q to have an execution ID", r.Item)
		}
	}

	if len(eq.queueCalls) != 2 {
		t.Errorf("expected 2 QueueBacklog calls, got %d", len(eq.queueCalls))
	}
}

func TestBatchQueue_ConfirmTrue_PreflightBlocks(t *testing.T) {
	eq := &mockExecutionQueuer{
		preflightResult: execution.ProcessPreflight{
			Ready:           false,
			BlockingReasons: []string{"missing target scenario"},
		},
	}
	h, rootDir := setupConfirmTestHandler(t, eq)

	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "blocked-item", Title: "Blocked", Status: StatusReady, Priority: 3, Tags: []string{},
	})

	payload := batchQueueRequest{
		Items:   []string{"idea/blocked-item"},
		Confirm: true,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}

	r := resp.Results[0]
	if r.Queued {
		t.Error("expected item not to be queued when preflight blocks")
	}
	if !contains(r.Message, "Blocked") {
		t.Errorf("expected message to contain 'Blocked', got %q", r.Message)
	}

	if len(eq.queueCalls) != 0 {
		t.Errorf("expected 0 QueueBacklog calls when preflight blocks, got %d", len(eq.queueCalls))
	}
}

func TestBatchQueue_ConfirmTrue_QueueFails(t *testing.T) {
	eq := &mockExecutionQueuer{
		preflightResult: execution.ProcessPreflight{Ready: true},
		queueErr:        fmt.Errorf("agent-manager unavailable"),
	}
	h, rootDir := setupConfirmTestHandler(t, eq)

	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "fail-item", Title: "Will Fail", Status: StatusReady, Priority: 3, Tags: []string{},
	})

	payload := batchQueueRequest{
		Items:   []string{"idea/fail-item"},
		Confirm: true,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}

	r := resp.Results[0]
	if r.Queued {
		t.Error("expected item not to be queued when QueueBacklog fails")
	}
	if !contains(r.Message, "Queue failed") {
		t.Errorf("expected message to contain 'Queue failed', got %q", r.Message)
	}
}

func TestBatchQueue_ConfirmTrue_PartialSuccess(t *testing.T) {
	eq := &mockExecutionQueuer{
		preflightResult: execution.ProcessPreflight{Ready: true},
	}
	h, rootDir := setupConfirmTestHandler(t, eq)

	// item-a is ready and will queue successfully.
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "succeeds", Title: "Succeeds", Status: StatusReady, Priority: 1, Tags: []string{},
	})

	// item-b depends on an external item that is NOT completed and NOT in the batch.
	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "external-dep", Title: "External", Status: StatusBacklog, Priority: 5, Tags: []string{},
	})
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:      "has-unmet-dep",
		Title:     "Has Unmet Dep",
		Status:    StatusReady,
		Priority:  2,
		Tags:      []string{},
		DependsOn: []string{"fix/external-dep"},
	})

	payload := batchQueueRequest{
		Items:   []string{"idea/succeeds", "idea/has-unmet-dep"},
		Confirm: true,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}

	// Find results by item reference.
	resultMap := make(map[string]batchQueueItemResult)
	for _, r := range resp.Results {
		resultMap[r.Item] = r
	}

	succeeds := resultMap["idea/succeeds"]
	if !succeeds.Queued {
		t.Errorf("expected 'idea/succeeds' to be queued, got message: %s", succeeds.Message)
	}
	if succeeds.ExecutionID == "" {
		t.Error("expected 'idea/succeeds' to have an execution ID")
	}

	blocked := resultMap["idea/has-unmet-dep"]
	if blocked.Queued {
		t.Error("expected 'idea/has-unmet-dep' not to be queued due to unmet deps")
	}
	if len(blocked.UnmetDependencies) == 0 {
		t.Error("expected unmet dependencies to be reported for 'idea/has-unmet-dep'")
	}
}

func TestBatchQueue_InvalidMode(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{Name: "alpha", Title: "Alpha", Status: StatusReady})

	rec := doBatchQueue(t, h, map[string]any{
		"items": []string{"idea/alpha"},
		"mode":  "invalid_mode",
	})
	testutil.AssertStatus(t, rec, 400)
	if body := rec.Body.String(); !strings.Contains(body, "invalid execution mode") {
		t.Errorf("expected 'invalid execution mode' in body, got: %s", body)
	}
}

func TestBatchQueue_DeletedDependencyDoesNotBlock(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Create an item whose dependency no longer exists on disk.
	// This simulates the common workflow where a dependency was completed
	// and then archived/deleted. It must NOT block execution.
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:      "has-archived-dep",
		Title:     "Has Archived Dep",
		Status:    StatusReady,
		Priority:  3,
		Tags:      []string{},
		DependsOn: []string{"idea/deleted-item"},
	})

	payload := batchQueueRequest{
		Items:   []string{"idea/has-archived-dep"},
		Confirm: false,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}

	r := resp.Results[0]
	if len(r.UnmetDependencies) != 0 {
		t.Errorf("deleted/archived dep should not block; got unmet: %v", r.UnmetDependencies)
	}
}

func TestBatchQueue_CycleDetection_ShowsPath(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Create items with a cycle: a depends on b, b depends on a.
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:      "cycle-a",
		Title:     "Cycle A",
		Status:    StatusReady,
		Priority:  3,
		Tags:      []string{},
		DependsOn: []string{"idea/cycle-b"},
	})
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:      "cycle-b",
		Title:     "Cycle B",
		Status:    StatusReady,
		Priority:  3,
		Tags:      []string{},
		DependsOn: []string{"idea/cycle-a"},
	})

	payload := batchQueueRequest{
		Items:   []string{"idea/cycle-a", "idea/cycle-b"},
		Confirm: false,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)

	body := w.Body.String()
	if !strings.Contains(body, "dependency cycle detected:") {
		t.Errorf("expected cycle path in error, got: %s", body)
	}
	// The cycle path should mention at least one of the items.
	if !strings.Contains(body, "cycle-a") && !strings.Contains(body, "cycle-b") {
		t.Errorf("expected cycle path to mention item names, got: %s", body)
	}
}

func TestBatchQueue_ForceTrue_NonForceableReasonStillBlocks(t *testing.T) {
	eq := &mockExecutionQueuer{
		preflightResult: execution.ProcessPreflight{
			Ready:           false,
			BlockingReasons: []string{"missing target scenario"},
		},
	}
	h, rootDir := setupConfirmTestHandler(t, eq)

	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "force-blocked", Title: "Force Blocked", Status: StatusReady, Priority: 3, Tags: []string{},
	})

	payload := batchQueueRequest{
		Items:   []string{"idea/force-blocked"},
		Confirm: true,
		Force:   true,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}

	r := resp.Results[0]
	if r.Queued {
		t.Error("expected item NOT to be queued — 'missing target scenario' is non-forceable")
	}
	if !contains(r.Message, "Blocked") {
		t.Errorf("expected 'Blocked' in message, got: %q", r.Message)
	}

	if len(eq.queueCalls) != 0 {
		t.Errorf("expected 0 QueueBacklog calls when force can't override, got %d", len(eq.queueCalls))
	}
}

func TestBatchQueue_NonQueueableStatus_Skipped(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Create items with non-queueable statuses.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "done-item", Title: "Done", Status: StatusCompleted, Priority: 3, Tags: []string{},
	})
	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "failed-item", Title: "Failed", Status: StatusFailed, Priority: 3, Tags: []string{},
	})

	payload := batchQueueRequest{
		Items:   []string{"idea/done-item", "fix/failed-item"},
		Confirm: false,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}

	for _, r := range resp.Results {
		if r.Queued {
			t.Errorf("expected %q not to be queued (non-queueable status)", r.Item)
		}
	}
}

func TestBatchQueue_PartialSuccess_DependencyChain(t *testing.T) {
	eq := &mockExecutionQueuer{
		preflightResult: execution.ProcessPreflight{Ready: true},
	}
	h, rootDir := setupConfirmTestHandler(t, eq)

	// Create chain: A (no deps) -> B (depends on A) -> C (depends on B).
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "chain-a", Title: "Chain A", Status: StatusReady, Priority: 1, Tags: []string{},
	})
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "chain-b", Title: "Chain B", Status: StatusReady, Priority: 2, Tags: []string{},
		DependsOn: []string{"idea/chain-a"},
	})
	createReadyTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "chain-c", Title: "Chain C", Status: StatusReady, Priority: 3, Tags: []string{},
		DependsOn: []string{"idea/chain-b"},
	})

	payload := batchQueueRequest{
		Items:   []string{"idea/chain-c", "idea/chain-b", "idea/chain-a"},
		Confirm: true,
	}

	w := doBatchQueue(t, h, payload)
	testutil.AssertStatusOK(t, w)

	var resp batchQueueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// All three should be queued — items queued in batch count as "met" for later items.
	if len(resp.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(resp.Results))
	}

	for _, r := range resp.Results {
		if !r.Queued {
			t.Errorf("expected %q to be queued, got message: %s", r.Item, r.Message)
		}
	}

	// Verify topological order: A before B before C.
	if len(resp.ExecutionOrder) != 3 {
		t.Fatalf("expected 3 in execution order, got %d", len(resp.ExecutionOrder))
	}
	orderMap := map[string]int{}
	for i, ref := range resp.ExecutionOrder {
		orderMap[ref] = i
	}
	if orderMap["idea/chain-a"] >= orderMap["idea/chain-b"] {
		t.Errorf("expected chain-a before chain-b in execution order")
	}
	if orderMap["idea/chain-b"] >= orderMap["idea/chain-c"] {
		t.Errorf("expected chain-b before chain-c in execution order")
	}

	// Verify 3 queue calls were made.
	if len(eq.queueCalls) != 3 {
		t.Errorf("expected 3 QueueBacklog calls, got %d", len(eq.queueCalls))
	}
}
