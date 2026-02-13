// DOC: docs/concepts/GRAPH.md#how-the-graph-is-built
package graph

import (
	"context"

	"prompt-manager/store"
)

// agentNodeSource provides agent listing for graph node construction.
type agentNodeSource interface {
	List(ctx context.Context) ([]store.Agent, error)
}

// teamNodeSource provides team listing for graph node construction.
type teamNodeSource interface {
	List(ctx context.Context) ([]store.Team, error)
}

// skillNodeSource provides skill listing for graph node construction.
type skillNodeSource interface {
	List(ctx context.Context) ([]store.Skill, error)
}

// graphScanner scans entities and returns graph edges.
type graphScanner interface {
	ScanAll(ctx context.Context) ([]Edge, error)
}

// Builder assembles the full graph from entity stores and the scanner.
type Builder struct {
	agentStore             agentNodeSource
	teamStore              teamNodeSource
	skillStore             skillNodeSource
	scanner                graphScanner
	scoreFns               []ScoreFn
	scenarioHealthProvider ScenarioHealthProvider
}

// NewBuilder creates a graph builder.
func NewBuilder(
	agentStore agentNodeSource,
	teamStore teamNodeSource,
	skillStore skillNodeSource,
	scanner graphScanner,
	scoreFns []ScoreFn,
) *Builder {
	return &Builder{
		agentStore: agentStore,
		teamStore:  teamStore,
		skillStore: skillStore,
		scanner:    scanner,
		scoreFns:   scoreFns,
	}
}

// SetScenarioHealthProvider configures scenario-level health lookup for CLI nodes.
func (b *Builder) SetScenarioHealthProvider(provider ScenarioHealthProvider) {
	b.scenarioHealthProvider = provider
}

// Build constructs the complete graph.
func (b *Builder) Build(ctx context.Context) (Graph, error) {
	var g Graph

	// Collect nodes from teams
	teams, err := b.teamStore.List(ctx)
	if err != nil {
		return g, err
	}
	for _, t := range teams {
		g.Nodes = append(g.Nodes, Node{
			ID:          t.ID,
			Type:        NodeTeam,
			Label:       t.DisplayName,
			Description: t.Mission,
		})
	}

	// Collect nodes from agents
	agents, err := b.agentStore.List(ctx)
	if err != nil {
		return g, err
	}
	for _, a := range agents {
		g.Nodes = append(g.Nodes, Node{
			ID:          a.ID,
			Type:        NodeAgent,
			Label:       a.DisplayName,
			Description: a.Description,
			Status:      a.Status,
			Tags:        a.Tags,
		})
	}

	// Collect nodes from skills
	skills, err := b.skillStore.List(ctx)
	if err != nil {
		return g, err
	}
	for _, s := range skills {
		g.Nodes = append(g.Nodes, Node{
			ID:          s.ID,
			Type:        NodeSkill,
			Label:       s.Name,
			Description: s.Description,
			Status:      s.Status,
			Tags:        s.Tags,
		})
	}

	// Scan for edges
	edges, err := b.scanner.ScanAll(ctx)
	if err != nil {
		return g, err
	}
	g.Edges = edges

	// Extract CLI nodes from edges
	cliNodes := extractCLINodes(g.Edges, g.Nodes)
	g.Nodes = append(g.Nodes, cliNodes...)

	// Compute health scores
	if len(b.scoreFns) > 0 {
		g.HealthScores = ScoreAll(g, b.scoreFns)
		g.HealthScores = ApplyCLIHealthPolicy(ctx, g, g.HealthScores, b.scenarioHealthProvider)
	}

	return g, nil
}

// extractCLINodes finds CLI node IDs referenced in edges that don't
// already exist as nodes and creates nodes for them.
func extractCLINodes(edges []Edge, existingNodes []Node) []Node {
	existing := make(map[string]bool, len(existingNodes))
	for _, n := range existingNodes {
		existing[n.ID] = true
	}

	seen := make(map[string]bool)
	var nodes []Node
	for _, e := range edges {
		if e.Kind == EdgeCodeUsage && !existing[e.To] && !seen[e.To] {
			seen[e.To] = true
			label := e.To
			if len(label) > 4 && label[:4] == "cli:" {
				label = label[4:]
			}
			nodes = append(nodes, Node{
				ID:    e.To,
				Type:  NodeCLI,
				Label: label,
			})
		}
	}
	return nodes
}
