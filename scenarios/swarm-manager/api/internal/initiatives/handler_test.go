package initiatives

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"swarm-manager/internal/backlog"
)

func setupTestHandler(t *testing.T) *Handler {
	t.Helper()
	store := setupTestStore(t)
	loader := &mockBacklogLoader{items: map[string]backlog.BacklogItem{
		"idea/foo": {Status: backlog.StatusBacklog},
		"fix/bar":  {Status: backlog.StatusCompleted},
	}}
	svc := NewService(store, loader)
	return NewHandler(svc)
}

func requestWithVars(method, path string, body any, vars map[string]string) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if vars != nil {
		req = mux.SetURLVars(req, vars)
	}
	return req
}

func TestHandler_CreateAndList(t *testing.T) {
	h := setupTestHandler(t)

	// Create initiative.
	createReq := CreateRequest{
		Name:  "test-init",
		Title: "Test Initiative",
		Items: []string{"idea/foo", "fix/bar"},
	}
	rec := httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", createReq, nil))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var createResp InitiativeWithRollup
	if err := json.NewDecoder(rec.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResp.Initiative.Name != "test-init" {
		t.Errorf("expected name test-init, got %q", createResp.Initiative.Name)
	}
	if createResp.Rollup.Total != 2 {
		t.Errorf("expected rollup total 2, got %d", createResp.Rollup.Total)
	}

	// List initiatives.
	rec = httptest.NewRecorder()
	h.List(rec, requestWithVars("GET", "/api/v1/initiatives", nil, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		Items []InitiativeWithRollup `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResp.Items) != 1 {
		t.Fatalf("expected 1 initiative, got %d", len(listResp.Items))
	}
}

func TestHandler_Get(t *testing.T) {
	h := setupTestHandler(t)

	// Create first.
	createReq := CreateRequest{Name: "get-test", Title: "Get Test"}
	rec := httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", createReq, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", rec.Code)
	}

	// Get.
	rec = httptest.NewRecorder()
	h.Get(rec, requestWithVars("GET", "/api/v1/initiatives/get-test", nil, map[string]string{"name": "get-test"}))

	if rec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Get_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	rec := httptest.NewRecorder()
	h.Get(rec, requestWithVars("GET", "/api/v1/initiatives/missing", nil, map[string]string{"name": "missing"}))

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_Update(t *testing.T) {
	h := setupTestHandler(t)

	// Create.
	createReq := CreateRequest{Name: "upd-test", Title: "Original"}
	rec := httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", createReq, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", rec.Code)
	}

	// Update.
	updateReq := UpdateRequest{Title: "Updated", Status: "completed", Items: []string{"idea/foo"}}
	rec = httptest.NewRecorder()
	h.Update(rec, requestWithVars("PUT", "/api/v1/initiatives/upd-test", updateReq, map[string]string{"name": "upd-test"}))

	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp InitiativeWithRollup
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if resp.Initiative.Title != "Updated" {
		t.Errorf("expected title Updated, got %q", resp.Initiative.Title)
	}
	if resp.Initiative.Status != "completed" {
		t.Errorf("expected status completed, got %q", resp.Initiative.Status)
	}
}

func TestHandler_Update_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	updateReq := UpdateRequest{Title: "Updated", Status: "active"}
	rec := httptest.NewRecorder()
	h.Update(rec, requestWithVars("PUT", "/api/v1/initiatives/missing", updateReq, map[string]string{"name": "missing"}))

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_Update_BadStatus(t *testing.T) {
	h := setupTestHandler(t)

	// Create.
	createReq := CreateRequest{Name: "bad-status", Title: "Test"}
	rec := httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", createReq, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", rec.Code)
	}

	// Update with bad status.
	updateReq := UpdateRequest{Title: "Test", Status: "invalid"}
	rec = httptest.NewRecorder()
	h.Update(rec, requestWithVars("PUT", "/api/v1/initiatives/bad-status", updateReq, map[string]string{"name": "bad-status"}))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_Delete(t *testing.T) {
	h := setupTestHandler(t)

	// Create.
	createReq := CreateRequest{Name: "del-test", Title: "Delete Me"}
	rec := httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", createReq, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", rec.Code)
	}

	// Delete.
	rec = httptest.NewRecorder()
	h.Delete(rec, requestWithVars("DELETE", "/api/v1/initiatives/del-test", nil, map[string]string{"name": "del-test"}))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify deleted.
	rec = httptest.NewRecorder()
	h.Get(rec, requestWithVars("GET", "/api/v1/initiatives/del-test", nil, map[string]string{"name": "del-test"}))
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rec.Code)
	}
}

func TestHandler_Create_MissingName(t *testing.T) {
	h := setupTestHandler(t)

	rec := httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", CreateRequest{Title: "No Name"}, nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_Create_MissingTitle(t *testing.T) {
	h := setupTestHandler(t)

	rec := httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", CreateRequest{Name: "test"}, nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_Create_Duplicate(t *testing.T) {
	h := setupTestHandler(t)

	req := CreateRequest{Name: "dup", Title: "First"}
	rec := httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", req, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("first create: expected 201, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", req, nil))
	if rec.Code != http.StatusConflict {
		t.Errorf("duplicate create: expected 409, got %d", rec.Code)
	}
}

func TestHandler_AddItems_Success(t *testing.T) {
	h := setupTestHandler(t)

	// Create an initiative first.
	createReq := CreateRequest{Name: "add-test", Title: "Add Test", Items: []string{"idea/foo"}}
	rec := httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", createReq, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Add items.
	rec = httptest.NewRecorder()
	h.AddItems(rec, requestWithVars("POST", "/api/v1/initiatives/add-test/items",
		itemsRequest{Items: []string{"fix/bar"}},
		map[string]string{"name": "add-test"}))

	if rec.Code != http.StatusOK {
		t.Fatalf("add-items: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result InitiativeWithRollup
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Initiative.Items) != 2 {
		t.Errorf("expected 2 items after add, got %d", len(result.Initiative.Items))
	}
}

func TestHandler_AddItems_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	rec := httptest.NewRecorder()
	h.AddItems(rec, requestWithVars("POST", "/api/v1/initiatives/nonexistent/items",
		itemsRequest{Items: []string{"idea/foo"}},
		map[string]string{"name": "nonexistent"}))

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_AddItems_EmptyItems(t *testing.T) {
	h := setupTestHandler(t)

	rec := httptest.NewRecorder()
	h.AddItems(rec, requestWithVars("POST", "/api/v1/initiatives/test/items",
		itemsRequest{Items: []string{}},
		map[string]string{"name": "test"}))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_RemoveItems_Success(t *testing.T) {
	h := setupTestHandler(t)

	// Create an initiative with 2 items.
	createReq := CreateRequest{Name: "rm-test", Title: "Remove Test", Items: []string{"idea/foo", "fix/bar"}}
	rec := httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", createReq, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Remove one item.
	rec = httptest.NewRecorder()
	h.RemoveItems(rec, requestWithVars("DELETE", "/api/v1/initiatives/rm-test/items",
		itemsRequest{Items: []string{"idea/foo"}},
		map[string]string{"name": "rm-test"}))

	if rec.Code != http.StatusOK {
		t.Fatalf("remove-items: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result InitiativeWithRollup
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Initiative.Items) != 1 {
		t.Errorf("expected 1 item after remove, got %d", len(result.Initiative.Items))
	}
	if result.Initiative.Items[0] != "fix/bar" {
		t.Errorf("expected remaining item 'fix/bar', got %q", result.Initiative.Items[0])
	}
}

func TestHandler_AddItems_InvalidFormat(t *testing.T) {
	h := setupTestHandler(t)

	// Create an initiative first.
	createReq := CreateRequest{Name: "validate-test", Title: "Validate Test"}
	rec := httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", createReq, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	tests := []struct {
		name  string
		items []string
	}{
		{"no-slash", []string{"bad-ref"}},
		{"empty-kind", []string{"/name"}},
		{"empty-name", []string{"kind/"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.AddItems(rec, requestWithVars("POST", "/api/v1/initiatives/validate-test/items",
				itemsRequest{Items: tt.items},
				map[string]string{"name": "validate-test"}))

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}
