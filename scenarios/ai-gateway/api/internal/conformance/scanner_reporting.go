package conformance

import (
	"sort"
	"strings"

	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"
)

func maturityLevel(findings []*conformancev1.ConformanceFinding) string {
	if len(findings) == 0 {
		return "gateway-ready"
	}
	for _, f := range findings {
		if f.GetSeverity() == "high" {
			return "blocked-needs-investigation"
		}
	}
	return "resource-boundary-review"
}

func severityToken(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "high":
		return "SEVERITY_ERROR"
	case "medium":
		return "SEVERITY_WARNING"
	case "low", "advisory":
		return "SEVERITY_INFO"
	default:
		return "SEVERITY_UNSPECIFIED"
	}
}

func titleForRule(ruleID string) string {
	if title, ok := ruleTitles[ruleID]; ok {
		return title
	}
	return "AI conformance finding"
}

var ruleTitles = map[string]string{
	"ai.direct_ollama_http":                 "Direct Ollama HTTP usage",
	"ai.direct_openrouter_http":             "Direct OpenRouter HTTP usage",
	"ai.non_gateway_provider_client_import": "Provider client imported outside AI Gateway",
	"ai.invalid_provider_secret_env":        "Provider secret owned by scenario",
	"ai.invalid_provider_url_env":           "Provider URL owned by scenario",
	"ai.concrete_model_slug":                "Concrete model slug in scenario code",
	"ai.gateway_role_unreachable":           "Gateway role missing from task catalog",
	"ai.hardcoded_embedding_dimensions":     "Hard-coded embedding dimensions",
	"ai.hardcoded_context_window":           "Hard-coded context window",
	"ai.embedding_metadata_missing":         "Embedding metadata missing",
	"ai.resource_gateway_missing_role":      "Resource role policy missing",
	"ai.gateway_not_adopted":                "AI Gateway adoption signal missing",
	"ai.unreviewed_exception":               "Unreviewed AI exception",
}

func recommendations(findings []*conformancev1.ConformanceFinding) []string {
	if len(findings) == 0 {
		return []string{"no blocking AI conformance findings detected"}
	}
	seen := map[string]struct{}{}
	var out []string
	for _, f := range findings {
		if _, ok := seen[f.GetRuleId()]; ok {
			continue
		}
		seen[f.GetRuleId()] = struct{}{}
		out = append(out, f.GetRemediation())
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
