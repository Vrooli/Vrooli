package courier

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/operatordecision"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	convv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"

	"research-desk/internal/home"
)

// eventsServer stands in for Vrooli Events and keeps what was published.
type eventsServer struct {
	mu     sync.Mutex
	events []*domain.EventEnvelope
	hub    *EventHub
}

func newEventsServer(t *testing.T) *eventsServer {
	t.Helper()
	s := &eventsServer{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var envelope domain.EventEnvelope
		if err := protojson.Unmarshal(body, &envelope); err != nil {
			t.Errorf("an event the Hub could not read: %v", err)
		}
		s.mu.Lock()
		s.events = append(s.events, &envelope)
		s.mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)
	s.hub = &EventHub{Events: func(context.Context) (string, error) { return srv.URL, nil }}
	return s
}

func (s *eventsServer) payload(t *testing.T, i int) map[string]any {
	t.Helper()
	var payload structpb.Struct
	if err := s.events[i].GetData().UnmarshalTo(&payload); err != nil {
		t.Fatal(err)
	}
	return payload.AsMap()
}

var testSpec = AskSpec{
	Key:              AskPrefix + "prog-1/AV4",
	Subject:          "Keep the video tool?",
	Question:         "Keep the video tool?",
	TechnicalDetails: "Desk key: " + AskPrefix + "prog-1/AV4",
	Options:          []home.Option{{Key: "keep", Label: "Keep"}, {Key: "drop", Label: "Drop"}},
	Recommended:      "keep",
	Reason:           "Cheapest.",
	Explainer:        "<!doctype html><h1>Keep the video tool?</h1>",
}

// An ask carries the item's key as its subject, so the Hub keeps one open
// decision per item; its request id is a function of the content, so the
// courier knows whether an open ask is current.
func TestAnAskCarriesTheItemAsItsSubject(t *testing.T) {
	s := newEventsServer(t)
	if err := s.hub.Ask(context.Background(), testSpec); err != nil {
		t.Fatal(err)
	}
	if got := s.events[0].GetEventType(); got != DecisionEvent {
		t.Fatalf("event type %q", got)
	}
	if got := s.payload(t, 0)["subject_key"]; got != testSpec.Key {
		t.Fatalf("subject_key = %v, want the item key", got)
	}
	if p := s.payload(t, 0); p["subject"] != testSpec.Subject || p["technical_details"] != testSpec.TechnicalDetails {
		t.Fatalf("the Hub title names the decision and the technical details carry the desk key: %v", p)
	}
	if got := s.payload(t, 0)["explainer_html"]; got != testSpec.Explainer {
		t.Fatalf("the decision's page rides inline in the event, where the Hub stores it as the explainer: %v", got)
	}
	if s.hub.RequestRef(testSpec) != s.events[0].GetEventId() {
		t.Fatalf("RequestRef %q is not the id of the event sent, %q", s.hub.RequestRef(testSpec), s.events[0].GetEventId())
	}
	changed := testSpec
	changed.Question = "Keep the video tool for now?"
	if s.hub.RequestRef(changed) == s.hub.RequestRef(testSpec) || s.hub.RequestRef(testSpec) != s.hub.RequestRef(testSpec) {
		t.Fatal("a changed question is another request; an unchanged one is the same")
	}
	if s.hub.RequestRef(AskSpec{Key: testSpec.Key, Question: "no options"}) != "" {
		t.Fatal("an item that cannot be a Hub decision has no request id")
	}
}

func TestSubjectKeysFollowTheHubRule(t *testing.T) {
	long := AskPrefix + strings.Repeat("c", 90) + "/" + strings.Repeat("I", 90)
	for _, key := range []string{testSpec.Key, TestPrefix + "20261112T010000Z", AskPrefix + "prog 1/A#4", long, AskPrefix + "prog/" + strings.Repeat("é", 80)} {
		got := subjectKey(key)
		if !operatordecision.SubjectKeyRule.MatchString(got) {
			t.Errorf("subjectKey(%q) = %q breaks the Hub's rule", key, got)
		}
	}
	if subjectKey(testSpec.Key) != testSpec.Key {
		t.Error("a valid key is used as it is")
	}
	if subjectKey(AskPrefix+"prog 1/A#4") == subjectKey(AskPrefix+"prog_1/A_4") {
		t.Error("two different keys never share a subject")
	}
}

// A withdrawal closes the ask by the id of the request that opened it, with a
// reason in plain words, and is a distinct event on each pass.
func TestWithdrawalClosesTheAskByItsRequestID(t *testing.T) {
	s := newEventsServer(t)
	ask := Ask{ID: "ask-1", Key: testSpec.Key, State: "pending", RequesterRef: "request-event-id"}
	at := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		if err := s.hub.Withdraw(context.Background(), ask, withdrawAnswered, at.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	if got := s.events[0].GetEventType(); got != "research_desk.needs_you.decision_withdrawn.v1" {
		t.Fatalf("event type %q", got)
	}
	payload := s.payload(t, 0)
	if payload["requester_ref"] != "request-event-id" || payload["reason"] != withdrawAnswered || payload["source_event_type"] != DecisionEvent {
		t.Fatalf("payload %v", payload)
	}
	if s.events[0].GetEventId() == s.events[1].GetEventId() {
		t.Fatal("a withdrawal lost before the Hub subscribed must be sent again, not deduplicated")
	}
	if err := s.hub.Withdraw(context.Background(), Ask{ID: "ask-2"}, withdrawAnswered, at); err == nil {
		t.Fatal("an ask the Hub gave no request id for cannot be closed, and says so")
	}
	if len(s.events) != 2 {
		t.Fatalf("nothing is sent for the ask that cannot be closed: %d events", len(s.events))
	}
}

// The Hub adds its own free-form "other" answer to an ask; reading an ask back
// must not count it, or an unchanged item would look changed on every pass.
func TestAnAskReadBackSaysWhatTheDeskSent(t *testing.T) {
	spec := testSpec
	spec.BigPicture = "Why it matters."
	a := askFromProto(&convv1.Ask{
		Id: "a1", Question: spec.Question, BigPicture: spec.BigPicture, TechnicalDetails: spec.TechnicalDetails, Recommended: "keep", State: "pending", RequesterRef: "r1",
		Options: []*convv1.AskOption{{Key: "keep", Label: "Keep"}, {Key: "drop", Label: "Drop"}, {Key: "other", Label: "Other"}},
	})
	if a.Key != spec.Key || a.RequesterRef != "r1" || !a.says(spec) {
		t.Fatalf("an ask as sent reads back as current: %+v", a)
	}
	changed := spec
	changed.Options = spec.Options[:1]
	if a.says(changed) {
		t.Fatal("a changed answer menu is not current")
	}
}

// The Hub reserves "other" for the operator's own words; an item that lists it
// must still reach the operator (2026-11-15..20: five asks refused, none delivered).
func TestAnItemOfferingOtherStillBecomesAnAsk(t *testing.T) {
	s := newEventsServer(t)
	spec := testSpec
	spec.Options = append(append([]home.Option{}, testSpec.Options...), home.Option{Key: "other", Label: "Something else"})
	if err := s.hub.Ask(context.Background(), spec); err != nil {
		t.Fatalf("an ask offering other was refused: %v", err)
	}
	if s.hub.RequestRef(spec) == "" {
		t.Fatal("an item offering other must still be a Hub decision")
	}
}
