package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestHandleGetDraftTargetsNoDB tests handleGetDraftTargets without database
func TestHandleGetDraftTargetsNoDB(t *testing.T) {
	db = nil

	req := httptest.NewRequest(http.MethodGet, "/api/v1/drafts/test-id/targets", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-draft-id"})
	w := httptest.NewRecorder()

	handleGetDraftTargets(w, req)

	// Should fail gracefully when database is unavailable
	if w.Code == http.StatusOK {
		t.Error("Expected error status when database is unavailable, got 200")
	}
}

// TestHandleGetDraftTargetsMissingDraft tests non-existent draft
func TestHandleGetDraftTargetsMissingDraft(t *testing.T) {
	

	req := httptest.NewRequest(http.MethodGet, "/api/v1/drafts/nonexistent-id/targets", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent-id"})
	w := httptest.NewRecorder()

	handleGetDraftTargets(w, req)

	// Should return error when draft doesn't exist
	if w.Code == http.StatusOK {
		t.Error("Expected error status for nonexistent draft, got 200")
	}
}

// TestHandleUpdateDraftTargetsNoDB tests handleUpdateDraftTargets without database
func TestHandleUpdateDraftTargetsNoDB(t *testing.T) {
	db = nil

	requestBody := UpdateTargetsRequest{
		Targets: []OperationalTargetUpdate{
			{
				ID:                 "target-1",
				Title:              "Updated title",
				Notes:              "Updated notes",
				Status:             "complete",
				LinkedRequirements: []string{"REQ-001"},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/drafts/test-id/targets", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "test-draft-id"})
	w := httptest.NewRecorder()

	handleUpdateDraftTargets(w, req)

	// Should fail gracefully when database is unavailable
	if w.Code == http.StatusOK {
		t.Error("Expected error status when database is unavailable, got 200")
	}
}

// TestHandleUpdateDraftTargetsInvalidJSON tests invalid JSON input
func TestHandleUpdateDraftTargetsInvalidJSON(t *testing.T) {
	

	req := httptest.NewRequest(http.MethodPut, "/api/v1/drafts/test-id/targets", bytes.NewReader([]byte("{invalid json")))
	req = mux.SetURLVars(req, map[string]string{"id": "test-draft-id"})
	w := httptest.NewRecorder()

	handleUpdateDraftTargets(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}

	var errorResp map[string]string
	json.NewDecoder(w.Body).Decode(&errorResp)
	if errorResp["error"] == "" {
		t.Error("Expected error message in response")
	}
}

// TestHandleUpdateDraftTargetsEmptyTargets tests empty targets list
func TestHandleUpdateDraftTargetsEmptyTargets(t *testing.T) {
	

	requestBody := UpdateTargetsRequest{
		Targets: []OperationalTargetUpdate{},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/drafts/test-id/targets", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "test-draft-id"})
	w := httptest.NewRecorder()

	handleUpdateDraftTargets(w, req)

	// Empty targets should be accepted (it's valid to update zero targets)
	// The request should fail at the draft lookup stage with mock DB
	if w.Code == http.StatusBadRequest {
		t.Error("Empty targets should not cause bad request error")
	}
}

// TestHandleUpdateDraftTargetsMultipleTargets tests updating multiple targets
func TestHandleUpdateDraftTargetsMultipleTargets(t *testing.T) {
	

	requestBody := UpdateTargetsRequest{
		Targets: []OperationalTargetUpdate{
			{
				ID:                 "target-1",
				Title:              "First target",
				Notes:              "First notes",
				Status:             "complete",
				LinkedRequirements: []string{"REQ-001"},
			},
			{
				ID:                 "target-2",
				Title:              "Second target",
				Notes:              "Second notes",
				Status:             "pending",
				LinkedRequirements: []string{"REQ-002", "REQ-003"},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/drafts/test-id/targets", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "test-draft-id"})
	w := httptest.NewRecorder()

	handleUpdateDraftTargets(w, req)

	// Should handle multiple targets in request (will fail at DB lookup with mock)
	if w.Code == http.StatusBadRequest {
		t.Error("Multiple targets should not cause bad request error")
	}
}

// TestHandleUpdateDraftTargetsNewTarget tests adding a new target
func TestHandleUpdateDraftTargetsNewTarget(t *testing.T) {
	

	requestBody := UpdateTargetsRequest{
		Targets: []OperationalTargetUpdate{
			{
				ID:                 "new-target-id",
				Title:              "Brand new target",
				Notes:              "This target didn't exist before",
				Status:             "pending",
				LinkedRequirements: []string{},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/drafts/test-id/targets", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "test-draft-id"})
	w := httptest.NewRecorder()

	handleUpdateDraftTargets(w, req)

	// New target creation should be handled (will fail at DB lookup with mock)
	if w.Code == http.StatusBadRequest {
		t.Error("New target should not cause bad request error")
	}
}

// TestUpdateTargetsRequestStructure tests the UpdateTargetsRequest structure
func TestUpdateTargetsRequestStructure(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		wantErr  bool
		validate func(*testing.T, UpdateTargetsRequest)
	}{
		{
			name:    "valid request with single target",
			jsonStr: `{"targets":[{"id":"t1","title":"Test","notes":"Notes","status":"complete","linked_requirement_ids":["REQ-001"]}]}`,
			wantErr: false,
			validate: func(t *testing.T, req UpdateTargetsRequest) {
				if len(req.Targets) != 1 {
					t.Errorf("Expected 1 target, got %d", len(req.Targets))
				}
				if req.Targets[0].ID != "t1" {
					t.Errorf("Expected target ID 't1', got %s", req.Targets[0].ID)
				}
				if len(req.Targets[0].LinkedRequirements) != 1 {
					t.Errorf("Expected 1 linked requirement, got %d", len(req.Targets[0].LinkedRequirements))
				}
			},
		},
		{
			name:    "valid request with multiple targets",
			jsonStr: `{"targets":[{"id":"t1","title":"First","notes":"","status":"pending","linked_requirement_ids":[]},{"id":"t2","title":"Second","notes":"Notes","status":"complete","linked_requirement_ids":["REQ-001","REQ-002"]}]}`,
			wantErr: false,
			validate: func(t *testing.T, req UpdateTargetsRequest) {
				if len(req.Targets) != 2 {
					t.Errorf("Expected 2 targets, got %d", len(req.Targets))
				}
			},
		},
		{
			name:    "invalid JSON",
			jsonStr: `{invalid}`,
			wantErr: true,
		},
		{
			name:    "empty targets array",
			jsonStr: `{"targets":[]}`,
			wantErr: false,
			validate: func(t *testing.T, req UpdateTargetsRequest) {
				if len(req.Targets) != 0 {
					t.Errorf("Expected 0 targets, got %d", len(req.Targets))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req UpdateTargetsRequest
			err := json.Unmarshal([]byte(tt.jsonStr), &req)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.validate != nil {
				tt.validate(t, req)
			}
		})
	}
}
