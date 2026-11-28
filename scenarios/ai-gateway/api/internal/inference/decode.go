package inference

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
)

func requestAttachments(request ProviderRequest) []*sharedv1.Attachment {
	out := append([]*sharedv1.Attachment{}, request.Attachments...)
	for _, turn := range request.Turns {
		if turn != nil {
			out = append(out, turn.GetAttachments()...)
		}
	}
	return out
}

func providerPrompt(request ProviderRequest, coordinateConvention string) string {
	instruction := strings.TrimSpace(request.Instruction)
	if instruction == "" {
		instruction = "Produce the value described by the schema for the supplied source."
	}
	source := strings.TrimSpace(request.Source)
	if len(request.Turns) > 0 {
		var turns []string
		for _, turn := range request.Turns {
			if turn == nil {
				continue
			}
			turns = append(turns, strings.TrimSpace(turn.GetRole())+": "+strings.TrimSpace(turn.GetText()))
		}
		if len(turns) > 0 {
			source = strings.Join(turns, "\n")
		}
	}
	prompt := "You are a typed inference provider. Return only one JSON value that satisfies the supplied schema. Do not return Markdown fences or commentary.\n\n" +
		"Instruction (caller intent; separate from schema metadata):\n" + instruction + "\n\n" +
		"JSON Schema:\n" + strings.TrimSpace(providerSchema(request)) + "\n\n" +
		"Source:\n" + source
	if IsVisualLocateRole(request.Role) {
		prompt += "\n\nVisual grounding contract: return an object with found, bounds, and confidence. bounds must contain exactly four numbers in the model-declared coordinate convention " + coordinateConvention + ". For normalized_1000, every x and y value is in the 0..1000 space, not the 0..1 space. For absolute_pixels, use the submitted image pixel space. If the target is not found, set found to false and bounds to [0,0,0,0]. Return confidence unchanged as a number from 0 to 1."
	}
	return prompt
}

func providerSchema(request ProviderRequest) string {
	if !IsVisualLocateRole(request.Role) {
		return request.SchemaJSON
	}
	// The provider sees its declared coordinate space. The gateway converts this
	// shape to the caller's canonical [0,1] contract before local validation.
	return `{"type":"object","required":["found","bounds","confidence"],"properties":{"found":{"type":"boolean"},"bounds":{"type":"array","items":{"type":"number"}},"confidence":{"type":"number","minimum":0,"maximum":1}}}`
}

type providerUsage struct {
	input, output, cost                               int64
	rootOperationID                                   string
	creditsCharged                                    *int64
	accountingStatus                                  string
	inputProvenance, outputProvenance, costProvenance inferencev1.UsageProvenance
}

func decodeProviderResponse(provider, raw, prompt string) (string, providerUsage, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case providers.ProviderOllama:
		return decodeOllamaResponse(raw, prompt)
	case providers.ProviderOpenRouter, providers.ProviderOrcaRouter, providers.ProviderAbliteration:
		return decodeRouterResponse(provider, raw, prompt)
	case providers.ProviderMetered:
		return decodeMeteredResponse(raw, prompt)
	default:
		return "", providerUsage{}, fmt.Errorf("unsupported provider %q", provider)
	}
}

func decodeOllamaResponse(raw, prompt string) (string, providerUsage, error) {
	var response struct {
		Response        string `json:"response"`
		PromptEvalCount *int64 `json:"prompt_eval_count"`
		EvalCount       *int64 `json:"eval_count"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		return "", providerUsage{}, fmt.Errorf("decode Ollama response: %w", err)
	}
	if strings.TrimSpace(response.Response) == "" {
		return "", providerUsage{}, errors.New("Ollama returned an empty response")
	}
	usage := providerUsage{input: estimateTokens(prompt), inputProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE, costProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}
	usage.input, usage.inputProvenance = reportedCount(response.PromptEvalCount, usage.input, usage.inputProvenance)
	usage.output, usage.outputProvenance = reportedCount(response.EvalCount, 0, inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN)
	return ExtractJSONValue(response.Response), usage, nil
}

func decodeRouterResponse(provider, raw, prompt string) (string, providerUsage, error) {
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage *routerResponseUsage `json:"usage"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		return "", providerUsage{}, fmt.Errorf("decode %s response: %w", provider, err)
	}
	if len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
		return "", providerUsage{}, fmt.Errorf("%s returned no completion", provider)
	}
	usage := providerUsage{}
	usage.input, usage.inputProvenance = reportedCount(responsePromptTokens(response.Usage), estimateTokens(prompt), inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE)
	usage.output, usage.outputProvenance = reportedCount(responseCompletionTokens(response.Usage), 0, inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN)
	usage.cost, usage.costProvenance = routerCost(response.Usage)
	return ExtractJSONValue(response.Choices[0].Message.Content), usage, nil
}

type routerResponseUsage struct {
	PromptTokens     *int64   `json:"prompt_tokens"`
	CompletionTokens *int64   `json:"completion_tokens"`
	Cost             *float64 `json:"cost"`
}

func responsePromptTokens(usage *routerResponseUsage) *int64 {
	if usage == nil {
		return nil
	}
	return usage.PromptTokens
}

func responseCompletionTokens(usage *routerResponseUsage) *int64 {
	if usage == nil {
		return nil
	}
	return usage.CompletionTokens
}

func routerCost(usage *routerResponseUsage) (int64, inferencev1.UsageProvenance) {
	unknown := inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
	if usage == nil || usage.Cost == nil || math.IsNaN(*usage.Cost) || math.IsInf(*usage.Cost, 0) || *usage.Cost < 0 {
		return 0, unknown
	}
	return int64(math.Round(*usage.Cost * 1_000_000)), inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
}

func reportedCount(value *int64, fallback int64, provenance inferencev1.UsageProvenance) (int64, inferencev1.UsageProvenance) {
	if value != nil && *value >= 0 {
		return *value, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
	}
	return fallback, provenance
}

func meteredCredits(status string, charged *int64) *int64 {
	if (status == "" || status == "settled") && charged != nil && *charged >= 0 {
		value := *charged
		return &value
	}
	return nil
}

func recognizedAccountingStatus(status string) string {
	if status == "settled" || status == "pending_reconciliation" {
		return status
	}
	return ""
}

func decodeMeteredResponse(raw, prompt string) (string, providerUsage, error) {
	var response struct {
		RootOperationID  string `json:"root_operation_id"`
		Content          string `json:"content"`
		PromptTokens     *int64 `json:"prompt_tokens"`
		CompletionTokens *int64 `json:"completion_tokens"`
		CreditsCharged   *int64 `json:"credits_charged"`
		AccountingStatus string `json:"accounting_status"`
		ActualCostMicros *int64 `json:"actual_cost_micros"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		return "", providerUsage{}, fmt.Errorf("decode metered inference response: %w", err)
	}
	if strings.TrimSpace(response.Content) == "" {
		return "", providerUsage{}, errors.New("metered inference returned an empty response")
	}
	usage := providerUsage{rootOperationID: response.RootOperationID, costProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}
	usage.input, usage.inputProvenance = reportedCount(response.PromptTokens, estimateTokens(prompt), inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE)
	usage.output, usage.outputProvenance = reportedCount(response.CompletionTokens, 0, inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN)
	usage.creditsCharged = meteredCredits(response.AccountingStatus, response.CreditsCharged)
	usage.accountingStatus = recognizedAccountingStatus(response.AccountingStatus)
	if response.ActualCostMicros != nil && *response.ActualCostMicros >= 0 {
		usage.cost, usage.costProvenance = *response.ActualCostMicros, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
	}
	return ExtractJSONValue(response.Content), usage, nil
}

func estimateTokens(text string) int64 {
	count := int64(len([]rune(strings.TrimSpace(text))) / 4)
	if count < 1 {
		return 1
	}
	return count
}
