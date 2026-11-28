package aisearch

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"business-health/internal/wizard"
	"github.com/stretchr/testify/require"
)

func TestWizardHinterBoundsConcurrencyAndPreservesTargetOrder(t *testing.T) {
	var active atomic.Int64
	var peak atomic.Int64
	h := &WizardHinter{search: func(ctx context.Context, query string, _ int, _ SearchMode) (*SearchResponse, error) {
		current := active.Add(1)
		defer active.Add(-1)
		for observed := peak.Load(); current > observed; observed = peak.Load() {
			if peak.CompareAndSwap(observed, current) {
				break
			}
		}
		select {
		case <-time.After(15 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return &SearchResponse{Results: []SearchHit{{Scenario: "other", Title: query}}}, nil
	}}
	targets := make([]wizard.OTAnswer, 12)
	for i := range targets {
		targets[i] = wizard.OTAnswer{Title: fmt.Sprintf("target-%02d", i)}
	}

	hints := h.Hints(context.Background(), "current", targets)
	require.Len(t, hints, len(targets))
	require.GreaterOrEqual(t, peak.Load(), int64(2), "queries should overlap")
	require.LessOrEqual(t, peak.Load(), int64(maxConcurrentHintSearches))
	for i, hint := range hints {
		require.Equal(t, targets[i].Title, hint.Capability)
	}
}

func TestWizardHinterHonorsRequestCancellation(t *testing.T) {
	started := make(chan struct{}, 1)
	h := &WizardHinter{search: func(ctx context.Context, _ string, _ int, _ SearchMode) (*SearchResponse, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	targets := []wizard.OTAnswer{{Title: "first"}, {Title: "second"}}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	start := time.Now()
	hints := h.Hints(ctx, "current", targets)
	require.Empty(t, hints)
	require.Less(t, time.Since(start), 500*time.Millisecond)
	select {
	case <-started:
	default:
		t.Fatal("search was not started")
	}
}
