package journeytest

import (
	"encoding/json"
	"os"
	"sync"
	"testing"

	"landing-page-business-suite-api/internal/aidoor"
)

// Replay observations from the real mounted Door and crash-recovery subprocess.
// The oracle is the consumed declarative contract, not another fake Door.
type requestTrace struct {
	mu      sync.Mutex
	events  []aidoor.RequestTransition
	observe func(aidoor.RequestTransition)
}

func newRequestTrace(t *testing.T) *requestTrace {
	t.Helper()
	data, err := os.ReadFile("../internal/aidoor/request.spec.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Transitions []struct{ From, Event, To string }
	}
	if err = json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	pairs := map[[2]string]string{}
	for _, row := range spec.Transitions {
		pairs[[2]string{row.From, row.Event}] = row.To
	}
	trace := &requestTrace{}
	trace.observe = func(row aidoor.RequestTransition) {
		want, valid := pairs[[2]string{row.From, row.Event}]
		if !valid || row.Rejected || want != row.To {
			t.Errorf("real Door violated request spec: %+v want %q", row, want)
		}
		trace.mu.Lock()
		trace.events = append(trace.events, row)
		trace.mu.Unlock()
		t.Logf("door trace %s: %s --%s--> %s", row.OperationID, row.From, row.Event, row.To)
	}
	return trace
}

func (trace *requestTrace) require(t *testing.T, from string, events ...string) {
	t.Helper()
	trace.mu.Lock()
	defer trace.mu.Unlock()
	for start, row := range trace.events {
		if row.From != from {
			continue
		}
		at := 0
		state := from
		for _, next := range trace.events[start:] {
			if next.OperationID != row.OperationID {
				continue
			}
			if next.From != state || next.Event != events[at] {
				break
			}
			state = next.To
			at++
			if at == len(events) {
				return
			}
		}
	}
	t.Fatalf("real Door trace lacks %s %v; got %+v", from, events, trace.events)
}
