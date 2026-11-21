package experimentarms

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
	"search-hub/internal/aigateway"
)

type trustedEncoderGatewayStub struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	request *inferencev1.EmbedRequest
}

func (g *trustedEncoderGatewayStub) Embed(_ context.Context, request *connect.Request[inferencev1.EmbedRequest]) (*connect.Response[inferencev1.EmbedResponse], error) {
	g.request = request.Msg
	values := make([]float64, trustedNomicDimensions)
	for i := range values {
		values[i] = 1
	}
	return connect.NewResponse(&inferencev1.EmbedResponse{
		Model: trustedNomicModel, Dimension: trustedNomicDimensions,
		Vectors: []*inferencev1.EmbeddingVector{{Values: values}},
	}), nil
}

func TestTrustedFleetEncoderUsesTypedGatewayAndPreservesRecipeIdentity(t *testing.T) {
	stub := &trustedEncoderGatewayStub{}
	path, handler := inferenceconnect.NewInferenceServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	resolve := func(context.Context) (string, error) { return server.URL, nil }
	embedder := aigateway.NewTaskEmbedder(
		resolve, server.Client(), ollamaEmbeddingRole, trustedNomicModel, trustedNomicDimensions,
		"search_query: ", "search_document: ", "search_hub.provider_description_fleet.query_and_document_text",
	)
	encoder, err := newTrustedOllamaFleetDescriptionEncoder(trustedNomicRole(), embedder)
	require.NoError(t, err)
	vector, err := encoder.EmbedQuery(context.Background(), "private provider description")
	require.NoError(t, err)
	require.Len(t, vector, trustedNomicDimensions)
	recipe, err := encoder.Recipe().Input()
	require.NoError(t, err)
	require.Equal(t, trustedNomicModel, recipe.ModelID)
	require.Equal(t, "search_query: private provider description", stub.request.GetTexts()[0])
	require.Equal(t, ollamaEmbeddingRole, stub.request.GetRole())
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, stub.request.GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, stub.request.GetPrivacyClass())
	require.Equal(t, "search_hub.provider_description_fleet.query_and_document_text", stub.request.GetPrivacyProvenance())
}
