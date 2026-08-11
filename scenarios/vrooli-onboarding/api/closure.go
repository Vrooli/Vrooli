package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type dependencySpec struct {
	Enabled       *bool  `json:"enabled"`
	Required      bool   `json:"required"`
	StartupPolicy string `json:"startup_policy"`
}

type closureManifest struct {
	Service struct {
		Name string `json:"name"`
	} `json:"service"`
	Dependencies struct {
		Scenarios map[string]dependencySpec `json:"scenarios"`
		Resources map[string]dependencySpec `json:"resources"`
	} `json:"dependencies"`
	OptionalDependencies struct {
		Scenarios map[string]dependencySpec `json:"scenarios"`
		Resources map[string]dependencySpec `json:"resources"`
	} `json:"optional_dependencies"`
}

type closureProvenance struct {
	Kind string `json:"kind"`
	From string `json:"from,omitempty"`
}

type closureMember struct {
	Name       string              `json:"name"`
	Provenance []closureProvenance `json:"provenance"`
	Required   bool                `json:"required"`
	Direct     bool                `json:"direct"`
}

type closureResult struct {
	Scenarios []closureMember `json:"scenarios"`
	Resources []closureMember `json:"resources"`
}

func readClosureManifest(root, name string) (closureManifest, error) {
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return closureManifest{}, fmt.Errorf("scenario dependency %q has no manifest", name)
		}
		return closureManifest{}, err
	}
	var manifest closureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return closureManifest{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return manifest, nil
}

func dependencyEnabled(spec dependencySpec) bool {
	return spec.Enabled == nil || *spec.Enabled
}

func dependencyKind(spec dependencySpec) string {
	if spec.Required {
		return "required"
	}
	return "try_start"
}

func appendClosureMember(members map[string]*closureMember, name string, provenance closureProvenance, required, direct bool) {
	member, ok := members[name]
	if !ok {
		member = &closureMember{Name: name}
		members[name] = member
	}
	member.Required = member.Required || required
	member.Direct = member.Direct || direct
	for _, existing := range member.Provenance {
		if existing == provenance {
			return
		}
	}
	member.Provenance = append(member.Provenance, provenance)
}

func sortedDependencyNames(required, optional map[string]dependencySpec) []string {
	seen := map[string]struct{}{}
	names := make([]string, 0, len(required)+len(optional))
	for name := range required {
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for name := range optional {
		if _, ok := seen[name]; !ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func resolveClosure(root string, models []ScenarioReadModel) (closureResult, error) {
	scenarios := map[string]*closureMember{}
	resources := map[string]*closureMember{}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	stack := make([]string, 0)

	var visit func(string, closureProvenance, bool) error
	visit = func(name string, provenance closureProvenance, direct bool) error {
		if visiting[name] {
			cycleStart := 0
			for i, item := range stack {
				if item == name {
					cycleStart = i
					break
				}
			}
			cycle := append(append([]string(nil), stack[cycleStart:]...), name)
			return fmt.Errorf("scenario dependency cycle detected: %s", strings.Join(cycle, " -> "))
		}
		if visited[name] {
			appendClosureMember(scenarios, name, provenance, provenance.Kind == "required", direct)
			return nil
		}
		manifest, err := readClosureManifest(root, name)
		if err != nil {
			return err
		}
		visiting[name] = true
		stack = append(stack, name)
		appendClosureMember(scenarios, name, provenance, provenance.Kind == "required", direct)

		for _, depName := range sortedDependencyNames(manifest.Dependencies.Resources, manifest.OptionalDependencies.Resources) {
			spec, ok := manifest.Dependencies.Resources[depName]
			if !ok {
				spec = manifest.OptionalDependencies.Resources[depName]
			}
			if !dependencyEnabled(spec) {
				continue
			}
			appendClosureMember(resources, depName, closureProvenance{Kind: dependencyKind(spec), From: name}, spec.Required, false)
		}
		for _, depName := range sortedDependencyNames(manifest.Dependencies.Scenarios, manifest.OptionalDependencies.Scenarios) {
			spec, ok := manifest.Dependencies.Scenarios[depName]
			if !ok {
				spec = manifest.OptionalDependencies.Scenarios[depName]
			}
			if !dependencyEnabled(spec) {
				continue
			}
			if err := visit(depName, closureProvenance{Kind: dependencyKind(spec), From: name}, false); err != nil {
				return err
			}
		}
		delete(visiting, name)
		stack = stack[:len(stack)-1]
		visited[name] = true
		return nil
	}

	for _, model := range models {
		if !model.Enabled {
			continue
		}
		if err := visit(model.Name, closureProvenance{Kind: "selected"}, true); err != nil {
			return closureResult{}, err
		}
	}

	result := closureResult{
		Scenarios: mapClosureMembers(scenarios),
		Resources: mapClosureMembers(resources),
	}
	return result, nil
}

func mapClosureMembers(members map[string]*closureMember) []closureMember {
	result := make([]closureMember, 0, len(members))
	for _, member := range members {
		sort.Slice(member.Provenance, func(i, j int) bool {
			if member.Provenance[i].Kind == member.Provenance[j].Kind {
				return member.Provenance[i].From < member.Provenance[j].From
			}
			return member.Provenance[i].Kind < member.Provenance[j].Kind
		})
		result = append(result, *member)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (s *Server) handleV2Closure(w http.ResponseWriter, _ *http.Request) {
	root, err := manifestRoot()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	models, err := loadScenarioReadModels()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	closure, err := resolveClosure(root, models)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, closure)
}

func (s *Server) handleV2Union(w http.ResponseWriter, _ *http.Request) {
	root, err := manifestRoot()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	models, err := loadScenarioReadModels()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	closure, err := resolveClosure(root, models)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	selected := make([]ScenarioReadModel, 0, len(closure.Scenarios))
	byName := map[string]ScenarioReadModel{}
	for _, model := range models {
		byName[model.Name] = model
	}
	for _, member := range closure.Scenarios {
		if model, ok := byName[member.Name]; ok {
			selected = append(selected, model)
		}
	}
	state, err := loadOperatorState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	requirements, err := deriveV2HostRequirements(root, state, selected)
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	tools := make([]string, 0, len(requirements.Tools))
	for _, item := range requirements.Tools {
		tools = append(tools, item.Name)
	}
	safeguards := make([]string, 0, len(requirements.Safeguards))
	for _, item := range requirements.Safeguards {
		safeguards = append(safeguards, item.Name)
	}
	sort.Strings(tools)
	sort.Strings(safeguards)
	writeJSON(w, http.StatusOK, map[string]any{
		"scenarios":     closure.Scenarios,
		"resources":     closure.Resources,
		"host_tools":    tools,
		"safeguards":    safeguards,
		"catalog_paths": []string{"catalog/scenarios", "catalog/resources", "catalog/internal/tools", "catalog/internal/safeguards"},
	})
}
