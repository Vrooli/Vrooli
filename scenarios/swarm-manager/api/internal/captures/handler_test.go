package captures

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitions"

	"github.com/gorilla/mux"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

type classificationWorkflowStub struct {
	start      agentmanager.WorkflowStart
	completion agentmanager.InvocationCompletion
	invocation agentmanager.Invocation
}

func (s *classificationWorkflowStub) StartWorkflow(_ context.Context, invocation agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	s.invocation = invocation
	return s.start, nil
}

func (s *classificationWorkflowStub) CollectWorkflow(_ context.Context, _ string) (agentmanager.InvocationCompletion, error) {
	return s.completion, nil
}

func setupTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	rootDir := t.TempDir()
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatalf("load transition registry: %v", err)
	}
	h := NewHandler(rootDir, rootDir, registry)
	return h, rootDir
}

// newMultipartRequest builds a multipart POST request with a text field and optional file parts.
func newMultipartRequest(t *testing.T, text string, files map[string]string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if text != "" {
		if err := w.WriteField("text", text); err != nil {
			t.Fatal(err)
		}
	}

	for name, contentType := range files {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="files"; filename="%s"`, name))
		h.Set("Content-Type", contentType)
		part, err := w.CreatePart(h)
		if err != nil {
			t.Fatal(err)
		}
		// Write some fake image bytes.
		_, _ = part.Write([]byte("fake-image-data"))
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/api/v1/captures", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestList_Empty(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest("GET", "/api/v1/captures", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string][]capture
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp["captures"]) != 0 {
		t.Errorf("expected 0 captures, got %d", len(resp["captures"]))
	}
}

func TestCreate_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	h.SetClassificationWorkflow(&classificationWorkflowStub{start: agentmanager.WorkflowStart{ExecutionID: "execution-create", DefinitionDigest: "sha256:capture"}})
	req := newMultipartRequest(t, "we should add a backup cron", nil)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	capMap, ok := resp["capture"].(map[string]any)
	if !ok {
		t.Fatal("response missing capture object")
	}

	id, ok := capMap["id"].(string)
	if !ok || len(id) < 4 || id[:4] != "cap-" {
		t.Errorf("expected cap- prefixed ID, got %v", capMap["id"])
	}

	if capMap["text"] != "we should add a backup cron" {
		t.Errorf("unexpected text: %v", capMap["text"])
	}

	// Starting a declared workflow leaves the capture classifying until its
	// terminal typed result is explicitly applied.
	if capMap["status"] != "classifying" {
		t.Errorf("expected status classifying, got %v", capMap["status"])
	}

	// Verify file was created on disk.
	specPath := filepath.Join(rootDir, "captures", id, "capture.json")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Error("capture.json not created on disk")
	}
}

func TestCreate_EmptyText(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := newMultipartRequest(t, "", nil)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_InvalidForm(t *testing.T) {
	h, _ := setupTestHandler(t)
	// Send a non-multipart body.
	req := httptest.NewRequest("POST", "/api/v1/captures", bytes.NewReader([]byte("not-multipart")))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_WithImageFiles(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	req := newMultipartRequest(t, "screenshot of the bug", map[string]string{
		"bug.png":    "image/png",
		"detail.jpg": "image/jpeg",
	})
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	capMap := resp["capture"].(map[string]any)
	id := capMap["id"].(string)

	attachments, ok := capMap["attachments"].([]any)
	if !ok || len(attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %v", capMap["attachments"])
	}

	// Verify files exist on disk.
	for _, att := range attachments {
		attPath := filepath.Join(rootDir, "captures", id, att.(string))
		if _, err := os.Stat(attPath); os.IsNotExist(err) {
			t.Errorf("attachment file not found: %s", attPath)
		}
	}
}

func TestCreate_RejectsNonImage(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := newMultipartRequest(t, "here is a pdf", map[string]string{
		"doc.pdf": "application/pdf",
	})
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAndDelete(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Create a capture manually.
	id := "cap-test-123"
	dir := filepath.Join(rootDir, "captures", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cap := capture{
		ID:          id,
		Text:        "test thought",
		Attachments: []string{},
		Created:     time.Now().UTC().Format(time.RFC3339),
		Status:      "classifying",
	}
	data, _ := json.Marshal(cap)
	if err := os.WriteFile(filepath.Join(dir, "capture.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Get.
	req := httptest.NewRequest("GET", "/api/v1/captures/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", w.Code)
	}

	var getResp map[string]capture
	if err := json.Unmarshal(w.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("unmarshal get: %v", err)
	}
	if getResp["capture"].Text != "test thought" {
		t.Errorf("unexpected text: %v", getResp["capture"].Text)
	}

	// Delete.
	req = httptest.NewRequest("DELETE", "/api/v1/captures/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w = httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("delete: expected 204, got %d", w.Code)
	}

	// Verify deleted.
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("capture directory still exists after delete")
	}
}

func TestGet_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest("GET", "/api/v1/captures/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDelete_Idempotent(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest("DELETE", "/api/v1/captures/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for idempotent delete, got %d", w.Code)
	}
}

func TestList_WithCaptures(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Create two captures manually.
	for i, text := range []string{"first thought", "second thought"} {
		id := "cap-test-" + string(rune('a'+i))
		dir := filepath.Join(rootDir, "captures", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		cap := capture{
			ID:          id,
			Text:        text,
			Attachments: []string{},
			Created:     time.Now().UTC().Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
			Status:      "classifying",
		}
		data, _ := json.Marshal(cap)
		if err := os.WriteFile(filepath.Join(dir, "capture.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/captures", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string][]capture
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp["captures"]) != 2 {
		t.Errorf("expected 2 captures, got %d", len(resp["captures"]))
	}
	// Newest first.
	if resp["captures"][0].Text != "second thought" {
		t.Errorf("expected newest first, got %q", resp["captures"][0].Text)
	}
}

func TestLoadCapture_MergesClassification(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	id := "cap-test-classify"
	dir := filepath.Join(rootDir, "captures", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write capture.
	cap := capture{
		ID:          id,
		Text:        "test with classification",
		Attachments: []string{},
		Created:     time.Now().UTC().Format(time.RFC3339),
		Status:      "classifying",
	}
	data, _ := json.Marshal(cap)
	if err := os.WriteFile(filepath.Join(dir, "capture.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write classification.
	cls := classification{
		Items: []classificationItem{
			{Kind: "idea", Title: "Test idea", Priority: 3, Tags: []string{"test"}, Confidence: 0.9},
		},
		ClassifiedAt: time.Now().UTC().Format(time.RFC3339),
	}
	clsData, _ := json.Marshal(cls)
	if err := os.WriteFile(filepath.Join(dir, "classification.json"), clsData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Load and verify merge.
	loaded, err := h.loadCapture(id)
	if err != nil {
		t.Fatalf("loadCapture: %v", err)
	}
	if loaded.Status != "classified" {
		t.Errorf("expected status classified, got %q", loaded.Status)
	}
	if loaded.Classification == nil {
		t.Fatal("classification not merged")
	}
	if len(loaded.Classification.Items) != 1 {
		t.Errorf("expected 1 classification item, got %d", len(loaded.Classification.Items))
	}
	if loaded.Classification.Items[0].Kind != "idea" {
		t.Errorf("expected kind idea, got %q", loaded.Classification.Items[0].Kind)
	}
}

// mockBacklogCreator implements BacklogItemCreator for tests.
type mockBacklogCreator struct {
	items []struct {
		kind, name, title, description string
		tags                           []string
	}
}

func (m *mockBacklogCreator) SaveItem(draft BacklogItemDraft) error {
	m.items = append(m.items, struct {
		kind, name, title, description string
		tags                           []string
	}{draft.Kind, draft.Name, draft.Title, draft.Description, draft.Tags})
	return nil
}

// [REQ:SWM-P0-001] backlog work intake: item from classified capture
func TestCreateItem_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	creator := &mockBacklogCreator{}
	h.SetBacklogCreator(creator)

	// Create a classified capture.
	id := "cap-create-item"
	dir := filepath.Join(rootDir, "captures", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cap := capture{
		ID:          id,
		Text:        "we should add a backup cron job",
		Attachments: []string{},
		Created:     time.Now().UTC().Format(time.RFC3339),
		Status:      "classified",
		Classification: &classification{
			Items: []classificationItem{
				{Kind: "execute", Title: "backup cron", Tags: []string{"ops", "infra"}, Priority: 3, Confidence: 0.9},
			},
			ClassifiedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}
	data, _ := json.Marshal(cap)
	if err := os.WriteFile(filepath.Join(dir, "capture.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	// Also write classification.json so loadCapture merges it.
	clsData, _ := json.Marshal(cap.Classification)
	if err := os.WriteFile(filepath.Join(dir, "classification.json"), clsData, 0o644); err != nil {
		t.Fatal(err)
	}

	body := bytes.NewBufferString(`{"kind": "execute"}`)
	req := httptest.NewRequest("POST", "/api/v1/captures/"+id+"/create-item", body)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()

	h.CreateItem(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	if len(creator.items) != 1 {
		t.Fatalf("expected 1 item created, got %d", len(creator.items))
	}
	item := creator.items[0]
	if item.kind != "execute" {
		t.Errorf("expected kind execute, got %q", item.kind)
	}
	if item.title == "" {
		t.Error("expected non-empty title")
	}
	if item.title != "backup cron" || item.description != "" {
		t.Errorf("classification fields were not carried through: %#v", item)
	}
	if len(item.tags) != 2 || item.tags[0] != "ops" || item.tags[1] != "infra" {
		t.Errorf("unexpected tags: %v", item.tags)
	}

	// Verify capture status was updated.
	loaded, err := h.loadCapture(id)
	if err != nil {
		t.Fatalf("loadCapture: %v", err)
	}
	if loaded.Status != "classified" {
		t.Errorf("expected status classified, got %q", loaded.Status)
	}
}

func TestCreateItem_CreatesEveryClassificationItemWithProvenance(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	creator := &mockBacklogCreator{}
	h.SetBacklogCreator(creator)
	id := "cap-multiple-items"
	dir := filepath.Join(rootDir, "captures", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cap := capture{ID: id, Text: "two intents", Created: time.Now().UTC().Format(time.RFC3339), Status: "classified", Classification: &classification{Items: []classificationItem{
		{Kind: "fix", Title: "Fix login", Description: "Repair login", Priority: 2, Tags: []string{"auth"}},
		{Kind: "research", Title: "Research backups", Description: "Compare options", Priority: 9, Tags: []string{"ops"}},
		{Kind: "execute", Title: "Ship dashboard", Description: "Implement dashboard", Priority: 6, Tags: []string{"ui"}},
	}}}
	data, _ := json.Marshal(cap)
	if err := os.WriteFile(filepath.Join(dir, "capture.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	request := mux.SetURLVars(httptest.NewRequest("POST", "/api/v1/captures/"+id+"/create-item", nil), map[string]string{"id": id})
	recorder := httptest.NewRecorder()
	h.CreateItem(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create items = %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(creator.items) != 3 {
		t.Fatalf("created %d items, want 3", len(creator.items))
	}
	for _, item := range creator.items {
		if item.kind == "" || item.title == "" {
			t.Errorf("incomplete draft: %#v", item)
		}
	}
	// The mock records the shape; use the real adapter contract's explicit
	// provenance assertion through the response payload.
	var response struct {
		Items []struct {
			Priority    int    `json:"priority"`
			SpawnedFrom string `json:"spawned_from"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 3 {
		t.Fatalf("response items = %d, want 3", len(response.Items))
	}
	for _, item := range response.Items {
		if item.SpawnedFrom != id || item.Priority < 1 || item.Priority > 10 {
			t.Errorf("bad response provenance/priority: %#v", item)
		}
	}
}

func TestCaptureVersionChangesWhenNoteChanges(t *testing.T) {
	base := &capture{ID: "cap-note", Text: "same", Attachments: []string{"attachments/a.png"}, Note: "first"}
	first := captureVersion(base)
	base.Note = "second"
	second := captureVersion(base)
	if first == second {
		t.Fatal("capture version did not change when note changed")
	}
}

func TestBuildClassificationInputUsesReadableAttachmentPathsAndNote(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	id := "cap-attachment-input"
	attachment := filepath.Join(rootDir, "captures", id, "attachments", "screen.png")
	if err := os.MkdirAll(filepath.Dir(attachment), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(attachment, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	cap := &capture{ID: id, Text: "inspect screenshot", Note: "look at the error banner", Attachments: []string{"attachments/screen.png"}}
	if err := h.writeCapture(cap); err != nil {
		t.Fatal(err)
	}
	snapshot, err := h.buildClassificationInput(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	root := snapshot.Input.GetStructValue().AsMap()["capture"].(map[string]any)
	paths := root["attachments"].([]any)
	if len(paths) != 1 {
		t.Fatalf("attachments = %#v", paths)
	}
	path := paths[0].(string)
	if !filepath.IsAbs(path) {
		t.Fatalf("attachment path is not absolute: %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("attachment path is unreachable: %v", err)
	}
	if root["note"] != cap.Note {
		t.Fatalf("note = %#v, want %q", root["note"], cap.Note)
	}
}

func TestCreateItem_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)
	creator := &mockBacklogCreator{}
	h.SetBacklogCreator(creator)

	req := httptest.NewRequest("POST", "/api/v1/captures/nonexistent/create-item", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	h.CreateItem(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCreateItem_NoCreator(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	// Don't set backlog creator.

	id := "cap-no-creator"
	dir := filepath.Join(rootDir, "captures", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cap := capture{
		ID:      id,
		Text:    "test",
		Created: time.Now().UTC().Format(time.RFC3339),
		Status:  "classified",
	}
	data, _ := json.Marshal(cap)
	_ = os.WriteFile(filepath.Join(dir, "capture.json"), data, 0o644)

	req := httptest.NewRequest("POST", "/api/v1/captures/"+id+"/create-item", nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()

	h.CreateItem(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestClassify_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest("POST", "/api/v1/captures/nonexistent/classify", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	h.Classify(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestClassify_StartsDeclaredWorkflowWithVersionedCapture(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	stub := &classificationWorkflowStub{start: agentmanager.WorkflowStart{ExecutionID: "execution-1", DefinitionDigest: "sha256:capture"}}
	h.SetClassificationWorkflow(stub)
	cap := &capture{ID: "cap-1", Text: "add a backup job", Attachments: []string{}, Created: time.Now().UTC().Format(time.RFC3339), Status: "failed"}
	if err := os.MkdirAll(filepath.Join(rootDir, "captures", cap.ID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := h.writeCapture(cap); err != nil {
		t.Fatal(err)
	}
	req := mux.SetURLVars(httptest.NewRequest("POST", "/api/v1/captures/cap-1/classify", nil), map[string]string{"id": cap.ID})
	w := httptest.NewRecorder()
	h.Classify(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if stub.invocation.WorkflowKey != "swarm-manager/capture-classify" || stub.invocation.IdempotencyKey == "" {
		t.Fatalf("unexpected invocation: %#v", stub.invocation)
	}
	stored, err := h.loadCapture(cap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "classifying" {
		t.Fatalf("capture state = %q, want classifying", stored.Status)
	}
}

func TestApplyClassification_OnlyAppliesMatchingTerminalSnapshot(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "suggested", "items": []any{map[string]any{"kind": "idea", "title": "Backups", "description": "add backups", "priority": 2, "tags": []any{"ops"}, "confidence": 0.9}}}})
	if err != nil {
		t.Fatal(err)
	}
	stub := &classificationWorkflowStub{start: agentmanager.WorkflowStart{ExecutionID: "execution-apply", DefinitionDigest: "sha256:capture"}, completion: agentmanager.InvocationCompletion{ExecutionID: "execution-apply", DefinitionDigest: "sha256:capture", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Output: output}}
	h.SetClassificationWorkflow(stub)
	cap := &capture{ID: "cap-apply", Text: "add backups", Attachments: []string{}, Created: time.Now().UTC().Format(time.RFC3339), Status: "classifying"}
	if err := os.MkdirAll(filepath.Join(rootDir, "captures", cap.ID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := h.writeCapture(cap); err != nil {
		t.Fatal(err)
	}
	startReq := mux.SetURLVars(httptest.NewRequest("POST", "/api/v1/captures/cap-apply/classify", nil), map[string]string{"id": cap.ID})
	startRecorder := httptest.NewRecorder()
	h.Classify(startRecorder, startReq)
	if startRecorder.Code != http.StatusOK {
		t.Fatalf("start classification: %d %s", startRecorder.Code, startRecorder.Body.String())
	}
	req := mux.SetURLVars(httptest.NewRequest("POST", "/api/v1/captures/cap-apply/classify/execution-apply/apply", nil), map[string]string{"id": cap.ID, "executionID": "execution-apply"})
	w := httptest.NewRecorder()
	h.ApplyClassification(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	stored, err := h.loadCapture(cap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "classified" || stored.Classification == nil || len(stored.Classification.Items) != 1 {
		t.Fatalf("expected applied classification, got %#v", stored)
	}
	second := httptest.NewRecorder()
	h.ApplyClassification(second, req)
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"already_applied":true`) {
		t.Fatalf("expected idempotent apply response, got %d: %s", second.Code, second.Body.String())
	}
}

func TestApplyClassificationRejectsCaptureChangedAfterStart(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "suggested", "items": []any{map[string]any{"kind": "idea", "title": "Backups", "description": "add backups", "priority": 2, "tags": []any{"ops"}, "confidence": 0.9}}}})
	if err != nil {
		t.Fatal(err)
	}
	stub := &classificationWorkflowStub{start: agentmanager.WorkflowStart{ExecutionID: "execution-stale", DefinitionDigest: "sha256:capture"}, completion: agentmanager.InvocationCompletion{ExecutionID: "execution-stale", DefinitionDigest: "sha256:capture", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Output: output}}
	h.SetClassificationWorkflow(stub)
	cap := &capture{ID: "cap-stale", Text: "add backups", Created: time.Now().UTC().Format(time.RFC3339), Status: "failed"}
	if err := os.MkdirAll(filepath.Join(rootDir, "captures", cap.ID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := h.writeCapture(cap); err != nil {
		t.Fatal(err)
	}
	start := mux.SetURLVars(httptest.NewRequest("POST", "/api/v1/captures/cap-stale/classify", nil), map[string]string{"id": cap.ID})
	h.Classify(httptest.NewRecorder(), start)
	cap.Text = "the capture changed while classification ran"
	if err := h.writeCapture(cap); err != nil {
		t.Fatal(err)
	}
	apply := mux.SetURLVars(httptest.NewRequest("POST", "/api/v1/captures/cap-stale/classify/execution-stale/apply", nil), map[string]string{"id": cap.ID, "executionID": "execution-stale"})
	recorder := httptest.NewRecorder()
	h.ApplyClassification(recorder, apply)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("apply changed capture = %d: %s", recorder.Code, recorder.Body.String())
	}
	stored, err := h.loadCapture(cap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Text != cap.Text || stored.Classification != nil {
		t.Fatalf("stale completion mutated capture: %#v", stored)
	}
}

func TestApplyClassificationRecoversAfterHandlerRestart(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "discarded", "reason": "not actionable"}})
	if err != nil {
		t.Fatal(err)
	}
	stub := &classificationWorkflowStub{start: agentmanager.WorkflowStart{ExecutionID: "execution-recover", DefinitionDigest: "sha256:capture"}, completion: agentmanager.InvocationCompletion{ExecutionID: "execution-recover", DefinitionDigest: "sha256:capture", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Output: output}}
	h.SetClassificationWorkflow(stub)
	cap := &capture{ID: "cap-recover", Text: "discard this", Created: time.Now().UTC().Format(time.RFC3339), Status: "failed"}
	if err := os.MkdirAll(filepath.Join(rootDir, "captures", cap.ID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := h.writeCapture(cap); err != nil {
		t.Fatal(err)
	}
	start := mux.SetURLVars(httptest.NewRequest("POST", "/api/v1/captures/cap-recover/classify", nil), map[string]string{"id": cap.ID})
	if recorder := httptest.NewRecorder(); func() *httptest.ResponseRecorder { h.Classify(recorder, start); return recorder }().Code != http.StatusOK {
		t.Fatalf("start classification = %d", recorder.Code)
	}

	// A new handler models a process restart. It reads the correlation from the
	// shared on-disk journal rather than any handler-local state.
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewHandler(rootDir, rootDir, registry)
	restarted.SetClassificationWorkflow(stub)
	apply := mux.SetURLVars(httptest.NewRequest("POST", "/api/v1/captures/cap-recover/classify/execution-recover/apply", nil), map[string]string{"id": cap.ID, "executionID": "execution-recover"})
	recorder := httptest.NewRecorder()
	restarted.ApplyClassification(recorder, apply)
	if recorder.Code != http.StatusOK {
		t.Fatalf("recovered apply = %d: %s", recorder.Code, recorder.Body.String())
	}
	stored, err := restarted.loadCapture(cap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "classified" || stored.Classification == nil {
		t.Fatalf("recovered capture = %#v", stored)
	}
}
