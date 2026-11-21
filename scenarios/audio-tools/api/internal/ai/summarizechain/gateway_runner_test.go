package summarizechain

import (
	"context"
	"encoding/json"
	"testing"

	"audio-tools/internal/summarize"
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
				Model:     "resource-reported-model", Validated: true,
				Usage: &inferencev1.Usage{OutputTokens: 9},
			}), nil
		}), nil
	})

	resultBytes, err := runner(context.Background(), summarize.TaskRequest{
		Selector: "summarize.default", SelectorIsRole: true,
		Instruction: "Preserve names and dates.", Source: "The release is ready. Deploy it tomorrow.", MaxOutputTokens: 120,
	})
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
	_, err := runner(context.Background(), summarize.TaskRequest{Selector: "qwen3.5:9b", Instruction: "summarize", Source: "text", MaxOutputTokens: 120})
	require.ErrorContains(t, err, "not concrete model selectors")
}

func TestAIGatewayRunnerRejectsUnexpectedRole(t *testing.T) {
	runner := NewAIGatewayRunnerWithResolver(func(context.Context) (InferenceRunner, error) {
		t.Fatal("unsupported role rejection must happen before gateway resolution")
		return nil, nil
	})
	_, err := runner(context.Background(), summarize.TaskRequest{Selector: "chat.default", SelectorIsRole: true, Instruction: "summarize", Source: "text", MaxOutputTokens: 120})
	require.ErrorContains(t, err, "unsupported audio-tools summarization role")
}

func TestAIGatewayRunnerRejectsResourceOnlyExecutionSelectors(t *testing.T) {
	runner := NewAIGatewayRunnerWithResolver(func(context.Context) (InferenceRunner, error) {
		t.Fatal("resource-only selectors must fail before Gateway resolution")
		return nil, nil
	})
	_, err := runner(context.Background(), summarize.TaskRequest{
		Selector: "summarize.default", SelectorIsRole: true, Instruction: "summarize", Source: "text", MaxOutputTokens: 120,
		ContextTier: 4096, ExpectedModel: "qwen3.5:9b", ExpectedArtifactDigest: "sha256:artifact",
	})
	require.ErrorContains(t, err, "resource-specific context or rights selectors")
}

func TestAIGatewayRunnerRejectsUnvalidatedTypedResult(t *testing.T) {
	runner := NewAIGatewayRunnerWithResolver(func(context.Context) (InferenceRunner, error) {
		return inferenceFunc(func(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
			return connect.NewResponse(&inferencev1.RunResponse{ValueJson: `{"summary":"looks plausible"}`, Model: "model"}), nil
		}), nil
	})
	_, err := runner(context.Background(), summarize.TaskRequest{
		Selector: "summarize.default", SelectorIsRole: true, Instruction: "summarize", Source: "text", MaxOutputTokens: 120,
	})
	require.ErrorContains(t, err, "unvalidated summary result")
}
