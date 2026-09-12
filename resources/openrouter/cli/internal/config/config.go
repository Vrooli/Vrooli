package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"

	resourceenv "github.com/vrooli/vrooli/resources/openrouter/cli/internal/env"
)

var ErrContentNotFound = errors.New("openrouter content not found")

// ContentStore owns repo-external OpenRouter prompt/config/route storage.
type ContentStore struct {
	Runtime resourceenv.Runtime
}

// ModelInfo is the normalized OpenRouter model entry shape returned by the CLI.
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

// ModelsResponse is the JSON shape expected by repo callers using
// `resource-openrouter content models --json`.
type ModelsResponse struct {
	Source        string      `json:"source"`
	FetchedAt     string      `json:"fetched_at"`
	DefaultModel  string      `json:"default_model"`
	ProviderCount int         `json:"provider_count"`
	Count         int         `json:"count"`
	Models        []ModelInfo `json:"models"`
}

// NewContentStore returns a content store rooted in the provided runtime.
func NewContentStore(runtime resourceenv.Runtime) ContentStore {
	return ContentStore{Runtime: runtime}
}

// EnsureInitialized creates the content storage directories.
func (s ContentStore) EnsureInitialized() error {
	return s.Runtime.EnsureDirectories()
}

// Add stores a named OpenRouter content asset.
func (s ContentStore) Add(kind, name string, payload []byte) error {
	path, err := s.pathFor(kind, name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

// Get returns a named OpenRouter content asset.
func (s ContentStore) Get(kind, name string) ([]byte, error) {
	path, err := s.pathFor(kind, name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}
	return data, nil
}

// List enumerates stored content names by kind or across all kinds.
func (s ContentStore) List(kind string) ([]string, error) {
	kinds := []string{"prompt", "config", "route"}
	if normalized := normalizeKind(kind); normalized != "" && normalized != "all" {
		kinds = []string{normalized}
	}

	var out []string
	for _, current := range kinds {
		dir, err := s.dirFor(current)
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			out = append(out, current+":"+strings.TrimSuffix(name, filepath.Ext(name)))
		}
	}
	sort.Strings(out)
	return out, nil
}

// Remove deletes a named OpenRouter content asset.
func (s ContentStore) Remove(kind, name string) error {
	path, err := s.pathFor(kind, name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrContentNotFound
		}
		return err
	}
	return nil
}

// SaveCredentialsFile writes a file-backed OpenRouter API key.
func SaveCredentialsFile(path, apiKey string) error {
	payload := struct {
		Data struct {
			APIKey string `json:"apiKey"`
		} `json:"data"`
	}{
		Data: struct {
			APIKey string `json:"apiKey"`
		}{APIKey: apiKey},
	}
	data, err := cliout.MarshalIndent(payload)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// ModelsEndpoint returns the OpenRouter model catalog URL.
func ModelsEndpoint(apiBase string) string {
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if base == "" {
		return ""
	}
	return base + "/models"
}

// ChatCompletionsEndpoint returns the OpenRouter chat completions URL.
func ChatCompletionsEndpoint(apiBase string) string {
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if base == "" {
		return ""
	}
	return base + "/chat/completions"
}

// ImagesEndpoint returns the OpenRouter image-generation URL (POST target).
func ImagesEndpoint(apiBase string) string {
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if base == "" {
		return ""
	}
	return base + "/images"
}

// ImagesModelsEndpoint returns the OpenRouter dedicated image-model catalog URL.
func ImagesModelsEndpoint(apiBase string) string {
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if base == "" {
		return ""
	}
	return base + "/images/models"
}

// NormalizeModelsResponse converts raw OpenRouter API JSON into the repo's
// expected normalized response shape.
func NormalizeModelsResponse(raw []byte, defaultModel, fetchedAt string, providerFilter, search string, limit int, manualModelsPath string) (ModelsResponse, error) {
	var payload struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Pricing     struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
				Request    string `json:"request"`
				Image      string `json:"image"`
			} `json:"pricing"`
			ContextLength       int      `json:"context_length"`
			SupportedParameters []string `json:"supported_parameters"`
			TopProvider         struct {
				ContextLength       int `json:"context_length"`
				MaxCompletionTokens int `json:"max_completion_tokens"`
			} `json:"top_provider"`
			Architecture struct {
				Modality         string   `json:"modality"`
				InputModalities  []string `json:"input_modalities"`
				OutputModalities []string `json:"output_modalities"`
			} `json:"architecture"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ModelsResponse{}, err
	}

	models := make([]ModelInfo, 0, len(payload.Data))
	for _, item := range payload.Data {
		model := ModelInfo{
			ID:          strings.TrimSpace(item.ID),
			Name:        strings.TrimSpace(item.Name),
			DisplayName: firstNonEmpty(strings.TrimSpace(item.Name), strings.TrimSpace(item.ID)),
			Provider:    providerForModel(item.ID),
			Description: strings.TrimSpace(item.Description),
			Pricing: &Pricing{
				Prompt:     parsePrice(item.Pricing.Prompt),
				Completion: parsePrice(item.Pricing.Completion),
				Request:    parsePrice(item.Pricing.Request),
				Image:      parsePrice(item.Pricing.Image),
			},
			ContextLength:       firstNonZero(item.TopProvider.ContextLength, item.ContextLength),
			MaxCompletionTokens: item.TopProvider.MaxCompletionTokens,
			Architecture: &Architecture{
				Modality: item.Architecture.Modality,
				Input:    append([]string(nil), item.Architecture.InputModalities...),
				Output:   append([]string(nil), item.Architecture.OutputModalities...),
			},
			SupportedParameters: append([]string(nil), item.SupportedParameters...),
		}
		if !matchesModel(model, providerFilter, search) {
			continue
		}
		models = append(models, model)
	}

	if manualModelsPath != "" {
		if manual, err := loadManualModels(manualModelsPath); err == nil {
			for _, model := range manual {
				if matchesModel(model, providerFilter, search) && !containsModel(models, model.ID) {
					models = append(models, model)
				}
			}
		}
	}

	sort.SliceStable(models, func(i, j int) bool {
		return strings.ToLower(models[i].ID) < strings.ToLower(models[j].ID)
	})
	if limit > 0 && len(models) > limit {
		models = models[:limit]
	}

	providers := make(map[string]struct{}, len(models))
	for _, model := range models {
		if model.Provider != "" {
			providers[model.Provider] = struct{}{}
		}
	}

	return ModelsResponse{
		Source:        "openrouter",
		FetchedAt:     fetchedAt,
		DefaultModel:  defaultModel,
		ProviderCount: len(providers),
		Count:         len(models),
		Models:        models,
	}, nil
}

func (s ContentStore) pathFor(kind, name string) (string, error) {
	dir, err := s.dirFor(kind)
	if err != nil {
		return "", err
	}
	name = sanitizeName(name)
	if name == "" {
		return "", fmt.Errorf("content name is required")
	}
	return filepath.Join(dir, name+".json"), nil
}

func (s ContentStore) dirFor(kind string) (string, error) {
	switch normalizeKind(kind) {
	case "prompt":
		return s.Runtime.PromptsDir, nil
	case "config":
		return s.Runtime.ConfigContentDir, nil
	case "route":
		return s.Runtime.RoutesDir, nil
	default:
		return "", fmt.Errorf("unknown content kind: %s", kind)
	}
}

func normalizeKind(kind string) string {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "", "all":
		return strings.TrimSpace(strings.ToLower(kind))
	case "prompt", "prompts":
		return "prompt"
	case "config", "configs":
		return "config"
	case "route", "routes":
		return "route"
	default:
		return strings.TrimSpace(strings.ToLower(kind))
	}
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "..", "")
	name = filepath.Base(name)
	return strings.TrimSpace(strings.TrimSuffix(name, filepath.Ext(name)))
}

func parsePrice(value string) float64 {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if value == "" {
		return 0
	}
	parsed, _ := json.Number(value).Float64()
	return parsed
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func providerForModel(id string) string {
	id = strings.TrimSpace(id)
	if index := strings.Index(id, "/"); index > 0 {
		return id[:index]
	}
	return ""
}

func matchesModel(model ModelInfo, providerFilter, search string) bool {
	if providerFilter = strings.TrimSpace(providerFilter); providerFilter != "" {
		prefix := providerFilter
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		if !strings.HasPrefix(model.ID, prefix) {
			return false
		}
	}
	if search = strings.ToLower(strings.TrimSpace(search)); search != "" {
		haystacks := []string{model.ID, model.Name, model.DisplayName, model.Description}
		matched := false
		for _, haystack := range haystacks {
			if strings.Contains(strings.ToLower(haystack), search) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func loadManualModels(path string) ([]ModelInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var models []ModelInfo
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, err
	}
	return models, nil
}

func containsModel(models []ModelInfo, id string) bool {
	for _, model := range models {
		if model.ID == id {
			return true
		}
	}
	return false
}
