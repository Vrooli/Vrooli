package routing_test

import (
	"context"
	"strings"
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

func TestModelEvalConsumersUsesDistinctRecordedScenarios(t *testing.T) {
	db := newSchemaDB(t)
	repo := routing.NewSQLRepository(db)
	ctx := context.Background()
	createdAt := "2026-11-14T14:00:00Z"
	for _, item := range []struct{ id, scenario, model string }{{"route-1", "vrooli-memory", "model-a"}, {"route-2", "client-rehearsal", "model-b"}, {"route-3", "vrooli-memory", "model-a"}, {"route-4", "", ""}} {
		require.NoError(t, repo.Create(ctx, &routingv1.RouteEvidence{EventId: item.id, RequestId: item.id, Scenario: item.scenario, Role: "chat.default", SelectedProvider: "ollama", SelectedLocality: "local", SelectedModel: item.model, Status: "succeeded", CreatedAt: createdAt}))
	}
	got, err := repo.ModelEvalConsumers(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"client-rehearsal", "vrooli-memory"}, got)
	runtimeModels, err := repo.ModelEvalRuntimeModels(ctx)
	require.NoError(t, err)
	require.Equal(t, []routing.ModelEvalRouteModel{
		{Scenario: "client-rehearsal", Role: "chat.default", Provider: "ollama", Locality: "local", Model: "model-b", Status: "succeeded", SampleCount: 1, LatestObservedAt: createdAt},
		{Scenario: "vrooli-memory", Role: "chat.default", Provider: "ollama", Locality: "local", Model: "model-a", Status: "succeeded", SampleCount: 2, LatestObservedAt: createdAt},
	}, runtimeModels)
}

func TestExecuteRecordsFailureClassAndRedactsOnFailure(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE-CODES]
	runner := roleRunner()
	runner.ErrorMatch = func(command providers.Command) error {
		if strings.HasPrefix(command.String(), "resource-ollama gateway generate --role chat.default ") {
			return &providers.CommandError{Code: "timeout", Command: "resource-ollama"}
		}
		return nil
	}
	db := newSchemaDB(t)
	svc := newService(testAdapters(runner), routing.NewSQLRepository(db),
		routing.WithHealth(routing.NewSQLHealthRepository(db)),
		routing.WithClock(func() time.Time { return breakerBaseTime }))

	resp, err := svc.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), "secret prompt")
	require.NoError(t, err)
	require.Equal(t, "failed", resp.GetEvidence().GetStatus())
	require.Equal(t, "timeout", resp.GetEvidence().GetFailureClass())
	// No PII-detection pass ran (TEXT_GENERATION) and the request failed, so the
	// redacted seam is false. The real privacy guarantee is that the input does
	// not leak into the failure reasons (below).
	require.False(t, resp.GetEvidence().GetPromptRedacted())
	require.False(t, resp.GetEvidence().GetResponseRedacted())
	require.NotContains(t, joinReasons(resp.GetEvidence().GetFailureReasons()), "secret prompt")
}

func TestPersistedAdapterFailureReasonsContainStableCodesOnly(t *testing.T) {
	const promptSentinel = "LM4_PROMPT_SENTINEL"
	const outputSentinel = "LM4_OUTPUT_SENTINEL"

	for _, tc := range []struct {
		name       string
		command    string
		wantReason string
	}{
		{name: "role resolution", command: " policy resolve ", wantReason: "provider.role_resolution_failed"},
		{name: "provider execution", command: " gateway generate ", wantReason: "provider.execution_failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := roleRunner()
			runner.ErrorMatch = func(command providers.Command) error {
				if strings.Contains(command.String(), tc.command) && strings.Contains(command.String(), "resource-ollama") {
					return &providers.CommandError{
						Code:    "provider_failed",
						Command: "resource-ollama",
						Stderr:  "provider rejected prompt=" + promptSentinel + " response=" + outputSentinel,
					}
				}
				return nil
			}

			db := newSchemaDB(t)
			svc := newService(testAdapters(runner), routing.NewSQLRepository(db))
			response, err := svc.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), promptSentinel)
			require.NoError(t, err)
			require.Equal(t, "failed", response.GetEvidence().GetStatus())

			stored, err := svc.ListEvidence(context.Background(), routing.EvidenceFilter{Limit: 10})
			require.NoError(t, err)
			require.Len(t, stored, 1)
			reasons := stored[0].GetFailureReasons()
			require.Contains(t, reasons, tc.wantReason)
			joined := strings.Join(reasons, "\n")
			require.NotContains(t, joined, promptSentinel)
			require.NotContains(t, joined, outputSentinel)
			require.Regexp(t, `^[a-z][a-z0-9_.]*$`, joined, "persisted failure reasons must be stable codes only")
		})
	}
}

func TestExecuteRecordsRejectionReasonWhenBlocked(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE-CODES]
	runner := roleRunner()
	db := newSchemaDB(t)
	svc := newService(testAdapters(runner), routing.NewSQLRepository(db))

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
	svc := newService(testAdapters(runner), routing.NewSQLRepository(db),
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

func TestExecuteUsesInjectedClockForEvidenceTimestampAndLatency(t *testing.T) {
	runner := roleRunner()
	runner.Results[ollamaGenerateCmd] = providers.Result{Stdout: `{"response":"ok"}`}
	db := newSchemaDB(t)
	fixed := time.Date(2026, 11, 20, 12, 34, 56, 0, time.UTC)
	svc := newService(testAdapters(runner), routing.NewSQLRepository(db),
		routing.WithClock(func() time.Time { return fixed }))

	resp, err := svc.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), "hello")
	require.NoError(t, err)
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Equal(t, fixed.Format(time.RFC3339Nano), resp.GetEvidence().GetCreatedAt())
	require.Zero(t, resp.GetEvidence().GetLatencyMs())
}
