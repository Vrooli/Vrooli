package summarizechain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"audio-tools/internal/summarize"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	GatewaySummaryRole            = "summarize.faithful"
	localSummaryPrivacyProvenance = "audio-tools.summary:caller-text-or-local-transcript-for-speech-output"
)

type InferenceRunner interface {
	Run(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error)
}

type InferenceResolver func(context.Context) (InferenceRunner, error)

// NewAIGatewayRunner adapts the maintained Summarizer CLI-shaped seam to the
// typed AI Gateway contract. The resource role selector is translated into one
// task role; concrete model selectors never cross the caller boundary.
func NewAIGatewayRunner() summarize.GatewayRunner {
	return NewAIGatewayRunnerWithResolver(func(ctx context.Context) (InferenceRunner, error) {
		url, err := discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		if err != nil {
			return nil, err
		}
		return inferenceconnect.NewInferenceServiceClient(&http.Client{Timeout: 3 * time.Minute}, url), nil
	})
}

func NewAIGatewayRunnerWithResolver(resolve InferenceResolver) summarize.GatewayRunner {
	return func(ctx context.Context, args []string, source string) ([]byte, error) {
		if resolve == nil {
			return nil, errors.New("ai-gateway resolver is required")
		}
		if model := argValue(args, "--model"); model != "" {
			return nil, errors.New("audio-tools local summary accepts policy roles, not concrete model selectors")
		}
		selector := argValue(args, "--role")
		if selector != "summarize.default" && selector != GatewaySummaryRole {
			return nil, fmt.Errorf("unsupported audio-tools summarization role %q", selector)
		}
		instruction := strings.TrimSpace(argValue(args, "--system"))
		if instruction == "" {
			return nil, errors.New("audio-tools summary task instruction is required")
		}
		outputTokens, err := strconv.ParseInt(argValue(args, "--max-tokens"), 10, 32)
		if err != nil || outputTokens <= 0 {
			return nil, errors.New("audio-tools summary output-token cap must be positive")
		}
		client, err := resolve(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve ai-gateway: %w", err)
		}
		response, err := client.Run(ctx, connect.NewRequest(&inferencev1.RunRequest{
			Role:              GatewaySummaryRole,
			Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
			PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
			PrivacyProvenance: localSummaryPrivacyProvenance,
			Instruction:       instruction,
			Source:            source,
			SchemaJson:        `{"type":"object","required":["summary"],"properties":{"summary":{"type":"string"}},"additionalProperties":false}`,
			MaxOutputTokens:   int32(outputTokens),
		}))
		if err != nil {
			return nil, fmt.Errorf("ai-gateway local summary: %w", err)
		}
		if response == nil || response.Msg == nil {
			return nil, errors.New("ai-gateway returned an empty summary response")
		}
		if gatewayError := response.Msg.GetError(); gatewayError != nil {
			return nil, fmt.Errorf("ai-gateway summary failed (%s): %s", gatewayError.GetCode(), gatewayError.GetMessage())
		}
		if strings.TrimSpace(response.Msg.GetModel()) == "" {
			return nil, errors.New("ai-gateway did not report the selected summary model")
		}
		var decoded struct {
			Summary string `json:"summary"`
		}
		decoder := json.NewDecoder(bytes.NewBufferString(response.Msg.GetValueJson()))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&decoded); err != nil {
			return nil, fmt.Errorf("decode ai-gateway summary result: %w", err)
		}
		if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			return nil, errors.New("ai-gateway summary result has trailing content")
		}
		if strings.TrimSpace(decoded.Summary) == "" {
			return nil, errors.New("ai-gateway returned an empty summary")
		}
		result := struct {
			Model      string `json:"model"`
			Response   string `json:"response"`
			EvalCount  int64  `json:"eval_count"`
			DoneReason string `json:"done_reason,omitempty"`
		}{Model: response.Msg.GetModel(), Response: decoded.Summary}
		if usage := response.Msg.GetUsage(); usage != nil {
			result.EvalCount = usage.GetOutputTokens()
		}
		return json.Marshal(result)
	}
}

func argValue(args []string, flag string) string {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == flag {
			return args[index+1]
		}
	}
	return ""
}
