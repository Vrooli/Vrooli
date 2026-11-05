package routing_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/routing"
)

const (
	ollamaGenerateCmd     = "resource-ollama gateway generate --role chat.default --json --prompt-stdin --max-tokens 64"
	openRouterGenerateCmd = "resource-openrouter generate --role chat.default --json --max-tokens 64"
)

func candidateByProvider(candidates []*routingv1.RouteCandidate, provider string) *routingv1.RouteCandidate {
	for _, c := range candidates {
		if c.GetProvider() == provider {
			return c
		}
	}
	return nil
}

func TestExecuteOpensBreakerAfterFailureAndFallsBackToHealthyProvider(t *testing.T) { // [REQ:AIGW-PROVIDER-BREAKER]
	runner := roleRunner()
	runner.Errors[ollamaGenerateCmd] = &providers.CommandError{Code: "timeout", Command: "resource-ollama"}
	runner.Results[openRouterGenerateCmd] = providers.Result{Stdout: `{"response":"ok"}`}

	db := newSchemaDB(t)
	clock := breakerBaseTime
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(db),
		routing.WithHealth(routing.NewSQLHealthRepository(db)),
		routing.WithBreakerPolicy(routing.BreakerPolicy{FailureThreshold: 1, Cooldown: 30 * time.Second}),
		routing.WithClock(func() time.Time { return clock }))
	ctx := context.Background()

	resp, err := svc.Execute(ctx, baseRequest(sharedv1.Profile_PROFILE_LOCAL_FIRST), "hello")
	require.NoError(t, err)
	require.True(t, resp.GetValid())
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Equal(t, "openrouter", resp.GetEvidence().GetSelectedProvider(), "healthy remote fallback is selected when local fails")
	require.True(t, resp.GetEvidence().GetFallbackUsed())
	require.Equal(t, "timeout", resp.GetEvidence().GetFailureClass(), "successful fallback preserves the primary failure class")

	// The failed local provider now has an open breaker; the healthy remote
	// provider is unaffected (provider isolation).
	repo := routing.NewSQLHealthRepository(db)
	ollama, found, err := repo.Get(ctx, routing.HealthKey{Provider: "ollama", Role: "chat.default", Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION})
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, routing.BreakerOpen, ollama.State)
	_, foundRemote, err := repo.Get(ctx, routing.HealthKey{Provider: "openrouter", Role: "chat.default", Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION})
	require.NoError(t, err)
	require.True(t, foundRemote, "the successful provider records a closed health row")

	// Preview now skips the open-breaker local candidate and selects remote.
	prev, err := svc.Preview(ctx, baseRequest(sharedv1.Profile_PROFILE_LOCAL_FIRST))
	require.NoError(t, err)
	require.Equal(t, "openrouter", prev.GetSelectedProvider())
	ollamaCand := candidateByProvider(prev.GetCandidates(), "ollama")
	require.NotNil(t, ollamaCand)
	require.Equal(t, string(routing.BreakerOpen), ollamaCand.GetBreakerState())
	require.False(t, ollamaCand.GetSelected())
	require.False(t, ollamaCand.GetFallbackEligible())
}

func TestBreakerIsolatesByRequestKind(t *testing.T) { // [REQ:AIGW-PROVIDER-BREAKER]
	runner := roleRunner()
	db := newSchemaDB(t)
	repo := routing.NewSQLHealthRepository(db)
	ctx := context.Background()
	// Open the breaker for embeddings only.
	require.NoError(t, repo.Upsert(ctx, routing.ProviderHealth{
		Provider:      "ollama",
		Role:          "chat.default",
		Kind:          sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING,
		State:         routing.BreakerOpen,
		CooldownUntil: breakerBaseTime.Add(time.Hour),
		UpdatedAt:     breakerBaseTime,
	}))

	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(db),
		routing.WithHealth(repo),
		routing.WithClock(func() time.Time { return breakerBaseTime }))

	// A text-generation preview is unaffected by the embedding breaker.
	prev, err := svc.Preview(ctx, baseRequest(sharedv1.Profile_PROFILE_LOCAL_FIRST))
	require.NoError(t, err)
	require.Equal(t, "ollama", prev.GetSelectedProvider())
	ollamaCand := candidateByProvider(prev.GetCandidates(), "ollama")
	require.NotNil(t, ollamaCand)
	require.Equal(t, string(routing.BreakerClosed), ollamaCand.GetBreakerState())
}

func TestListProviderHealthReportsEffectiveState(t *testing.T) { // [REQ:AIGW-PROVIDER-BREAKER]
	db := newSchemaDB(t)
	repo := routing.NewSQLHealthRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Upsert(ctx, routing.ProviderHealth{
		Provider:      "ollama",
		Role:          "chat.default",
		Kind:          sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		State:         routing.BreakerOpen,
		CooldownUntil: breakerBaseTime, // elapsed at read time below
		UpdatedAt:     breakerBaseTime,
	}))
	svc := routing.NewService(testAdapters(roleRunner()), routing.NewSQLRepository(db),
		routing.WithHealth(repo),
		routing.WithClock(func() time.Time { return breakerBaseTime.Add(time.Minute) }))

	items, err := svc.ListProviderHealth(ctx)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "ollama", items[0].GetProvider())
	require.Equal(t, string(routing.BreakerOpen), items[0].GetState(), "stored state is open")
	require.Equal(t, string(routing.BreakerHalfOpen), items[0].GetEffectiveState(), "cooldown elapsed → effective half-open")
}

func TestListProviderHealthEmptyWhenDisabled(t *testing.T) { // [REQ:AIGW-PROVIDER-BREAKER]
	svc := routing.NewService(testAdapters(roleRunner()), nil)
	items, err := svc.ListProviderHealth(context.Background())
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestHalfOpenProbeSuccessClosesBreaker(t *testing.T) { // [REQ:AIGW-PROVIDER-BREAKER]
	runner := roleRunner()
	runner.Results[ollamaGenerateCmd] = providers.Result{Stdout: `{"response":"ok"}`}
	db := newSchemaDB(t)
	repo := routing.NewSQLHealthRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Upsert(ctx, routing.ProviderHealth{
		Provider:            "ollama",
		Role:                "chat.default",
		Kind:                sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		State:               routing.BreakerOpen,
		ConsecutiveFailures: 1,
		CooldownUntil:       breakerBaseTime, // elapsed at probe time below
		OpenedAt:            breakerBaseTime.Add(-30 * time.Second),
		Generation:          1,
		UpdatedAt:           breakerBaseTime,
	}))

	probeTime := breakerBaseTime.Add(time.Second)
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(db),
		routing.WithHealth(repo),
		routing.WithClock(func() time.Time { return probeTime }))

	// Preview surfaces the half-open probe eligibility.
	prev, err := svc.Preview(ctx, baseRequest(sharedv1.Profile_PROFILE_LOCAL_FIRST))
	require.NoError(t, err)
	ollamaCand := candidateByProvider(prev.GetCandidates(), "ollama")
	require.NotNil(t, ollamaCand)
	require.Equal(t, string(routing.BreakerHalfOpen), ollamaCand.GetBreakerState())
	require.True(t, ollamaCand.GetHalfOpenProbe())
	require.Equal(t, "ollama", prev.GetSelectedProvider(), "half-open candidate remains selectable as a probe")

	// A successful probe execution closes the breaker.
	resp, err := svc.Execute(ctx, baseRequest(sharedv1.Profile_PROFILE_LOCAL_FIRST), "hello")
	require.NoError(t, err)
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	got, found, err := repo.Get(ctx, routing.HealthKey{Provider: "ollama", Role: "chat.default", Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION})
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, routing.BreakerClosed, got.State)
	require.Equal(t, 0, got.ConsecutiveFailures)
}
