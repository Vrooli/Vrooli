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

func TestEvidenceRoundTripPersistsStructuredCodes(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE-CODES]
	db := newSchemaDB(t)
	repo := routing.NewSQLRepository(db)
	ctx := context.Background()

	ev := &routingv1.RouteEvidence{
		EventId:                 "rt-codes-1",
		RequestId:               "req-1",
		Scenario:                "fixture",
		Operation:               "summarize",
		Role:                    "chat.default",
		Profile:                 sharedv1.Profile_PROFILE_LOCAL_FIRST,
		PrivacyClass:            sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		SelectedProvider:        "openrouter",
		SelectedLocality:        "remote",
		Status:                  "succeeded",
		PolicyReasons:           []string{"local-first"},
		FailureReasons:          []string{"ollama: timeout"},
		FallbackUsed:            true,
		PromptRedacted:          true,
		ResponseRedacted:        true,
		LatencyMs:               42,
		CreatedAt:               "2026-07-06T12:00:00Z",
		BreakerState:            "closed",
		FailureClass:            "timeout",
		RejectionReason:         "",
		CapacityVerdict:         "fit",
		CapacityClaimId:         "claim-9",
		CapacityRequiredBytes:   1024,
		CapacityGrantedBytes:    2048,
		CapacityReclaimRequired: true,
		InputTokens:             12,
		OutputTokens:            34,
		CostEstimate:            0.0025,
		SelectedModel:           "some-model",
	}
	require.NoError(t, repo.Create(ctx, ev))

	got, err := repo.Get(ctx, "rt-codes-1")
	require.NoError(t, err)
	require.Equal(t, "closed", got.GetBreakerState())
	require.Equal(t, "timeout", got.GetFailureClass())
	require.Equal(t, "fit", got.GetCapacityVerdict())
	require.Equal(t, "claim-9", got.GetCapacityClaimId())
	require.Equal(t, int64(1024), got.GetCapacityRequiredBytes())
	require.Equal(t, int64(2048), got.GetCapacityGrantedBytes())
	require.True(t, got.GetCapacityReclaimRequired())
	require.Equal(t, int64(12), got.GetInputTokens())
	require.Equal(t, int64(34), got.GetOutputTokens())
	require.InDelta(t, 0.0025, got.GetCostEstimate(), 1e-9)
	require.Equal(t, "some-model", got.GetSelectedModel())
	// Backward-compatible fields still round-trip.
	require.True(t, got.GetFallbackUsed())
	require.True(t, got.GetPromptRedacted())
	require.Equal(t, "openrouter", got.GetSelectedProvider())
}

func TestExecuteRecordsFailureClassAndRedactsOnFailure(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE-CODES]
	runner := roleRunner()
	runner.Errors[ollamaGenerateCmd] = &providers.CommandError{Code: "timeout", Command: "resource-ollama"}
	db := newSchemaDB(t)
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(db),
		routing.WithHealth(routing.NewSQLHealthRepository(db)),
		routing.WithClock(func() time.Time { return breakerBaseTime }))

	resp, err := svc.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), "secret prompt")
	require.NoError(t, err)
	require.Equal(t, "failed", resp.GetEvidence().GetStatus())
	require.Equal(t, "timeout", resp.GetEvidence().GetFailureClass())
	require.True(t, resp.GetEvidence().GetPromptRedacted())
	require.True(t, resp.GetEvidence().GetResponseRedacted())
	require.NotContains(t, joinReasons(resp.GetEvidence().GetFailureReasons()), "secret prompt")
}

func TestExecuteRecordsRejectionReasonWhenBlocked(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE-CODES]
	runner := roleRunner()
	db := newSchemaDB(t)
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(db))

	req := baseRequest(sharedv1.Profile_PROFILE_LOCAL_FIRST)
	req.Role = "missing.role" // not exposed by any provider policy fixture
	resp, err := svc.Execute(context.Background(), req, "hello")
	require.NoError(t, err)
	require.Equal(t, "blocked", resp.GetEvidence().GetStatus())
	require.Equal(t, "no_eligible_route", resp.GetEvidence().GetRejectionReason())
}

func TestExecuteRecordsBreakerStateOnHalfOpenSuccess(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE-CODES]
	runner := roleRunner()
	runner.Results[ollamaGenerateCmd] = providers.Result{Stdout: `{"response":"ok"}`}
	db := newSchemaDB(t)
	repo := routing.NewSQLHealthRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Upsert(ctx, routing.ProviderHealth{
		Provider:      "ollama",
		Role:          "chat.default",
		Kind:          sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		State:         routing.BreakerOpen,
		CooldownUntil: breakerBaseTime,
		UpdatedAt:     breakerBaseTime,
	}))
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(db),
		routing.WithHealth(repo),
		routing.WithClock(func() time.Time { return breakerBaseTime.Add(time.Second) }))

	resp, err := svc.Execute(ctx, baseRequest(sharedv1.Profile_PROFILE_LOCAL_FIRST), "hello")
	require.NoError(t, err)
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Equal(t, string(routing.BreakerHalfOpen), resp.GetEvidence().GetBreakerState())
}

func joinReasons(reasons []string) string {
	out := ""
	for _, r := range reasons {
		out += r + " "
	}
	return out
}
