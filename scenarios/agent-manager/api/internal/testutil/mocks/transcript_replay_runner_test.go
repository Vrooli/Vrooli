package mocks

import (
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestTranscriptReplayRunnerParsesSessionMessageAndTerminal(t *testing.T) {
	runID := uuid.New()
	runner := NewTranscriptReplayRunner(domain.RunnerTypeCodex)

	session := runner.ParseTranscriptLine(runID, "session:thread-123")
	if session.SessionID != "thread-123" {
		t.Fatalf("session id = %q, want thread-123", session.SessionID)
	}

	message := runner.ParseTranscriptLine(runID, "message:hello")
	if len(message.Events) != 1 {
		t.Fatalf("message event = %+v, want one assistant hello event", message.Events)
	}
	messageData, ok := message.Events[0].Data.(*domain.MessageEventData)
	if !ok || messageData.Role != "assistant" || messageData.Content != "hello" {
		t.Fatalf("message event = %+v, want assistant hello", message.Events)
	}

	terminal := runner.ParseTranscriptLine(runID, "done:complete")
	if terminal.Terminal == nil || !terminal.Terminal.Success || terminal.Terminal.Summary.Description != "complete" {
		t.Fatalf("terminal = %+v, want successful complete summary", terminal.Terminal)
	}
}

func TestTranscriptReplayRunnerParsesFailureAndLogFallback(t *testing.T) {
	runID := uuid.New()
	runner := NewTranscriptReplayRunner(domain.RunnerTypeCodex)

	failure := runner.ParseTranscriptLine(runID, "fail:runner crashed")
	if failure.Terminal == nil || failure.Terminal.Success || failure.Terminal.ErrorMessage != "runner crashed" {
		t.Fatalf("failure terminal = %+v, want runner crashed failure", failure.Terminal)
	}

	log := runner.ParseTranscriptLine(runID, "plain output")
	if len(log.Events) != 1 {
		t.Fatalf("log event = %+v, want one info plain output event", log.Events)
	}
	logData, ok := log.Events[0].Data.(*domain.LogEventData)
	if !ok || logData.Level != "info" || logData.Message != "plain output" {
		t.Fatalf("log event = %+v, want info plain output", log.Events)
	}
}
