package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"ai-gateway/internal/providers"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// Typed inference asks for a schema-shaped value, not prose, so it samples
// deterministically. Without this, identical sources classify differently
// between calls, which makes results unreproducible and defeats caching.
// Providers whose CLI exposes no temperature control simply ignore it.
var deterministicTemperature = 0.0

type ResourceRepository struct {
	catalog  RoleCatalog
	adapters map[string]providers.Adapter
}

func NewResourceRepository(catalog RoleCatalog, adapters []providers.Adapter) (*ResourceRepository, error) {
	if err := catalog.Validate(); err != nil {
		return nil, err
	}
	repository := &ResourceRepository{catalog: catalog, adapters: make(map[string]providers.Adapter, len(adapters))}
	for _, adapter := range adapters {
		repository.adapters[strings.ToLower(strings.TrimSpace(adapter.Provider))] = adapter
	}
	return repository, nil
}

func (r *ResourceRepository) Run(ctx context.Context, request ProviderRequest) (ProviderResult, error) {
	definition, ok := r.catalog.Roles[strings.TrimSpace(request.Role)]
	if !ok {
		return ProviderResult{}, fmt.Errorf("%w: inference role %q is not declared", ErrUnavailable, request.Role)
	}
	var failures []string
	for _, candidate := range definition.Candidates {
		provider := strings.ToLower(strings.TrimSpace(candidate.Provider))
		if request.Profile == sharedv1.Profile_PROFILE_REMOTE_ONLY && provider != providers.ProviderOpenRouter && provider != providers.ProviderMetered {
			continue
		}
		if request.Profile == sharedv1.Profile_PROFILE_LOCAL_ONLY && provider != providers.ProviderOllama {
			continue
		}
		adapter, ok := r.adapters[strings.ToLower(strings.TrimSpace(candidate.Provider))]
		if !ok {
			failures = append(failures, candidate.Provider+": adapter unavailable")
			continue
		}
		resolved, err := adapter.ResolveRole(ctx, candidate.ResourceRole)
		if err != nil {
			failures = append(failures, candidate.Provider+": "+err.Error())
			continue
		}
		execution, err := adapter.Execute(ctx, providers.ExecutionRequest{
			Kind:        sharedv1.RequestKind_REQUEST_KIND_STRUCTURED_EXTRACTION,
			Role:        candidate.ResourceRole,
			Profile:     request.Profile,
			InputText:   providerPrompt(request, resolved.CoordinateConvention),
			Timeout:     definition.Timeout(),
			Temperature: &deterministicTemperature,
			SchemaJSON:  providerSchema(request),
			Attachments: requestAttachments(request),
		})
		if err != nil {
			failures = append(failures, candidate.Provider+": "+err.Error())
			continue
		}
		prompt := providerPrompt(request, resolved.CoordinateConvention)
		value, inputTokens, outputTokens, costMicros, err := decodeProviderResponse(candidate.Provider, execution.OutputText, prompt)
		if err != nil {
			failures = append(failures, candidate.Provider+": "+err.Error())
			continue
		}
		return ProviderResult{
			ValueJSON: value, Provider: resolved.Provider, Model: resolved.Model,
			CoordinateConvention: resolved.CoordinateConvention,
			InputTokens:          inputTokens, OutputTokens: outputTokens, CostMicros: costMicros,
		}, nil
	}
	return ProviderResult{}, fmt.Errorf("%w: no inference candidate completed for role %q (%s)", ErrUnavailable, request.Role, strings.Join(failures, "; "))
}

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
	if strings.TrimSpace(request.Role) == "locate.visual" {
		prompt += "\n\nVisual grounding contract: return an object with found, bounds, and confidence. bounds must contain exactly four numbers in the model-declared coordinate convention " + coordinateConvention + ". For normalized_1000, every x and y value is in the 0..1000 space, not the 0..1 space. For absolute_pixels, use the submitted image pixel space. If the target is not found, set found to false and bounds to [0,0,0,0]. Return confidence unchanged as a number from 0 to 1."
	}
	return prompt
}

func providerSchema(request ProviderRequest) string {
	if strings.TrimSpace(request.Role) != "locate.visual" {
		return request.SchemaJSON
	}
	// The provider sees its declared coordinate space. The gateway converts this
	// shape to the caller's canonical [0,1] contract before local validation.
	return `{"type":"object","required":["found","bounds","confidence"],"properties":{"found":{"type":"boolean"},"bounds":{"type":"array","items":{"type":"number"}},"confidence":{"type":"number","minimum":0,"maximum":1}}}`
}

func decodeProviderResponse(provider, raw, prompt string) (string, int64, int64, int64, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case providers.ProviderOllama:
		var response struct {
			Response  string `json:"response"`
			EvalCount int64  `json:"eval_count"`
		}
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			return "", 0, 0, 0, fmt.Errorf("decode Ollama response: %w", err)
		}
		if strings.TrimSpace(response.Response) == "" {
			return "", 0, 0, 0, errors.New("Ollama returned an empty response")
		}
		return ExtractJSONValue(response.Response), estimateTokens(prompt), response.EvalCount, 0, nil
	case providers.ProviderOpenRouter:
		var response struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int64   `json:"prompt_tokens"`
				CompletionTokens int64   `json:"completion_tokens"`
				Cost             float64 `json:"cost"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			return "", 0, 0, 0, fmt.Errorf("decode OpenRouter response: %w", err)
		}
		if len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
			return "", 0, 0, 0, errors.New("OpenRouter returned no completion")
		}
		inputTokens := response.Usage.PromptTokens
		if inputTokens == 0 {
			inputTokens = estimateTokens(prompt)
		}
		return ExtractJSONValue(response.Choices[0].Message.Content), inputTokens, response.Usage.CompletionTokens, int64(math.Round(response.Usage.Cost * 1_000_000)), nil
	case providers.ProviderMetered:
		var response struct {
			Content          string `json:"content"`
			PromptTokens     int64  `json:"prompt_tokens"`
			CompletionTokens int64  `json:"completion_tokens"`
			CreditsCharged   int64  `json:"credits_charged"`
		}
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			return "", 0, 0, 0, fmt.Errorf("decode metered inference response: %w", err)
		}
		if strings.TrimSpace(response.Content) == "" {
			return "", 0, 0, 0, errors.New("metered inference returned an empty response")
		}
		inputTokens := response.PromptTokens
		if inputTokens == 0 {
			inputTokens = estimateTokens(prompt)
		}
		return ExtractJSONValue(response.Content), inputTokens, response.CompletionTokens, 0, nil
	default:
		return "", 0, 0, 0, fmt.Errorf("unsupported provider %q", provider)
	}
}

func estimateTokens(text string) int64 {
	count := int64(len([]rune(strings.TrimSpace(text))) / 4)
	if count < 1 {
		return 1
	}
	return count
}
