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

	"data-backup-manager/internal/attention"
	"data-backup-manager/internal/coverage"
	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/sources"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"

	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage"
)

type sensitiveFlowService struct {
	report     coverage.Report
	registered []string
	// declared are the ids AcceptDefaults registered (non-sensitive, owner-declared).
	declared  []string
	acceptErr error
	// consume drops accepted candidates from the report, like real discovery does
	// once a target is registered.
	consume bool
}

func (s *sensitiveFlowService) Report(context.Context) (coverage.Report, error) { return s.report, nil }
func (s *sensitiveFlowService) AcceptDefaults(_ context.Context, opts coverage.AcceptOptions) (coverage.AcceptResult, error) {
	if opts.IncludeSensitive {
		return coverage.AcceptResult{}, coverage.ErrSensitiveReviewRequired
	}
	var accepted []coverage.AcceptedTarget
	for _, c := range s.report.Recommended {
		s.declared = append(s.declared, c.ID)
		accepted = append(accepted, coverage.AcceptedTarget{TargetID: "target-" + c.ID, SuggestionID: c.ID})
	}
	s.report.Recommended = nil
	return coverage.AcceptResult{Accepted: accepted}, nil
}

func (s *sensitiveFlowService) AcceptSensitive(_ context.Context, ids []string) (coverage.AcceptResult, error) {
	if s.acceptErr != nil {
		return coverage.AcceptResult{}, s.acceptErr
	}
	s.registered = append(s.registered, ids...)
	if s.consume {
		kept := s.report.Sensitive[:0:0]
		for _, c := range s.report.Sensitive {
			if !contains(ids, c.ID) {
				kept = append(kept, c)
			}
		}
		s.report.Sensitive = kept
	}
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
		AnsweredBy: "operator@example.test", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "notification-hub",
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
	technical, ok := publisher.events[0].Payload["technical_details"].(string)
	if !ok || !strings.Contains(technical, "secret-a") || !strings.Contains(technical, attention.File()) || !strings.Contains(technical, "Actionable now") {
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
	if _, _, err := choicesForResolution(resolution, snapshotCandidates([]coverage.Suggestion{{ID: "secret-a", Sensitive: true}, {ID: "secret-b", Sensitive: true}}), time.Time{}); err == nil || !strings.Contains(err.Error(), "owner evidence") {
		t.Fatalf("reacquire without owner proof error = %v", err)
	}
}

// realisticSensitive mirrors the host's review: an AI tool's home folder, an
// app key, a service secret and setup-written host settings, none of which
// declares how to get it back.
func realisticSensitive() []coverage.Suggestion {
	return []coverage.Suggestion{
		{ID: "a58721892ba534c6", Owner: "claude-code", OwnerKind: "tool", Name: "home", Locator: "/home/u/.claude", ApproxBytes: 2_287_318_942, Sensitive: true, Rationale: "Claude Code keeps transcripts, settings, plans and credentials here."},
		{ID: "dbd0bda4864ffc49", Owner: "notification-hub", OwnerKind: "scenario", Name: "vapid-private-key", Locator: "/home/u/.vrooli/state/nh/vapid", ApproxBytes: 44, Sensitive: true, Rationale: "Private key that signs Web Push deliveries."},
		{ID: "ddcb84c33ac12841", Owner: "vault", OwnerKind: "resource", Name: "data", Locator: "/home/u/.vrooli/data/vault", ApproxBytes: 17_828, Sensitive: true, Critical: true, Rationale: "Encrypted secret state that must be preserved."},
		{ID: "f1d3160e7e889ab5", Owner: "host_hardening", OwnerKind: "safeguard", Name: "sysctl_config", Locator: "/etc/sysctl.d/99-vrooli.conf", ApproxBytes: 399, Sensitive: true, Rationale: "The safeguard writes Linux kernel hardening settings."},
	}
}

func TestSensitiveReviewLeadsWithWhatWouldBeLostAndOffersTheStandingRule(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	svc.report.Sensitive = realisticSensitive()
	askID, _, err := flow.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	payload := publisher.events[0].Payload
	question, _ := payload["question"].(string)
	subject, _ := payload["subject"].(string)
	if !strings.Contains(subject, "AI coding history has no backup") || !strings.Contains(question, "Your Claude Code history and 3 other places that hold secrets have no backup (2.1GiB)") || !strings.Contains(question, "new ones from now on") {
		t.Errorf("cover does not lead with what would be lost:\nsubject %q\nquestion %q", subject, question)
	}
	if problems := operatordecision.PayloadCoverProblems(payload); len(problems) > 0 {
		t.Errorf("a cold reader cannot parse the cover: %v", problems)
	}
	encoded, _ := json.Marshal(payload)
	for _, leak := range []string{"/home/u/", "/etc/sysctl.d"} {
		if strings.Contains(string(encoded), leak) {
			t.Errorf("decision leaked locator %q", leak)
		}
	}
	if got := payload["recommended"]; got != answerStanding {
		t.Errorf("recommended = %v, want the standing rule that ends the question", got)
	}
	keys := optionKeys(t, payload)
	for _, want := range []string{answerStanding, answerProtectAll, answerGroups, answerLater} {
		if !contains(keys, want) {
			t.Errorf("menu %v is missing %q", keys, want)
		}
	}
	if contains(keys, answerReacquireAll) {
		t.Errorf("menu %v offers reacquire-all although no owner declares how to get its data back; the callback would refuse it", keys)
	}
	questions, _ := payload["questions"].([]any)
	if len(questions) != 1 {
		t.Fatalf("payload carries %d form questions, want the groups question", len(questions))
	}
	big, _ := payload["big_picture"].(string)
	for _, want := range []string{"weekly note", "turn the rule off", "If the disk fails"} {
		if !strings.Contains(big, want) {
			t.Errorf("big picture lacks %q: %s", want, big)
		}
	}
	page, _ := payload["explainer_html"].(string)
	if len(page) > operatordecision.MaxExplainerBytes {
		t.Fatalf("page is %d bytes", len(page))
	}

	ask, _ := store.Get(ctx, askID)
	if handled, err := flow.Apply(ctx, humanResolution(ask, answerLater, "")); err != nil || !handled {
		t.Fatalf("Apply(later) = %v, %v", handled, err)
	}
	stored, _ := store.Get(ctx, askID)
	if len(svc.registered) != 0 || stored.State != operatordecision.PendingStateRejected || !strings.Contains(stored.Extra["result"], "not now") {
		t.Fatalf("not now must register nothing and say so: registered=%v result=%q", svc.registered, stored.Extra["result"])
	}
	if policy, _ := flow.Policy(ctx); policy.Active {
		t.Fatal("not now turned the standing rule on")
	}
}

func optionKeys(t *testing.T, payload map[string]any) []string {
	t.Helper()
	options, _ := payload["options"].([]any)
	keys := make([]string, 0, len(options))
	for _, option := range options {
		m, _ := option.(map[string]any)
		key, _ := m["key"].(string)
		keys = append(keys, key)
	}
	return keys
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func TestSensitiveDecisionFlow_AskedBeforeIsSaidOnTheCover(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	svc.report.Sensitive = realisticSensitive()
	for i, id := range []string{"dbm-sensitive-coverage-old-1", "dbm-sensitive-coverage-old-2"} {
		raised := time.Date(2026, 11, 13+i, 8, 0, 0, 0, time.UTC)
		if err := store.Put(ctx, &operatordecision.PendingAsk{ID: id, Subject: "sensitive-coverage", State: operatordecision.PendingStateRejected, RaisedAt: raised, Extra: map[string]string{}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := flow.Request(ctx); err != nil {
		t.Fatal(err)
	}
	if big, _ := publisher.events[0].Payload["big_picture"].(string); !strings.Contains(big, "third time") {
		t.Fatalf("a question asked twice before does not say so: %s", big)
	}
}

func TestSensitiveReviewFollowsPerItemRecommendationWhenOwnersCanReacquire(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	svc.report.Sensitive = append(realisticSensitive()[:1], coverage.Suggestion{ID: "model-cache", Owner: "ollama", OwnerKind: "resource", Name: "credentials", Sensitive: true, ReacquisitionCommand: "ollama signin", ReacquisitionEvidence: "docs"})
	askID, _, err := flow.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := publisher.events[0].Payload["recommended"]; got != answerAsRecommended {
		t.Fatalf("recommended = %v, want the per-item recommendation", got)
	}
	ask, _ := store.Get(ctx, askID)
	if handled, err := flow.Apply(ctx, humanResolution(ask, answerAsRecommended, "")); err != nil || !handled {
		t.Fatalf("Apply = %v, %v", handled, err)
	}
	if len(svc.registered) != 1 || svc.registered[0] != "a58721892ba534c6" {
		t.Fatalf("registered %v, want only the item that exists nowhere else", svc.registered)
	}
}

func resolveWith(t *testing.T, flow *SensitiveDecisionFlow, store *destinationreadiness.FileDecisionStore, askID, answer, note string) {
	t.Helper()
	resolveWithGroups(t, flow, store, askID, answer, note, nil)
}

func resolveWithGroups(t *testing.T, flow *SensitiveDecisionFlow, store *destinationreadiness.FileDecisionStore, askID, answer, note string, selected []string) {
	t.Helper()
	ask, err := store.Get(context.Background(), askID)
	if err != nil || ask == nil {
		t.Fatalf("get ask %s = %+v, %v", askID, ask, err)
	}
	res := humanResolution(ask, answer, note)
	res.AnsweredAt = "2026-11-15T09:30:00Z"
	if selected != nil {
		res.Responses = []operatordecision.Response{{QuestionID: groupsQuestionID, Selected: selected}}
	}
	if handled, err := flow.Apply(context.Background(), res); err != nil || !handled {
		t.Fatalf("Apply(%s) = %v, %v", answer, handled, err)
	}
}

func TestStandingAnswerRegistersEverythingAndRecordsWhoConsentedAndWhen(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	svc.report.Sensitive = realisticSensitive()
	askID, _, err := flow.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if policy, _ := flow.Policy(ctx); policy.Active {
		t.Fatal("the rule was on before anyone consented")
	}
	resolveWith(t, flow, store, askID, answerStanding, "")
	if len(svc.registered) != 4 {
		t.Fatalf("registered %v, want every current item", svc.registered)
	}
	policy, err := flow.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !policy.Active || policy.GrantAskID != askID || policy.GrantedBy != "operator@example.test" || policy.GrantedByKind != "verified_human" || policy.GrantedBySource != "notification-hub" {
		t.Fatalf("consent not recorded with who and which decision: %+v", policy)
	}
	if policy.GrantedAt == nil || !policy.GrantedAt.Equal(time.Date(2026, 11, 15, 9, 30, 0, 0, time.UTC)) {
		t.Fatalf("consent time = %v, want the answer time", policy.GrantedAt)
	}
	stored, _ := store.Get(ctx, askID)
	if !strings.Contains(stored.Extra["result"], "Standing rule on") || !strings.Contains(stored.Extra["result"], "revoke-sensitive-policy") {
		t.Fatalf("result does not say how the rule is turned off: %q", stored.Extra["result"])
	}
}

func TestStandingConsentNeedsAVerifiedHuman(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	askID, _, _ := flow.Request(ctx)
	ask, _ := store.Get(ctx, askID)
	res := humanResolution(ask, answerStanding, "")
	res.AnsweredByIdentityKind = "agent"
	if handled, err := flow.Apply(ctx, res); err != nil || !handled {
		t.Fatalf("Apply = %v, %v", handled, err)
	}
	if policy, _ := flow.Policy(ctx); policy.Active || len(svc.registered) != 0 {
		t.Fatalf("an agent turned the standing rule on: %+v registered=%v", policy, svc.registered)
	}
}

func TestStandingAnswerReplayChangesNothing(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	askID, _, _ := flow.Request(ctx)
	resolveWith(t, flow, store, askID, answerStanding, "")
	first, _ := flow.Policy(ctx)
	resolveWith(t, flow, store, askID, answerStanding, "")
	second, _ := flow.Policy(ctx)
	if len(svc.registered) != 2 || len(second.Ledger) != len(first.Ledger) {
		t.Fatalf("replay changed state: registered=%v ledger %d -> %d", svc.registered, len(first.Ledger), len(second.Ledger))
	}
}

func TestGroupAnswerBacksUpOnlyTheChosenGroups(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	svc.report.Sensitive = realisticSensitive()
	askID, _, _ := flow.Request(ctx)
	resolveWithGroups(t, flow, store, askID, answerGroups, "", []string{"tool", "resource", "scenario"})
	if len(svc.registered) != 3 {
		t.Fatalf("registered %v, want the three non-host-setting items", svc.registered)
	}
	for _, id := range svc.registered {
		if id == "f1d3160e7e889ab5" {
			t.Fatal("the host setting group was not chosen but was registered")
		}
	}
	stored, _ := store.Get(ctx, askID)
	var decisions []sensitiveChoice
	_ = json.Unmarshal([]byte(stored.Extra["decisions"]), &decisions)
	left := 0
	for _, d := range decisions {
		if d.Decision == "exclude" && d.Reason != "" {
			left++
		}
	}
	if left != 1 {
		t.Fatalf("decisions %+v, want the left-out group recorded as excluded with a reason", decisions)
	}
	if policy, _ := flow.Policy(ctx); policy.Active {
		t.Fatal("choosing groups turned the standing rule on")
	}
}

func TestGroupAnswerWithNothingSwitchedOnIsRefusedWithoutRegistering(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	svc.report.Sensitive = realisticSensitive()
	askID, _, _ := flow.Request(ctx)
	resolveWith(t, flow, store, askID, answerGroups, "")
	stored, _ := store.Get(ctx, askID)
	if len(svc.registered) != 0 || stored.State != operatordecision.PendingStateRejected || !strings.Contains(stored.Extra["result"], "no group was switched on") {
		t.Fatalf("registered=%v state=%s result=%q", svc.registered, stored.State, stored.Extra["result"])
	}
}

func TestGroupAnswerNamingNoCurrentGroupIsRefusedWithoutRegistering(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	svc.report.Sensitive = realisticSensitive()
	askID, _, _ := flow.Request(ctx)
	resolveWithGroups(t, flow, store, askID, answerGroups, "", []string{"nonsense"})
	stored, _ := store.Get(ctx, askID)
	if len(svc.registered) != 0 || stored.State != operatordecision.PendingStateRejected {
		t.Fatalf("registered=%v state=%s", svc.registered, stored.State)
	}
}

func TestStandingWithExceptionsKeepsExcludedItemsOutOfTheRule(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	askID, _, _ := flow.Request(ctx)
	note := `{"standing":true,"decisions":[{"suggestion_id":"secret-a","decision":"exclude","reason":"Too large"},{"suggestion_id":"secret-b","decision":"protect"}]}`
	resolveWith(t, flow, store, askID, "other", note)
	if len(svc.registered) != 1 || svc.registered[0] != "secret-b" {
		t.Fatalf("registered %v, want only the protected item", svc.registered)
	}
	policy, _ := flow.Policy(ctx)
	if !policy.Active || len(policy.Excluded) != 1 || policy.Excluded[0].Owner != "codex" || policy.Excluded[0].Reason != "Too large" {
		t.Fatalf("policy = %+v, want active with the exception recorded", policy)
	}
	// A new candidate arrives; the excluded one stays out and the new one is added.
	svc.report.Sensitive = append(svc.report.Sensitive, coverage.Suggestion{ID: "secret-c", Owner: "grok", OwnerKind: "resource", Name: "home", Sensitive: true, ApproxBytes: 10})
	swept, err := flow.Sweep(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range svc.registered[1:] {
		if id == "secret-a" {
			t.Fatal("the sweep added an item the operator excluded")
		}
	}
	if len(swept.Registered) == 0 || svc.registered[len(svc.registered)-1] != "secret-c" {
		t.Fatalf("the sweep did not add the new candidate: %+v registered=%v", swept.Registered, svc.registered)
	}
}

func TestSweepDoesNothingWithoutConsent(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, _ := sensitiveFlowFixture(t)
	result, err := flow.Sweep(ctx)
	if err != nil || result.Active || len(svc.registered) != 0 || len(publisher.events) != 0 {
		t.Fatalf("sweep = %+v, %v registered=%v events=%d; a secret was touched without consent", result, err, svc.registered, len(publisher.events))
	}
}

func grantStanding(t *testing.T, flow *SensitiveDecisionFlow, svc *sensitiveFlowService, store *destinationreadiness.FileDecisionStore) {
	t.Helper()
	svc.consume = true
	askID, _, err := flow.Request(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	resolveWith(t, flow, store, askID, answerStanding, "")
	svc.registered = nil
}

func TestSweepAddsNewCandidatesAndReportsThemInOneWeeklyNote(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	grantStanding(t, flow, svc, store)
	before := len(publisher.events)
	svc.report.Sensitive = append(svc.report.Sensitive, coverage.Suggestion{ID: "new-1", Owner: "grok", OwnerKind: "resource", Name: "home", Sensitive: true, ApproxBytes: 5 << 20})

	result, err := flow.Sweep(ctx)
	if err != nil || len(result.Registered) != 1 || result.NoteSent {
		t.Fatalf("first sweep = %+v, %v; want additions and no note yet", result, err)
	}
	if len(publisher.events) != before {
		t.Fatalf("a note went out before a week passed: %d events", len(publisher.events)-before)
	}
	if got := svc.registered[len(svc.registered)-1]; got != "new-1" {
		t.Fatalf("registered %v, want the new item", svc.registered)
	}

	now := flow.now()
	flow.now = func() time.Time { return now.Add(10 * 24 * time.Hour) }
	result, err = flow.Sweep(ctx)
	if err != nil || !result.NoteSent {
		t.Fatalf("sweep after a week = %+v, %v; want the note", result, err)
	}
	note := publisher.events[len(publisher.events)-1]
	if note.EventType != sensitiveNoteEvent || !strings.HasSuffix(note.EventType, ".notification.v1") {
		t.Fatalf("note event = %+v", note)
	}
	body, _ := note.Payload["body"].(string)
	if !strings.Contains(body, "Grok / home") || !strings.Contains(body, "revoke-sensitive-policy") || strings.Contains(body, "/secret/") {
		t.Fatalf("note body = %q", body)
	}
	if policy, _ := flow.Policy(ctx); len(policy.Unreported) != 0 || policy.AddedTotal == 0 {
		t.Fatalf("after the note: %+v", policy)
	}
	// A sweep with nothing new sends nothing.
	n := len(publisher.events)
	flow.now = func() time.Time { return now.Add(20 * 24 * time.Hour) }
	svc.report.Sensitive = nil
	if _, err := flow.Sweep(ctx); err != nil || len(publisher.events) != n {
		t.Fatalf("an empty sweep sent %d events (err %v)", len(publisher.events)-n, err)
	}
}

func TestRevokedRuleStopsAddingAndNewCandidatesAskAgain(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	grantStanding(t, flow, svc, store)
	if _, err := flow.Revoke(ctx, "operator via CLI"); err != nil {
		t.Fatal(err)
	}
	policy, _ := flow.Policy(ctx)
	if policy.Active || policy.RevokedBy != "operator via CLI" || policy.RevokedAt == nil || policy.GrantedBy == "" {
		t.Fatalf("revocation not recorded beside the grant: %+v", policy)
	}
	if _, err := flow.Revoke(ctx, ""); !errors.Is(err, ErrPolicyRequest) {
		t.Fatalf("revoking twice = %v, want a refused request", err)
	}
	svc.report.Sensitive = []coverage.Suggestion{{ID: "later-1", Owner: "grok", OwnerKind: "resource", Name: "home", Sensitive: true}}
	if result, _ := flow.Sweep(ctx); len(result.Registered) != 0 || len(svc.registered) != 0 {
		t.Fatalf("a revoked rule still added %v", svc.registered)
	}
	events := len(publisher.events)
	askID, count, err := flow.Request(ctx)
	if err != nil || askID == "" || count != 1 || len(publisher.events) != events+1 {
		t.Fatalf("after revoking, Request = %q, %d, %v; want a new question", askID, count, err)
	}
}

func TestRequestWithTheRuleOnAppliesItInsteadOfAsking(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	grantStanding(t, flow, svc, store)
	asks := len(publisher.events)
	svc.report.Sensitive = append(svc.report.Sensitive, coverage.Suggestion{ID: "new-2", Owner: "grok", OwnerKind: "resource", Name: "home", Sensitive: true})
	askID, count, err := flow.Request(ctx)
	if err != nil || askID != "" || count == 0 || len(publisher.events) != asks {
		t.Fatalf("Request = %q, %d, %v events+%d; want the rule applied and no ask", askID, count, err, len(publisher.events)-asks)
	}
}

func TestExcludeKeepsAnItemOutOfTheRuleAndRequiresAReason(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	grantStanding(t, flow, svc, store)
	if _, err := flow.Exclude(ctx, "grok", "home", "", ""); !errors.Is(err, ErrPolicyRequest) {
		t.Fatalf("exclusion without a reason = %v", err)
	}
	if _, err := flow.Exclude(ctx, "grok", "home", "Too large to copy daily", "operator via CLI"); err != nil {
		t.Fatal(err)
	}
	svc.report.Sensitive = []coverage.Suggestion{{ID: "grok-home", Owner: "grok", OwnerKind: "resource", Name: "home", Sensitive: true}}
	if _, err := flow.Sweep(ctx); err != nil || len(svc.registered) != 0 {
		t.Fatalf("an excluded item was added: %v (err %v)", svc.registered, err)
	}
}

func TestPolicyRPCsShowRevokeAndExcludeTheStandingRule(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	h := NewConnectHandler(Deps{Service: svc, Decisions: flow})
	got, err := h.GetSensitivePolicy(ctx, connect.NewRequest(&coveragev1.GetSensitivePolicyRequest{}))
	if err != nil || got.Msg.Policy.Active {
		t.Fatalf("before consent: %+v, %v", got, err)
	}
	if _, err := h.RevokeSensitivePolicy(ctx, connect.NewRequest(&coveragev1.RevokeSensitivePolicyRequest{})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("revoking a rule that was never granted = %v, want failed precondition", err)
	}
	grantStanding(t, flow, svc, store)
	got, _ = h.GetSensitivePolicy(ctx, connect.NewRequest(&coveragev1.GetSensitivePolicyRequest{}))
	p := got.Msg.Policy
	if !p.Active || p.GrantedBy != "operator@example.test" || p.GrantedAt == nil || p.GrantAskId == "" || len(p.History) == 0 {
		t.Fatalf("visible policy lacks the consent record: %+v", p)
	}
	if _, err := h.ExcludeSensitiveTarget(ctx, connect.NewRequest(&coveragev1.ExcludeSensitiveTargetRequest{Owner: "grok", Name: "home"})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("exclusion without a reason = %v", err)
	}
	ex, err := h.ExcludeSensitiveTarget(ctx, connect.NewRequest(&coveragev1.ExcludeSensitiveTargetRequest{Owner: "grok", Name: "home", Reason: "Too large"}))
	if err != nil || len(ex.Msg.Policy.Excluded) != 1 {
		t.Fatalf("exclude = %+v, %v", ex, err)
	}
	rev, err := h.RevokeSensitivePolicy(ctx, connect.NewRequest(&coveragev1.RevokeSensitivePolicyRequest{RevokedBy: "operator via CLI"}))
	if err != nil || rev.Msg.Policy.Active || rev.Msg.Policy.RevokedBy != "operator via CLI" || rev.Msg.Policy.GrantedBy == "" {
		t.Fatalf("revoke = %+v, %v", rev, err)
	}
}

func TestRequestSensitiveReviewWithTheRuleOnIsNotPending(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	grantStanding(t, flow, svc, store)
	svc.report.Sensitive = append(svc.report.Sensitive, coverage.Suggestion{ID: "n1", Owner: "grok", OwnerKind: "resource", Name: "home", Sensitive: true})
	h := NewConnectHandler(Deps{Service: svc, Decisions: flow})
	resp, err := h.RequestSensitiveReview(ctx, connect.NewRequest(&coveragev1.RequestSensitiveReviewRequest{}))
	if err != nil || resp.Msg.Pending || resp.Msg.AskId != "" || resp.Msg.CandidateCount != 1 {
		t.Fatalf("response = %+v, %v; want the rule applied to one new item and no pending ask", resp, err)
	}
}

func TestChangedCandidatesReviseTheOpenDecisionUnderTheSameSubjectKeyWithoutAWithdrawal(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	first, _, err := flow.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	svc.report.Sensitive = append(svc.report.Sensitive, coverage.Suggestion{ID: "secret-c", Owner: "grok", OwnerKind: "resource", Name: "home", Sensitive: true})
	second, count, err := flow.Request(ctx)
	if err != nil || second == first || count != 3 {
		t.Fatalf("second Request = %q, %d, %v", second, count, err)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("events %d; a revision is one request under the subject key, with no separate withdrawal that would leave no ask open", len(publisher.events))
	}
	for _, event := range publisher.events {
		if got := event.Payload["subject_key"]; got != "dbm-sensitive-coverage" {
			t.Errorf("event %s subject_key = %v", event.EventType, got)
		}
	}
	old, _ := store.Get(ctx, first)
	fresh, _ := store.Get(ctx, second)
	if old.State != operatordecision.PendingStateWithdrawn || fresh.State != operatordecision.PendingStatePending {
		t.Fatalf("old=%s new=%s", old.State, fresh.State)
	}
}

func TestFailedRevisionReopensTheStaleReview(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	first, _, _ := flow.Request(ctx)
	svc.report.Sensitive = append(svc.report.Sensitive, coverage.Suggestion{ID: "secret-c", Owner: "grok", Name: "home", Sensitive: true})
	publisher.failEvent, publisher.failures = operatordecision.DecisionRequestedEventType("data-backup-manager", sensitiveDecisionKind), 1
	if _, _, err := flow.Request(ctx); err == nil {
		t.Fatal("expected the failed publish to surface")
	}
	old, _ := store.Get(ctx, first)
	if old.State != operatordecision.PendingStatePending {
		t.Fatalf("the stale review stayed closed (%s) although nothing replaced it", old.State)
	}
}

func TestSignedCallbackCarriesTheGroupSelectionToTheFlow(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)
	svc.report.Sensitive = realisticSensitive()
	askID, _, err := flow.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ask, _ := store.Get(ctx, askID)
	resolution := humanResolution(ask, answerGroups, "")
	resolution.Responses = []operatordecision.Response{{QuestionID: groupsQuestionID, Selected: []string{"tool", "scenario"}}}
	body, _ := json.Marshal(map[string]any{"event_type": operatordecision.AskResolvedEventType, "payload": resolution})
	const secret = "signed-test-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(string(body)))
	req.Header.Set(operatordecision.SignatureHeader, hex.EncodeToString(mac.Sum(nil)))
	resp := httptest.NewRecorder()
	operatordecision.NewWebhook(operatordecision.WebhookConfig{Secret: operatordecision.StaticSecret(secret), Effects: []operatordecision.DomainEffect{flow}}).ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d %s", resp.Code, resp.Body.String())
	}
	if len(svc.registered) != 2 {
		t.Fatalf("registered %v, want the tool and scenario items only", svc.registered)
	}
}

type fakeAskReader struct {
	ask askreader.Ask
	err error
}

func (f fakeAskReader) ReadAsk(context.Context, string) (askreader.Ask, error) { return f.ask, f.err }

const operatorWords = "I\u2019m pretty sure data backup manager is meant to back up any data that declares storage targets the right way."

func answeredOther() askreader.Ask {
	return askreader.Ask{
		ID: "hub-ask-1", Source: "data-backup-manager", SubjectKey: sensitiveSubjectKey, State: operatordecision.StateAnswered,
		Answer: "other", Note: operatorWords, AnsweredBy: "d47163a3", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "notification-hub",
		AnsweredAt: "2026-11-15T22:16:43.287276843Z",
	}
}

func TestAdoptedFreeTextAnswerRecordsConsentVerbatimAppliesItAndClosesTheStaleReview(t *testing.T) {
	ctx := context.Background()
	flow, svc, publisher, store := sensitiveFlowFixture(t)
	svc.consume = true
	askID, _, err := flow.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	events := len(publisher.events)
	flow.WithAskReader(fakeAskReader{ask: answeredOther()})
	policy, swept, err := flow.AdoptAnswerAsConsent(ctx, "hub-ask-1")
	if err != nil {
		t.Fatal(err)
	}
	if !policy.Active || policy.GrantAskID != "hub-ask-1" || policy.GrantBasis != "free_text" || policy.GrantText != operatorWords ||
		policy.GrantedBy != "d47163a3" || policy.GrantedByKind != "verified_human" || policy.GrantedBySource != "notification-hub" ||
		policy.GrantedAt == nil || !policy.GrantedAt.Equal(time.Date(2026, 11, 15, 22, 16, 43, 287276843, time.UTC)) {
		t.Fatalf("consent record = %+v", policy)
	}
	if len(swept.Registered) != 2 || len(svc.registered) != 2 {
		t.Fatalf("sweep registered %v, want both declared candidates", svc.registered)
	}
	stale, _ := store.Get(ctx, askID)
	if stale.State == operatordecision.PendingStatePending {
		t.Fatalf("the stale local review still says pending")
	}
	for _, e := range publisher.events[events:] {
		if strings.HasSuffix(e.EventType, ".decision_requested.v1") {
			t.Fatalf("adopting the answer raised a new Hub ask: %s", e.EventType)
		}
	}
	if note := publisher.events[len(publisher.events)-1]; note.EventType != sensitiveNoteEvent {
		t.Fatalf("the first additions were not reported at once: %+v", note)
	}
}

func TestAdoptRefusesAnswersThatAreNotTheOperatorsOwnWords(t *testing.T) {
	ctx := context.Background()
	cases := map[string]func(a *askreader.Ask){
		"another scenario": func(a *askreader.Ask) { a.Source = "storage-manager" },
		"another subject":  func(a *askreader.Ask) { a.SubjectKey = "something-else" },
		"unanswered":       func(a *askreader.Ask) { a.State = "pending" },
		"agent identity":   func(a *askreader.Ask) { a.AnsweredByIdentityKind = "agent" },
		"menu answer":      func(a *askreader.Ask) { a.Answer = "protect_all" },
		"empty words":      func(a *askreader.Ask) { a.Note = "  " },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			flow, svc, _, _ := sensitiveFlowFixture(t)
			ask := answeredOther()
			mutate(&ask)
			flow.WithAskReader(fakeAskReader{ask: ask})
			_, _, err := flow.AdoptAnswerAsConsent(ctx, "hub-ask-1")
			if !errors.Is(err, ErrConsentNotAdoptable) {
				t.Fatalf("err = %v, want a refusal", err)
			}
			if policy, _ := flow.Policy(ctx); policy.Active || len(svc.registered) != 0 {
				t.Fatalf("a refused answer still changed state: %+v %v", policy, svc.registered)
			}
		})
	}
	flow, _, _, _ := sensitiveFlowFixture(t)
	if _, _, err := flow.AdoptAnswerAsConsent(ctx, "hub-ask-1"); err == nil {
		t.Fatal("adopting without a Hub reader succeeded")
	}
	flow.WithAskReader(fakeAskReader{err: askreader.ErrHubUnavailable})
	if _, _, err := flow.AdoptAnswerAsConsent(ctx, "hub-ask-1"); err == nil || errors.Is(err, ErrConsentNotAdoptable) {
		t.Fatalf("an unreadable Hub = %v, want an availability error, not a verdict on the answer", err)
	}
}

func TestAdoptTwiceChangesNothingAndRevokeStillWorks(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, _ := sensitiveFlowFixture(t)
	svc.consume = true
	flow.WithAskReader(fakeAskReader{ask: answeredOther()})
	if _, _, err := flow.AdoptAnswerAsConsent(ctx, "hub-ask-1"); err != nil {
		t.Fatal(err)
	}
	first, _ := flow.Policy(ctx)
	if _, _, err := flow.AdoptAnswerAsConsent(ctx, "hub-ask-1"); err != nil {
		t.Fatal(err)
	}
	second, _ := flow.Policy(ctx)
	if len(second.Ledger) != len(first.Ledger) {
		t.Fatalf("ledger %d -> %d", len(first.Ledger), len(second.Ledger))
	}
	if p, err := flow.Revoke(ctx, "operator via CLI"); err != nil || p.Active || p.GrantText != operatorWords {
		t.Fatalf("revoke = %+v, %v; the consent record must survive revocation", p, err)
	}
}

// "A target declared by a scenario or resource is registered and backed up by
// default": under the same consent, the sweep registers what owners declare even
// though it holds no secrets, so nobody has to remember `coverage accept-defaults`.
func TestSweepRegistersDeclaredTargetsUnderTheSameConsent(t *testing.T) {
	ctx := context.Background()
	flow, svc, _, store := sensitiveFlowFixture(t)

	svc.report.Recommended = []coverage.Suggestion{{ID: "decl-1", Owner: "research-desk", Name: "research_archive_rehearsals", ApproxBytes: 4 << 20}}
	if result, err := flow.Sweep(ctx); err != nil || len(svc.declared) != 0 || len(result.Registered) != 0 {
		t.Fatalf("without the operator's answer a sweep registered %v (%+v, %v)", svc.declared, result, err)
	}

	grantStanding(t, flow, svc, store)
	svc.declared = nil
	svc.report.Recommended = []coverage.Suggestion{{ID: "decl-1", Owner: "research-desk", Name: "research_archive_rehearsals", ApproxBytes: 4 << 20}}
	result, err := flow.Sweep(ctx)
	if err != nil || len(svc.declared) != 1 || svc.declared[0] != "decl-1" || len(result.Registered) != 1 {
		t.Fatalf("with the rule on, sweep = %+v, %v, declared=%v; want the declared target registered", result, err, svc.declared)
	}
	if got := result.Registered[0]; got.Group != declaredGroup || got.Name != "research_archive_rehearsals" {
		t.Fatalf("declared addition = %+v", got)
	}

	note := standingNote(result.Registered, flow.now())
	if strings.Contains(note.title, "hold secrets") || !strings.Contains(note.body, "declared") {
		t.Fatalf("a note about declared targets must not call them secrets: %q / %q", note.title, note.body)
	}
	mixed := standingNote([]PolicyAddition{result.Registered[0], {Owner: "grok", Name: "home", Group: "Resource", Bytes: 1}}, flow.now())
	if !strings.Contains(mixed.title, "1 hold secrets") {
		t.Fatalf("mixed note title = %q", mixed.title)
	}
}
