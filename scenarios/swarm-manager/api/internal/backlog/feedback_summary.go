// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package backlog

import (
	"net/http"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
)

// FeedbackItemSummary describes feedback state for a single backlog item.
type FeedbackItemSummary struct {
	Kind             BacklogKind `json:"kind"`
	Name             string      `json:"name"`
	Title            string      `json:"title"`
	PendingDecisions int         `json:"pending_decisions"`
}

// FeedbackSummaryResponse is the response for the feedback-summary endpoint.
type FeedbackSummaryResponse struct {
	Items              []FeedbackItemSummary `json:"items"`
	TotalPending       int                   `json:"total_pending"`
	TotalItemsAffected int                   `json:"total_items_affected"`
}

// FeedbackSummary returns a summary of pending decisions across all backlog
// items by reading the latest workshop round for each item.
func (h *Handler) FeedbackSummary(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.LoadAll(nil)
	if err != nil {
		apierr.MapError(w, "[backlog] feedback-summary", apierr.Internal("%s", err.Error()))
		return
	}

	var summaryItems []FeedbackItemSummary
	totalPending := 0

	for _, item := range items {
		itemDir := h.store.ItemDir(item.Kind, item.Name)

		latestRound, _, err := LoadLatestRound(itemDir)
		if err != nil || latestRound == nil {
			continue
		}

		pending := CountPendingDecisions(latestRound)
		if pending == 0 {
			continue
		}

		summaryItems = append(summaryItems, FeedbackItemSummary{
			Kind:             item.Kind,
			Name:             item.Name,
			Title:            item.Title,
			PendingDecisions: pending,
		})
		totalPending += pending
	}

	if summaryItems == nil {
		summaryItems = []FeedbackItemSummary{}
	}

	resp := FeedbackSummaryResponse{
		Items:              summaryItems,
		TotalPending:       totalPending,
		TotalItemsAffected: len(summaryItems),
	}
	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] feedback-summary", apierr.Internal("failed to encode response"))
	}
}
