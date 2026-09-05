package assertx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/adapters/sandbox"
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

func TestDomainAssertionsAcceptMatchingContractsAndAllTypedEvidence(t *testing.T) {
	runID := uuid.New()
	run := &domain.Run{ID: runID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting}
	RunStatus(t, run, domain.RunStatusRunning)
	RunPhase(t, run, domain.RunPhaseExecuting)

	events := []*domain.RunEvent{
		{Data: &domain.LogEventData{Message: "log evidence"}},
		{Data: &domain.MessageEventData{Content: "message evidence"}},
		{Data: &domain.StatusEventData{Reason: "status evidence"}},
		{Data: &domain.ErrorEventData{Message: "error evidence"}},
		{Data: &domain.MetricEventData{}},
		nil,
	}
	for _, needle := range []string{"log", "message", "status", "error"} {
		EventMessageContains(t, events, needle)
	}
	if eventMessageContains(events[4], "anything") || eventMessageContains(nil, "anything") {
		t.Fatal("non-message event payload matched")
	}

	want := sandbox.ApplyAtRunEndRequest{SandboxID: uuid.New(), RunID: runID.String(), ConversationID: "conversation", Actor: "agent"}
	SandboxApplyRequest(t, want, want)
	recorder := httptest.NewRecorder()
	recorder.WriteHeader(http.StatusNoContent)
	HTTPStatus(t, recorder, http.StatusNoContent)
}
