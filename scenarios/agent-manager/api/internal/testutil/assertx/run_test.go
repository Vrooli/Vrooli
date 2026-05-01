package assertx

import (
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestEventMessageContainsAcceptsTypedMessages(t *testing.T) {
	runID := uuid.New()
	events := []*domain.RunEvent{
		domain.NewLogEvent(runID, "info", "runner started"),
		domain.NewMessageEvent(runID, "assistant", "final summary"),
	}

	EventMessageContains(t, events, "final")
}
