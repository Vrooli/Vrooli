package workflows

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
)

func labelsMap(doc map[string]any) map[string]string {
	labels := make(map[string]string)
	raw, ok := nested(doc, "metadata", "labels").(map[string]any)
	if !ok {
		return labels
	}
	for key, value := range raw {
		switch typed := value.(type) {
		case string:
			labels[key] = strings.TrimSpace(typed)
		case bool:
			if typed {
				labels[key] = "true"
			} else {
				labels[key] = "false"
			}
		default:
			if data, err := json.Marshal(typed); err == nil {
				labels[key] = string(data)
			}
		}
	}
	return labels
}

func extractSelectorRefs(doc map[string]any) []SelectorRef {
	var refs []SelectorRef
	nodes, _ := doc["nodes"].([]any)
	for _, nodeAny := range nodes {
		node, ok := nodeAny.(map[string]any)
		if !ok {
			continue
		}
		nodeID := getStringFromMap(node, "id")
		walkStrings(node, "", func(path, value string) {
			value = strings.TrimSpace(value)
			if value == "" {
				return
			}
			if strings.Contains(value, "@selector/") {
				for _, token := range selectorTokens(value) {
					refs = append(refs, SelectorRef{NodeID: nodeID, Key: token, Raw: value, Path: path})
				}
				return
			}
			if strings.HasSuffix(path, ".selector") || strings.HasSuffix(path, ".success_selector") {
				refs = append(refs, SelectorRef{NodeID: nodeID, Raw: value, Path: path})
			}
		})
	}
	return dedupeSelectorRefs(refs)
}

func extractRouteRefs(doc map[string]any) []RouteRef {
	var refs []RouteRef
	nodes, _ := doc["nodes"].([]any)
	for _, nodeAny := range nodes {
		node, ok := nodeAny.(map[string]any)
		if !ok {
			continue
		}
		nodeID := getStringFromMap(node, "id")
		scenario := firstNonEmpty(getString(node, "action", "navigate", "scenario"), getString(node, "data", "scenario"))
		for _, candidate := range []struct {
			path   string
			source string
		}{
			{path: getString(node, "action", "navigate", "scenario_path"), source: "action.navigate.scenario_path"},
			{path: getString(node, "action", "navigate", "scenarioPath"), source: "action.navigate.scenarioPath"},
			{path: getString(node, "data", "scenario_path"), source: "data.scenario_path"},
			{path: getString(node, "data", "scenarioPath"), source: "data.scenarioPath"},
			{path: getString(node, "data", "path"), source: "data.path"},
		} {
			routePath := strings.TrimSpace(candidate.path)
			if routePath == "" {
				continue
			}
			refs = append(refs, RouteRef{
				NodeID:   nodeID,
				Scenario: scenario,
				Path:     routePath,
				Source:   candidate.source,
			})
		}
	}
	return dedupeRouteRefs(refs)
}

func selectorTokens(value string) []string {
	var tokens []string
	for {
		idx := strings.Index(value, "@selector/")
		if idx < 0 {
			break
		}
		value = value[idx+len("@selector/"):]
		end := len(value)
		for i, r := range value {
			if !(r == '.' || r == '-' || r == '_' || r == '/' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
				end = i
				break
			}
		}
		token := strings.TrimSpace(value[:end])
		if token != "" {
			tokens = append(tokens, token)
		}
		value = value[end:]
	}
	return normalizeStrings(tokens)
}

func requirementLinksFromIDs(ids []string, source string) []RequirementLink {
	ids = normalizeStrings(ids)
	links := make([]RequirementLink, 0, len(ids))
	for _, id := range ids {
		links = append(links, RequirementLink{ID: id, Source: source})
	}
	return links
}

func normalizeWorkflowTargetPath(targetPath string) string {
	targetPath = filepath.ToSlash(strings.TrimSpace(targetPath))
	targetPath = strings.TrimPrefix(targetPath, "./")
	targetPath = strings.TrimPrefix(targetPath, "bas/")
	return targetPath
}

func safetyProfile(mode, reset string, labels map[string]string) SafetyProfile {
	mode = normalizeMode(mode)
	reset = normalizeReset(reset)
	mutating := mode == "mutating" || mode == "destructive" || reset == "full"
	requiresConfirmation := labelBool(labels["requires_confirmation"], "true", "yes")
	requiresIsolation := labelBool(labels["routed_isolation"], "true", "yes", "routed")
	return SafetyProfile{
		ExecutionMode:        mode,
		Reset:                reset,
		Mutating:             mutating,
		RequiresIsolation:    requiresIsolation,
		RequiresConfirmation: requiresConfirmation,
	}
}

func labelBool(value string, truthy ...string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range truthy {
		if value == candidate {
			return true
		}
	}
	return false
}

func normalizeMode(value string) string {
	mode := strings.ToLower(strings.TrimSpace(value))
	// BAS serializes ExecutionMode as its protobuf JSON enum name while authored
	// playbooks may use the concise form. Both are one contract and must retain
	// their safety classification during catalog validation.
	return strings.TrimPrefix(mode, "execution_mode_")
}

func normalizeReset(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "database":
		return "full"
	case "none", "full":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func nodeCount(doc map[string]any) int {
	nodes, ok := doc["nodes"].([]any)
	if !ok {
		return 0
	}
	return len(nodes)
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return strings.TrimSpace(string(raw))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var normalized []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}

func nested(m map[string]any, path ...string) any {
	var current any = m
	for _, key := range path {
		next, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = next[key]
	}
	return current
}

func getString(m map[string]any, path ...string) string {
	value, ok := nested(m, path...).(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func getStringFromMap(m map[string]any, key string) string {
	value, ok := m[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func walkStrings(value any, path string, visit func(path, value string)) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			nextPath := key
			if path != "" {
				nextPath = path + "." + key
			}
			walkStrings(typed[key], nextPath, visit)
		}
	case []any:
		for i, item := range typed {
			walkStrings(item, path, visit)
			_ = i
		}
	case string:
		visit(path, typed)
	}
}

func dedupeSelectorRefs(refs []SelectorRef) []SelectorRef {
	seen := make(map[string]struct{}, len(refs))
	var out []SelectorRef
	for _, ref := range refs {
		key := ref.NodeID + "\x00" + ref.Key + "\x00" + ref.Raw + "\x00" + ref.Path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}
	sortSelectorRefs(out)
	return out
}

func dedupeRouteRefs(refs []RouteRef) []RouteRef {
	seen := make(map[string]struct{}, len(refs))
	var out []RouteRef
	for _, ref := range refs {
		key := ref.NodeID + "\x00" + ref.Scenario + "\x00" + ref.Path + "\x00" + ref.Source
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}
	sortRouteRefs(out)
	return out
}

func dedupeEdges(edges []DependencyEdge) []DependencyEdge {
	seen := make(map[string]struct{}, len(edges))
	var out []DependencyEdge
	for _, edge := range edges {
		key := edge.FromPath + "\x00" + edge.ToPath + "\x00" + edge.Kind + "\x00" + edge.Source
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, edge)
	}
	sortDependencyEdges(out)
	return out
}

func sortAssets(values []WorkflowAsset) {
	sort.Slice(values, func(i, j int) bool {
		return values[i].Path < values[j].Path
	})
}

func sortWorkflowCases(values []WorkflowCase) {
	sort.Slice(values, func(i, j int) bool {
		return values[i].Path < values[j].Path
	})
}

func sortWorkflowFlows(values []WorkflowFlow) {
	sort.Slice(values, func(i, j int) bool {
		return values[i].Path < values[j].Path
	})
}

func sortWorkflowActions(values []WorkflowAction) {
	sort.Slice(values, func(i, j int) bool {
		return values[i].Path < values[j].Path
	})
}

func sortSeeds(values []SeedContract) {
	sort.Slice(values, func(i, j int) bool {
		return values[i].Path < values[j].Path
	})
}

func sortDependencyEdges(values []DependencyEdge) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].FromPath != values[j].FromPath {
			return values[i].FromPath < values[j].FromPath
		}
		if values[i].ToPath != values[j].ToPath {
			return values[i].ToPath < values[j].ToPath
		}
		return values[i].Source < values[j].Source
	})
}

func sortRequirementLinks(values []RequirementLink) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].ID != values[j].ID {
			return values[i].ID < values[j].ID
		}
		return values[i].Source < values[j].Source
	})
}

func sortSelectorRefs(values []SelectorRef) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].NodeID != values[j].NodeID {
			return values[i].NodeID < values[j].NodeID
		}
		if values[i].Path != values[j].Path {
			return values[i].Path < values[j].Path
		}
		return values[i].Raw < values[j].Raw
	})
}

func sortRouteRefs(values []RouteRef) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].NodeID != values[j].NodeID {
			return values[i].NodeID < values[j].NodeID
		}
		if values[i].Path != values[j].Path {
			return values[i].Path < values[j].Path
		}
		return values[i].Source < values[j].Source
	})
}
