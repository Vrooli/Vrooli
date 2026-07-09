package sttchain

import "testing"

// TestStreamEventDurability pins the single event-durability contract that all
// three streaming hops consume: Partial is the only disposable (droppable)
// event class; every other kind is durable (ordered, lossless). See
// docs/domains/stt/streaming-pipeline.md#event-durability-contract.
func TestStreamEventDurability(t *testing.T) {
	durable := []StreamEventKind{
		StreamEventSegment,
		StreamEventSpeakerRejection,
		StreamEventError,
		StreamEventDone,
		StreamEventWakeWord,
		StreamEventVadState,
	}
	for _, k := range durable {
		ev := StreamEvent{Kind: k}
		if !ev.Durable() {
			t.Errorf("event kind %q must be durable (lossless, ordered)", k)
		}
		if ev.IsDroppable() {
			t.Errorf("event kind %q must not be droppable", k)
		}
	}

	partial := StreamEvent{Kind: StreamEventPartial}
	if partial.Durable() {
		t.Error("partial events must NOT be durable — they are the sole disposable class")
	}
	if !partial.IsDroppable() {
		t.Error("partial events must be droppable (coalesce-to-latest / drop under pressure)")
	}
}
