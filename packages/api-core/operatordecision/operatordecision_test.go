package operatordecision

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
)

// --- EventID determinism ----------------------------------------------------

func TestEventIDDeterministicAndContentSensitive(t *testing.T) {
	facts := Facts{
		Subject: "Run abc needs review", Question: "Apply changes?",
		Options:     []Option{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}},
		Recommended: "approve", Reversible: true, Urgency: "normal",
		Correlation: map[string]string{"run_id": "abc"},
	}
	occurred := time.Date(2026, 10, 16, 1, 2, 3, 0, time.UTC)
	a := BuildEvent("agent-manager", "run", occurred, facts)
	b := BuildEvent("agent-manager", "run", occurred, facts)
	if EventID(a) != EventID(b) {
		t.Fatalf("same event must produce the same id: %s != %s", EventID(a), EventID(b))
	}
	// A different occurrence changes the id.
	c := BuildEvent("agent-manager", "run", occurred.Add(time.Second), facts)
	if EventID(a) == EventID(c) {
		t.Fatalf("a different occurrence must change the id")
	}
	// Matches the eventbus.Client id (SHA-256 of JSON) byte-for-byte.
	seed, _ := json.Marshal(a)
	sum := sha256.Sum256(seed)
	if EventID(a) != hex.EncodeToString(sum[:]) {
		t.Fatalf("EventID must equal eventbus client's SHA-256-of-JSON id")
	}
}

func TestDecisionRequestedEventType(t *testing.T) {
	if got := DecisionRequestedEventType("deployment-manager", "release"); got != "deployment-manager.release.decision_requested.v1" {
		t.Fatalf("unexpected event type: %s", got)
	}
	if got := BuildEvent("agent-manager", "run", time.Now(), Facts{}).EventType; got != "agent-manager.run.decision_requested.v1" {
		t.Fatalf("BuildEvent event type: %s", got)
	}
}

// --- CorrelationID stability ------------------------------------------------

func TestCorrelationIDStability(t *testing.T) {
	a := CorrelationID("auth", "subject-1", "run.create", "target-x")
	b := CorrelationID("auth", " subject-1 ", "run.create", " target-x ") // trims
	if a != b {
		t.Fatalf("CorrelationID must be stable under whitespace: %s != %s", a, b)
	}
	if !strings.HasPrefix(a, "auth-") {
		t.Fatalf("expected prefix auth-, got %s", a)
	}
	if CorrelationID("auth", "subject-1", "run.create", "target-y") == a {
		t.Fatalf("different parts must produce a different id")
	}
	// An empty prefix yields the bare 32-char hex id — no separator prepended.
	p := CorrelationID("", "x")
	if len(p) != 32 {
		t.Fatalf("empty-prefix id must be 32 hex chars, got %d (%q)", len(p), p)
	}
	if strings.Contains(p, "-") {
		t.Fatalf("empty prefix must not prepend a separator: %q", p)
	}
}

// --- Facts.Validate ---------------------------------------------------------

func TestFactsValidate(t *testing.T) {
	base := Facts{
		Question:    "Apply changes?",
		Options:     []Option{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}},
		Recommended: "approve", Urgency: "normal",
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid facts rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*Facts)
		want   string
	}{
		{"too few options", func(f *Facts) { f.Options = f.Options[:1] }, "needs 2-"},
		{"missing recommended", func(f *Facts) { f.Recommended = "" }, "recommended"},
		{"recommended not an option", func(f *Facts) { f.Recommended = "maybe" }, "recommended"},
		{"default on non-reversible", func(f *Facts) { f.DefaultAnswer = "approve"; f.Reversible = false }, "reversible"},
		{"default not an option", func(f *Facts) { f.DefaultAnswer = "maybe"; f.Reversible = true }, "default"},
		{"empty question", func(f *Facts) { f.Question = "" }, "question"},
		{"bad urgency", func(f *Facts) { f.Urgency = "immediate" }, "urgency"},
		{"duplicate option key", func(f *Facts) {
			f.Options = []Option{{Key: "approve"}, {Key: "approve"}}
		}, "repeated"},
		{"bad option key", func(f *Facts) {
			f.Options = []Option{{Key: "approve"}, {Key: "no good key!"}}
		}, "option key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := base
			f.Options = append([]Option(nil), base.Options...)
			tc.mutate(&f)
			err := f.Validate()
			if err == nil {
				t.Fatalf("expected rejection for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q should mention %q", err.Error(), tc.want)
			}
		})
	}

	// A default IS allowed on a reversible decision.
	ok := base
	ok.Reversible, ok.DefaultAnswer = true, "approve"
	if err := ok.Validate(); err != nil {
		t.Fatalf("reversible default rejected: %v", err)
	}
}

// TestFactsValidateContextURL pins the context_url scheme allow-list: a
// producer-supplied link becomes an operator-clickable href, so only empty,
// http(s), and root-relative values pass; javascript:/data:/etc. are refused.
func TestFactsValidateContextURL(t *testing.T) {
	base := Facts{
		Question:    "Apply?",
		Options:     []Option{{Key: "approve"}, {Key: "reject"}},
		Recommended: "approve", Urgency: "normal",
	}
	good := []string{"", "https://console.example.test/?run=abc", "http://localhost:16382/", "/?run=abc", "/approvals"}
	for _, u := range good {
		f := base
		f.ContextURL = u
		if err := f.Validate(); err != nil {
			t.Fatalf("context_url %q should be allowed: %v", u, err)
		}
	}
	bad := []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)", // scheme match is case-insensitive
		"data:text/html,<script>alert(1)</script>",
		"vbscript:msgbox(1)",
		"file:///etc/passwd",
		"//evil.example.test/phish", // scheme-relative inherits page scheme off-origin
	}
	for _, u := range bad {
		f := base
		f.ContextURL = u
		if err := f.Validate(); err == nil {
			t.Fatalf("context_url %q must be rejected", u)
		}
	}
}

func TestFactsPayload(t *testing.T) {
	f := Facts{
		Subject: "S", Question: "Q",
		Options:     []Option{{Key: "a", Label: "Approve"}, {Key: "b"}},
		Recommended: "a", Reversible: true, Urgency: "high",
		Correlation: map[string]string{"run_id": "r1"},
		Deadline:    time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	p := f.Payload()
	if p["subject"] != "S" || p["question"] != "Q" || p["recommended"] != "a" || p["reversible"] != true {
		t.Fatalf("unexpected payload scalars: %#v", p)
	}
	opts := p["options"].([]any)
	if len(opts) != 2 || opts[1].(map[string]any)["label"] != "b" { // label falls back to key
		t.Fatalf("options should carry key as label fallback: %#v", opts)
	}
	if p["correlation"].(map[string]any)["run_id"] != "r1" {
		t.Fatalf("correlation not carried verbatim: %#v", p["correlation"])
	}
	if p["deadline"] != "2026-01-02T03:04:05Z" {
		t.Fatalf("deadline not RFC3339: %v", p["deadline"])
	}
	// Empty recommendation_reason/default_answer/note are omitted (parity: a run
	// approval must not emit a silent default_answer key).
	if _, ok := p["recommendation_reason"]; ok {
		t.Fatalf("empty recommendation_reason must be omitted, got %#v", p["recommendation_reason"])
	}
	if _, ok := p["default_answer"]; ok {
		t.Fatalf("empty default_answer must be omitted, got %#v", p["default_answer"])
	}
	if _, ok := p["note"]; ok {
		t.Fatalf("empty note must be omitted")
	}

	// When set, they ARE carried verbatim.
	f.RecommendationReason, f.DefaultAnswer, f.Note = "because", "a", "n"
	f.BigPicture, f.TechnicalDetails = "purpose", "evidence"
	p = f.Payload()
	if p["recommendation_reason"] != "because" || p["default_answer"] != "a" || p["note"] != "n" || p["big_picture"] != "purpose" || p["technical_details"] != "evidence" {
		t.Fatalf("set reason/default/note must be carried: %#v", p)
	}
}

func TestFactsValidateExplanationBoundsUseUnicodeCodePoints(t *testing.T) {
	base := Facts{
		Question: "Apply?", Options: []Option{{Key: "yes"}, {Key: "no"}},
		Recommended: "yes", Urgency: "normal",
	}
	base.BigPicture = strings.Repeat("😀", MaxBigPictureCodePoints)
	base.TechnicalDetails = strings.Repeat("🧭", MaxTechnicalDetailsCodePoints)
	if err := base.Validate(); err != nil {
		t.Fatalf("valid code-point limits rejected: %v", err)
	}
	base.BigPicture += "x"
	if err := base.Validate(); err == nil || !strings.Contains(err.Error(), "big_picture") {
		t.Fatalf("oversize big_picture must be rejected, got %v", err)
	}
	base.BigPicture = ""
	base.TechnicalDetails += "x"
	if err := base.Validate(); err == nil || !strings.Contains(err.Error(), "technical_details") {
		t.Fatalf("oversize technical_details must be rejected, got %v", err)
	}
}

// TestBuildEventTypedParity confirms the explicit-event-type path lets a producer
// keep a Source that differs from the token in its event type (agent-manager's
// hyphen source vs underscore event type).
func TestBuildEventTypedParity(t *testing.T) {
	ev := BuildEventTyped("agent-manager", "agent_manager.run.decision_requested.v1", time.Unix(1000, 0), Facts{
		Question: "Q", Recommended: "a", Options: []Option{{Key: "a"}, {Key: "b"}},
	})
	if ev.Source != "agent-manager" || ev.EventType != "agent_manager.run.decision_requested.v1" {
		t.Fatalf("BuildEventTyped must keep source and event type independent: src=%q type=%q", ev.Source, ev.EventType)
	}
}

// --- test doubles -----------------------------------------------------------

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

type memStore struct {
	rows map[string]*PendingAsk
}

func newMemStore() *memStore { return &memStore{rows: map[string]*PendingAsk{}} }

func (m *memStore) Get(_ context.Context, id string) (*PendingAsk, error) {
	if a, ok := m.rows[id]; ok {
		cp := *a
		return &cp, nil
	}
	return nil, nil
}

func (m *memStore) Put(_ context.Context, ask *PendingAsk) error {
	cp := *ask
	m.rows[ask.ID] = &cp
	return nil
}

func (m *memStore) CountOutstandingForSubject(_ context.Context, subject string, now time.Time) (int, error) {
	n := 0
	for _, a := range m.rows {
		if a.Subject == subject && a.Outstanding(now) {
			n++
		}
	}
	return n, nil
}

type capturePublisher struct {
	events []eventbus.DomainEvent
	err    error
}

func (p *capturePublisher) PublishDomainEvent(_ context.Context, e eventbus.DomainEvent) error {
	p.events = append(p.events, e)
	return p.err
}

// --- Raiser: idempotency, flood cap, determinism ----------------------------

func TestRaiserIdempotentAndPublishes(t *testing.T) {
	clock := fakeClock{t: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)}
	store := newMemStore()
	pub := &capturePublisher{}
	r := Raiser{Source: "agent-manager", Kind: "authorization", Store: store, Publisher: pub, Clock: clock}

	facts := Facts{
		Question: "Approve grant?", Recommended: "reject", Reversible: true, Urgency: "normal",
		Options: []Option{{Key: "approve"}, {Key: "reject"}},
	}
	ask := PendingAsk{ID: CorrelationID("auth", "subj", "run.create", ""), Subject: "subj", Extra: map[string]string{"kind": "standing"}}

	raised, err := r.Raise(context.Background(), ask, facts)
	if err != nil || !raised {
		t.Fatalf("first raise: raised=%v err=%v", raised, err)
	}
	if len(pub.events) != 1 {
		t.Fatalf("first raise should publish once, got %d", len(pub.events))
	}
	// Second raise of the same outstanding ask is idempotent: no new publish/row count.
	raised, err = r.Raise(context.Background(), ask, facts)
	if err != nil || !raised {
		t.Fatalf("second raise: raised=%v err=%v", raised, err)
	}
	if len(pub.events) != 1 {
		t.Fatalf("idempotent raise must not re-publish, got %d", len(pub.events))
	}
	// The stored row carries scenario Extra + owned lifecycle fields.
	got, _ := store.Get(context.Background(), ask.ID)
	if got.State != PendingStatePending || got.Extra["kind"] != "standing" || !got.ExpiresAt.After(got.RaisedAt) {
		t.Fatalf("stored ask wrong: %#v", got)
	}
}

func TestRaiserRetriesFailedPublicationWithSameEvent(t *testing.T) {
	clock := fakeClock{t: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)}
	store := newMemStore()
	pub := &capturePublisher{err: errors.New("events unavailable")}
	r := Raiser{Source: "agent-manager", Kind: "authorization", Store: store, Publisher: pub, Clock: clock}
	facts := Facts{Question: "Approve?", Recommended: "reject", Options: []Option{{Key: "approve"}, {Key: "reject"}}}
	ask := PendingAsk{ID: "auth-1", Subject: "runner"}
	if raised, err := r.Raise(context.Background(), ask, facts); !raised || err == nil {
		t.Fatalf("failed first publication: raised=%v err=%v", raised, err)
	}
	changedFacts := facts
	changedFacts.Question = "Approve expanded access?"
	if raised, err := r.Raise(context.Background(), ask, changedFacts); !raised || !errors.Is(err, ErrPendingFactsChanged) {
		t.Fatalf("an unpublished generation must refuse changed terms: raised=%v err=%v", raised, err)
	}
	clock.t = clock.t.Add(time.Minute)
	pub.err = nil
	if raised, err := r.Raise(context.Background(), ask, facts); !raised || err != nil {
		t.Fatalf("retry: raised=%v err=%v", raised, err)
	}
	if len(pub.events) != 2 || EventID(pub.events[0]) != EventID(pub.events[1]) {
		t.Fatalf("retry must publish the same event identity exactly once more: %#v", pub.events)
	}
	if got, _ := store.Get(context.Background(), ask.ID); got == nil || !got.Published {
		t.Fatalf("successful retry must persist publication receipt: %#v", got)
	}
	if _, err := r.Raise(context.Background(), ask, facts); err != nil || len(pub.events) != 2 {
		t.Fatalf("successful publication must suppress later retries: events=%d err=%v", len(pub.events), err)
	}
}

func TestRaiserFloodCap(t *testing.T) {
	clock := fakeClock{t: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)}
	store := newMemStore()
	pub := &capturePublisher{}
	r := Raiser{Source: "s", Kind: "k", Store: store, Publisher: pub, Clock: clock, MaxOutstanding: 2}
	facts := Facts{Question: "Q", Recommended: "a", Options: []Option{{Key: "a"}, {Key: "b"}}}

	for i, target := range []string{"t1", "t2"} {
		raised, err := r.Raise(context.Background(), PendingAsk{ID: CorrelationID("", "subj", target), Subject: "subj"}, facts)
		if err != nil || !raised {
			t.Fatalf("raise %d: raised=%v err=%v", i, raised, err)
		}
	}
	// Third distinct ask for the same subject trips the cap WITHOUT a new publish.
	raised, err := r.Raise(context.Background(), PendingAsk{ID: CorrelationID("", "subj", "t3"), Subject: "subj"}, facts)
	if !errors.Is(err, ErrFlood) {
		t.Fatalf("expected ErrFlood, got raised=%v err=%v", raised, err)
	}
	if raised {
		t.Fatalf("flooded raise must report raised=false")
	}
	if len(pub.events) != 2 {
		t.Fatalf("flood must not publish a 3rd notification, got %d", len(pub.events))
	}
}

func TestRaiserDeterministicOccurredAndFloodReraiseAfterWindow(t *testing.T) {
	clock := fakeClock{t: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)}
	store := newMemStore()
	pub := &capturePublisher{}
	r := Raiser{Source: "s", Kind: "k", Store: store, Publisher: pub, Clock: clock, Window: time.Hour}
	facts := Facts{Question: "Q", Recommended: "a", Options: []Option{{Key: "a"}, {Key: "b"}}}
	ask := PendingAsk{ID: CorrelationID("", "subj", "t"), Subject: "subj"}

	if _, err := r.Raise(context.Background(), ask, facts); err != nil {
		t.Fatalf("raise: %v", err)
	}
	// Occurred is truncated to the window, so the EventID is stable across the window.
	want := clock.t.UTC().Truncate(time.Hour)
	if !pub.events[0].Occurred.Equal(want) {
		t.Fatalf("occurred not window-truncated: got %v want %v", pub.events[0].Occurred, want)
	}
}

func TestRaiserEventTypeOverride(t *testing.T) {
	clock := fakeClock{t: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)}
	store := newMemStore()
	pub := &capturePublisher{}
	r := Raiser{Source: "agent-manager", Kind: "authorization", EventType: "agent_manager.authorization.decision_requested.v1", Store: store, Publisher: pub, Clock: clock}
	facts := Facts{Question: "Q", Recommended: "a", Options: []Option{{Key: "a"}, {Key: "b"}}}
	if _, err := r.Raise(context.Background(), PendingAsk{ID: "id", Subject: "subj"}, facts); err != nil {
		t.Fatalf("raise: %v", err)
	}
	if pub.events[0].EventType != "agent_manager.authorization.decision_requested.v1" {
		t.Fatalf("EventType override ignored: %q", pub.events[0].EventType)
	}
	if pub.events[0].Source != "agent-manager" {
		t.Fatalf("Source must stay as configured: %q", pub.events[0].Source)
	}
}

func TestRaiserNoStoreDegradesToRefuse(t *testing.T) {
	raised, err := Raiser{Source: "s", Kind: "k"}.Raise(context.Background(), PendingAsk{ID: "x", Subject: "y"}, Facts{})
	if raised || err != nil {
		t.Fatalf("no-store Raiser must report raised=false,nil; got raised=%v err=%v", raised, err)
	}
}

func TestRaiserPublishErrorIsBestEffort(t *testing.T) {
	clock := fakeClock{t: time.Now().UTC()}
	store := newMemStore()
	pub := &capturePublisher{err: errors.New("events down")}
	r := Raiser{Source: "s", Kind: "k", Store: store, Publisher: pub, Clock: clock}
	raised, err := r.Raise(context.Background(), PendingAsk{ID: "id", Subject: "subj"}, Facts{Question: "Q", Recommended: "a", Options: []Option{{Key: "a"}, {Key: "b"}}})
	if !raised || err == nil {
		t.Fatalf("publish error must still report raised=true and surface the error; raised=%v err=%v", raised, err)
	}
	if got, _ := store.Get(context.Background(), "id"); got == nil {
		t.Fatalf("durable row must stand despite publish failure")
	}
}

// --- Signature verify -------------------------------------------------------

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestValidSignature(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	secret := "s3cr3t"
	good := sign(body, secret)
	if !ValidSignature(body, good, secret) {
		t.Fatalf("good signature rejected")
	}
	if !ValidSignature(body, strings.TrimPrefix(good, "sha256="), secret) {
		t.Fatalf("prefix-less good signature rejected")
	}
	if ValidSignature(body, sign([]byte(`{"hello":"tampered"}`), secret), secret) {
		t.Fatalf("tampered body accepted")
	}
	if ValidSignature(body, sign(body, "wrong-secret"), secret) {
		t.Fatalf("forged (wrong-secret) signature accepted")
	}
	if ValidSignature(body, good, "") {
		t.Fatalf("empty secret must reject")
	}
	if ValidSignature(body, "not-hex", secret) {
		t.Fatalf("undecodable signature accepted")
	}
}

// --- Webhook fan-out --------------------------------------------------------

type recordingEffect struct {
	owns    string
	handled bool
	err     error
	applied *bool
}

func (e *recordingEffect) Handles(src string) bool { return e.owns == "" || e.owns == src }
func (e *recordingEffect) Apply(_ context.Context, _ DecisionResolution) (bool, error) {
	if e.applied != nil {
		*e.applied = true
	}
	return e.handled, e.err
}

func postResolution(t *testing.T, h http.Handler, secret, eventType, sourceEventType string) *httptest.ResponseRecorder {
	t.Helper()
	envelope := map[string]any{
		"event_id": "e1", "event_type": eventType, "source_scenario": "notification-hub",
		"payload": map[string]any{"ask_id": "a1", "state": "answered", "answer": "approve", "source_event_type": sourceEventType},
	}
	body, _ := json.Marshal(envelope)
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set(SignatureHeader, sign(body, secret))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestWebhookFanOutFirstHandledWins(t *testing.T) {
	secret := "whsec"
	var firstApplied, secondApplied bool
	first := &recordingEffect{owns: "agent-manager.run.decision_requested.v1", handled: true, applied: &firstApplied}
	second := &recordingEffect{owns: "agent-manager.run.decision_requested.v1", handled: true, applied: &secondApplied}
	h := Webhook(secret, first, second)

	rec := postResolution(t, h, secret, AskResolvedEventType, "agent-manager.run.decision_requested.v1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rec.Code, rec.Body.String())
	}
	if !firstApplied || secondApplied {
		t.Fatalf("first handled must win and stop fan-out: first=%v second=%v", firstApplied, secondApplied)
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["handled"] != true {
		t.Fatalf("expected handled:true, got %v", out)
	}
}

func TestWebhookFallThroughOnHandlesAndUnhandled(t *testing.T) {
	secret := "whsec"
	var ranFirst, ranSecond bool
	// First does not OWN the source event type → skipped entirely.
	first := &recordingEffect{owns: "other.kind.decision_requested.v1", handled: true, applied: &ranFirst}
	// Second owns it and handles it.
	second := &recordingEffect{owns: "agent-manager.run.decision_requested.v1", handled: true, applied: &ranSecond}
	h := Webhook(secret, first, second)
	rec := postResolution(t, h, secret, AskResolvedEventType, "agent-manager.run.decision_requested.v1")
	if ranFirst {
		t.Fatalf("effect that does not own the source type must be skipped")
	}
	if !ranSecond || rec.Code != http.StatusOK {
		t.Fatalf("owning effect must run; code=%d", rec.Code)
	}
}

func TestWebhookUnclaimedResolutionIgnored(t *testing.T) {
	secret := "whsec"
	h := Webhook(secret, &recordingEffect{owns: "nobody", handled: false})
	rec := postResolution(t, h, secret, AskResolvedEventType, "unknown.kind.decision_requested.v1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["ignored"] != true {
		t.Fatalf("expected ignored:true, got %v", out)
	}
}

func TestWebhookNonResolutionEventIgnored(t *testing.T) {
	secret := "whsec"
	h := Webhook(secret, &recordingEffect{handled: true})
	rec := postResolution(t, h, secret, "some.other.event.v1", "agent-manager.run.decision_requested.v1")
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if rec.Code != http.StatusOK || out["ignored"] != true {
		t.Fatalf("non-resolution event must be ignored; code=%d out=%v", rec.Code, out)
	}
}

func TestWebhookEffectErrorIs500(t *testing.T) {
	secret := "whsec"
	h := Webhook(secret, &recordingEffect{owns: "agent-manager.run.decision_requested.v1", err: errors.New("boom")})
	rec := postResolution(t, h, secret, AskResolvedEventType, "agent-manager.run.decision_requested.v1")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("effect error must be 500, got %d", rec.Code)
	}
}

func TestWebhookRejectsBadSignatureAndMethod(t *testing.T) {
	secret := "whsec"
	h := Webhook(secret, &recordingEffect{handled: true})

	body := []byte(`{"event_type":"notification_hub.ask.resolved.v1"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set(SignatureHeader, sign(body, "wrong"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("forged signature must be 401, got %d", rec.Code)
	}

	get := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, get)
	if rec2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET must be 405, got %d", rec2.Code)
	}
}

// TestWebhookOverCapBodyIs413 pins that an over-limit body is rejected with an
// explicit 413 (RequestEntityTooLarge) rather than silently truncated — a
// truncated body would fail the HMAC check and masquerade as a permanent 401.
func TestWebhookOverCapBodyIs413(t *testing.T) {
	secret := "whsec"
	h := Webhook(secret, &recordingEffect{handled: true})

	// Build a correctly-signed body that is > 1 MiB by padding the note field.
	envelope := map[string]any{
		"event_type": AskResolvedEventType,
		"payload":    map[string]any{"ask_id": "a1", "state": "answered", "note": strings.Repeat("x", 1<<20)},
	}
	body, _ := json.Marshal(envelope)
	if len(body) <= 1<<20 {
		t.Fatalf("test body must exceed the cap, got %d", len(body))
	}
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set(SignatureHeader, sign(body, secret)) // valid signature over the full body
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("over-cap body must be 413, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	// A body AT the cap (exactly 1 MiB) is still accepted and processed (here it
	// verifies and reaches the effect → 200).
	pad := 1<<20 - len(mustMarshal(map[string]any{"event_type": AskResolvedEventType, "payload": map[string]any{"ask_id": "a1", "state": "answered", "note": ""}}))
	atCap := mustMarshal(map[string]any{"event_type": AskResolvedEventType, "payload": map[string]any{"ask_id": "a1", "state": "answered", "note": strings.Repeat("x", pad)}})
	if len(atCap) != 1<<20 {
		t.Fatalf("at-cap body must be exactly 1 MiB, got %d", len(atCap))
	}
	req2 := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(atCap)))
	req2.Header.Set(SignatureHeader, sign(atCap, secret))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("at-cap body must be processed (200), got %d (body=%s)", rec2.Code, rec2.Body.String())
	}
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
