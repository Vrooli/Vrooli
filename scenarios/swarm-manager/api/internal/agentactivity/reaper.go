package agentactivity

import "time"

// ReapExpiredNeedsReview expires parked review activities and records why
// each one stopped occupying its lane. It is safe to call repeatedly.
func (s *Service) ReapExpiredNeedsReview() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.store.Load()
	if err != nil {
		return err
	}
	return s.reapExpiredNeedsReviewLocked(records, time.Now())
}

func (s *Service) reapExpiredNeedsReviewLocked(records []Record, now time.Time) error {
	changed := false
	changedIDs := make(map[string]struct{})
	for i := range records {
		record := &records[i]
		if record.Status != StatusNeedsReview {
			continue
		}
		at := record.UpdatedAt
		if at == "" {
			at = record.FinishedAt
		}
		if at == "" {
			at = record.RequestedAt
		}
		parkedAt, err := time.Parse(time.RFC3339, at)
		if err != nil || now.Sub(parkedAt) <= s.needsReviewTTL {
			continue
		}
		record.Status = StatusFailed
		record.FinishedAt = now.UTC().Format(time.RFC3339)
		record.UpdatedAt = record.FinishedAt
		record.FailureReason = "needs_review expired after " + s.needsReviewTTL.String() + "; no operator review was recorded"
		changed = true
		changedIDs[record.ActivityID] = struct{}{}
	}
	if !changed {
		return nil
	}
	if err := s.store.Save(records); err != nil {
		return err
	}
	for _, record := range records {
		if _, ok := changedIDs[record.ActivityID]; ok {
			s.dispatchStatusUpdate(record)
		}
	}
	return nil
}
