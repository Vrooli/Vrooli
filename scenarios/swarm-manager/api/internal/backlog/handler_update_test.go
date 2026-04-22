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

	"swarm-manager/internal/testutil"

	"github.com/gorilla/mux"
)

// doUpdate sends a PATCH request to the Update handler and returns the recorder.
func doUpdate(t *testing.T, h *Handler, kind, name string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/"+kind+"/"+name, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": kind, "name": name})
	w := httptest.NewRecorder()
	h.Update(w, req)
	return w
}

// doUpdateRaw sends a PATCH with a raw JSON string body.
func doUpdateRaw(t *testing.T, h *Handler, kind, name, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/"+kind+"/"+name, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": kind, "name": name})
	w := httptest.NewRecorder()
	h.Update(w, req)
	return w
}

// newTestItem creates a minimal BacklogItem for update tests.
func newTestItem(name, title string) BacklogItem {
	return BacklogItem{
		Name: name, Title: title, Description: "Desc",
		Status: StatusBacklog, Priority: 5, Tags: []string{},
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
}

func TestUpdate_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "update-test", Title: "Update Test", Description: "Original",
		Status: StatusBacklog, Priority: 5, Tags: []string{"old"},
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	})

	w := doUpdate(t, h, "idea", "update-test", map[string]any{
		"status": "ready", "tags": []string{"new"},
	})

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
		t.Errorf("expected title unchanged, got %q", saved.Title)
	}
	if saved.Priority != 5 {
		t.Errorf("expected priority unchanged, got %d", saved.Priority)
	}
}

func TestUpdate_RejectsUnknownField(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "update-test", Title: "Update Test", Description: "Original",
		Status: StatusBacklog, Priority: 5, Tags: []string{"old"},
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	})

	w := doUpdateRaw(t, h, "idea", "update-test", `{"scope": "scenarios/swarm-manager"}`)
	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), `unknown field "scope"`) {
		t.Fatalf("expected unknown field error, got: %s", w.Body.String())
	}
}

func TestUpdate_PreservesOmittedFields(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "preserve-test", Title: "Preserve Test", Description: "Keep me",
		Status: StatusBacklog, Priority: 7, Tags: []string{"alpha", "beta"},
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	})

	w := doUpdateRaw(t, h, "idea", "preserve-test", `{"status":"ready"}`)
	testutil.AssertStatusOK(t, w)

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "preserve-test", "spec.json"))
	if saved.Title != "Preserve Test" {
		t.Fatalf("expected title unchanged, got %q", saved.Title)
	}
	if saved.Description != "Keep me" {
		t.Fatalf("expected description unchanged, got %q", saved.Description)
	}
	if saved.Priority != 7 {
		t.Fatalf("expected priority unchanged, got %d", saved.Priority)
	}
	if len(saved.Tags) != 2 || saved.Tags[0] != "alpha" || saved.Tags[1] != "beta" {
		t.Fatalf("expected tags unchanged, got %v", saved.Tags)
	}
}

func TestUpdate_ClearsFields(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindResearch, BacklogItem{
		Name: "clear-test", Title: "Clear Test", Description: "To be cleared",
		Status: StatusBacklog, Priority: 4, Tags: []string{"stale"},
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
		DependsOn: []string{"idea/alpha"}, Initiative: "release-hardening",
		Effort: "L", AcceptanceAllow: []string{"api/**"}, AcceptanceDeny: []string{"secrets/**"},
	})

	w := doUpdate(t, h, "research", "clear-test", map[string]any{
		"description": "", "tags": []string{}, "depends_on": []string{},
		"initiative": "", "effort": "",
		"acceptance_allow": []string{}, "acceptance_deny": []string{},
	})
	testutil.AssertStatusOK(t, w)

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "research", "clear-test", "spec.json"))
	if saved.Description != "" {
		t.Fatalf("expected description cleared, got %q", saved.Description)
	}
	if len(saved.Tags) != 0 {
		t.Fatalf("expected tags cleared, got %v", saved.Tags)
	}
	if len(saved.DependsOn) != 0 {
		t.Fatalf("expected depends_on cleared, got %v", saved.DependsOn)
	}
	if saved.Initiative != "" {
		t.Fatalf("expected initiative cleared, got %q", saved.Initiative)
	}
	if saved.Effort != "" {
		t.Fatalf("expected effort cleared, got %q", saved.Effort)
	}
	if len(saved.AcceptanceAllow) != 0 {
		t.Fatalf("expected acceptance_allow cleared, got %v", saved.AcceptanceAllow)
	}
	if len(saved.AcceptanceDeny) != 0 {
		t.Fatalf("expected acceptance_deny cleared, got %v", saved.AcceptanceDeny)
	}
}

func TestUpdate_RejectsNullField(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, newTestItem("null-test", "Null Test"))

	w := doUpdateRaw(t, h, "idea", "null-test", `{"description":null}`)
	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), "description cannot be null") {
		t.Fatalf("expected null-field error, got: %s", w.Body.String())
	}
}

func TestSaveItem_PreservesUnknownSpecFields(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	raw := map[string]any{
		"name": "metadata-keep", "title": "Metadata Keep", "description": "desc",
		"status": "completed", "archived_at": "2026-01-01T00:00:00Z",
		"priority": 5, "tags": []string{"x"},
		"created": "2026-01-28T00:00:00Z", "updated": "2026-01-28T00:00:00Z",
		"kind": "idea", "archiveReason": "scenario deleted with archive=true",
		"sourceScenario": "web-console", "preservedFiles": []string{"PRD.md"},
		"archivedByHuman": true,
	}
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", "metadata-keep", "spec.json"), raw)

	item := BacklogItem{
		Name: "metadata-keep", Kind: KindIdea, Title: "Metadata Keep Updated",
		Description: "updated", Status: StatusCompleted,
		ArchivedAt: strPtr("2026-01-01T00:00:00Z"),
		Priority:   6, Tags: []string{"y"},
		Created: "2026-01-28T00:00:00Z", Updated: "2026-01-29T00:00:00Z",
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
	createTestItem(t, rootDir, KindIdea, newTestItem("failed-test", "Failed Test"))

	w := doUpdate(t, h, "idea", "failed-test", map[string]any{"status": "failed"})
	testutil.AssertStatusOK(t, w)

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "failed-test", "spec.json"))
	if saved.Status != StatusFailed {
		t.Errorf("expected status failed, got '%s'", saved.Status)
	}
}

func TestUpdate_QueuedStatus_Rejected(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, newTestItem("queued-reject", "Queued Reject"))

	w := doUpdate(t, h, "idea", "queued-reject", map[string]any{"status": "queued"})
	testutil.AssertStatus(t, w, http.StatusBadRequest)
}

func TestUpdate_InProgressStatus_Rejected(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, newTestItem("inprog-reject", "InProgress Reject"))

	w := doUpdate(t, h, "idea", "inprog-reject", map[string]any{"status": "in_progress"})
	testutil.AssertStatus(t, w, http.StatusBadRequest)
}

func TestUpdate_ChangeDependsOn(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "alpha", Title: "Alpha", Status: StatusBacklog, Priority: 5,
	})
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "beta", Title: "Beta", Status: StatusBacklog, Priority: 5,
	})

	w := doUpdate(t, h, "idea", "beta", map[string]any{"depends_on": []string{"idea/alpha"}})
	testutil.AssertStatus(t, w, http.StatusOK)

	item, err := h.store.LoadItem(KindIdea, "beta")
	if err != nil {
		t.Fatalf("LoadItem: %v", err)
	}
	if len(item.DependsOn) != 1 || item.DependsOn[0] != "idea/alpha" {
		t.Errorf("expected depends_on=['idea/alpha'], got %v", item.DependsOn)
	}

	w = doUpdate(t, h, "idea", "beta", map[string]any{"depends_on": []string{"idea/alpha"}})
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
	createTestItem(t, rootDir, KindIdea, newTestItem("update-effort-test", "Update Effort Test"))

	w := doUpdate(t, h, "idea", "update-effort-test", map[string]any{"effort": "M"})
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
	createTestItem(t, rootDir, KindIdea, newTestItem("update-bad-effort", "Update Bad Effort"))

	w := doUpdate(t, h, "idea", "update-bad-effort", map[string]any{"effort": "XXXL"})
	testutil.AssertStatusBadRequest(t, w)
}

func TestUpdate_Acceptance(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, newTestItem("update-acceptance-test", "Update Acceptance Test"))

	w := doUpdate(t, h, "idea", "update-acceptance-test", map[string]any{
		"acceptance_allow": []string{"scenarios/target/src/**"},
		"acceptance_deny":  []string{"scenarios/target/test/**"},
	})
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
	createTestItem(t, rootDir, KindExecute, newTestItem("update-sf-test", "Update SF Test"))

	w := doUpdate(t, h, "execute", "update-sf-test", map[string]any{"spawned_from": "research/my-research"})
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
