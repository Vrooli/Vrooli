package backlog

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"swarm-manager/internal/testutil"
	"testing"

	"github.com/gorilla/mux"
)

func TestList_Empty(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/backlog", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogListResponse](t, w)
	if len(resp.Items) != 0 {
		t.Errorf("expected empty list, got %d items", len(resp.Items))
	}
}

func TestList_WithItems(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item1 := BacklogItem{
		Name:        "idea-one",
		Title:       "Idea One",
		Description: "First idea",
		Status:      StatusBacklog,
		Priority:    1,
		Tags:        []string{"test"},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	item2 := BacklogItem{
		Name:        "fix-two",
		Title:       "Fix Two",
		Description: "Second item",
		Status:      StatusReady,
		Priority:    2,
		Tags:        []string{"fix"},
		Created:     "2026-01-27T00:00:00Z",
		Updated:     "2026-01-27T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item1)
	createTestItem(t, rootDir, KindFix, item2)

	req := httptest.NewRequest("GET", "/api/v1/backlog", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogListResponse](t, w)
	if len(resp.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].Priority > resp.Items[1].Priority {
		t.Error("items should be sorted by priority ascending")
	}
}

func TestList_ExcludesArchivedByDefault(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	active := BacklogItem{
		Name:        "active-item",
		Title:       "Active Item",
		Description: "An active item",
		Status:      StatusBacklog,
		Priority:    1,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	archived := BacklogItem{
		Name:        "archived-item",
		Title:       "Archived Item",
		Description: "An archived item",
		Status:      StatusCompleted,
		Priority:    2,
		Tags:        []string{},
		Created:     "2026-01-27T00:00:00Z",
		Updated:     "2026-01-27T00:00:00Z",
		ArchivedAt:  strPtr("2026-01-01T00:00:00Z"),
	}
	createTestItem(t, rootDir, KindIdea, active)
	createTestItem(t, rootDir, KindIdea, archived)

	// Default: no archived param → archived excluded
	req := httptest.NewRequest("GET", "/api/v1/backlog", nil)
	w := httptest.NewRecorder()
	h.List(w, req)
	testutil.AssertStatusOK(t, w)
	resp := testutil.DecodeJSON[backlogListResponse](t, w)
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 non-archived item, got %d", len(resp.Items))
	}
	if resp.Items[0].Name != "active-item" {
		t.Errorf("expected active-item, got %s", resp.Items[0].Name)
	}

	// archived=all → everything returned
	req = httptest.NewRequest("GET", "/api/v1/backlog?archived=all", nil)
	w = httptest.NewRecorder()
	h.List(w, req)
	testutil.AssertStatusOK(t, w)
	resp = testutil.DecodeJSON[backlogListResponse](t, w)
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items with archived=all, got %d", len(resp.Items))
	}

	// archived=true → only archived
	req = httptest.NewRequest("GET", "/api/v1/backlog?archived=true", nil)
	w = httptest.NewRecorder()
	h.List(w, req)
	testutil.AssertStatusOK(t, w)
	resp = testutil.DecodeJSON[backlogListResponse](t, w)
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 archived item, got %d", len(resp.Items))
	}
	if resp.Items[0].Name != "archived-item" {
		t.Errorf("expected archived-item, got %s", resp.Items[0].Name)
	}
}

func TestGet_Found(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:        "get-test",
		Title:       "Get Test",
		Description: "Test get endpoint",
		Status:      StatusBacklog,
		Priority:    1,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest("GET", "/api/v1/backlog/idea/get-test", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "get-test"})
	w := httptest.NewRecorder()

	h.Get(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	result := resp.Item
	if result.Name != "get-test" {
		t.Errorf("expected name 'get-test', got '%s'", result.Name)
	}
	if result.Kind != KindIdea {
		t.Errorf("expected kind 'idea', got '%s'", result.Kind)
	}
}

func TestGet_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/backlog/idea/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "nonexistent"})
	w := httptest.NewRecorder()

	h.Get(w, req)

	testutil.AssertStatusNotFound(t, w)
}

func TestListFiles_And_GetFileContent(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:        "files-test",
		Title:       "Files Test",
		Description: "",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "files-test", "notes.md"), "hello")

	req := httptest.NewRequest("GET", "/api/v1/backlog/idea/files-test/files", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "files-test"})
	w := httptest.NewRecorder()

	h.ListFiles(w, req)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogFilesResponse](t, w)
	if len(resp.Files) == 0 {
		t.Fatalf("expected files")
	}

	contentReq := httptest.NewRequest("GET", "/api/v1/backlog/idea/files-test/files/notes.md", nil)
	contentReq = mux.SetURLVars(contentReq, map[string]string{"kind": "idea", "name": "files-test", "filepath": "notes.md"})
	contentRec := httptest.NewRecorder()

	h.GetFileContent(contentRec, contentReq)
	testutil.AssertStatusOK(t, contentRec)
	if strings.TrimSpace(contentRec.Body.String()) != "hello" {
		t.Errorf("expected file content")
	}
}

func TestGetFileContent_PathIsDirectory(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:    "dir-test",
		Title:   "Dir Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	})

	// Create a subdirectory inside the item folder
	subDir := filepath.Join(rootDir, "ideas", "dir-test", "clarify")
	testutil.MakeDir(t, subDir)

	req := httptest.NewRequest("GET", "/api/v1/backlog/idea/dir-test/files/clarify", nil)
	req = mux.SetURLVars(req, map[string]string{
		"kind":     "idea",
		"name":     "dir-test",
		"filepath": "clarify",
	})
	w := httptest.NewRecorder()

	h.GetFileContent(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestList_FilterBySpawnedFrom(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Create two items, one with spawned_from
	item1 := BacklogItem{
		Name: "spawned-a", Title: "Spawned A", Status: StatusBacklog, Priority: 5,
		Tags: []string{}, Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
		SpawnedFrom: "research/my-research",
	}
	item2 := BacklogItem{
		Name: "unrelated", Title: "Unrelated", Status: StatusBacklog, Priority: 5,
		Tags: []string{}, Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindExecute, item1)
	createTestItem(t, rootDir, KindExecute, item2)

	req := httptest.NewRequest("GET", "/api/v1/backlog?spawned_from=research/my-research", nil)
	w := httptest.NewRecorder()

	h.List(w, req)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogListResponse](t, w)
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].Name != "spawned-a" {
		t.Errorf("expected 'spawned-a', got %q", resp.Items[0].Name)
	}
}

func TestList_FilterByScenario(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name: "sm-task", Title: "SM Task", Status: StatusBacklog, Priority: 5,
		Tags: []string{}, Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
		AcceptanceAllow: []string{"scenarios/swarm-manager/**"},
	})
	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "tg-fix", Title: "TG Fix", Status: StatusBacklog, Priority: 3,
		Tags: []string{}, Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
		AcceptanceAllow: []string{"scenarios/test-genie/**"},
	})
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "no-scope", Title: "No Scope", Status: StatusBacklog, Priority: 7,
		Tags: []string{}, Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	})

	t.Run("single scenario", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/backlog?scenario=swarm-manager", nil)
		w := httptest.NewRecorder()
		h.List(w, req)
		testutil.AssertStatusOK(t, w)
		resp := testutil.DecodeJSON[backlogListResponse](t, w)
		if len(resp.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(resp.Items))
		}
		if resp.Items[0].Name != "sm-task" {
			t.Errorf("expected 'sm-task', got %q", resp.Items[0].Name)
		}
	})

	t.Run("multiple scenarios", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/backlog?scenario=swarm-manager,test-genie", nil)
		w := httptest.NewRecorder()
		h.List(w, req)
		testutil.AssertStatusOK(t, w)
		resp := testutil.DecodeJSON[backlogListResponse](t, w)
		if len(resp.Items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(resp.Items))
		}
	})

	t.Run("no filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/backlog", nil)
		w := httptest.NewRecorder()
		h.List(w, req)
		testutil.AssertStatusOK(t, w)
		resp := testutil.DecodeJSON[backlogListResponse](t, w)
		if len(resp.Items) != 3 {
			t.Fatalf("expected 3 items, got %d", len(resp.Items))
		}
	})

	t.Run("no match", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/backlog?scenario=nonexistent", nil)
		w := httptest.NewRecorder()
		h.List(w, req)
		testutil.AssertStatusOK(t, w)
		resp := testutil.DecodeJSON[backlogListResponse](t, w)
		if len(resp.Items) != 0 {
			t.Fatalf("expected 0 items, got %d", len(resp.Items))
		}
	})
}

// ---------------------------------------------------------------------------
// List blocking map tests
// ---------------------------------------------------------------------------

type listBlockingInfo struct {
	Blocked         bool     `json:"blocked"`
	BlockingDepKeys []string `json:"blocking_dep_keys"`
	AllForceable    bool     `json:"all_forceable"`
}

type backlogListWithBlockingResponse struct {
	Items    []BacklogItem               `json:"items"`
	Blocking map[string]listBlockingInfo `json:"blocking"`
}

func TestList_IncludesBlockingMap(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Create a dep in "backlog" status and a child that depends on it.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "list-dep", Title: "List Dep", Status: StatusBacklog, Priority: 1,
		Tags: []string{}, Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
	})
	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "list-child", Title: "List Child", Status: StatusBacklog, Priority: 2,
		Tags: []string{}, Created: "2026-01-28T00:00:00Z", Updated: "2026-01-28T00:00:00Z",
		DependsOn: []string{"idea/list-dep"},
	})

	req := httptest.NewRequest("GET", "/api/v1/backlog", nil)
	w := httptest.NewRecorder()

	h.List(w, req)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[backlogListWithBlockingResponse](t, w)
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}

	info, found := resp.Blocking["fix/list-child"]
	if !found {
		t.Fatalf("expected blocking info for fix/list-child, got blocking=%+v", resp.Blocking)
	}
	if !info.Blocked {
		t.Error("expected Blocked=true for child with unmet dep")
	}
	if len(info.BlockingDepKeys) != 1 || info.BlockingDepKeys[0] != "idea/list-dep" {
		t.Errorf("expected BlockingDepKeys=[idea/list-dep], got %v", info.BlockingDepKeys)
	}
	if !info.AllForceable {
		t.Error("expected AllForceable=true")
	}

	// The dep itself should not appear in the blocking map.
	if _, found := resp.Blocking["idea/list-dep"]; found {
		t.Error("items without blocked deps should not appear in blocking map")
	}
}
