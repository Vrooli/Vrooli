package conversations

import (
	"connectrpc.com/connect"
	"context"
	"encoding/json"
	"errors"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/owneridentity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"net/http"
	"net/http/httptest"
	health "notification-hub/handlers/health"
	conversation "notification-hub/internal/conversations"
	localdb "notification-hub/internal/database"
	delivery "notification-hub/internal/delivery"
	"notification-hub/internal/identity"
	integration "notification-hub/internal/integrationconfig"
	notification "notification-hub/internal/notifications"
	preference "notification-hub/internal/preferences"
	recipient "notification-hub/internal/recipients"
	routing "notification-hub/internal/routing"
	"notification-hub/internal/testutil/hubtest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	apiidentity "github.com/vrooli/api-core/identity"
	"notification-hub/internal/hub"
)

func TestRootPasswordRequiresExactBoundOwnerAndCapability(t *testing.T) {
	bindings := []authn.OwnerBinding{{Source: apiidentity.SourceScenarioAuthenticator, Subject: "account", CanonicalSubject: "owner", Scopes: []string{"*"}}}
	if !rootPasswordOwnerAllowed(bindings, "account", "owner") {
		t.Fatal("bound owner denied")
	}
	for _, tc := range []struct{ subject, recipient string }{{"other", "owner"}, {"account", "other"}, {"", "owner"}} {
		if rootPasswordOwnerAllowed(bindings, tc.subject, tc.recipient) {
			t.Fatal("unbound identity admitted")
		}
	}
	bindings[0].Scopes = []string{"notification-hub:read"}
	if rootPasswordOwnerAllowed(bindings, "account", "owner") {
		t.Fatal("unrelated scope admitted")
	}
	bindings[0].Scopes = []string{"*"}
	bindings[0].Source = apiidentity.SourceCloudflareAccess
	if rootPasswordOwnerAllowed(bindings, "account", "owner") {
		t.Fatal("wrong provider admitted")
	}
}

func TestRootPasswordDecisionBindsVersionSourceAndDeadline(t *testing.T) {
	now := time.Now().UTC()
	ask := hub.AskRecord{Source: "root-rebaseline", SourceEventType: "root.rebaseline.decision_requested.v1", State: hub.AskPending, RevisionCount: 3, Deadline: now.Add(time.Minute).Format(time.RFC3339Nano), ProviderDecisionContext: &hub.ProviderDecisionContext{SchemaVersion: 1, Scope: hub.ProviderDecisionScopeRootRebaseline}}
	if !rootPasswordDecisionEligible(ask, 3, now) {
		t.Fatal("current root decision denied")
	}
	if rootPasswordDecisionEligible(ask, 2, now) || rootPasswordDecisionEligible(ask, 3, now.Add(time.Minute)) {
		t.Fatal("stale or expired decision admitted")
	}
	ask.Source = "other"
	if rootPasswordDecisionEligible(ask, 3, now) {
		t.Fatal("other decision admitted")
	}
}

type fakeRootLogin struct {
	fail bool
	mfa  bool
}

func (f *fakeRootLogin) VerifyDecision(_ context.Context, _, password, _, _, _, _ string) (identity.DecisionProof, error) {
	if f.fail || password != "private-password" {
		return identity.DecisionProof{}, errors.New("private-password")
	}
	if f.mfa {
		return identity.DecisionProof{MFARequired: true, MFAChallenge: "challenge"}, nil
	}
	return identity.DecisionProof{Subject: "account"}, nil
}

type blockingRootLogin struct {
	entered chan struct{}
	release chan struct{}
}

func (f *blockingRootLogin) VerifyDecision(context.Context, string, string, string, string, string, string) (identity.DecisionProof, error) {
	close(f.entered)
	<-f.release
	return identity.DecisionProof{Subject: "account"}, nil
}

type fakeRootVerifier struct{}

func (fakeRootVerifier) Validate(_ context.Context, token string) (owneridentity.Identity, error) {
	if token != "private-token" {
		return owneridentity.Identity{}, errors.New("invalid")
	}
	return owneridentity.Identity{Subject: "account", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func TestRootPasswordAnswerPersistsOnceAndKeepsSecretsOutOfResponse(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	if err := database.EnsureSchemas(ctx, primary, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(health.Schema), database.SchemaProviderFunc(integration.Schema), database.SchemaProviderFunc(recipient.Schema), database.SchemaProviderFunc(notification.Schema), database.SchemaProviderFunc(preference.Schema), database.SchemaProviderFunc(routing.Schema), database.SchemaProviderFunc(delivery.Schema), database.SchemaProviderFunc(conversation.Schema)); err != nil {
		t.Fatal(err)
	}
	svc := hubtest.New(t, database.NewFromPrimary(primary), nil, nil)
	ask, _, err := svc.AskWithSpec(ctx, hub.AskSpec{Recipient: "owner", Title: "Rebaseline", Body: "Review", Question: "Approve?", Options: []hub.AskOption{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}}, Deadline: time.Now().Add(time.Hour), DeadlineSpecified: true, SensitivityLabel: "private", Source: "root-rebaseline", SourceEventType: "root.rebaseline.decision_requested.v1", IdempotencyKey: "test-root", ProviderDecisionContext: &hub.ProviderDecisionContext{SchemaVersion: 1, Scope: hub.ProviderDecisionScopeRootRebaseline, CanonicalTermsSHA256: strings.Repeat("a", 64)}})
	if err != nil {
		t.Fatal(err)
	}
	svc.SetDefaultRecipient("owner")
	reader := &handler{service: svc, verifier: fakeRootVerifier{}}
	headers := http.Header{"Authorization": []string{"Bearer private-token"}}
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[]`)
	if _, err := reader.reader(ctx, headers); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatal("unbound native account read recipient asks")
	}
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[{"source":"scenario_authenticator","subject":"account","canonical_subject":"owner","scopes":["notification-hub:write"]}]`)
	if _, err := reader.reader(ctx, headers); err != nil {
		t.Fatalf("bound owner denied: %v", err)
	}
	read := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/identity/root-decision-read", strings.NewReader(`{"askId":"`+ask.ID+`","email":"owner@example.test","password":"private-password"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		(&handler{service: svc, stepUp: &fakeRootLogin{}}).rootDecisionRead(w, req)
		return w
	}
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[]`)
	if w := read(); w.Code != 401 || strings.Contains(w.Body.String(), "Rebaseline") {
		t.Fatalf("unbound read: %d %s", w.Code, w.Body.String())
	}
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[{"source":"scenario_authenticator","subject":"account","canonical_subject":"owner","scopes":["root:rebaseline"]}]`)
	if w := read(); w.Code != 200 || !strings.Contains(w.Body.String(), "Rebaseline") {
		t.Fatalf("bound read: %d %s", w.Code, w.Body.String())
	}
	// Pause an answer after credential verification and before its transaction.
	// A committed revocation must win against that in-flight answer.
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[{"source":"cloudflare_access","subject":"cf","canonical_subject":"owner","scopes":["root:rebaseline"]}]`)
	generation, err := svc.RootEnrollmentGeneration(ctx, "owner", "cf")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.EnrollRootAccount(ctx, "owner", "cf", "account", generation); err != nil {
		t.Fatal(err)
	}
	if w := read(); w.Code != 200 {
		t.Fatalf("enrolled read: %d", w.Code)
	}
	// A revoke and re-enrollment while credentials are being verified must
	// not turn the old attempt into authority under the new generation.
	blocking := &blockingRootLogin{entered: make(chan struct{}), release: make(chan struct{})}
	verificationOutcome := make(chan error, 1)
	go func() {
		_, err := (&handler{service: svc, stepUp: blocking}).AnswerRootWithPassword(ctx, connect.NewRequest(&v1.AnswerRootWithPasswordRequest{AskId: ask.ID, Answer: "approve", Email: "owner@example.test", Password: "private-password"}))
		verificationOutcome <- err
	}()
	<-blocking.entered
	if err := svc.RevokeRootEnrollments(ctx, "owner", "cf"); err != nil {
		close(blocking.release)
		t.Fatal(err)
	}
	generation, err = svc.RootEnrollmentGeneration(ctx, "owner", "cf")
	if err != nil {
		close(blocking.release)
		t.Fatal(err)
	}
	if err := svc.EnrollRootAccount(ctx, "owner", "cf", "account", generation); err != nil {
		close(blocking.release)
		t.Fatal(err)
	}
	close(blocking.release)
	if err := <-verificationOutcome; connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("old proof survived reenrollment: %v", err)
	}
	mfa := &fakeRootLogin{mfa: true}
	mh := &handler{service: svc, stepUp: mfa}
	mr := connect.NewRequest(&v1.AnswerRootWithPasswordRequest{AskId: ask.ID, Answer: "approve", Email: "owner@example.test", Password: "private-password"})
	challenge, err := mh.AnswerRootWithPassword(ctx, mr)
	if err != nil || !challenge.Msg.GetMfaRequired() {
		t.Fatalf("MFA: %v", err)
	}
	if err := svc.RevokeRootEnrollments(ctx, "owner", "cf"); err != nil {
		t.Fatal(err)
	}
	generation, err = svc.RootEnrollmentGeneration(ctx, "owner", "cf")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.EnrollRootAccount(ctx, "owner", "cf", "account", generation); err != nil {
		t.Fatal(err)
	}
	mfa.mfa = false
	mr.Msg.MfaChallenge = challenge.Msg.GetMfaChallenge()
	if _, err := mh.AnswerRootWithPassword(ctx, mr); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("old MFA survived reenrollment: %v", err)
	}
	// A read challenge is also bound to the pre-revocation generation.
	readMFA := &fakeRootLogin{mfa: true}
	readMFAHandler := &handler{service: svc, stepUp: readMFA}
	readRequest := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/identity/root-decision-read", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		readMFAHandler.rootDecisionRead(w, req)
		return w
	}
	readChallengeResponse := readRequest(`{"askId":"` + ask.ID + `","email":"owner@example.test","password":"private-password"}`)
	var readChallenge struct {
		MFAChallenge string `json:"mfaChallenge"`
	}
	if err := json.Unmarshal(readChallengeResponse.Body.Bytes(), &readChallenge); err != nil || readChallenge.MFAChallenge == "" {
		t.Fatal("missing read MFA challenge")
	}
	if err := svc.RevokeRootEnrollments(ctx, "owner", "cf"); err != nil {
		t.Fatal(err)
	}
	generation, err = svc.RootEnrollmentGeneration(ctx, "owner", "cf")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.EnrollRootAccount(ctx, "owner", "cf", "account", generation); err != nil {
		t.Fatal(err)
	}
	readMFA.mfa = false
	readBody, _ := json.Marshal(map[string]string{"askId": ask.ID, "email": "owner@example.test", "password": "private-password", "mfaChallenge": readChallenge.MFAChallenge, "totpCode": "123456"})
	if w := readRequest(string(readBody)); w.Code != 401 {
		t.Fatalf("old read MFA survived reenrollment: %d", w.Code)
	}
	readReached, readRelease := make(chan struct{}), make(chan struct{})
	readOutcome := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/identity/root-decision-read", strings.NewReader(`{"askId":"`+ask.ID+`","email":"owner@example.test","password":"private-password"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		(&handler{service: svc, stepUp: &fakeRootLogin{}, beforeRootReadDisclosure: func() { close(readReached); <-readRelease }}).rootDecisionRead(w, req)
		readOutcome <- w
	}()
	<-readReached
	if err := svc.RevokeRootEnrollments(ctx, "owner", "cf"); err != nil {
		close(readRelease)
		t.Fatal(err)
	}
	generation, err = svc.RootEnrollmentGeneration(ctx, "owner", "cf")
	if err != nil {
		close(readRelease)
		t.Fatal(err)
	}
	if err := svc.EnrollRootAccount(ctx, "owner", "cf", "account", generation); err != nil {
		close(readRelease)
		t.Fatal(err)
	}
	close(readRelease)
	if w := <-readOutcome; w.Code != 401 || strings.Contains(w.Body.String(), "Rebaseline") {
		t.Fatalf("revoked in-flight read disclosed: %d %s", w.Code, w.Body.String())
	}
	generation, err = svc.RootEnrollmentGeneration(ctx, "owner", "cf")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.EnrollRootAccount(ctx, "owner", "cf", "account", generation); err != nil {
		t.Fatal(err)
	}
	reached, release, outcome := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	svc.SetBeforeAnswerCommitForTest(func() { close(reached); <-release })
	go func() {
		h := &handler{service: svc, stepUp: &fakeRootLogin{}}
		_, err := h.AnswerRootWithPassword(ctx, connect.NewRequest(&v1.AnswerRootWithPasswordRequest{AskId: ask.ID, Answer: "approve", Email: "owner@example.test", Password: "private-password"}))
		outcome <- err
	}()
	<-reached
	if err := svc.RevokeRootEnrollments(ctx, "owner", "cf"); err != nil {
		close(release)
		t.Fatal(err)
	}
	generation, err = svc.RootEnrollmentGeneration(ctx, "owner", "cf")
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	if err := svc.EnrollRootAccount(ctx, "owner", "cf", "account", generation); err != nil {
		close(release)
		t.Fatal(err)
	}
	close(release)
	if err := <-outcome; connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("revoked in-flight answer: %v", err)
	}
	svc.SetBeforeAnswerCommitForTest(nil)
	if err := svc.RevokeRootEnrollments(ctx, "owner", "cf"); err != nil {
		t.Fatal(err)
	}
	if w := read(); w.Code != 401 {
		t.Fatalf("revoked read: %d", w.Code)
	}
	if current, err := svc.GetAsk(ctx, ask.ID); err != nil || current.State != hub.AskPending {
		t.Fatalf("revoked answer changed ask: %+v %v", current, err)
	}
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[{"source":"scenario_authenticator","subject":"account","canonical_subject":"owner","scopes":["*"]}]`)
	f := &fakeRootLogin{fail: true}
	h := &handler{service: svc, verifier: fakeRootVerifier{}, stepUp: f}
	req := connect.NewRequest(&v1.AnswerRootWithPasswordRequest{AskId: ask.ID, Answer: "approve", Email: "owner@example.test", Password: "private-password"})
	_, err = h.AnswerRootWithPassword(ctx, req)
	if connect.CodeOf(err) != connect.CodeUnauthenticated || strings.Contains(err.Error(), "private-password") {
		t.Fatalf("credential failure exposed details: %v", err)
	}
	f.fail = false
	f.mfa = true
	challenge, err = h.AnswerRootWithPassword(ctx, req)
	if err != nil || !challenge.Msg.GetMfaRequired() {
		t.Fatalf("MFA challenge: %v", err)
	}
	current, err := svc.GetAsk(ctx, ask.ID)
	if err != nil || current.State != hub.AskPending {
		t.Fatalf("MFA resolved decision: %v", err)
	}
	f.mfa = false
	result, err := h.AnswerRootWithPassword(ctx, req)
	if err != nil || result.Msg.GetAnsweredAt() == "" || strings.Contains(result.Msg.String(), "private-") {
		t.Fatalf("answer: %v", err)
	}
	current, err = svc.GetAsk(ctx, ask.ID)
	if err != nil || current.Answer != "approve" || current.AnsweredByIdentitySource != hub.AnswerIdentitySourcePassword {
		t.Fatalf("answer not retained: %+v %v", current, err)
	}
	if _, err = h.AnswerRootWithPassword(ctx, req); err == nil {
		t.Fatal("replayed answer admitted")
	}
	if _, _, err := svc.ConsumeRootRebaseline(ctx, "test-root", strings.Repeat("a", 64)); err != nil {
		t.Fatalf("fresh password approval could not be consumed: %v", err)
	}
	if _, _, err := svc.ConsumeRootRebaseline(ctx, "test-root", strings.Repeat("a", 64)); err == nil {
		t.Fatal("password approval was consumed twice")
	}
}

func TestRootPasswordDigestBindsEachDecisionField(t *testing.T) {
	base := rootPasswordDigest("ask-1", 1, "approve")
	for _, other := range []string{rootPasswordDigest("ask-2", 1, "approve"), rootPasswordDigest("ask-1", 2, "approve"), rootPasswordDigest("ask-1", 1, "reject")} {
		if base == other {
			t.Fatal("distinct decision reused proof audience")
		}
	}
}

func TestRootReadRefusesUnavailableEvidence(t *testing.T) {
	if !rootReadComplete(hub.AskRecord{}, 0) {
		t.Fatal("simple decision refused")
	}
	if rootReadComplete(hub.AskRecord{}, 1) || rootReadComplete(hub.AskRecord{AttachmentCount: 1}, 0) || rootReadComplete(hub.AskRecord{Questions: []hub.AskQuestion{{ID: "q"}}}, 0) {
		t.Fatal("partial decision admitted")
	}
}
