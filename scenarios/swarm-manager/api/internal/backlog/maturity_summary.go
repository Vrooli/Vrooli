package backlog

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"swarm-manager/internal/httputil"
	"swarm-manager/internal/jsonutil"
)

// MaturityItemSummary describes the refinement maturity of a single backlog idea.
type MaturityItemSummary struct {
	Kind                    BacklogKind `json:"kind"`
	Name                    string      `json:"name"`
	Title                   string      `json:"title"`
	ClarifyCount            int         `json:"clarify_count"`
	SuggestCount            int         `json:"suggest_count"`
	EnhanceCount            int         `json:"enhance_count"`
	QuestionsTotal          int         `json:"questions_total"`
	QuestionsAnswered       int         `json:"questions_answered"`
	SuggestionsTotal        int         `json:"suggestions_total"`
	SuggestionsDecided      int         `json:"suggestions_decided"`
	QuestionsNewOrUpdated   int         `json:"questions_new_or_updated"`
	SuggestionsNewOrUpdated int         `json:"suggestions_new_or_updated"`
	HasEnhanceSummary       bool        `json:"has_enhance_summary"`
}

// MaturitySummaryResponse is the response for the maturity-summary endpoint.
type MaturitySummaryResponse struct {
	Items []MaturityItemSummary `json:"items"`
}

// maturityQuestionsFile captures the fields needed for maturity computation.
type maturityQuestionsFile struct {
	Questions []struct {
		Answer        string `json:"answer"`
		LastSynthesis *struct {
			Answer string `json:"answer"`
			Round  int    `json:"round"`
		} `json:"lastSynthesis,omitempty"`
	} `json:"questions"`
	ClarifyCount int `json:"clarifyCount,omitempty"`
	EnhanceCount int `json:"enhanceCount,omitempty"`
}

// maturitySuggestionsFile captures the fields needed for maturity computation.
type maturitySuggestionsFile struct {
	Suggestions []struct {
		Status        string `json:"status"`
		LastSynthesis *struct {
			Status string `json:"status"`
			Round  int    `json:"round"`
		} `json:"lastSynthesis,omitempty"`
	} `json:"suggestions"`
	SuggestCount int `json:"suggestCount,omitempty"`
	EnhanceCount int `json:"enhanceCount,omitempty"`
}

// MaturitySummary returns maturity data for all idea backlog items.
func (h *Handler) MaturitySummary(w http.ResponseWriter, r *http.Request) {
	ideaKinds := []BacklogKind{KindIdea}
	items, err := h.loadAllItems(ideaKinds)
	if err != nil {
		httputil.InternalError(w, "[backlog] maturity-summary", err.Error())
		return
	}

	summaryItems := make([]MaturityItemSummary, 0, len(items))

	for _, item := range items {
		itemDir := h.itemDir(item.Kind, item.Name)
		ms := MaturityItemSummary{
			Kind:  item.Kind,
			Name:  item.Name,
			Title: item.Title,
		}

		// Read questions
		qPath := filepath.Join(itemDir, "clarify", "questions.json")
		if qf, ok := readMaturityQuestions(qPath); ok {
			ms.ClarifyCount = qf.ClarifyCount
			ms.EnhanceCount = qf.EnhanceCount
			ms.QuestionsTotal = len(qf.Questions)
			for _, q := range qf.Questions {
				if q.Answer != "" {
					ms.QuestionsAnswered++
				}
				if q.LastSynthesis == nil {
					ms.QuestionsNewOrUpdated++
				} else if q.Answer != "" && q.LastSynthesis.Answer != q.Answer {
					ms.QuestionsNewOrUpdated++
				}
			}
		}

		// Read suggestions
		sPath := filepath.Join(itemDir, "suggest", "suggestions.json")
		if sf, ok := readMaturitySuggestions(sPath); ok {
			ms.SuggestCount = sf.SuggestCount
			// Use the max enhanceCount from either file
			if sf.EnhanceCount > ms.EnhanceCount {
				ms.EnhanceCount = sf.EnhanceCount
			}
			ms.SuggestionsTotal = len(sf.Suggestions)
			for _, s := range sf.Suggestions {
				status := s.Status
				if status != "" && status != "pending" {
					ms.SuggestionsDecided++
				}
				if s.LastSynthesis == nil {
					ms.SuggestionsNewOrUpdated++
				} else if status != "" && s.LastSynthesis.Status != status {
					ms.SuggestionsNewOrUpdated++
				}
			}
		}

		// Check enhance summary existence
		enhancePath := filepath.Join(itemDir, "enhance", "summary.md")
		if _, err := os.Stat(enhancePath); err == nil {
			ms.HasEnhanceSummary = true
		}

		summaryItems = append(summaryItems, ms)
	}

	resp := MaturitySummaryResponse{Items: summaryItems}
	if err := httputil.JSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] maturity-summary", "failed to encode response")
	}
}

func readMaturityQuestions(path string) (maturityQuestionsFile, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return maturityQuestionsFile{}, false
	}
	var f maturityQuestionsFile
	if err := json.Unmarshal(data, &f); err != nil {
		if repaired := jsonutil.RepairTruncatedJSON(data); repaired != nil {
			if json.Unmarshal(repaired, &f) == nil {
				return f, true
			}
		}
		return maturityQuestionsFile{}, false
	}
	return f, true
}

func readMaturitySuggestions(path string) (maturitySuggestionsFile, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return maturitySuggestionsFile{}, false
		}
		return maturitySuggestionsFile{}, false
	}
	var f maturitySuggestionsFile
	if err := json.Unmarshal(data, &f); err != nil {
		if repaired := jsonutil.RepairTruncatedJSON(data); repaired != nil {
			if json.Unmarshal(repaired, &f) == nil {
				return f, true
			}
		}
		return maturitySuggestionsFile{}, false
	}
	return f, true
}
