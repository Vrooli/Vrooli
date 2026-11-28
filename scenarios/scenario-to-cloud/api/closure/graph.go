package closure

import (
	"errors"
	"fmt"
	"scenario-to-cloud/domain"
	"sort"
	"strings"

	apicoreset "github.com/vrooli/api-core/coreset"
)

func toSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			set[value] = true
		}
	}
	return set
}

func (r *resolver) loadScenario(id, declaredBy string) (*scenarioNode, error) {
	if node, ok := r.scenarios[id]; ok {
		return node, nil
	}
	decl, err := r.in.Catalog.Scenario(r.ctx, id)
	if err != nil {
		details := map[string]any{"component": id, "kind": "scenario"}
		if declaredBy != "" {
			details["declared_by"] = declaredBy
		}
		if errors.Is(err, ErrNotFound) {
			return nil, unavailable(fmt.Sprintf("scenario %q is not in the catalog", id), err, details)
		}
		return nil, unavailable(fmt.Sprintf("scenario %q declaration is unreadable", id), err, details)
	}
	node := &scenarioNode{decl: decl}
	r.scenarios[id] = node
	return node, nil
}

func (r *resolver) loadResource(id, declaredBy string) (*resourceNode, error) {
	if node, ok := r.resources[id]; ok {
		return node, nil
	}
	decl, err := r.in.Catalog.Resource(r.ctx, id)
	if err != nil {
		details := map[string]any{"component": id, "kind": "resource", "declared_by": declaredBy}
		if errors.Is(err, ErrNotFound) {
			return nil, unavailable(fmt.Sprintf("resource %q declared by %s is not in the catalog", id, declaredBy), err, details)
		}
		return nil, unavailable(fmt.Sprintf("resource %q declaration is unreadable", id), err, details)
	}
	node := &resourceNode{decl: decl}
	r.resources[id] = node
	return node, nil
}

// edgePasses decides whether a declared edge brings its target into the
// closure: required edges always do; optional edges do when the manifest
// enables them (and the operator has not deselected them) or when the
// operator selected them.
func (r *resolver) edgePasses(to string, edge DependencyEdge) (passes, selected bool) {
	if edge.Required {
		return true, false
	}
	if r.selectSet[to] {
		return true, true
	}
	if edge.Enabled && !r.deselectSet[to] {
		return true, true
	}
	return false, false
}

type declaredEdge struct {
	to     string
	edge   DependencyEdge
	detail string
}

func (r *resolver) scenarioEdges(decl *ScenarioDeclaration) (scenarios, resources []declaredEdge, err error) {
	for id, edge := range decl.Scenarios {
		scenarios = append(scenarios, declaredEdge{to: id, edge: edge, detail: "declared in service.json"})
	}
	for id, edge := range decl.Resources {
		resources = append(resources, declaredEdge{to: id, edge: edge, detail: "declared in service.json"})
	}
	if r.in.Analyzer != nil {
		result, err := r.in.Analyzer.Analyze(r.ctx, decl.ID)
		if err != nil {
			return nil, nil, unavailable(fmt.Sprintf("dependency analyzer is unavailable for scenario %q", decl.ID), err, map[string]any{"component": decl.ID, "source": "analyzer"})
		}
		r.analyzer.used = true
		if result.Tool != "" {
			r.analyzer.tool = result.Tool
		} else {
			r.analyzer.tool = analyzerTool
		}
		convert := func(items []AnalyzedDependency, into *[]declaredEdge) {
			for _, item := range items {
				if !validComponentID(item.Name) {
					continue
				}
				*into = append(*into, declaredEdge{
					to:     item.Name,
					edge:   DependencyEdge{Enabled: item.Enabled, Required: item.Required, StartupPolicy: policyFor(item.Required)},
					detail: "detected by " + r.analyzer.tool,
				})
			}
		}
		convert(result.Scenarios, &scenarios)
		convert(result.Resources, &resources)
	}
	sort.Slice(scenarios, func(i, j int) bool { return lessEdge(scenarios[i], scenarios[j]) })
	sort.Slice(resources, func(i, j int) bool { return lessEdge(resources[i], resources[j]) })
	return scenarios, resources, nil
}

func lessEdge(a, b declaredEdge) bool {
	if a.to != b.to {
		return a.to < b.to
	}
	return a.detail < b.detail
}

func policyFor(required bool) string {
	if required {
		return "must_start"
	}
	return "try_start"
}

// buildSharedClosure adapts included declarations into the shared dependency
// graph. Catalog loading and domain-specific optional-selection policy stay in
// scenario-to-cloud; the shared package owns graph closure, cycle semantics,
// deduplication, provenance paths, canonical ordering and digest.
func (r *resolver) buildSharedClosure(roots []*scenarioNode) error {
	rootIDs := make([]apicoreset.ComponentID, 0, len(roots))
	for _, root := range roots {
		id := apicoreset.ComponentID{Kind: string(domain.ClosureKindScenario), Name: root.decl.ID}
		rootIDs = append(rootIDs, id)
	}
	source := apicoreset.DependencySourceFunc(func(id apicoreset.ComponentID) ([]apicoreset.DependencyEdge, error) {
		if id.Kind == string(domain.ClosureKindScenario) {
			node := r.scenarios[id.Name]
			if node == nil {
				return nil, unavailable(fmt.Sprintf("scenario %q is not loaded into the dependency graph", id.Name), nil, map[string]any{"component": id.Name, "kind": id.Kind})
			}
			scenarioEdges, resourceEdges, err := r.scenarioEdges(node.decl)
			if err != nil {
				return nil, err
			}
			out := make([]apicoreset.DependencyEdge, 0, len(scenarioEdges)+len(resourceEdges))
			for _, declared := range scenarioEdges {
				if declared.to == id.Name && !declared.edge.Required {
					continue
				}
				passes, selected := r.edgePasses(declared.to, declared.edge)
				if !passes {
					continue
				}
				child, err := r.loadScenario(declared.to, id.Name)
				if err != nil {
					return nil, err
				}
				child.included = true
				child.includeUI = child.includeUI || declared.edge.IncludeUI
				child.optionalSelected = child.optionalSelected || selected
				child.startupPolicies = append(child.startupPolicies, declared.edge.StartupPolicy)
				if declared.edge.StartupPolicy != "ignore" {
					child.member = true
				}
				child.reasons = append(child.reasons, r.reasonsForEdge(id.Name, declared, selected)...)
				to := apicoreset.ComponentID{Kind: string(domain.ClosureKindScenario), Name: declared.to}
				out = append(out, apicoreset.DependencyEdge{
					From: id, To: to, Relation: string(domain.ClosureKindScenario), DeclaredBy: id.Name,
					Detail: declared.detail, Required: declared.edge.Required, Policy: declared.edge.StartupPolicy,
				})
				r.edges = append(r.edges, edge{fromKind: domain.ClosureKindScenario, from: id.Name, toKind: domain.ClosureKindScenario, to: declared.to, required: declared.edge.Required, selected: selected, policy: declared.edge.StartupPolicy, detail: declared.detail})
			}
			for _, declared := range resourceEdges {
				passes, selected := r.edgePasses(declared.to, declared.edge)
				if !passes {
					continue
				}
				child, err := r.loadResource(declared.to, id.Name)
				if err != nil {
					return nil, err
				}
				child.included = true
				child.optionalSelected = child.optionalSelected || selected
				child.reasons = append(child.reasons, r.reasonsForEdge(id.Name, declared, selected)...)
				to := apicoreset.ComponentID{Kind: string(domain.ClosureKindResource), Name: declared.to}
				out = append(out, apicoreset.DependencyEdge{
					From: id, To: to, Relation: string(domain.ClosureKindResource), DeclaredBy: id.Name,
					Detail: declared.detail, Required: declared.edge.Required, Policy: declared.edge.StartupPolicy,
				})
				r.edges = append(r.edges, edge{fromKind: domain.ClosureKindScenario, from: id.Name, toKind: domain.ClosureKindResource, to: declared.to, required: declared.edge.Required, selected: selected, policy: declared.edge.StartupPolicy, detail: declared.detail})
			}
			return out, nil
		}
		if id.Kind == string(domain.ClosureKindResource) {
			node := r.resources[id.Name]
			if node == nil {
				return nil, unavailable(fmt.Sprintf("resource %q is not loaded into the dependency graph", id.Name), nil, map[string]any{"component": id.Name, "kind": id.Kind})
			}
			out := make([]apicoreset.DependencyEdge, 0, len(node.decl.Dependencies))
			dependencies := append([]string(nil), node.decl.Dependencies...)
			sort.Strings(dependencies)
			for _, dependency := range dependencies {
				dependency = strings.TrimSpace(dependency)
				if dependency == "" {
					continue
				}
				child, err := r.loadResource(dependency, "resource:"+id.Name)
				if err != nil {
					return nil, err
				}
				child.included = true
				child.reasons = append(child.reasons,
					domain.ClosureReason{Kind: domain.ClosureReasonDeclaredBy, From: "resource:" + id.Name, Detail: "declared in resource.json"},
					domain.ClosureReason{Kind: domain.ClosureReasonTransitiveVia, From: "resource:" + id.Name, Detail: "resource dependency"},
				)
				to := apicoreset.ComponentID{Kind: string(domain.ClosureKindResource), Name: dependency}
				out = append(out, apicoreset.DependencyEdge{
					From: id, To: to, Relation: string(domain.ClosureKindResource), DeclaredBy: "resource:" + id.Name,
					Detail: "declared in resource.json", Required: true, Policy: "must_start",
				})
				r.edges = append(r.edges, edge{fromKind: domain.ClosureKindResource, from: id.Name, toKind: domain.ClosureKindResource, to: dependency, required: true, detail: "declared in resource.json"})
			}
			return out, nil
		}
		return nil, nil
	})

	closure, err := apicoreset.ResolveFromSource(source, rootIDs, apicoreset.ResolveOptions{AllowCyclesWithOptionalEdges: true})
	if err != nil {
		if errors.Is(err, apicoreset.ErrCycle) {
			pathText := strings.TrimSpace(strings.TrimPrefix(err.Error(), apicoreset.ErrCycle.Error()+":"))
			parts := strings.Split(pathText, " -> ")
			for i := range parts {
				parts[i] = strings.TrimPrefix(parts[i], string(domain.ClosureKindScenario)+":")
			}
			return newError(CodeCycle, err.Error(), map[string]any{"path": parts})
		}
		if typed, ok := As(err); ok {
			return typed
		}
		return unavailable("shared dependency closure could not be resolved", err, nil)
	}
	r.dependencyDigest = closure.Digest
	closureByID := make(map[apicoreset.ComponentID]apicoreset.ClosureMember, len(closure.Members))
	for _, member := range closure.Members {
		closureByID[member.Component] = member
	}
	// The shared result is authoritative for included membership. Catalog nodes
	// are loaded only to provide owner-specific metadata and projections.
	for _, member := range closure.Members {
		switch domain.ClosureComponentKind(member.Component.Kind) {
		case domain.ClosureKindScenario:
			if node := r.scenarios[member.Component.Name]; node != nil {
				node.included = true
				r.setTransitivePaths(node.reasons, member, closureByID)
			}
		case domain.ClosureKindResource:
			if node := r.resources[member.Component.Name]; node != nil {
				node.included = true
				r.setTransitivePaths(node.reasons, member, closureByID)
			}
		}
	}
	return nil
}

func (r *resolver) setTransitivePaths(reasons []domain.ClosureReason, member apicoreset.ClosureMember, closure map[apicoreset.ComponentID]apicoreset.ClosureMember) {
	for i := range reasons {
		if reasons[i].Kind != domain.ClosureReasonTransitiveVia {
			continue
		}
		parentID := apicoreset.ComponentID{Kind: string(domain.ClosureKindScenario), Name: reasons[i].From}
		if strings.HasPrefix(reasons[i].From, "resource:") {
			parentID.Kind = string(domain.ClosureKindResource)
			parentID.Name = strings.TrimPrefix(reasons[i].From, "resource:")
		}
		parent, ok := closure[parentID]
		if !ok {
			continue
		}
		parts := []string{parent.Root.Name}
		for _, edge := range parent.Path {
			parts = append(parts, edge.To.Name)
		}
		parts = append(parts, member.Component.Name)
		reasons[i].Detail = "path " + strings.Join(parts, " -> ")
	}
}

func (r *resolver) reasonsForEdge(from string, declared declaredEdge, selected bool) []domain.ClosureReason {
	detail := declared.detail
	if declared.edge.Description != "" {
		detail += ": " + declared.edge.Description
	}
	reasons := []domain.ClosureReason{{Kind: domain.ClosureReasonDeclaredBy, From: from, Detail: detail}}
	if from != r.in.ScenarioID {
		reasons = append(reasons, domain.ClosureReason{Kind: domain.ClosureReasonTransitiveVia, From: from, Detail: "path dependency"})
	}
	if selected {
		source := "manifest default (enabled optional dependency)"
		if r.selectSet[declared.to] {
			source = "operator selection"
		}
		reasons = append(reasons, domain.ClosureReason{Kind: domain.ClosureReasonSelectedBy, From: from, Detail: source})
	}
	return reasons
}

// propagateRequired computes requiredness as a fixpoint over required edges
// from required, included components. Optional selection never flips it.
func (r *resolver) propagateRequired() {
	changed := true
	for changed {
		changed = false
		for _, e := range r.edges {
			if !e.required {
				continue
			}
			var fromRequired bool
			switch e.fromKind {
			case domain.ClosureKindScenario:
				fromRequired = r.scenarios[e.from].required
			case domain.ClosureKindResource:
				fromRequired = r.resources[e.from].required
			}
			if !fromRequired {
				continue
			}
			switch e.toKind {
			case domain.ClosureKindScenario:
				if node := r.scenarios[e.to]; !node.required {
					node.required = true
					changed = true
				}
			case domain.ClosureKindResource:
				if node := r.resources[e.to]; !node.required {
					node.required = true
					changed = true
				}
			}
		}
	}
}

// checkConflicts rejects contradictory operating modes: a one-shot scenario
// with auto-restart, or a requested provider mode the resource forbids.
func (r *resolver) checkConflicts() error {
	for _, id := range sortedScenarioIDs(r.scenarios) {
		node := r.scenarios[id]
		if !node.included {
			continue
		}
		autoRestart, _ := r.autoRestart(node)
		if node.decl.RuntimeKind == "one_shot" && autoRestart {
			return newError(CodeConflict, fmt.Sprintf("scenario %q is one_shot but auto-restart is enabled", id), map[string]any{"component": id, "runtime_kind": "one_shot", "auto_restart": true})
		}
	}
	for _, id := range sortedResourceIDs(r.resources) {
		node := r.resources[id]
		if !node.included {
			continue
		}
		mode, ok := r.in.Overrides.OperatingMode[id]
		if !ok || mode == "" {
			continue
		}
		if len(node.decl.AllowedModes) > 0 && !containsFold(node.decl.AllowedModes, mode) {
			return newError(CodeConflict, fmt.Sprintf("resource %q does not allow operating mode %q", id, mode), map[string]any{"component": id, "requested_mode": mode, "allowed_modes": node.decl.AllowedModes})
		}
	}
	return nil
}

func (r *resolver) autoRestart(node *scenarioNode) (value bool, source string) {
	if override, ok := r.in.Overrides.AutoRestart[node.decl.ID]; ok {
		return override, "override"
	}
	return node.decl.AutoRestartDefault, "declared"
}

func (r *resolver) supervisionMember(node *scenarioNode) bool {
	if override, ok := r.in.Overrides.SupervisionMember[node.decl.ID]; ok {
		return override
	}
	return node.member
}

func strongestPolicy(policies []string) string {
	rank := map[string]int{"must_start": 3, "try_start": 2, "ignore": 1}
	best := ""
	for _, policy := range policies {
		if rank[policy] > rank[best] {
			best = policy
		}
	}
	if best == "" {
		return "try_start"
	}
	return best
}

func sortedScenarioIDs(nodes map[string]*scenarioNode) []string {
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedResourceIDs(nodes map[string]*resourceNode) []string {
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (r *resolver) addUnsupported(component, code, detail string) {
	r.unsupported = append(r.unsupported, domain.ClosureUnsupported{Component: component, ReasonCode: code, Detail: detail})
}
