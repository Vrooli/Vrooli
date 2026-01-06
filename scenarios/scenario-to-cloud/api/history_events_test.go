package main

import (
	"context"
	"strings"
	"testing"

	"scenario-to-cloud/domain"
)

type fakeHistoryRecorder struct {
	events []domain.HistoryEvent
}

func (f *fakeHistoryRecorder) AppendHistoryEvent(_ context.Context, _ string, event domain.HistoryEvent) error {
	f.events = append(f.events, event)
	return nil
}

func TestAppendHistoryEventSetsTimestamp(t *testing.T) {
	recorder := &fakeHistoryRecorder{}
	srv := &Server{historyRecorder: recorder}

	srv.appendHistoryEvent(context.Background(), "deployment-1", domain.HistoryEvent{
		Type: domain.EventDeployStarted,
	})

	if len(recorder.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(recorder.events))
	}
	if recorder.events[0].Timestamp.IsZero() {
		t.Fatalf("expected timestamp to be set")
	}
}

func TestFormatPreflightFailureDetails(t *testing.T) {
	resp := PreflightResponse{
		Checks: []PreflightCheck{
			{
				ID:      "ssh_connect",
				Title:   "SSH connectivity",
				Status:  PreflightFail,
				Details: "Unable to run a remote command over SSH",
				Hint:    "Confirm SSH key auth works",
			},
			{
				ID:      "firewall_inbound",
				Title:   "Inbound firewall rules",
				Status:  PreflightWarn,
				Details: "UFW is inactive",
			},
		},
	}

	details := formatPreflightFailureDetails(resp)
	if !strings.Contains(details, "SSH connectivity") {
		t.Fatalf("expected failing check title in details, got: %s", details)
	}
	if !strings.Contains(details, "Confirm SSH key auth works") {
		t.Fatalf("expected failing check hint in details, got: %s", details)
	}
	if strings.Contains(details, "Inbound firewall rules") {
		t.Fatalf("did not expect warning check in details, got: %s", details)
	}
}
