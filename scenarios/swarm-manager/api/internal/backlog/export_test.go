package backlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/testutil"
)

// createWorkshopRound writes workshop/round-001.json for a backlog item.
func createWorkshopRound(t *testing.T, tmpDir string, kind BacklogKind, name string, round WorkshopRound) {
	t.Helper()
	wDir := filepath.Join(tmpDir, backlogKindDirs[kind], name, "workshop")
	testutil.MakeDir(t, wDir)
	data, err := json.MarshalIndent(round, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	fileName := fmt.Sprintf("round-%03d.json", round.RoundNum)
	if err := os.WriteFile(filepath.Join(wDir, fileName), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// createPRD writes archive/PRD.md for a backlog item.
func createPRD(t *testing.T, tmpDir string, kind BacklogKind, name string, content string) {
	t.Helper()
	archiveDir := filepath.Join(tmpDir, backlogKindDirs[kind], name, "archive")
	testutil.MakeDir(t, archiveDir)
	if err := os.WriteFile(filepath.Join(archiveDir, "PRD.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExport_EmptyBacklog(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()

	h.Export(w, req)

	testutil.AssertStatusOK(t, w)

	body := w.Body.String()
	if !strings.Contains(body, "version: 1") {
		t.Error("expected version: 1 in frontmatter")
	}
	if !strings.Contains(body, "items_count: 0") {
		t.Error("expected items_count: 0")
	}
}

func TestExport_SingleItem(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:        "my-app",
		Title:       "My Application",
		Description: "Build a great app",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{"saas", "tools"},
		Created:     "2026-02-10T00:00:00Z",
		Updated:     "2026-02-15T00:00:00Z",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()

	h.Export(w, req)

	testutil.AssertStatusOK(t, w)

	body := w.Body.String()
	if !strings.Contains(body, "<!-- item:idea/my-app -->") {
		t.Error("expected item marker")
	}
	if !strings.Contains(body, "My Application") {
		t.Error("expected title")
	}
	if !strings.Contains(body, "Build a great app") {
		t.Error("expected description")
	}
	if !strings.Contains(body, "saas, tools") {
		t.Error("expected tags")
	}
	if !strings.Contains(body, "items_count: 1") {
		t.Error("expected items_count: 1")
	}
	if w.Header().Get("Content-Type") != "text/markdown; charset=utf-8" {
		t.Errorf("expected text/markdown content type, got %s", w.Header().Get("Content-Type"))
	}
}

func TestExport_WithDecisions(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:     "test-item",
		Title:    "Test Item",
		Status:   StatusBacklog,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-02-10T00:00:00Z",
		Updated:  "2026-02-10T00:00:00Z",
	})

	selectedKey := "A"
	createWorkshopRound(t, tmpDir, KindIdea, "test-item", WorkshopRound{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 1, "testable": 0, "risk_awareness": 0},
		Items: []WorkshopItem{
			{ID: "w1", Type: "decision", Topic: "What auth method?", Options: []WorkshopOption{{Key: "A", Label: "OAuth", Rationale: "Industry standard"}, {Key: "B", Label: "JWT", Rationale: "Stateless"}}, Selected: nil},
			{ID: "w2", Type: "decision", Topic: "Target user base?", Options: []WorkshopOption{{Key: "A", Label: "Developers", Rationale: "Primary audience"}, {Key: "B", Label: "End users", Rationale: "Broad reach"}, {Key: "C", Label: "Both", Rationale: "Maximum coverage"}}, Selected: &selectedKey},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()

	h.Export(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Workshop Items") {
		t.Error("expected Workshop Items heading")
	}
	if !strings.Contains(body, "D1: What auth method?") {
		t.Error("expected D1 text")
	}
	if !strings.Contains(body, "[ ] **A**: OAuth") {
		t.Error("expected unchecked option for D1")
	}
	if !strings.Contains(body, "[x] **A**: Developers") {
		t.Error("expected checked option for D2")
	}
}

func TestExport_WithDecisionOptions(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:     "test-item",
		Title:    "Test Item",
		Status:   StatusBacklog,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-02-10T00:00:00Z",
		Updated:  "2026-02-10T00:00:00Z",
	})

	selectedKey := "B"
	createWorkshopRound(t, tmpDir, KindIdea, "test-item", WorkshopRound{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 1, "testable": 0, "risk_awareness": 0},
		Items: []WorkshopItem{
			{ID: "w1", Type: "decision", Topic: "Transport protocol", Options: []WorkshopOption{{Key: "A", Label: "Use WebSocket", Rationale: "For real-time updates"}, {Key: "B", Label: "Use SSE", Rationale: "Simpler implementation"}}, Selected: nil},
			{ID: "w2", Type: "decision", Topic: "Caching strategy", Options: []WorkshopOption{{Key: "A", Label: "No caching", Rationale: "Simple"}, {Key: "B", Label: "Add caching", Rationale: "Improves mobile experience"}}, Selected: &selectedKey},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()

	h.Export(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Use WebSocket") {
		t.Error("expected option label in output")
	}
	if !strings.Contains(body, "[ ] **A**: Use WebSocket") {
		t.Error("expected unchecked option for D1")
	}
	if !strings.Contains(body, "[x] **B**: Add caching") {
		t.Error("expected checked option for D2 selected choice")
	}
}

func TestExport_WithPRD(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:     "test-item",
		Title:    "Test Item",
		Status:   StatusBacklog,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-02-10T00:00:00Z",
		Updated:  "2026-02-10T00:00:00Z",
	})

	createPRD(t, tmpDir, KindIdea, "test-item", "# Product Requirements\n\nThis is the PRD content.")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()

	h.Export(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "<details>") {
		t.Error("expected PRD in details tag")
	}
	if !strings.Contains(body, "This is the PRD content.") {
		t.Error("expected PRD content")
	}
}

func TestExport_MultipleItems_SortedByPriority(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:     "low-priority",
		Title:    "Low Priority Item",
		Status:   StatusBacklog,
		Priority: 8,
		Tags:     []string{},
		Created:  "2026-02-10T00:00:00Z",
		Updated:  "2026-02-10T00:00:00Z",
	})

	createTestItem(t, tmpDir, KindFix, BacklogItem{
		Name:     "high-priority",
		Title:    "High Priority Item",
		Status:   StatusBacklog,
		Priority: 1,
		Tags:     []string{},
		Created:  "2026-02-10T00:00:00Z",
		Updated:  "2026-02-10T00:00:00Z",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()

	h.Export(w, req)

	body := w.Body.String()

	highIdx := strings.Index(body, "High Priority Item")
	lowIdx := strings.Index(body, "Low Priority Item")
	if highIdx == -1 || lowIdx == -1 {
		t.Fatal("expected both items in export")
	}
	if highIdx > lowIdx {
		t.Error("expected high priority item before low priority item")
	}
}

func TestExport_StatusFilter(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:     "active-item",
		Title:    "Active Item",
		Status:   StatusBacklog,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-02-10T00:00:00Z",
		Updated:  "2026-02-10T00:00:00Z",
	})

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:     "archived-item",
		Title:    "Archived Item",
		Status:   StatusArchived,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-02-10T00:00:00Z",
		Updated:  "2026-02-10T00:00:00Z",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()

	h.Export(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Active Item") {
		t.Error("expected active item in export")
	}
	if strings.Contains(body, "Archived Item") {
		t.Error("expected archived item to be excluded by default")
	}
}

func TestExport_NewItemTemplate(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()

	h.Export(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "New Item Template") {
		t.Error("expected new item template section")
	}
	if !strings.Contains(body, "<!-- item:NEW -->") {
		t.Error("expected template marker")
	}
}

// exportWithBody sends a POST with the given JSON body and returns the response body.
func exportWithBody(t *testing.T, h *Handler, jsonBody string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", bytes.NewReader([]byte(jsonBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Export(w, req)
	testutil.AssertStatusOK(t, w)
	return w.Body.String()
}

func TestExport_ToggleClarifyQuestionsOff(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:     "toggle-test",
		Title:    "Toggle Test",
		Status:   StatusBacklog,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-02-10T00:00:00Z",
		Updated:  "2026-02-10T00:00:00Z",
	})
	createWorkshopRound(t, tmpDir, KindIdea, "toggle-test", WorkshopRound{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 1, "approach_solid": 1, "testable": 0, "risk_awareness": 0},
		Items: []WorkshopItem{
			{ID: "w1", Type: "decision", Topic: "Test decision?", Options: []WorkshopOption{{Key: "A", Label: "Yes", Rationale: "Confirm"}}},
		},
	})

	// Default: workshop items included.
	bodyDefault := exportWithBody(t, h, `{}`)
	if !strings.Contains(bodyDefault, "Workshop Items") {
		t.Error("expected workshop items by default")
	}

	// Both includeClarifyQuestions and includeSuggestions off => workshop excluded.
	bodyOff := exportWithBody(t, h, `{"includeClarifyQuestions": false, "includeSuggestions": false}`)
	if strings.Contains(bodyOff, "Workshop Items") {
		t.Error("expected workshop items to be omitted when both clarify and suggestions toggled off")
	}
}

func TestExport_ToggleSuggestionsOff(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:     "toggle-test",
		Title:    "Toggle Test",
		Status:   StatusBacklog,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-02-10T00:00:00Z",
		Updated:  "2026-02-10T00:00:00Z",
	})
	createWorkshopRound(t, tmpDir, KindIdea, "toggle-test", WorkshopRound{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 1, "approach_solid": 1, "testable": 0, "risk_awareness": 0},
		Items: []WorkshopItem{
			{ID: "w1", Type: "decision", Topic: "Test suggestion", Context: "Some details", Options: []WorkshopOption{{Key: "A", Label: "Accept", Rationale: "Good idea"}}},
		},
	})

	bodyDefault := exportWithBody(t, h, `{}`)
	if !strings.Contains(bodyDefault, "Workshop Items") {
		t.Error("expected workshop items by default")
	}

	// Both includeClarifyQuestions and includeSuggestions off => workshop excluded.
	bodyOff := exportWithBody(t, h, `{"includeClarifyQuestions": false, "includeSuggestions": false}`)
	if strings.Contains(bodyOff, "Workshop Items") {
		t.Error("expected workshop items to be omitted when both clarify and suggestions toggled off")
	}
}

func TestExport_ToggleNotesOff(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:     "toggle-test",
		Title:    "Toggle Test",
		Status:   StatusBacklog,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-02-10T00:00:00Z",
		Updated:  "2026-02-10T00:00:00Z",
	})

	bodyDefault := exportWithBody(t, h, `{}`)
	if !strings.Contains(bodyDefault, "### Notes") {
		t.Error("expected notes section by default")
	}

	// Also disable template since it contains "### Notes" too.
	bodyOff := exportWithBody(t, h, `{"includeNotes": false, "includeTemplate": false}`)
	if strings.Contains(bodyOff, "### Notes") {
		t.Error("expected notes to be omitted when toggled off")
	}
}

func TestExport_ToggleTemplateOff(t *testing.T) {
	h, _ := setupTestHandler(t)

	bodyDefault := exportWithBody(t, h, `{}`)
	if !strings.Contains(bodyDefault, "New Item Template") {
		t.Error("expected template by default")
	}

	bodyOff := exportWithBody(t, h, `{"includeTemplate": false}`)
	if strings.Contains(bodyOff, "New Item Template") {
		t.Error("expected template to be omitted when toggled off")
	}
}
