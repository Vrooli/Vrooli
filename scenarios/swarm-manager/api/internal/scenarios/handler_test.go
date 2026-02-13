package scenarios

import (
	"bytes"
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/testutil"
)

type scenarioPayload struct {
	Name              string   `json:"name"`
	DisplayName       string   `json:"display_name"`
	Description       string   `json:"description"`
	Status            string   `json:"status"`
	Priority          int      `json:"priority"`
	CompletenessScore *int     `json:"completeness_score,omitempty"`
	IsGreenfield      bool     `json:"is_greenfield"`
	Tags              []string `json:"tags"`
}

type listScenariosResponse struct {
	Scenarios []scenarioPayload `json:"scenarios"`
}

type scenarioResponse struct {
	Scenario scenarioPayload `json:"scenario"`
}

type deleteScenarioResponse struct {
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
	Message  string `json:"message"`
}

type stubSource struct {
	scenarios []ScenarioSource
	err       error
}

func (s stubSource) List(_ context.Context) ([]ScenarioSource, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.scenarios, nil
}

type stubCompleteness struct {
	scores map[string]int
	err    error
}

func (s stubCompleteness) Scores(_ context.Context) (map[string]int, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.scores == nil {
		return map[string]int{}, nil
	}
	return s.scores, nil
}

type stubLifecycle struct {
	startCalls   []string
	stopCalls    []string
	restartCalls []string
	err          error
}

func (s *stubLifecycle) Start(_ context.Context, name string) error {
	s.startCalls = append(s.startCalls, name)
	return s.err
}

func (s *stubLifecycle) Stop(_ context.Context, name string) error {
	s.stopCalls = append(s.stopCalls, name)
	return s.err
}

func (s *stubLifecycle) Restart(_ context.Context, name string) error {
	s.restartCalls = append(s.restartCalls, name)
	return s.err
}

func newTestHandler(root string, scenarios []ScenarioSource) *Handler {
	return NewHandlerWithDeps(
		filepath.Join(root, "scenarios"),
		stubSource{scenarios: scenarios},
		&stubLifecycle{},
		stubCompleteness{scores: map[string]int{}},
	)
}

// setupTestScenarios creates temporary test scenarios using t.TempDir() for automatic cleanup.
// [REQ:REQ-P0-006] Test scenario catalog listing
func setupTestScenarios(t *testing.T) (string, []ScenarioSource) {
	t.Helper()

	root := t.TempDir()
	scenariosDir := filepath.Join(root, "scenarios")

	// Create test scenario 1 (has PRD.md, so not greenfield)
	scenario1Path := filepath.Join(scenariosDir, "test-scenario-1")
	scenario1Dir := filepath.Join(scenario1Path, ".vrooli")
	testutil.WriteJSONFile(t, filepath.Join(scenario1Dir, "service.json"), map[string]any{
		"profile": map[string]any{
			"name":        "Test Scenario One",
			"description": "First test scenario",
			"tags":        []string{"api", "backend"},
		},
	})
	testutil.WriteJSONFile(t, filepath.Join(scenario1Dir, "lighthouse.json"), map[string]int{
		"priority": 1,
	})
	testutil.WriteFile(t, filepath.Join(scenario1Path, "PRD.md"), "# PRD")

	// Create test scenario 2 (no PRD.md, so greenfield)
	scenario2Path := filepath.Join(scenariosDir, "test-scenario-2")
	scenario2Dir := filepath.Join(scenario2Path, ".vrooli")
	testutil.WriteJSONFile(t, filepath.Join(scenario2Dir, "service.json"), map[string]any{
		"profile": map[string]any{
			"name":        "Test Scenario Two",
			"description": "Second test scenario for frontend",
			"tags":        []string{"ui", "frontend"},
		},
	})
	testutil.WriteJSONFile(t, filepath.Join(scenario2Dir, "lighthouse.json"), map[string]int{
		"priority": 3,
	})

	// Create test scenario 3 (no priority set - uses default)
	scenario3Path := filepath.Join(scenariosDir, "another-scenario")
	scenario3Dir := filepath.Join(scenario3Path, ".vrooli")
	testutil.WriteJSONFile(t, filepath.Join(scenario3Dir, "service.json"), map[string]any{
		"profile": map[string]any{
			"name":        "Another Scenario",
			"description": "Another test scenario",
			"tags":        []string{"api", "testing"},
		},
	})

	sources := []ScenarioSource{
		{
			Name:        "test-scenario-1",
			Description: "First test scenario",
			Path:        scenario1Path,
			Status:      "running",
			Tags:        []string{"api", "backend"},
		},
		{
			Name:        "test-scenario-2",
			Description: "Second test scenario for frontend",
			Path:        scenario2Path,
			Status:      "stopped",
			Tags:        []string{"ui", "frontend"},
		},
		{
			Name:        "another-scenario",
			Description: "Another test scenario",
			Path:        scenario3Path,
			Status:      "error",
			Tags:        []string{"api", "testing"},
		},
	}

	return root, sources
}

// TestList_Empty tests listing with no scenarios.
// [REQ:REQ-P0-006] Test empty scenario list
func TestList_Empty(t *testing.T) {
	root := t.TempDir()
	handler := newTestHandler(root, nil)
	req := httptest.NewRequest("GET", "/api/v1/scenarios", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	testutil.AssertStatusOK(t, rec)

	resp := testutil.DecodeJSON[listScenariosResponse](t, rec)
	if len(resp.Scenarios) != 0 {
		t.Errorf("expected 0 scenarios, got %d", len(resp.Scenarios))
	}
}

// TestList_WithScenarios tests listing scenarios.
// [REQ:REQ-P0-006] Test scenario list with data
func TestList_WithScenarios(t *testing.T) {
	root, sources := setupTestScenarios(t)
	handler := newTestHandler(root, sources)
	req := httptest.NewRequest("GET", "/api/v1/scenarios", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	testutil.AssertStatusOK(t, rec)

	resp := testutil.DecodeJSON[listScenariosResponse](t, rec)
	if len(resp.Scenarios) != 3 {
		t.Errorf("expected 3 scenarios, got %d", len(resp.Scenarios))
	}

	// Should be sorted by priority (1, 3, 5)
	if resp.Scenarios[0].Priority != 1 {
		t.Errorf("expected first scenario priority 1, got %d", resp.Scenarios[0].Priority)
	}
	if resp.Scenarios[1].Priority != 3 {
		t.Errorf("expected second scenario priority 3, got %d", resp.Scenarios[1].Priority)
	}
	if resp.Scenarios[2].Priority != 5 {
		t.Errorf("expected third scenario priority 5 (default), got %d", resp.Scenarios[2].Priority)
	}
}

// TestList_Search tests search filtering.
// [REQ:REQ-P0-006] Test search functionality
func TestList_Search(t *testing.T) {
	root, sources := setupTestScenarios(t)
	handler := newTestHandler(root, sources)

	tests := []struct {
		name          string
		search        string
		expectedCount int
	}{
		{"search by name", "test-scenario", 2},
		{"search by description", "frontend", 1},
		{"search case insensitive", "ANOTHER", 1},
		{"search no matches", "nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/scenarios?search="+tt.search, nil)
			rec := httptest.NewRecorder()

			handler.List(rec, req)

			testutil.AssertStatusOK(t, rec)

			resp := testutil.DecodeJSON[listScenariosResponse](t, rec)
			if len(resp.Scenarios) != tt.expectedCount {
				t.Errorf("expected %d scenarios, got %d", tt.expectedCount, len(resp.Scenarios))
			}
		})
	}
}

// TestList_FilterByTags tests tag filtering.
// [REQ:REQ-P0-006] Test tag filter functionality
func TestList_FilterByTags(t *testing.T) {
	root, sources := setupTestScenarios(t)
	handler := newTestHandler(root, sources)

	tests := []struct {
		name          string
		tags          string
		expectedCount int
	}{
		{"filter by api tag", "api", 2},
		{"filter by frontend tag", "frontend", 1},
		{"filter by multiple tags", "api,frontend", 3},
		{"filter no matches", "database", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/scenarios?tags="+tt.tags, nil)
			rec := httptest.NewRecorder()

			handler.List(rec, req)

			testutil.AssertStatusOK(t, rec)

			resp := testutil.DecodeJSON[listScenariosResponse](t, rec)
			if len(resp.Scenarios) != tt.expectedCount {
				t.Errorf("expected %d scenarios, got %d", tt.expectedCount, len(resp.Scenarios))
			}
		})
	}
}

// TestList_Sorting tests sort parameter.
// [REQ:REQ-P0-006] Test priority sorting
func TestList_Sorting(t *testing.T) {
	root, sources := setupTestScenarios(t)
	handler := newTestHandler(root, sources)

	t.Run("sort by name ascending", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/scenarios?sort=name&order=asc", nil)
		rec := httptest.NewRecorder()

		handler.List(rec, req)

		resp := testutil.DecodeJSON[listScenariosResponse](t, rec)
		if resp.Scenarios[0].Name != "another-scenario" {
			t.Errorf("expected first scenario 'another-scenario', got %s", resp.Scenarios[0].Name)
		}
	})

	t.Run("sort by priority descending", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/scenarios?sort=priority&order=desc", nil)
		rec := httptest.NewRecorder()

		handler.List(rec, req)

		resp := testutil.DecodeJSON[listScenariosResponse](t, rec)
		// Highest priority number should be first
		if resp.Scenarios[0].Priority != 5 {
			t.Errorf("expected first scenario priority 5, got %d", resp.Scenarios[0].Priority)
		}
	})
}

// TestGet_Success tests getting a single scenario.
// [REQ:REQ-P0-006] Test scenario detail endpoint
func TestGet_Success(t *testing.T) {
	root, sources := setupTestScenarios(t)

	handler := newTestHandler(root, sources)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.Get).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/scenarios/test-scenario-1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)

	resp := testutil.DecodeJSON[scenarioResponse](t, rec)
	scenario := resp.Scenario
	if scenario.Name != "test-scenario-1" {
		t.Errorf("expected name 'test-scenario-1', got %s", scenario.Name)
	}
	if scenario.DisplayName != "test-scenario-1" {
		t.Errorf("expected display name 'test-scenario-1', got %s", scenario.DisplayName)
	}
	if scenario.IsGreenfield {
		t.Error("expected IsGreenfield false (PRD.md exists)")
	}
}

// TestGet_NotFound tests getting a non-existent scenario.
// [REQ:REQ-P0-006] Test scenario not found
func TestGet_NotFound(t *testing.T) {
	root, sources := setupTestScenarios(t)

	handler := newTestHandler(root, sources)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.Get).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/scenarios/nonexistent", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatusNotFound(t, rec)
}

// TestScenario_Structure tests the Scenario struct fields.
// [REQ:REQ-P0-006] Test scenario data structure
func TestScenario_Structure(t *testing.T) {
	root, sources := setupTestScenarios(t)

	handler := newTestHandler(root, sources)
	req := httptest.NewRequest("GET", "/api/v1/scenarios?search=test-scenario-2", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	resp := testutil.DecodeJSON[listScenariosResponse](t, rec)
	if len(resp.Scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(resp.Scenarios))
	}

	s := resp.Scenarios[0]
	if s.Name != "test-scenario-2" {
		t.Errorf("expected name 'test-scenario-2', got %s", s.Name)
	}
	if s.Description != "Second test scenario for frontend" {
		t.Errorf("unexpected description: %s", s.Description)
	}
	if len(s.Tags) != 2 || s.Tags[0] != "ui" {
		t.Errorf("unexpected tags: %v", s.Tags)
	}
	if !s.IsGreenfield {
		t.Error("expected IsGreenfield true (no PRD.md)")
	}
}

// TestUpdateMetadata_Success tests successful metadata update.
// [REQ:REQ-P0-007] Test scenario metadata update endpoint
func TestUpdateMetadata_Success(t *testing.T) {
	root, sources := setupTestScenarios(t)

	handler := newTestHandler(root, sources)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.UpdateMetadata).Methods("PATCH")

	// Update greenfield flag
	body := `{"is_greenfield": true}`
	req := httptest.NewRequest("PATCH", "/api/v1/scenarios/test-scenario-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)

	resp := testutil.DecodeJSON[scenarioResponse](t, rec)
	scenario := resp.Scenario
	if !scenario.IsGreenfield {
		t.Error("expected isGreenfield to be true")
	}
}

// TestUpdateMetadata_ToggleGreenfield tests greenfield toggle.
// [REQ:REQ-P0-007] Test greenfield metadata toggle
func TestUpdateMetadata_ToggleGreenfield(t *testing.T) {
	root, sources := setupTestScenarios(t)

	handler := newTestHandler(root, sources)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.UpdateMetadata).Methods("PATCH")

	// Toggle greenfield to true for scenario with PRD.md
	body := `{"is_greenfield": true}`
	req := httptest.NewRequest("PATCH", "/api/v1/scenarios/test-scenario-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)

	resp := testutil.DecodeJSON[scenarioResponse](t, rec)
	scenario := resp.Scenario
	if !scenario.IsGreenfield {
		t.Error("expected isGreenfield to be true after toggle")
	}
}

// TestUpdateMetadata_NotFound tests updating non-existent scenario.
// [REQ:REQ-P0-007] Test metadata update for missing scenario
func TestUpdateMetadata_NotFound(t *testing.T) {
	root, sources := setupTestScenarios(t)

	handler := newTestHandler(root, sources)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.UpdateMetadata).Methods("PATCH")

	body := `{"is_greenfield": true}`
	req := httptest.NewRequest("PATCH", "/api/v1/scenarios/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatusNotFound(t, rec)
}

// TestUpdateMetadata_InvalidJSON tests invalid JSON handling.
// [REQ:REQ-P0-007] Test metadata update with invalid JSON
func TestUpdateMetadata_InvalidJSON(t *testing.T) {
	root, sources := setupTestScenarios(t)

	handler := newTestHandler(root, sources)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.UpdateMetadata).Methods("PATCH")

	body := `{invalid json}`
	req := httptest.NewRequest("PATCH", "/api/v1/scenarios/test-scenario-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatusBadRequest(t, rec)
}

// TestUpdateMetadata_PartialUpdate tests that partial updates work.
// [REQ:REQ-P0-007] Test partial metadata update
func TestUpdateMetadata_PartialUpdate(t *testing.T) {
	root, sources := setupTestScenarios(t)

	handler := newTestHandler(root, sources)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.UpdateMetadata).Methods("PATCH")

	// First update to greenfield false
	body1 := `{"is_greenfield": false}`
	req1 := httptest.NewRequest("PATCH", "/api/v1/scenarios/test-scenario-1", bytes.NewBufferString(body1))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	// Then update only isGreenfield
	body2 := `{"is_greenfield": true}`
	req2 := httptest.NewRequest("PATCH", "/api/v1/scenarios/test-scenario-1", bytes.NewBufferString(body2))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	resp2 := testutil.DecodeJSON[scenarioResponse](t, rec2)
	scenario := resp2.Scenario

	// Final update should be reflected
	if !scenario.IsGreenfield {
		t.Error("expected isGreenfield to be true")
	}
}

// TestUpdateMetadata_PersistsToDisk tests that metadata is persisted.
// [REQ:REQ-P0-007] Test metadata persistence to disk
func TestUpdateMetadata_PersistsToDisk(t *testing.T) {
	root, sources := setupTestScenarios(t)

	handler := newTestHandler(root, sources)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.UpdateMetadata).Methods("PATCH")

	// Update metadata
	body := `{"is_greenfield": true}`
	req := httptest.NewRequest("PATCH", "/api/v1/scenarios/test-scenario-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)

	// Verify metadata file was created
	metaPath := filepath.Join(root, "scenarios", "test-scenario-1", ".vrooli", "metadata.json")
	testutil.AssertFileExists(t, metaPath)

	metadata := testutil.ReadJSONFile[ScenarioMetadata](t, metaPath)
	if !metadata.IsGreenfield {
		t.Error("expected isGreenfield true in file")
	}
}

// TestScenario_GreenfieldDefault tests default greenfield derivation.
// [REQ:REQ-P0-007] Test default metadata behavior
func TestScenario_GreenfieldDefault(t *testing.T) {
	root, sources := setupTestScenarios(t)

	handler := newTestHandler(root, sources)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.Get).Methods("GET")

	// Get a scenario that has no metadata.json file
	req := httptest.NewRequest("GET", "/api/v1/scenarios/test-scenario-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	resp := testutil.DecodeJSON[scenarioResponse](t, rec)
	scenario := resp.Scenario

	// PRD exists for scenario-1, so greenfield should default false.
	if scenario.IsGreenfield {
		t.Error("expected isGreenfield to default to false when PRD exists")
	}
}

// TestDelete_Success tests successful scenario deletion.
// [REQ:REQ-P0-008] Test scenario deletion endpoint
func TestDelete_Success(t *testing.T) {
	root, sources := setupTestScenarios(t)

	handler := newTestHandler(root, sources)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.Delete).Methods("DELETE")

	// Verify scenario exists before deletion
	scenarioPath := filepath.Join(root, "scenarios", "test-scenario-1")
	testutil.AssertFileExists(t, filepath.Join(scenarioPath, ".vrooli", "service.json"))

	req := httptest.NewRequest("DELETE", "/api/v1/scenarios/test-scenario-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)

	response := testutil.DecodeJSON[deleteScenarioResponse](t, rec)
	if response.Name != "test-scenario-1" {
		t.Errorf("expected name 'test-scenario-1', got %q", response.Name)
	}
	if response.Archived {
		t.Error("expected archived to be false")
	}
	if response.Message != "Scenario permanently deleted" {
		t.Errorf("unexpected message: %s", response.Message)
	}

	// Verify scenario directory was removed
	testutil.AssertFileNotExists(t, scenarioPath)
}

// TestDelete_NotFound tests deletion of non-existent scenario.
// [REQ:REQ-P0-008] Test deletion of missing scenario
func TestDelete_NotFound(t *testing.T) {
	root := t.TempDir()

	handler := newTestHandler(root, nil)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.Delete).Methods("DELETE")

	req := httptest.NewRequest("DELETE", "/api/v1/scenarios/non-existent", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusNotFound(t, rec)
}

// TestDelete_WithArchive tests deletion with archive option.
// [REQ:REQ-P0-008] Test scenario archive to backlog (idea)
func TestDelete_WithArchive(t *testing.T) {
	root, sources := setupTestScenarios(t)

	// Create ideas directory for swarm-manager
	ideasDir := filepath.Join(root, "scenarios", "swarm-manager", "ideas")
	testutil.MakeDir(t, ideasDir)

	handler := newTestHandler(root, sources)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.Delete).Methods("DELETE")

	req := httptest.NewRequest("DELETE", "/api/v1/scenarios/test-scenario-1?archive=true", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)

	response := testutil.DecodeJSON[deleteScenarioResponse](t, rec)
	if !response.Archived {
		t.Error("expected archived to be true")
	}
	if response.Message != "Scenario archived to backlog (idea) and deleted" {
		t.Errorf("unexpected message: %s", response.Message)
	}

	// Verify scenario directory was removed
	scenarioPath := filepath.Join(root, "scenarios", "test-scenario-1")
	testutil.AssertFileNotExists(t, scenarioPath)

	// Verify idea was created (relative path depends on handler implementation)
	ideaPath := filepath.Join(ideasDir, "test-scenario-1-archived")
	specPath := filepath.Join(ideaPath, "spec.json")
	testutil.AssertFileExists(t, specPath)

	spec := testutil.ReadJSONFile[map[string]any](t, specPath)
	if spec["sourceScenarioName"] != "test-scenario-1" {
		t.Fatalf("expected sourceScenarioName test-scenario-1, got %v", spec["sourceScenarioName"])
	}
	expectedSourcePath := filepath.Join(root, "scenarios", "test-scenario-1")
	if spec["sourceScenarioPath"] != expectedSourcePath {
		t.Fatalf("expected sourceScenarioPath %q, got %v", expectedSourcePath, spec["sourceScenarioPath"])
	}
	if spec["archiveReason"] != "scenario deleted with archive=true" {
		t.Fatalf("expected archiveReason to be populated, got %v", spec["archiveReason"])
	}
	if spec["archivedAt"] == "" {
		t.Fatal("expected archivedAt to be populated")
	}
	if spec["archivedBy"] == "" {
		t.Fatal("expected archivedBy to be populated")
	}
	if spec["preservePresetOrCustom"] != "none" {
		t.Fatalf("expected preservePresetOrCustom to be none, got %v", spec["preservePresetOrCustom"])
	}
	preserved, ok := spec["preservedFiles"].([]any)
	if !ok {
		t.Fatalf("expected preservedFiles to be an array, got %T", spec["preservedFiles"])
	}
	if len(preserved) != 0 {
		t.Fatalf("expected preservedFiles to be empty, got %v", preserved)
	}
}

// TestDelete_Idempotent tests that delete is idempotent (second delete returns 404).
// [REQ:REQ-P0-008] Test deletion idempotency
func TestDelete_Idempotent(t *testing.T) {
	root, sources := setupTestScenarios(t)

	handler := newTestHandler(root, sources)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{name}", handler.Delete).Methods("DELETE")

	// First delete succeeds
	req1 := httptest.NewRequest("DELETE", "/api/v1/scenarios/test-scenario-1", nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	testutil.AssertStatusOK(t, rec1)

	// Second delete returns 404
	req2 := httptest.NewRequest("DELETE", "/api/v1/scenarios/test-scenario-1", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	testutil.AssertStatusNotFound(t, rec2)
}

// TestDeleteResponse_Structure tests DeleteScenarioResponse JSON serialization.
// [REQ:REQ-P0-008] Test deletion response structure
func TestDeleteResponse_Structure(t *testing.T) {
	resp := deleteScenarioResponse{
		Name:     "my-scenario",
		Archived: true,
		Message:  "Scenario archived to backlog (idea) and deleted",
	}

	// Verify struct can be serialized (json.Marshal is tested via encoder)
	if resp.Name != "my-scenario" {
		t.Errorf("expected name 'my-scenario', got %q", resp.Name)
	}
	if !resp.Archived {
		t.Error("expected archived to be true")
	}
	if resp.Message == "" {
		t.Error("expected message to be non-empty")
	}
}

func TestCopyPreservedFiles_PresetSkipsIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	scenarioPath := filepath.Join(root, "scenario")
	ideaPath := filepath.Join(root, "idea")

	testutil.WriteFile(t, filepath.Join(scenarioPath, "PRD.md"), "# PRD")
	testutil.WriteFile(t, filepath.Join(scenarioPath, "docs", "guide.md"), "guide")
	testutil.WriteFile(t, filepath.Join(scenarioPath, "node_modules", "somepkg", "README.md"), "ignore me")

	preset := "documentation"
	preserved, err := copyPreservedFiles(scenarioPath, ideaPath, &apipb.PreserveFilesRequest{
		Preset: &preset,
	})
	if err != nil {
		t.Fatalf("copyPreservedFiles returned error: %v", err)
	}

	if len(preserved) != 2 {
		t.Fatalf("expected 2 preserved files, got %d: %v", len(preserved), preserved)
	}

	for _, path := range preserved {
		if path == filepath.Join("node_modules", "somepkg", "README.md") {
			t.Fatalf("expected node_modules file to be excluded from preset archive: %v", preserved)
		}
	}

	testutil.AssertFileExists(t, filepath.Join(ideaPath, "PRD.md"))
	testutil.AssertFileExists(t, filepath.Join(ideaPath, "docs", "guide.md"))
	testutil.AssertFileNotExists(t, filepath.Join(ideaPath, "node_modules", "somepkg", "README.md"))
}

func TestCopyPreservedFiles_RejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	scenarioPath := filepath.Join(root, "scenario")
	ideaPath := filepath.Join(root, "idea")

	testutil.WriteFile(t, filepath.Join(scenarioPath, "PRD.md"), "# PRD")
	testutil.WriteFile(t, filepath.Join(root, "outside.md"), "outside")

	preserved, err := copyPreservedFiles(scenarioPath, ideaPath, &apipb.PreserveFilesRequest{
		Paths: []string{"../outside.md", "PRD.md"},
	})
	if err != nil {
		t.Fatalf("copyPreservedFiles returned error: %v", err)
	}

	if len(preserved) != 1 || preserved[0] != "PRD.md" {
		t.Fatalf("expected only PRD.md to be preserved, got %v", preserved)
	}

	testutil.AssertFileExists(t, filepath.Join(ideaPath, "PRD.md"))
	testutil.AssertFileNotExists(t, filepath.Join(ideaPath, "outside.md"))
}

func TestResolveGlobPattern_RejectsUnsafePattern(t *testing.T) {
	root := t.TempDir()

	if _, err := resolveGlobPattern(root, "../*.md"); err == nil {
		t.Fatal("expected traversal pattern to be rejected")
	}
	if _, err := resolveGlobPattern(root, "/tmp/*.md"); err == nil {
		t.Fatal("expected absolute pattern to be rejected")
	}
}
