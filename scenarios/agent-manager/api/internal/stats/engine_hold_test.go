package stats

import (
	"context"
	"testing"
	"time"
)

// TestEngineHoldWatermarkBlocksRefreshUntilReleased: storage compaction must be
// able to stop the engine reading run_events by rowid while VACUUM renumbers
// them, and hand it the rewritten position.
func TestEngineHoldWatermarkBlocksRefreshUntilReleased(t *testing.T) {
	engine, _, _ := newTestEngine(t)
	current, set, release := engine.HoldWatermark()
	if current != 0 {
		t.Fatalf("fresh watermark=%d", current)
	}
	done := make(chan error, 1)
	go func() { done <- engine.Refresh(context.Background()) }()
	select {
	case <-done:
		t.Fatal("Refresh ran while the watermark was held")
	case <-time.After(50 * time.Millisecond):
	}
	set(7)
	release()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	engine.mu.RLock()
	defer engine.mu.RUnlock()
	if engine.watermark != 7 {
		t.Fatalf("watermark=%d, want the rewritten 7", engine.watermark)
	}
}
