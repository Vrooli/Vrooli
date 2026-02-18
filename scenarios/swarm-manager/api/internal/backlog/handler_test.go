package backlog

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	Item    BacklogItem `json:"item"`
	TaskID  string      `json:"task_id"`
	RunID   string      `json:"run_id"`
	BaseURL string      `json:"base_url"`
	Created string      `json:"created"`
}

type backlogFileOperationResponse struct {
	File        *BacklogFile `json:"file,omitempty"`
	DeletedPath *string      `json:"deleted_path,omitempty"`
}

// setupTestHandler creates a handler with a temporary root directory.
func setupTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	return NewHandler(rootDir), rootDir
}

func setupTestHandlerWithAgent(t *testing.T, agent agentmanager.Service) (*Handler, string) {
	t.Helper()
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	return NewHandlerWithClients(rootDir, agent, &promptmanager.MockClient{Result: "test prompt"}), rootDir
}

// createTestItem creates a test backlog item in the specified kind directory.
func createTestItem(t *testing.T, rootDir string, kind BacklogKind, item BacklogItem) {
	t.Helper()
	item.Kind = kind
	itemDir := filepath.Join(rootDir, backlogKindDirs[kind], item.Name)
	testutil.WriteJSONFile(t, filepath.Join(itemDir, "spec.json"), item)
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

func TestQueue_SpawnsAgent(t *testing.T) {
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
	testutil.AssertStatus(t, w, http.StatusAccepted)

	resp := testutil.DecodeJSON[backlogQueueResponse](t, w)
	if resp.Item.Status != StatusQueued {
		t.Errorf("expected queued status, got %s", resp.Item.Status)
	}
	if agent.lastReq == nil || agent.lastReq.Purpose != "process" {
		t.Errorf("expected agent spawn for processing")
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
	createTestItem(t, rootDir, KindIdea, item)

	reqBody := bytes.NewBufferString(`{"mode":"manual","operation":"generator"}`)
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
	createTestItem(t, rootDir, KindIdea, item)

	reqBody := bytes.NewBufferString(`{"mode":"scheduled","delay_seconds":60}`)
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
	createTestItem(t, rootDir, KindIdea, item)

	reqBody := bytes.NewBufferString(`{"mode":"manual","operation":"generator"}`)
	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/archived-idea/queue", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "archived-idea"})
	w := httptest.NewRecorder()

	h.Queue(w, req)
	testutil.AssertStatus(t, w, http.StatusAccepted)
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

	payload := map[string]any{"mode": "clarify", "prompt": "focus"}
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

	traceReq := httptest.NewRequest("GET", "/api/v1/backlog/idea/research-test/prompt-trace", nil)
	traceReq = mux.SetURLVars(traceReq, map[string]string{"kind": "idea", "name": "research-test"})
	traceW := httptest.NewRecorder()
	h.GetPromptTrace(traceW, traceReq)
	testutil.AssertStatusOK(t, traceW)
	if !strings.Contains(traceW.Body.String(), `"skill_id":"swarm-manager-clarify-idea"`) {
		t.Fatalf("expected prompt trace with clarify skill, got %s", traceW.Body.String())
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
	if err := h.saveItem(item); err != nil {
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
	lastReq *agentmanager.BacklogSpawnRequest
	result  agentmanager.RunResult
	err     error
}

func (m *mockAgentService) IsEnabled() bool                    { return true }
func (m *mockAgentService) IsAvailable(_ context.Context) bool { return true }
func (m *mockAgentService) ResolveURL(_ context.Context) (string, error) {
	return "http://agent", nil
}
func (m *mockAgentService) GetProfileID() string { return "" }
func (m *mockAgentService) SpawnBacklog(_ context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	m.lastReq = &req
	return m.result, m.err
}

func (m *mockAgentService) SpawnResearch(_ context.Context, _ agentmanager.ResearchSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, nil
}

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
	part.Write([]byte(`{"test":true}`))
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
