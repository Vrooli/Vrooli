package summarizechain

import (
	"context"
	"errors"
	"testing"

	"audio-tools/internal/summarize"
	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

func validTask() summarize.TaskRequest {
	return summarize.TaskRequest{
		Selector: "summarize.default", SelectorIsRole: true,
		Instruction: "summarize", Source: "source text", MaxOutputTokens: 120,
	}
}

func responseRunner(t *testing.T, resp *inferencev1.RunResponse, err error) func(context.Context) (InferenceRunner, error) {
	t.Helper()
	return func(context.Context) (InferenceRunner, error) {
		return inferenceFunc(func(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
			if err != nil {
				return nil, err
			}
			return connect.NewResponse(resp), nil
		}), nil
	}
}

func TestAIGatewayRunnerRejectsInvalidTaskBeforeResolution(t *testing.T) {
	noResolve := func(context.Context) (InferenceRunner, error) {
		t.Fatal("invalid task must be rejected before gateway resolution")
		return nil, nil
	}
	cases := []struct {
		name    string
		mutate  func(*summarize.TaskRequest)
		wantErr string
	}{
		{"missing instruction", func(task *summarize.TaskRequest) { task.Instruction = " " }, "instruction is required"},
		{"missing source", func(task *summarize.TaskRequest) { task.Source = "" }, "source is required"},
		{"zero token cap", func(task *summarize.TaskRequest) { task.MaxOutputTokens = 0 }, "output-token cap must be positive"},
		{"negative token cap", func(task *summarize.TaskRequest) { task.MaxOutputTokens = -1 }, "output-token cap must be positive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := NewAIGatewayRunnerWithResolver(noResolve)
			task := validTask()
			tc.mutate(&task)
			_, err := runner(context.Background(), task)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestAIGatewayRunnerRequiresResolver(t *testing.T) {
	_, err := NewAIGatewayRunnerWithResolver(nil)(context.Background(), validTask())
	require.ErrorContains(t, err, "resolver is required")
}

func TestAIGatewayRunnerWrapsResolutionAndCallErrors(t *testing.T) {
	resolveErr := errors.New("bridge unavailable")
	_, err := NewAIGatewayRunnerWithResolver(func(context.Context) (InferenceRunner, error) {
		return nil, resolveErr
	})(context.Background(), validTask())
	require.ErrorIs(t, err, resolveErr)
	require.ErrorContains(t, err, "resolve ai-gateway")

	callErr := errors.New("deadline exceeded")
	_, err = NewAIGatewayRunnerWithResolver(responseRunner(t, nil, callErr))(context.Background(), validTask())
	require.ErrorIs(t, err, callErr)
	require.ErrorContains(t, err, "ai-gateway local summary")
}

func TestAIGatewayRunnerSurfacesGatewayError(t *testing.T) {
	_, err := NewAIGatewayRunnerWithResolver(responseRunner(t, &inferencev1.RunResponse{
		Error: &inferencev1.InferenceError{Code: inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_PROVIDER_FAILED, Message: "no capacity"},
	}, nil))(context.Background(), validTask())
	require.ErrorContains(t, err, "no capacity")
	require.ErrorContains(t, err, "PROVIDER_FAILED")
}

func TestAIGatewayRunnerRejectsMalformedResults(t *testing.T) {
	cases := []struct {
		name    string
		resp    *inferencev1.RunResponse
		wantErr string
	}{
		{"empty response", nil, "empty summary response"},
		{"empty value", &inferencev1.RunResponse{Validated: true, Model: "m"}, "unvalidated summary result"},
		{"missing model", &inferencev1.RunResponse{Validated: true, ValueJson: `{"summary":"ok"}`}, "did not report the selected summary model"},
		{"invalid json", &inferencev1.RunResponse{Validated: true, Model: "m", ValueJson: `{`}, "decode ai-gateway summary result"},
		{"unknown field", &inferencev1.RunResponse{Validated: true, Model: "m", ValueJson: `{"summary":"ok","extra":1}`}, "decode ai-gateway summary result"},
		{"trailing content", &inferencev1.RunResponse{Validated: true, Model: "m", ValueJson: `{"summary":"ok"}{"summary":"again"}`}, "trailing content"},
		{"empty summary", &inferencev1.RunResponse{Validated: true, Model: "m", ValueJson: `{"summary":"  "}`}, "empty summary"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAIGatewayRunnerWithResolver(responseRunner(t, tc.resp, nil))(context.Background(), validTask())
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestAIGatewayRunnerSuccessWithoutUsage(t *testing.T) {
	result, err := NewAIGatewayRunnerWithResolver(responseRunner(t, &inferencev1.RunResponse{
		Validated: true,
		Model:     "local-model",
		ValueJson: `{"summary":"condensed"}`,
	}, nil))(context.Background(), validTask())
	require.NoError(t, err)
	require.JSONEq(t, `{"model":"local-model","response":"condensed","eval_count":0}`, string(result))
}

// NewAIGatewayRunner itself only builds the discovery-backed resolver. Exercise
// construction here; the resolution path requires a live scenario URL and is
// covered by integration surfaces, not this unit package.
func TestNewAIGatewayRunnerReturnsTaskRunner(t *testing.T) {
	runner := NewAIGatewayRunner()
	require.NotNil(t, runner)
	_, err := runner(context.Background(), summarize.TaskRequest{Selector: "qwen3.5:9b", Instruction: "x", Source: "y", MaxOutputTokens: 1})
	require.ErrorContains(t, err, "not concrete model selectors")
}
