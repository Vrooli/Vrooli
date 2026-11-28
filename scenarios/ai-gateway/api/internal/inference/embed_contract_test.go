package inference

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// An embedding request states its locality and privacy attribution; none is
// inferred. A refusal names every field to fix and what to send, so a caller
// that omitted them (as Source Ledger's batch path once did) corrects its
// request in one step instead of one field per round trip.
func TestEmbedRefusalNamesEveryMissingContractField(t *testing.T) {
	repository, runner := newEmbeddingFixture(t)
	service := NewService(repository)

	response := service.Embed(context.Background(), EmbeddingRequest{Role: "embedding.default", Texts: []string{"hello"}})
	require.NotNil(t, response.GetError())
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, response.GetError().GetCode())
	require.Equal(t, "profile", response.GetError().GetConstruct())
	for _, want := range []string{"profile: missing; send PROFILE_LOCAL_ONLY", "privacy_class: missing", "privacy_provenance:"} {
		require.Contains(t, response.GetError().GetMessage(), want)
	}
	require.Zero(t, runner.count("gateway embed"), "an incomplete contract never reaches the resource")

	request := embeddingRequest("hello")
	request.Profile = sharedv1.Profile_PROFILE_LOCAL_FIRST
	response = service.Embed(context.Background(), request)
	require.Equal(t, "profile", response.GetError().GetConstruct())
	require.Contains(t, response.GetError().GetMessage(), "PROFILE_LOCAL_FIRST is not allowed")
	require.NotContains(t, response.GetError().GetMessage(), "privacy_class", "only the wrong field is named")

	request = embeddingRequest("hello")
	request.PrivacyClass = sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED
	response = service.Embed(context.Background(), request)
	require.Equal(t, "privacy_class", response.GetError().GetConstruct())

	response = service.Embed(context.Background(), embeddingRequest("hello"))
	require.Nil(t, response.GetError(), "a complete contract embeds")
}
