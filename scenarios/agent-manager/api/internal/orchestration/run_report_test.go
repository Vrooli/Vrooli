package orchestration

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"

	"github.com/google/uuid"
)

func TestRunProjectionCanonicalTokensMatchTimeAccountingBuckets(t *testing.T) {
	start := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	end := start.Add(9 * time.Minute)
	runID := uuid.New()
	run := &domain.Run{ID: runID, CreatedAt: start, StartedAt: &start, EndedAt: &end, Status: domain.RunStatusComplete}
	newInvocation := func(offset time.Duration) *domain.RunEvent {
		event := domain.NewStatusEvent(runID, "complete", "running", "continuation started")
		event.Timestamp = start.Add(offset)
		return event
	}
	usage := func(offset time.Duration, data *domain.UsageEventData) *domain.RunEvent {
		return &domain.RunEvent{ID: uuid.New(), RunID: runID, Timestamp: start.Add(offset), Data: data}
	}
	events := []*domain.RunEvent{
		newInvocation(0),
		usage(time.Minute, &domain.UsageEventData{InputTokens: 20, OutputTokens: 10, CacheCreationTokens: 5, CacheReadTokens: 5}),
		usage(2*time.Minute, &domain.UsageEventData{InputTokens: 50, OutputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 20, ReconciliationAuthority: true}),
		usage(2*time.Minute+time.Second, &domain.UsageEventData{InputTokens: 50, OutputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 20, ReconciliationAuthority: true}),
		newInvocation(3 * time.Minute),
		usage(4*time.Minute, &domain.UsageEventData{InputTokens: 20, OutputTokens: 10, CacheCreationTokens: 5, CacheReadTokens: 5}),
		usage(5*time.Minute, &domain.UsageEventData{InputTokens: 50, OutputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 20, ReconciliationAuthority: true}),
		usage(5*time.Minute+time.Second, &domain.UsageEventData{InputTokens: 50, OutputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 20, ReconciliationAuthority: true}),
		newInvocation(6 * time.Minute),
		usage(7*time.Minute, &domain.UsageEventData{InputTokens: 20, OutputTokens: 10, CacheCreationTokens: 5, CacheReadTokens: 5}),
		usage(8*time.Minute, &domain.UsageEventData{InputTokens: 50, OutputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 20, ReconciliationAuthority: true}),
		usage(8*time.Minute+time.Second, &domain.UsageEventData{InputTokens: 50, OutputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 20, ReconciliationAuthority: true}),
	}

	fact := invocationreadmodel.ProjectRun(run, events, end)
	bucketTokens := fact.TimeAccounting.Tokens()
	if fact.TotalTokens != 300 {
		t.Fatalf("canonical aggregate tokens=%d, want 300 across three invocations", fact.TotalTokens)
	}
	if bucketTokens != fact.TotalTokens {
		t.Fatalf("time-accounting buckets=%d, canonical aggregate=%d", bucketTokens, fact.TotalTokens)
	}
	if got, want := fact.TimeAccounting.DurationMS(), end.Sub(start).Milliseconds(); got != want {
		t.Fatalf("duration=%d, want conserved interval %d", got, want)
	}
	if fact.TimeAccounting.UnattributableTokens != 300 {
		t.Fatalf("selected token state buckets changed: %+v", fact.TimeAccounting)
	}
}
