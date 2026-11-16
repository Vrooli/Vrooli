package conversations

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/owneridentity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
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
	challenge, err := h.AnswerRootWithPassword(ctx, req)
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
