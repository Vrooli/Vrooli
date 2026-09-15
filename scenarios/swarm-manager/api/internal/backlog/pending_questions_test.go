package backlog

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"
)

func doPendingQuestions(t *testing.T, h *Handler, query string) *httptest.ResponseRecorder {
	t.Helper()
	path := "/api/v1/backlog/pending-questions"
	if query != "" {
		path += "?" + query
	}
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	h.PendingQuestions(w, req)
	return w
}

func TestPendingQuestions_ReturnsUnreviewedReviewItems(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{Name: "pending-review", Title: "Pending Review", Status: StatusBacklog, Priority: 2})
	itemDir := filepath.Join(rootDir, "ideas", "pending-review")
	testutil.WriteFile(t, filepath.Join(itemDir, "archive", "PRD.md"), "# PRD\n\n## \U0001f3af Operational Targets\n\n### \U0001f534 P0\n- [ ] OT-P0-001 | Core target | Must support X\n")

	w := doPendingQuestions(t, h, "")
	testutil.AssertStatusOK(t, w)
	var response PendingQuestionsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 || len(response.Items[0].Questions) != 1 {
		t.Fatalf("expected one pending review question, got %+v", response)
	}
	question := response.Items[0].Questions[0]
	if question.Source != "review" || question.ReviewType != "target" || question.ID != "OT-P0-001" {
		t.Fatalf("unexpected review question: %+v", question)
	}
}

func TestPendingQuestions_RejectsRetiredWorkshopSource(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := doPendingQuestions(t, h, "source=workshop")
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
