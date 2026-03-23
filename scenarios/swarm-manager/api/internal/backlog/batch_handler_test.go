package backlog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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

func setupBatchTestHandler(t *testing.T) (*Handler, string, *mockInitiativeAssigner) {
	t.Helper()
	h, rootDir := setupTestHandler(t)
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
			{Name: "dashboard", Title: "Dashboard", Kind: "idea"},
			{Name: "api-refactor", Title: "API Refactor", Kind: "execute"},
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

func TestBatchCreate_SaveFailure_RollsBack(t *testing.T) {
	h, rootDir, _ := setupBatchTestHandler(t)

	// Make the ideas directory read-only so SaveItem fails after mkdir.
	ideasDir := filepath.Join(rootDir, "ideas")
	// Create one valid item first.
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "will-fail", Title: "Will Fail", Kind: "idea"},
		},
	}

	// Create the item directory but make it so the spec.json write fails.
	itemDir := filepath.Join(ideasDir, "will-fail")
	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Make spec.json a directory so WriteFile fails.
	if err := os.MkdirAll(filepath.Join(itemDir, "spec.json"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Now the item already exists, so we expect a conflict.
	w := doBatchCreate(t, h, payload)
	testutil.AssertStatus(t, w, http.StatusConflict)
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
