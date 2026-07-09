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

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/testutil"

	"github.com/gorilla/mux"
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

type queueBlockingReason struct {
	Message   string `json:"message"`
	Forceable bool   `json:"forceable"`
}

type backlogQueueResponse struct {
	Item                BacklogItem           `json:"item"`
	TaskID              string                `json:"task_id"`
	RunID               string                `json:"run_id"`
	BaseURL             string                `json:"base_url"`
	Created             string                `json:"created"`
	DryRun              bool                  `json:"dry_run"`
	Queued              bool                  `json:"queued"`
	Message             string                `json:"message"`
	BlockingReasons     []queueBlockingReason `json:"blocking_reasons"`
	UnansweredQuestions int                   `json:"unanswered_questions"`
	PendingSuggestions  int                   `json:"pending_suggestions"`
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
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "config", "settings.json"), map[string]any{
		"theme":                    "dark",
		"default_mode":             "manual",
		"max_auto_rounds":          10,
		"auto_initialize_workshop": false,
		"auto_advance_workshop":    false,
		"auto_cascade_workshop":    false,
		"agent_max_turns":          600,
		"agent_timeout_seconds":    900,
		"search_debounce_ms":       300,
		"toast_duration_ms":        5000,
		"delete_confirmation":      map[string]any{"backlog": "simple", "initiative": "strong", "capture": "none"},
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
	h := NewHandler(rootDir, rootDir)
	scopeExecutionQueuerForTest(t, h, rootDir, nil)
	return h, rootDir
}

func setupTestHandlerWithAgent(t *testing.T, agent agentmanager.Service) (*Handler, string) {
	t.Helper()
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	disableAutoWorkshopSettings(t, rootDir)
	h := NewHandlerWithClients(rootDir, rootDir, agent, &promptmanager.MockClient{Result: "test prompt"})
	scopeExecutionQueuerForTest(t, h, rootDir, agent)
	return h, rootDir
}

// scopeExecutionQueuerForTest wires the handler's secondary execution service
// to a tempdir-scoped StorePath. Without this, queue_ops.go falls back to the
// user-wide runtimepaths.StatePath("execution-runs.json"), so tests pick up
// whatever is in the developer's real queue and hit unrelated depth-limit or
// circuit-breaker errors.
func scopeExecutionQueuerForTest(t *testing.T, h *Handler, rootDir string, agent agentmanager.Service) {
	t.Helper()
	storePath := filepath.Join(rootDir, ".vrooli", "execution-runs.json")
	cfg := execution.ServiceConfig{
		DataRoot:     rootDir,
		RepoRoot:     rootDir,
		StorePath:    storePath,
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	}
	h.SetExecutionQueuer(execution.NewService(cfg))
}

// createTestItem creates a test backlog item in the specified kind directory.
func createTestItem(t *testing.T, rootDir string, kind BacklogKind, item BacklogItem) {
	t.Helper()
	item.Kind = kind
	itemDir := filepath.Join(rootDir, backlogKindDirs[kind], item.Name)
	testutil.WriteJSONFile(t, filepath.Join(itemDir, "spec.json"), item)
}

// createReadyTestItem creates a test backlog item that passes workshop readiness preflight.
func createReadyTestItem(t *testing.T, rootDir string, kind BacklogKind, item BacklogItem) {
	t.Helper()
	if kind != KindResearch && item.PlanRef == nil {
		item.PlanRef = &PlanRef{
			Provider: PlanRefProviderPlanManager,
			PlanID:   "test-plan-" + item.Name,
			Slug:     "test-plan-" + item.Name,
			Role:     PlanRefRoleExecutionSpec,
		}
	}
	createTestItem(t, rootDir, kind, item)
	if kind == KindResearch {
		itemDir := filepath.Join(rootDir, backlogKindDirs[kind], item.Name)
		testutil.WriteFile(t, filepath.Join(itemDir, "conclusion.md"), "# Conclusion\nManually created conclusion for testing.")
	}
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

func (m *mockAgentService) ApproveRun(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (m *mockAgentService) ContinueRun(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockAgentService) SpawnResearch(_ context.Context, _ agentmanager.ResearchSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, nil
}

func (m *mockAgentService) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	return agentmanager.RunState{}, nil
}

func (m *mockAgentService) GetRunDiff(_ context.Context, runID string) (agentmanager.RunDiff, error) {
	return agentmanager.RunDiff{RunID: runID}, nil
}
func (m *mockAgentService) StopRun(_ context.Context, _ string) error { return nil }

// ---------------------------------------------------------------------------
// File operation tests
// ---------------------------------------------------------------------------

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

// uploadMultipart constructs a multipart upload request with the given path
// (server-side directory) and content body. Returns the HTTP request ready to
// hand to UploadFile.
func uploadMultipart(t *testing.T, kind BacklogKind, name, dir, filename, content string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, _ = part.Write([]byte(content))
	if dir != "" {
		_ = writer.WriteField("path", dir)
	}
	writer.Close()
	req := httptest.NewRequest("POST", "/api/v1/backlog/"+string(kind)+"/"+name+"/files", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = mux.SetURLVars(req, map[string]string{"kind": string(kind), "name": name})
	return req
}

// TestUploadFile_StaleAcceptanceWritesValidationArtifact confirms that any
// upload to an item with stale acceptance globs persists a structured
// acceptance-validation.json artifact, even when the upload itself is not a
// finalize round (which would block).
func TestUploadFile_StaleAcceptanceWritesValidationArtifact(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:            "stale-allow",
		Title:           "Stale acceptance",
		Status:          StatusBacklog,
		AcceptanceAllow: []string{"path-that-does-not-exist-anywhere/**"},
		Created:         "2026-01-28T00:00:00Z",
		Updated:         "2026-01-28T00:00:00Z",
	})

	req := uploadMultipart(t, KindIdea, "stale-allow", "workshop", "round-001.json", `{"round":1,"mode":"workshop","items":[]}`)
	w := httptest.NewRecorder()
	h.UploadFile(w, req)
	testutil.AssertStatusCreated(t, w)

	artifactPath := filepath.Join(rootDir, "ideas", "stale-allow", "acceptance-validation.json")
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("expected acceptance-validation.json, got: %v", err)
	}
	var artifact AcceptanceValidationArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		t.Fatalf("decode artifact: %v", err)
	}
	if len(artifact.Problems) == 0 {
		t.Errorf("expected at least one problem, got none")
	}
}

// TestUploadFile_FinalizeRoundBlockedOnStale confirms that uploading a
// workshop/round-N.json with mode=finalize against an item with stale
// acceptance globs returns plan_stale and rolls the file back from disk.
func TestUploadFile_FinalizeRoundBlockedOnStale(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:            "stale-finalize",
		Title:           "Stale finalize",
		Status:          StatusBacklog,
		AcceptanceAllow: []string{"path-that-does-not-exist-anywhere/**"},
		Created:         "2026-01-28T00:00:00Z",
		Updated:         "2026-01-28T00:00:00Z",
	})

	req := uploadMultipart(t, KindIdea, "stale-finalize", "workshop", "round-099.json",
		`{"round":99,"mode":"finalize","items":[]}`)
	w := httptest.NewRecorder()
	h.UploadFile(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 plan_stale, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "plan_stale") {
		t.Errorf("expected body to contain plan_stale, got %s", w.Body.String())
	}
	uploadedPath := filepath.Join(rootDir, "ideas", "stale-finalize", "workshop", "round-099.json")
	if _, err := os.Stat(uploadedPath); !os.IsNotExist(err) {
		t.Errorf("expected stale finalize file rolled back from disk, but it still exists")
	}
}

// TestUploadFile_FinalizeRoundCleanPasses confirms that a finalize round
// upload against an item whose acceptance globs all exist (or are declared
// in `creates`) succeeds normally.
func TestUploadFile_FinalizeRoundCleanPasses(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:            "clean-finalize",
		Title:           "Clean finalize",
		Status:          StatusBacklog,
		AcceptanceAllow: []string{"path-that-does-not-exist-anywhere/**"},
		Creates:         []string{"path-that-does-not-exist-anywhere/**"},
		Created:         "2026-01-28T00:00:00Z",
		Updated:         "2026-01-28T00:00:00Z",
	})

	req := uploadMultipart(t, KindIdea, "clean-finalize", "workshop", "round-099.json",
		`{"round":99,"mode":"finalize","items":[]}`)
	w := httptest.NewRecorder()
	h.UploadFile(w, req)

	testutil.AssertStatusCreated(t, w)
	uploadedPath := filepath.Join(rootDir, "ideas", "clean-finalize", "workshop", "round-099.json")
	if _, err := os.Stat(uploadedPath); err != nil {
		t.Errorf("expected clean finalize file persisted, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Queue / Process / Research tests
// ---------------------------------------------------------------------------

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

func TestQueue_AllowsArchivedIdea(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:        "archived-idea",
		Title:       "Archived Idea",
		Description: "",
		Status:      StatusCompleted,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
		Kind:        KindIdea,
		ArchivedAt:  strPtr("2026-01-01T00:00:00Z"),
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
		Status:      StatusCompleted,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
		Kind:        KindIdea,
		ArchivedAt:  strPtr("2026-01-01T00:00:00Z"),
	}
	createTestItem(t, rootDir, KindIdea, item)
	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "archived-preflight", "spec.json"), `{
  "name":"archived-preflight",
  "title":"[Archived] web-console",
  "description":"",
  "status":"completed",
  "priority":3,
  "tags":[],
  "created":"2026-01-28T00:00:00Z",
  "updated":"2026-01-28T00:00:00Z",
  "kind":"idea",
  "archived_at":"2026-01-01T00:00:00Z",
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

func TestProcessPreflight_ResearchUsesConclusionDeliverable(t *testing.T) {
	h, rootDir := setupTestHandlerWithAgent(t, &mockAgentService{})

	item := BacklogItem{
		Name:        "research-preflight",
		Title:       "Research Preflight",
		Description: "Verify research preflight accepts conclusion deliverables.",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
		Kind:        KindResearch,
	}
	createTestItem(t, rootDir, KindResearch, item)
	testutil.WriteFile(t, filepath.Join(rootDir, "research", "research-preflight", "conclusion.md"), "# Conclusion\nReady for processing.")

	req := httptest.NewRequest("GET", "/api/v1/backlog/research/research-preflight/process-preflight", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "research", "name": "research-preflight"})
	w := httptest.NewRecorder()

	h.ProcessPreflight(w, req)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[processPreflightEnvelope](t, w)
	if ready, ok := resp.Preflight["ready"].(bool); !ok || !ready {
		t.Fatalf("expected preflight ready=true, got %#v", resp.Preflight["ready"])
	}
	if reasons, ok := resp.Preflight["blocking_reasons"].([]any); ok && len(reasons) > 0 {
		t.Fatalf("expected no blocking reasons, got %#v", reasons)
	}
}

func TestQueue_BlocksWhenProcessPreflightFails(t *testing.T) {
	h, rootDir := setupTestHandlerWithAgent(t, &mockAgentService{})

	item := BacklogItem{
		Name:        "blocked-queue",
		Title:       "[Archived] web-console",
		Description: "",
		Status:      StatusCompleted,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
		Kind:        KindIdea,
		ArchivedAt:  strPtr("2026-01-01T00:00:00Z"),
	}
	// No plan_ref or workshop rounds — preflight will block.
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
	createReadyTestItem(t, rootDir, KindIdea, item)

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

func TestQueue_BlocksOnUnmetDeps_ForceOverrides(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Create dep in "backlog" status — not yet planned.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "dep-unmet", Title: "Unmet Dep", Status: StatusBacklog, Priority: 5,
		Tags: []string{}, Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	})
	// Create the item that depends on it, with a canonical plan_ref so preflight passes.
	createReadyTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "dep-child", Title: "Dep Child", Status: StatusReady, Priority: 3,
		Tags: []string{}, Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
		DependsOn: []string{"idea/dep-unmet"},
	})

	// Without force: should be blocked.
	blockedBody := bytes.NewBufferString(`{"mode":"manual","confirm":true}`)
	blockedReq := httptest.NewRequest("POST", "/api/v1/backlog/fix/dep-child/queue", blockedBody)
	blockedReq.Header.Set("Content-Type", "application/json")
	blockedReq = mux.SetURLVars(blockedReq, map[string]string{"kind": "fix", "name": "dep-child"})
	blockedW := httptest.NewRecorder()
	h.Queue(blockedW, blockedReq)
	testutil.AssertStatusOK(t, blockedW)

	resp := testutil.DecodeJSON[backlogQueueResponse](t, blockedW)
	if !resp.DryRun {
		t.Error("expected dry_run=true when blocked by deps")
	}
	if resp.Queued {
		t.Error("expected queued=false when blocked by deps")
	}
	foundDepReason := false
	for _, r := range resp.BlockingReasons {
		if strings.Contains(r.Message, "dependencies") && r.Forceable {
			foundDepReason = true
		}
	}
	if !foundDepReason {
		t.Errorf("expected a forceable dependency blocking reason, got %+v", resp.BlockingReasons)
	}

	// With force: should succeed.
	forcedBody := bytes.NewBufferString(`{"mode":"manual","confirm":true,"force":true}`)
	forcedReq := httptest.NewRequest("POST", "/api/v1/backlog/fix/dep-child/queue", forcedBody)
	forcedReq.Header.Set("Content-Type", "application/json")
	forcedReq = mux.SetURLVars(forcedReq, map[string]string{"kind": "fix", "name": "dep-child"})
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

// ---------------------------------------------------------------------------
// Workshop tests
// ---------------------------------------------------------------------------

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

// mockActivityChecker is a test double for AgentActivityChecker.
type mockActivityChecker struct {
	active bool
}

func (m *mockActivityChecker) HasActiveAgent(_ context.Context, _, _ string) bool {
	return m.active
}

func TestWorkshopDeleteRound_ActiveAgentConflict(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	// Simulate an active agent for this item.
	h.activityChecker = &mockActivityChecker{active: true}

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

	reqBody := bytes.NewBufferString(`{"round_number":1}`)
	req := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/ws-delete-lock/workshop/round", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "ws-delete-lock"})
	w := httptest.NewRecorder()

	h.WorkshopDeleteRound(w, req)
	testutil.AssertStatus(t, w, http.StatusConflict)
}

// ---------------------------------------------------------------------------
// WorkshopReset
// ---------------------------------------------------------------------------

func TestWorkshopReset_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name: "ws-reset-ok", Title: "WS Reset OK",
		Status: StatusBacklog, Priority: 3,
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
	createReadyTestItem(t, rootDir, KindIdea, item)

	itemDir := filepath.Join(rootDir, "ideas", "ws-reset-ok")
	workshopDir := filepath.Join(itemDir, "workshop")
	if err := os.MkdirAll(workshopDir, 0o755); err != nil {
		t.Fatalf("mkdir workshop: %v", err)
	}
	for i := 1; i <= 2; i++ {
		content := fmt.Sprintf(`{"round":%d,"generated_at":"2026-01-01T00:00:00Z","readiness":{},"items":[]}`, i)
		testutil.WriteFile(t, filepath.Join(workshopDir, fmt.Sprintf("round-%03d.json", i)), content)
	}
	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/ws-reset-ok/workshop/reset", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "ws-reset-ok"})
	w := httptest.NewRecorder()

	h.WorkshopReset(w, req)
	testutil.AssertStatusOK(t, w)

	// Workshop dir should be gone.
	testutil.AssertFileNotExists(t, workshopDir)
	// plan_ref should survive.
	saved, err := h.store.LoadItem(KindIdea, "ws-reset-ok")
	if err != nil {
		t.Fatalf("load item after reset: %v", err)
	}
	if saved.PlanRef == nil {
		t.Fatal("expected plan_ref to survive reset")
	}
	// spec.json should survive.
	testutil.AssertFileExists(t, filepath.Join(itemDir, "spec.json"))

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["deleted_rounds"] != float64(2) {
		t.Errorf("expected deleted_rounds=2, got %v", resp["deleted_rounds"])
	}
	// status_reverted is omitted when false (proto JSON default value behavior).
	if _, ok := resp["status_reverted"]; ok {
		t.Errorf("expected status_reverted to be absent (false), got %v", resp["status_reverted"])
	}
}

func TestWorkshopReset_RevertsReadyStatus(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name: "ws-reset-ready", Title: "WS Reset Ready",
		Status: StatusReady, Priority: 3,
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	itemDir := filepath.Join(rootDir, "ideas", "ws-reset-ready")
	workshopDir := filepath.Join(itemDir, "workshop")
	if err := os.MkdirAll(workshopDir, 0o755); err != nil {
		t.Fatalf("mkdir workshop: %v", err)
	}
	testutil.WriteFile(t, filepath.Join(workshopDir, "round-001.json"),
		`{"round":1,"generated_at":"2026-01-01T00:00:00Z","readiness":{},"items":[]}`)

	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/ws-reset-ready/workshop/reset", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "ws-reset-ready"})
	w := httptest.NewRecorder()

	h.WorkshopReset(w, req)
	testutil.AssertStatusOK(t, w)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["status_reverted"] != true {
		t.Errorf("expected status_reverted=true, got %v", resp["status_reverted"])
	}

	// Reload item and check status was reverted.
	reloaded, err := h.store.LoadItem(KindIdea, "ws-reset-ready")
	if err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if reloaded.Status != StatusBacklog {
		t.Errorf("expected status=backlog, got %s", reloaded.Status)
	}
}

func TestWorkshopReset_NoStatusChangeWhenNotReady(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name: "ws-reset-inprog", Title: "WS Reset InProg",
		Status: StatusInProgress, Priority: 3,
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/ws-reset-inprog/workshop/reset", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "ws-reset-inprog"})
	w := httptest.NewRecorder()

	h.WorkshopReset(w, req)
	testutil.AssertStatusOK(t, w)

	reloaded, err := h.store.LoadItem(KindIdea, "ws-reset-inprog")
	if err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if reloaded.Status != StatusInProgress {
		t.Errorf("expected status=in_progress, got %s", reloaded.Status)
	}
}

func TestWorkshopReset_409WhenAgentActive(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	// Simulate an active agent for this item.
	h.activityChecker = &mockActivityChecker{active: true}

	item := BacklogItem{
		Name: "ws-reset-lock", Title: "WS Reset Lock",
		Status: StatusBacklog, Priority: 3,
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/ws-reset-lock/workshop/reset", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "ws-reset-lock"})
	w := httptest.NewRecorder()

	h.WorkshopReset(w, req)
	testutil.AssertStatus(t, w, http.StatusConflict)
}

func TestWorkshopReset_404WhenItemMissing(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/nonexistent/workshop/reset", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "nonexistent"})
	w := httptest.NewRecorder()

	h.WorkshopReset(w, req)
	testutil.AssertStatus(t, w, http.StatusNotFound)
}

func TestWorkshopReset_NoopWhenEmpty(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name: "ws-reset-empty", Title: "WS Reset Empty",
		Status: StatusBacklog, Priority: 3,
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest("POST", "/api/v1/backlog/idea/ws-reset-empty/workshop/reset", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "ws-reset-empty"})
	w := httptest.NewRecorder()

	h.WorkshopReset(w, req)
	testutil.AssertStatusOK(t, w)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	// deleted_rounds is omitted when 0 (proto JSON default value behavior).
	if _, ok := resp["deleted_rounds"]; ok {
		t.Errorf("expected deleted_rounds to be absent (0), got %v", resp["deleted_rounds"])
	}
}

func TestArchive_TransitionsReviewStatusToCompleted(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name: "archive-from-review", Title: "Archive From Review",
		Status: StatusInReview, Priority: 3,
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
		Kind: KindExecute,
	}
	createTestItem(t, rootDir, KindExecute, item)

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/execute/archive-from-review/archive-item", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "archive-from-review"})
	w := httptest.NewRecorder()

	h.Archive(w, req)
	testutil.AssertStatusOK(t, w)

	reloaded, err := h.store.LoadItem(KindExecute, "archive-from-review")
	if err != nil {
		t.Fatalf("load after archive: %v", err)
	}
	if reloaded.Status != StatusCompleted {
		t.Errorf("status after archive: got %q, want %q", reloaded.Status, StatusCompleted)
	}
	if reloaded.ArchivedAt == nil || *reloaded.ArchivedAt == "" {
		t.Errorf("archived_at not set")
	}
}

func TestArchive_LeavesTerminalStatusUnchanged(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name: "archive-already-failed", Title: "Archive Already Failed",
		Status: StatusFailed, Priority: 3,
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
		Kind: KindExecute,
	}
	createTestItem(t, rootDir, KindExecute, item)

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/execute/archive-already-failed/archive-item", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "archive-already-failed"})
	w := httptest.NewRecorder()

	h.Archive(w, req)
	testutil.AssertStatusOK(t, w)

	reloaded, err := h.store.LoadItem(KindExecute, "archive-already-failed")
	if err != nil {
		t.Fatalf("load after archive: %v", err)
	}
	if reloaded.Status != StatusFailed {
		t.Errorf("status after archive: got %q, want %q (terminal statuses should be preserved)", reloaded.Status, StatusFailed)
	}
	if reloaded.ArchivedAt == nil {
		t.Errorf("archived_at not set")
	}
}

func TestWorkshopReset_ResearchDeletesConclusion(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name: "ws-reset-research", Title: "WS Reset Research",
		Status: StatusBacklog, Priority: 3,
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindResearch, item)

	itemDir := filepath.Join(rootDir, "research", "ws-reset-research")
	workshopDir := filepath.Join(itemDir, "workshop")
	if err := os.MkdirAll(workshopDir, 0o755); err != nil {
		t.Fatalf("mkdir workshop: %v", err)
	}
	testutil.WriteFile(t, filepath.Join(workshopDir, "round-001.json"),
		`{"round":1,"generated_at":"2026-01-01T00:00:00Z","readiness":{},"items":[]}`)
	testutil.WriteFile(t, filepath.Join(itemDir, "conclusion.md"), "# Conclusion")

	req := httptest.NewRequest("POST", "/api/v1/backlog/research/ws-reset-research/workshop/reset", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "research", "name": "ws-reset-research"})
	w := httptest.NewRecorder()

	h.WorkshopReset(w, req)
	testutil.AssertStatusOK(t, w)

	testutil.AssertFileNotExists(t, filepath.Join(itemDir, "conclusion.md"))
	testutil.AssertFileNotExists(t, workshopDir)
}
