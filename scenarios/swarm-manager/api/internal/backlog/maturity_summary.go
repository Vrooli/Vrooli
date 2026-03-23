package backlog

import (
	"net/http"

	"swarm-manager/internal/httputil"
)

// MaturityItemSummary describes the workshop readiness of a single backlog item.
type MaturityItemSummary struct {
	Kind            BacklogKind    `json:"kind"`
	Name            string         `json:"name"`
	Title           string         `json:"title"`
	RoundsCompleted int            `json:"rounds_completed"`
	RawScores       map[string]int `json:"raw_scores"`
	EffectiveScores map[string]int `json:"effective_scores"`
	Ready           bool           `json:"ready"`
	PendingItems    int            `json:"pending_items"`
	HasPlan         bool           `json:"has_plan"`
}

// MaturitySummaryResponse is the response for the maturity-summary endpoint.
type MaturitySummaryResponse struct {
	Items []MaturityItemSummary `json:"items"`
}

// MaturitySummary returns workshop readiness data for all backlog items.
func (h *Handler) MaturitySummary(w http.ResponseWriter, r *http.Request) {
	items, err := h.loadAllItems(nil) // all kinds
	if err != nil {
		httputil.InternalError(w, "[backlog] maturity-summary", err.Error())
		return
	}

	summaryItems := make([]MaturityItemSummary, 0, len(items))

	for _, item := range items {
		itemDir := h.itemDir(item.Kind, item.Name)

		latestRound, roundCount, err := LoadLatestRound(itemDir)
		if err != nil {
			continue
		}

		rawScores := make(map[string]int, len(ReadinessDimensions))
		for _, dim := range ReadinessDimensions {
			rawScores[dim] = 0
		}
		if latestRound != nil {
			for _, dim := range ReadinessDimensions {
				if v, ok := latestRound.Readiness[dim]; ok {
					rawScores[dim] = v
				}
			}
		}

		effectiveScores := ComputeEffectiveScores(rawScores, roundCount, item.Kind)

		summaryItems = append(summaryItems, MaturityItemSummary{
			Kind:            item.Kind,
			Name:            item.Name,
			Title:           item.Title,
			RoundsCompleted: roundCount,
			RawScores:       rawScores,
			EffectiveScores: effectiveScores,
			Ready:           IsReady(effectiveScores),
			PendingItems:    CountPendingDecisions(latestRound),
			HasPlan:         HasPlan(itemDir),
		})
	}

	resp := MaturitySummaryResponse{Items: summaryItems}
	if err := httputil.JSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] maturity-summary", "failed to encode response")
	}
}
