package execution

import (
	"context"
	"fmt"
	"strings"
)

// FollowUpLatestForBacklog starts a steered follow-up from the newest eligible
// terminal execution for an item. Item-level review decisions should not need
// to know execution record identifiers or reach into execution persistence.
func (s *Service) FollowUpLatestForBacklog(ctx context.Context, backlogKind, backlogName, steering string) (Record, error) {
	records, err := s.store.Load()
	if err != nil {
		return Record{}, err
	}
	var parent *Record
	for index := range records {
		candidate := &records[index]
		if candidate.BacklogKind != strings.TrimSpace(backlogKind) || candidate.BacklogName != strings.TrimSpace(backlogName) {
			continue
		}
		switch candidate.Status {
		case StatusCompleted, StatusFailed, StatusNeedsFixup:
		default:
			continue
		}
		if parent == nil || candidate.UpdatedAt > parent.UpdatedAt {
			parent = candidate
		}
	}
	if parent == nil {
		return Record{}, fmt.Errorf("no terminal execution is available for %s/%s", backlogKind, backlogName)
	}
	return s.FollowUp(ctx, FollowUpRequest{
		ExecutionID:     parent.ExecutionID,
		FollowUpType:    "followup",
		Context:         strings.TrimSpace(steering),
		RunMode:         "new",
		SourceReviewRef: "backlog/" + strings.TrimSpace(backlogKind) + "/" + strings.TrimSpace(backlogName),
	})
}
