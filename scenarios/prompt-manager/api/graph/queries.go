// DOC: docs/concepts/GRAPH.md#analytical-queries
package graph

// OrphanedSkills returns skills with zero incoming edges.
func OrphanedSkills(g Graph) []Node {
	incoming := make(map[string]int)
	for _, e := range g.Edges {
		incoming[e.To]++
	}

	var result []Node
	for _, n := range g.Nodes {
		if n.Type == NodeSkill && incoming[n.ID] == 0 {
			result = append(result, n)
		}
	}
	return result
}

// SkilllessAgents returns agents with zero outgoing edges to skills.
func SkilllessAgents(g Graph) []Node {
	hasSkill := make(map[string]bool)
	for _, e := range g.Edges {
		if isSkillEdge(e.Kind) {
			hasSkill[e.From] = true
		}
	}

	var result []Node
	for _, n := range g.Nodes {
		if n.Type == NodeAgent && !hasSkill[n.ID] {
			result = append(result, n)
		}
	}
	return result
}

// EmptyTeams returns teams with zero membership edges.
func EmptyTeams(g Graph) []Node {
	hasMember := make(map[string]bool)
	for _, e := range g.Edges {
		if e.Kind == EdgeMembership {
			hasMember[e.From] = true
		}
	}

	var result []Node
	for _, n := range g.Nodes {
		if n.Type == NodeTeam && !hasMember[n.ID] {
			result = append(result, n)
		}
	}
	return result
}

// UnaffiliatedAgents returns agents that belong to zero teams.
func UnaffiliatedAgents(g Graph) []Node {
	inTeam := make(map[string]bool)
	for _, e := range g.Edges {
		if e.Kind == EdgeMembership {
			inTeam[e.To] = true
		}
	}

	var result []Node
	for _, n := range g.Nodes {
		if n.Type == NodeAgent && !inTeam[n.ID] {
			result = append(result, n)
		}
	}
	return result
}

// CLIlessSkills returns skills that have no Vrooli CLI code-usage edges.
// A skill that only uses external tools (grep, curl, etc.) still appears here
// because it doesn't use Vrooli-controlled CLIs.
func CLIlessSkills(g Graph) []Node {
	hasCLI := make(map[string]bool)
	for _, e := range g.Edges {
		if e.Kind == EdgeCodeUsage && e.Category == CodeScenarioCLI {
			hasCLI[e.From] = true
		}
	}

	var result []Node
	for _, n := range g.Nodes {
		if n.Type == NodeSkill && !hasCLI[n.ID] {
			result = append(result, n)
		}
	}
	return result
}

// ExternalToolSkills returns skills that have external tool or script code-usage edges.
// These are skills that need their external tools wrapped in Vrooli CLIs.
func ExternalToolSkills(g Graph) []Node {
	hasExternal := make(map[string]bool)
	for _, e := range g.Edges {
		if e.Kind == EdgeCodeUsage && (e.Category == CodeExternalTool || e.Category == CodeScript) {
			hasExternal[e.From] = true
		}
	}

	var result []Node
	for _, n := range g.Nodes {
		if n.Type == NodeSkill && hasExternal[n.ID] {
			result = append(result, n)
		}
	}
	return result
}

// Popular returns the top N nodes by incoming edge count.
func Popular(g Graph, limit int) []Node {
	incoming := make(map[string]int)
	for _, e := range g.Edges {
		incoming[e.To]++
	}

	nodeMap := make(map[string]Node)
	for _, n := range g.Nodes {
		nodeMap[n.ID] = n
	}

	// Sort by incoming count descending
	type ranked struct {
		node  Node
		count int
	}
	var items []ranked
	for id, count := range incoming {
		if n, ok := nodeMap[id]; ok {
			items = append(items, ranked{n, count})
		}
	}

	// Simple selection sort for top N (N is typically small)
	for i := 0; i < len(items) && i < limit; i++ {
		maxIdx := i
		for j := i + 1; j < len(items); j++ {
			if items[j].count > items[maxIdx].count {
				maxIdx = j
			}
		}
		items[i], items[maxIdx] = items[maxIdx], items[i]
	}

	result := make([]Node, 0, limit)
	for i := 0; i < len(items) && i < limit; i++ {
		result = append(result, items[i].node)
	}
	return result
}

// DetectCircularRefs finds cycles in the graph using DFS.
// Returns a list of cycles, each cycle is a list of node IDs.
func DetectCircularRefs(g Graph) [][]string {
	// Build adjacency list (only skill-to-skill edges for meaningful cycles)
	adj := make(map[string][]string)
	for _, e := range g.Edges {
		if isSkillEdge(e.Kind) {
			adj[e.From] = append(adj[e.From], e.To)
		}
	}

	var cycles [][]string
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	path := make([]string, 0)

	var dfs func(node string)
	dfs = func(node string) {
		visited[node] = true
		inStack[node] = true
		path = append(path, node)

		for _, neighbor := range adj[node] {
			if !visited[neighbor] {
				dfs(neighbor)
			} else if inStack[neighbor] {
				// Found a cycle - extract it
				cycle := extractCycle(path, neighbor)
				if len(cycle) > 0 {
					cycles = append(cycles, cycle)
				}
			}
		}

		path = path[:len(path)-1]
		inStack[node] = false
	}

	skillNodes := make(map[string]bool)
	for _, n := range g.Nodes {
		if n.Type == NodeSkill {
			skillNodes[n.ID] = true
		}
	}

	for id := range skillNodes {
		if !visited[id] {
			dfs(id)
		}
	}

	return cycles
}

// extractCycle extracts the cycle from the DFS path.
func extractCycle(path []string, target string) []string {
	for i, id := range path {
		if id == target {
			cycle := make([]string, len(path)-i)
			copy(cycle, path[i:])
			return cycle
		}
	}
	return nil
}

// isSkillEdge returns true for edge kinds that represent skill references.
func isSkillEdge(kind EdgeKind) bool {
	switch kind {
	case EdgeCLIRead, EdgeBoldListed, EdgeDefaultScope, EdgePathRef:
		return true
	default:
		return false
	}
}
