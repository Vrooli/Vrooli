package inference_test

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
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
		Source:            "a source",
		SchemaJson:        `{"type":"object"}`,
		Instruction:       "Return the empty object.",
		Role:              "extract.structured",
		PrivacyProvenance: "connect_test.explicit_public_source",
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
		PrivacyProvenance:       "caller.owned.public_sample",
		AllowSemanticEscalation: true,
	}
	_, err := connectHandler.Run(t.Context(), connect.NewRequest(message))
	require.NoError(t, err)
	first := repository.requests[0]
	require.Equal(t, message.GetRootOperationId(), first.RootOperationID)
	require.Equal(t, "caller.owned.public_sample", first.PrivacyProvenance)
	require.True(t, first.AllowSemanticEscalation)
	require.NotEmpty(t, first.RequestDigest)

	message.RootOperationId = "99887766554433221100ffeeddccbbaa"
	_, err = connectHandler.Run(t.Context(), connect.NewRequest(message))
	require.NoError(t, err)
	require.Equal(t, first.RequestDigest, repository.requests[1].RequestDigest, "root is an idempotency name, not canonical request content")

	message.Source = "changed source"
	_, err = connectHandler.Run(t.Context(), connect.NewRequest(message))
	require.NoError(t, err)
	require.NotEqual(t, first.RequestDigest, repository.requests[2].RequestDigest)

	message.Source = "source"
	message.PrivacyProvenance = "different.caller_basis"
	_, err = connectHandler.Run(t.Context(), connect.NewRequest(message))
	require.NoError(t, err)
	require.NotEqual(t, first.RequestDigest, repository.requests[3].RequestDigest, "privacy provenance is part of the request contract")

	message.PrivacyProvenance = "caller.owned.public_sample"
	message.AllowSemanticEscalation = false
	_, err = connectHandler.Run(t.Context(), connect.NewRequest(message))
	require.NoError(t, err)
	require.NotEqual(t, first.RequestDigest, repository.requests[4].RequestDigest, "caller consent is part of the request contract")
}

type sourceCaptureRepository struct {
	sources            []string
	privacyProvenances []string
}

func (r *sourceCaptureRepository) Run(ctx context.Context, request inference.ProviderRequest) (inference.ProviderResult, error) {
	r.sources = append(r.sources, providers.Source(ctx))
	r.privacyProvenances = append(r.privacyProvenances, request.PrivacyProvenance)
	return inference.ProviderResult{ValueJSON: `{}`}, nil
}

func (r *sourceCaptureRepository) Decide(ctx context.Context, request inference.DecisionRequest, _ inference.TypedRunner) *inferencev1.DecideResponse {
	r.sources = append(r.sources, providers.Source(ctx))
	r.privacyProvenances = append(r.privacyProvenances, request.PrivacyProvenance)
	return &inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}
}

type embeddingCaptureRepository struct{ requests []inference.EmbeddingRequest }

func (r *embeddingCaptureRepository) Run(context.Context, inference.ProviderRequest) (inference.ProviderResult, error) {
	return inference.ProviderResult{}, nil
}

func (r *embeddingCaptureRepository) Embed(_ context.Context, request inference.EmbeddingRequest) (inference.EmbeddingResult, error) {
	r.requests = append(r.requests, request)
	return inference.EmbeddingResult{Vectors: [][]float64{{1, 0}}, Provider: "ollama", Model: "nomic", Dimension: 2}, nil
}

func TestEmbedPreservesSourcedPrivacyAndLocalOnlyContract(t *testing.T) {
	repository := &embeddingCaptureRepository{}
	connectHandler := handler.NewConnectHandler(inference.NewService(repository))
	_, err := connectHandler.Embed(t.Context(), connect.NewRequest(&inferencev1.EmbedRequest{
		Role: "embedding.default", Texts: []string{"private draft"},
		Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: "prose_studio.private_editorial_content",
	}))
	require.NoError(t, err)
	require.Len(t, repository.requests, 1)
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, repository.requests[0].Profile)
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, repository.requests[0].PrivacyClass)
	require.Equal(t, "prose_studio.private_editorial_content", repository.requests[0].PrivacyProvenance)
}

func TestEmbedConnectHandlerRejectsMismatchedContractBeforeRepository(t *testing.T) {
	repository := &embeddingCaptureRepository{}
	connectHandler := handler.NewConnectHandler(inference.NewService(repository))
	response, err := connectHandler.Embed(t.Context(), connect.NewRequest(&inferencev1.EmbedRequest{
		Role: "embedding.default", Texts: []string{"private draft"},
		Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: "malformed private source !",
	}))
	require.NoError(t, err)
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, response.Msg.GetError().GetCode())
	require.Empty(t, repository.requests, "invalid provenance must be refused before embedding dispatch")
}

func TestEmbedGeneratedConnectRouteRejectsInvalidProvenanceBeforeRepository(t *testing.T) {
	repository := &embeddingCaptureRepository{}
	path, route := inferenceconnect.NewInferenceServiceHandler(handler.NewConnectHandler(inference.NewService(repository)))
	mux := http.NewServeMux()
	mux.Handle(path, route)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := inferenceconnect.NewInferenceServiceClient(server.Client(), server.URL)
	response, err := client.Embed(t.Context(), connect.NewRequest(&inferencev1.EmbedRequest{
		Role: "embedding.default", Texts: []string{"private draft"},
		Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: "malformed private source !",
	}))
	require.NoError(t, err)
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, response.Msg.GetError().GetCode())
	require.Empty(t, repository.requests, "the generated HTTP route must refuse invalid provenance before embedding dispatch")
}

func TestEveryTypedRoutePreservesCallerSourceAndSharedProfile(t *testing.T) {
	repository := &sourceCaptureRepository{}
	connectHandler := handler.NewConnectHandler(inference.NewService(repository))
	run := connect.NewRequest(&inferencev1.RunRequest{Source: "source", SchemaJson: `{"type":"object"}`, Role: "extract.structured",
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		PrivacyProvenance: "run-source-basis"})
	run.Header().Set("X-Vrooli-AI-Source", "own-key")
	_, err := connectHandler.Run(t.Context(), run)
	require.NoError(t, err)
	require.Equal(t, []string{"own-key"}, repository.sources)
	require.Equal(t, []string{"run-source-basis"}, repository.privacyProvenances)

	runBatch := connect.NewRequest(&inferencev1.RunBatchRequest{
		Items: []*inferencev1.RunBatchItem{{Source: "source"}}, SchemaJson: `{"type":"object"}`, Role: "extract.structured",
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		PrivacyProvenance: "batch-source-basis",
	})
	runBatch.Header().Set("X-Vrooli-AI-Source", "own-key")
	_, err = connectHandler.RunBatch(t.Context(), runBatch)
	require.NoError(t, err)
	require.Equal(t, []string{"own-key", "own-key"}, repository.sources)
	require.Equal(t, []string{"run-source-basis", "batch-source-basis"}, repository.privacyProvenances)

	question := &inferencev1.DecisionQuestion{Id: "kind", Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE,
		Instructions: "Choose one", Options: []*inferencev1.DecisionOption{{Label: "yes", Description: "yes"}, {Label: "no", Description: "no"}}}
	decide := connect.NewRequest(&inferencev1.DecideRequest{Role: "decide.fast", State: "state", Questions: []*inferencev1.DecisionQuestion{question},
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		PrivacyProvenance: "decision-source-basis"})
	decide.Header().Set("X-Vrooli-AI-Source", "own-key")
	_, err = connectHandler.Decide(t.Context(), decide)
	require.NoError(t, err)
	require.Equal(t, []string{"own-key", "own-key", "own-key"}, repository.sources)
	require.Equal(t, []string{"run-source-basis", "batch-source-basis", "decision-source-basis"}, repository.privacyProvenances)

	decideBatch := connect.NewRequest(&inferencev1.DecideBatchRequest{Role: "decide.fast", Questions: []*inferencev1.DecisionQuestion{question},
		Items: []*inferencev1.DecideBatchItem{{State: "state"}}, Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL, PrivacyProvenance: "decision-batch-source-basis"})
	decideBatch.Header().Set("X-Vrooli-AI-Source", "own-key")
	_, err = connectHandler.DecideBatch(t.Context(), decideBatch)
	require.NoError(t, err)
	require.Equal(t, []string{"own-key", "own-key", "own-key", "own-key"}, repository.sources)
	require.Equal(t, []string{"run-source-basis", "batch-source-basis", "decision-source-basis", "decision-batch-source-basis"}, repository.privacyProvenances)
}
