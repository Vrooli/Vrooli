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

func TestAIGatewayRunnerUsesAudioToolsScopedSummaryTaskContract(t *testing.T) {
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
		Selector: summarize.DefaultSummarizeModel, SelectorIsRole: true,
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

func TestAIGatewayRunnerMapsQualitySelectorToQualityTaskRole(t *testing.T) {
	var received *inferencev1.RunRequest
	runner := NewAIGatewayRunnerWithResolver(func(context.Context) (InferenceRunner, error) {
		return inferenceFunc(func(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
			received = request.Msg
			return connect.NewResponse(&inferencev1.RunResponse{ValueJson: `{"summary":"facts"}`, Model: "qwen3.5:4b", Validated: true}), nil
		}), nil
	})
	_, err := runner(context.Background(), summarize.TaskRequest{Selector: "summarize.audio_tools.quality", SelectorIsRole: true, Instruction: "Keep facts", Source: "text", MaxOutputTokens: 120})
	require.NoError(t, err)
	require.NotNil(t, received)
	require.Equal(t, "summarize.audio_tools.quality", received.GetRole())
}

func TestAIGatewayRunnerPassesNewDeclaredAudioToolsRoleWithoutMappingCode(t *testing.T) {
	var received *inferencev1.RunRequest
	runner := NewAIGatewayRunnerWithResolver(func(context.Context) (InferenceRunner, error) {
		return inferenceFunc(func(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
			received = request.Msg
			return connect.NewResponse(&inferencev1.RunResponse{ValueJson: `{"summary":"ok"}`, Model: "declared-model", Validated: true}), nil
		}), nil
	})
	role := "summarize.audio_tools.experimental"
	_, err := runner(context.Background(), summarize.TaskRequest{Selector: role, SelectorIsRole: true, Instruction: "summarize", Source: "text", MaxOutputTokens: 120})
	require.NoError(t, err)
	require.NotNil(t, received)
	require.Equal(t, role, received.GetRole())
}

func TestLocalProviderQualityRoleResolvesThroughGatewayAndRecordsConcreteModel(t *testing.T) {
	var receivedRole string
	taskRunner := NewAIGatewayRunnerWithResolver(func(context.Context) (InferenceRunner, error) {
		return inferenceFunc(func(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
			receivedRole = request.Msg.GetRole()
			return connect.NewResponse(&inferencev1.RunResponse{ValueJson: `{"summary":"preserved facts"}`, Model: "qwen3.5:4b", Validated: true}), nil
		}), nil
	})
	provider := NewLocalProvider(summarize.NewSummarizerWithTaskRunner(taskRunner), summarize.DefaultSummarizeModel)
	result, err := provider.Summarize(context.Background(), Request{Text: "facts", Level: "light", LocalModelRole: "summarize.audio_tools.quality"})
	require.NoError(t, err)
	require.Equal(t, "summarize.audio_tools.quality", receivedRole)
	require.Equal(t, "qwen3.5:4b", result.ModelID)
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
	for _, role := range []string{"chat.default", "summarize.default", "summarize.faithful", "summarize.not_audio_tools"} {
		t.Run(role, func(t *testing.T) {
			_, err := runner(context.Background(), summarize.TaskRequest{Selector: role, SelectorIsRole: true, Instruction: "summarize", Source: "text", MaxOutputTokens: 120})
			require.ErrorContains(t, err, "unsupported audio-tools summarization role")
		})
	}
}

func TestAIGatewayRunnerRejectsResourceOnlyExecutionSelectors(t *testing.T) {
	runner := NewAIGatewayRunnerWithResolver(func(context.Context) (InferenceRunner, error) {
		t.Fatal("resource-only selectors must fail before Gateway resolution")
		return nil, nil
	})
	_, err := runner(context.Background(), summarize.TaskRequest{
		Selector: summarize.DefaultSummarizeModel, SelectorIsRole: true, Instruction: "summarize", Source: "text", MaxOutputTokens: 120,
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
		Selector: summarize.DefaultSummarizeModel, SelectorIsRole: true, Instruction: "summarize", Source: "text", MaxOutputTokens: 120,
	})
	require.ErrorContains(t, err, "unvalidated summary result")
}
