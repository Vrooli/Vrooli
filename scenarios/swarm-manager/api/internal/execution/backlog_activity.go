package execution

import "context"

// HasActiveForBacklog reports whether a backlog item has a queued or live
// execution. It is a read-only guard for the plan-acceptance boundary.
func (s *Service) HasActiveForBacklog(ctx context.Context, kind, name string) bool {
	records, err := s.ListSnapshot(ctx, ListFilters{})
	if err != nil {
		return false
	}
	for _, record := range records {
		if record.BacklogKind != kind || record.BacklogName != name {
			continue
		}
		switch record.Status {
		case StatusPending, StatusStarting, StatusRunning, StatusNeedsReview, StatusValidating:
			return true
		}
	}
	return false
}
