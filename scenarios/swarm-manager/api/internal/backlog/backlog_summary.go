// Combined summary endpoint — returns feedback, maturity, and pending-questions
// data in a single response. This avoids three separate LoadAll + workshop-round
// traversals, cutting the BacklogPage bootstrap from 3 sequential RPCs to 1.
package backlog

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/backlogrank"
	"swarm-manager/internal/httputil"
)

// BacklogSummaryResponse combines feedback, maturity, and pending-questions data.
type BacklogSummaryResponse struct {
	Feedback         FeedbackSummaryResponse  `json:"feedback"`
	Maturity         MaturitySummaryResponse  `json:"maturity"`
	PendingQuestions PendingQuestionsResponse `json:"pending_questions"`
}

// itemRoundData holds the workshop round data loaded once per item.
type itemRoundData struct {
	item       BacklogItem
	itemDir    string
	round      *WorkshopRound
	roundCount int
	roundOK    bool
}

// BacklogSummary returns a combined summary covering feedback, maturity, and
// pending questions in one round-trip. Internally it loads all items once and
// performs a single pass over workshop rounds.
func (h *Handler) BacklogSummary(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.LoadAll(nil)
	if err != nil {
		apierr.MapError(w, "[backlog] summary", apierr.Internal("%s", err.Error()))
		return
	}

	rounds := make([]itemRoundData, len(items))
	for i, item := range items {
		itemDir := h.store.ItemDir(item.Kind, item.Name)
		latestRound, roundCount, roundErr := LoadLatestRound(itemDir)
		rounds[i] = itemRoundData{
			item: item, itemDir: itemDir,
			round: latestRound, roundCount: roundCount,
			roundOK: roundErr == nil,
		}
	}

	feedbackItems, totalPending := buildFeedbackSummary(rounds)
	maturityItems := buildMaturitySummary(rounds)
	pendingItems := buildPendingQuestions(rounds)

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
		apierr.MapError(w, "[backlog] summary", apierr.Internal("failed to encode response"))
	}
}

func buildFeedbackSummary(rounds []itemRoundData) ([]FeedbackItemSummary, int) {
	var items []FeedbackItemSummary
	totalPending := 0
	for _, rd := range rounds {
		if !rd.roundOK || rd.round == nil {
			continue
		}
		pending := CountPendingDecisions(rd.round)
		if pending > 0 {
			items = append(items, FeedbackItemSummary{
				Kind: rd.item.Kind, Name: rd.item.Name,
				Title: rd.item.Title, PendingDecisions: pending,
			})
			totalPending += pending
		}
	}
	if items == nil {
		items = []FeedbackItemSummary{}
	}
	return items, totalPending
}

func buildMaturitySummary(rounds []itemRoundData) []MaturityItemSummary {
	items := make([]MaturityItemSummary, 0, len(rounds))
	for _, rd := range rounds {
		rawScores := make(map[string]int, len(ReadinessDimensions))
		for _, dim := range ReadinessDimensions {
			rawScores[dim] = 0
		}
		if rd.roundOK && rd.round != nil {
			for _, dim := range ReadinessDimensions {
				if v, ok := rd.round.Readiness[dim]; ok {
					rawScores[dim] = v
				}
			}
		}
		effectiveScores := ComputeEffectiveScores(rawScores, rd.roundCount, rd.item.Kind)
		items = append(items, MaturityItemSummary{
			Kind: rd.item.Kind, Name: rd.item.Name, Title: rd.item.Title,
			RoundsCompleted:  rd.roundCount,
			RawScores:        rawScores,
			EffectiveScores:  effectiveScores,
			Ready:            IsReady(effectiveScores),
			PendingItems:     CountPendingDecisions(rd.round),
			PendingSynthesis: NeedsSynthesis(rd.round),
			HasPlan:          hasCanonicalPlan(rd.item, rd.itemDir),
		})
	}
	return items
}

func buildPendingQuestions(rounds []itemRoundData) []PendingQuestionsItem {
	var items []PendingQuestionsItem
	backlogItems := make([]BacklogItem, 0, len(rounds))
	for _, rd := range rounds {
		backlogItems = append(backlogItems, rd.item)
	}
	depthMap, unblockingMap, itemsByKey := rankPendingQuestionItems(backlogItems)
	for _, rd := range rounds {
		var questions []PendingQuestion
		questions = append(questions, collectWorkshopQuestionsFromRound(rd.round, rd.item.Kind, rd.item.Name)...)
		questions = append(questions, collectReviewQuestionsFromDir(rd.itemDir, rd.item.Kind, rd.item.Name)...)
		if len(questions) > 0 {
			items = append(items, PendingQuestionsItem{
				Kind: rd.item.Kind, Name: rd.item.Name, Questions: questions,
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
			ID:              wi.ID,
			Source:          "workshop",
			ItemKind:        string(kind),
			ItemName:        name,
			Topic:           wi.Topic,
			Text:            wi.Text,
			Context:         wi.Context,
			Options:         wi.Options,
			Selected:        wi.Selected,
			Freeform:        wi.Freeform,
			Notes:           wi.Notes,
			RoundNumber:     latestRound.RoundNum,
			ClarificationID: wi.ClarificationID,
			ContextNote:     wi.ContextNote,
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
