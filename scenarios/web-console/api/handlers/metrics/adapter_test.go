package metrics

import (
	"context"
	"testing"

	intmetrics "web-console/internal/metrics"
)

func TestAdapterProjectsOperationalMetrics(t *testing.T) {
	m := intmetrics.New()
	m.SessionsCreated.Store(3)
	m.ActiveSessions.Store(2)
	m.VoiceSkipVerificationTotal.Store(7)
	m.RecoveryPreservedForNow.Store(4)
	got := (&Adapter{Metrics: m}).Snapshot(context.Background())
	if got.Sessions.Created != 3 || got.Sessions.Active != 2 || got.VoiceSkipVerificationTotal != 7 || got.Recovery.PreservedForNow != 4 {
		t.Fatalf("snapshot=%+v", got)
	}
}
