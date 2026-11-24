package summarize

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	DefaultSummarizeModel = "summarize.audio_tools"
)

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

func SummarizeRoleOptions(roles []DeclaredSummaryRole) []SummarizeModelInfo {
	options := make([]SummarizeModelInfo, 0, len(roles))
	seen := map[string]struct{}{}
	for _, role := range roles {
		id := strings.TrimSpace(role.ID)
		if !IsAudioToolsSummarizeRole(id) {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		notes := strings.TrimSpace(role.Description)
		if notes == "" {
			notes = "AI Gateway-declared Audio Tools local summary role. Review the role's model-specific quality evidence before using it for fact-dense input."
		}
		options = append(options, SummarizeModelInfo{
			ID: id, DisplayName: id, Family: "ai-gateway-role",
			Recommended: id == DefaultSummarizeModel, DefaultEligible: true,
			StatusLabel: "Declared local summary role", Notes: notes,
		})
	}
	sort.Slice(options, func(i, j int) bool {
		if options[i].ID == DefaultSummarizeModel {
			return true
		}
		if options[j].ID == DefaultSummarizeModel {
			return false
		}
		return options[i].ID < options[j].ID
	})
	return options
}

func IsReasoningModel(id string) bool {
	n := normalizeModelID(id)
	return strings.Contains(n, "reasoning")
}

func IsDefaultEligibleSummarizeModel(id string) bool {
	n := normalizeModelID(id)
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
// a role does not (e.g. "summarize.audio_tools"). This is the single source of
// truth for the role-vs-model distinction shared by the summarizer (which
// passes --role vs --model to the gateway) and the capability health checker
// (which decides whether to resolve a role before verifying installation).
// Keeping both call sites on this one rule is what prevents the work path and
// the health path from diverging.
func SelectorIsRole(selector string) bool {
	selector = strings.TrimSpace(selector)
	return selector != "" && !strings.Contains(selector, ":")
}

// ResolveRoleModel resolves a logical role selector (e.g. "summarize.audio_tools")
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

func ListSummarizeModels(ctx context.Context) ([]SummarizeModelInfo, error) {
	roles, err := ListAudioToolsSummarizeRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list declared Audio Tools summarize roles: %w", err)
	}
	return SummarizeRoleOptions(roles), nil
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
