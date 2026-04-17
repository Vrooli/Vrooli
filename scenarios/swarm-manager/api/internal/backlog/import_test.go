package backlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"swarm-manager/internal/testutil"
	"testing"
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

// postImportRequest creates an import request with a markdown file and optional apply flag.
func postImportRequest(t *testing.T, markdown string, apply bool) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fw, err := mw.CreateFormFile("file", "backlog-export.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(markdown)); err != nil {
		t.Fatal(err)
	}

	if apply {
		if err := mw.WriteField("apply", "true"); err != nil {
			t.Fatal(err)
		}
	}

	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/import", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// importResp is a simplified representation of the JSON import response.
type importResp struct {
	DryRun  bool `json:"dry_run"`
	Changes []struct {
		Item    string   `json:"item"`
		Action  string   `json:"action"`
		Details []string `json:"details"`
	} `json:"changes"`
	Errors  []string `json:"errors"`
	Summary string   `json:"summary"`
}

func parseImportResp(t *testing.T, w *httptest.ResponseRecorder) importResp {
	t.Helper()
	testutil.AssertStatusOK(t, w)
	var resp importResp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nbody: %s", err, w.Body.String())
	}
	return resp
}

func TestImport_InvalidFrontmatter(t *testing.T) {
	h, _ := setupTestHandler(t)

	md := `# No frontmatter here
Just some markdown.
`
	req := postImportRequest(t, md, false)
	w := httptest.NewRecorder()
	h.Import(w, req)

	resp := parseImportResp(t, w)
	if len(resp.Errors) == 0 {
		t.Error("expected errors for missing frontmatter")
	}
}

func TestImport_WrongVersion(t *testing.T) {
	h, _ := setupTestHandler(t)

	md := `---
version: 2
items_count: 0
---
`
	req := postImportRequest(t, md, false)
	w := httptest.NewRecorder()
	h.Import(w, req)

	resp := parseImportResp(t, w)
	if len(resp.Errors) == 0 {
		t.Error("expected errors for wrong version")
	}
}

func TestImport_EmptyFile(t *testing.T) {
	h, _ := setupTestHandler(t)

	md := `---
version: 1
items_count: 0
---
`
	req := postImportRequest(t, md, false)
	w := httptest.NewRecorder()
	h.Import(w, req)

	resp := parseImportResp(t, w)
	if len(resp.Changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(resp.Changes))
	}
	if len(resp.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(resp.Errors), resp.Errors)
	}
}

func TestImport_DryRun_UpdateDescription(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:        "my-app",
		Title:       "My Application",
		Description: "Old description",
		Status:      StatusBacklog,
		Priority:    5,
		Tags:        []string{"saas"},
		Created:     "2026-02-10T00:00:00Z",
		Updated:     "2026-02-15T00:00:00Z",
	})

	md := `---
version: 1
items_count: 1
---

<!-- item:idea/my-app -->
## My Application

| Field | Value |
|-------|-------|
| **Status** | backlog |
| **Priority** | 5 |
| **Tags** | saas |

### Description

New description that was edited offline
`
	req := postImportRequest(t, md, false)
	w := httptest.NewRecorder()
	h.Import(w, req)

	resp := parseImportResp(t, w)
	if !resp.DryRun {
		t.Error("expected dry_run to be true")
	}
	if len(resp.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(resp.Changes))
	}
	if resp.Changes[0].Item != "idea/my-app" {
		t.Errorf("expected item idea/my-app, got %s", resp.Changes[0].Item)
	}
	if resp.Changes[0].Action != "update" {
		t.Errorf("expected action update, got %s", resp.Changes[0].Action)
	}

	// Verify file on disk was NOT changed (dry run).
	item, err := h.store.LoadItem(KindIdea, "my-app")
	if err != nil {
		t.Fatal(err)
	}
	if item.Description != "Old description" {
		t.Errorf("expected description unchanged in dry run, got %q", item.Description)
	}
}

func TestImport_Apply_UpdateDescription(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:        "my-app",
		Title:       "My Application",
		Description: "Old description",
		Status:      StatusBacklog,
		Priority:    5,
		Tags:        []string{"saas"},
		Created:     "2026-02-10T00:00:00Z",
		Updated:     "2026-02-15T00:00:00Z",
	})

	md := `---
version: 1
items_count: 1
---

<!-- item:idea/my-app -->
## My Application

| Field | Value |
|-------|-------|
| **Status** | backlog |
| **Priority** | 5 |
| **Tags** | saas |

### Description

New description that was edited offline
`
	req := postImportRequest(t, md, true)
	w := httptest.NewRecorder()
	h.Import(w, req)

	resp := parseImportResp(t, w)
	if resp.DryRun {
		t.Error("expected dry_run to be false")
	}

	item, err := h.store.LoadItem(KindIdea, "my-app")
	if err != nil {
		t.Fatal(err)
	}
	if item.Description != "New description that was edited offline" {
		t.Errorf("expected updated description, got %q", item.Description)
	}
}

func TestImport_Apply_UpdateStatusAndPriority(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindFix, BacklogItem{
		Name:     "my-fix",
		Title:    "Fix Bug",
		Status:   StatusBacklog,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-02-10T00:00:00Z",
		Updated:  "2026-02-10T00:00:00Z",
	})

	md := `---
version: 1
items_count: 1
---

<!-- item:fix/my-fix -->
## Fix Bug

| Field | Value |
|-------|-------|
| **Status** | ready |
| **Priority** | 2 |
| **Tags** | urgent |
`
	req := postImportRequest(t, md, true)
	w := httptest.NewRecorder()
	h.Import(w, req)

	resp := parseImportResp(t, w)
	if len(resp.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(resp.Changes))
	}

	item, err := h.store.LoadItem(KindFix, "my-fix")
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != StatusReady {
		t.Errorf("expected status ready, got %s", item.Status)
	}
	if item.Priority != 2 {
		t.Errorf("expected priority 2, got %d", item.Priority)
	}
	if len(item.Tags) != 1 || item.Tags[0] != "urgent" {
		t.Errorf("expected tags [urgent], got %v", item.Tags)
	}
}

func TestImport_Apply_ClarifyAnswers(t *testing.T) {
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
			Options:    []string{"OAuth 2.0", "JWT tokens"},
			Answer:     "",
			Notes:      "",
		},
	})

	md := `---
version: 1
items_count: 1
---

<!-- item:idea/test-item -->
## Test Item

| Field | Value |
|-------|-------|
| **Status** | backlog |
| **Priority** | 5 |

<!-- clarify:idea/test-item -->
### Clarify Questions

**Q1: What auth method?** (technical, critical)

- [ ] OAuth 2.0
- [x] JWT tokens

> Notes: Good for microservices
<!-- /clarify -->
`
	req := postImportRequest(t, md, true)
	w := httptest.NewRecorder()
	h.Import(w, req)

	resp := parseImportResp(t, w)
	if len(resp.Errors) > 0 {
		t.Errorf("unexpected errors: %v", resp.Errors)
	}

	qPath := filepath.Join(tmpDir, "ideas", "test-item", "clarify", "questions.json")
	data, err := os.ReadFile(qPath)
	if err != nil {
		t.Fatal(err)
	}
	var questions []clarifyQuestion
	if err := json.Unmarshal(data, &questions); err != nil {
		t.Fatal(err)
	}
	if len(questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(questions))
	}
	if questions[0].Answer != "JWT tokens" {
		t.Errorf("expected answer 'JWT tokens', got %q", questions[0].Answer)
	}
	if questions[0].Notes != "Good for microservices" {
		t.Errorf("expected notes 'Good for microservices', got %q", questions[0].Notes)
	}
}

func TestImport_Apply_SuggestionAcceptReject(t *testing.T) {
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
			ID:       "s1",
			Title:    "Use WebSocket",
			Impact:   "high",
			Category: "architecture",
			Accepted: false,
		},
		{
			ID:       "s2",
			Title:    "Add caching",
			Impact:   "medium",
			Category: "ux",
			Accepted: false,
		},
	})

	md := `---
version: 1
items_count: 1
---

<!-- item:idea/test-item -->
## Test Item

| Field | Value |
|-------|-------|
| **Status** | backlog |
| **Priority** | 5 |

<!-- suggest:idea/test-item -->
### Suggestions

#### S1: Use WebSocket
- [x] Accept this suggestion

#### S2: Add caching
- [ ] Accept this suggestion
> Rejection reason: Too complex for MVP
<!-- /suggest -->
`
	req := postImportRequest(t, md, true)
	w := httptest.NewRecorder()
	h.Import(w, req)

	resp := parseImportResp(t, w)
	if len(resp.Errors) > 0 {
		t.Errorf("unexpected errors: %v", resp.Errors)
	}

	sPath := filepath.Join(tmpDir, "ideas", "test-item", "suggest", "suggestions.json")
	data, err := os.ReadFile(sPath)
	if err != nil {
		t.Fatal(err)
	}
	var suggestions []suggestion
	if err := json.Unmarshal(data, &suggestions); err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(suggestions))
	}
	if !suggestions[0].Accepted {
		t.Error("expected S1 to be accepted")
	}
	if suggestions[1].Accepted {
		t.Error("expected S2 to not be accepted")
	}
	if suggestions[1].RejectionReason != "Too complex for MVP" {
		t.Errorf("expected rejection reason, got %q", suggestions[1].RejectionReason)
	}
}

func TestImport_Apply_CreateNewItem(t *testing.T) {
	h, _ := setupTestHandler(t)

	md := `---
version: 1
items_count: 0
---

<!-- item:NEW -->
## idea/flight-idea -- Great Flight Idea

| Field | Value |
|-------|-------|
| **Status** | backlog |
| **Priority** | 3 |
| **Tags** | mobile, offline |

### Description

An idea I had on the plane about offline-first mobile apps.
`
	req := postImportRequest(t, md, true)
	w := httptest.NewRecorder()
	h.Import(w, req)

	resp := parseImportResp(t, w)
	if len(resp.Errors) > 0 {
		t.Errorf("unexpected errors: %v", resp.Errors)
	}
	if len(resp.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(resp.Changes))
	}
	if resp.Changes[0].Action != "create" {
		t.Errorf("expected action create, got %s", resp.Changes[0].Action)
	}

	item, err := h.store.LoadItem(KindIdea, "flight-idea")
	if err != nil {
		t.Fatalf("expected item to be created: %v", err)
	}
	if item.Title != "Great Flight Idea" {
		t.Errorf("expected title 'Great Flight Idea', got %q", item.Title)
	}
	if item.Priority != 3 {
		t.Errorf("expected priority 3, got %d", item.Priority)
	}
	if len(item.Tags) != 2 || item.Tags[0] != "mobile" {
		t.Errorf("expected tags [mobile, offline], got %v", item.Tags)
	}
}

func TestImport_NoChangesWhenUnmodified(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:        "my-app",
		Title:       "My Application",
		Description: "Original description",
		Status:      StatusBacklog,
		Priority:    5,
		Tags:        []string{"saas"},
		Created:     "2026-02-10T00:00:00Z",
		Updated:     "2026-02-15T00:00:00Z",
	})

	md := `---
version: 1
items_count: 1
---

<!-- item:idea/my-app -->
## My Application

| Field | Value |
|-------|-------|
| **Status** | backlog |
| **Priority** | 5 |
| **Tags** | saas |

### Description

Original description
`
	req := postImportRequest(t, md, false)
	w := httptest.NewRecorder()
	h.Import(w, req)

	resp := parseImportResp(t, w)
	if len(resp.Changes) != 0 {
		t.Errorf("expected 0 changes for unmodified item, got %d: %+v", len(resp.Changes), resp.Changes)
	}
}

// TestRoundTrip_ExportEditImport is the main round-trip test:
// 1. Create items with questions and suggestions
// 2. Export them to markdown
// 3. Edit the markdown (answer questions, accept suggestions, modify description, add new item)
// 4. Import with dry-run -> verify change list
// 5. Import with apply -> verify disk state
func TestRoundTrip_ExportEditImport(t *testing.T) {
	h, tmpDir := setupTestHandler(t)

	// Step 1: Create test data.
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

	createWorkshopRound(t, tmpDir, KindIdea, "my-app", WorkshopRound{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 1, "testable": 0, "risk_awareness": 0},
		Items: []WorkshopItem{
			{ID: "w1", Type: "decision", Topic: "What auth method?", Options: []WorkshopOption{{Key: "A", Label: "OAuth 2.0", Rationale: "Industry standard"}, {Key: "B", Label: "JWT tokens", Rationale: "Stateless auth"}, {Key: "C", Label: "Session-based", Rationale: "Traditional approach"}}},
			{ID: "w2", Type: "decision", Topic: "Target platform?", Options: []WorkshopOption{{Key: "A", Label: "Web", Rationale: "Broad reach"}, {Key: "B", Label: "Mobile", Rationale: "On the go"}, {Key: "C", Label: "Both", Rationale: "Maximum coverage"}}},
			{ID: "w3", Type: "decision", Topic: "Use WebSocket", Context: "Reduces latency by 10x", Options: []WorkshopOption{{Key: "A", Label: "Yes", Rationale: "Real-time benefits"}, {Key: "B", Label: "No", Rationale: "Added complexity"}}},
			{ID: "w4", Type: "decision", Topic: "Add caching", Context: "Improves mobile experience", Options: []WorkshopOption{{Key: "A", Label: "Yes", Rationale: "Performance boost"}, {Key: "B", Label: "No", Rationale: "Simplicity"}}},
		},
	})

	createTestItem(t, tmpDir, KindFix, BacklogItem{
		Name:        "bug-fix",
		Title:       "Fix Login Bug",
		Description: "Login fails on slow networks",
		Status:      StatusBacklog,
		Priority:    1,
		Tags:        []string{"bug"},
		Created:     "2026-02-12T00:00:00Z",
		Updated:     "2026-02-12T00:00:00Z",
	})

	// Step 2: Export to markdown.
	exportReq := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil)
	exportReq.ContentLength = 0
	exportW := httptest.NewRecorder()
	h.Export(exportW, exportReq)
	testutil.AssertStatusOK(t, exportW)

	exported := exportW.Body.String()
	if !strings.Contains(exported, "<!-- item:idea/my-app -->") {
		t.Fatal("export missing idea/my-app")
	}
	if !strings.Contains(exported, "<!-- item:fix/bug-fix -->") {
		t.Fatal("export missing fix/bug-fix")
	}

	// Step 3: Edit the markdown.
	edited := exported

	// Update my-app description.
	edited = strings.Replace(edited, "Build a great app", "Build a great app with real-time features and offline support", 1)
	// Update bug-fix priority from 1 to 2.
	edited = strings.Replace(edited, "| **Priority** | 1 |", "| **Priority** | 2 |", 1)

	// Add a new item before the template section.
	newItem := `<!-- item:NEW -->
## idea/offline-mode -- Offline Mode Support

| Field | Value |
|-------|-------|
| **Status** | backlog |
| **Priority** | 4 |
| **Tags** | mobile, offline |

### Description

Add offline-first capabilities using service workers.

---

`
	templateIdx := strings.Index(edited, "## New Item Template")
	if templateIdx == -1 {
		t.Fatal("couldn't find new item template in export")
	}
	edited = edited[:templateIdx] + newItem + edited[templateIdx:]

	// Step 4: Import with dry-run.
	dryReq := postImportRequest(t, edited, false)
	dryW := httptest.NewRecorder()
	h.Import(dryW, dryReq)

	dryResp := parseImportResp(t, dryW)
	if !dryResp.DryRun {
		t.Error("expected dry_run=true")
	}
	if len(dryResp.Errors) > 0 {
		t.Errorf("unexpected dry-run errors: %v", dryResp.Errors)
	}
	if len(dryResp.Changes) < 2 {
		t.Errorf("expected at least 2 changes, got %d: %+v", len(dryResp.Changes), dryResp.Changes)
	}

	// Verify dry-run didn't modify disk.
	origItem, _ := h.store.LoadItem(KindIdea, "my-app")
	if origItem.Description != "Build a great app" {
		t.Error("dry-run modified item on disk!")
	}

	// Step 5: Import with apply.
	applyReq := postImportRequest(t, edited, true)
	applyW := httptest.NewRecorder()
	h.Import(applyW, applyReq)

	applyResp := parseImportResp(t, applyW)
	if applyResp.DryRun {
		t.Error("expected dry_run=false")
	}
	if len(applyResp.Errors) > 0 {
		t.Errorf("unexpected apply errors: %v", applyResp.Errors)
	}

	// Verify my-app description was updated.
	myApp, err := h.store.LoadItem(KindIdea, "my-app")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(myApp.Description, "real-time features and offline support") {
		t.Errorf("expected updated description, got %q", myApp.Description)
	}

	// Verify bug-fix priority was updated.
	bugFix, err := h.store.LoadItem(KindFix, "bug-fix")
	if err != nil {
		t.Fatal(err)
	}
	if bugFix.Priority != 2 {
		t.Errorf("expected priority 2 for bug-fix, got %d", bugFix.Priority)
	}

	// Verify new item was created.
	newItemLoaded, err := h.store.LoadItem(KindIdea, "offline-mode")
	if err != nil {
		t.Fatalf("expected new item to be created: %v", err)
	}
	if newItemLoaded.Title != "Offline Mode Support" {
		t.Errorf("expected title 'Offline Mode Support', got %q", newItemLoaded.Title)
	}
	if newItemLoaded.Priority != 4 {
		t.Errorf("expected priority 4, got %d", newItemLoaded.Priority)
	}
	if !strings.Contains(newItemLoaded.Description, "service workers") {
		t.Errorf("expected description with 'service workers', got %q", newItemLoaded.Description)
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"valid", "---\nversion: 1\nitems_count: 0\n---\n", true},
		{"wrong version", "---\nversion: 2\n---\n", false},
		{"no frontmatter", "just text\n", false},
		{"single delimiter", "---\nversion: 1\n", false},
		{"no version field", "---\nitems_count: 5\n---\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(tt.input, "\n")
			_, _, valid := parseFrontmatter(lines)
			if valid != tt.valid {
				t.Errorf("parseFrontmatter valid=%v, want %v", valid, tt.valid)
			}
		})
	}
}

func TestSplitItemSections(t *testing.T) {
	lines := strings.Split(`Some preamble text
<!-- item:idea/my-app -->
## My App
Description here
<!-- item:fix/bug-123 -->
## Bug Fix
Another description
<!-- item:NEW -->
## idea/new-idea -- New
`, "\n")

	sections := splitItemSections(lines)
	if len(sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(sections))
	}
	if sections[0].marker != "idea/my-app" {
		t.Errorf("expected marker idea/my-app, got %s", sections[0].marker)
	}
	if sections[1].marker != "fix/bug-123" {
		t.Errorf("expected marker fix/bug-123, got %s", sections[1].marker)
	}
	if sections[2].marker != "NEW" {
		t.Errorf("expected marker NEW, got %s", sections[2].marker)
	}
}

func TestExtractQuestionNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"#### Q1: What auth?", 0},
		{"#### Q2: Target?", 1},
		{"**Q3: Deadline?**", 2},
		{"#### Something", -1},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractQuestionNumber(tt.input)
			if got != tt.expected {
				t.Errorf("extractQuestionNumber(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractSuggestionNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"#### S1: Use WebSocket", 0},
		{"#### S2: Add caching", 1},
		{"#### 1. First", 0},
		{"#### 3. Third", 2},
		{"#### Something", -1},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractSuggestionNumber(tt.input)
			if got != tt.expected {
				t.Errorf("extractSuggestionNumber(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTagsEqual(t *testing.T) {
	tests := []struct {
		a, b     []string
		expected bool
	}{
		{[]string{"a", "b"}, []string{"a", "b"}, true},
		{[]string{"a", "b"}, []string{"b", "a"}, false},
		{[]string{"a"}, []string{"a", "b"}, false},
		{nil, nil, true},
		{[]string{}, []string{}, true},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			got := tagsEqual(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("tagsEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}
