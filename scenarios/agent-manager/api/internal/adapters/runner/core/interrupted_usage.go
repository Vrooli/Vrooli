package core

import (
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// interruptedUsageInputs identifies the invocation whose process just exited.
type interruptedUsageInputs struct {
	runID     uuid.UUID
	sessionID string
	env       []string
	config    *domain.RunConfig
	startedAt time.Time
	sink      runner.EventSink
}

// recoverInterruptedUsage persists usage the harness recorded in its own
// session log when this invocation's stream carried none, which happens when
// the process stops mid-turn (a park ends the turn this way). It runs after the
// process has exited and before the result is classified, so the recovered
// usage lands inside this invocation and in its metrics. An invocation whose
// stream produced any usage is left alone, so nothing is counted twice.
func (r *Runner) recoverInterruptedUsage(in interruptedUsageInputs, observed *[]*domain.RunEvent, metrics *runner.ExecutionMetrics, lastAssistant *string) {
	recoverer, ok := r.codec.(codecs.InterruptedUsageRecoverer)
	if !ok || in.sessionID == "" {
		return
	}
	for _, event := range *observed {
		if _, isUsage := event.Data.(*domain.UsageEventData); isUsage {
			return
		}
	}
	req := codecs.InterruptedUsageRequest{
		RunID:     in.runID,
		SessionID: in.sessionID,
		Env:       in.env,
		StartedAt: in.startedAt,
		EndedAt:   time.Now(),
	}
	if in.config != nil {
		req.Model, req.Billing = in.config.Model, in.config.Billing
	}
	events, note := recoverer.RecoverInterruptedUsage(req)
	if note != "" && in.sink != nil {
		_ = in.sink.Emit(domain.NewLogEvent(in.runID, "info", note))
	}
	for _, event := range events {
		if event == nil {
			continue
		}
		r.codec.UpdateMetrics(event, metrics, lastAssistant)
		*observed = append(*observed, event)
		if in.sink != nil {
			_ = in.sink.Emit(event)
		}
	}
}
