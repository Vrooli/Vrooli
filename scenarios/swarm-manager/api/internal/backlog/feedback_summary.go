// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package backlog

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"swarm-manager/internal/httputil"
	"swarm-manager/internal/jsonutil"
)

// FeedbackItemSummary describes feedback state for a single backlog item.
type FeedbackItemSummary struct {
	Kind                BacklogKind     `json:"kind"`
	Name                string          `json:"name"`
	Title               string          `json:"title"`
	UnansweredQuestions int             `json:"unanswered_questions"`
	PendingSuggestions  int             `json:"pending_suggestions"`
	QuestionsContent    json.RawMessage `json:"questions_content"`
	SuggestionsContent  json.RawMessage `json:"suggestions_content"`
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

		qContent, unanswered := readQuestionsFile(filepath.Join(itemDir, "clarify", "questions.json"))
		sContent, pending := readSuggestionsFile(filepath.Join(itemDir, "suggest", "suggestions.json"))

		if unanswered == 0 && pending == 0 {
			continue
		}

		summaryItems = append(summaryItems, FeedbackItemSummary{
			Kind:                item.Kind,
			Name:                item.Name,
			Title:               item.Title,
			UnansweredQuestions: unanswered,
			PendingSuggestions:  pending,
			QuestionsContent:    qContent,
			SuggestionsContent:  sContent,
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

// readQuestionsFile returns the raw JSON content and the count of unanswered questions.
func readQuestionsFile(path string) (json.RawMessage, int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0
	}
	var f questionsFile
	if err := json.Unmarshal(data, &f); err != nil {
		// Attempt to salvage complete questions from truncated JSON
		if repaired := jsonutil.RepairTruncatedJSON(data); repaired != nil {
			if json.Unmarshal(repaired, &f) == nil {
				return readQuestionsFrom(repaired, f)
			}
		}
		return nil, 0
	}
	return readQuestionsFrom(data, f)
}

func readQuestionsFrom(data []byte, f questionsFile) (json.RawMessage, int) {
	count := 0
	for _, q := range f.Questions {
		if q.Answer == "" {
			count++
		}
	}
	return json.RawMessage(data), count
}

// readSuggestionsFile returns the raw JSON content and the count of pending suggestions.
func readSuggestionsFile(path string) (json.RawMessage, int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0
	}
	var f suggestionsFile
	if err := json.Unmarshal(data, &f); err != nil {
		if repaired := jsonutil.RepairTruncatedJSON(data); repaired != nil {
			if json.Unmarshal(repaired, &f) == nil {
				return readSuggestionsFrom(repaired, f)
			}
		}
		return nil, 0
	}
	return readSuggestionsFrom(data, f)
}

func readSuggestionsFrom(data []byte, f suggestionsFile) (json.RawMessage, int) {
	count := 0
	for _, s := range f.Suggestions {
		if s.Status == "" || s.Status == "pending" {
			count++
		}
	}
	return json.RawMessage(data), count
}

// countUnansweredQuestions returns just the count (used by queue handler).
func countUnansweredQuestions(path string) int {
	_, count := readQuestionsFile(path)
	return count
}

// countPendingSuggestions returns just the count (used by queue handler).
func countPendingSuggestions(path string) int {
	_, count := readSuggestionsFile(path)
	return count
}

// ---------------------------------------------------------------------------
// Round-trip structs for reading, mutating, and writing back JSON files.
// These capture all known fields so snapshots can be stamped in place.
// ---------------------------------------------------------------------------

type questionLastSynthesis struct {
	Answer string `json:"answer"`
	Round  int    `json:"round"`
}

type questionRT struct {
	ID            string                 `json:"id"`
	Question      string                 `json:"question"`
	Options       []string               `json:"options,omitempty"`
	Answer        string                 `json:"answer,omitempty"`
	LastSynthesis *questionLastSynthesis `json:"lastSynthesis,omitempty"`
}

type questionsFileRT struct {
	Questions    []questionRT `json:"questions"`
	GeneratedAt  string       `json:"generatedAt,omitempty"`
	UpdatedAt    string       `json:"updatedAt,omitempty"`
	ClarifyCount int          `json:"clarifyCount,omitempty"`
	EnhanceCount int          `json:"enhanceCount,omitempty"`
}

type suggestionLastSynthesis struct {
	Status string `json:"status"`
	Round  int    `json:"round"`
}

type suggestionRT struct {
	ID            string                   `json:"id"`
	Suggestion    string                   `json:"suggestion"`
	Details       string                   `json:"details,omitempty"`
	Status        string                   `json:"status,omitempty"`
	LastSynthesis *suggestionLastSynthesis `json:"lastSynthesis,omitempty"`
}

type suggestionsFileRT struct {
	Suggestions  []suggestionRT `json:"suggestions"`
	GeneratedAt  string         `json:"generatedAt,omitempty"`
	UpdatedAt    string         `json:"updatedAt,omitempty"`
	SuggestCount int            `json:"suggestCount,omitempty"`
	EnhanceCount int            `json:"enhanceCount,omitempty"`
}

// snapshotQuestionsForEnhance reads questions.json, increments enhanceCount,
// stamps lastSynthesis on each answered question, and writes the file back.
// No-ops if the file does not exist.
func snapshotQuestionsForEnhance(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var f questionsFileRT
	if err := json.Unmarshal(data, &f); err != nil {
		if repaired := jsonutil.RepairTruncatedJSON(data); repaired != nil {
			if err2 := json.Unmarshal(repaired, &f); err2 != nil {
				return err2
			}
		} else {
			return err
		}
	}
	f.EnhanceCount++
	for i := range f.Questions {
		if f.Questions[i].Answer != "" {
			f.Questions[i].LastSynthesis = &questionLastSynthesis{
				Answer: f.Questions[i].Answer,
				Round:  f.EnhanceCount,
			}
		}
	}
	f.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// snapshotSuggestionsForEnhance reads suggestions.json, increments enhanceCount,
// stamps lastSynthesis on each decided suggestion, and writes the file back.
// No-ops if the file does not exist.
func snapshotSuggestionsForEnhance(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var f suggestionsFileRT
	if err := json.Unmarshal(data, &f); err != nil {
		if repaired := jsonutil.RepairTruncatedJSON(data); repaired != nil {
			if err2 := json.Unmarshal(repaired, &f); err2 != nil {
				return err2
			}
		} else {
			return err
		}
	}
	f.EnhanceCount++
	for i := range f.Suggestions {
		s := f.Suggestions[i].Status
		if s != "" && s != "pending" {
			f.Suggestions[i].LastSynthesis = &suggestionLastSynthesis{
				Status: s,
				Round:  f.EnhanceCount,
			}
		}
	}
	f.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// incrementClarifyCount increments clarifyCount in questions.json.
// No-ops if the file does not exist.
func incrementClarifyCount(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var f questionsFileRT
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	f.ClarifyCount++
	f.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// incrementSuggestCount increments suggestCount in suggestions.json.
// No-ops if the file does not exist.
func incrementSuggestCount(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var f suggestionsFileRT
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	f.SuggestCount++
	f.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
