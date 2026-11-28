package summarizechain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	GatewaySummaryRole            = summarize.AudioToolsSummarizeRoleNamespace
	localSummaryPrivacyProvenance = "audio-tools.summary:caller-text-or-local-transcript-for-speech-output"
	maxGatewaySummaryResultBytes  = 4 << 20
)

type InferenceRunner interface {
	Run(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error)
}

type InferenceResolver func(context.Context) (InferenceRunner, error)

// NewAIGatewayRunner adapts Audio Tools' owner-typed task request directly to
// its application-scoped AI Gateway summary role. The resource role selector
// is translated into one task role;
// CLI serialization and parsing stay on the resource-ollama experiment path.
func NewAIGatewayRunner() summarize.TaskRunner {
	return NewAIGatewayRunnerWithResolver(func(ctx context.Context) (InferenceRunner, error) {
		url, err := discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		if err != nil {
			return nil, err
		}
		return inferenceconnect.NewInferenceServiceClient(&http.Client{Timeout: 3 * time.Minute}, url), nil
	})
}

func NewAIGatewayRunnerWithResolver(resolve InferenceResolver) summarize.TaskRunner {
	return func(ctx context.Context, task summarize.TaskRequest) ([]byte, error) {
		if resolve == nil {
			return nil, errors.New("ai-gateway resolver is required")
		}
		if !task.SelectorIsRole {
			return nil, errors.New("audio-tools local summary accepts policy roles, not concrete model selectors")
		}
		selector := strings.TrimSpace(task.Selector)
		if !summarize.IsAudioToolsSummarizeRole(selector) {
			return nil, fmt.Errorf("unsupported audio-tools summarization role %q", selector)
		}
		gatewayRole := selector
		instruction := strings.TrimSpace(task.Instruction)
		if instruction == "" {
			return nil, errors.New("audio-tools summary task instruction is required")
		}
		if strings.TrimSpace(task.Source) == "" {
			return nil, errors.New("audio-tools summary source is required")
		}
		if task.MaxOutputTokens <= 0 || int64(task.MaxOutputTokens) > int64(^uint32(0)>>1) {
			return nil, errors.New("audio-tools summary output-token cap must be positive")
		}
		if task.ContextTier != 0 || task.ExpectedModel != "" || task.ExpectedArtifactDigest != "" || task.RightsDecisionSHA256 != "" || task.RightsEvaluatedAt != "" || task.RightsReceiptSHA256 != "" || task.PolicyFileSHA256 != "" {
			return nil, errors.New("audio-tools AI Gateway task cannot carry resource-specific context or rights selectors")
		}
		client, err := resolve(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve ai-gateway: %w", err)
		}
		response, err := client.Run(ctx, connect.NewRequest(&inferencev1.RunRequest{
			Role:              gatewayRole,
			Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
			PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
			PrivacyProvenance: localSummaryPrivacyProvenance,
			Instruction:       instruction,
			Source:            task.Source,
			SchemaJson:        `{"type":"object","required":["summary"],"properties":{"summary":{"type":"string"}},"additionalProperties":false}`,
			MaxOutputTokens:   int32(task.MaxOutputTokens),
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
		if !response.Msg.GetValidated() || strings.TrimSpace(response.Msg.GetValueJson()) == "" {
			return nil, errors.New("ai-gateway returned an unvalidated summary result")
		}
		if strings.TrimSpace(response.Msg.GetModel()) == "" {
			return nil, errors.New("ai-gateway did not report the selected summary model")
		}
		valueJSON := response.Msg.GetValueJson()
		if len(valueJSON) > maxGatewaySummaryResultBytes {
			return nil, fmt.Errorf("ai-gateway summary result exceeds %d-byte limit", maxGatewaySummaryResultBytes)
		}
		var decoded struct {
			Summary string `json:"summary"`
		}
		decoder := json.NewDecoder(strings.NewReader(valueJSON))
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
