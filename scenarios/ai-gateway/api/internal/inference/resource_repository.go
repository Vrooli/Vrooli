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
	prompt := providerPrompt(request)
	var failures []string
	for _, candidate := range definition.Candidates {
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
			InputText:   prompt,
			Timeout:     definition.Timeout(),
			Temperature: &deterministicTemperature,
		})
		if err != nil {
			failures = append(failures, candidate.Provider+": "+err.Error())
			continue
		}
		value, inputTokens, outputTokens, costMicros, err := decodeProviderResponse(candidate.Provider, execution.OutputText, prompt)
		if err != nil {
			failures = append(failures, candidate.Provider+": "+err.Error())
			continue
		}
		return ProviderResult{
			ValueJSON: value, Provider: resolved.Provider, Model: resolved.Model,
			InputTokens: inputTokens, OutputTokens: outputTokens, CostMicros: costMicros,
		}, nil
	}
	return ProviderResult{}, fmt.Errorf("%w: no inference candidate completed for role %q (%s)", ErrUnavailable, request.Role, strings.Join(failures, "; "))
}

func providerPrompt(request ProviderRequest) string {
	instruction := strings.TrimSpace(request.Instruction)
	if instruction == "" {
		instruction = "Produce the value described by the schema for the supplied source."
	}
	return "You are a typed inference provider. Return only one JSON value that satisfies the supplied schema. Do not return Markdown fences or commentary.\n\n" +
		"Instruction (caller intent; separate from schema metadata):\n" + instruction + "\n\n" +
		"JSON Schema:\n" + strings.TrimSpace(request.SchemaJSON) + "\n\n" +
		"Source:\n" + request.Source
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
