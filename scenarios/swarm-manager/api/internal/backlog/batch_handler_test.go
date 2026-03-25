package backlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"
)

// mockInitiativeAssigner implements InitiativeAssigner for testing.
type mockInitiativeAssigner struct {
	existingInitiatives map[string]bool
	addedItems          map[string][]string
	ensureErr           error
	addErr              error
}

func newMockInitiativeAssigner() *mockInitiativeAssigner {
	return &mockInitiativeAssigner{
		existingInitiatives: make(map[string]bool),
		addedItems:          make(map[string][]string),
	}
}

func (m *mockInitiativeAssigner) EnsureExists(name string) error {
	if m.ensureErr != nil {
		return m.ensureErr
	}
	m.existingInitiatives[name] = true
	return nil
}

func (m *mockInitiativeAssigner) AddItems(name string, items []string) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.addedItems[name] = append(m.addedItems[name], items...)
	return nil
}

func boolPtr(v bool) *bool { return &v }

func setupBatchTestHandler(t *testing.T) (*Handler, string, *mockInitiativeAssigner) {
	t.Helper()
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "batch-run", TaskID: "batch-task"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	ia := newMockInitiativeAssigner()
	h.SetInitiativeAssigner(ia)
	return h, rootDir, ia
}

func doBatchCreate(t *testing.T, h *Handler, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/v1/backlog/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.BatchCreate(w, req)
	return w
}

func TestBatchCreate_Success(t *testing.T) {
	h, rootDir, ia := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "auth-service", Title: "Auth Service", Kind: "idea"},
			{Name: "user-mgmt", Title: "User Management", Kind: "idea"},
			{Name: "fix-login", Title: "Fix Login Bug", Kind: "fix"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	var resp batchCreateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Count != 3 {
		t.Errorf("expected count 3, got %d", resp.Count)
	}
	if len(resp.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(resp.Items))
	}

	// Verify items exist on disk.
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "auth-service", "spec.json"))
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "user-mgmt", "spec.json"))
	testutil.AssertFileExists(t, filepath.Join(rootDir, "fix", "fix-login", "spec.json"))

	// No initiative was requested.
	if len(ia.existingInitiatives) != 0 {
		t.Errorf("expected no initiatives created, got %d", len(ia.existingInitiatives))
	}
}

func TestBatchCreate_WithInitiative(t *testing.T) {
	h, _, ia := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "dashboard", Title: "Dashboard", Kind: "idea", AutoWorkshop: boolPtr(false)},
			{Name: "api-refactor", Title: "API Refactor", Kind: "execute", AutoWorkshop: boolPtr(false)},
		},
		Initiative: "q1-sprint",
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	var resp batchCreateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Initiative != "q1-sprint" {
		t.Errorf("expected initiative 'q1-sprint', got %q", resp.Initiative)
	}

	// Verify initiative was ensured and items added.
	if !ia.existingInitiatives["q1-sprint"] {
		t.Error("expected initiative 'q1-sprint' to be ensured")
	}
	if len(ia.addedItems["q1-sprint"]) != 2 {
		t.Errorf("expected 2 items added to initiative, got %d", len(ia.addedItems["q1-sprint"]))
	}
}

func TestBatchCreate_EmptyBatch(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
}

func TestBatchCreate_DuplicateNameInBatch(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "same-item", Title: "First", Kind: "idea"},
			{Name: "same-item", Title: "Second", Kind: "idea"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)

	// Verify no items were created (all-or-nothing).
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "same-item", "spec.json"))
}

func TestBatchCreate_InvalidItem_RollsBackAll(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "valid-item", Title: "Valid", Kind: "idea"},
			{Name: "bad-item", Title: "", Kind: "idea"}, // missing title
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)

	// Verify no items were created.
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "valid-item", "spec.json"))
}

func TestBatchCreate_CycleDetection(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	// Create an existing item that item-b will depend on.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:   "existing",
		Title:  "Existing",
		Status: StatusBacklog,
		Tags:   []string{},
	})

	// Create a cycle: a depends on b, b depends on a.
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-a", Title: "Item A", Kind: "idea", DependsOn: []string{"idea/item-b"}},
			{Name: "item-b", Title: "Item B", Kind: "idea", DependsOn: []string{"idea/item-a"}},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)

	body := w.Body.String()
	if !contains(body, "cycle") {
		t.Errorf("expected cycle error, got: %s", body)
	}

	// Verify no items were created.
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "item-a", "spec.json"))
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "item-b", "spec.json"))
}

func TestBatchCreate_WithIntraBatchDependencies(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	// item-b depends on item-a (both in the batch) — valid, no cycle.
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-a", Title: "Item A", Kind: "idea"},
			{Name: "item-b", Title: "Item B", Kind: "idea", DependsOn: []string{"idea/item-a"}},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "item-a", "spec.json"))
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "item-b", "spec.json"))
}

func TestBatchCreate_DependencyOnNonexistent(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "orphan", Title: "Orphan", Kind: "idea", DependsOn: []string{"idea/ghost"}},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)

	body := w.Body.String()
	if !contains(body, "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %s", body)
	}
}

func TestBatchCreate_ConflictWithExisting(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:   "existing",
		Title:  "Existing",
		Status: StatusBacklog,
		Tags:   []string{},
	})

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "existing", Title: "Duplicate", Kind: "idea"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatus(t, w, http.StatusConflict)
}

func TestBatchCreate_InvalidKind(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item", Title: "Item", Kind: "invalid-kind"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
}

func TestBatchCreate_PriorityValidation(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	p := int32(15)
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item", Title: "Item", Kind: "idea", Priority: &p},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
}

// failingSaveStore wraps a FileStore but returns an error on the Nth SaveItem call.
// This allows testing the rollback path in batch-create.
type failingSaveStore struct {
	*FileStore
	failOnCall int // 1-based: fail on the Nth SaveItem call
	callCount  int
}

func (f *failingSaveStore) SaveItem(item BacklogItem) error {
	f.callCount++
	if f.callCount >= f.failOnCall {
		return fmt.Errorf("simulated disk write failure")
	}
	return f.FileStore.SaveItem(item)
}

func TestBatchCreate_SaveFailure_RollsBack(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	// Inject a store that fails on the 2nd SaveItem call.
	// The first item creates successfully, the second triggers rollback.
	h.store = &failingSaveStore{
		FileStore:  NewFileStore(rootDir),
		failOnCall: 2,
	}

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-ok", Title: "Item OK", Kind: "idea"},
			{Name: "item-fail", Title: "Item Fail", Kind: "idea"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatus(t, w, http.StatusInternalServerError)

	// Verify rollback: both item directories should be cleaned up.
	okDir := filepath.Join(rootDir, "ideas", "item-ok")
	failDir := filepath.Join(rootDir, "ideas", "item-fail")

	if _, err := os.Stat(okDir); !os.IsNotExist(err) {
		t.Errorf("expected %q to be removed during rollback, but it still exists", okDir)
	}
	if _, err := os.Stat(failDir); !os.IsNotExist(err) {
		t.Errorf("expected %q to be removed during rollback, but it still exists", failDir)
	}
}

func TestBatchCreate_ExceedsMaxBatchSize(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	items := make([]batchCreateItem, 101)
	for i := range items {
		items[i] = batchCreateItem{
			Name:  fmt.Sprintf("item-%03d", i),
			Title: fmt.Sprintf("Item %d", i),
			Kind:  "idea",
		}
	}

	payload := batchCreateRequest{Items: items}
	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestBatchCreate_InitiativeAddItemsFails_ReturnsWarning(t *testing.T) {
	h, rootDir, ia := setupBatchTestHandler(t)
	ia.addErr = fmt.Errorf("disk full")

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "widget-a", Title: "Widget A", Kind: "idea", AutoWorkshop: boolPtr(false)},
			{Name: "widget-b", Title: "Widget B", Kind: "fix", AutoWorkshop: boolPtr(false)},
		},
		Initiative: "my-init",
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	var resp batchCreateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Items should still be created on disk.
	if resp.Count != 2 {
		t.Errorf("expected count 2, got %d", resp.Count)
	}
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "widget-a", "spec.json"))
	testutil.AssertFileExists(t, filepath.Join(rootDir, "fix", "widget-b", "spec.json"))

	// Response should contain a warning about initiative assignment failure.
	if len(resp.Warnings) == 0 {
		t.Fatal("expected warnings in response, got none")
	}
	found := false
	for _, w := range resp.Warnings {
		if contains(w, "initiative assignment failed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning containing 'initiative assignment failed', got: %v", resp.Warnings)
	}
}

func TestBatchCreate_InitiativeEnsureExistsFails_Returns500(t *testing.T) {
	h, rootDir, ia := setupBatchTestHandler(t)
	ia.ensureErr = fmt.Errorf("store corrupt")

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "orphan-item", Title: "Orphan", Kind: "idea"},
		},
		Initiative: "broken-init",
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatus(t, w, http.StatusInternalServerError)

	// Item should NOT be on disk — EnsureExists fails before item creation.
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "orphan-item", "spec.json"))
}

func TestBatchCreate_DependencyValidation_ExistingAndBatch(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	// Create item-a on disk so it can be referenced.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:   "item-a",
		Title:  "Item A",
		Status: StatusBacklog,
		Tags:   []string{},
	})

	// Batch-create: item-b depends on existing item-a, item-c depends on batch item-b.
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-b", Title: "Item B", Kind: "idea", DependsOn: []string{"idea/item-a"}},
			{Name: "item-c", Title: "Item C", Kind: "idea", DependsOn: []string{"idea/item-b"}},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	// Verify both items exist with correct dependencies.
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "item-b", "spec.json"))
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "item-c", "spec.json"))

	// Now try to batch-create an item with a nonexistent dependency.
	payload2 := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-d", Title: "Item D", Kind: "idea", DependsOn: []string{"idea/nonexistent"}},
		},
	}

	w2 := doBatchCreate(t, h, payload2)
	testutil.AssertStatusBadRequest(t, w2)

	body := w2.Body.String()
	if !contains(body, "does not exist") {
		t.Errorf("expected 'does not exist' error for nonexistent dep, got: %s", body)
	}
}

func TestBatchCreate_WithEffort(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	effortS := "S"
	effortXL := "xl" // lowercase to test normalization
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item-with-effort", Title: "Item With Effort", Kind: "idea", Effort: &effortS, AutoWorkshop: boolPtr(false)},
			{Name: "item-with-xl", Title: "Item XL", Kind: "fix", Effort: &effortXL, AutoWorkshop: boolPtr(false)},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[batchCreateResponse](t, w)
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].Effort != "S" {
		t.Errorf("expected effort 'S', got %q", resp.Items[0].Effort)
	}
	if resp.Items[1].Effort != "XL" {
		t.Errorf("expected effort 'XL', got %q", resp.Items[1].Effort)
	}
}

func TestBatchCreate_InvalidEffort(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	badEffort := "HUGE"
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "bad-effort-item", Title: "Bad Effort", Kind: "idea", Effort: &badEffort},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)

	body := w.Body.String()
	if !contains(body, "effort must be") {
		t.Errorf("expected effort validation error, got: %s", body)
	}
}

func TestBatchCreate_EffortOptional(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "no-effort-item", Title: "No Effort", Kind: "idea"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[batchCreateResponse](t, w)
	if resp.Items[0].Effort != "" {
		t.Errorf("expected empty effort, got %q", resp.Items[0].Effort)
	}
}
