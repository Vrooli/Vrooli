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

const DefaultSummarizeModel = "summarize.default"

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
		ID:              DefaultSummarizeModel,
		DisplayName:     "Default summarization role",
		Family:          "resource-ollama-role",
		Recommended:     true,
		DefaultEligible: true,
		SourceURL:       "resources/ollama/model-policy.json",
		Notes:           "Central policy role for text summarization. The concrete model is resolved by resource-ollama.",
	},
	{
		ID:              "chat.small",
		DisplayName:     "Small chat role",
		Family:          "resource-ollama-role",
		Recommended:     true,
		DefaultEligible: true,
		SourceURL:       "resources/ollama/model-policy.json",
		Notes:           "Lower-latency fallback role resolved by resource-ollama.",
	},
}

func KnownSummarizeModels() []SummarizeModelInfo {
	out := make([]SummarizeModelInfo, len(knownSummarizeModels))
	copy(out, knownSummarizeModels)
	return out
}

func IsReasoningModel(id string) bool {
	n := normalizeModelID(id)
	return strings.Contains(n, "reasoning")
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
	return DefaultSummarizeModel
}

// SelectorIsRole reports whether a summarize selector names a logical policy
// role (resolved by resource-ollama) rather than a concrete Ollama model tag.
// A concrete tag always carries a ":" version separator (e.g. "qwen3.5:9b");
// a role does not (e.g. "summarize.default"). This is the single source of
// truth for the role-vs-model distinction shared by the summarizer (which
// passes --role vs --model to the gateway) and the capability health checker
// (which decides whether to resolve a role before verifying installation).
// Keeping both call sites on this one rule is what prevents the work path and
// the health path from diverging.
func SelectorIsRole(selector string) bool {
	selector = strings.TrimSpace(selector)
	return selector != "" && !strings.Contains(selector, ":")
}

// ResolveRoleModel resolves a logical role selector (e.g. "summarize.default")
// to the concrete Ollama model tag via resource-ollama's policy SSOT — the same
// authority the gateway consults at chat time. It lets the health checker verify
// the model that summarization would actually use, instead of comparing a role
// name against installed tags (which never matches).
func ResolveRoleModel(ctx context.Context, role string) (string, error) {
	return resolveRoleModel(ctx, defaultOllamaGatewayBin, role)
}

func resolveRoleModel(ctx context.Context, bin, role string) (string, error) {
	out, err := runGatewayCLI(ctx, bin, []string{"policy", "resolve", "--role", role, "--json"}, "")
	if err != nil {
		return "", fmt.Errorf("resource-ollama policy resolve: %w", err)
	}
	var res struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return "", fmt.Errorf("decode policy resolve response: %w", err)
	}
	if strings.TrimSpace(res.Model) == "" {
		return "", fmt.Errorf("policy resolve returned no model for role %q", role)
	}
	return res.Model, nil
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
		m.PullCommand = "resource-ollama ensure --role " + m.ID
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
