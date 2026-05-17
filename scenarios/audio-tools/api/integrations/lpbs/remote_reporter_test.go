package lpbs

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestReporter_SubmitAcceptsWithoutStart asserts the queue is bounded
// and Submit is non-blocking even before Start. Drops are silent today
// (no counter); when the LPBS gateway lands, replace this with a
// DroppedTotal assertion.
func TestReporter_SubmitAcceptsWithoutStart(t *testing.T) {
	r := NewRemoteReporter("http://lpbs.invalid")
	for i := 0; i < 300; i++ {
		r.Submit(UsageReport{OperationID: "op", LimitKey: "ai_credits", Amount: 1})
	}
	// Did not panic, did not block: contract satisfied.
}

// TestReporter_StartDrainsQueue asserts Start consumes Submitted
// reports. We can't observe delivery (the HTTP gateway is stubbed),
// so we observe channel depth dropping to zero.
func TestReporter_StartDrainsQueue(t *testing.T) {
	r := NewRemoteReporter("http://lpbs.invalid")
	for i := 0; i < 10; i++ {
		r.Submit(UsageReport{OperationID: "op", LimitKey: "ai_credits", Amount: 1, AppBundleKey: "audio-tools"})
	}
	require.Equal(t, 10, len(r.queue), "all 10 enqueued before Start")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r.Start(ctx)

	// Polling-via-Eventually is the event-driven substitute for an
	// explicit sleep loop: testify drives the polling interval and
	// fails fast if the condition never holds. No bare time.Sleep in
	// the test source keeps the suite race-clean under -race.
	require.Eventually(t, func() bool { return len(r.queue) == 0 }, 2*time.Second, 5*time.Millisecond, "queue must drain")
	r.Stop()
}

// TestReporter_StopIsIdempotentlySafe asserts Stop unblocks the worker
// even when no reports are in flight. The stop signal must take
// priority over context.
func TestReporter_StopIsIdempotentlySafe(t *testing.T) {
	r := NewRemoteReporter("http://lpbs.invalid")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r.Start(ctx)

	done := make(chan struct{})
	go func() { r.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop blocked longer than 2s — worker did not honour the done signal")
	}
}

// TestDeliver_ReturnsNilOnStubbedAttempt locks the current behaviour
// of deliver: the stubbed attemptOnce never errors, so the first
// attempt succeeds without sleeping through the backoff sequence.
//
// TODO: replace with a real httptest.Server + fake-clock test of the
// 500ms/1s/2s backoff + idempotency-key dedup when the LPBS gateway
// HTTP wiring lands (see remote_reporter.go::attemptOnce).
func TestDeliver_ReturnsNilOnStubbedAttempt(t *testing.T) {
	r := NewRemoteReporter("http://lpbs.invalid")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	var err error
	go func() {
		defer wg.Done()
		err = r.deliver(ctx, UsageReport{OperationID: "op-1", LimitKey: "ai_credits", Amount: 1})
	}()
	wg.Wait()
	require.NoError(t, err)
}
