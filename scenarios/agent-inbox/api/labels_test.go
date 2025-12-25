package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:LABEL-001] [REQ:LABEL-002] Test create and list labels
func TestLabels(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create a label
	body := bytes.NewBuffer([]byte(`{"name": "Important", "color": "#ef4444"}`))
	req := httptest.NewRequest("POST", "/api/v1/labels", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var label Label
	if err := json.Unmarshal(w.Body.Bytes(), &label); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if label.Name != "Important" {
		t.Errorf("Expected name 'Important', got %s", label.Name)
	}

	// List labels
	req = httptest.NewRequest("GET", "/api/v1/labels", nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var labels []Label
	if err := json.Unmarshal(w.Body.Bytes(), &labels); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(labels))
	}
}

// [REQ:LABEL-001] Test create label with default color
func TestCreateLabelDefaultColor(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	body := bytes.NewBuffer([]byte(`{"name": "Default Color Label"}`))
	req := httptest.NewRequest("POST", "/api/v1/labels", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var label Label
	if err := json.Unmarshal(w.Body.Bytes(), &label); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if label.Color != "#6366f1" {
		t.Errorf("Expected default color '#6366f1', got %s", label.Color)
	}
}

// [REQ:LABEL-001] Test create label requires name
func TestCreateLabelRequiresName(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	body := bytes.NewBuffer([]byte(`{"color": "#ef4444"}`))
	req := httptest.NewRequest("POST", "/api/v1/labels", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// [REQ:LABEL-001] Test create duplicate label fails
func TestCreateDuplicateLabel(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	// Create first label
	createTestLabel(t, ts, "Duplicate", "#ef4444")

	// Try to create duplicate
	body := bytes.NewBuffer([]byte(`{"name": "Duplicate"}`))
	req := httptest.NewRequest("POST", "/api/v1/labels", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

// [REQ:LABEL-003] Test assign label to chat
func TestAssignLabel(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)
	label := createTestLabel(t, ts, "Work", "#3b82f6")

	// Assign label
	req := httptest.NewRequest("PUT", "/api/v1/chats/"+chat.ID+"/labels/"+label.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify label is assigned
	req = httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID, nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var result struct {
		Chat Chat `json:"chat"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	found := false
	for _, labelID := range result.Chat.LabelIDs {
		if labelID == label.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected label to be assigned to chat")
	}
}

// [REQ:LABEL-003] Test assign multiple labels to chat
func TestAssignMultipleLabels(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)
	label1 := createTestLabel(t, ts, "Work", "#3b82f6")
	label2 := createTestLabel(t, ts, "Personal", "#10b981")

	// Assign first label
	req := httptest.NewRequest("PUT", "/api/v1/chats/"+chat.ID+"/labels/"+label1.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	// Assign second label
	req = httptest.NewRequest("PUT", "/api/v1/chats/"+chat.ID+"/labels/"+label2.ID, nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	// Verify both labels are assigned
	req = httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID, nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var result struct {
		Chat Chat `json:"chat"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(result.Chat.LabelIDs) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(result.Chat.LabelIDs))
	}
}

// [REQ:LABEL-003] Test assign label to nonexistent chat
func TestAssignLabelToNonexistentChat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	label := createTestLabel(t, ts, "Test", "#ef4444")

	req := httptest.NewRequest("PUT", "/api/v1/chats/00000000-0000-0000-0000-000000000000/labels/"+label.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// [REQ:LABEL-003] Test assign nonexistent label to chat
func TestAssignNonexistentLabel(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)

	req := httptest.NewRequest("PUT", "/api/v1/chats/"+chat.ID+"/labels/00000000-0000-0000-0000-000000000000", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// [REQ:LABEL-003] Test remove label from chat
func TestRemoveLabel(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)

	chat := createTestChat(t, ts)
	label := createTestLabel(t, ts, "ToRemove", "#ef4444")

	// Assign label
	req := httptest.NewRequest("PUT", "/api/v1/chats/"+chat.ID+"/labels/"+label.ID, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	// Remove label
	req = httptest.NewRequest("DELETE", "/api/v1/chats/"+chat.ID+"/labels/"+label.ID, nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify label is removed
	req = httptest.NewRequest("GET", "/api/v1/chats/"+chat.ID, nil)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	var result struct {
		Chat Chat `json:"chat"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	for _, labelID := range result.Chat.LabelIDs {
		if labelID == label.ID {
			t.Error("Label should have been removed from chat")
		}
	}
}

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

	var updated Label
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

	var updated Label
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

	var labels []Label
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
		Chat Chat `json:"chat"`
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

	var labels []Label
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
