package inference

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"ai-gateway/internal/caller"
	"ai-gateway/internal/providers"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// routeLog is a RouteRecorder that hands each recorded event to the test.
type routeLog chan *routingv1.RouteEvidence

func (l routeLog) Create(_ context.Context, ev *routingv1.RouteEvidence) error {
	l <- ev
	return nil
}

// next waits for the next recorded event; recording finishes after the
// response, beside it.
func (l routeLog) next(t *testing.T) *routingv1.RouteEvidence {
	t.Helper()
	select {
	case ev := <-l:
		return ev
	case <-time.After(5 * time.Second):
		t.Fatal("no route event was recorded")
		return nil
	}
}

func (l routeLog) none(t *testing.T) {
	t.Helper()
	select {
	case ev := <-l:
		t.Fatalf("unexpected extra route event: %+v", ev)
	case <-time.After(200 * time.Millisecond):
	}
}

var fixtureScenarios = caller.NewScenarios([]string{"agent", "agent-manager", "source-ledger"})

func recordingOptions() (routeLog, ServiceOption) {
	log := make(routeLog, 16)
	return log, WithRouteRecorder(log, fixtureScenarios)
}

const agentManagerCaller = "scenario:agent-manager-start-api/agent-manager-api"

// Typed inference wrote no route event, so a caller could keep a large local
// model busy while the route log showed nothing. Every typed request now
// records who sent it, what ran and how long the provider took.
func TestTypedEmbeddingRecordsItsCallerAndTheCandidateThatRan(t *testing.T) {
	repository, _ := newEmbeddingFixture(t)
	log, option := recordingOptions()
	service := NewService(repository, option)
	ctx := caller.WithLabel(context.Background(), "scenario:source-ledger-start-api/source-ledger-api")

	response := service.Embed(ctx, embeddingRequest("alpha", "beta"))
	require.Nil(t, response.GetError())

	ev := log.next(t)
	require.Equal(t, "scenario:source-ledger-start-api/source-ledger-api", ev.GetCaller())
	require.Equal(t, "source-ledger", ev.GetScenario(), "the scenario is the longest known name the scope starts with")
	require.Equal(t, OperationEmbed, ev.GetOperation())
	require.Equal(t, "embedding.default", ev.GetRole())
	require.Equal(t, "succeeded", ev.GetStatus())
	require.Equal(t, providers.ProviderOllama, ev.GetSelectedProvider())
	require.Equal(t, "embed.resource", ev.GetSelectedResourceRole())
	require.Equal(t, "embed-model", ev.GetSelectedModel())
	require.Equal(t, "local", ev.GetSelectedLocality())
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, ev.GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL, ev.GetPrivacyClass())
	require.Positive(t, ev.GetProviderExecutionDurationNs())
	require.GreaterOrEqual(t, ev.GetTotalDurationNs(), ev.GetProviderExecutionDurationNs())
	require.Equal(t, ev.GetTotalDurationNs()-ev.GetProviderExecutionDurationNs(), ev.GetOverheadDurationNs())
	require.False(t, ev.GetFallbackUsed())
}

func TestRefusedTypedRequestIsRecordedAsBlockedWithItsErrorCode(t *testing.T) {
	repository, _ := newEmbeddingFixture(t)
	log, option := recordingOptions()
	service := NewService(repository, option)
	request := embeddingRequest("alpha")
	request.Profile = sharedv1.Profile_PROFILE_UNSPECIFIED
	ctx := caller.WithLabel(context.Background(), "agent:codex/ai-gateway")

	require.NotNil(t, service.Embed(ctx, request).GetError())

	ev := log.next(t)
	require.Equal(t, "blocked", ev.GetStatus(), "no provider ran")
	require.Equal(t, []string{inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST.String()}, ev.GetFailureReasons(),
		"only the stable code is kept, never the message")
	require.Equal(t, "agent:codex", ev.GetScenario())
	require.Empty(t, ev.GetSelectedProvider())
}

// Decide's enum fallback runs the typed Run path inside the Decide request.
// It is one caller request, so it is one event, naming the resource role that
// actually ran rather than the fallback's catalog name.
func TestDecideAnsweredByItsEnumFallbackIsOneEventNamingTheResourceRole(t *testing.T) {
	world := newDecisionWorld().refusesResolve("decide.local", providers.CodeUnreachable)
	log, option := recordingOptions()
	service := serviceWorld(t, world, `"{\"kind\":\"approval\"}"`, option)
	ctx := caller.WithLabel(context.Background(), agentManagerCaller)

	response := service.Decide(ctx, DecisionRequest{
		PrivacyProvenance: "tests.explicit_internal_input",
		Role:              decideRole, State: "Looks great, ship it.",
		Questions: []*inferencev1.DecisionQuestion{choice("kind", "correction", "approval", "question")},
	})
	require.Nil(t, response.GetError())

	ev := log.next(t)
	require.Equal(t, OperationDecide, ev.GetOperation())
	require.Equal(t, decideRole, ev.GetRole())
	require.Equal(t, "agent-manager", ev.GetScenario(), "the longer known name wins over its prefix")
	require.Equal(t, "classify.routing", ev.GetSelectedResourceRole())
	require.Equal(t, "qwen3.5:4b", ev.GetSelectedModel())
	require.EqualValues(t, 250, ev.GetInputTokens())
	log.none(t)
}

func TestRunBatchRecordsOneEventPerItem(t *testing.T) {
	world := newDecisionWorld()
	log, option := recordingOptions()
	service := serviceWorld(t, world, `"{\"kind\":\"approval\"}"`, option)
	ctx := caller.WithLabel(context.Background(), agentManagerCaller)
	item := ProviderRequest{
		SchemaJSON: `{"type":"object","properties":{"kind":{"type":"string"}},"required":["kind"]}`,
		Role:       "classify.fast", Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL, PrivacyProvenance: "tests.explicit_internal_input",
	}
	first, second := item, item
	first.Source, second.Source = "first", "second"

	response := service.RunBatch(ctx, []ProviderRequest{first, second})
	for _, result := range response.GetResults() {
		require.Nil(t, result.GetError(), "%+v", result)
	}

	for range 2 {
		ev := log.next(t)
		require.Equal(t, OperationRun, ev.GetOperation())
		require.Equal(t, "classify.fast", ev.GetRole())
		require.Equal(t, "classify.routing", ev.GetSelectedResourceRole())
		require.Equal(t, "succeeded", ev.GetStatus())
		require.Equal(t, agentManagerCaller, ev.GetCaller())
	}
	log.none(t)
}

func TestServiceWithoutARecorderRecordsNothing(t *testing.T) {
	repository, _ := newEmbeddingFixture(t)
	require.Nil(t, NewService(repository).Embed(context.Background(), embeddingRequest("alpha")).GetError())
}

func TestRouteEventOverheadUsesTheResourceReportedRuntime(t *testing.T) {
	started := time.Unix(100, 0)
	ev := routeEvent(routeRequest{operation: OperationEmbed}, routeOutcome{}, routeTraceSnapshot{
		attempts: 1, candidates: 1, executionNS: int64(200 * time.Millisecond),
		runtimeNS: int64(90 * time.Millisecond), admissionNS: int64(10 * time.Millisecond), runtimeKnown: true,
	}, started, started.Add(250*time.Millisecond))
	require.EqualValues(t, 250*time.Millisecond, ev.GetTotalDurationNs())
	require.EqualValues(t, 90*time.Millisecond, ev.GetProviderRuntimeDurationNs(), "provider runtime field preserves the resource report")
	require.EqualValues(t, 10*time.Millisecond, ev.GetProviderAdmissionWaitNs(), "provider admission field preserves the resource report")
	require.EqualValues(t, 150*time.Millisecond, ev.GetOverheadDurationNs(), "total minus runtime and admission wait")
	require.EqualValues(t, 250, ev.GetLatencyMs())
}
