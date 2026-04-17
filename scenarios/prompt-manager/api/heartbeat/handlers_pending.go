package heartbeat

import (
	"encoding/json"
	"net/http"
	"prompt-manager/store"
)

// GetAllPendingDecisions handles GET /v1/decisions/pending
// Returns all pending decisions across all teams.
func (h *Handlers) GetAllPendingDecisions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	teams, err := h.teamStore.List(ctx)
	if err != nil {
		http.Error(w, "failed to list teams", http.StatusInternalServerError)
		return
	}

	var groups []PendingDecisionTeamGroup
	totalCount := 0

	for _, team := range teams {
		entries, _, err := h.teamStore.GetDecisions(ctx, team.ID, "", store.DecisionStatusPending, 0)
		if err != nil {
			continue // skip teams with read errors
		}
		if len(entries) == 0 {
			continue
		}

		groups = append(groups, PendingDecisionTeamGroup{
			TeamID:   team.ID,
			TeamName: team.DisplayName,
			Entries:  entries,
		})
		totalCount += len(entries)
	}

	if groups == nil {
		groups = []PendingDecisionTeamGroup{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(AllPendingDecisionsResponse{
		Teams:      groups,
		TotalCount: totalCount,
	})
}
