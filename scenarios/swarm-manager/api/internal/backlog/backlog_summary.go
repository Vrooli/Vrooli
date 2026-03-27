// Combined summary endpoint — returns feedback, maturity, and pending-questions
// data in a single response. This avoids three separate LoadAll + workshop-round
// traversals, cutting the BacklogPage bootstrap from 3 sequential RPCs to 1.
package backlog

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"swarm-manager/internal/httputil"
)

// BacklogSummaryResponse combines feedback, maturity, and pending-questions data.
type BacklogSummaryResponse struct {
	Feedback         FeedbackSummaryResponse  `json:"feedback"`
	Maturity         MaturitySummaryResponse  `json:"maturity"`
	PendingQuestions PendingQuestionsResponse `json:"pending_questions"`
}

// BacklogSummary returns a combined summary covering feedback, maturity, and
// pending questions in one round-trip. Internally it loads all items once and
// performs a single pass over workshop rounds.
func (h *Handler) BacklogSummary(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.LoadAll(nil)
	if err != nil {
		httputil.InternalError(w, "[backlog] summary", err.Error())
		return
	}

	// Pre-allocate result slices.
	feedbackItems := []FeedbackItemSummary{}
	maturityItems := make([]MaturityItemSummary, 0, len(items))
	var pendingItems []PendingQuestionsItem
	totalPending := 0

	for _, item := range items {
		itemDir := h.store.ItemDir(item.Kind, item.Name)

		// Load workshop round once per item (shared across feedback + maturity).
		latestRound, roundCount, roundErr := LoadLatestRound(itemDir)

		// --- Feedback ---
		if roundErr == nil && latestRound != nil {
			pending := CountPendingDecisions(latestRound)
			if pending > 0 {
				feedbackItems = append(feedbackItems, FeedbackItemSummary{
					Kind:             item.Kind,
					Name:             item.Name,
					Title:            item.Title,
					PendingDecisions: pending,
				})
				totalPending += pending
			}
		}

		// --- Maturity ---
		rawScores := make(map[string]int, len(ReadinessDimensions))
		for _, dim := range ReadinessDimensions {
			rawScores[dim] = 0
		}
		if roundErr == nil && latestRound != nil {
			for _, dim := range ReadinessDimensions {
				if v, ok := latestRound.Readiness[dim]; ok {
					rawScores[dim] = v
				}
			}
		}
		effectiveScores := ComputeEffectiveScores(rawScores, roundCount, item.Kind)
		maturityItems = append(maturityItems, MaturityItemSummary{
			Kind:             item.Kind,
			Name:             item.Name,
			Title:            item.Title,
			RoundsCompleted:  roundCount,
			RawScores:        rawScores,
			EffectiveScores:  effectiveScores,
			Ready:            IsReady(effectiveScores),
			PendingItems:     CountPendingDecisions(latestRound),
			PendingSynthesis: NeedsSynthesis(latestRound),
			HasPlan:          HasPlanByName(itemDir, DeliverableForKind(item.Kind)),
		})

		// --- Pending questions ---
		var questions []PendingQuestion
		questions = append(questions, collectWorkshopQuestionsFromRound(latestRound, item.Kind, item.Name)...)
		questions = append(questions, collectReviewQuestionsFromDir(itemDir, item.Kind, item.Name)...)
		if len(questions) > 0 {
			pendingItems = append(pendingItems, PendingQuestionsItem{
				Kind:      item.Kind,
				Name:      item.Name,
				Questions: questions,
			})
		}
	}

	if pendingItems == nil {
		pendingItems = []PendingQuestionsItem{}
	}

	resp := BacklogSummaryResponse{
		Feedback: FeedbackSummaryResponse{
			Items:              feedbackItems,
			TotalPending:       totalPending,
			TotalItemsAffected: len(feedbackItems),
		},
		Maturity:         MaturitySummaryResponse{Items: maturityItems},
		PendingQuestions: PendingQuestionsResponse{Items: pendingItems},
	}
	if err := httputil.JSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] summary", "failed to encode response")
	}
}

// collectWorkshopQuestionsFromRound extracts unanswered decisions from an
// already-loaded workshop round (avoids re-reading the file).
func collectWorkshopQuestionsFromRound(latestRound *WorkshopRound, kind BacklogKind, name string) []PendingQuestion {
	if latestRound == nil {
		return nil
	}
	var questions []PendingQuestion
	for _, wi := range latestRound.Items {
		if wi.Type != "decision" {
			continue
		}
		if wi.Selected != nil && strings.TrimSpace(*wi.Selected) != "" {
			continue
		}
		questions = append(questions, PendingQuestion{
			ID:          wi.ID,
			Source:      "workshop",
			ItemKind:    string(kind),
			ItemName:    name,
			Topic:       wi.Topic,
			Text:        wi.Text,
			Context:     wi.Context,
			Options:     wi.Options,
			Selected:    wi.Selected,
			Freeform:    wi.Freeform,
			Notes:       wi.Notes,
			RoundNumber: latestRound.RoundNum,
		})
	}
	return questions
}

// collectReviewQuestionsFromDir extracts unreviewed targets/requirements
// from the archive directory.
func collectReviewQuestionsFromDir(itemDir string, kind BacklogKind, name string) []PendingQuestion {
	archiveDir := filepath.Join(itemDir, "archive")
	info, err := os.Stat(archiveDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	var questions []PendingQuestion

	targets, err := ParseArchiveTargets(archiveDir)
	if err != nil {
		targets = []ArchiveTarget{}
	}

	reviewState, _ := ReadReviewState(itemDir)

	for _, t := range targets {
		rs, hasReview := reviewState[t.ID]
		status := ""
		comment := ""
		if hasReview {
			status = rs.ReviewStatus
			comment = rs.ReviewComment
		}
		if status == "approved" || status == "flagged" {
			continue
		}
		questions = append(questions, PendingQuestion{
			ID:            t.ID,
			Source:        "review",
			ItemKind:      string(kind),
			ItemName:      name,
			Title:         t.Title,
			Description:   t.Notes,
			Criticality:   t.Criticality,
			ReviewStatus:  status,
			ReviewComment: comment,
			ReviewType:    "target",
		})
	}

	reqGroups, err := ParseArchiveRequirements(archiveDir)
	if err != nil {
		reqGroups = []ArchiveRequirementGroup{}
	}
	collectReqsFromGroups(&questions, reqGroups, kind, name)

	return questions
}
