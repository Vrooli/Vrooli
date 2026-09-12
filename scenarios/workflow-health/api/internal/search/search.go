package search

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"workflow-health/internal/workflows"
)

const defaultLimit = 10

func SearchCatalog(catalog *workflows.ScenarioWorkflowCatalog, opts Options) []Result {
	if catalog == nil {
		return nil
	}
	allowed := typeFilter(opts.Types)
	queryTokens := tokenize(opts.Query)
	intent := classifyIntent(queryTokens)

	var results []Result
	for _, asset := range catalog.Assets {
		leafType, ok := leafTypeFor(asset)
		if !ok {
			continue
		}
		if !allowed[leafType] {
			if leafType != LeafTypeFragment || !opts.IncludeFragments || len(allowed) != 0 {
				continue
			}
		}
		if leafType == LeafTypeFragment && !opts.IncludeFragments && len(opts.Types) == 0 {
			continue
		}
		result := buildResult(asset, leafType)
		result.Score = scoreResult(result, queryTokens, intent)
		if len(queryTokens) > 0 && result.Score <= 0 {
			continue
		}
		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].LeafType != results[j].LeafType {
			return results[i].LeafType < results[j].LeafType
		}
		return results[i].Asset.Path < results[j].Asset.Path
	})

	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func leafTypeFor(asset workflows.WorkflowAsset) (string, bool) {
	switch asset.Role {
	case workflows.AssetRoleAgentFlow:
		return LeafTypeFlow, true
	case workflows.AssetRoleValidationCase:
		return LeafTypeTest, true
	case workflows.AssetRoleFragment:
		return LeafTypeFragment, true
	default:
		return "", false
	}
}

func typeFilter(types []string) map[string]bool {
	allowed := map[string]bool{}
	for _, typ := range types {
		typ = strings.TrimSpace(typ)
		if typ != "" {
			allowed[typ] = true
		}
	}
	if len(allowed) == 0 {
		allowed[LeafTypeFlow] = true
		allowed[LeafTypeTest] = true
	}
	return allowed
}

func buildResult(asset workflows.WorkflowAsset, leafType string) Result {
	title := strings.TrimSpace(asset.Name)
	if title == "" {
		title = strings.TrimSuffix(strings.TrimPrefix(asset.Path, "bas/"), ".json")
	}
	result := Result{
		ID:                   asset.ID,
		LeafType:             leafType,
		Asset:                asset,
		Title:                title,
		Snippet:              snippet(asset),
		Runnable:             leafType == LeafTypeFlow && !asset.Safety.Mutating,
		Mutating:             asset.Safety.Mutating,
		RequiresConfirmation: asset.Safety.RequiresConfirmation,
		RequiresIsolation:    asset.Safety.RequiresIsolation,
		SafetySummary:        safetySummary(asset),
		RequirementIDs:       requirementIDs(asset.Requirements),
		SelectorRefs:         selectorRefs(asset.Selectors),
		RouteRefs:            routeRefs(asset.Routes),
		LabelPairs:           labelPairs(asset.Labels),
		DependencyPaths:      dependencyPaths(asset.Dependencies),
	}
	if asset.Safety.Mutating {
		result.Guardrails = append(result.Guardrails, "requires explicit mutating confirmation")
		if asset.Safety.RequiresIsolation {
			result.Guardrails = append(result.Guardrails, "requires routed isolation proof")
		} else {
			result.Guardrails = append(result.Guardrails, "missing routed isolation metadata; do not auto-run")
		}
	}
	if leafType == LeafTypeTest {
		result.Guardrails = append(result.Guardrails, "validation evidence; not a default agent action")
	}
	if leafType == LeafTypeFragment {
		result.Guardrails = append(result.Guardrails, "dependency fragment; compose through a flow or case")
	}
	return result
}

func snippet(asset workflows.WorkflowAsset) string {
	parts := []string{}
	if asset.Description != "" {
		parts = append(parts, asset.Description)
	}
	if intent := strings.TrimSpace(asset.Labels["intent"]); intent != "" {
		parts = append(parts, "intent: "+intent)
	}
	if len(asset.Requirements) > 0 {
		parts = append(parts, "requirements: "+strings.Join(requirementIDs(asset.Requirements), ", "))
	}
	if asset.Safety.Mutating {
		parts = append(parts, "mutating")
	} else if asset.ExecutionMode != "" {
		parts = append(parts, asset.ExecutionMode)
	}
	return strings.Join(parts, " | ")
}

func safetySummary(asset workflows.WorkflowAsset) string {
	if asset.Safety.Mutating {
		if asset.Safety.RequiresConfirmation && asset.Safety.RequiresIsolation {
			return "mutating; confirmation and routed isolation required"
		}
		return "mutating; missing complete safety metadata"
	}
	return "observer; safe to inspect"
}

type queryIntent string

const (
	intentGeneral  queryIntent = "general"
	intentRun      queryIntent = "run"
	intentValidate queryIntent = "validate"
)

func classifyIntent(tokens []string) queryIntent {
	for _, token := range tokens {
		switch token {
		case "run", "do", "execute", "create", "open", "click", "perform", "workflow":
			return intentRun
		case "validate", "prove", "test", "check", "verify", "evidence", "requirement":
			return intentValidate
		}
	}
	return intentGeneral
}

func scoreResult(result Result, tokens []string, intent queryIntent) float64 {
	score := 0.1
	switch intent {
	case intentRun:
		if result.LeafType == LeafTypeFlow {
			score += 2.5
		}
		if result.Runnable {
			score += 0.3
		}
		if result.Mutating {
			score -= 0.2
		}
	case intentValidate:
		if result.LeafType == LeafTypeTest {
			score += 2.5
		}
		if len(result.RequirementIDs) > 0 {
			score += 0.8
		}
	default:
		if result.LeafType == LeafTypeFlow {
			score += 0.6
		}
	}
	haystack := strings.Join([]string{
		result.Title,
		result.Snippet,
		result.Asset.Path,
		strings.Join(result.LabelPairs, " "),
		strings.Join(result.RequirementIDs, " "),
		strings.Join(result.SelectorRefs, " "),
		strings.Join(result.RouteRefs, " "),
	}, " ")
	haystack = strings.ToLower(haystack)
	for _, token := range tokens {
		if strings.Contains(haystack, token) {
			score += 1
		}
		if strings.Contains(strings.ToLower(result.Title), token) {
			score += 0.5
		}
	}
	return score
}

func tokenize(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	out := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if len(field) < 2 || seen[field] {
			continue
		}
		seen[field] = true
		out = append(out, field)
	}
	sort.Strings(out)
	return out
}

func requirementIDs(links []workflows.RequirementLink) []string {
	out := make([]string, 0, len(links))
	for _, link := range links {
		if link.ID != "" {
			out = append(out, link.ID)
		}
	}
	sort.Strings(out)
	return out
}

func selectorRefs(refs []workflows.SelectorRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.Key != "" {
			out = append(out, ref.Key)
		} else if ref.Raw != "" {
			out = append(out, ref.Raw)
		}
	}
	sort.Strings(out)
	return out
}

func routeRefs(refs []workflows.RouteRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.Path != "" {
			out = append(out, strings.TrimSpace(ref.Scenario+" "+ref.Path))
		}
	}
	sort.Strings(out)
	return out
}

func labelPairs(labels map[string]string) []string {
	out := make([]string, 0, len(labels))
	for key, value := range labels {
		out = append(out, fmt.Sprintf("%s=%s", key, value))
	}
	sort.Strings(out)
	return out
}

func dependencyPaths(edges []workflows.DependencyEdge) []string {
	out := make([]string, 0, len(edges))
	for _, edge := range edges {
		if edge.ToPath != "" {
			out = append(out, edge.ToPath)
		}
	}
	sort.Strings(out)
	return out
}
