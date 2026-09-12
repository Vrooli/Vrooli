package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

func TestPruneOperationalHistoryUsesLiveConfiguredWindowAndIsRateLimited(t *testing.T) {
	store := &mockStore{}
	h := NewWithInterface(checks.NewRegistry(&platform.Capabilities{}), store, &platform.Capabilities{})
	h.SetHistoryRetentionHoursProvider(func() int { return 24 })
	h.pruneOperationalHistory(context.Background())
	if store.retentionCalls != 1 {
		t.Fatalf("retention calls = %d, want 1", store.retentionCalls)
	}
	if got := time.Since(store.retentionBefore); got < 23*time.Hour || got > 25*time.Hour {
		t.Fatalf("retention cutoff age = %s, want about 24h", got)
	}
	h.pruneOperationalHistory(context.Background())
	if store.retentionCalls != 1 {
		t.Fatalf("retention calls after immediate retry = %d, want 1", store.retentionCalls)
	}
}
