package inference_test

import (
	"context"
	"log"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	handler "ai-gateway/handlers/inference"
	"ai-gateway/internal/inference"
	"ai-gateway/internal/providers"
	"ai-gateway/internal/server"

	httpx "github.com/vrooli/api-core/servertest"

	"github.com/vrooli/api-core/schedule"
)

func TestRunIsReachableThroughGeneratedConnectClient(t *testing.T) {
	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		handler.Module(handler.Deps{Service: inference.NewService(nil)}),
	)
	live := httpx.NewLiveServer(t, srv)
	client := inferenceconnect.NewInferenceServiceClient(live.Client, live.URL)
	response, err := client.Run(t.Context(), connect.NewRequest(&inferencev1.RunRequest{
		Source:      "a source",
		SchemaJson:  `{"type":"object"}`,
		Instruction: "Return the empty object.",
		Role:        "extract.structured",
	}))
	require.NoError(t, err)
	require.True(t, response.Msg.GetValidated())
	require.Equal(t, `{}`, response.Msg.GetValueJson())
	require.NotNil(t, response.Msg.GetUsage())
}

type digestCaptureRepository struct{ requests []inference.ProviderRequest }

func (r *digestCaptureRepository) Run(_ context.Context, request inference.ProviderRequest) (inference.ProviderResult, error) {
	r.requests = append(r.requests, request)
	return inference.ProviderResult{ValueJSON: `{}`}, nil
}

func TestRunBindsCanonicalRequestDigestAndTreatsRootAsIdempotencyOnly(t *testing.T) {
	repository := &digestCaptureRepository{}
	connectHandler := handler.NewConnectHandler(inference.NewService(repository))
	message := &inferencev1.RunRequest{
		RootOperationId: "aabbccddeeff00112233445566778899",
		Source:          "source", SchemaJson: `{"type":"object"}`, Role: "classify.fast",
	}
	_, err := connectHandler.Run(t.Context(), connect.NewRequest(message))
	require.NoError(t, err)
	first := repository.requests[0]
	require.Equal(t, message.GetRootOperationId(), first.RootOperationID)
	require.NotEmpty(t, first.RequestDigest)

	message.RootOperationId = "99887766554433221100ffeeddccbbaa"
	_, err = connectHandler.Run(t.Context(), connect.NewRequest(message))
	require.NoError(t, err)
	require.Equal(t, first.RequestDigest, repository.requests[1].RequestDigest, "root is an idempotency name, not canonical request content")

	message.Source = "changed source"
	_, err = connectHandler.Run(t.Context(), connect.NewRequest(message))
	require.NoError(t, err)
	require.NotEqual(t, first.RequestDigest, repository.requests[2].RequestDigest)
}

type sourceCaptureRepository struct{ sources []string }

func (r *sourceCaptureRepository) Run(ctx context.Context, _ inference.ProviderRequest) (inference.ProviderResult, error) {
	r.sources = append(r.sources, providers.Source(ctx))
	return inference.ProviderResult{ValueJSON: `{}`}, nil
}

func (r *sourceCaptureRepository) Decide(ctx context.Context, _ inference.DecisionRequest, _ inference.TypedRunner) *inferencev1.DecideResponse {
	r.sources = append(r.sources, providers.Source(ctx))
	return &inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}
}

func TestEveryTypedRoutePreservesCallerSourceAndSharedProfile(t *testing.T) {
	repository := &sourceCaptureRepository{}
	connectHandler := handler.NewConnectHandler(inference.NewService(repository))
	run := connect.NewRequest(&inferencev1.RunRequest{Source: "source", SchemaJson: `{"type":"object"}`, Role: "extract.structured",
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL})
	run.Header().Set("X-Vrooli-AI-Source", "own-key")
	_, err := connectHandler.Run(t.Context(), run)
	require.NoError(t, err)
	require.Equal(t, []string{"own-key"}, repository.sources)

	runBatch := connect.NewRequest(&inferencev1.RunBatchRequest{
		Items: []*inferencev1.RunBatchItem{{Source: "source"}}, SchemaJson: `{"type":"object"}`, Role: "extract.structured",
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
	})
	runBatch.Header().Set("X-Vrooli-AI-Source", "own-key")
	_, err = connectHandler.RunBatch(t.Context(), runBatch)
	require.NoError(t, err)
	require.Equal(t, []string{"own-key", "own-key"}, repository.sources)

	question := &inferencev1.DecisionQuestion{Id: "kind", Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE,
		Instructions: "Choose one", Options: []*inferencev1.DecisionOption{{Label: "yes", Description: "yes"}, {Label: "no", Description: "no"}}}
	decide := connect.NewRequest(&inferencev1.DecideRequest{Role: "decide.fast", State: "state", Questions: []*inferencev1.DecisionQuestion{question},
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL})
	decide.Header().Set("X-Vrooli-AI-Source", "own-key")
	_, err = connectHandler.Decide(t.Context(), decide)
	require.NoError(t, err)
	require.Equal(t, []string{"own-key", "own-key", "own-key"}, repository.sources)

	decideBatch := connect.NewRequest(&inferencev1.DecideBatchRequest{Role: "decide.fast", Questions: []*inferencev1.DecisionQuestion{question},
		Items: []*inferencev1.DecideBatchItem{{State: "state"}}, Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL})
	decideBatch.Header().Set("X-Vrooli-AI-Source", "own-key")
	_, err = connectHandler.DecideBatch(t.Context(), decideBatch)
	require.NoError(t, err)
	require.Equal(t, []string{"own-key", "own-key", "own-key", "own-key"}, repository.sources)
}
