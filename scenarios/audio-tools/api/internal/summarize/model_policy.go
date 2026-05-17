package summarize

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"audio-tools/internal/httpc"
)

const DefaultSummarizeModel = "llama3.2:3b"

type ModelDecisionReason string

const (
	ModelDecisionKept             ModelDecisionReason = "kept"
	ModelDecisionEmptyDefault     ModelDecisionReason = "empty_default"
	ModelDecisionUnsafeReasoning  ModelDecisionReason = "unsafe_reasoning_default"
	ModelDecisionMissingFallback  ModelDecisionReason = "missing_fallback"
	ModelDecisionInstalledDefault ModelDecisionReason = "installed_default"
)

type ModelDecision struct {
	Model  string
	Reason ModelDecisionReason
}

type OllamaModel struct {
	Name          string
	SizeBytes     int64
	ParameterSize string
}

type SummarizeModelInfo struct {
	ID              string
	DisplayName     string
	Family          string
	ParameterSize   string
	Installed       bool
	Recommended     bool
	DefaultEligible bool
	Reasoning       bool
	StatusLabel     string
	PullCommand     string
	SizeBytes       int64
	SourceURL       string
	Notes           string
}

var knownSummarizeModels = []SummarizeModelInfo{
	{
		ID:              "gemma3:4b",
		DisplayName:     "Gemma 3 4B",
		Family:          "gemma3",
		ParameterSize:   "4B",
		Recommended:     true,
		DefaultEligible: true,
		SourceURL:       "https://ollama.com/library/gemma3",
		Notes:           "Current efficient instruction model; benchmark locally before making it the default.",
	},
	{
		ID:              "gemma3n:e2b",
		DisplayName:     "Gemma 3n E2B",
		Family:          "gemma3n",
		ParameterSize:   "effective 2B",
		Recommended:     true,
		DefaultEligible: true,
		SourceURL:       "https://ollama.com/library/gemma3n",
		Notes:           "Designed for efficient execution on everyday devices; requires local benchmark evidence.",
	},
	{
		ID:              "llama3.2:3b",
		DisplayName:     "Llama 3.2 3B",
		Family:          "llama3.2",
		ParameterSize:   "3B",
		Recommended:     true,
		DefaultEligible: true,
		SourceURL:       "https://ollama.com/library/llama3.2",
		Notes:           "Installed fallback validated for fast local TTS summarization.",
	},
	{
		ID:              "llama3.2:1b",
		DisplayName:     "Llama 3.2 1B",
		Family:          "llama3.2",
		ParameterSize:   "1B",
		Recommended:     true,
		DefaultEligible: true,
		SourceURL:       "https://ollama.com/library/llama3.2",
		Notes:           "Very small fallback candidate; verify quality before preferring it.",
	},
	{
		ID:              "qwen2.5:3b",
		DisplayName:     "Qwen 2.5 3B",
		Family:          "qwen2.5",
		ParameterSize:   "3B",
		Recommended:     true,
		DefaultEligible: true,
		SourceURL:       "https://ollama.com/library/qwen2.5",
		Notes:           "Non-reasoning local candidate to include in latency and quality benchmarks.",
	},
	{
		ID:          "mistral:latest",
		DisplayName: "Mistral",
		Family:      "mistral",
		Recommended: false,
		SourceURL:   "https://ollama.com/library/mistral",
		Notes:       "Installed benchmark candidate; not preferred as a default without local evidence.",
	},
	{
		ID:              "phi4-mini:3.8b",
		DisplayName:     "Phi-4 Mini 3.8B",
		Family:          "phi4-mini",
		ParameterSize:   "3.8B",
		Recommended:     true,
		DefaultEligible: true,
		SourceURL:       "https://ollama.com/library/phi4-mini",
		Notes:           "Latency-constrained candidate; benchmark locally before enabling as default.",
	},
	{
		ID:            "qwen3:4b",
		DisplayName:   "Qwen3 4B",
		Family:        "qwen3",
		ParameterSize: "4B",
		Reasoning:     true,
		SourceURL:     "https://ollama.com/library/qwen3:4b",
		Notes:         "Reasoning-capable; too slow/noisy for default TTS summaries.",
	},
	{
		ID:            "qwen3:1.7b",
		DisplayName:   "Qwen3 1.7B",
		Family:        "qwen3",
		ParameterSize: "1.7B",
		Reasoning:     true,
		SourceURL:     "https://ollama.com/library/qwen3",
		Notes:         "Reasoning-capable; only use explicitly after benchmark validation.",
	},
	{
		ID:            "deepseek-r1:8b",
		DisplayName:   "DeepSeek R1 8B",
		Family:        "deepseek-r1",
		ParameterSize: "8B",
		Reasoning:     true,
		SourceURL:     "https://ollama.com/library/deepseek-r1",
		Notes:         "Reasoning model; excluded from fast TTS defaults.",
	},
}

func KnownSummarizeModels() []SummarizeModelInfo {
	out := make([]SummarizeModelInfo, len(knownSummarizeModels))
	copy(out, knownSummarizeModels)
	return out
}

func IsReasoningModel(id string) bool {
	n := normalizeModelID(id)
	return strings.HasPrefix(n, "qwen3:") ||
		strings.HasPrefix(n, "deepseek-r1:") ||
		strings.Contains(n, "reasoning")
}

func IsDefaultEligibleSummarizeModel(id string) bool {
	n := normalizeModelID(id)
	for _, m := range knownSummarizeModels {
		if m.ID == n {
			return m.DefaultEligible && !m.Reasoning
		}
	}
	return n != "" && !IsReasoningModel(n)
}

func CoerceUnsafeStoredModel(id string, installed []OllamaModel) ModelDecision {
	n := normalizeModelID(id)
	if n == "" {
		return ModelDecision{Model: ResolveDefaultSummarizeModel(installed), Reason: ModelDecisionEmptyDefault}
	}
	if IsReasoningModel(n) {
		return ModelDecision{Model: ResolveDefaultSummarizeModel(installed), Reason: ModelDecisionUnsafeReasoning}
	}
	if len(installed) > 0 && !modelInstalled(n, installed) {
		return ModelDecision{Model: ResolveDefaultSummarizeModel(installed), Reason: ModelDecisionMissingFallback}
	}
	return ModelDecision{Model: n, Reason: ModelDecisionKept}
}

func ResolveDefaultSummarizeModel(installed []OllamaModel) string {
	if len(installed) == 0 {
		return DefaultSummarizeModel
	}
	for _, candidate := range []string{"gemma3:4b", "gemma3n:e2b", DefaultSummarizeModel, "llama3.2:1b", "qwen2.5:3b", "phi4-mini:3.8b"} {
		if modelInstalled(candidate, installed) {
			return candidate
		}
	}
	return DefaultSummarizeModel
}

func ListSummarizeModels(ctx context.Context, baseURL string, doer httpc.Doer) ([]SummarizeModelInfo, error) {
	installed, err := FetchOllamaModels(ctx, baseURL, doer)
	return MergeSummarizeModels(installed), err
}

func MergeSummarizeModels(installed []OllamaModel) []SummarizeModelInfo {
	byID := make(map[string]SummarizeModelInfo, len(knownSummarizeModels)+len(installed))
	order := make([]string, 0, len(knownSummarizeModels)+len(installed))
	for _, m := range knownSummarizeModels {
		m.ID = normalizeModelID(m.ID)
		m.PullCommand = "ollama pull " + m.ID
		m.StatusLabel = "Recommended, not installed"
		if m.Reasoning {
			m.DefaultEligible = false
			m.StatusLabel = "Reasoning model, not installed"
		}
		byID[m.ID] = m
		order = append(order, m.ID)
	}
	for _, local := range installed {
		id := normalizeModelID(local.Name)
		if id == "" {
			continue
		}
		m, ok := byID[id]
		if !ok {
			m = SummarizeModelInfo{
				ID:              id,
				DisplayName:     id,
				DefaultEligible: false,
				Reasoning:       IsReasoningModel(id),
				PullCommand:     "ollama pull " + id,
			}
			order = append(order, id)
		}
		m.Installed = true
		m.SizeBytes = local.SizeBytes
		if local.ParameterSize != "" {
			m.ParameterSize = local.ParameterSize
		}
		m.Reasoning = m.Reasoning || IsReasoningModel(id)
		m.DefaultEligible = m.DefaultEligible && !m.Reasoning
		m.StatusLabel = "Installed"
		if m.Reasoning {
			m.StatusLabel = "Installed reasoning model"
		} else if m.Recommended {
			m.StatusLabel = "Installed, recommended"
		}
		byID[id] = m
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := byID[order[i]], byID[order[j]]
		if a.Installed != b.Installed {
			return a.Installed
		}
		if a.DefaultEligible != b.DefaultEligible {
			return a.DefaultEligible
		}
		if a.Recommended != b.Recommended {
			return a.Recommended
		}
		return a.ID < b.ID
	})
	out := make([]SummarizeModelInfo, 0, len(order))
	seen := map[string]bool{}
	for _, id := range order {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, byID[id])
	}
	return out
}

func FetchOllamaModels(ctx context.Context, baseURL string, doer httpc.Doer) ([]OllamaModel, error) {
	if doer == nil {
		doer = httpc.DefaultDoer()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("create ollama tags request: %w", err)
	}
	resp, err := doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama tags unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama tags returned status %d", resp.StatusCode)
	}
	var tags struct {
		Models []struct {
			Name    string `json:"name"`
			Size    int64  `json:"size"`
			Details struct {
				ParameterSize string `json:"parameter_size"`
			} `json:"details"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("decode ollama tags: %w", err)
	}
	out := make([]OllamaModel, 0, len(tags.Models))
	for _, m := range tags.Models {
		out = append(out, OllamaModel{Name: m.Name, SizeBytes: m.Size, ParameterSize: m.Details.ParameterSize})
	}
	return out, nil
}

func modelInstalled(id string, installed []OllamaModel) bool {
	n := normalizeModelID(id)
	for _, m := range installed {
		if normalizeModelID(m.Name) == n {
			return true
		}
	}
	return false
}

func normalizeModelID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}
