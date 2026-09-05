package structuredresult

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

type fakeGatewayClient struct {
	request  *inferencev1.RunRequest
	response *inferencev1.RunResponse
}

func (f *fakeGatewayClient) Run(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	f.request = request.Msg
	return connect.NewResponse(f.response), nil
}

func TestGatewayExtractorUsesTypedInferenceWithoutRunnerLaunch(t *testing.T) {
	client := &fakeGatewayClient{response: &inferencev1.RunResponse{
		ValueJson: `"complete"`, Provider: "ollama", Model: "qwen3.5:9b", Validated: true,
	}}
	extractor := &GatewayExtractor{Client: client}
	response, err := extractor.Extract(t.Context(), ExtractRequest{
		RoleRef: DefaultExtractRole, Source: "The work is done.", Schema: []byte(`{"type":"string","enum":["complete","blocked"]}`),
	})
	require.NoError(t, err)
	require.Equal(t, `"complete"`, string(response.Candidate))
	require.Equal(t, "ollama", response.Provider)
	require.Equal(t, "qwen3.5:9b", response.Model)
	require.Equal(t, DefaultExtractRole, client.request.GetRole())
	require.Equal(t, `{"type":"string","enum":["complete","blocked"]}`, client.request.GetSchemaJson())
}

func TestGatewayExtractorSendsAnInstructionSeparateFromTheSchema(t *testing.T) {
	client := &fakeGatewayClient{response: &inferencev1.RunResponse{ValueJson: `"complete"`, Validated: true}}
	extractor := &GatewayExtractor{Client: client}

	// ai-gateway treats schema descriptions as metadata and never as
	// instruction, so an empty instruction would leave the provider with no
	// stated intent at all.
	_, err := extractor.Extract(t.Context(), ExtractRequest{Source: "The work is done.", Schema: []byte(`{"type":"string"}`)})
	require.NoError(t, err)
	require.NotEmpty(t, client.request.GetInstruction())

	_, err = extractor.Extract(t.Context(), ExtractRequest{
		Source: "The work is done.", Schema: []byte(`{"type":"string"}`), Instruction: "Report the terminal state only.",
	})
	require.NoError(t, err)
	require.Equal(t, "Report the terminal state only.", client.request.GetInstruction())
}

// http.DefaultClient has no timeout, so a stalled gateway would hang run
// finalization indefinitely.
func TestNewGatewayExtractorUsesABoundedHTTPClient(t *testing.T) {
	extractor := NewGatewayExtractor(nil)
	client, ok := extractor.HTTPClient.(*http.Client)
	require.True(t, ok, "the default client must be a bounded *http.Client")
	require.Equal(t, DefaultGatewayTimeout, client.Timeout)
	require.NotZero(t, client.Timeout)
	require.NotSame(t, http.DefaultClient, client)
}

func TestGatewayExtractorAbstainsOnTypedUnavailable(t *testing.T) {
	extractor := &GatewayExtractor{Client: &fakeGatewayClient{response: &inferencev1.RunResponse{
		Error: &inferencev1.InferenceError{Code: inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, Message: "provider unavailable"},
	}}}
	response, err := extractor.Extract(t.Context(), ExtractRequest{Source: "source", Schema: []byte(`{"type":"string"}`)})
	require.Error(t, err)
	require.True(t, response.Abstained)
}
