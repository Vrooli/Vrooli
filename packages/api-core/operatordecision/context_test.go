package operatordecision

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFactsAreValidWithoutARecommendation(t *testing.T) {
	facts := Facts{Question: "Q", Options: []Option{{Key: "a"}, {Key: "b"}}}
	if err := facts.Validate(); err != nil {
		t.Fatalf("a decision with no honest preference is valid: %v", err)
	}
	facts.Recommended = "c"
	if err := facts.Validate(); err == nil {
		t.Fatal("a recommendation that is not one of the options is still refused")
	}
}

func TestFactsPayloadCarriesTheContextRequestDeclarationOnlyWhenSet(t *testing.T) {
	facts := Facts{
		Question: "Q", Options: []Option{{Key: "a"}, {Key: "b"}}, Recommended: "a",
		Urgency: "normal", SubjectKey: "backup-failure:vol-1",
	}
	if _, ok := facts.Payload()["accepts_context_requests"]; ok {
		t.Fatalf("an undeclared decision must not carry the key (unchanged bytes): %#v", facts.Payload())
	}
	facts.AcceptsContextRequests = true
	if err := facts.Validate(); err != nil {
		t.Fatalf("a keyed, declared decision is valid: %v", err)
	}
	if facts.Payload()["accepts_context_requests"] != true {
		t.Fatalf("declaration not carried: %#v", facts.Payload())
	}
}

func TestContextRequestPayloadRoundTripsThroughTheDecoder(t *testing.T) {
	want := ContextRequest{
		AskID: "ask-1", RequestID: "req-1", Question: "Which volume?", RequestedAt: "2026-11-13T10:00:00Z",
		Source: "data-backup-manager", SourceEventType: "data-backup-manager.volume.decision_requested.v1",
		RequesterRef: "evt-9", SubjectKey: "volume:vol-1", Correlation: map[string]string{"recovery_id": "r1"},
	}
	raw, err := json.Marshal(want.Payload())
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeContextRequest(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.AskID != want.AskID || got.RequestID != want.RequestID || got.Question != want.Question ||
		got.Source != want.Source || got.SourceEventType != want.SourceEventType || got.RequesterRef != want.RequesterRef ||
		got.SubjectKey != want.SubjectKey || got.RequestedAt != want.RequestedAt || got.Correlation["recovery_id"] != "r1" {
		t.Fatalf("round trip lost fields: %+v", got)
	}
}

func TestDecodeContextRequestRefusesAPayloadWithoutAnAskOrAQuestion(t *testing.T) {
	for _, raw := range []string{`{"question":"q"}`, `{"ask_id":"a"}`, `{"ask_id":"a","question":"  "}`} {
		if _, err := DecodeContextRequest([]byte(raw)); !errors.Is(err, ErrContextRequestMalformed) {
			t.Fatalf("%s: err = %v, want ErrContextRequestMalformed", raw, err)
		}
	}
}

type answeringEffect struct {
	recordingEffect
	asked   *ContextRequest
	handle  bool
	respErr error
}

func (e *answeringEffect) RespondToContext(_ context.Context, request ContextRequest) (bool, error) {
	if e.asked != nil {
		*e.asked = request
	}
	return e.handle, e.respErr
}

func postContextRequest(h http.Handler, secret string, payload map[string]any) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]any{
		"event_id": "e1", "event_type": AskContextRequestedEventType, "source_scenario": "notification-hub", "payload": payload,
	})
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set(SignatureHeader, sign(body, secret))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestWebhookDeliversAContextRequestToTheProducerThatAnswersIt(t *testing.T) {
	const kind = "data-backup-manager.volume.decision_requested.v1"
	var asked ContextRequest
	// An effect that owns the kind but cannot answer is skipped, not an error.
	silent := &recordingEffect{owns: kind, handled: true}
	answering := &answeringEffect{recordingEffect: recordingEffect{owns: kind}, asked: &asked, handle: true}
	rec := postContextRequest(legacyWebhook("whsec", silent, answering), "whsec", map[string]any{
		"ask_id": "ask-1", "request_id": "req-1", "question": "Which volume?", "source_event_type": kind, "subject_key": "volume:vol-1",
	})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"handled":true`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if asked.AskID != "ask-1" || asked.Question != "Which volume?" || asked.SubjectKey != "volume:vol-1" {
		t.Fatalf("responder received %+v", asked)
	}
}

func TestWebhookIgnoresAContextRequestNoProducerAnswers(t *testing.T) {
	const kind = "data-backup-manager.volume.decision_requested.v1"
	for name, effect := range map[string]DomainEffect{
		"resolution-only effect":     &recordingEffect{owns: kind, handled: true},
		"responder for another kind": &answeringEffect{recordingEffect: recordingEffect{owns: "other"}, handle: true},
		"responder that declines":    &answeringEffect{recordingEffect: recordingEffect{owns: kind}},
	} {
		rec := postContextRequest(legacyWebhook("whsec", effect), "whsec", map[string]any{"ask_id": "a", "question": "q", "source_event_type": kind})
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ignored":true`) {
			t.Fatalf("%s: status=%d body=%s", name, rec.Code, rec.Body.String())
		}
	}
}

func TestWebhookContextRequestErrorsRetryAndMalformedIsBad(t *testing.T) {
	const kind = "k.decision_requested.v1"
	failing := &answeringEffect{recordingEffect: recordingEffect{owns: kind}, respErr: errors.New("store down")}
	if rec := postContextRequest(legacyWebhook("whsec", failing), "whsec", map[string]any{"ask_id": "a", "question": "q", "source_event_type": kind}); rec.Code != http.StatusInternalServerError {
		t.Fatalf("a responder error must be retried by vrooli-events: %d", rec.Code)
	}
	if rec := postContextRequest(legacyWebhook("whsec", failing), "whsec", map[string]any{"ask_id": "a"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("a malformed request is a 400: %d", rec.Code)
	}
}
