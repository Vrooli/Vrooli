package backlog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"swarm-manager/internal/testutil"
)

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
		"status": "ready",
		"tags":   []string{"new"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/update-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "update-test"})
	w := httptest.NewRecorder()

	h.Update(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Status != StatusReady {
		t.Errorf("expected updated status, got '%s'", resp.Item.Status)
	}
	if got := resp.Item.Tags; len(got) != 1 || got[0] != "new" {
		t.Errorf("expected updated tags, got %v", got)
	}

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "update-test", "spec.json"))
	if saved.Status != StatusReady {
		t.Errorf("expected status ready, got '%s'", saved.Status)
	}
	if saved.Title != "Update Test" {
		t.Errorf("expected title to remain unchanged, got %q", saved.Title)
	}
	if saved.Priority != 5 {
		t.Errorf("expected priority to remain unchanged, got %d", saved.Priority)
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

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/update-test", strings.NewReader(`{
		"scope": "scenarios/swarm-manager"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "update-test"})
	w := httptest.NewRecorder()

	h.Update(w, req)

	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), `unknown field "scope"`) {
		t.Fatalf("expected unknown field error, got: %s", w.Body.String())
	}
}

func TestUpdate_PreservesOmittedFields(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:        "preserve-test",
		Title:       "Preserve Test",
		Description: "Keep me",
		Status:      StatusBacklog,
		Priority:    7,
		Tags:        []string{"alpha", "beta"},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	})

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/preserve-test", strings.NewReader(`{"status":"ready"}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "preserve-test"})
	w := httptest.NewRecorder()

	h.Update(w, req)
	testutil.AssertStatusOK(t, w)

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "preserve-test", "spec.json"))
	if saved.Title != "Preserve Test" {
		t.Fatalf("expected title to remain unchanged, got %q", saved.Title)
	}
	if saved.Description != "Keep me" {
		t.Fatalf("expected description to remain unchanged, got %q", saved.Description)
	}
	if saved.Priority != 7 {
		t.Fatalf("expected priority to remain unchanged, got %d", saved.Priority)
	}
	if len(saved.Tags) != 2 || saved.Tags[0] != "alpha" || saved.Tags[1] != "beta" {
		t.Fatalf("expected tags to remain unchanged, got %v", saved.Tags)
	}
}

func TestUpdate_ClearsFields(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindResearch, BacklogItem{
		Name:            "clear-test",
		Title:           "Clear Test",
		Description:     "To be cleared",
		Status:          StatusBacklog,
		Priority:        4,
		Tags:            []string{"stale"},
		Created:         "2026-01-28T00:00:00Z",
		Updated:         "2026-01-28T00:00:00Z",
		DependsOn:       []string{"idea/alpha"},
		Initiative:      "release-hardening",
		Effort:          "L",
		AcceptanceAllow: []string{"api/**"},
		AcceptanceDeny:  []string{"secrets/**"},
	})

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/research/clear-test", strings.NewReader(`{
		"description":"",
		"tags":[],
		"depends_on":[],
		"initiative":"",
		"effort":"",
		"acceptance_allow":[],
		"acceptance_deny":[]
	}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "research", "name": "clear-test"})
	w := httptest.NewRecorder()

	h.Update(w, req)
	testutil.AssertStatusOK(t, w)

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "research", "clear-test", "spec.json"))
	if saved.Description != "" {
		t.Fatalf("expected description to be cleared, got %q", saved.Description)
	}
	if len(saved.Tags) != 0 {
		t.Fatalf("expected tags to be cleared, got %v", saved.Tags)
	}
	if len(saved.DependsOn) != 0 {
		t.Fatalf("expected depends_on to be cleared, got %v", saved.DependsOn)
	}
	if saved.Initiative != "" {
		t.Fatalf("expected initiative to be cleared, got %q", saved.Initiative)
	}
	if saved.Effort != "" {
		t.Fatalf("expected effort to be cleared, got %q", saved.Effort)
	}
	if len(saved.AcceptanceAllow) != 0 {
		t.Fatalf("expected acceptance_allow to be cleared, got %v", saved.AcceptanceAllow)
	}
	if len(saved.AcceptanceDeny) != 0 {
		t.Fatalf("expected acceptance_deny to be cleared, got %v", saved.AcceptanceDeny)
	}
}

func TestUpdate_RejectsNullField(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:        "null-test",
		Title:       "Null Test",
		Description: "Original",
		Status:      StatusBacklog,
		Priority:    5,
		Tags:        []string{"old"},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	})

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/null-test", strings.NewReader(`{"description":null}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "null-test"})
	w := httptest.NewRecorder()

	h.Update(w, req)
	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), "description cannot be null") {
		t.Fatalf("expected null-field error, got: %s", w.Body.String())
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
		"status": "failed",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/failed-test", bytes.NewReader(body))
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
		"status": "queued",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/queued-reject", bytes.NewReader(body))
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
		"status": "in_progress",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/inprog-reject", bytes.NewReader(body))
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
		"depends_on": []string{"idea/alpha"},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/beta", bytes.NewReader(body))
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
	req = httptest.NewRequest("PATCH", "/api/v1/backlog/idea/beta", bytes.NewReader(body))
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
		"effort": "M",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/update-effort-test", bytes.NewReader(body))
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
		"effort": "XXXL",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/update-bad-effort", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "update-bad-effort"})
	w := httptest.NewRecorder()

	h.Update(w, req)
	testutil.AssertStatusBadRequest(t, w)
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
		"acceptance_allow": []string{"scenarios/target/src/**"},
		"acceptance_deny":  []string{"scenarios/target/test/**"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/update-acceptance-test", bytes.NewReader(body))
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

func TestUpdate_SpawnedFrom(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name: "update-sf-test", Title: "Update SF Test", Status: StatusBacklog, Priority: 5,
		Tags: []string{}, Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindExecute, item)

	payload := map[string]any{
		"spawned_from": "research/my-research",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PATCH", "/api/v1/backlog/execute/update-sf-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "update-sf-test"})
	w := httptest.NewRecorder()

	h.Update(w, req)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.SpawnedFrom != "research/my-research" {
		t.Errorf("expected spawned_from 'research/my-research', got %q", resp.Item.SpawnedFrom)
	}

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "execute", "update-sf-test", "spec.json"))
	if saved.SpawnedFrom != "research/my-research" {
		t.Errorf("expected saved spawned_from 'research/my-research', got %q", saved.SpawnedFrom)
	}
}
