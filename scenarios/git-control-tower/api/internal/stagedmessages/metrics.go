package stagedmessages

import (
	"context"
	"time"

	"git-control-tower/internal/reviewjobs"
)

// OperationalMetrics is a snapshot derived only from durable staged-message
// rows. Rates are nil when there is no denominator; unknown start/token
// receipts are never counted as zero-work refusals.
type OperationalMetrics struct {
	ObservedAt             time.Time        `json:"observed_at"`
	Jobs                   int64            `json:"jobs"`
	TerminalJobs           int64            `json:"terminal_jobs"`
	AwaitingProviderJobs   int64            `json:"awaiting_provider_jobs"`
	FailedJobs             int64            `json:"failed_jobs"`
	FailureRate            *float64         `json:"failure_rate,omitempty"`
	FailuresByCode         map[string]int64 `json:"failures_by_code"`
	TerminalDurationCount  int64            `json:"terminal_duration_count"`
	TerminalDurationMillis int64            `json:"terminal_duration_millis_total"`
	ZeroWorkRefusals       int64            `json:"proven_zero_work_refusals"`
	DuplicateRetries       int64            `json:"duplicate_retries"`
	StuckDispatch          int64            `json:"stuck_dispatch"`
	StuckReconciliation    int64            `json:"stuck_reconciliation"`
	StuckThresholdSeconds  int64            `json:"stuck_threshold_seconds"`
}

// Metrics returns a read-only aggregate. Dispatching jobs (and legacy running
// jobs) without an execution handle await owner admission; jobs with a handle
// await reconciliation with their owner execution.
func (s *Service) Metrics(ctx context.Context, now time.Time, stuckAfter time.Duration) (OperationalMetrics, error) {
	jobs, err := s.store.List(ctx)
	if err != nil {
		return OperationalMetrics{}, err
	}
	if now.IsZero() {
		now = s.now()
	}
	if stuckAfter <= 0 {
		stuckAfter = 5 * time.Minute
	}
	out := OperationalMetrics{ObservedAt: now.UTC(), FailuresByCode: map[string]int64{}, StuckThresholdSeconds: int64(stuckAfter.Seconds())}
	var durationTotal time.Duration
	for _, job := range jobs {
		if job.Operation != operation {
			continue
		}
		state := projectedLifecycleState(job)
		out.Jobs++
		if state.Terminal() {
			out.TerminalJobs++
			duration := job.UpdatedAt.Sub(job.CreatedAt)
			if duration < 0 {
				duration = 0
			}
			durationTotal += duration
			out.TerminalDurationCount++
		}
		if state == reviewjobs.AwaitingProvider {
			out.AwaitingProviderJobs++
		}
		if state == reviewjobs.Failed || state == reviewjobs.AwaitingProvider {
			if state == reviewjobs.Failed {
				out.FailedJobs++
			}
			code := job.ErrorCode
			if code == "" {
				code = "unclassified"
			}
			out.FailuresByCode[code]++
		}
		if (state.Terminal() || state == reviewjobs.AwaitingProvider) && job.ExecutionStartedKnown && !job.ExecutionStarted && job.TokensKnown && job.TokensUsed == 0 && job.ChargeKnown && job.ChargeMicroUSD == 0 {
			out.ZeroWorkRefusals++
		}
		if !state.Terminal() && state != reviewjobs.AwaitingProvider && now.Sub(job.UpdatedAt) >= stuckAfter {
			if (state == reviewjobs.Queued || state == reviewjobs.Dispatching || state == reviewjobs.Running) && job.ExecutionID == "" {
				out.StuckDispatch++
			} else if job.ExecutionID != "" {
				out.StuckReconciliation++
			}
		}
		out.DuplicateRetries += job.DuplicateRetries
	}
	if out.TerminalJobs > 0 {
		rate := float64(out.FailedJobs) / float64(out.TerminalJobs)
		out.FailureRate = &rate
	}
	if out.TerminalDurationCount > 0 {
		out.TerminalDurationMillis = durationTotal.Milliseconds()
	}
	return out, nil
}
