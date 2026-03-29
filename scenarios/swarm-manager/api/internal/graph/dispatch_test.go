package graph

import "testing"

type recordingBroadcaster struct {
	events []WSMessage
}

func (r *recordingBroadcaster) BroadcastUpdate(event string, payload any) {
	r.events = append(r.events, NewWSMessage(event, payload))
}

type recordingInvalidator struct {
	lenses       []Lens
	focusNodeIDs []string
}

func (r *recordingInvalidator) Invalidate(lenses ...Lens) {
	r.lenses = append(r.lenses, lenses...)
}

func (r *recordingInvalidator) InvalidateFocus(focusNodeID string) {
	r.focusNodeIDs = append(r.focusNodeIDs, focusNodeID)
}

func TestDispatchInvalidateBroadcastsAndClearsCache(t *testing.T) {
	broadcaster := &recordingBroadcaster{}
	invalidator := &recordingInvalidator{}
	dispatch := NewDispatch(broadcaster, invalidator)

	dispatch.DispatchInvalidate("topology", "topology", "operations", "invalid")

	if len(invalidator.lenses) != 2 {
		t.Fatalf("expected 2 invalidated lenses, got %d", len(invalidator.lenses))
	}
	if invalidator.lenses[0] != LensTopology || invalidator.lenses[1] != LensOperations {
		t.Fatalf("unexpected invalidated lenses: %#v", invalidator.lenses)
	}

	if len(broadcaster.events) != 1 {
		t.Fatalf("expected 1 broadcast event, got %d", len(broadcaster.events))
	}
	if broadcaster.events[0].Type != WSInvalidate {
		t.Fatalf("expected invalidate websocket event, got %s", broadcaster.events[0].Type)
	}

	payload, ok := broadcaster.events[0].Data.(InvalidationPayload)
	if !ok {
		t.Fatalf("expected invalidation payload, got %T", broadcaster.events[0].Data)
	}
	if len(payload.Lenses) != 2 {
		t.Fatalf("expected 2 payload lenses, got %d", len(payload.Lenses))
	}
}
