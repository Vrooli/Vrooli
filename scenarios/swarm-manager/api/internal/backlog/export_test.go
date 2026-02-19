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
)

// createQuestions writes clarify/questions.json for a backlog item.
func createQuestions(t *testing.T, tmpDir string, kind BacklogKind, name string, questions []clarifyQuestion) {
	t.Helper()
	qDir := filepath.Join(tmpDir, backlogKindDirs[kind], name, "clarify")
	testutil.MakeDir(t, qDir)
	data, err := json.MarshalIndent(questions, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(qDir, "questions.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// createSuggestions writes suggest/suggestions.json for a backlog item.
func createSuggestions(t *testing.T, tmpDir string, kind BacklogKind, name string, suggestions []suggestion) {
	t.Helper()
	sDir := filepath.Join(tmpDir, backlogKindDirs[kind], name, "suggest")
	testutil.MakeDir(t, sDir)
	data, err := json.MarshalIndent(suggestions, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sDir, "suggestions.json"), data, 0o644); err != nil {
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

func TestExport_WithClarifyQuestions(t *testing.T) {
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

	createQuestions(t, tmpDir, KindIdea, "test-item", []clarifyQuestion{
		{
			ID:         "q1",
			Question:   "What auth method?",
			Category:   "technical",
			Importance: "critical",
			Options:    []string{"OAuth 2.0", "JWT tokens", "Session-based"},
			Answer:     "",
			Notes:      "",
		},
		{
			ID:         "q2",
			Question:   "Target user base?",
			Category:   "users",
			Importance: "important",
			Options:    []string{"Developers", "End users", "Both"},
			Answer:     "Both",
			Notes:      "Both developers and consumers",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()

	h.Export(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Q1: What auth method?") {
		t.Error("expected Q1 text")
	}
	if !strings.Contains(body, "[ ] OAuth 2.0") {
		t.Error("expected unchecked OAuth option")
	}
	if !strings.Contains(body, "[x] Both") {
		t.Error("expected checked Both option")
	}
	if !strings.Contains(body, "Both developers and consumers") {
		t.Error("expected notes for Q2")
	}
}

func TestExport_WithSuggestions(t *testing.T) {
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

	createSuggestions(t, tmpDir, KindIdea, "test-item", []suggestion{
		{
			ID:        "s1",
			Title:     "Use WebSocket",
			Impact:    "high",
			Category:  "architecture",
			Rationale: "Reduces latency by 10x",
			Accepted:  false,
		},
		{
			ID:        "s2",
			Title:     "Add caching",
			Impact:    "medium",
			Category:  "ux",
			Rationale: "Improves mobile experience",
			Accepted:  true,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()

	h.Export(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Use WebSocket") {
		t.Error("expected S1 title")
	}
	if !strings.Contains(body, "[ ] Accept this suggestion") {
		t.Error("expected unchecked accept checkbox for S1")
	}
	if !strings.Contains(body, "[x] Accept this suggestion") {
		t.Error("expected checked accept checkbox for S2")
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
	createQuestions(t, tmpDir, KindIdea, "toggle-test", []clarifyQuestion{
		{ID: "q1", Question: "Test question?", Category: "tech", Importance: "high"},
	})

	// Default: questions included.
	bodyDefault := exportWithBody(t, h, `{}`)
	if !strings.Contains(bodyDefault, "Clarify Questions") {
		t.Error("expected clarify questions by default")
	}

	// Explicitly off.
	bodyOff := exportWithBody(t, h, `{"includeClarifyQuestions": false}`)
	if strings.Contains(bodyOff, "Clarify Questions") {
		t.Error("expected clarify questions to be omitted when toggled off")
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
	createSuggestions(t, tmpDir, KindIdea, "toggle-test", []suggestion{
		{ID: "s1", Title: "Test suggestion", Impact: "high", Category: "arch", Accepted: false},
	})

	bodyDefault := exportWithBody(t, h, `{}`)
	if !strings.Contains(bodyDefault, "Suggestions") {
		t.Error("expected suggestions by default")
	}

	bodyOff := exportWithBody(t, h, `{"includeSuggestions": false}`)
	if strings.Contains(bodyOff, "Suggestions") {
		t.Error("expected suggestions to be omitted when toggled off")
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
