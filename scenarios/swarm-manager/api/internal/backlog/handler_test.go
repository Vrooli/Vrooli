package backlog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/testutil"
)

type backlogListResponse struct {
	Items []BacklogItem `json:"items"`
}

type backlogItemResponse struct {
	Item BacklogItem `json:"item"`
}

type backlogFilesResponse struct {
	Files []BacklogFile `json:"files"`
}

type backlogFileResponse struct {
	File BacklogFile `json:"file"`
}

type backlogQueueResponse struct {
	Item                BacklogItem `json:"item"`
	TaskID              string      `json:"task_id"`
	RunID               string      `json:"run_id"`
	BaseURL             string      `json:"base_url"`
	Created             string      `json:"created"`
	DryRun              bool        `json:"dry_run"`
	Queued              bool        `json:"queued"`
	Message             string      `json:"message"`
	BlockingReasons     []string    `json:"blocking_reasons"`
	UnansweredQuestions int         `json:"unanswered_questions"`
	PendingSuggestions  int         `json:"pending_suggestions"`
}

type processPreflightEnvelope struct {
	Item      BacklogItem    `json:"item"`
	Preflight map[string]any `json:"preflight"`
}

// disableAutoWorkshopSettings writes a settings.json that disables all auto-workshop
// behavior. Tests that specifically test auto-workshop should write their own settings.
func disableAutoWorkshopSettings(t *testing.T, rootDir string) {
	t.Helper()
	t.Setenv("SCENARIO_ROOT", rootDir)
	testutil.WriteJSONFile(t, filepath.Join(rootDir, ".vrooli", "settings.json"), map[string]any{
		"theme":                       "dark",
		"default_mode":                "manual",
		"max_auto_rounds":             10,
		"auto_initialize_workshop":    false,
		"auto_advance_workshop":       false,
		"auto_cascade_workshop":       false,
		"agent_max_turns":             60,
		"agent_timeout_seconds":       900,
		"agent_requires_approval":     true,
		"search_debounce_ms":          300,
		"toast_duration_ms":           5000,
		"confirm_destructive_actions": true,
	})
}

// setupTestHandler creates a handler with a temporary root directory.
// Auto-workshop is disabled by default to prevent goroutine leaks in tests
// that don't provide a mock agent.
func setupTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	disableAutoWorkshopSettings(t, rootDir)
	return NewHandler(rootDir), rootDir
}

func setupTestHandlerWithAgent(t *testing.T, agent agentmanager.Service) (*Handler, string) {
	t.Helper()
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	disableAutoWorkshopSettings(t, rootDir)
	return NewHandlerWithClients(rootDir, agent, &promptmanager.MockClient{Result: "test prompt"}), rootDir
}

// createTestItem creates a test backlog item in the specified kind directory.
func createTestItem(t *testing.T, rootDir string, kind BacklogKind, item BacklogItem) {
	t.Helper()
	item.Kind = kind
	itemDir := filepath.Join(rootDir, backlogKindDirs[kind], item.Name)
	testutil.WriteJSONFile(t, filepath.Join(itemDir, "spec.json"), item)
}

// createReadyTestItem creates a test backlog item that passes workshop readiness preflight
// by writing both spec.json and plan.md (plan exists with no workshop rounds = manually created plan).
func createReadyTestItem(t *testing.T, rootDir string, kind BacklogKind, item BacklogItem) {
	t.Helper()
	createTestItem(t, rootDir, kind, item)
	itemDir := filepath.Join(rootDir, backlogKindDirs[kind], item.Name)
	testutil.WriteFile(t, filepath.Join(itemDir, "plan.md"), "# Plan\nManually created plan for testing.")
}

func TestList_Empty(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/backlog", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogListResponse](t, w)
	if len(resp.Items) != 0 {
		t.Errorf("expected empty list, got %d items", len(resp.Items))
	}
}

func TestList_WithItems(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item1 := BacklogItem{
		Name:        "idea-one",
		Title:       "Idea One",
		Description: "First idea",
		Status:      StatusBacklog,
		Priority:    1,
		Tags:        []string{"test"},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	item2 := BacklogItem{
		Name:        "fix-two",
		Title:       "Fix Two",
		Description: "Second item",
		Status:      StatusReady,
		Priority:    2,
		Tags:        []string{"fix"},
		Created:     "2026-01-27T00:00:00Z",
		Updated:     "2026-01-27T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item1)
	createTestItem(t, rootDir, KindFix, item2)

	req := httptest.NewRequest("GET", "/api/v1/backlog", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogListResponse](t, w)
	if len(resp.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].Priority > resp.Items[1].Priority {
		t.Error("items should be sorted by priority ascending")
	}
}

func TestList_ExcludesArchivedByDefault(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	active := BacklogItem{
		Name:        "active-item",
		Title:       "Active Item",
		Description: "An active item",
		Status:      StatusBacklog,
		Priority:    1,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	archived := BacklogItem{
		Name:        "archived-item",
		Title:       "Archived Item",
		Description: "An archived item",
		Status:      StatusArchived,
		Priority:    2,
		Tags:        []string{},
		Created:     "2026-01-27T00:00:00Z",
		Updated:     "2026-01-27T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, active)
	createTestItem(t, rootDir, KindIdea, archived)

	// Default: no status param → archived excluded
	req := httptest.NewRequest("GET", "/api/v1/backlog", nil)
	w := httptest.NewRecorder()
	h.List(w, req)
	testutil.AssertStatusOK(t, w)
	resp := testutil.DecodeJSON[backlogListResponse](t, w)
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 non-archived item, got %d", len(resp.Items))
	}
	if resp.Items[0].Name != "active-item" {
		t.Errorf("expected active-item, got %s", resp.Items[0].Name)
	}

	// status=all → everything returned
	req = httptest.NewRequest("GET", "/api/v1/backlog?status=all", nil)
	w = httptest.NewRecorder()
	h.List(w, req)
	testutil.AssertStatusOK(t, w)
	resp = testutil.DecodeJSON[backlogListResponse](t, w)
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items with status=all, got %d", len(resp.Items))
	}

	// status=archived → only archived
	req = httptest.NewRequest("GET", "/api/v1/backlog?status=archived", nil)
	w = httptest.NewRecorder()
	h.List(w, req)
	testutil.AssertStatusOK(t, w)
	resp = testutil.DecodeJSON[backlogListResponse](t, w)
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 archived item, got %d", len(resp.Items))
	}
	if resp.Items[0].Name != "archived-item" {
		t.Errorf("expected archived-item, got %s", resp.Items[0].Name)
	}
}

func TestGet_Found(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:        "get-test",
		Title:       "Get Test",
		Description: "Test get endpoint",
		Status:      StatusBacklog,
		Priority:    1,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest("GET", "/api/v1/backlog/idea/get-test", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "get-test"})
	w := httptest.NewRecorder()

	h.Get(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	result := resp.Item
	if result.Name != "get-test" {
		t.Errorf("expected name 'get-test', got '%s'", result.Name)
	}
	if result.Kind != KindIdea {
		t.Errorf("expected kind 'idea', got '%s'", result.Kind)
	}
}

func TestGet_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/backlog/idea/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "nonexistent"})
	w := httptest.NewRecorder()

	h.Get(w, req)

	testutil.AssertStatusNotFound(t, w)
}

func TestCreate_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	payload := map[string]any{
		"name":        "New Test Idea",
		"title":       "New Test Idea",
		"description": "A new test idea",
		"priority":    3,
		"tags":        []string{"new", "test"},
		"kind":        "idea",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	result := resp.Item

	if result.Name != "new-test-idea" {
		t.Errorf("expected sanitized name 'new-test-idea', got '%s'", result.Name)
	}
	if result.Status != StatusBacklog {
		t.Errorf("expected status 'backlog', got '%s'", result.Status)
	}
	if result.Kind != KindIdea {
		t.Errorf("expected kind 'idea', got '%s'", result.Kind)
	}

	specPath := filepath.Join(rootDir, "ideas", "new-test-idea", "spec.json")
	testutil.AssertFileExists(t, specPath)
}

func TestCreate_RejectsUnknownField(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	req := httptest.NewRequest("POST", "/api/v1/backlog", strings.NewReader(`{
		"name": "new-test-idea",
		"title": "New Test Idea",
		"kind": "idea",
		"scope": "scenarios/swarm-manager"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), "invalid request body") {
		t.Fatalf("expected invalid request body error, got: %s", w.Body.String())
	}

	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "new-test-idea", "spec.json"))
}

func TestUpdate_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:        "update-test",
		Title:       "Update Test",
		Description: "Original",
		Status:      StatusBacklog,
		Priority:    5,
		Tags:        []string{"old"},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	payload := map[string]any{
		"title":       "Updated Title",
		"description": "Updated description",
		"status":      "ready",
		"priority":    2,
		"tags":        []string{"new"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", "/api/v1/backlog/idea/update-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "update-test"})
	w := httptest.NewRecorder()

	h.Update(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Title != "Updated Title" {
		t.Errorf("expected updated title, got '%s'", resp.Item.Title)
	}

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "update-test", "spec.json"))
	if saved.Status != StatusReady {
		t.Errorf("expected status ready, got '%s'", saved.Status)
	}
}

func TestUpdate_RejectsUnknownField(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:        "update-test",
		Title:       "Update Test",
		Description: "Original",
		Status:      StatusBacklog,
		Priority:    5,
		Tags:        []string{"old"},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	})

	req := httptest.NewRequest("PUT", "/api/v1/backlog/idea/update-test", strings.NewReader(`{
		"scope": "scenarios/swarm-manager"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "update-test"})
	w := httptest.NewRecorder()

	h.Update(w, req)

	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), "invalid request body") {
		t.Fatalf("expected invalid request body error, got: %s", w.Body.String())
	}
}

func TestDelete_Idempotent(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:        "delete-test",
		Title:       "Delete Test",
		Description: "To delete",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/delete-test", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "delete-test"})
	w := httptest.NewRecorder()

	h.Delete(w, req)
	testutil.AssertStatus(t, w, http.StatusNoContent)

	req2 := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/delete-test", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"kind": "idea", "name": "delete-test"})
	w2 := httptest.NewRecorder()

	h.Delete(w2, req2)
	testutil.AssertStatus(t, w2, http.StatusNoContent)
}

func TestListFiles_And_GetFileContent(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:        "files-test",
		Title:       "Files Test",
		Description: "",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "files-test", "notes.md"), "hello")

	req := httptest.NewRequest("GET", "/api/v1/backlog/idea/files-test/files", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "files-test"})
	w := httptest.NewRecorder()

	h.ListFiles(w, req)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogFilesResponse](t, w)
	if len(resp.Files) == 0 {
		t.Fatalf("expected files")
	}

	contentReq := httptest.NewRequest("GET", "/api/v1/backlog/idea/files-test/files/notes.md", nil)
	contentReq = mux.SetURLVars(contentReq, map[string]string{"kind": "idea", "name": "files-test", "filepath": "notes.md"})
	contentRec := httptest.NewRecorder()

	h.GetFileContent(contentRec, contentReq)
	testutil.AssertStatusOK(t, contentRec)
	if strings.TrimSpace(contentRec.Body.String()) != "hello" {
		t.Errorf("expected file content")
	}
}

func TestUploadFile_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:        "upload-test",
		Title:       "Upload Test",
		Description: "",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "test-upload.txt")
	_, _ = part.Write([]byte("test content"))
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/upload-test/files", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "upload-test"})
	w := httptest.NewRecorder()

	h.UploadFile(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogFileResponse](t, w)
	if resp.File.Name != "test-upload.txt" {
		t.Errorf("expected uploaded file name")
	}

	filePath := filepath.Join(rootDir, "ideas", "upload-test", "test-upload.txt")
	testutil.AssertFileExists(t, filePath)
}

func TestOperateFile_Rename_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:     "rename-file-test",
		Title:    "Rename File Test",
		Status:   StatusBacklog,
		Priority: 3,
		Created:  "2026-01-28T00:00:00Z",
		Updated:  "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)
	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "rename-file-test", "notes.md"), "hello")

	reqBody := bytes.NewBufferString(`{"operation":"rename","source_path":"notes.md","destination_path":"notes-renamed.md"}`)
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/rename-file-test/files", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "rename-file-test"})
	w := httptest.NewRecorder()

	h.OperateFile(w, req)
	testutil.AssertStatusOK(t, w)

	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "rename-file-test", "notes-renamed.md"))
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "rename-file-test", "notes.md"))
}

func TestOperateFile_Delete_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:     "delete-file-test",
		Title:    "Delete File Test",
		Status:   StatusBacklog,
		Priority: 3,
		Created:  "2026-01-28T00:00:00Z",
		Updated:  "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)
	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "delete-file-test", "notes.md"), "hello")

	reqBody := bytes.NewBufferString(`{"operation":"delete","source_path":"notes.md"}`)
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/delete-file-test/files", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "delete-file-test"})
	w := httptest.NewRecorder()

	h.OperateFile(w, req)
	testutil.AssertStatusOK(t, w)
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "delete-file-test", "notes.md"))
}

func TestOperateFile_CopyDirectory_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:     "copy-dir-test",
		Title:    "Copy Dir Test",
		Status:   StatusBacklog,
		Priority: 3,
		Created:  "2026-01-28T00:00:00Z",
		Updated:  "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)
	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "copy-dir-test", "docs", "a.txt"), "A")
	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "copy-dir-test", "docs", "nested", "b.txt"), "B")

	reqBody := bytes.NewBufferString(`{"operation":"copy","source_path":"docs","destination_path":"docs-copy"}`)
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/copy-dir-test/files", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "copy-dir-test"})
	w := httptest.NewRecorder()

	h.OperateFile(w, req)
	testutil.AssertStatusOK(t, w)
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "copy-dir-test", "docs-copy", "a.txt"))
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "copy-dir-test", "docs-copy", "nested", "b.txt"))
}

func TestOperateFile_ProtectsSpecJSON(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:     "protect-spec-test",
		Title:    "Protect Spec Test",
		Status:   StatusBacklog,
		Priority: 3,
		Created:  "2026-01-28T00:00:00Z",
		Updated:  "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	reqBody := bytes.NewBufferString(`{"operation":"delete","source_path":"spec.json"}`)
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/protect-spec-test/files", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "protect-spec-test"})
	w := httptest.NewRecorder()

	h.OperateFile(w, req)
	testutil.AssertStatus(t, w, http.StatusForbidden)
	testutil.AssertFileExists(t, filepath.Join(rootDir, "ideas", "protect-spec-test", "spec.json"))
}

func TestOperateFile_DestinationConflict(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:     "conflict-test",
		Title:    "Conflict Test",
		Status:   StatusBacklog,
		Priority: 3,
		Created:  "2026-01-28T00:00:00Z",
		Updated:  "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)
	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "conflict-test", "one.txt"), "1")
	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "conflict-test", "two.txt"), "2")

	reqBody := bytes.NewBufferString(`{"operation":"move","source_path":"one.txt","destination_path":"two.txt"}`)
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/conflict-test/files", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "conflict-test"})
	w := httptest.NewRecorder()

	h.OperateFile(w, req)
	testutil.AssertStatus(t, w, http.StatusConflict)

	content, err := os.ReadFile(filepath.Join(rootDir, "ideas", "conflict-test", "two.txt"))
	if err != nil {
		t.Fatalf("read destination file: %v", err)
	}
	if strings.TrimSpace(string(content)) != "2" {
		t.Fatalf("expected destination to remain unchanged")
	}
}

func TestQueue_DefaultIsPreviewAndDoesNotSpawn(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:        "queue-test",
		Title:       "Queue Test",
		Description: "",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/queue-test/queue", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "queue-test"})
	w := httptest.NewRecorder()

	h.Queue(w, req)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogQueueResponse](t, w)
	if resp.Item.Status != StatusBacklog {
		t.Errorf("expected unchanged backlog status, got %s", resp.Item.Status)
	}
	if !resp.DryRun || resp.Queued {
		t.Errorf("expected dry_run preview response, got dry_run=%t queued=%t", resp.DryRun, resp.Queued)
	}
	if agent.lastReq != nil {
		t.Errorf("expected no agent spawn for preview mode")
	}
}

func TestQueue_ManualMode_DoesNotSpawnImmediately(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:        "queue-manual-test",
		Title:       "Queue Manual Test",
		Description: "",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createReadyTestItem(t, rootDir, KindIdea, item)

	reqBody := bytes.NewBufferString(`{"mode":"manual","operation":"generator","confirm":true}`)
	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/queue-manual-test/queue", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "queue-manual-test"})
	w := httptest.NewRecorder()

	h.Queue(w, req)
	testutil.AssertStatus(t, w, http.StatusAccepted)

	resp := testutil.DecodeJSON[backlogQueueResponse](t, w)
	if resp.Item.Status != StatusQueued {
		t.Errorf("expected queued status, got %s", resp.Item.Status)
	}
	if resp.TaskID != "" || resp.RunID != "" {
		t.Errorf("expected no run metadata for manual mode, got task=%q run=%q", resp.TaskID, resp.RunID)
	}
	if agent.lastReq != nil {
		t.Errorf("expected no agent spawn in manual mode")
	}
}

func TestQueue_ScheduledMode_DoesNotSpawnImmediately(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:        "queue-scheduled-test",
		Title:       "Queue Scheduled Test",
		Description: "",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createReadyTestItem(t, rootDir, KindIdea, item)

	reqBody := bytes.NewBufferString(`{"mode":"scheduled","delay_seconds":60,"confirm":true}`)
	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/queue-scheduled-test/queue", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "queue-scheduled-test"})
	w := httptest.NewRecorder()

	h.Queue(w, req)
	testutil.AssertStatus(t, w, http.StatusAccepted)

	resp := testutil.DecodeJSON[backlogQueueResponse](t, w)
	if resp.Item.Status != StatusQueued {
		t.Errorf("expected queued status, got %s", resp.Item.Status)
	}
	if resp.TaskID != "" || resp.RunID != "" {
		t.Errorf("expected no run metadata for scheduled mode, got task=%q run=%q", resp.TaskID, resp.RunID)
	}
	if agent.lastReq != nil {
		t.Errorf("expected no agent spawn in scheduled mode")
	}
}

func TestQueue_AllowsArchivedIdea(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:        "archived-idea",
		Title:       "Archived Idea",
		Description: "",
		Status:      StatusArchived,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
		Kind:        KindIdea,
	}
	createReadyTestItem(t, rootDir, KindIdea, item)

	reqBody := bytes.NewBufferString(`{"mode":"manual","operation":"generator","confirm":true}`)
	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/archived-idea/queue", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "archived-idea"})
	w := httptest.NewRecorder()

	h.Queue(w, req)
	testutil.AssertStatus(t, w, http.StatusAccepted)
}

func TestProcessPreflight_BlocksMissingWorkshopReadiness(t *testing.T) {
	h, rootDir := setupTestHandlerWithAgent(t, &mockAgentService{})

	item := BacklogItem{
		Name:        "archived-preflight",
		Title:       "[Archived] web-console",
		Description: "",
		Status:      StatusArchived,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
		Kind:        KindIdea,
	}
	createTestItem(t, rootDir, KindIdea, item)
	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "archived-preflight", "spec.json"), `{
  "name":"archived-preflight",
  "title":"[Archived] web-console",
  "description":"",
  "status":"archived",
  "priority":3,
  "tags":[],
  "created":"2026-01-28T00:00:00Z",
  "updated":"2026-01-28T00:00:00Z",
  "kind":"idea",
  "sourceScenarioName":"web-console"
}`)

	req := httptest.NewRequest("GET", "/api/v1/backlog/idea/archived-preflight/process-preflight", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "archived-preflight"})
	w := httptest.NewRecorder()

	h.ProcessPreflight(w, req)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[processPreflightEnvelope](t, w)
	if ready, ok := resp.Preflight["ready"].(bool); !ok || ready {
		t.Fatalf("expected preflight ready=false, got %#v", resp.Preflight["ready"])
	}
	if target, _ := resp.Preflight["resolved_target_scenario_id"].(string); target != "web-console" {
		t.Fatalf("expected resolved target web-console, got %q", target)
	}
}

func TestQueue_BlocksWhenProcessPreflightFails(t *testing.T) {
	h, rootDir := setupTestHandlerWithAgent(t, &mockAgentService{})

	item := BacklogItem{
		Name:        "blocked-queue",
		Title:       "[Archived] web-console",
		Description: "",
		Status:      StatusArchived,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
		Kind:        KindIdea,
	}
	// No plan.md or workshop rounds — preflight will block.
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/blocked-queue/queue", bytes.NewBufferString(`{"mode":"manual","confirm":true}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "blocked-queue"})
	w := httptest.NewRecorder()

	h.Queue(w, req)
	testutil.AssertStatusOK(t, w)
	if !strings.Contains(w.Body.String(), "blocking_reasons") {
		t.Fatalf("expected blocking reasons in response, got %s", w.Body.String())
	}
}

func TestQueue_DryRun_DoesNotMutateState(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:        "queue-dry-run",
		Title:       "Queue Dry Run",
		Description: "",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/queue-dry-run/queue", bytes.NewBufferString(`{"mode":"yolo"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dry-Run", "true")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "queue-dry-run"})
	w := httptest.NewRecorder()

	h.Queue(w, req)
	testutil.AssertStatus(t, w, http.StatusOK)

	resp := testutil.DecodeJSON[map[string]any](t, w)
	if dryRun, ok := resp["dry_run"].(bool); !ok || !dryRun {
		t.Fatalf("expected dry_run=true, got %#v", resp["dry_run"])
	}
	if agent.lastReq != nil {
		t.Fatalf("expected no agent spawn during dry-run")
	}
	updated := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "queue-dry-run", "spec.json"))
	if updated.Status != StatusBacklog {
		t.Fatalf("expected status backlog after dry-run, got %s", updated.Status)
	}
}

func TestQueue_BlocksOnPendingSuggestionsUnlessForced(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:        "pending-suggest",
		Title:       "Pending Suggest",
		Description: "",
		Status:      StatusReady,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
		Kind:        KindIdea,
	}
	createReadyTestItem(t, rootDir, KindIdea, item)
	// Add a workshop round with a pending decision to trigger blocking.
	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "pending-suggest", "workshop", "round-001.json"), `{
  "round": 1,
  "generated_at": "2026-01-01T00:00:00Z",
  "readiness": {"problem_clarity": 3, "scope_defined": 3, "approach_solid": 3, "testable": 3, "risk_awareness": 3},
  "items": [
    {"id": "d1", "type": "decision", "topic": "Use caching?", "options": [{"key": "A", "label": "Yes", "rationale": "Faster"}, {"key": "B", "label": "No", "rationale": "Simpler"}], "selected": null}
  ]
}`)

	blockedReq := httptest.NewRequest("POST", "/api/v1/backlog/idea/pending-suggest/queue", bytes.NewBufferString(`{"mode":"manual","confirm":true}`))
	blockedReq.Header.Set("Content-Type", "application/json")
	blockedReq = mux.SetURLVars(blockedReq, map[string]string{"kind": "idea", "name": "pending-suggest"})
	blockedW := httptest.NewRecorder()
	h.Queue(blockedW, blockedReq)
	testutil.AssertStatusOK(t, blockedW)

	forcedReq := httptest.NewRequest("POST", "/api/v1/backlog/idea/pending-suggest/queue", bytes.NewBufferString(`{"mode":"manual","confirm":true,"force":true}`))
	forcedReq.Header.Set("Content-Type", "application/json")
	forcedReq = mux.SetURLVars(forcedReq, map[string]string{"kind": "idea", "name": "pending-suggest"})
	forcedW := httptest.NewRecorder()
	h.Queue(forcedW, forcedReq)
	testutil.AssertStatus(t, forcedW, http.StatusAccepted)
}

func TestResearch_SpawnsAgent(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:        "research-test",
		Title:       "Research Test",
		Description: "",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	payload := map[string]any{"mode": "workshop", "prompt": "focus"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/research-test/research", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "research-test"})
	w := httptest.NewRecorder()

	h.Research(w, req)
	testutil.AssertStatus(t, w, http.StatusCreated)

	if agent.lastReq == nil || agent.lastReq.Purpose != "research" {
		t.Errorf("expected agent spawn for research")
	}
}

func TestResearch_DryRun_DoesNotSpawnAgent(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:        "research-dry-run",
		Title:       "Research Dry Run",
		Description: "",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	payload := map[string]any{"mode": "clarify"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/research-dry-run/research", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dry-Run", "true")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "research-dry-run"})
	w := httptest.NewRecorder()

	h.Research(w, req)
	testutil.AssertStatus(t, w, http.StatusOK)

	resp := testutil.DecodeJSON[map[string]any](t, w)
	if dryRun, ok := resp["dry_run"].(bool); !ok || !dryRun {
		t.Fatalf("expected dry_run=true, got %#v", resp["dry_run"])
	}
	if agent.lastReq != nil {
		t.Fatalf("expected no agent spawn during dry-run")
	}
}

func TestSaveItem_PreservesUnknownSpecFields(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	raw := map[string]any{
		"name":            "metadata-keep",
		"title":           "Metadata Keep",
		"description":     "desc",
		"status":          "archived",
		"priority":        5,
		"tags":            []string{"x"},
		"created":         "2026-01-28T00:00:00Z",
		"updated":         "2026-01-28T00:00:00Z",
		"kind":            "idea",
		"archiveReason":   "scenario deleted with archive=true",
		"sourceScenario":  "web-console",
		"preservedFiles":  []string{"PRD.md"},
		"archivedByHuman": true,
	}
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", "metadata-keep", "spec.json"), raw)

	item := BacklogItem{
		Name:        "metadata-keep",
		Kind:        KindIdea,
		Title:       "Metadata Keep Updated",
		Description: "updated",
		Status:      StatusArchived,
		Priority:    6,
		Tags:        []string{"y"},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-29T00:00:00Z",
	}
	if err := h.store.SaveItem(item); err != nil {
		t.Fatalf("saveItem error: %v", err)
	}

	var persisted map[string]any
	data, err := os.ReadFile(filepath.Join(rootDir, "ideas", "metadata-keep", "spec.json"))
	if err != nil {
		t.Fatalf("read spec.json: %v", err)
	}
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("unmarshal spec.json: %v", err)
	}

	if persisted["archiveReason"] != "scenario deleted with archive=true" {
		t.Fatalf("expected archiveReason preserved, got %#v", persisted["archiveReason"])
	}
	if persisted["sourceScenario"] != "web-console" {
		t.Fatalf("expected sourceScenario preserved, got %#v", persisted["sourceScenario"])
	}
}

func TestConvert_MovesFolder(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:           "convert-test",
		Title:          "Convert Test",
		Description:    "",
		Status:         StatusBacklog,
		Priority:       3,
		Tags:           []string{},
		Created:        "2026-01-28T00:00:00Z",
		Updated:        "2026-01-28T00:00:00Z",
		ResearchTarget: "idea",
	}
	createTestItem(t, rootDir, KindResearch, item)

	payload := map[string]any{"target_kind": "fix"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/backlog/research/convert-test/convert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "research", "name": "convert-test"})
	w := httptest.NewRecorder()

	h.Convert(w, req)
	testutil.AssertStatusOK(t, w)

	movedPath := filepath.Join(rootDir, "fix", "convert-test", "spec.json")
	testutil.AssertFileExists(t, movedPath)
}

type mockAgentService struct {
	lastReq  *agentmanager.BacklogSpawnRequest
	result   agentmanager.RunResult
	err      error
	spawnedC chan struct{} // closed on SpawnBacklog call (optional)
}

func (m *mockAgentService) IsEnabled() bool                    { return true }
func (m *mockAgentService) IsAvailable(_ context.Context) bool { return true }
func (m *mockAgentService) ResolveURL(_ context.Context) (string, error) {
	return "http://agent", nil
}
func (m *mockAgentService) GetProfileID() string { return "" }
func (m *mockAgentService) SpawnBacklog(_ context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	m.lastReq = &req
	if m.spawnedC != nil {
		select {
		case <-m.spawnedC:
		default:
			close(m.spawnedC)
		}
	}
	return m.result, m.err
}

func (m *mockAgentService) SpawnResearch(_ context.Context, _ agentmanager.ResearchSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, nil
}

func (m *mockAgentService) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	return agentmanager.RunState{}, nil
}
func (m *mockAgentService) StopRun(_ context.Context, _ string) error { return nil }

func TestGetFileContent_PathIsDirectory(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:    "dir-test",
		Title:   "Dir Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	})

	// Create a subdirectory inside the item folder
	subDir := filepath.Join(rootDir, "ideas", "dir-test", "clarify")
	testutil.MakeDir(t, subDir)

	req := httptest.NewRequest("GET", "/api/v1/backlog/idea/dir-test/files/clarify", nil)
	req = mux.SetURLVars(req, map[string]string{
		"kind":     "idea",
		"name":     "dir-test",
		"filepath": "clarify",
	})
	w := httptest.NewRecorder()

	h.GetFileContent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUploadFile_ConflictWithDirectory(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:    "upload-dir-test",
		Title:   "Upload Dir Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	})

	// Pre-create a directory at the target path
	targetDir := filepath.Join(rootDir, "ideas", "upload-dir-test", "questions.json")
	testutil.MakeDir(t, targetDir)

	// Build multipart upload
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "questions.json")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, _ = part.Write([]byte(`{"test":true}`))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/upload-dir-test/files", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = mux.SetURLVars(req, map[string]string{
		"kind": "idea",
		"name": "upload-dir-test",
	})
	w := httptest.NewRecorder()

	h.UploadFile(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestUpdate_FailedStatus_Accepted(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:        "failed-test",
		Title:       "Failed Test",
		Description: "Desc",
		Status:      StatusBacklog,
		Priority:    5,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	payload := map[string]any{
		"title":       "Failed Test",
		"description": "Desc",
		"status":      "failed",
		"priority":    5,
		"tags":        []string{},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", "/api/v1/backlog/idea/failed-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "failed-test"})
	w := httptest.NewRecorder()

	h.Update(w, req)

	testutil.AssertStatusOK(t, w)

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "failed-test", "spec.json"))
	if saved.Status != StatusFailed {
		t.Errorf("expected status failed, got '%s'", saved.Status)
	}
}

func TestUpdate_QueuedStatus_Rejected(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:        "queued-reject",
		Title:       "Queued Reject",
		Description: "Desc",
		Status:      StatusBacklog,
		Priority:    5,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	payload := map[string]any{
		"title":       "Queued Reject",
		"description": "Desc",
		"status":      "queued",
		"priority":    5,
		"tags":        []string{},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", "/api/v1/backlog/idea/queued-reject", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "queued-reject"})
	w := httptest.NewRecorder()

	h.Update(w, req)

	testutil.AssertStatus(t, w, http.StatusBadRequest)
}

func TestUpdate_InProgressStatus_Rejected(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:        "inprog-reject",
		Title:       "InProgress Reject",
		Description: "Desc",
		Status:      StatusBacklog,
		Priority:    5,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	payload := map[string]any{
		"title":       "InProgress Reject",
		"description": "Desc",
		"status":      "in_progress",
		"priority":    5,
		"tags":        []string{},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", "/api/v1/backlog/idea/inprog-reject", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "inprog-reject"})
	w := httptest.NewRecorder()

	h.Update(w, req)

	testutil.AssertStatus(t, w, http.StatusBadRequest)
}

func TestUpdate_ChangeDependsOn(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Create items A and B.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "alpha", Title: "Alpha", Status: StatusBacklog, Priority: 5,
	})
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "beta", Title: "Beta", Status: StatusBacklog, Priority: 5,
	})

	// Update B to depend on A.
	payload := map[string]any{
		"title":       "Beta",
		"description": "",
		"status":      "backlog",
		"priority":    5,
		"tags":        []string{},
		"depends_on":  []string{"idea/alpha"},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PUT", "/api/v1/backlog/idea/beta", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "beta"})
	w := httptest.NewRecorder()
	h.Update(w, req)
	testutil.AssertStatus(t, w, http.StatusOK)

	// Verify dependency stored.
	item, err := h.store.LoadItem(KindIdea, "beta")
	if err != nil {
		t.Fatalf("LoadItem: %v", err)
	}
	if len(item.DependsOn) != 1 || item.DependsOn[0] != "idea/alpha" {
		t.Errorf("expected depends_on=['idea/alpha'], got %v", item.DependsOn)
	}

	// Update with different dependencies.
	payload["depends_on"] = []string{"idea/alpha"}
	body, _ = json.Marshal(payload)
	req = httptest.NewRequest("PUT", "/api/v1/backlog/idea/beta", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "beta"})
	w = httptest.NewRecorder()
	h.Update(w, req)
	testutil.AssertStatus(t, w, http.StatusOK)

	item, err = h.store.LoadItem(KindIdea, "beta")
	if err != nil {
		t.Fatalf("LoadItem after update: %v", err)
	}
	if len(item.DependsOn) != 1 || item.DependsOn[0] != "idea/alpha" {
		t.Errorf("expected depends_on=['idea/alpha'] preserved, got %v", item.DependsOn)
	}
}

func TestQueue_InvalidMode(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "qmode-test", Title: "Queue Mode Test", Status: StatusReady, Priority: 5,
	})

	// Proto validation rejects invalid modes via buf.validate constraint.
	payload := map[string]any{"mode": "invalid_mode"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/qmode-test/queue", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "qmode-test"})
	w := httptest.NewRecorder()
	h.Queue(w, req)

	testutil.AssertStatus(t, w, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// Auto-initialize workshop on Create
// ---------------------------------------------------------------------------

func TestCreate_AutoInitializesWorkshop(t *testing.T) {
	spawned := make(chan struct{})
	agent := &mockAgentService{
		result:   agentmanager.RunResult{RunID: "run-auto", TaskID: "task-auto"},
		spawnedC: spawned,
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Re-enable auto-initialize for this test.
	testutil.WriteJSONFile(t, filepath.Join(rootDir, ".vrooli", "settings.json"), map[string]any{
		"theme":                       "dark",
		"default_mode":                "manual",
		"max_auto_rounds":             10,
		"auto_initialize_workshop":    true,
		"auto_advance_workshop":       true,
		"auto_cascade_workshop":       true,
		"agent_max_turns":             60,
		"agent_timeout_seconds":       900,
		"agent_requires_approval":     true,
		"search_debounce_ms":          300,
		"toast_duration_ms":           5000,
		"confirm_destructive_actions": true,
	})

	payload := map[string]any{
		"name":  "auto-init-test",
		"title": "Auto Init Test",
		"kind":  "idea",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Create(w, req)

	testutil.AssertStatusCreated(t, w)

	// Wait for the background goroutine to call SpawnBacklog.
	select {
	case <-spawned:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for auto-init spawn")
	}

	if agent.lastReq == nil {
		t.Fatal("expected agent spawn to be called")
	}
	if agent.lastReq.Name != "auto-init-test" {
		t.Errorf("expected spawn for 'auto-init-test', got %q", agent.lastReq.Name)
	}
}

func TestCreate_AutoInitializeDisabledViaSetting(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Disable auto-initialize via settings.
	t.Setenv("SCENARIO_ROOT", rootDir)
	testutil.WriteJSONFile(t, filepath.Join(rootDir, ".vrooli", "settings.json"), map[string]any{
		"theme":                       "dark",
		"default_mode":                "manual",
		"max_auto_rounds":             10,
		"auto_initialize_workshop":    false,
		"auto_advance_workshop":       true,
		"auto_cascade_workshop":       true,
		"agent_max_turns":             60,
		"agent_timeout_seconds":       900,
		"agent_requires_approval":     true,
		"search_debounce_ms":          300,
		"toast_duration_ms":           5000,
		"confirm_destructive_actions": true,
	})

	payload := map[string]any{
		"name":  "no-auto-test",
		"title": "No Auto Test",
		"kind":  "idea",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Create(w, req)

	testutil.AssertStatusCreated(t, w)

	// Give a brief window for any goroutine to fire (it shouldn't).
	time.Sleep(100 * time.Millisecond)

	if agent.lastReq != nil {
		t.Error("expected NO agent spawn when auto_initialize_workshop is false")
	}
}

func TestCreate_AutoInit_AgentDown_StillCreates(t *testing.T) {
	spawned := make(chan struct{})
	agent := &mockAgentService{
		err:      fmt.Errorf("agent down"),
		spawnedC: spawned,
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Re-enable auto-initialize to test agent-down resilience.
	testutil.WriteJSONFile(t, filepath.Join(rootDir, ".vrooli", "settings.json"), map[string]any{
		"theme":                       "dark",
		"default_mode":                "manual",
		"max_auto_rounds":             10,
		"auto_initialize_workshop":    true,
		"auto_advance_workshop":       true,
		"auto_cascade_workshop":       true,
		"agent_max_turns":             60,
		"agent_timeout_seconds":       900,
		"agent_requires_approval":     true,
		"search_debounce_ms":          300,
		"toast_duration_ms":           5000,
		"confirm_destructive_actions": true,
	})

	payload := map[string]any{
		"name":  "agent-down-test",
		"title": "Agent Down Test",
		"kind":  "fix",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Create(w, req)

	// Item should be created successfully regardless of agent error.
	testutil.AssertStatusCreated(t, w)

	specPath := filepath.Join(rootDir, "fix", "agent-down-test", "spec.json")
	testutil.AssertFileExists(t, specPath)

	// Wait for the goroutine to attempt spawn (and fail).
	select {
	case <-spawned:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for auto-init attempt")
	}
}

func TestCreate_WithEffort(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	payload := map[string]any{
		"name":     "effort-test",
		"title":    "Effort Test",
		"kind":     "idea",
		"effort":   "L",
		"priority": 3,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Effort != "L" {
		t.Errorf("expected effort 'L', got %q", resp.Item.Effort)
	}

	// Verify persisted to disk.
	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "effort-test", "spec.json"))
	if saved.Effort != "L" {
		t.Errorf("expected saved effort 'L', got %q", saved.Effort)
	}
}

func TestCreate_EffortNormalizesCase(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := map[string]any{
		"name":   "effort-case-test",
		"title":  "Effort Case Test",
		"kind":   "fix",
		"effort": "xl",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Effort != "XL" {
		t.Errorf("expected effort 'XL', got %q", resp.Item.Effort)
	}
}

func TestCreate_InvalidEffort(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := map[string]any{
		"name":   "bad-effort",
		"title":  "Bad Effort",
		"kind":   "idea",
		"effort": "HUGE",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusBadRequest(t, w)
}

func TestCreate_EffortOptional(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := map[string]any{
		"name":  "no-effort",
		"title": "No Effort",
		"kind":  "idea",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Effort != "" {
		t.Errorf("expected empty effort, got %q", resp.Item.Effort)
	}
}

func TestUpdate_WithEffort(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:     "update-effort-test",
		Title:    "Update Effort Test",
		Status:   StatusBacklog,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-01-28T00:00:00Z",
		Updated:  "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	payload := map[string]any{
		"title":    "Update Effort Test",
		"status":   "backlog",
		"priority": 5,
		"tags":     []string{},
		"effort":   "M",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", "/api/v1/backlog/idea/update-effort-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "update-effort-test"})
	w := httptest.NewRecorder()

	h.Update(w, req)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Effort != "M" {
		t.Errorf("expected effort 'M', got %q", resp.Item.Effort)
	}

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "update-effort-test", "spec.json"))
	if saved.Effort != "M" {
		t.Errorf("expected saved effort 'M', got %q", saved.Effort)
	}
}

func TestUpdate_InvalidEffort(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:     "update-bad-effort",
		Title:    "Update Bad Effort",
		Status:   StatusBacklog,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-01-28T00:00:00Z",
		Updated:  "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	payload := map[string]any{
		"title":    "Update Bad Effort",
		"status":   "backlog",
		"priority": 5,
		"tags":     []string{},
		"effort":   "XXXL",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", "/api/v1/backlog/idea/update-bad-effort", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "update-bad-effort"})
	w := httptest.NewRecorder()

	h.Update(w, req)
	testutil.AssertStatusBadRequest(t, w)
}

func TestCreate_WithAcceptanceGlobs(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	payload := map[string]any{
		"name":             "globs-test",
		"title":            "Globs Test",
		"kind":             "fix",
		"acceptance_allow": []string{"api/**", "*.go"},
		"acceptance_deny":  []string{"vendor/**"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if len(resp.Item.AcceptanceAllow) != 2 {
		t.Errorf("expected 2 allow globs, got %d", len(resp.Item.AcceptanceAllow))
	}
	if len(resp.Item.AcceptanceDeny) != 1 {
		t.Errorf("expected 1 deny glob, got %d", len(resp.Item.AcceptanceDeny))
	}

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "fix", "globs-test", "spec.json"))
	if len(saved.AcceptanceAllow) != 2 {
		t.Errorf("expected 2 saved allow globs, got %d", len(saved.AcceptanceAllow))
	}
}

func TestUpdate_Acceptance(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:     "update-acceptance-test",
		Title:    "Update Acceptance Test",
		Status:   StatusBacklog,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-01-28T00:00:00Z",
		Updated:  "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	payload := map[string]any{
		"title":            "Update Acceptance Test",
		"status":           "backlog",
		"priority":         5,
		"tags":             []string{},
		"acceptance_allow": []string{"scenarios/target/src/**"},
		"acceptance_deny":  []string{"scenarios/target/test/**"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", "/api/v1/backlog/idea/update-acceptance-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "update-acceptance-test"})
	w := httptest.NewRecorder()

	h.Update(w, req)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if len(resp.Item.AcceptanceAllow) != 1 || resp.Item.AcceptanceAllow[0] != "scenarios/target/src/**" {
		t.Errorf("expected acceptance_allow ['scenarios/target/src/**'], got %v", resp.Item.AcceptanceAllow)
	}
	if len(resp.Item.AcceptanceDeny) != 1 || resp.Item.AcceptanceDeny[0] != "scenarios/target/test/**" {
		t.Errorf("expected acceptance_deny ['scenarios/target/test/**'], got %v", resp.Item.AcceptanceDeny)
	}

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "update-acceptance-test", "spec.json"))
	if len(saved.AcceptanceAllow) != 1 || saved.AcceptanceAllow[0] != "scenarios/target/src/**" {
		t.Errorf("expected saved acceptance_allow ['scenarios/target/src/**'], got %v", saved.AcceptanceAllow)
	}
}

func TestWorkshopDeleteRound_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name: "ws-delete-test", Title: "WS Delete Test",
		Status: StatusBacklog, Priority: 3,
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	workshopDir := filepath.Join(rootDir, "ideas", "ws-delete-test", "workshop")
	if err := os.MkdirAll(workshopDir, 0o755); err != nil {
		t.Fatalf("mkdir workshop: %v", err)
	}
	for i := 1; i <= 3; i++ {
		content := fmt.Sprintf(`{"round":%d,"generated_at":"2026-01-01T00:00:00Z","readiness":{},"items":[]}`, i)
		testutil.WriteFile(t, filepath.Join(workshopDir, fmt.Sprintf("round-%03d.json", i)), content)
	}

	reqBody := bytes.NewBufferString(`{"round_number":2}`)
	req := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/ws-delete-test/workshop/round", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "ws-delete-test"})
	w := httptest.NewRecorder()

	h.WorkshopDeleteRound(w, req)
	testutil.AssertStatusOK(t, w)

	// Round 2 should be gone, old round 3 should now be round 2.
	testutil.AssertFileNotExists(t, filepath.Join(workshopDir, "round-003.json"))
	testutil.AssertFileExists(t, filepath.Join(workshopDir, "round-001.json"))
	testutil.AssertFileExists(t, filepath.Join(workshopDir, "round-002.json"))

	// Verify the renumbered file has round=2 in its JSON.
	data, err := os.ReadFile(filepath.Join(workshopDir, "round-002.json"))
	if err != nil {
		t.Fatalf("read round-002.json: %v", err)
	}
	if !bytes.Contains(data, []byte(`"round": 2`)) {
		t.Errorf("round-002.json should contain round=2, got: %s", string(data))
	}

	// Verify response.
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["deleted_round"] != float64(2) {
		t.Errorf("expected deleted_round=2, got %v", resp["deleted_round"])
	}
	if resp["remaining_rounds"] != float64(2) {
		t.Errorf("expected remaining_rounds=2, got %v", resp["remaining_rounds"])
	}
}

func TestWorkshopDeleteRound_LastRound(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name: "ws-delete-last", Title: "WS Delete Last",
		Status: StatusBacklog, Priority: 3,
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	workshopDir := filepath.Join(rootDir, "ideas", "ws-delete-last", "workshop")
	if err := os.MkdirAll(workshopDir, 0o755); err != nil {
		t.Fatalf("mkdir workshop: %v", err)
	}
	testutil.WriteFile(t, filepath.Join(workshopDir, "round-001.json"),
		`{"round":1,"generated_at":"2026-01-01T00:00:00Z","readiness":{},"items":[]}`)

	reqBody := bytes.NewBufferString(`{"round_number":1}`)
	req := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/ws-delete-last/workshop/round", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "ws-delete-last"})
	w := httptest.NewRecorder()

	h.WorkshopDeleteRound(w, req)
	testutil.AssertStatusOK(t, w)
	testutil.AssertFileNotExists(t, filepath.Join(workshopDir, "round-001.json"))
}

func TestWorkshopDeleteRound_NotFound(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name: "ws-delete-nf", Title: "WS Delete NF",
		Status: StatusBacklog, Priority: 3,
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	workshopDir := filepath.Join(rootDir, "ideas", "ws-delete-nf", "workshop")
	if err := os.MkdirAll(workshopDir, 0o755); err != nil {
		t.Fatalf("mkdir workshop: %v", err)
	}
	testutil.WriteFile(t, filepath.Join(workshopDir, "round-001.json"),
		`{"round":1,"generated_at":"2026-01-01T00:00:00Z","readiness":{},"items":[]}`)

	reqBody := bytes.NewBufferString(`{"round_number":5}`)
	req := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/ws-delete-nf/workshop/round", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "ws-delete-nf"})
	w := httptest.NewRecorder()

	h.WorkshopDeleteRound(w, req)
	testutil.AssertStatus(t, w, http.StatusNotFound)
}

func TestWorkshopDeleteRound_LockedConflict(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name: "ws-delete-lock", Title: "WS Delete Lock",
		Status: StatusBacklog, Priority: 3,
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	itemDir := filepath.Join(rootDir, "ideas", "ws-delete-lock")
	workshopDir := filepath.Join(itemDir, "workshop")
	if err := os.MkdirAll(workshopDir, 0o755); err != nil {
		t.Fatalf("mkdir workshop: %v", err)
	}
	testutil.WriteFile(t, filepath.Join(workshopDir, "round-001.json"),
		`{"round":1,"generated_at":"2026-01-01T00:00:00Z","readiness":{},"items":[]}`)

	// Create a fresh lock file.
	testutil.WriteFile(t, filepath.Join(itemDir, ".workshop-lock"), time.Now().UTC().Format(time.RFC3339))

	reqBody := bytes.NewBufferString(`{"round_number":1}`)
	req := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/ws-delete-lock/workshop/round", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "ws-delete-lock"})
	w := httptest.NewRecorder()

	h.WorkshopDeleteRound(w, req)
	testutil.AssertStatus(t, w, http.StatusConflict)
}
