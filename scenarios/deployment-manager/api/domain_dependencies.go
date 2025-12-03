package main

import "strings"

// DetectCircularDependencies finds circular references in the dependency graph.
// This is a pure domain algorithm that operates on analysis data structure.
func DetectCircularDependencies(analysisData map[string]interface{}) []string {
	circular := []string{}

	deps, ok := analysisData["dependencies"].(map[string]interface{})
	if !ok {
		return circular
	}

	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var dfs func(string, []string) bool
	dfs = func(node string, path []string) bool {
		if stack[node] {
			cycleStart := -1
			for i, p := range path {
				if p == node {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cyclePath := append(path[cycleStart:], node)
				circular = append(circular, strings.Join(cyclePath, " → "))
			}
			return true
		}

		if visited[node] {
			return false
		}

		visited[node] = true
		stack[node] = true
		path = append(path, node)

		if nodeDeps, ok := deps[node].(map[string]interface{}); ok {
			if subDeps, ok := nodeDeps["dependencies"].(map[string]interface{}); ok {
				for depName := range subDeps {
					if dfs(depName, path) {
						stack[node] = false
						return true
					}
				}
			}
		}

		stack[node] = false
		return false
	}

	for depName := range deps {
		if !visited[depName] {
			dfs(depName, []string{})
		}
	}

	return circular
}

// CalculateAggregateRequirements computes total resource requirements from analysis data.
// Returns a map of resource type to required amount.
func CalculateAggregateRequirements(analysisData map[string]interface{}) map[string]interface{} {
	// Placeholder for aggregate requirement calculation.
	// A full implementation would traverse analysisData["dependencies"] and sum
	// resource annotations (memory, cpu, gpu, storage, network) from each node.
	return map[string]interface{}{
		"memory":  "512MB",
		"cpu":     "1 core",
		"gpu":     "none",
		"storage": "1GB",
		"network": "broadband",
	}
}
