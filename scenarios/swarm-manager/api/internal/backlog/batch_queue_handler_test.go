package backlog

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"
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
