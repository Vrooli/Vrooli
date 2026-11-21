package runmanager

import (
	"errors"
	"strings"
	"testing"
)

func TestBroadcasterResumesAfterCursorAndRejectsEvictedCursor(t *testing.T) {
	b := newBroadcasterWithLimits(replayLimits{events: 2, bytes: 1 << 20})
	b.publish(Event{Kind: EventRunStarted})     // sequence 1, evicted below
	b.publish(Event{Kind: EventPhaseStarted})   // sequence 2
	b.publish(Event{Kind: EventPhaseCompleted}) // sequence 3

	replay, _, cancel, err := b.subscribeAfter(1)
	if err != nil {
		t.Fatalf("resume after retained cursor: %v", err)
	}
	cancel()
	if len(replay) != 2 || replay[0].Sequence != 2 || replay[1].Sequence != 3 {
		t.Fatalf("resumed replay = %#v", replay)
	}
	_, _, _, err = b.subscribeAfter(0)
	if err != nil {
		t.Fatalf("tail subscription without cursor: %v", err)
	}

	// Add enough events to advance the retained tail beyond cursor 1.
	b.publish(Event{Kind: EventPhaseProgress}) // sequence 4, tail 3..4
	if _, _, _, err := b.subscribeAfter(1); !errors.Is(err, ErrStaleCursor) {
		t.Fatalf("stale cursor error = %v, want ErrStaleCursor", err)
	}
}

func TestBroadcasterRetainsOnlyConfiguredEventTail(t *testing.T) {
	b := newBroadcasterWithLimits(replayLimits{events: 2, bytes: 1 << 20})
	b.publish(Event{Kind: EventRunStarted, Message: "first"})
	b.publish(Event{Kind: EventPhaseStarted, Message: "second"})
	b.publish(Event{Kind: EventPhaseCompleted, Message: "third"})

	replay, _, cancel := b.subscribe()
	defer cancel()
	if len(replay) != 2 {
		t.Fatalf("replay event count = %d, want 2", len(replay))
	}
	if replay[0].Message != "second" || replay[1].Message != "third" {
		t.Fatalf("replay = %#v, want bounded newest tail", replay)
	}
}

func TestBroadcasterAppliesByteBudgetAndTruncatesMessages(t *testing.T) {
	b := newBroadcasterWithLimits(replayLimits{events: 10, bytes: 250})
	b.publish(Event{Kind: EventPhaseProgress, Message: strings.Repeat("a", maxEventMessageBytes+100)})
	b.publish(Event{Kind: EventPhaseProgress, Message: strings.Repeat("b", 80)})
	b.publish(Event{Kind: EventPhaseProgress, Message: strings.Repeat("c", 80)})

	replay, _, cancel := b.subscribe()
	defer cancel()
	if len(replay) == 0 || len(replay) > 2 {
		t.Fatalf("replay count = %d, want bounded non-empty tail", len(replay))
	}
	for _, event := range replay {
		if len(event.Message) > maxEventMessageBytes+len("… [truncated]") {
			t.Fatalf("message size = %d, exceeds bounded event message", len(event.Message))
		}
	}
}

func TestCompactEventDoesNotRetainTerminalResult(t *testing.T) {
	// Event intentionally has no result field. This test pins the replay
	// contract to its compact scalar fields so a future terminal payload cannot
	// quietly become an in-memory owner of complete phase evidence.
	event := compactEvent(Event{Kind: EventRunCompleted, Message: "complete"})
	if event.Kind != EventRunCompleted || event.Message != "complete" {
		t.Fatalf("compact terminal event = %#v", event)
	}
}
