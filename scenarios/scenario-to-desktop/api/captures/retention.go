package captures

import (
	"context"
	"sort"
	"time"
)

// RetentionPolicy bounds active capture storage. Voided captures remain
// readable for audit and are therefore never selected for pruning.
type RetentionPolicy struct {
	MaxAge   time.Duration
	MaxBytes int64
}

// Prune removes the oldest non-voided captures until both configured bounds
// hold. A zero bound is disabled. The operation is deterministic and does not
// follow files that lack metadata.
func (s *Service) Prune(ctx context.Context, scenarioName string, policy RetentionPolicy) (int, error) {
	captures, err := s.store.List(scenarioName)
	if err != nil {
		return 0, err
	}
	eligible := make([]Capture, 0, len(captures))
	var bytes int64
	for _, capture := range captures {
		if capture.VoidReason == "" {
			eligible = append(eligible, capture)
			bytes += capture.FileSizeBytes
		}
	}
	sort.SliceStable(eligible, func(i, j int) bool { return eligible[i].CreatedAt.Before(eligible[j].CreatedAt) })
	removed := 0
	for _, capture := range eligible {
		if err := ctx.Err(); err != nil {
			return removed, err
		}
		tooOld := policy.MaxAge > 0 && time.Since(capture.CreatedAt) > policy.MaxAge
		tooLarge := policy.MaxBytes > 0 && bytes > policy.MaxBytes
		if !tooOld && !tooLarge {
			continue
		}
		if err := s.DeleteCapture(scenarioName, capture.ID); err != nil {
			return removed, err
		}
		bytes -= capture.FileSizeBytes
		removed++
	}
	return removed, nil
}
