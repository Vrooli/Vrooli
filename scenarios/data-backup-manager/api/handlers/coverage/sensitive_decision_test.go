package coverage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"data-backup-manager/internal/coverage"
	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/sources"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

type sensitiveFlowService struct {
	report     coverage.Report
	registered []string
	acceptErr  error
}

func (s *sensitiveFlowService) Report(context.Context) (coverage.Report, error) { return s.report, nil }
func (s *sensitiveFlowService) AcceptDefaults(context.Context, coverage.AcceptOptions) (coverage.AcceptResult, error) {
	return coverage.AcceptResult{}, nil
}

func (s *sensitiveFlowService) AcceptSensitive(_ context.Context, ids []string) (coverage.AcceptResult, error) {
	if s.acceptErr != nil {
		return coverage.AcceptResult{}, s.acceptErr
	}
	s.registered = append(s.registered, ids...)
	accepted := make([]coverage.AcceptedTarget, 0, len(ids))
	for _, id := range ids {
		accepted = append(accepted, coverage.AcceptedTarget{TargetID: "target-" + id, SuggestionID: id})
	}
	return coverage.AcceptResult{Accepted: accepted}, nil
}

func (s *sensitiveFlowService) UnregisteredDefaultTargets(context.Context) ([]coverage.Suggestion, error) {
	return nil, nil
}

type sensitiveFlowPublisher struct {
	events    []eventbus.DomainEvent
	failEvent string
	failures  int
}

func (p *sensitiveFlowPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.events = append(p.events, event)
	if event.EventType == p.failEvent && p.failures > 0 {
		p.failures--
		return errors.New("temporary publisher failure")
	}
	return nil
}

func sensitiveFlowFixture(t *testing.T) (*SensitiveDecisionFlow, *sensitiveFlowService, *sensitiveFlowPublisher, *destinationreadiness.FileDecisionStore) {
	t.Helper()
	svc := &sensitiveFlowService{report: coverage.Report{Sensitive: []coverage.Suggestion{
		{ID: "secret-a", Owner: "codex", Name: "auth token", Locator: "/secret/token", SourceKind: sources.KindFilesystem, Sensitive: true, Warning: "Contains credentials"},
		{ID: "secret-b", Owner: "vrooli", Name: "provider key", Locator: "/secret/key", SourceKind: sources.KindFilesystem, Sensitive: true, Critical: true, ReacquisitionCommand: "owner-import", ReacquisitionEvidence: "owner-runbook", ReacquisitionNeedsOperator: true, ReacquisitionAuthReason: "External provider approval is required"},
	}}}
	publisher := &sensitiveFlowPublisher{}
	store := &destinationreadiness.FileDecisionStore{Root: filepath.Join(t.TempDir(), "asks")}
	flow := NewSensitiveDecisionFlow(svc, store, publisher, true, nil)
	flow.now = func() time.Time { return time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC) }
	flow.readFile = func(string) ([]byte, error) {
		return []byte("## Actionable now\nNo actionable items are recorded.\n"), nil
	}
	return flow, svc, publisher, store
}

func humanResolution(ask *operatordecision.PendingAsk, answer, note string) operatordecision.DecisionResolution {
	return operatordecision.DecisionResolution{
		AskID: ask.ID, State: operatordecision.StateAnswered, Answer: answer, Note: note,
		AnsweredBy: "operator@example.test", AnsweredByIdentityKind: "verified-human", AnsweredByIdentitySource: "notification-hub",
		SourceEventType: operatordecision.DecisionRequestedEventType("data-backup-manager", sensitiveDecisionKind),
		Correlation:     map[string]string{"ask_id": ask.ID, "candidate_fingerprint": ask.Extra["candidate_fingerprint"]},
	}
}

func TestSensitiveDecisionFlow_ProtectsCurrentBatchOnceAndRecordsResult(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	askID, count, err := flow.Request(ctx)
	if err != nil || count != 2 {
		t.Fatalf("Request = %q, %d, %v", askID, count, err)
	}
	ask, err := store.Get(ctx, askID)
	if err != nil || ask == nil || !ask.Published {
		t.Fatalf("stored ask = %+v, %v", ask, err)
	}
	requestText, _ := json.Marshal(publisher.events[0].Payload)
	if strings.Contains(string(requestText), "/secret/") {
		t.Fatalf("notification leaked sensitive locator: %s", requestText)
	}
	if _, ok := publisher.events[0].Payload["recommended"]; !ok {
		t.Fatal("review has no recommendation")
	}
	if _, ok := publisher.events[0].Payload["note"]; !ok {
		t.Fatal("review omitted attention context")
	}
	technical, ok := publisher.events[0].Payload["technical_details"].(string)
	if !ok || !strings.Contains(technical, "secret-a") || !strings.Contains(technical, attentionFile) || !strings.Contains(technical, "Actionable now") {
		t.Fatalf("candidate list or attention context is not visible in technical details: %q", technical)
	}

	resolution := humanResolution(ask, "protect_all", "")
	if handled, err := flow.Apply(ctx, resolution); err != nil || !handled {
		t.Fatalf("Apply = %v, %v", handled, err)
	}
	if len(svc.registered) != 2 {
		t.Fatalf("registered %v, want both candidates", svc.registered)
	}
	if handled, err := flow.Apply(ctx, resolution); err != nil || !handled {
		t.Fatalf("replay = %v, %v", handled, err)
	}
	if len(svc.registered) != 2 {
		t.Fatalf("replay registered again: %v", svc.registered)
	}
	stored, _ := store.Get(ctx, askID)
	if stored.State != operatordecision.PendingStateApproved || stored.Extra["result_notice_pending"] != "false" || stored.Extra["decisions"] == "" {
		t.Fatalf("outcome not durably recorded: %+v", stored)
	}
	if len(publisher.events) != 2 || publisher.events[1].EventType != sensitiveResultEvent {
		t.Fatalf("events = %+v, want request and one result notice", publisher.events)
	}
}

func TestSensitiveDecisionFlow_UsesSignedWebhookEndToEnd(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	askID, _, err := flow.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ask, err := store.Get(ctx, askID)
	if err != nil || ask == nil {
		t.Fatalf("get ask = %+v, %v", ask, err)
	}
	resolution := humanResolution(ask, "protect_all", "")
	body, err := json.Marshal(map[string]any{
		"event_type": operatordecision.AskResolvedEventType,
		"payload":    resolution,
	})
	if err != nil {
		t.Fatal(err)
	}
	const secret = "signed-test-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	makeRequest := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(string(body)))
		req.Header.Set(operatordecision.SignatureHeader, hex.EncodeToString(mac.Sum(nil)))
		resp := httptest.NewRecorder()
		operatordecision.NewWebhook(operatordecision.WebhookConfig{Secret: operatordecision.StaticSecret(secret), Effects: []operatordecision.DomainEffect{flow}}).ServeHTTP(resp, req)
		return resp
	}
	if resp := makeRequest(); resp.Code != http.StatusOK {
		t.Fatalf("signed callback status=%d body=%s", resp.Code, resp.Body.String())
	}
	if len(svc.registered) != 2 || len(publisher.events) != 2 || publisher.events[1].EventType != sensitiveResultEvent {
		t.Fatalf("registered=%v events=%v; want one batch and result notice", svc.registered, publisher.events)
	}
	if resp := makeRequest(); resp.Code != http.StatusOK {
		t.Fatalf("replayed signed callback status=%d body=%s", resp.Code, resp.Body.String())
	}
	if len(svc.registered) != 2 || len(publisher.events) != 2 {
		t.Fatalf("signed replay duplicated effects: registered=%v events=%d", svc.registered, len(publisher.events))
	}
}

func TestSensitiveDecisionFlow_RejectsChangedCandidatesWithoutRegistering(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	askID, _, err := flow.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ask, _ := store.Get(ctx, askID)
	svc.report.Sensitive[1].Locator = "/secret/rotated"
	if handled, err := flow.Apply(ctx, humanResolution(ask, "protect_all", "")); err != nil || !handled {
		t.Fatalf("Apply = %v, %v", handled, err)
	}
	if len(svc.registered) != 0 {
		t.Fatalf("stale approval registered %v", svc.registered)
	}
	stored, _ := store.Get(ctx, askID)
	if stored.State != operatordecision.PendingStateRejected || !strings.Contains(stored.Extra["result"], "changed") {
		t.Fatalf("stale result = %+v", stored)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("events = %d, want request and result", len(publisher.events))
	}
}

func TestSensitiveDecisionFlow_OtherRequiresCompleteExplicitReasons(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	askID, _, err := flow.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ask, _ := store.Get(ctx, askID)
	choices := `[{"suggestion_id":"secret-a","decision":"exclude","reason":"Owner confirms this is a temporary token"},{"suggestion_id":"secret-b","decision":"reacquire"}]`
	if handled, err := flow.Apply(ctx, humanResolution(ask, "other", choices)); err != nil || !handled {
		t.Fatalf("Apply = %v, %v", handled, err)
	}
	if len(svc.registered) != 0 {
		t.Fatalf("non-protect decisions registered targets: %v", svc.registered)
	}
	stored, _ := store.Get(ctx, askID)
	var got []sensitiveChoice
	if err := json.Unmarshal([]byte(stored.Extra["decisions"]), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Decision != "exclude" || got[0].Reason == "" || !got[1].OwnerNeedsAuthorization {
		t.Fatalf("stored decisions = %+v", got)
	}
}

func TestSensitiveDecisionFlow_NeverPublishesAskWithoutConfiguredCallback(t *testing.T) {
	flow, _, publisher, _ := sensitiveFlowFixture(t)
	flow.enabled = false
	if _, _, err := flow.Request(context.Background()); err == nil {
		t.Fatal("Request succeeded without callback")
	}
	if len(publisher.events) != 0 {
		t.Fatalf("published %d events without callback", len(publisher.events))
	}
}

func TestSensitiveDecisionFlow_UnverifiedApprovalCannotRegister(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	askID, _, err := flow.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ask, _ := store.Get(ctx, askID)
	resolution := humanResolution(ask, "protect_all", "")
	resolution.AnsweredByIdentityKind = "agent"
	if handled, err := flow.Apply(ctx, resolution); err != nil || !handled {
		t.Fatalf("Apply = %v, %v", handled, err)
	}
	if len(svc.registered) != 0 {
		t.Fatalf("unverified identity registered targets: %v", svc.registered)
	}
}

func TestSensitiveDecisionFlow_RetriesResultNoticeWithoutReplayingRegistration(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	askID, _, err := flow.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ask, _ := store.Get(ctx, askID)
	resolution := humanResolution(ask, "protect_all", "")
	publisher.failEvent, publisher.failures = sensitiveResultEvent, 1
	if handled, err := flow.Apply(ctx, resolution); !handled || err == nil {
		t.Fatalf("first Apply = %v, %v; want callback retry", handled, err)
	}
	if len(svc.registered) != 2 {
		t.Fatalf("first callback registered %v", svc.registered)
	}
	if handled, err := flow.Apply(ctx, resolution); err != nil || !handled {
		t.Fatalf("result notice retry = %v, %v", handled, err)
	}
	if len(svc.registered) != 2 {
		t.Fatalf("notice retry replayed registration: %v", svc.registered)
	}
	stored, _ := store.Get(ctx, askID)
	if stored.Extra["result_notice_pending"] != "false" {
		t.Fatalf("result notice was not recorded delivered: %+v", stored.Extra)
	}
}

func TestSensitiveDecisionFlow_ReacquireRequiresOwnerEvidence(t *testing.T) {
	flow, _, _, store := sensitiveFlowFixture(t)
	askID, _, err := flow.Request(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ask, _ := store.Get(context.Background(), askID)
	resolution := humanResolution(ask, "other", `[{"suggestion_id":"secret-a","decision":"reacquire"},{"suggestion_id":"secret-b","decision":"protect"}]`)
	if _, err := choicesForResolution(resolution, snapshotCandidates([]coverage.Suggestion{{ID: "secret-a", Sensitive: true}, {ID: "secret-b", Sensitive: true}})); err == nil || !strings.Contains(err.Error(), "owner evidence") {
		t.Fatalf("reacquire without owner proof error = %v", err)
	}
}
