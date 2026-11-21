package operatordecision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/serviceprincipal"
)

const (
	verifySecret = "whsec"
	verifyKind   = "storage-manager.deletion-grant.decision_requested.v1"
)

// legacyWebhook is a webhook with no Hub readback, which is what a scenario with
// no ask-reader credential has.
func legacyWebhook(secret string, effects ...DomainEffect) *Webhook {
	return NewWebhook(WebhookConfig{Secret: StaticSecret(secret), Effects: effects, Logf: func(string, ...any) {}})
}

// hubFake stands in for Notification Hub as one scenario's ask-reader sees it.
type hubFake struct {
	mu    sync.Mutex
	asks  map[string]askreader.Ask
	err   error
	reads int
}

func (h *hubFake) ReadAsk(_ context.Context, id string) (askreader.Ask, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reads++
	if h.err != nil {
		return askreader.Ask{}, h.err
	}
	ask, ok := h.asks[id]
	if !ok {
		return askreader.Ask{}, fmt.Errorf("%w: unknown", askreader.ErrAskNotFound)
	}
	return ask, nil
}

func (h *hubFake) setErr(err error) {
	h.mu.Lock()
	h.err = err
	h.mu.Unlock()
}

// answeredGrant is the Hub's record of a genuine, approved deletion-grant ask.
func answeredGrant() askreader.Ask {
	return askreader.Ask{
		ID: "ask-1", State: StateAnswered, Answer: "approve", AnsweredBy: "alice", AnsweredByIdentityKind: "verified_human",
		AnsweredByIdentitySource: "cloudflare_access", AnsweredAt: "2026-11-14T10:00:00Z", ResolvedAt: "2026-11-14T10:00:00Z",
		Source: "storage-manager", SourceEventType: verifyKind, RequesterRef: "evt-1", SubjectKey: "grant:g1",
		Correlation: map[string]string{"ask_id": "grant-g1"},
	}
}

// claimOf is the event body the Hub publishes for ask: what a genuine delivery says.
func claimOf(ask askreader.Ask) map[string]any {
	return map[string]any{
		"ask_id": ask.ID, "state": ask.State, "answer": ask.Answer, "answered_by": ask.AnsweredBy,
		"source": ask.Source, "source_event_type": ask.SourceEventType, "requester_ref": ask.RequesterRef,
		"subject_key": ask.SubjectKey, "correlation": ask.Correlation,
	}
}

type verifyFixture struct {
	hub     *hubFake
	webhook *Webhook
	applied []DecisionResolution
	logs    []string
}

func newVerifyFixture(t *testing.T, asks ...askreader.Ask) *verifyFixture {
	t.Helper()
	f := &verifyFixture{hub: &hubFake{asks: map[string]askreader.Ask{}}}
	for _, ask := range asks {
		f.hub.asks[ask.ID] = ask
	}
	effect := &captureEffect{owns: verifyKind, applied: &f.applied}
	f.webhook = NewWebhook(WebhookConfig{
		Secret: StaticSecret(verifySecret), Source: "storage-manager", Reader: f.hub, Effects: []DomainEffect{effect},
		Logf: func(format string, args ...any) { f.logs = append(f.logs, fmt.Sprintf(format, args...)) },
	})
	return f
}

type captureEffect struct {
	owns    string
	applied *[]DecisionResolution
}

func (e *captureEffect) Handles(kind string) bool { return kind == e.owns }
func (e *captureEffect) Apply(_ context.Context, resolution DecisionResolution) (bool, error) {
	*e.applied = append(*e.applied, resolution)
	return true, nil
}

func (f *verifyFixture) post(eventType string, payload map[string]any) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]any{"event_id": "e1", "event_type": eventType, "source_scenario": "notification-hub", "payload": payload})
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set(SignatureHeader, sign(body, verifySecret))
	rec := httptest.NewRecorder()
	f.webhook.ServeHTTP(rec, req)
	return rec
}

func TestAGenuineResolutionIsAppliedFromTheHubsRecord(t *testing.T) {
	f := newVerifyFixture(t, answeredGrant())
	rec := f.post(AskResolvedEventType, claimOf(answeredGrant()))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"handled":true`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(f.applied) != 1 {
		t.Fatalf("applied %d times", len(f.applied))
	}
	got := f.applied[0]
	if got.AnsweredBy != "alice" || got.AnsweredByIdentityKind != "verified_human" || got.Correlation["ask_id"] != "grant-g1" || got.ByDefault {
		t.Fatalf("the effect must receive the Hub's record: %+v", got)
	}
	if stats := f.webhook.Stats(); stats.Verified != 1 || stats.UnverifiedApplied != 0 || len(stats.Refused) != 0 {
		t.Fatalf("stats = %+v", stats)
	}
}

// The body can say anything, because the bus accepts an event from any caller.
func TestAForgedResolutionOfAPendingAskIsRefusedAndAppliesNothing(t *testing.T) {
	pending := answeredGrant()
	pending.State, pending.Answer, pending.AnsweredBy = "pending", "", ""
	f := newVerifyFixture(t, pending)
	rec := f.post(AskResolvedEventType, claimOf(answeredGrant()))
	if rec.Code != http.StatusConflict || len(f.applied) != 0 {
		t.Fatalf("status=%d applied=%d body=%s", rec.Code, len(f.applied), rec.Body.String())
	}
	if stats := f.webhook.Stats(); stats.Refused["ask_open"] != 1 || stats.Verified != 0 {
		t.Fatalf("stats = %+v", stats)
	}
	if len(f.logs) == 0 || !strings.Contains(f.logs[0], "REFUSED") {
		t.Fatalf("a refusal must be logged: %v", f.logs)
	}
}

func TestAForgedFieldOfAGenuineAnswerIsRefused(t *testing.T) {
	cases := map[string]func(map[string]any){
		"answer":            func(c map[string]any) { c["answer"] = "approve-everything" },
		"answerer":          func(c map[string]any) { c["answered_by"] = "mallory" },
		"state":             func(c map[string]any) { c["state"] = "defaulted" },
		"producer event":    func(c map[string]any) { c["source_event_type"] = "storage-manager.other.decision_requested.v1" },
		"requester ref":     func(c map[string]any) { c["requester_ref"] = "evt-forged" },
		"subject key":       func(c map[string]any) { c["subject_key"] = "grant:other" },
		"correlation value": func(c map[string]any) { c["correlation"] = map[string]string{"ask_id": "grant-HIGH-RISK"} },
		"correlation extra": func(c map[string]any) { c["correlation"] = map[string]string{"ask_id": "grant-g1", "also": "x"} },
		"no correlation":    func(c map[string]any) { c["correlation"] = map[string]string{} },
	}
	for name, mutate := range cases {
		f := newVerifyFixture(t, answeredGrant())
		claim := claimOf(answeredGrant())
		mutate(claim)
		// The effect handles the producer event the body names, so a forged event
		// type still reaches an effect if verification did not stop it.
		if kind, _ := claim["source_event_type"].(string); kind != verifyKind {
			f.webhook.cfg.Effects = append(f.webhook.cfg.Effects, &captureEffect{owns: kind, applied: &f.applied})
		}
		rec := f.post(AskResolvedEventType, claim)
		if rec.Code != http.StatusConflict || len(f.applied) != 0 {
			t.Fatalf("%s: status=%d applied=%d body=%s", name, rec.Code, len(f.applied), rec.Body.String())
		}
		if f.webhook.Stats().Refused["mismatch"] != 1 {
			t.Fatalf("%s: stats = %+v", name, f.webhook.Stats())
		}
	}
}

func TestAResolutionForAnAskTheHubDoesNotHoldIsRefused(t *testing.T) {
	f := newVerifyFixture(t) // the Hub holds nothing
	rec := f.post(AskResolvedEventType, claimOf(answeredGrant()))
	if rec.Code != http.StatusConflict || len(f.applied) != 0 || f.webhook.Stats().Refused["no_such_ask"] != 1 {
		t.Fatalf("status=%d applied=%d stats=%+v", rec.Code, len(f.applied), f.webhook.Stats())
	}
}

func TestAnAskTheHubSaysIsAnotherScenariosIsRefused(t *testing.T) {
	other := answeredGrant()
	other.Source = "agent-manager"
	f := newVerifyFixture(t, other)
	rec := f.post(AskResolvedEventType, claimOf(answeredGrant()))
	if rec.Code != http.StatusConflict || len(f.applied) != 0 || f.webhook.Stats().Refused["other_source"] != 1 {
		t.Fatalf("status=%d applied=%d stats=%+v", rec.Code, len(f.applied), f.webhook.Stats())
	}
}

// Every scenario receives every resolution on the bus; one about another
// scenario's ask is acknowledged without a read.
func TestAResolutionClaimingAnotherScenariosAskIsIgnoredWithoutAHubRead(t *testing.T) {
	f := newVerifyFixture(t, answeredGrant())
	claim := claimOf(answeredGrant())
	claim["source"] = "agent-manager"
	rec := f.post(AskResolvedEventType, claim)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ignored":true`) || f.hub.reads != 0 || len(f.applied) != 0 {
		t.Fatalf("status=%d reads=%d applied=%d body=%s", rec.Code, f.hub.reads, len(f.applied), rec.Body.String())
	}
}

// A real approval must survive a Hub outage: the bus redelivers a 503.
func TestAHubOutageIsRetriedNotLost(t *testing.T) {
	f := newVerifyFixture(t, answeredGrant())
	f.hub.setErr(fmt.Errorf("%w: connection refused", askreader.ErrHubUnavailable))
	rec := f.post(AskResolvedEventType, claimOf(answeredGrant()))
	if rec.Code != http.StatusServiceUnavailable || len(f.applied) != 0 || f.webhook.Stats().Unavailable != 1 {
		t.Fatalf("status=%d applied=%d stats=%+v", rec.Code, len(f.applied), f.webhook.Stats())
	}
	f.hub.setErr(nil)
	rec = f.post(AskResolvedEventType, claimOf(answeredGrant()))
	if rec.Code != http.StatusOK || len(f.applied) != 1 {
		t.Fatalf("the redelivery must apply it: status=%d applied=%d", rec.Code, len(f.applied))
	}
}

func TestWithoutACredentialTheBodyIsAppliedCountedAndLoggedLoudly(t *testing.T) {
	f := newVerifyFixture(t)
	f.hub.setErr(fmt.Errorf("%w: no credential", askreader.ErrNotIssued))
	rec := f.post(AskResolvedEventType, claimOf(answeredGrant()))
	if rec.Code != http.StatusOK || len(f.applied) != 1 {
		t.Fatalf("status=%d applied=%d", rec.Code, len(f.applied))
	}
	if stats := f.webhook.Stats(); stats.UnverifiedApplied != 1 || stats.Verified != 0 {
		t.Fatalf("stats = %+v", stats)
	}
	if len(f.logs) != 1 || !strings.Contains(f.logs[0], "UNVERIFIED") || !strings.Contains(f.logs[0], serviceprincipal.IssueCommand) {
		t.Fatalf("the gap must be logged with the operator's command: %v", f.logs)
	}
}

func TestAWebhookWithNoReaderIsTheCountedLegacyPath(t *testing.T) {
	var applied []DecisionResolution
	w := NewWebhook(WebhookConfig{Secret: StaticSecret(verifySecret), Source: "storage-manager", Effects: []DomainEffect{&captureEffect{owns: verifyKind, applied: &applied}}, Logf: func(string, ...any) {}})
	body, _ := json.Marshal(map[string]any{"event_id": "e1", "event_type": AskResolvedEventType, "payload": claimOf(answeredGrant())})
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set(SignatureHeader, sign(body, verifySecret))
	rec := httptest.NewRecorder()
	w.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || len(applied) != 1 || w.Stats().UnverifiedApplied != 1 {
		t.Fatalf("status=%d applied=%d stats=%+v", rec.Code, len(applied), w.Stats())
	}
}

// A verified ask the effect does not own is ignored, as before.
func TestAVerifiedResolutionNoEffectOwnsIsIgnoredWithoutAHubRead(t *testing.T) {
	f := newVerifyFixture(t, answeredGrant())
	claim := claimOf(answeredGrant())
	claim["source_event_type"] = "storage-manager.unknown.decision_requested.v1"
	rec := f.post(AskResolvedEventType, claim)
	if rec.Code != http.StatusOK || f.hub.reads != 0 || len(f.applied) != 0 {
		t.Fatalf("status=%d reads=%d applied=%d", rec.Code, f.hub.reads, len(f.applied))
	}
}

func TestAnUnrelatedHubFailureIsTreatedAsUnavailableNotAsApproval(t *testing.T) {
	f := newVerifyFixture(t, answeredGrant())
	f.hub.setErr(errors.New("boom"))
	rec := f.post(AskResolvedEventType, claimOf(answeredGrant()))
	if rec.Code != http.StatusServiceUnavailable || len(f.applied) != 0 {
		t.Fatalf("status=%d applied=%d", rec.Code, len(f.applied))
	}
}

// ---- context requests ----

type answeringContext struct {
	captureEffect
	asked *[]ContextRequest
}

func (e *answeringContext) RespondToContext(_ context.Context, request ContextRequest) (bool, error) {
	*e.asked = append(*e.asked, request)
	return true, nil
}

func newContextFixture(t *testing.T) (*verifyFixture, *[]ContextRequest) {
	t.Helper()
	open := answeredGrant()
	open.State, open.Answer, open.AnsweredBy = "pending", "", ""
	open.ContextRequests = []askreader.ContextRequest{{ID: "req-1", Question: "Which partition?", RequestedAt: "2026-11-14T09:00:00Z"}}
	f := newVerifyFixture(t, open)
	var asked []ContextRequest
	f.webhook.cfg.Effects = []DomainEffect{&answeringContext{captureEffect: captureEffect{owns: verifyKind, applied: &f.applied}, asked: &asked}}
	return f, &asked
}

func contextClaim() map[string]any {
	return map[string]any{
		"ask_id": "ask-1", "request_id": "req-1", "question": "Which partition?", "source": "storage-manager",
		"source_event_type": verifyKind, "requester_ref": "evt-1", "subject_key": "grant:g1",
		"correlation": map[string]string{"ask_id": "grant-g1"},
	}
}

func TestAGenuineContextRequestIsAnsweredFromTheHubsRecord(t *testing.T) {
	f, asked := newContextFixture(t)
	if rec := f.post(AskContextRequestedEventType, contextClaim()); rec.Code != http.StatusOK || len(*asked) != 1 {
		t.Fatalf("status=%d asked=%d body=%s", rec.Code, len(*asked), rec.Body.String())
	}
	if got := (*asked)[0]; got.Question != "Which partition?" || got.Correlation["ask_id"] != "grant-g1" || got.SubjectKey != "grant:g1" {
		t.Fatalf("request = %+v", got)
	}
}

func TestAForgedContextRequestIsRefused(t *testing.T) {
	cases := map[string]func(map[string]any){
		"unknown request": func(c map[string]any) { c["request_id"] = "req-forged" },
		"question":        func(c map[string]any) { c["question"] = "Delete everything?" },
		"subject key":     func(c map[string]any) { c["subject_key"] = "grant:other" },
	}
	for name, mutate := range cases {
		f, asked := newContextFixture(t)
		claim := contextClaim()
		mutate(claim)
		if rec := f.post(AskContextRequestedEventType, claim); rec.Code != http.StatusConflict || len(*asked) != 0 {
			t.Fatalf("%s: status=%d asked=%d", name, rec.Code, len(*asked))
		}
	}
}

// A forged correlation cannot redirect the answer: the producer acts on the Hub's.
func TestAContextRequestIsAnsweredWithTheHubsCorrelationNotTheBodys(t *testing.T) {
	f, asked := newContextFixture(t)
	claim := contextClaim()
	claim["correlation"] = map[string]string{"ask_id": "grant-HIGH-RISK"}
	claim["requester_ref"] = "evt-since-revised"
	if rec := f.post(AskContextRequestedEventType, claim); rec.Code != http.StatusOK || len(*asked) != 1 {
		t.Fatalf("status=%d asked=%d", rec.Code, len(*asked))
	}
	if got := (*asked)[0]; got.Correlation["ask_id"] != "grant-g1" || got.RequesterRef != "evt-1" {
		t.Fatalf("the producer must act on the Hub's record: %+v", got)
	}
}
