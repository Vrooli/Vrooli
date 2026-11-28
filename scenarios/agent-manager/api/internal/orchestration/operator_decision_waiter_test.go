package orchestration

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestOperatorDecisionWaiterProducer(t *testing.T) {
	if NewOperatorDecisionWaiter().Producer() != ProducerOperatorDecision {
		t.Fatalf("producer key must be %q", ProducerOperatorDecision)
	}
}

func TestOperatorDecisionWaiterBlocksUntilContextDone(t *testing.T) {
	w := NewOperatorDecisionWaiter()

	// It must not return before the context is done (an early return would wake
	// the agent spuriously before the deadline or an external resolution).
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var result string
	var err error
	go func() {
		result, err = w.Wait(ctx, "deletion-grant-abc")
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("Wait returned before the context was done")
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return promptly after the context was cancelled")
	}
	if err != nil {
		t.Fatalf("Wait must not error on cancellation, got %v", err)
	}
	// The returned status is only consumed on the deadline path; it should name
	// the decision so a timed-out wake is actionable.
	if !strings.Contains(result, "deletion-grant-abc") {
		t.Fatalf("pending status should reference the decision key, got %q", result)
	}
}
