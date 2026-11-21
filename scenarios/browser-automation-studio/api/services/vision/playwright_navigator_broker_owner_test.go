package vision

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestPlaywrightNavigatorBroker_PreservesSessionRequestAndActionEvidence(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	broker := &mockHTTPDoer{response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{"navigation_id":"nav_broker_owner","status":"navigating"}`)),
	}}
	var recordedSession string
	var recorded *RecordedNavigationAction
	nav := NewPlaywrightVisionNavigator(log,
		WithPlaywrightSessionBroker(broker),
		WithActionRecordCallback(func(sessionID string, action *RecordedNavigationAction) {
			recordedSession, recorded = sessionID, action
		}),
	)
	ctx := context.WithValue(context.Background(), struct{ name string }{"trace"}, "caller")
	handle, err := nav.Navigate(ctx, NavigationRequest{
		SessionID:           "session-owned-by-bas",
		Prompt:              "inspect the page",
		Model:               "local_first",
		RootOperationID:     "a14d8e7c0c014dbe8ad773ae25c8c781",
		AccessToken:         "signed-session-token",
		AISource:            "vrooli-credits",
		PrivacyConsentGrant: "opaque-consent-receipt",
	})
	if err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	if handle.SessionID() != "session-owned-by-bas" || handle.ID() != "nav_broker_owner" {
		t.Fatalf("handle identity = (%q, %q)", handle.SessionID(), handle.ID())
	}
	request := broker.lastRequest
	if request == nil || request.Context() != ctx || request.URL.Path != "/session/session-owned-by-bas/ai-navigate" {
		t.Fatalf("broker request lost caller context or exact session target: %#v", request)
	}
	if got := request.Header.Get("Authorization"); got != "Bearer signed-session-token" {
		t.Fatalf("authorization = %q", got)
	}
	var body map[string]any
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		t.Fatalf("decode broker request: %v", err)
	}
	if body["root_operation_id"] != "a14d8e7c0c014dbe8ad773ae25c8c781" || body["privacy_consent_grant"] != "opaque-consent-receipt" {
		t.Fatalf("operation/consent evidence lost: %#v", body)
	}
	if err := nav.HandleStepCallback(ctx, &NavigationStep{
		NavigationID: "nav_broker_owner", StepNumber: 1,
		Action:     map[string]any{"type": "click", "selector": "button.submit"},
		CurrentURL: "https://fixture.test/form",
	}); err != nil {
		t.Fatalf("step callback: %v", err)
	}
	if recordedSession != "session-owned-by-bas" || recorded == nil || recorded.ActionType != "click" || recorded.Selector != "button.submit" {
		t.Fatalf("recording evidence = (%q, %#v)", recordedSession, recorded)
	}
	snapshot, ok := nav.GetSession("nav_broker_owner")
	if !ok || snapshot.SessionID != "session-owned-by-bas" || snapshot.StepCount != 1 {
		t.Fatalf("navigation evidence missing after brokered callback: %#v, present=%v", snapshot, ok)
	}
}

func TestPlaywrightNavigatorBroker_CanceledRouteCleansNavigationState(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	broker := &mockHTTPDoer{err: context.Canceled}
	nav := NewPlaywrightVisionNavigator(log, WithPlaywrightSessionBroker(broker))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := nav.Navigate(ctx, NavigationRequest{SessionID: "session-cancel", Prompt: "inspect", Model: "local_first"})
	if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("Navigate error = %v, want cancellation", err)
	}
	if broker.lastRequest == nil || broker.lastRequest.Context() != ctx || broker.lastRequest.URL.Path != "/session/session-cancel/ai-navigate" {
		t.Fatalf("canceled route did not retain target/context: %#v", broker.lastRequest)
	}
	if len(nav.activeNavigations) != 0 {
		t.Fatalf("canceled route leaked %d active navigation entries", len(nav.activeNavigations))
	}
}
