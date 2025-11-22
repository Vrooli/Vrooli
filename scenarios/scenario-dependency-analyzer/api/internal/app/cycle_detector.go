package app

import (
	"fmt"

	types "scenario-dependency-analyzer/internal/types"
)

// CycleDetectionResult represents the outcome of circular dependency analysis
type CycleDetectionResult struct {
	HasCycles    bool                `json:"has_cycles"`
	CycleCount   int                 `json:"cycle_count"`
	Cycles       []DependencyCycle   `json:"cycles"`
	AffectedDeps []string            `json:"affected_dependencies"`
	Severity     string              `json:"severity"` // "none", "warning", "critical"
	Message      string              `json:"message"`
}

// DependencyCycle represents a single circular dependency path
type DependencyCycle struct {
	Path        []string               `json:"path"`
	Length      int                    `json:"length"`
	CycleType   string                 `json:"cycle_type"` // "scenario", "resource", "mixed"
	Required    bool                   `json:"required"`   // true if all edges are required
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// detectCycles performs circular dependency detection on the dependency graph
func detectCycles(graph *types.DependencyGraph) *CycleDetectionResult {
	if graph == nil || len(graph.Edges) == 0 {
		return &CycleDetectionResult{
			HasCycles: false,
			Severity:  "none",
			Message:   "No dependencies to analyze",
		}
	}

	// Build adjacency list for efficient traversal
	adjList := buildAdjacencyList(graph.Edges)

	// Track visited nodes and current path for cycle detection
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)
	var cycles []DependencyCycle

	// Check each node as a potential cycle start
	for _, node := range graph.Nodes {
		if !visited[node.ID] {
			foundCycles := dfsDetectCycles(node.ID, adjList, visited, recursionStack, graph.Edges, []string{})
			cycles = append(cycles, foundCycles...)
		}
	}

	// Deduplicate cycles (same cycle discovered from different starting points)
	uniqueCycles := deduplicateCycles(cycles)

	// Determine severity
	severity := calculateCycleSeverity(uniqueCycles)

	// Extract affected dependencies
	affectedDeps := extractAffectedDependencies(uniqueCycles)

	message := generateCycleMessage(len(uniqueCycles), severity)

	return &CycleDetectionResult{
		HasCycles:    len(uniqueCycles) > 0,
		CycleCount:   len(uniqueCycles),
		Cycles:       uniqueCycles,
		AffectedDeps: affectedDeps,
		Severity:     severity,
		Message:      message,
	}
}

// buildAdjacencyList creates a map of node -> list of nodes it depends on
func buildAdjacencyList(edges []types.GraphEdge) map[string][]types.GraphEdge {
	adjList := make(map[string][]types.GraphEdge)

	for _, edge := range edges {
		adjList[edge.Source] = append(adjList[edge.Source], edge)
	}

	return adjList
}

// dfsDetectCycles performs depth-first search to find cycles
func dfsDetectCycles(
	nodeID string,
	adjList map[string][]types.GraphEdge,
	visited map[string]bool,
	recursionStack map[string]bool,
	allEdges []types.GraphEdge,
	currentPath []string,
) []DependencyCycle {
	visited[nodeID] = true
	recursionStack[nodeID] = true
	currentPath = append(currentPath, nodeID)

	var cycles []DependencyCycle

	// Explore all neighbors
	for _, edge := range adjList[nodeID] {
		targetID := edge.Target

		if !visited[targetID] {
			// Recursively explore
			foundCycles := dfsDetectCycles(targetID, adjList, visited, recursionStack, allEdges, currentPath)
			cycles = append(cycles, foundCycles...)
		} else if recursionStack[targetID] {
			// Cycle detected! Build the cycle path
			cycleStart := findInPath(currentPath, targetID)
			if cycleStart >= 0 {
				cyclePath := append(currentPath[cycleStart:], targetID) // Close the cycle
				cycle := buildCycleInfo(cyclePath, allEdges)
				cycles = append(cycles, cycle)
			}
		}
	}

	recursionStack[nodeID] = false
	return cycles
}

// findInPath returns the index of nodeID in path, or -1 if not found
func findInPath(path []string, nodeID string) int {
	for i, id := range path {
		if id == nodeID {
			return i
		}
	}
	return -1
}

// buildCycleInfo creates a DependencyCycle from a path
func buildCycleInfo(path []string, allEdges []types.GraphEdge) DependencyCycle {
	cycleType := determineCycleType(path, allEdges)
	allRequired := checkAllEdgesRequired(path, allEdges)
	description := generateCycleDescription(path, cycleType)

	return DependencyCycle{
		Path:        path,
		Length:      len(path) - 1, // Exclude the duplicate end node
		CycleType:   cycleType,
		Required:    allRequired,
		Description: description,
		Metadata: map[string]interface{}{
			"cycle_signature": generateCycleSignature(path),
		},
	}
}

// determineCycleType identifies if cycle involves scenarios, resources, or both
func determineCycleType(path []string, allEdges []types.GraphEdge) string {
	hasScenario := false
	hasResource := false

	for i := 0; i < len(path)-1; i++ {
		edge := findEdge(path[i], path[i+1], allEdges)
		if edge != nil {
			if edge.Type == "scenario" {
				hasScenario = true
			} else if edge.Type == "resource" {
				hasResource = true
			}
		}
	}

	if hasScenario && hasResource {
		return "mixed"
	} else if hasScenario {
		return "scenario"
	}
	return "resource"
}

// checkAllEdgesRequired returns true if all edges in the cycle are required
func checkAllEdgesRequired(path []string, allEdges []types.GraphEdge) bool {
	for i := 0; i < len(path)-1; i++ {
		edge := findEdge(path[i], path[i+1], allEdges)
		if edge == nil || !edge.Required {
			return false
		}
	}
	return true
}

// findEdge finds an edge between source and target
func findEdge(source, target string, edges []types.GraphEdge) *types.GraphEdge {
	for i := range edges {
		if edges[i].Source == source && edges[i].Target == target {
			return &edges[i]
		}
	}
	return nil
}

// generateCycleSignature creates a normalized signature for cycle deduplication
func generateCycleSignature(path []string) string {
	if len(path) <= 1 {
		return ""
	}

	// Find the lexicographically smallest node as the starting point
	minIdx := 0
	for i := 1; i < len(path)-1; i++ { // Exclude last element (duplicate)
		if path[i] < path[minIdx] {
			minIdx = i
		}
	}

	// Build normalized path starting from the smallest node
	normalized := make([]string, len(path)-1)
	for i := 0; i < len(normalized); i++ {
		normalized[i] = path[(minIdx+i)%(len(path)-1)]
	}

	signature := ""
	for _, node := range normalized {
		signature += node + "->"
	}
	return signature
}

// deduplicateCycles removes duplicate cycles based on their signature
func deduplicateCycles(cycles []DependencyCycle) []DependencyCycle {
	seen := make(map[string]bool)
	var unique []DependencyCycle

	for _, cycle := range cycles {
		sig, ok := cycle.Metadata["cycle_signature"].(string)
		if !ok {
			continue
		}

		if !seen[sig] {
			seen[sig] = true
			unique = append(unique, cycle)
		}
	}

	return unique
}

// calculateCycleSeverity determines the severity level
func calculateCycleSeverity(cycles []DependencyCycle) string {
	if len(cycles) == 0 {
		return "none"
	}

	// Check for critical cycles (all required dependencies)
	for _, cycle := range cycles {
		if cycle.Required {
			return "critical"
		}
	}

	// If we have cycles but none are all-required, it's a warning
	return "warning"
}

// extractAffectedDependencies returns a sorted list of unique dependencies in cycles
func extractAffectedDependencies(cycles []DependencyCycle) []string {
	depSet := make(map[string]bool)

	for _, cycle := range cycles {
		for _, dep := range cycle.Path {
			depSet[dep] = true
		}
	}

	var deps []string
	for dep := range depSet {
		deps = append(deps, dep)
	}

	return deps
}

// generateCycleDescription creates a human-readable description
func generateCycleDescription(path []string, cycleType string) string {
	if len(path) <= 1 {
		return "Invalid cycle path"
	}

	// Remove duplicate end node for cleaner description
	cleanPath := path[:len(path)-1]

	return fmt.Sprintf("%s cycle: %s forms a circular dependency loop (%d hops)",
		cycleType,
		joinPath(cleanPath),
		len(cleanPath))
}

// joinPath creates "A → B → C" style path string
func joinPath(path []string) string {
	if len(path) == 0 {
		return ""
	}
	result := path[0]
	for i := 1; i < len(path); i++ {
		result += " → " + path[i]
	}
	result += " → " + path[0] // Show it cycles back
	return result
}

// generateCycleMessage creates an overall message about cycle detection
func generateCycleMessage(cycleCount int, severity string) string {
	if cycleCount == 0 {
		return "No circular dependencies detected. Dependency graph is acyclic."
	}

	if severity == "critical" {
		return fmt.Sprintf("CRITICAL: Found %d circular dependency cycle(s) involving required dependencies. This will cause deployment deadlocks.", cycleCount)
	}

	return fmt.Sprintf("WARNING: Found %d circular dependency cycle(s) involving optional dependencies. Review and consider breaking these cycles.", cycleCount)
}
