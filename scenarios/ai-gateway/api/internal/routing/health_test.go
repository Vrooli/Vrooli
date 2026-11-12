package routing_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/routing"
)

var breakerBaseTime = time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)

func mustBreakerFailure(t *testing.T, breaker routing.Breaker, health routing.ProviderHealth, class routing.FailureClass, now time.Time) routing.ProviderHealth {
	t.Helper()
	updated, err := breaker.OnFailure(health, class, now)
	require.NoError(t, err)
	return updated
}

func mustBreakerSuccess(t *testing.T, breaker routing.Breaker, health routing.ProviderHealth, now time.Time) routing.ProviderHealth {
	t.Helper()
	updated, err := breaker.OnSuccess(health, now)
	require.NoError(t, err)
	return updated
}

func mustProviderFailure(t *testing.T, breaker routing.Breaker, health routing.ProviderHealth, failure routing.ProviderFailure, now time.Time) routing.ProviderHealth {
	t.Helper()
	updated, err := breaker.OnProviderFailure(health, failure, now)
	require.NoError(t, err)
	return updated
}

func TestProviderHealthStateTransitionGraph(t *testing.T) {
	statuses := []routing.BreakerState{routing.BreakerClosed, routing.BreakerOpen, routing.BreakerHalfOpen}
	allowed := map[routing.BreakerState]map[routing.BreakerState]bool{
		routing.BreakerClosed:   {routing.BreakerClosed: true, routing.BreakerOpen: true},
		routing.BreakerOpen:     {routing.BreakerOpen: true, routing.BreakerClosed: true},
		routing.BreakerHalfOpen: {routing.BreakerHalfOpen: true, routing.BreakerOpen: true, routing.BreakerClosed: true},
	}
	for _, from := range statuses {
		for _, to := range statuses {
			t.Run(string(from)+"-to-"+string(to), func(t *testing.T) {
				health := routing.ProviderHealth{State: from}
				updated, err := health.TransitionState(to)
				if allowed[from][to] {
					require.NoError(t, err)
					require.Equal(t, to, updated.State)
					return
				}
				require.ErrorIs(t, err, routing.ErrInvalidBreakerStateTransition)
				require.Equal(t, from, updated.State)
			})
		}
	}
	for _, test := range []struct {
		name string
		from routing.BreakerState
		to   routing.BreakerState
	}{
		{name: "unknown current", from: "corrupt", to: routing.BreakerClosed},
		{name: "unknown next", from: routing.BreakerClosed, to: "corrupt"},
	} {
		t.Run(test.name, func(t *testing.T) {
			health := routing.ProviderHealth{State: test.from}
			updated, err := health.TransitionState(test.to)
			require.ErrorIs(t, err, routing.ErrInvalidBreakerStateTransition)
			require.Equal(t, test.from, updated.State)
		})
	}
}

func TestBreakerOpensAfterConsecutiveFailures(t *testing.T) { // [REQ:AIGW-PROVIDER-BREAKER]
	b := routing.NewBreaker(routing.BreakerPolicy{FailureThreshold: 3, Cooldown: 30 * time.Second})
	h := routing.ProviderHealth{Provider: "ollama", Role: "chat.default", Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION, State: routing.BreakerClosed}

	h = mustBreakerFailure(t, b, h, routing.FailureTimeout, breakerBaseTime)
	require.Equal(t, routing.BreakerClosed, b.Effective(h, breakerBaseTime), "one failure must not open")
	h = mustBreakerFailure(t, b, h, routing.FailureTimeout, breakerBaseTime)
	require.Equal(t, routing.BreakerClosed, b.Effective(h, breakerBaseTime), "two failures must not open")
	h = mustBreakerFailure(t, b, h, routing.FailureExecution, breakerBaseTime)
	require.Equal(t, routing.BreakerOpen, b.Effective(h, breakerBaseTime), "threshold failure opens breaker")
	require.Equal(t, 3, h.ConsecutiveFailures)
	require.Equal(t, routing.FailureExecution, h.LastFailureClass)
	require.Equal(t, breakerBaseTime.Add(30*time.Second), h.CooldownUntil)
}

func TestBreakerSuccessResetsBeforeThreshold(t *testing.T) { // [REQ:AIGW-PROVIDER-BREAKER]
	b := routing.NewBreaker(routing.BreakerPolicy{FailureThreshold: 3, Cooldown: 30 * time.Second})
	h := routing.ProviderHealth{Provider: "ollama", Role: "chat.default", State: routing.BreakerClosed}
	h = mustBreakerFailure(t, b, h, routing.FailureTimeout, breakerBaseTime)
	h = mustBreakerFailure(t, b, h, routing.FailureTimeout, breakerBaseTime)
	h = mustBreakerSuccess(t, b, h, breakerBaseTime)
	require.Equal(t, 0, h.ConsecutiveFailures)
	require.Equal(t, routing.BreakerClosed, b.Effective(h, breakerBaseTime))

	h = mustBreakerFailure(t, b, h, routing.FailureTimeout, breakerBaseTime)
	require.Equal(t, routing.BreakerClosed, b.Effective(h, breakerBaseTime), "counter reset means a single new failure stays closed")
}

func TestBreakerCooldownSurfacesHalfOpen(t *testing.T) { // [REQ:AIGW-PROVIDER-BREAKER]
	b := routing.NewBreaker(routing.BreakerPolicy{FailureThreshold: 1, Cooldown: 30 * time.Second})
	h := routing.ProviderHealth{Provider: "ollama", Role: "chat.default", State: routing.BreakerClosed}
	h = mustBreakerFailure(t, b, h, routing.FailureUnavailable, breakerBaseTime)
	require.Equal(t, routing.BreakerOpen, b.Effective(h, breakerBaseTime))
	require.Equal(t, routing.BreakerOpen, b.Effective(h, breakerBaseTime.Add(29*time.Second)), "still open before cooldown elapses")
	require.Equal(t, routing.BreakerHalfOpen, b.Effective(h, breakerBaseTime.Add(30*time.Second)), "cooldown elapsed surfaces half-open")
}

func TestBreakerHalfOpenSuccessCloses(t *testing.T) { // [REQ:AIGW-PROVIDER-BREAKER]
	b := routing.NewBreaker(routing.BreakerPolicy{FailureThreshold: 1, Cooldown: 30 * time.Second})
	h := routing.ProviderHealth{Provider: "ollama", Role: "chat.default", State: routing.BreakerClosed}
	h = mustBreakerFailure(t, b, h, routing.FailureExecution, breakerBaseTime)
	probeAt := breakerBaseTime.Add(31 * time.Second)
	require.Equal(t, routing.BreakerHalfOpen, b.Effective(h, probeAt))
	h = mustBreakerSuccess(t, b, h, probeAt)
	require.Equal(t, routing.BreakerClosed, b.Effective(h, probeAt))
	require.Equal(t, 0, h.ConsecutiveFailures)
	require.True(t, h.CooldownUntil.IsZero())
}

func TestBreakerHalfOpenFailureReopens(t *testing.T) { // [REQ:AIGW-PROVIDER-BREAKER]
	b := routing.NewBreaker(routing.BreakerPolicy{FailureThreshold: 1, Cooldown: 30 * time.Second})
	h := routing.ProviderHealth{Provider: "ollama", Role: "chat.default", State: routing.BreakerClosed}
	h = mustBreakerFailure(t, b, h, routing.FailureExecution, breakerBaseTime)
	probeAt := breakerBaseTime.Add(31 * time.Second)
	require.Equal(t, routing.BreakerHalfOpen, b.Effective(h, probeAt))
	h = mustBreakerFailure(t, b, h, routing.FailureTimeout, probeAt)
	require.Equal(t, routing.BreakerOpen, b.Effective(h, probeAt), "half-open failure reopens immediately")
	require.Equal(t, probeAt.Add(30*time.Second), h.CooldownUntil, "cooldown extends from the probe failure")
	require.Equal(t, int64(2), h.Generation, "reopen bumps generation")
}

func TestClassifyProviderError(t *testing.T) { // [REQ:AIGW-PROVIDER-BREAKER]
	cases := []struct {
		name string
		err  error
		want routing.FailureClass
	}{
		{"nil", nil, routing.FailureNone},
		{"canceled", context.Canceled, routing.FailureCancellation},
		{"deadline", context.DeadlineExceeded, routing.FailureTimeout},
		{"missing_binary", &providers.CommandError{Code: "missing_binary"}, routing.FailureMissingBinary},
		{"timeout", &providers.CommandError{Code: "timeout"}, routing.FailureTimeout},
		{"malformed", &providers.CommandError{Code: "malformed_json"}, routing.FailureMalformedJSON},
		{"exit", &providers.CommandError{Code: "exit_error"}, routing.FailureExecution},
		{"policy", &providers.CommandError{Code: "unsupported_kind"}, routing.FailurePolicyError},
		{"unavailable", &providers.CommandError{Code: "unavailable"}, routing.FailureUnavailable},
		{"unknown", errors.New("boom"), routing.FailureExecution},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, routing.ClassifyProviderError(tc.err))
		})
	}
}

func TestClassifyProviderFailurePreservesObservedRecoveryWindow(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantClass  routing.FailureClass
		wantRetry  string
		wantReset  string
		wantSource string
		wantWindow bool
	}{
		{
			name:       "metered insufficient credits is an account condition",
			err:        &providers.CommandError{Code: "insufficient_credits", HTTPStatus: 402},
			wantClass:  routing.FailureInsufficientCredits,
			wantSource: "http:402",
		},
		{
			name:       "metered rate limit keeps the observed retry-after",
			err:        &providers.CommandError{Code: "rate_limited", HTTPStatus: 429, RetryAfter: "45"},
			wantClass:  routing.FailureRateLimited,
			wantRetry:  "45",
			wantSource: "http:429",
			wantWindow: true,
		},
		{
			name:       "metered overload keeps the observed reset",
			err:        &providers.CommandError{Code: "provider_overloaded", HTTPStatus: 503, ResetAt: "2026-09-13T21:00:00Z"},
			wantClass:  routing.FailureProviderOverloaded,
			wantReset:  "2026-09-13T21:00:00Z",
			wantSource: "http:503",
			wantWindow: true,
		},
		{
			name:       "raw sentinel still classifies as insufficient credits",
			err:        providers.ErrInsufficientCredits,
			wantClass:  routing.FailureInsufficientCredits,
			wantSource: "provider-command",
		},
		{
			name:       "generic execution error stays generic",
			err:        &providers.CommandError{Code: "exit_error"},
			wantClass:  routing.FailureExecution,
			wantSource: "provider-command",
		},
		{
			name:       "openrouter stream failure after 200 is transient",
			err:        &providers.CommandError{Code: "stream_failed", HTTPStatus: 200},
			wantClass:  routing.FailureProviderOverloaded,
			wantSource: "http:200",
		},
		{
			name:       "openrouter transport failure is unavailable",
			err:        &providers.CommandError{Code: "unreachable"},
			wantClass:  routing.FailureUnavailable,
			wantSource: "provider-command",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := routing.ClassifyProviderFailure(tc.err)
			require.Equal(t, tc.wantClass, got.Class)
			require.Equal(t, tc.wantRetry, got.RetryAfter)
			require.Equal(t, tc.wantReset, got.ResetAt)
			require.Equal(t, tc.wantSource, got.Source)
			require.Equal(t, tc.wantWindow, got.HasRecoveryWindow())
			require.Equal(t, got.Class, routing.ClassifyProviderError(tc.err), "class-only helper must agree")
		})
	}
}

func TestClassifyProviderErrorUnchangedForLegacyCodes(t *testing.T) {
	require.Equal(t, routing.FailureExecution, routing.ClassifyProviderError(&providers.CommandError{Code: "provider_failed"}))
	require.Equal(t, routing.FailureExecution, routing.ClassifyProviderError(errors.New("boom")))
}

func TestBreakerOnProviderFailureRecordsObservedProvenance(t *testing.T) {
	b := routing.NewBreaker(routing.BreakerPolicy{FailureThreshold: 1, Cooldown: 30 * time.Second})
	h := routing.ProviderHealth{Provider: "openrouter", Role: "code.flatrate", State: routing.BreakerClosed}

	h = mustProviderFailure(t, b, h, routing.ProviderFailure{
		Class:      routing.FailureRateLimited,
		HTTPStatus: 429,
		RetryAfter: "45",
		Source:     "http:429",
	}, breakerBaseTime)

	require.Equal(t, routing.FailureRateLimited, h.LastFailureClass)
	require.Equal(t, 429, h.HTTPStatus)
	require.Equal(t, "45", h.RetryAfter)
	require.Empty(t, h.ResetAt)
	require.Equal(t, "http:429", h.FailureSource)
	require.Equal(t, routing.BreakerOpen, b.Effective(h, breakerBaseTime))

	// A later failure with no observed window clears the previous provenance
	// rather than leaving a stale recovery window attached to the new class.
	h = mustProviderFailure(t, b, h, routing.ProviderFailure{
		Class:  routing.FailureExecution,
		Source: "provider-command",
	}, breakerBaseTime.Add(time.Minute))
	require.Equal(t, routing.FailureExecution, h.LastFailureClass)
	require.Zero(t, h.HTTPStatus)
	require.Empty(t, h.RetryAfter)
	require.Empty(t, h.ResetAt)
	require.Equal(t, "provider-command", h.FailureSource)
}

func TestSQLHealthRepositoryRoundTrip(t *testing.T) { // [REQ:AIGW-PROVIDER-BREAKER]
	db := newSchemaDB(t)
	repo := routing.NewSQLHealthRepository(db)
	ctx := context.Background()
	key := routing.HealthKey{Provider: "ollama", Role: "chat.default", Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION}

	_, found, err := repo.Get(ctx, key)
	require.NoError(t, err)
	require.False(t, found, "missing record is not an error")

	rec := routing.ProviderHealth{
		Provider:            "ollama",
		Role:                "chat.default",
		Kind:                sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		State:               routing.BreakerOpen,
		ConsecutiveFailures: 3,
		LastFailureClass:    routing.FailureRateLimited,
		HTTPStatus:          429,
		RetryAfter:          "45",
		ResetAt:             "2026-07-06T12:01:00Z",
		FailureSource:       "http:429",
		LastFailureAt:       breakerBaseTime,
		CooldownUntil:       breakerBaseTime.Add(30 * time.Second),
		OpenedAt:            breakerBaseTime,
		Generation:          1,
		UpdatedAt:           breakerBaseTime,
	}
	require.NoError(t, repo.Upsert(ctx, rec))

	got, found, err := repo.Get(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, routing.BreakerOpen, got.State)
	require.Equal(t, 3, got.ConsecutiveFailures)
	require.Equal(t, routing.FailureRateLimited, got.LastFailureClass)
	require.Equal(t, 429, got.HTTPStatus)
	require.Equal(t, "45", got.RetryAfter)
	require.Equal(t, "2026-07-06T12:01:00Z", got.ResetAt)
	require.Equal(t, "http:429", got.FailureSource)
	require.True(t, got.CooldownUntil.Equal(breakerBaseTime.Add(30*time.Second)))
	require.Equal(t, int64(1), got.Generation)

	// Upsert conflict path updates in place rather than inserting a duplicate.
	rec.State = routing.BreakerClosed
	rec.ConsecutiveFailures = 0
	rec.LastSuccessAt = breakerBaseTime.Add(time.Minute)
	require.NoError(t, repo.Upsert(ctx, rec))
	all, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, routing.BreakerClosed, all[0].State)
}

func TestSQLHealthRepositoryRejectsInvalidBreakerStates(t *testing.T) {
	db := newSchemaDB(t)
	repo := routing.NewSQLHealthRepository(db)
	ctx := context.Background()
	key := routing.HealthKey{Provider: "ollama", Role: "chat.default", Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION}

	err := repo.Upsert(ctx, routing.ProviderHealth{
		Provider: key.Provider,
		Role:     key.Role,
		Kind:     key.Kind,
		State:    routing.BreakerState("degraded"),
	})
	require.ErrorContains(t, err, `invalid provider breaker state "degraded"`)
	_, found, err := repo.Get(ctx, key)
	require.NoError(t, err)
	require.False(t, found, "invalid state must not create a row")

	valid := routing.ProviderHealth{Provider: key.Provider, Role: key.Role, Kind: key.Kind, State: routing.BreakerClosed}
	require.NoError(t, repo.Upsert(ctx, valid))
	_, err = db.ExecContext(ctx, `UPDATE provider_health SET state = 'degraded' WHERE provider = ? AND role = ? AND kind = ?`, key.Provider, key.Role, int32(key.Kind))
	require.NoError(t, err)
	_, _, err = repo.Get(ctx, key)
	require.ErrorContains(t, err, `stored provider breaker state "degraded" is invalid`)
}
