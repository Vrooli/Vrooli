package dispatch

import "testing"

// recordingDispatcher is a NodeDispatcher used to assert the interface-tier
// contract: a NodeDispatcher must satisfy Invalidator (the tier below it).
type recordingDispatcher struct {
	invalidated [][]string
	nodeUpdates []nodeUpdate
}

type nodeUpdate struct {
	nodeType string
	nodeID   string
	data     any
}

func (r *recordingDispatcher) DispatchInvalidate(lenses ...string) {
	r.invalidated = append(r.invalidated, lenses)
}

func (r *recordingDispatcher) DispatchNodeUpdate(nodeType, nodeID string, data any) {
	r.nodeUpdates = append(r.nodeUpdates, nodeUpdate{nodeType, nodeID, data})
}

// Compile-time tier guarantees: NodeDispatcher embeds Invalidator, so a value
// satisfying NodeDispatcher is usable everywhere an Invalidator is required.
var (
	_ Invalidator    = (*recordingDispatcher)(nil)
	_ NodeDispatcher = (*recordingDispatcher)(nil)
)

func TestNodeDispatcherIsAnInvalidator(t *testing.T) {
	rec := &recordingDispatcher{}

	// A NodeDispatcher must be assignable to the Invalidator tier.
	var inv Invalidator = rec
	inv.DispatchInvalidate("backlog", "captures")

	if len(rec.invalidated) != 1 || len(rec.invalidated[0]) != 2 {
		t.Fatalf("DispatchInvalidate not recorded as expected: %+v", rec.invalidated)
	}
	if rec.invalidated[0][0] != "backlog" || rec.invalidated[0][1] != "captures" {
		t.Errorf("unexpected lenses: %v", rec.invalidated[0])
	}
}

func TestNodeDispatcherForwardsNodeUpdates(t *testing.T) {
	rec := &recordingDispatcher{}
	var nd NodeDispatcher = rec

	nd.DispatchNodeUpdate("execution", "exec-1", map[string]string{"status": "running"})

	if len(rec.nodeUpdates) != 1 {
		t.Fatalf("expected one node update, got %d", len(rec.nodeUpdates))
	}
	if rec.nodeUpdates[0].nodeType != "execution" || rec.nodeUpdates[0].nodeID != "exec-1" {
		t.Errorf("unexpected node update: %+v", rec.nodeUpdates[0])
	}
}
