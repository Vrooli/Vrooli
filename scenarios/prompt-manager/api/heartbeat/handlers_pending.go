package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"prompt-manager/store"
	"strings"
	"time"
)

// GetAllPendingDecisions handles GET /v1/decisions/pending
// Returns all pending decisions across all teams. Deferred decisions whose
// revisit_after has elapsed are promoted back to pending in-place (durable
// status flip) and included in the response.
func (h *Handlers) GetAllPendingDecisions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	teams, err := h.teamStore.List(ctx)
	if err != nil {
		http.Error(w, "failed to list teams", http.StatusInternalServerError)
		return
	}

	var groups []PendingDecisionTeamGroup
	totalCount := 0

	today := time.Now().UTC().Truncate(24 * time.Hour)

	for _, team := range teams {
		// Promote any due-deferred rows for this team to pending before reading.
		h.resurfaceDueDeferred(ctx, team.ID, today)

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

// resurfaceDueDeferred flips status=deferred → pending for any rows whose
// revisit_after is on or before today, and appends an audit note recording
// the original deferral window. Failures are best-effort; promotion will be
// retried on the next pending-queue read.
func (h *Handlers) resurfaceDueDeferred(ctx context.Context, teamID string, today time.Time) {
	deferred, _, err := h.teamStore.GetDecisions(ctx, teamID, "", store.DecisionStatusDeferred, 0)
	if err != nil {
		return
	}
	for _, d := range deferred {
		if d.RevisitAfter == nil {
			continue
		}
		parsed, err := time.Parse("2006-01-02", *d.RevisitAfter)
		if err != nil {
			continue
		}
		if parsed.UTC().Truncate(24 * time.Hour).After(today) {
			continue
		}
		prevDate := *d.RevisitAfter
		todayStr := today.Format("2006-01-02")
		note := fmt.Sprintf("[re-surfaced after defer] deferred %s → revisit %s", prevDate, todayStr)
		_ = h.teamStore.UpdateDecision(ctx, teamID, d.ID, func(e *store.DecisionEntry) {
			e.Status = store.DecisionStatusPending
			e.RevisitAfter = nil
			if strings.TrimSpace(e.Notes) == "" {
				e.Notes = note
			} else {
				e.Notes = e.Notes + "\n" + note
			}
		})
	}
}
