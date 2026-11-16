package inference_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	handler "ai-gateway/handlers/inference"
	internalinference "ai-gateway/internal/inference"
)

type privacyGateRepository struct {
	runCalls    int
	embedCalls  int
	decideCalls int
}

func (r *privacyGateRepository) Run(context.Context, internalinference.ProviderRequest) (internalinference.ProviderResult, error) {
	r.runCalls++
	return internalinference.ProviderResult{ValueJSON: `{}`}, nil
}

func (r *privacyGateRepository) Embed(context.Context, internalinference.EmbeddingRequest) (internalinference.EmbeddingResult, error) {
	r.embedCalls++
	return internalinference.EmbeddingResult{Vectors: [][]float64{{1, 0}}, Dimension: 2}, nil
}

func (r *privacyGateRepository) Decide(context.Context, internalinference.DecisionRequest, internalinference.TypedRunner) *inferencev1.DecideResponse {
	r.decideCalls++
	return &inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}
}

func TestTypedExecutionFormsRejectInvalidProvenanceBeforeRepository(t *testing.T) {
	repository := &privacyGateRepository{}
	path, route := inferenceconnect.NewInferenceServiceHandler(handler.NewConnectHandler(internalinference.NewService(repository)))
	mux := http.NewServeMux()
	mux.Handle(path, route)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := inferenceconnect.NewInferenceServiceClient(server.Client(), server.URL)

	const invalidSource = "private source !"
	confidentialLocal := sharedv1.Profile_PROFILE_LOCAL_ONLY
	confidentialClass := sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL
	assertInvalid := func(response *inferencev1.InferenceError) {
		t.Helper()
		require.NotNil(t, response)
		require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, response.GetCode())
		require.NotContains(t, response.GetMessage(), invalidSource, "a malformed source value must not be echoed in refusal text")
	}

	run, err := client.Run(t.Context(), connect.NewRequest(&inferencev1.RunRequest{
		Source: "synthetic", SchemaJson: `{"type":"object"}`, Role: "extract.structured",
		Profile: confidentialLocal, PrivacyClass: confidentialClass, PrivacyProvenance: invalidSource,
	}))
	require.NoError(t, err)
	assertInvalid(run.Msg.GetError())

	runBatch, err := client.RunBatch(t.Context(), connect.NewRequest(&inferencev1.RunBatchRequest{
		Items: []*inferencev1.RunBatchItem{{Source: "synthetic"}}, SchemaJson: `{"type":"object"}`, Role: "extract.structured",
		Profile: confidentialLocal, PrivacyClass: confidentialClass, PrivacyProvenance: invalidSource,
	}))
	require.NoError(t, err)
	require.Len(t, runBatch.Msg.GetResults(), 1)
	assertInvalid(runBatch.Msg.GetResults()[0].GetError())

	question := &inferencev1.DecisionQuestion{Id: "kind", Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE,
		Instructions: "Choose one", Options: []*inferencev1.DecisionOption{{Label: "yes", Description: "yes"}, {Label: "no", Description: "no"}}}
	decide, err := client.Decide(t.Context(), connect.NewRequest(&inferencev1.DecideRequest{
		Role: "decide.fast", State: "synthetic", Questions: []*inferencev1.DecisionQuestion{question},
		Profile: confidentialLocal, PrivacyClass: confidentialClass, PrivacyProvenance: invalidSource,
	}))
	require.NoError(t, err)
	assertInvalid(decide.Msg.GetError())

	decideBatch, err := client.DecideBatch(t.Context(), connect.NewRequest(&inferencev1.DecideBatchRequest{
		Role: "decide.fast", Questions: []*inferencev1.DecisionQuestion{question}, Items: []*inferencev1.DecideBatchItem{{State: "synthetic"}},
		Profile: confidentialLocal, PrivacyClass: confidentialClass, PrivacyProvenance: invalidSource,
	}))
	require.NoError(t, err)
	require.Len(t, decideBatch.Msg.GetResults(), 1)
	assertInvalid(decideBatch.Msg.GetResults()[0].GetError())

	embed, err := client.Embed(t.Context(), connect.NewRequest(&inferencev1.EmbedRequest{
		Role: "embedding.default", Texts: []string{"synthetic"},
		Profile: confidentialLocal, PrivacyClass: confidentialClass, PrivacyProvenance: invalidSource,
	}))
	require.NoError(t, err)
	assertInvalid(embed.Msg.GetError())

	require.Zero(t, repository.runCalls, "Run and RunBatch must refuse before provider repository execution")
	require.Zero(t, repository.decideCalls, "Decide and DecideBatch must refuse before provider repository execution")
	require.Zero(t, repository.embedCalls, "Embed must refuse before provider repository execution")
}

func TestTypedEmbedRequiresCompleteLocalPrivacyContractBeforeRepository(t *testing.T) {
	tests := []struct {
		name    string
		request *inferencev1.EmbedRequest
	}{
		{name: "missing profile", request: &inferencev1.EmbedRequest{Role: "embedding.default", Texts: []string{"synthetic"}, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL, PrivacyProvenance: "gateway_test.fixture"}},
		{name: "missing class", request: &inferencev1.EmbedRequest{Role: "embedding.default", Texts: []string{"synthetic"}, Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyProvenance: "gateway_test.fixture"}},
		{name: "missing provenance", request: &inferencev1.EmbedRequest{Role: "embedding.default", Texts: []string{"synthetic"}, Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL}},
		{name: "unknown class", request: &inferencev1.EmbedRequest{Role: "embedding.default", Texts: []string{"synthetic"}, Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass(999), PrivacyProvenance: "gateway_test.fixture"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repository := &privacyGateRepository{}
			path, route := inferenceconnect.NewInferenceServiceHandler(handler.NewConnectHandler(internalinference.NewService(repository)))
			mux := http.NewServeMux()
			mux.Handle(path, route)
			server := httptest.NewServer(mux)
			t.Cleanup(server.Close)
			client := inferenceconnect.NewInferenceServiceClient(server.Client(), server.URL)
			response, err := client.Embed(t.Context(), connect.NewRequest(tc.request))
			require.NoError(t, err)
			require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, response.Msg.GetError().GetCode())
			require.Zero(t, repository.embedCalls, "incomplete privacy contract must fail before repository dispatch")
		})
	}
}
