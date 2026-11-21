// Package impact inventories rights decisions across the maintained local
// model registries. Registry data remains the source of truth; this package
// derives selection paths from role and operation references.
package impact

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/ai-provider-go/policy"
)

type Registry struct {
	Provider string
	Path     string
	Label    string
	Document map[string]any
}

type PathDecision struct {
	Path           string                `json:"path"`
	Use            policy.RightsUse      `json:"use"`
	PreviousStatus policy.RightsStatus   `json:"previous_status,omitempty"`
	Changed        bool                  `json:"changed,omitempty"`
	Decision       policy.RightsDecision `json:"decision"`
}

type Model struct {
	Provider string         `json:"provider"`
	ID       string         `json:"model"`
	Pointer  string         `json:"registry_pointer"`
	Paths    []PathDecision `json:"paths"`
}

type Report struct {
	Context      policy.RightsContext  `json:"context"`
	ComparedFrom *policy.RightsContext `json:"compared_from,omitempty"`
	Models       []Model               `json:"models"`
}

// Compare evaluates the same registries under target and attaches the status
// from base to each path so callers can show only context-sensitive changes.
func Compare(registries []Registry, base, target policy.RightsContext, at time.Time) (Report, error) {
	before, err := Load(registries, base, at)
	if err != nil {
		return Report{}, err
	}
	after, err := Load(registries, target, at)
	if err != nil {
		return Report{}, err
	}
	prior := map[string]policy.RightsStatus{}
	for _, model := range before.Models {
		for _, path := range model.Paths {
			prior[model.Provider+"/"+model.ID+"|"+string(path.Use)+"|"+path.Path] = path.Decision.Status
		}
	}
	for mi := range after.Models {
		for pi := range after.Models[mi].Paths {
			path := &after.Models[mi].Paths[pi]
			key := after.Models[mi].Provider + "/" + after.Models[mi].ID + "|" + string(path.Use) + "|" + path.Path
			path.PreviousStatus = prior[key]
			path.Changed = path.PreviousStatus != path.Decision.Status
		}
	}
	after.ComparedFrom = &base
	return after, nil
}

// DefaultRegistries returns the maintained rights-bearing model registries.
func DefaultRegistries(repoRoot string) []Registry {
	return []Registry{
		{Provider: "ollama", Path: filepath.Join(repoRoot, "resources/ollama/model-policy.json"), Label: "resources/ollama/model-policy.json"},
		{Provider: "ollaya", Path: filepath.Join(repoRoot, "resources/ollaya/model-policy.json"), Label: "resources/ollaya/model-policy.json"},
		{Provider: "image", Path: filepath.Join(repoRoot, "scenarios/image-tools/api/internal/models/registry.seed.json"), Label: "scenarios/image-tools/api/internal/models/registry.seed.json"},
		{Provider: "music", Path: filepath.Join(repoRoot, "scenarios/music-tools/api/internal/models/registry.seed.json"), Label: "scenarios/music-tools/api/internal/models/registry.seed.json"},
	}
}

func Load(registries []Registry, context policy.RightsContext, at time.Time) (Report, error) {
	report := Report{Context: context, Models: []Model{}}
	for _, registry := range registries {
		if registry.Document == nil {
			data, err := os.ReadFile(registry.Path)
			if err != nil {
				return Report{}, fmt.Errorf("read %s registry: %w", registry.Provider, err)
			}
			if err := json.Unmarshal(data, &registry.Document); err != nil {
				root, rootErr := DecodeRootArray(data)
				if rootErr != nil {
					return Report{}, fmt.Errorf("decode %s registry: %w", registry.Provider, err)
				}
				registry.Document = root
			}
		}
		for _, item := range registryModels(registry.Document) {
			facts := item.rights
			decisions := derivePaths(registry, item, context, at)
			model := Model{Provider: registry.Provider, ID: item.id, Pointer: item.pointer, Paths: decisions}
			// Always make the selected inference decision visible, even if a
			// registry has not yet established any role/caller reference.
			if !hasUse(decisions, policy.RightsUseLocalInference) {
				model.Paths = append(model.Paths, PathDecision{Path: registryLabel(registry) + "#" + item.pointer, Use: policy.RightsUseLocalInference, Decision: policy.EvaluateRights(facts, policy.RightsUseLocalInference, context, at)})
			}
			report.Models = append(report.Models, model)
		}
	}
	sort.Slice(report.Models, func(i, j int) bool {
		if report.Models[i].Provider == report.Models[j].Provider {
			return report.Models[i].ID < report.Models[j].ID
		}
		return report.Models[i].Provider < report.Models[j].Provider
	})
	return report, nil
}

type registryItem struct {
	id, pointer string
	data        map[string]any
	rights      *policy.ArtifactRights
}

func registryModels(doc map[string]any) []registryItem {
	var out []registryItem
	base := "/models/"
	if doc["__root_array"] == true {
		base = "/"
	}
	switch value := doc["models"].(type) {
	case map[string]any:
		for id, raw := range value {
			pointer := base + escape(id)
			if item, ok := makeItem(id, raw, pointer); ok {
				out = append(out, item)
			}
		}
	case []any:
		for i, raw := range value {
			if m, ok := raw.(map[string]any); ok {
				id := stringField(m, "id")
				if id == "" {
					id = fmt.Sprintf("%d", i)
				}
				pointer := fmt.Sprintf("/models/%d", i)
				if base == "/" {
					pointer = fmt.Sprintf("/%d", i)
				}
				if item, ok := makeItem(id, raw, pointer); ok {
					out = append(out, item)
				}
			}
		}
	}
	return out
}

func makeItem(id string, raw any, pointer string) (registryItem, bool) {
	m, ok := raw.(map[string]any)
	if !ok {
		return registryItem{}, false
	}
	item := registryItem{id: id, pointer: pointer, data: m}
	rightsRaw, ok := m["rights"]
	if !ok {
		return item, true
	}
	b, err := json.Marshal(rightsRaw)
	if err != nil {
		return item, true
	}
	var rights policy.ArtifactRights
	if json.Unmarshal(b, &rights) == nil {
		item.rights = &rights
	}
	return item, true
}

func derivePaths(reg Registry, item registryItem, context policy.RightsContext, at time.Time) []PathDecision {
	var out []PathDecision
	label := registryLabel(reg)
	add := func(path string, use policy.RightsUse) {
		out = append(out, PathDecision{Path: path, Use: use, Decision: policy.EvaluateRights(item.rights, use, context, at)})
	}
	if roles, ok := reg.Document["roles"].(map[string]any); ok {
		for role, raw := range roles {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if refMatches(m["model"], item.id) {
				add(label+"#/roles/"+escape(role), policy.RightsUseLocalInference)
			}
			if refs, ok := m["fallbacks"].([]any); ok {
				for _, ref := range refs {
					if refMatches(ref, item.id) {
						add(label+"#/roles/"+escape(role)+"/fallbacks", policy.RightsUseLocalInference)
					}
				}
			}
			if refMatches(m["upstream"], item.id) {
				add(label+"#/roles/"+escape(role), policy.RightsUseLocalInference)
			}
		}
	}
	for _, field := range []string{"operations", "default_for"} {
		if values, ok := item.data[field].([]any); ok {
			for _, op := range values {
				if name, ok := op.(string); ok {
					add(label+"#"+item.pointer+"/"+field+"/"+escape(name), policy.RightsUseLocalInference)
				}
			}
		}
	}
	return dedupe(out)
}

func registryLabel(reg Registry) string {
	if reg.Label != "" {
		return reg.Label
	}
	return reg.Path
}

func refMatches(value any, id string) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}
	return normalizeRef(s) == normalizeRef(id)
}
func normalizeRef(s string) string                    { return strings.TrimPrefix(strings.TrimSpace(s), "ollama/") }
func stringField(m map[string]any, key string) string { s, _ := m[key].(string); return s }
func escape(s string) string                          { return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1") }
func hasUse(paths []PathDecision, use policy.RightsUse) bool {
	for _, p := range paths {
		if p.Use == use {
			return true
		}
	}
	return false
}
func dedupe(paths []PathDecision) []PathDecision {
	type key struct {
		p string
		u policy.RightsUse
	}
	seen := map[key]bool{}
	out := make([]PathDecision, 0, len(paths))
	for _, p := range paths {
		k := key{p.Path, p.Use}
		if !seen[k] {
			seen[k] = true
			out = append(out, p)
		}
	}
	return out
}

// DecodeRootArray reads registries whose JSON root is an array and adapts it
// to the same inventory representation used by object-root registries.
func DecodeRootArray(data []byte) (map[string]any, error) {
	var rows []any
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	return map[string]any{"models": rows, "__root_array": true}, nil
}
