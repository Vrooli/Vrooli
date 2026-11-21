package openai

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ModelInfo is the normalized OpenAI /models entry shape.
type ModelInfo struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name,omitempty"`
	DisplayName         string        `json:"display_name,omitempty"`
	Provider            string        `json:"provider,omitempty"`
	Description         string        `json:"description,omitempty"`
	Pricing             *Pricing      `json:"pricing,omitempty"`
	ContextLength       int           `json:"context_length,omitempty"`
	MaxCompletionTokens int           `json:"max_completion_tokens,omitempty"`
	Architecture        *Architecture `json:"architecture,omitempty"`
	SupportedParameters []string      `json:"supported_parameters,omitempty"`
}

type Pricing struct {
	Prompt     float64 `json:"prompt,omitempty"`
	Completion float64 `json:"completion,omitempty"`
	Request    float64 `json:"request,omitempty"`
	Image      float64 `json:"image,omitempty"`
}

type Architecture struct {
	Modality string   `json:"modality,omitempty"`
	Input    []string `json:"input,omitempty"`
	Output   []string `json:"output,omitempty"`
}

// ModelsResponse is the normalized model listing returned by provider CLIs.
type ModelsResponse struct {
	Source        string      `json:"source"`
	FetchedAt     string      `json:"fetched_at"`
	DefaultModel  string      `json:"default_model,omitempty"`
	ProviderCount int         `json:"provider_count"`
	Count         int         `json:"count"`
	Models        []ModelInfo `json:"models"`
}

// NormalizeModels decodes an OpenAI /models response. Providers that expose
// extra metadata keep it; missing fields are left empty rather than guessed.
func NormalizeModels(raw []byte, source, fetchedAt string, limit int) (ModelsResponse, error) {
	var envelope struct {
		Data   []ModelInfo `json:"data"`
		Models []ModelInfo `json:"models"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ModelsResponse{}, fmt.Errorf("decode models response: %w", err)
	}
	models := envelope.Data
	if len(models) == 0 {
		models = envelope.Models
	}
	for i := range models {
		models[i].ID = strings.TrimSpace(models[i].ID)
		if models[i].Provider == "" {
			if slash := strings.Index(models[i].ID, "/"); slash > 0 {
				models[i].Provider = models[i].ID[:slash]
			}
		}
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	total := len(models)
	if limit > 0 && len(models) > limit {
		models = models[:limit]
	}
	providers := map[string]struct{}{}
	for _, model := range models {
		if model.Provider != "" {
			providers[model.Provider] = struct{}{}
		}
	}
	return ModelsResponse{
		Source:        source,
		FetchedAt:     fetchedAt,
		ProviderCount: len(providers),
		Count:         total,
		Models:        models,
	}, nil
}

// ModelIDs returns the ids from a normalized listing.
func (r ModelsResponse) ModelIDs() []string {
	out := make([]string, 0, len(r.Models))
	for _, model := range r.Models {
		out = append(out, model.ID)
	}
	return out
}
