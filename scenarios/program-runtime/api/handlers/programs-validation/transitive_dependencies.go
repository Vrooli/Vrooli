package programsvalidation

import (
	"fmt"
	"strings"

	"program-runtime/internal/bindings"
	"program-runtime/internal/contracts"
)

// transitiveDependencyFindings makes scenario startup dependencies follow the
// complete declared-program call tree. A parent's service manifest must cover
// scenarios used by a child or grandchild because the composed program runs
// inside the parent's session and cannot assume those services were started.
func transitiveDependencyFindings(rootScenario, rootProgram string, root contracts.Contract, all []contracts.Contract, registry *bindings.Registry, declared map[string]struct{}, details ...*[]any) []string {
	byID := make(map[string]contracts.Contract, len(all))
	for _, contract := range all {
		byID[contract.ID] = contract
	}

	missing := make(map[string]map[string]struct{})
	visited := make(map[string]struct{})
	var visit func(contracts.Contract, int)
	visit = func(current contracts.Contract, depth int) {
		if _, ok := visited[current.ID]; ok {
			return
		}
		visited[current.ID] = struct{}{}

		for _, binding := range current.Bindings {
			scenario := bindingScenario(binding.ID, registry, current)
			addTransitiveDependency(missing, rootScenario, scenario, binding.ID, depth)
		}
		for _, match := range programLibraryTargetPattern.FindAllStringSubmatch(current.Source, -1) {
			if len(match) != 3 {
				continue
			}
			scenario := normalizeScenarioName(match[1])
			call := match[0]
			addTransitiveDependency(missing, rootScenario, scenario, call, depth)

			childName := strings.ReplaceAll(match[2], "_", "-")
			if child, ok := byID[scenario+"."+childName]; ok {
				visit(child, depth+1)
			}
		}
		if strings.Contains(current.Source, "learn.") {
			addTransitiveDependency(missing, rootScenario, "vrooli-memory", "learn.*", depth)
		}
	}
	visit(root, 0)

	findings := make([]string, 0)
	for scenario, calls := range missing {
		if _, ok := declared[scenario]; ok {
			continue
		}
		for call := range calls {
			findings = append(findings, "programs.dependency_undeclared")
			if len(details) > 0 && details[0] != nil {
				fix := fmt.Sprintf(`"%s": {"required": false, "startup_policy": "try_start", "purpose": "%s transitively uses %s", "degraded_behavior": "<fill in>", "bundle_policy": "either"}`, scenario, rootProgram, call)
				*details[0] = append(*details[0], map[string]any{
					"program": rootProgram, "binding_or_call": call, "target_scenario": scenario,
					"fix": fix, "kind": "dependency", "transitive": true,
					"message": fmt.Sprintf("%s: program %s transitively uses %s on %s through a declared child program, but .vrooli/service.json does not declare the scenario; add dependencies.scenarios.%s with startup_policy try_start", rootScenario, rootProgram, call, scenario, scenario),
				})
			}
		}
	}
	return uniqueStrings(findings)
}

func addTransitiveDependency(missing map[string]map[string]struct{}, rootScenario, target, call string, depth int) {
	target = normalizeScenarioName(target)
	if depth == 0 || target == "" || target == "vrooli" || target == normalizeScenarioName(rootScenario) || target == "program-runtime" {
		return
	}
	if missing[target] == nil {
		missing[target] = make(map[string]struct{})
	}
	missing[target][call] = struct{}{}
}

func bindingScenario(id string, registry *bindings.Registry, owner contracts.Contract) string {
	if registry != nil {
		binding, ok := registry.Binding(id)
		if !ok {
			binding, ok = registry.ProgramPrivateBinding(id, owner.ID, owner.Digest)
		}
		if ok {
			return binding.GetScenario()
		}
	}
	if prefix, _, ok := strings.Cut(id, "/"); ok {
		return prefix
	}
	return ""
}

// transitiveBindingEffectFindings checks the effect declaration at every
// indexed child boundary. Each child is independently admitted at runtime, so
// its effect must match the governed binding it actually invokes.
func transitiveBindingEffectFindings(rootProgram string, root contracts.Contract, all []contracts.Contract, registry *bindings.Registry, details ...*[]any) []string {
	if registry == nil {
		return nil
	}
	byID := make(map[string]contracts.Contract, len(all))
	for _, contract := range all {
		byID[contract.ID] = contract
	}
	visited := make(map[string]struct{})
	findings := make([]string, 0)
	var visit func(contracts.Contract, bool)
	visit = func(current contracts.Contract, transitive bool) {
		if _, ok := visited[current.ID]; ok {
			return
		}
		visited[current.ID] = struct{}{}
		for _, declared := range current.Bindings {
			binding, ok := registry.Binding(declared.ID)
			if !ok {
				binding, ok = registry.ProgramPrivateBinding(declared.ID, current.ID, current.Digest)
			}
			if !ok || binding.GetEffect() == declared.Effect || !transitive {
				continue
			}
			findings = append(findings, "programs.binding_effect_mismatch")
			if len(details) > 0 && details[0] != nil {
				*details[0] = append(*details[0], map[string]any{
					"kind": "binding_effect", "program": rootProgram, "child_program": current.ID,
					"binding_id": declared.ID, "declared_effect": declared.Effect,
					"governed_effect": binding.GetEffect(),
					"message":         fmt.Sprintf("program %s composes %s, whose declaration assigns %s to %s but the governed binding effect is %s", rootProgram, current.ID, declared.Effect, declared.ID, binding.GetEffect()),
				})
			}
		}
		for _, match := range programLibraryTargetPattern.FindAllStringSubmatch(current.Source, -1) {
			if len(match) != 3 {
				continue
			}
			childID := normalizeScenarioName(match[1]) + "." + strings.ReplaceAll(match[2], "_", "-")
			if child, ok := byID[childID]; ok {
				visit(child, true)
			}
		}
	}
	visit(root, false)
	return uniqueStrings(findings)
}
