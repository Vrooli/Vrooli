// Pending questions endpoint — returns actual question content for all backlog
// items that have unanswered workshop decisions or unreviewed targets/requirements.
// Used by the All tab inline question stepper to render questions without N+1 queries.
package backlog

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/backlogrank"
	"swarm-manager/internal/httputil"
)

// PendingQuestion represents a single question from either a workshop decision
// or a target/requirement review.
type PendingQuestion struct {
	ID       string `json:"id"`
	Source   string `json:"source"` // "workshop" | "review"
	ItemKind string `json:"item_kind"`
	ItemName string `json:"item_name"`

	// Workshop decision fields
	Topic           string           `json:"topic,omitempty"`
	Text            string           `json:"text,omitempty"`
	Context         string           `json:"context,omitempty"`
	Options         []WorkshopOption `json:"options,omitempty"`
	Selected        *string          `json:"selected,omitempty"`
	Freeform        *string          `json:"freeform,omitempty"`
	Notes           *string          `json:"notes,omitempty"`
	RoundNumber     int              `json:"round_number,omitempty"`
	ClarificationID *string          `json:"clarification_id,omitempty"`
	ContextNote     *string          `json:"context_note,omitempty"`

	// Review fields
	Title         string `json:"title,omitempty"`
	Description   string `json:"description,omitempty"`
	Criticality   string `json:"criticality,omitempty"`
	ReviewStatus  string `json:"review_status,omitempty"`
	ReviewComment string `json:"review_comment,omitempty"`
	ReviewType    string `json:"review_type,omitempty"` // "target" | "requirement"
	ModuleID      string `json:"module_id,omitempty"`
}

// PendingQuestionsItem groups pending questions for a single backlog item.
type PendingQuestionsItem struct {
	Kind      BacklogKind       `json:"kind"`
	Name      string            `json:"name"`
	Questions []PendingQuestion `json:"questions"`
}

// PendingQuestionsResponse is the response for the pending-questions endpoint.
type PendingQuestionsResponse struct {
	Items []PendingQuestionsItem `json:"items"`
}

type pendingQuestionSource string

const (
	pendingQuestionSourceWorkshop pendingQuestionSource = "workshop"
	pendingQuestionSourceReview   pendingQuestionSource = "review"
	pendingQuestionSourceAll      pendingQuestionSource = "all"
)

// PendingQuestions returns the actual question content for all backlog items with
// pending workshop decisions or unreviewed targets/requirements.
func (h *Handler) PendingQuestions(w http.ResponseWriter, r *http.Request) {
	source, limit, initiative, ok := parsePendingQuestionsQuery(w, r)
	if !ok {
		return
	}

	items, err := h.store.LoadAll(nil)
	if err != nil {
		apierr.MapError(w, "[backlog] pending-questions", apierr.Internal("%s", err.Error()))
		return
	}

	depthMap, unblockingMap, itemsByKey := rankPendingQuestionItems(items)
	var result []PendingQuestionsItem

	for _, item := range items {
		if initiative != "" && item.Initiative != initiative {
			continue
		}

		itemDir := h.store.ItemDir(item.Kind, item.Name)
		var questions []PendingQuestion

		if source != pendingQuestionSourceReview {
			questions = append(questions, collectWorkshopQuestions(itemDir, item.Kind, item.Name)...)
		}

		if source != pendingQuestionSourceWorkshop {
			questions = append(questions, collectReviewQuestions(itemDir, item.Kind, item.Name)...)
		}

		if len(questions) > 0 {
			result = append(result, PendingQuestionsItem{
				Kind:      item.Kind,
				Name:      item.Name,
				Questions: questions,
			})
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		left := itemsByKey[backlogrank.Key(string(result[i].Kind), result[i].Name)]
		right := itemsByKey[backlogrank.Key(string(result[j].Kind), result[j].Name)]
		return backlogrank.Less(left, right, depthMap, unblockingMap)
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	if result == nil {
		result = []PendingQuestionsItem{}
	}

	resp := PendingQuestionsResponse{Items: result}
	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] pending-questions", apierr.Internal("failed to encode response"))
	}
}

func parsePendingQuestionsQuery(w http.ResponseWriter, r *http.Request) (pendingQuestionSource, int, string, bool) {
	sourceRaw := strings.TrimSpace(r.URL.Query().Get("source"))
	if sourceRaw == "" {
		sourceRaw = string(pendingQuestionSourceWorkshop)
	}
	source := pendingQuestionSource(strings.ToLower(sourceRaw))
	switch source {
	case pendingQuestionSourceWorkshop, pendingQuestionSourceReview, pendingQuestionSourceAll:
	default:
		apierr.MapError(w, "[backlog] pending-questions", apierr.BadRequest("invalid source: must be workshop, review, or all"))
		return "", 0, "", false
	}

	limitRaw := strings.TrimSpace(r.URL.Query().Get("limit"))
	limit := 0
	if limitRaw != "" {
		parsed, err := strconv.Atoi(limitRaw)
		if err != nil || parsed < 0 {
			apierr.MapError(w, "[backlog] pending-questions", apierr.BadRequest("invalid limit: must be a non-negative integer"))
			return "", 0, "", false
		}
		limit = parsed
	}

	initiative := strings.TrimSpace(r.URL.Query().Get("initiative"))
	return source, limit, initiative, true
}

func rankPendingQuestionItems(items []BacklogItem) (map[string]int, map[string]int, map[string]backlogrank.Item) {
	rankItems := make([]backlogrank.Item, 0, len(items))
	itemsByKey := make(map[string]backlogrank.Item, len(items))
	for _, item := range items {
		rankItem := backlogrank.Item{
			Kind:      string(item.Kind),
			Name:      item.Name,
			Status:    string(item.Status),
			DependsOn: item.DependsOn,
			Archived:  item.ArchivedAt != nil && strings.TrimSpace(*item.ArchivedAt) != "",
			Priority:  item.Priority,
			UpdatedAt: parseBacklogUpdatedAt(item.Updated),
		}
		rankItems = append(rankItems, rankItem)
		itemsByKey[backlogrank.ItemKey(rankItem)] = rankItem
	}
	return backlogrank.ComputeDepthMap(rankItems), backlogrank.ComputeUnblockingMap(rankItems), itemsByKey
}

func parseBacklogUpdatedAt(raw string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}
	}
	return parsed
}

// collectWorkshopQuestions extracts unanswered decision items from the latest workshop round.
func collectWorkshopQuestions(itemDir string, kind BacklogKind, name string) []PendingQuestion {
	latestRound, _, err := LoadLatestRound(itemDir)
	if err != nil || latestRound == nil {
		return nil
	}

	var questions []PendingQuestion
	for _, wi := range latestRound.Items {
		if wi.Type != "decision" {
			continue
		}
		if wi.Selected != nil && strings.TrimSpace(*wi.Selected) != "" {
			continue // already answered
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

// collectReviewQuestions extracts unreviewed targets and requirements.
func collectReviewQuestions(itemDir string, kind BacklogKind, name string) []PendingQuestion {
	archiveDir := filepath.Join(itemDir, "archive")
	info, err := os.Stat(archiveDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	var questions []PendingQuestion

	// Targets
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
		// Include if unreviewed (no review state, or explicitly "unreviewed")
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

	// Requirements
	reqGroups, err := ParseArchiveRequirements(archiveDir)
	if err != nil {
		reqGroups = []ArchiveRequirementGroup{}
	}

	collectReqsFromGroups(&questions, reqGroups, kind, name)

	return questions
}

// collectReqsFromGroups recursively collects unreviewed requirements from groups.
func collectReqsFromGroups(questions *[]PendingQuestion, groups []ArchiveRequirementGroup, kind BacklogKind, name string) {
	for _, group := range groups {
		for _, req := range group.Requirements {
			status := req.ReviewStatus
			if status == "approved" || status == "flagged" {
				continue
			}
			*questions = append(*questions, PendingQuestion{
				ID:            req.ID,
				Source:        "review",
				ItemKind:      string(kind),
				ItemName:      name,
				Title:         req.Title,
				Description:   req.Description,
				Criticality:   req.Category,
				ReviewStatus:  status,
				ReviewComment: req.ReviewComment,
				ReviewType:    "requirement",
				ModuleID:      group.ID,
			})
		}
		// Recurse into children
		collectReqsFromGroups(questions, group.Children, kind, name)
	}
}
