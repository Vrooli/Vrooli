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
		StreamEventAcknowledgement,
		StreamEventSessionStatus,
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

	for _, k := range []StreamEventKind{StreamEventPartial, StreamEventVadState} {
		ev := StreamEvent{Kind: k}
		if ev.Durable() {
			t.Errorf("event kind %q must be coalescible progress", k)
		}
		if !ev.IsDroppable() {
			t.Errorf("event kind %q must be droppable progress", k)
		}
	}
}

func TestConsumptionCursorAcknowledgesCoverageAtBoundedIntervals(t *testing.T) {
	var acknowledgements []AcknowledgementEvent
	cursor := NewConsumptionCursor(func(ev StreamEvent) {
		if ev.Acknowledgement != nil {
			acknowledgements = append(acknowledgements, *ev.Acknowledgement)
		}
	}, 100)
	cursor.Observe(AudioChunk{Sequence: 0, StartSample: 0, EndSample: 40})
	cursor.Observe(AudioChunk{Sequence: 1, StartSample: 40, EndSample: 80})
	if len(acknowledgements) != 1 {
		t.Fatalf("expected the first coverage acknowledgement, got %d", len(acknowledgements))
	}
	cursor.Observe(AudioChunk{Sequence: 2, StartSample: 80, EndSample: 140})
	if len(acknowledgements) != 2 || acknowledgements[1].ProcessedSequence != 2 {
		t.Fatalf("expected interval acknowledgement for sequence 2, got %+v", acknowledgements)
	}
	cursor.Observe(AudioChunk{Sequence: 2, StartSample: 80, EndSample: 140})
	cursor.Flush()
	if len(acknowledgements) != 2 {
		t.Fatalf("duplicate observation moved the cursor: %+v", acknowledgements)
	}
}

func TestConsumptionCursorFlushesShortTailAndRejectsGaps(t *testing.T) {
	var got []AcknowledgementEvent
	cursor := NewConsumptionCursor(func(ev StreamEvent) {
		if ev.Acknowledgement != nil {
			got = append(got, *ev.Acknowledgement)
		}
	}, 1_000)
	cursor.Observe(AudioChunk{Sequence: 0, StartSample: 0, EndSample: 80})
	cursor.Observe(AudioChunk{Sequence: 2, StartSample: 160, EndSample: 240})
	cursor.Flush()
	if len(got) != 1 || got[0].ProcessedSequence != 0 || got[0].ProcessedEndSample != 80 {
		t.Fatalf("gap or short-tail handling is incorrect: %+v", got)
	}
}
