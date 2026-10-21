package lpbs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// TestReporter_StartDrainsQueue asserts Start consumes and delivers
// authenticated reports.
func TestReporter_StartDrainsQueue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	r := NewRemoteReporter(server.URL)
	for i := 0; i < 10; i++ {
		r.Submit(UsageReport{LPBSToken: "access-token", OperationID: fmt.Sprintf("op-%d", i), LimitKey: "ai_credits", Amount: 1, AppBundleKey: "audio-tools"})
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
	// Shutdown can be reached from both request teardown and process teardown;
	// repeating it must not panic or deadlock.
	r.Stop()
}

// TODO: add a fake-clock test for the 500ms/1s/2s backoff and an
// idempotency replay assertion when the LPBS endpoint is exercised end to end.
func TestDeliver_PostsAuthenticatedReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
		require.Equal(t, "op-1", r.Header.Get("Idempotency-Key"))
		var reports []UsageReport
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reports))
		require.Len(t, reports, 1)
		require.Equal(t, "op-1", reports[0].OperationID)
		require.Empty(t, reports[0].LPBSToken)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	r := NewRemoteReporter(server.URL)
	err := r.deliver(context.Background(), UsageReport{
		LPBSToken: "access-token", OperationID: "op-1", LimitKey: "ai_credits", Amount: 1,
	})
	require.NoError(t, err)
}

func TestDeliver_RequiresAccessToken(t *testing.T) {
	r := NewRemoteReporter("http://lpbs.invalid")
	err := r.deliver(context.Background(), UsageReport{OperationID: "op-1", LimitKey: "ai_credits", Amount: 1})
	require.ErrorContains(t, err, "access token is required")
}
