package hub_test

import (
	"context"
	"strings"
	"testing"
	"time"

	hub "notification-hub/internal/hub"

	"github.com/stretchr/testify/require"
)

func TestProviderDecisionContextIsTypedAndIdempotencyBound(t *testing.T) {
	service, _ := testService(t)
	now := time.Date(2026, time.November, 1, 12, 0, 0, 0, time.UTC)
	digest := strings.Repeat("a", 64)
	spec := decisionSpec(now, "provider-decision-1")
	spec.ProviderDecisionContext = &hub.ProviderDecisionContext{
		SchemaVersion: 1, CanonicalTermsSHA256: digest, Scope: hub.ProviderDecisionScopeRule,
	}

	first, _, err := service.AskWithSpec(context.Background(), spec)
	require.NoError(t, err)
	got, err := service.GetAsk(context.Background(), first.ID)
	require.NoError(t, err)
	require.Equal(t, spec.ProviderDecisionContext, got.ProviderDecisionContext)

	retry, _, err := service.AskWithSpec(context.Background(), spec)
	require.NoError(t, err)
	require.Equal(t, first.ID, retry.ID)

	changed := spec
	changed.ProviderDecisionContext = &hub.ProviderDecisionContext{
		SchemaVersion: 1, CanonicalTermsSHA256: strings.Repeat("b", 64), Scope: hub.ProviderDecisionScopeRule,
	}
	_, _, err = service.AskWithSpec(context.Background(), changed)
	require.ErrorIs(t, err, hub.ErrAskIdentityConflict, "a stable requester ref cannot be rebound to changed provider terms")
}

func TestProviderDecisionContextRejectsUnboundOrUnknownScope(t *testing.T) {
	service, _ := testService(t)
	now := time.Now().UTC()
	for name, decisionContext := range map[string]*hub.ProviderDecisionContext{
		"wrong version": {SchemaVersion: 2, CanonicalTermsSHA256: strings.Repeat("a", 64), Scope: hub.ProviderDecisionScopeRule},
		"bad digest":    {SchemaVersion: 1, CanonicalTermsSHA256: "terms", Scope: hub.ProviderDecisionScopeRule},
		"unknown scope": {SchemaVersion: 1, CanonicalTermsSHA256: strings.Repeat("a", 64), Scope: "supplier"},
	} {
		spec := decisionSpec(now, "provider-decision-"+name)
		spec.ProviderDecisionContext = decisionContext
		_, _, err := service.AskWithSpec(context.Background(), spec)
		require.ErrorIs(t, err, hub.ErrInvalidArgument, name)
	}
	reservedKey := decisionSpec(now, "provider-decision-reserved-key")
	reservedKey.Correlation = map[string]string{"notification_hub.provider_decision_context.v1": "forged"}
	_, _, err := service.AskWithSpec(context.Background(), reservedKey)
	require.ErrorIs(t, err, hub.ErrInvalidArgument, "requester correlations cannot impersonate the typed context")
}

func TestProviderDecisionHumanAttributionFailsClosedOnRecipientFallback(t *testing.T) {
	service, _ := testService(t)
	now := time.Date(2026, time.November, 1, 12, 0, 0, 0, time.UTC)
	ask, _, err := service.AskWithSpec(context.Background(), decisionSpec(now, "provider-answer-fallback"))
	require.NoError(t, err)

	answered, err := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: ask.ID, Answer: "close", Actor: "alice"})
	require.NoError(t, err)
	require.Equal(t, "alice", answered.AnsweredBy, "legacy owner fallback remains attributable to recipient")
	require.Equal(t, "unknown", answered.AnsweredByIdentityKind, "recipient fallback must not be presented as verified human")
	require.Empty(t, answered.AnsweredByIdentitySource)
}

func TestProviderDecisionHumanAttributionPersistsAuthenticatorSource(t *testing.T) {
	service, _ := testService(t)
	now := time.Date(2026, time.November, 1, 12, 0, 0, 0, time.UTC)
	ask, _, err := service.AskWithSpec(context.Background(), decisionSpec(now, "provider-answer-human"))
	require.NoError(t, err)

	answered, err := service.AnswerAsk(context.Background(), hub.AnswerInput{
		AskID: ask.ID, Answer: "close", Actor: "alice", AnsweredBy: "cf-user-1",
		AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
	})
	require.NoError(t, err)
	require.Equal(t, "cf-user-1", answered.AnsweredBy)
	require.Equal(t, "verified_human", answered.AnsweredByIdentityKind)
	require.Equal(t, "cloudflare_access", answered.AnsweredByIdentitySource)
}
