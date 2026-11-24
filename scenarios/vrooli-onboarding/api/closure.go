package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/api-core/coreset"
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
	HostTools      []hostRequirement `json:"hostTools"`
	HostSafeguards []hostRequirement `json:"hostSafeguards"`
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
	State      string              `json:"state"`
	Reason     string              `json:"reason,omitempty"`
	Policy     string              `json:"policy,omitempty"`
}

type closureResult struct {
	ClosureDigest string          `json:"closureDigest"`
	Scenarios     []closureMember `json:"scenarios"`
	Resources     []closureMember `json:"resources"`
	Tools         []closureMember `json:"host_tools"`
	Safeguards    []closureMember `json:"safeguards"`
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

func markClosureMember(members map[string]*closureMember, name, state, reason, policy string) {
	member, ok := members[name]
	if !ok {
		member = &closureMember{Name: name}
		members[name] = member
	}
	if member.State == "" || state == "ignored" {
		member.State = state
	}
	if strings.TrimSpace(member.Reason) == "" {
		member.Reason = strings.TrimSpace(reason)
	}
	if strings.TrimSpace(member.Policy) == "" {
		member.Policy = strings.TrimSpace(policy)
	}
}

func dependencyPolicy(spec dependencySpec) string {
	policy := strings.ToLower(strings.TrimSpace(spec.StartupPolicy))
	if policy == "" {
		if spec.Required {
			return "must_start"
		}
		return "try_start"
	}
	return policy
}

func dependencyIgnored(spec dependencySpec) bool {
	return dependencyPolicy(spec) == "ignore"
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
	return resolveClosureForState(root, models, OperatorState{})
}

func resolveClosureForState(root string, models []ScenarioReadModel, state OperatorState) (closureResult, error) {
	scenarios := map[string]*closureMember{}
	resources := map[string]*closureMember{}
	tools := map[string]*closureMember{}
	safeguards := map[string]*closureMember{}
	roots := make([]coreset.ComponentID, 0, len(models)+len(state.Resources))
	rootNames := map[string]bool{}
	var projectTools, projectSafeguards []hostRequirement
	for _, model := range models {
		if !model.Enabled {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(model.Name))
		if name == "" {
			continue
		}
		id := coreset.ComponentID{Kind: "scenario", Name: name}
		roots = append(roots, id)
		rootNames[name] = true
		markClosureMember(scenarios, name, "selected", "selected by operator", "must_start")
	}
	// Each adapter call reads and adapts one declaration. Reachability, cycle
	// handling, provenance and canonical union are all owned by api-core.
	dependencySource := coreset.DependencySourceFunc(func(from coreset.ComponentID) ([]coreset.DependencyEdge, error) {
		if from.Kind == "scenario" {
			manifest, err := readClosureManifest(root, from.Name)
			if err != nil {
				return nil, err
			}
			graph := coreset.DependencyGraph{}
			appendHostRequirementEdges(&graph, from, from.Name, manifest.HostTools, manifest.HostSafeguards)
			out := graph.Edges
			for _, depName := range sortedDependencyNames(manifest.Dependencies.Resources, manifest.OptionalDependencies.Resources) {
				spec, required := manifest.Dependencies.Resources[depName]
				if !required {
					spec = manifest.OptionalDependencies.Resources[depName]
				}
				policy := dependencyPolicy(spec)
				if dependencyIgnored(spec) {
					markClosureMember(resources, depName, "ignored", "startup policy is ignore", policy)
					continue
				}
				if !dependencyEnabled(spec) {
					if spec.Required {
						return nil, fmt.Errorf("required resource %q from scenario %q is disabled", depName, from.Name)
					}
					markClosureMember(resources, depName, "ignored", "dependency is disabled by manifest", policy)
					continue
				}
				if !spec.Required {
					if choice, ok := state.Resources[depName]; ok && choice.Enabled != nil && !*choice.Enabled {
						markClosureMember(resources, depName, "ignored", "dependency declined by operator", policy)
						continue
					}
				}
				target := coreset.ComponentID{Kind: "resource", Name: depName}
				out = append(out, coreset.DependencyEdge{
					From: from, To: target, Relation: "resource", DeclaredBy: from.Name,
					Required: spec.Required, Policy: policy,
				})
				markClosureMember(resources, depName, "included", "declared by dependency", policy)
			}
			for _, depName := range sortedDependencyNames(manifest.Dependencies.Scenarios, manifest.OptionalDependencies.Scenarios) {
				spec, required := manifest.Dependencies.Scenarios[depName]
				if !required {
					spec = manifest.OptionalDependencies.Scenarios[depName]
				}
				policy := dependencyPolicy(spec)
				if dependencyIgnored(spec) {
					markClosureMember(scenarios, depName, "ignored", "startup policy is ignore", policy)
					continue
				}
				if !dependencyEnabled(spec) {
					if spec.Required {
						return nil, fmt.Errorf("required scenario %q from scenario %q is disabled", depName, from.Name)
					}
					markClosureMember(scenarios, depName, "ignored", "dependency is disabled by manifest", policy)
					continue
				}
				targetName := strings.ToLower(strings.TrimSpace(depName))
				target := coreset.ComponentID{Kind: "scenario", Name: targetName}
				out = append(out, coreset.DependencyEdge{
					From: from, To: target, Relation: "scenario", DeclaredBy: from.Name,
					Required: spec.Required, Policy: policy,
				})
				markClosureMember(scenarios, targetName, "included", "declared by dependency", policy)
			}
			return out, nil
		}
		if from.Kind == "resource" {
			manifestPath := filepath.Join(root, "resources", from.Name, "resource.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil, nil
				}
				return nil, err
			}
			var manifest struct {
				Dependencies []string `json:"dependencies"`
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return nil, fmt.Errorf("decode %s: %w", manifestPath, err)
			}
			var graph coreset.DependencyGraph
			resourceTools, resourceSafeguards, err := loadHostRequirements(manifestPath)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			appendHostRequirementEdges(&graph, from, "resource:"+from.Name, resourceTools, resourceSafeguards)
			out := graph.Edges
			dependencies := append([]string(nil), manifest.Dependencies...)
			sort.Strings(dependencies)
			for _, dependency := range dependencies {
				dependency = strings.ToLower(strings.TrimSpace(dependency))
				if dependency == "" {
					continue
				}
				out = append(out, coreset.DependencyEdge{From: from, To: coreset.ComponentID{Kind: "resource", Name: dependency}, Relation: "resource", DeclaredBy: "resource:" + from.Name, Detail: "declared in resource.json", Required: true, Policy: "must_start"})
			}
			return out, nil
		}
		if from.Kind == "project" {
			var graph coreset.DependencyGraph
			appendHostRequirementEdges(&graph, from, "project", projectTools, projectSafeguards)
			return graph.Edges, nil
		}
		return nil, nil
	})

	// Standalone resources are also roots, but only explicit operator enables
	// are included. They share the same union and digest as scenario roots.
	for name, choice := range state.Resources {
		if choice.Enabled == nil || !*choice.Enabled {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		if _, err := os.Stat(filepath.Join(root, "resources", name, "resource.json")); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return closureResult{}, err
		}
		id := coreset.ComponentID{Kind: "resource", Name: name}
		roots = append(roots, id)
		markClosureMember(resources, name, "selected", "selected as a standalone resource", "try_start")
	}

	// Host tools and safeguards declared by selected resources join the same
	// graph. A missing resource manifest remains an absent optional catalog
	// entry here, matching the existing host requirement projection contract.

	// Repository-scoped host requirements are rooted independently from the
	// selected scenarios. A staged bundle has no project scope by design.
	if projectScopeAvailable() {
		var err error
		projectTools, projectSafeguards, err = loadHostRequirements(filepath.Join(root, ".vrooli", "service.json"))
		if err != nil && !os.IsNotExist(err) {
			return closureResult{}, fmt.Errorf("read project host requirements: %w", err)
		}
		if err == nil && (len(projectTools) > 0 || len(projectSafeguards) > 0) {
			project := coreset.ComponentID{Kind: "project", Name: "repository"}
			roots = append(roots, project)
		}
	}

	// The onboarding preview is a set union, not a startup ordering. Optional
	// try_start edges can form a cycle when two services have reciprocal runtime
	// integrations; keep those members in the closure while still rejecting a
	// cycle made entirely of required edges.
	closure, err := coreset.ResolveFromSource(dependencySource, roots, coreset.ResolveOptions{AllowCyclesWithOptionalEdges: true})
	if err != nil {
		return closureResult{}, err
	}
	for _, item := range closure.Members {
		name := item.Component.Name
		var target map[string]*closureMember
		if item.Component.Kind == "scenario" {
			target = scenarios
		} else if item.Component.Kind == "resource" {
			target = resources
		} else if item.Component.Kind == "tool" {
			target = tools
		} else if item.Component.Kind == "safeguard" {
			target = safeguards
		} else {
			continue
		}
		if target[name] == nil {
			stateName, reason, policy := "included", "declared by dependency", "try_start"
			if item.Selected {
				stateName, reason, policy = "selected", "selected by operator", "must_start"
			}
			markClosureMember(target, name, stateName, reason, policy)
		}
		if item.Selected && (item.Component.Kind == "scenario" || item.Component.Kind == "resource") {
			provenance := closureProvenance{Kind: "selected"}
			if item.Component.Kind == "resource" {
				provenance = closureProvenance{Kind: "operator_enabled", From: "operator-state"}
			}
			appendClosureMember(target, name, provenance, false, true)
		}
		for _, edge := range item.Reasons {
			kind := "try_start"
			if edge.Required {
				kind = "required"
			}
			fromName := edge.From.Name
			if edge.From.Kind == "resource" {
				fromName = "resource:" + fromName
			} else if edge.From.Kind == "project" {
				fromName = "project"
			}
			appendClosureMember(target, name, closureProvenance{Kind: kind, From: fromName}, edge.Required, item.Selected && rootNames[name])
			if edge.Required {
				target[name].Required = true
			}
			if target[name].Policy == "" || edge.Required {
				target[name].Policy = edge.Policy
			}
		}
	}

	result := closureResult{
		ClosureDigest: closure.Digest,
		Scenarios:     mapClosureMembers(scenarios),
		Resources:     mapClosureMembers(resources),
		Tools:         mapClosureMembers(tools),
		Safeguards:    mapClosureMembers(safeguards),
	}
	return result, nil
}

func appendHostRequirementEdges(graph *coreset.DependencyGraph, from coreset.ComponentID, declaredBy string, tools, safeguards []hostRequirement) {
	appendItems := func(kind string, items []hostRequirement) {
		for _, item := range items {
			name := strings.ToLower(strings.TrimSpace(item.Name))
			if name == "" {
				continue
			}
			to := coreset.ComponentID{Kind: kind, Name: name}
			policy := "optional"
			if item.Required {
				policy = "required"
			}
			graph.Components = append(graph.Components, to)
			graph.Edges = append(graph.Edges, coreset.DependencyEdge{
				From: from, To: to, Relation: kind, DeclaredBy: declaredBy,
				Detail: item.Reason, Required: item.Required, Policy: policy,
			})
		}
	}
	appendItems("tool", tools)
	appendItems("safeguard", safeguards)
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
	if err := validateSharedHostMembers(closure, requirements); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	tools := make([]string, 0, len(closure.Tools))
	for _, item := range closure.Tools {
		tools = append(tools, item.Name)
	}
	safeguards := make([]string, 0, len(closure.Safeguards))
	for _, item := range closure.Safeguards {
		safeguards = append(safeguards, item.Name)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"closure_digest": closure.ClosureDigest,
		"scenarios":      closure.Scenarios,
		"resources":      closure.Resources,
		"host_tools":     tools,
		"safeguards":     safeguards,
		"catalog_paths":  []string{"catalog/scenarios", "catalog/resources", "catalog/internal/tools", "catalog/internal/safeguards"},
	})
}
