// Combined summary endpoint — returns current independent-review questions.
package backlog

import (
	"net/http"
	"sort"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/backlogrank"
	"swarm-manager/internal/httputil"
)

// BacklogSummaryResponse contains current independent-review questions.
type BacklogSummaryResponse struct {
	PendingQuestions PendingQuestionsResponse `json:"pending_questions"`
}

// BacklogSummary returns independent-review questions in one round-trip.
// Plan Workshop questions are intentionally resolved inside their session,
// rather than re-exposed through the legacy cross-item decision stream.
func (h *Handler) BacklogSummary(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.LoadAll(nil)
	if err != nil {
		apierr.MapError(w, "[backlog] summary", apierr.Internal("%s", err.Error()))
		return
	}

	pendingItems := buildPendingQuestions(items, h.store)

	resp := BacklogSummaryResponse{
		PendingQuestions: PendingQuestionsResponse{Items: pendingItems},
	}
	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] summary", apierr.Internal("failed to encode response"))
	}
}

func buildPendingQuestions(backlogItems []BacklogItem, store Store) []PendingQuestionsItem {
	var items []PendingQuestionsItem
	depthMap, unblockingMap, itemsByKey := rankPendingQuestionItems(backlogItems)
	for _, item := range backlogItems {
		questions := collectReviewQuestions(store.ItemDir(item.Kind, item.Name), item.Kind, item.Name)
		if len(questions) > 0 {
			items = append(items, PendingQuestionsItem{
				Kind: item.Kind, Name: item.Name, Questions: questions,
			})
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := itemsByKey[backlogrank.Key(string(items[i].Kind), items[i].Name)]
		right := itemsByKey[backlogrank.Key(string(items[j].Kind), items[j].Name)]
		return backlogrank.Less(left, right, depthMap, unblockingMap)
	})
	if items == nil {
		items = []PendingQuestionsItem{}
	}
	return items
}
