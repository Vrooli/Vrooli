package searcheval

import (
	"strings"
	"testing"
)

func TestAgentManagerReplayRequestIDBindsSourceAndRouteScope(t *testing.T) {
	got, err := AgentManagerReplayRequestID("run-1", "event-2", 0, "classify.fast", "agent-manager")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "agent-manager-replay:v1:run-1:event-2:0:") || len(strings.TrimPrefix(got, "agent-manager-replay:v1:run-1:event-2:0:")) != 64 {
		t.Fatalf("request ID = %q, want source identity and full scope digest", got)
	}
	otherRole, err := AgentManagerReplayRequestID("run-1", "event-2", 0, "chat.default", "agent-manager")
	if err != nil {
		t.Fatal(err)
	}
	if got == otherRole {
		t.Fatal("different task roles must not share a route request identity")
	}
	otherConsumer, err := AgentManagerReplayRequestID("run-1", "event-2", 0, "classify.fast", "search-hub")
	if err != nil {
		t.Fatal(err)
	}
	if got == otherConsumer {
		t.Fatal("different consumers must not share a route request identity")
	}
}

func TestAgentManagerReplayRequestIDRequiresCompleteIdentity(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		runID, eventID, role, consumer string
		sequence                       int64
	}{
		{name: "missing run", eventID: "event", role: "chat.default", consumer: "agent-manager"},
		{name: "missing event", runID: "run", role: "chat.default", consumer: "agent-manager"},
		{name: "negative sequence", runID: "run", eventID: "event", sequence: -1, role: "chat.default", consumer: "agent-manager"},
		{name: "missing role", runID: "run", eventID: "event", consumer: "agent-manager"},
		{name: "missing consumer", runID: "run", eventID: "event", role: "chat.default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := AgentManagerReplayRequestID(tc.runID, tc.eventID, tc.sequence, tc.role, tc.consumer); err == nil {
				t.Fatal("incomplete identity accepted")
			}
		})
	}
}
