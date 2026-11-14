package summarizechain

import (
	"context"
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type inferenceFunc func(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error)

func (f inferenceFunc) Run(ctx context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	return f(ctx, request)
}

func TestAIGatewayRunnerUsesLocalFaithfulSummaryTaskContract(t *testing.T) {
	var received *inferencev1.RunRequest
	runner := NewAIGatewayRunnerWithResolver(func(context.Context) (InferenceRunner, error) {
		return inferenceFunc(func(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
			received = request.Msg
			return connect.NewResponse(&inferencev1.RunResponse{
				ValueJson: `{"summary":"Keep the release date and action."}`,
				Model:     "resource-reported-model",
				Usage:     &inferencev1.Usage{OutputTokens: 9},
			}), nil
		}), nil
	})

	resultBytes, err := runner(context.Background(), []string{
		"gateway", "chat", "--role", "summarize.default", "--system", "Preserve names and dates.", "--max-tokens", "120", "--temperature", "0.2",
	}, "The release is ready. Deploy it tomorrow.")
	require.NoError(t, err)
	require.NotNil(t, received)
	require.Equal(t, GatewaySummaryRole, received.GetRole())
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, received.GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, received.GetPrivacyClass())
	require.Equal(t, localSummaryPrivacyProvenance, received.GetPrivacyProvenance())
	require.Equal(t, "Preserve names and dates.", received.GetInstruction())
	require.Equal(t, "The release is ready. Deploy it tomorrow.", received.GetSource())
	require.Equal(t, int32(120), received.GetMaxOutputTokens())
	require.Nil(t, received.GetSampling(), "the task uses the role's declared sampling policy")
	require.False(t, received.GetAllowSemanticEscalation())
	require.JSONEq(t, `{"type":"object","required":["summary"],"properties":{"summary":{"type":"string"}},"additionalProperties":false}`, received.GetSchemaJson())

	var result struct {
		Model     string `json:"model"`
		Response  string `json:"response"`
		EvalCount int64  `json:"eval_count"`
	}
	require.NoError(t, json.Unmarshal(resultBytes, &result))
	require.Equal(t, "resource-reported-model", result.Model)
	require.Equal(t, "Keep the release date and action.", result.Response)
	require.EqualValues(t, 9, result.EvalCount)
}

func TestAIGatewayRunnerRejectsCallerModelSelectors(t *testing.T) {
	runner := NewAIGatewayRunnerWithResolver(func(context.Context) (InferenceRunner, error) {
		t.Fatal("model selector rejection must happen before gateway resolution")
		return nil, nil
	})
	_, err := runner(context.Background(), []string{"gateway", "chat", "--model", "qwen3.5:9b", "--system", "summarize", "--max-tokens", "120"}, "text")
	require.ErrorContains(t, err, "not concrete model selectors")
}

func TestAIGatewayRunnerRejectsUnexpectedRole(t *testing.T) {
	runner := NewAIGatewayRunnerWithResolver(func(context.Context) (InferenceRunner, error) {
		t.Fatal("unsupported role rejection must happen before gateway resolution")
		return nil, nil
	})
	_, err := runner(context.Background(), []string{"gateway", "chat", "--role", "chat.default", "--system", "summarize", "--max-tokens", "120"}, "text")
	require.ErrorContains(t, err, "unsupported audio-tools summarization role")
}
