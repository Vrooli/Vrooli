package aigateway

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
)

type embeddingGatewayStub struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	request *inferencev1.EmbedRequest
	model   string
}

func (g *embeddingGatewayStub) Embed(_ context.Context, request *connect.Request[inferencev1.EmbedRequest]) (*connect.Response[inferencev1.EmbedResponse], error) {
	g.request = request.Msg
	return connect.NewResponse(&inferencev1.EmbedResponse{
		Model: g.model, Dimension: 3,
		Vectors: []*inferencev1.EmbeddingVector{{Values: []float64{1, 2, 3}}},
	}), nil
}

func TestTaskEmbeddingClientUsesTypedLocalConfidentialGatewayRequest(t *testing.T) {
	stub := &embeddingGatewayStub{model: "pinned-embedding-model"}
	var observations []EmbeddingObservation
	path, handler := inferenceconnect.NewInferenceServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := NewTaskEmbedder(
		func(context.Context) (string, error) { return server.URL, nil }, server.Client(),
		"embedding.default", "pinned-embedding-model", 3, "query: ", "document: ",
		"search_hub.provider_description_fleet.query_and_document_text",
		func(observation EmbeddingObservation) { observations = append(observations, observation) },
	)
	vector, err := client.EmbedQuery(context.Background(), "private title")
	require.NoError(t, err)
	require.Equal(t, []float64{1, 2, 3}, vector)
	require.Equal(t, []string{"query: private title"}, stub.request.GetTexts())
	require.Equal(t, "embedding.default", stub.request.GetRole())
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, stub.request.GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, stub.request.GetPrivacyClass())
	require.Equal(t, "search_hub.provider_description_fleet.query_and_document_text", stub.request.GetPrivacyProvenance())
	require.Len(t, observations, 1)
	require.Equal(t, "query", observations[0].Kind)
	require.Equal(t, "embedding.default", observations[0].Role)
	require.Equal(t, "pinned-embedding-model", observations[0].Model)
	require.Equal(t, 1, observations[0].VectorCount)
	require.False(t, observations[0].Failed)
}

func TestTaskEmbeddingClientRejectsModelDriftAgainstPinnedRecipe(t *testing.T) {
	stub := &embeddingGatewayStub{model: "different-model"}
	var observation EmbeddingObservation
	path, handler := inferenceconnect.NewInferenceServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := NewTaskEmbedder(
		func(context.Context) (string, error) { return server.URL, nil }, server.Client(),
		"embedding.default", "pinned-embedding-model", 3, "query: ", "document: ", "owner.source",
		func(value EmbeddingObservation) { observation = value },
	)
	_, err := client.Embed(context.Background(), "private title")
	require.ErrorContains(t, err, "model identity differs from the pinned recipe")
	require.Equal(t, "generic", observation.Kind)
	require.True(t, observation.Failed)
	require.Zero(t, observation.VectorCount)
}
