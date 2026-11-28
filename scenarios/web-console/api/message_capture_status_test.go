package main

import (
	"os"
	"path/filepath"
	"testing"

	"web-console/internal/sessionstore"

	conversationH "web-console/handlers/conversation"
)

// The behavior under test is the distinction that did not exist before: an
// empty conversation must be able to say whether it is empty because nothing
// has been said, or because nothing can be read.

func TestDiagnoseNoAgentIsNotAFault(t *testing.T) {
	a := &conversationAdapter{}
	got := a.diagnose(sessionstore.Metadata{ID: "s1", AgentType: sessionstore.AgentNone})
	if got.State != conversationH.CaptureNotApplicable {
		t.Fatalf("state = %q, want %q", got.State, conversationH.CaptureNotApplicable)
	}
	if got.Remediation != "" {
		t.Errorf("a plain terminal must not ask the operator to fix anything, got %q", got.Remediation)
	}
}

func TestDiagnoseSelfIdentifyingAgentAwaitsFirstTurn(t *testing.T) {
	for _, agent := range []sessionstore.Agent{sessionstore.AgentCodex, sessionstore.AgentGrok, sessionstore.AgentOpenCode} {
		got := diagnoseSelfIdentifyingAgent(sessionstore.Metadata{ID: "s1", AgentType: agent})
		// These agents recover identity from their own transcripts, so a
		// missing id only ever means "nothing written yet".
		if got.State != conversationH.CapturePending {
			t.Errorf("%s: state = %q, want %q", agent, got.State, conversationH.CapturePending)
		}
		if got.ReasonCode != captureReasonAwaitingFirstTurn {
			t.Errorf("%s: reason = %q, want %q", agent, got.ReasonCode, captureReasonAwaitingFirstTurn)
		}
	}
}

func TestTranscriptFileStatusReportsMissingHistoryAsAFault(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone.jsonl")
	got := transcriptFileStatus(missing)
	if got.State != conversationH.CaptureUnavailable {
		t.Fatalf("state = %q, want %q", got.State, conversationH.CaptureUnavailable)
	}
	if got.ReasonCode != captureReasonTranscriptMissing {
		t.Errorf("reason = %q, want %q", got.ReasonCode, captureReasonTranscriptMissing)
	}
	if got.TranscriptPath != missing {
		t.Errorf("the path an operator needs must survive into the status, got %q", got.TranscriptPath)
	}
}

func TestTranscriptFileStatusReportsPresentTranscriptAsCapturing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "present.jsonl")
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	got := transcriptFileStatus(path)
	if got.State != conversationH.CaptureCapturing {
		t.Fatalf("state = %q, want %q", got.State, conversationH.CaptureCapturing)
	}
	if got.ReasonCode != "" {
		t.Errorf("a healthy status carries no cause, got %q", got.ReasonCode)
	}
}
