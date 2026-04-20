package initiatives

import (
	"fmt"
	"strings"
	"swarm-manager/internal/depgraph"
)

// normalizeDependsOn trims whitespace, removes blanks, and dedupes while
// preserving order.
func normalizeDependsOn(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		name := strings.TrimSpace(r)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// validateDependsOn verifies every dep name resolves to an existing initiative
// (other than self), and that adding/replacing `self`'s dependency list does
// not introduce a cycle in the initiative DAG.
func (s *Service) validateDependsOn(self string, deps []string) error {
	if len(deps) == 0 {
		return nil
	}
	for _, d := range deps {
		if d == self {
			return fmt.Errorf("initiative %q cannot depend on itself", self)
		}
	}

	all, err := s.store.LoadAll()
	if err != nil {
		return fmt.Errorf("load initiatives for dependency validation: %w", err)
	}
	graph := make(map[string][]string, len(all)+1)
	known := make(map[string]bool, len(all)+1)
	for _, it := range all {
		graph[it.Name] = it.DependsOn
		known[it.Name] = true
	}
	// Overlay the candidate node's proposed deps.
	graph[self] = deps
	known[self] = true

	for _, d := range deps {
		if !known[d] {
			return fmt.Errorf("depends_on references unknown initiative %q", d)
		}
	}

	if cyclePath := depgraph.DetectCycleFrom(graph, self); cyclePath != "" {
		return fmt.Errorf("depends_on introduces a cycle: %s", cyclePath)
	}
	return nil
}
