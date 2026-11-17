package prose

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/proto"
)

func TestHTTPGatewayEmbedTransmitsCallerPrivacyContract(t *testing.T) {
	var observed inferencev1.EmbedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, proto.Unmarshal(body, &observed))
		response := &inferencev1.EmbedResponse{
			Provider: "ollama", Model: "nomic", Dimension: 2,
			Vectors: []*inferencev1.EmbeddingVector{{Values: []float64{1, 0}}},
		}
		responseBody, err := proto.Marshal(response)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/proto")
		_, _ = w.Write(responseBody)
	}))
	defer server.Close()

	request := EmbeddingRequest{
		Role: "embedding.default", Texts: []string{"confidential draft"},
		Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: prosePrivacySource,
	}
	response, err := (HTTPGateway{BaseURL: server.URL}).Embed(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, 2, response.Dimension)
	require.Equal(t, request.Role, observed.GetRole())
	require.Equal(t, request.Texts, observed.GetTexts())
	require.Equal(t, request.Profile, observed.GetProfile())
	require.Equal(t, request.PrivacyClass, observed.GetPrivacyClass())
	require.Equal(t, request.PrivacyProvenance, observed.GetPrivacyProvenance())
}

func TestHTTPGatewayEmbedRefusesMismatchedProvenanceBeforeDispatch(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	_, err := (HTTPGateway{BaseURL: server.URL}).Embed(context.Background(), EmbeddingRequest{
		Role: "embedding.default", Texts: []string{"confidential draft"},
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: "another_scenario.public_content",
	})
	require.ErrorContains(t, err, "sourced confidential local-only contract")
	require.False(t, called, "mismatched provenance must not reach AI Gateway")
}
