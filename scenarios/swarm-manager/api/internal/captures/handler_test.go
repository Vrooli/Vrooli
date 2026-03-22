package captures

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	body := strings.NewReader(`{"text":"we should add a backup cron"}`)
	req := httptest.NewRequest("POST", "/api/v1/captures", body)
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
	if !ok || !strings.HasPrefix(id, "cap-") {
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
	body := strings.NewReader(`{"text":""}`)
	req := httptest.NewRequest("POST", "/api/v1/captures", body)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	h, _ := setupTestHandler(t)
	body := strings.NewReader(`{invalid`)
	req := httptest.NewRequest("POST", "/api/v1/captures", body)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
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
