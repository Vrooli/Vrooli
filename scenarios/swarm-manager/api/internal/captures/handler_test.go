package captures

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func setupTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	rootDir := t.TempDir()
	h := NewHandler(rootDir, nil, nil)
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

	// Status should be "failed" since no agent service is configured.
	if capMap["status"] != "failed" {
		t.Errorf("expected status failed (no agent), got %v", capMap["status"])
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
	items []struct{ kind, name, title, description string; tags []string }
}

func (m *mockBacklogCreator) ItemDir(kind, name string) string {
	return "/tmp/test/" + kind + "/" + name
}

func (m *mockBacklogCreator) SaveItem(kind, name, title, description string, tags []string) error {
	m.items = append(m.items, struct{ kind, name, title, description string; tags []string }{kind, name, title, description, tags})
	return nil
}

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
