package supervision

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/eventlog"
)

// TestSchedulerPauseBlocksWatchProcessing: while storage compaction renumbers
// run_events rowids, no watch may read or commit a rowid cursor.
func TestSchedulerPauseBlocksWatchProcessing(t *testing.T) {
	repo, _ := testRepository(t)
	service := NewService(repo, &cohortSource{retention: eventlog.RetentionState{Generation: 1}})
	scheduler := NewScheduler(repo, NewProcessor(service, nil, nil), nil)

	resume := scheduler.Pause()
	done := make(chan error, 1)
	go func() {
		_, err := scheduler.RecoverOnce(context.Background())
		done <- err
	}()
	select {
	case <-done:
		t.Fatal("watch processing ran while paused")
	case <-time.After(50 * time.Millisecond):
	}
	resume()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
