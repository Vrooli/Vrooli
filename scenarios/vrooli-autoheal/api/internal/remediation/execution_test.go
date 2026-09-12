package remediation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"google.golang.org/protobuf/proto"
)

func TestNotificationHubAskVerifierAcceptsOnlyDurableApproval(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vrooli.notification_hub.v1.conversations.ConversationsService/Wait" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/proto")
		body, marshalErr := proto.Marshal(&conversationv1.WaitResponse{State: "answered", Answer: "approve"})
		if marshalErr != nil {
			t.Fatalf("marshal response: %v", marshalErr)
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	verifier, err := NewNotificationHubAskVerifier(server.URL)
	if err != nil {
		t.Fatalf("NewNotificationHubAskVerifier() error = %v", err)
	}
	approval, err := verifier.Verify(context.Background(), "ask-1")
	if err != nil || !ApprovedAsk(approval) || approval.AskID != "ask-1" {
		t.Fatalf("approval = %+v error = %v", approval, err)
	}
}

type recordingRunner struct {
	preflight string
	run       string
}

func (r *recordingRunner) Preflight(_ context.Context, path string) error {
	r.preflight = path
	return nil
}

func (r *recordingRunner) Run(_ context.Context, path string) (int, string, error) {
	r.run = path
	return 0, "fixture executed", nil
}

func executionFixture() incidents.Incident {
	return incidents.Incident{
		ID: "inc-1", Fingerprint: "fp-1", SourceCheckIDs: []string{"host-kernel-module-drift"},
		RemediationCandidates: []incidents.RemediationCandidate{{ID: "candidate-1", Applicability: "applicable"}},
		RemediationArtifacts:  []incidents.RemediationArtifact{{RemediationID: "candidate-1", Path: "/tmp/remediation"}},
	}
}

func executionAuth(approved, autoHeal bool) Authorisation {
	return Authorisation{AskID: "ask-1", IncidentID: "inc-1", IncidentFingerprint: "fp-1", CandidateID: "candidate-1", Approved: approved, AutoHealEnabled: autoHeal}
}

func TestExecuteRefusesWhenAutoHealDisabled(t *testing.T) {
	service := &Service{}
	_, err := service.Execute(context.Background(), executionFixture(), "candidate-1", executionAuth(true, false))
	if !errors.Is(err, ErrAutoHealDisabled) {
		t.Fatalf("error = %v, want ErrAutoHealDisabled", err)
	}
}

func TestExecuteRefusesWhenAskIsNotApproved(t *testing.T) {
	service := &Service{}
	_, err := service.Execute(context.Background(), executionFixture(), "candidate-1", executionAuth(false, true))
	if !errors.Is(err, ErrAskNotApproved) {
		t.Fatalf("error = %v, want ErrAskNotApproved", err)
	}
}

func TestExecuteRunsAfterBothGatesAndPreflight(t *testing.T) {
	runner := &recordingRunner{}
	service := &Service{runner: runner}
	result, err := service.Execute(context.Background(), executionFixture(), "candidate-1", executionAuth(true, true))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || runner.preflight == "" || runner.run == "" || result.Output != "fixture executed" {
		t.Fatalf("result = %+v runner = %+v", result, runner)
	}
}
