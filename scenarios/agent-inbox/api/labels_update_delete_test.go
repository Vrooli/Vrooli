package main

import (
	"agent-inbox/domain"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:LABEL-005] Test update label (edit name and color)
func TestUpdateLabel(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	label := createTestLabel(t, ts, "Original", "#ef4444")

	// Update label
	body := bytes.NewBuffer([]byte(`{"name": "Updated", "color": "#3b82f6"}`))
	req := httptest.NewRequest("PATCH", "/api/v1/labels/"+label.ID, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated domain.Label
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if updated.Name != "Updated" {
		t.Errorf("Expected name 'Updated', got %s", updated.Name)
	}

	if updated.Color != "#3b82f6" {
		t.Errorf("Expected color '#3b82f6', got %s", updated.Color)
	}
}

// [REQ:LABEL-005] Test update label name only
func TestUpdateLabelNameOnly(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	label := createTestLabel(t, ts, "Original", "#ef4444")

	body := bytes.NewBuffer([]byte(`{"name": "New Name"}`))
	req := httptest.NewRequest("PATCH", "/api/v1/labels/"+label.ID, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var updated domain.Label
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Expected name 'New Name', got %s", updated.Name)
	}

	// Color should remain unchanged
	if updated.Color != "#ef4444" {
		t.Errorf("Expected color '#ef4444' (unchanged), got %s", updated.Color)
	}
}

// [REQ:LABEL-005] Test update nonexistent label
func TestUpdateNonexistentLabel(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	body := bytes.NewBuffer([]byte(`{"name": "Test"}`))
	req := httptest.NewRequest("PATCH", "/api/v1/labels/00000000-0000-0000-0000-000000000000", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// [REQ:LABEL-006] Test delete label
func TestDeleteLabel(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	label := createTestLabel(t, ts, "ToDelete", "#ef4444")

	req := httptest.NewRequest("DELETE", "/api/v1/labels/"+label.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify label is deleted
	req = httptest.NewRequest("GET", "/api/v1/labels", nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var labels []domain.Label
	if err := json.Unmarshal(w.Body.Bytes(), &labels); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	for _, l := range labels {
		if l.ID == label.ID {
			t.Error("Label should have been deleted")
		}
	}
}

// [REQ:LABEL-006] Test delete label removes from chats
func TestDeleteLabelRemovesFromChats(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)
	label := createTestLabel(t, ts, "ToDelete", "#ef4444")

	// Assign label to chat
	req := httptest.NewRequest("PUT", "/api/v1/chats/"+chat.ID+"/labels/"+label.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	// Delete the label
	req = httptest.NewRequest("DELETE", "/api/v1/labels/"+label.ID, nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	// Verify label is removed from chat
	req = httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID, nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var result struct {
		Chat domain.Chat `json:"chat"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	for _, labelID := range result.Chat.LabelIDs {
		if labelID == label.ID {
			t.Error("Label should have been removed from chat when deleted")
		}
	}
}

// [REQ:LABEL-006] Test delete nonexistent label
func TestDeleteNonexistentLabel(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	req := httptest.NewRequest("DELETE", "/api/v1/labels/00000000-0000-0000-0000-000000000000", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// [REQ:LABEL-002] Test list labels ordered by name
func TestListLabelsOrdered(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	createTestLabel(t, ts, "Zebra", "#ef4444")
	createTestLabel(t, ts, "Alpha", "#3b82f6")
	createTestLabel(t, ts, "Beta", "#10b981")

	req := httptest.NewRequest("GET", "/api/v1/labels", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var labels []domain.Label
	if err := json.Unmarshal(w.Body.Bytes(), &labels); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(labels) != 3 {
		t.Fatalf("Expected 3 labels, got %d", len(labels))
	}

	// Should be ordered alphabetically
	if labels[0].Name != "Alpha" {
		t.Errorf("Expected first label 'Alpha', got '%s'", labels[0].Name)
	}
	if labels[1].Name != "Beta" {
		t.Errorf("Expected second label 'Beta', got '%s'", labels[1].Name)
	}
	if labels[2].Name != "Zebra" {
		t.Errorf("Expected third label 'Zebra', got '%s'", labels[2].Name)
	}
}
