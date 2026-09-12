// Package assertx contains domain-aware test assertions.
package assertx

import (
	"strings"
	"testing"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
)

// RunStatus fails if got.Status does not match want.
func RunStatus(t *testing.T, got *domain.Run, want domain.RunStatus) {
	t.Helper()
	if got == nil {
		t.Fatalf("RunStatus: run is nil, want %s", want)
	}
	if got.Status != want {
		t.Errorf("RunStatus: got %s, want %s (run %s)", got.Status, want, got.ID)
	}
}

// RunPhase fails if got.Phase does not match want.
func RunPhase(t *testing.T, got *domain.Run, want domain.RunPhase) {
	t.Helper()
	if got == nil {
		t.Fatalf("RunPhase: run is nil, want %s", want)
	}
	if got.Phase != want {
		t.Errorf("RunPhase: got %s, want %s (run %s)", got.Phase, want, got.ID)
	}
}

// EventMessageContains fails if no event contains substring in a log, message,
// status reason, or error message payload.
func EventMessageContains(t *testing.T, events []*domain.RunEvent, substring string) {
	t.Helper()
	for _, evt := range events {
		if eventMessageContains(evt, substring) {
			return
		}
	}
	t.Errorf("EventMessageContains: no event message contained %q across %d events", substring, len(events))
}

// SandboxApplyRequest fails if the apply request does not target the expected
// sandbox and conversation.
func SandboxApplyRequest(t *testing.T, got sandbox.ApplyAtRunEndRequest, want sandbox.ApplyAtRunEndRequest) {
	t.Helper()
	if got.SandboxID != want.SandboxID {
		t.Errorf("SandboxApplyRequest: SandboxID got %s, want %s", got.SandboxID, want.SandboxID)
	}
	if got.RunID != want.RunID {
		t.Errorf("SandboxApplyRequest: RunID got %s, want %s", got.RunID, want.RunID)
	}
	if got.ConversationID != want.ConversationID {
		t.Errorf("SandboxApplyRequest: ConversationID got %q, want %q", got.ConversationID, want.ConversationID)
	}
	if got.Actor != want.Actor {
		t.Errorf("SandboxApplyRequest: Actor got %q, want %q", got.Actor, want.Actor)
	}
}

func eventMessageContains(evt *domain.RunEvent, substring string) bool {
	if evt == nil || evt.Data == nil {
		return false
	}
	switch data := evt.Data.(type) {
	case *domain.LogEventData:
		return strings.Contains(data.Message, substring)
	case *domain.MessageEventData:
		return strings.Contains(data.Content, substring)
	case *domain.StatusEventData:
		return strings.Contains(data.Reason, substring)
	case *domain.ErrorEventData:
		return strings.Contains(data.Message, substring)
	default:
		return false
	}
}
