// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package backlog

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"swarm-manager/internal/httputil"
)

// FeedbackItemSummary describes feedback state for a single backlog item.
type FeedbackItemSummary struct {
	Kind                BacklogKind `json:"kind"`
	Name                string      `json:"name"`
	Title               string      `json:"title"`
	UnansweredQuestions int         `json:"unanswered_questions"`
	PendingSuggestions  int         `json:"pending_suggestions"`
}

// FeedbackSummaryResponse is the response for the feedback-summary endpoint.
type FeedbackSummaryResponse struct {
	Items              []FeedbackItemSummary `json:"items"`
	TotalUnanswered    int                   `json:"total_unanswered"`
	TotalPending       int                   `json:"total_pending"`
	TotalItemsAffected int                   `json:"total_items_affected"`
}

// questionsFile is the JSON structure for clarify/questions.json.
type questionsFile struct {
	Questions []struct {
		Answer string `json:"answer"`
	} `json:"questions"`
}

// suggestionsFile is the JSON structure for suggest/suggestions.json.
type suggestionsFile struct {
	Suggestions []struct {
		Status string `json:"status"`
	} `json:"suggestions"`
}

// FeedbackSummary returns a summary of unanswered questions and pending suggestions
// across all backlog items.
func (h *Handler) FeedbackSummary(w http.ResponseWriter, r *http.Request) {
	items, err := h.loadAllItems(nil)
	if err != nil {
		httputil.InternalError(w, "[backlog] feedback-summary", err.Error())
		return
	}

	var summaryItems []FeedbackItemSummary
	totalUnanswered := 0
	totalPending := 0

	for _, item := range items {
		itemDir := h.itemDir(item.Kind, item.Name)

		unanswered := countUnansweredQuestions(filepath.Join(itemDir, "clarify", "questions.json"))
		pending := countPendingSuggestions(filepath.Join(itemDir, "suggest", "suggestions.json"))

		if unanswered == 0 && pending == 0 {
			continue
		}

		summaryItems = append(summaryItems, FeedbackItemSummary{
			Kind:                item.Kind,
			Name:                item.Name,
			Title:               item.Title,
			UnansweredQuestions: unanswered,
			PendingSuggestions:  pending,
		})
		totalUnanswered += unanswered
		totalPending += pending
	}

	if summaryItems == nil {
		summaryItems = []FeedbackItemSummary{}
	}

	resp := FeedbackSummaryResponse{
		Items:              summaryItems,
		TotalUnanswered:    totalUnanswered,
		TotalPending:       totalPending,
		TotalItemsAffected: len(summaryItems),
	}
	if err := httputil.JSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] feedback-summary", "failed to encode response")
	}
}

func countUnansweredQuestions(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var f questionsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return 0
	}
	count := 0
	for _, q := range f.Questions {
		if q.Answer == "" {
			count++
		}
	}
	return count
}

func countPendingSuggestions(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var f suggestionsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return 0
	}
	count := 0
	for _, s := range f.Suggestions {
		if s.Status == "" || s.Status == "pending" {
			count++
		}
	}
	return count
}
