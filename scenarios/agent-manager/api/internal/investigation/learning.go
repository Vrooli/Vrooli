package investigation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// LearningUpdater is the small persistence seam used after a result has been
// committed. A memory outage must not reopen or rewrite the diagnosis; only
// its delivery state may advance from pending to recorded.
type LearningUpdater interface {
	UpdateLearning(context.Context, string, Learning) (*Lifecycle, error)
}

func (r *SQLiteRepository) UpdateLearning(ctx context.Context, id string, learning Learning) (*Lifecycle, error) {
	item, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Result == nil || item.OperationStatus != OperationCompleted {
		return nil, fmt.Errorf("%w: learning can only be updated for a completed result", ErrInvalidTransition)
	}
	item.Result.Learning = learning
	raw, err := json.Marshal(item.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal investigation learning state: %w", err)
	}
	now := r.now().UTC()
	if _, err := r.db.ExecContext(ctx, `UPDATE investigations SET result_json=?,updated_at=? WHERE investigation_id=? AND operation_status=?`, string(raw), formatTime(now), id, OperationCompleted); err != nil {
		return nil, fmt.Errorf("update investigation learning state: %w", err)
	}
	return r.Get(ctx, id)
}

// StableLearningAttempt returns the identity and timestamps used by the
// outcome-linked memory record. It intentionally keeps diagnostic success
// separate from task success: a completed diagnosis alone is not causal
// evidence that advice helped.
func StableLearningAttempt(item *Lifecycle) (Learning, time.Time, time.Time) {
	if item == nil || item.Result == nil {
		return Learning{}, time.Time{}, time.Time{}
	}
	attemptID := item.ID + "/attempt-1"
	if item.Result != nil && item.Result.Learning.AttemptID != "" {
		attemptID = item.Result.Learning.AttemptID
	}
	started := item.CreatedAt.UTC()
	finished := item.UpdatedAt.UTC()
	if item.Result != nil {
		if parsed, err := time.Parse(time.RFC3339Nano, item.Result.EvidenceCut.CapturedAt); err == nil {
			started = parsed.UTC()
		}
		if parsed, err := time.Parse(time.RFC3339Nano, item.Result.Applicability.CheckedAt); err == nil {
			finished = parsed.UTC()
		}
	}
	return Learning{AttemptID: attemptID, CaptureState: item.Result.Learning.CaptureState, AdviceVerdict: item.Result.Learning.AdviceVerdict}, started, finished
}
