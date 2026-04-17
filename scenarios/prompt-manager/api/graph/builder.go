// DOC: docs/concepts/GRAPH.md#how-the-graph-is-built
package graph

import (
	"context"
	"prompt-manager/store"
	"strings"
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
	healthConfigProvider   HealthConfigProvider
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

// SetHealthConfigProvider configures graph health scoring controls.
func (b *Builder) SetHealthConfigProvider(provider HealthConfigProvider) {
	b.healthConfigProvider = provider
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

	// Collect raw metrics for entity-specific health factors.
	g.NodeMetrics = b.collectNodeMetrics(ctx, g, skills, agents, teams)

	// Compute health scores
	if len(b.scoreFns) > 0 {
		if b.healthConfigProvider != nil {
			cfg, err := b.healthConfigProvider.Get(ctx)
			if err != nil {
				return g, err
			}
			g.HealthScores = ScoreAllWithConfig(g, cfg)
			g.HealthScores = ApplyCLIHealthPolicyWithConfig(ctx, g, g.HealthScores, b.scenarioHealthProvider, cfg.CLI)
		} else {
			g.HealthScores = ScoreAll(g, b.scoreFns)
			g.HealthScores = ApplyCLIHealthPolicy(ctx, g, g.HealthScores, b.scenarioHealthProvider)
		}
	}

	return g, nil
}

func (b *Builder) collectNodeMetrics(ctx context.Context, g Graph, skills []store.Skill, agents []store.Agent, teams []store.Team) map[string]NodeMetricSet {
	metrics := make(map[string]NodeMetricSet, len(g.Nodes))
	for _, n := range g.Nodes {
		metrics[n.ID] = NodeMetricSet{}
	}

	// Team member count from graph membership edges.
	for _, e := range g.Edges {
		if e.Kind != EdgeMembership {
			continue
		}
		if _, ok := metrics[e.From]; !ok {
			metrics[e.From] = NodeMetricSet{}
		}
		metrics[e.From][metricTeamMemberCount]++
	}

	// Skill content tokens from SKILL.md.
	if skillContentStore, ok := b.skillStore.(interface {
		GetWithContent(ctx context.Context, id string) (*store.Skill, string, error)
	}); ok {
		for _, skill := range skills {
			_, content, err := skillContentStore.GetWithContent(ctx, skill.ID)
			if err != nil {
				continue
			}
			metrics[skill.ID][metricSkillContentTokens] = float64(countTokens(content))
		}
	}

	// Agent context load from markdown files under the agent directory.
	if agentFilesStore, ok := b.agentStore.(interface {
		ListFiles(ctx context.Context, agentID string) ([]store.AgentFileEntry, error)
		ReadFile(ctx context.Context, agentID, relPath string) (string, error)
	}); ok {
		for _, agent := range agents {
			files, err := agentFilesStore.ListFiles(ctx, agent.ID)
			if err != nil {
				continue
			}
			totalTokens := 0
			for _, f := range files {
				if f.IsDir || !strings.HasSuffix(strings.ToLower(f.Path), ".md") {
					continue
				}
				content, err := agentFilesStore.ReadFile(ctx, agent.ID, f.Path)
				if err != nil {
					continue
				}
				totalTokens += countTokens(content)
			}
			metrics[agent.ID][metricAgentContextTokens] = float64(totalTokens)
		}
	}

	// Team role metrics from membership records.
	if teamMembersStore, ok := b.teamStore.(interface {
		GetMembers(ctx context.Context, teamID string) ([]store.TeamMemberRelation, error)
	}); ok {
		for _, team := range teams {
			members, err := teamMembersStore.GetMembers(ctx, team.ID)
			if err != nil {
				continue
			}
			distinct := make(map[string]bool)
			withRole := 0
			for _, m := range members {
				if len(m.Roles) > 0 {
					withRole++
				}
				for _, role := range m.Roles {
					trimmed := strings.TrimSpace(role)
					if trimmed != "" {
						distinct[trimmed] = true
					}
				}
			}
			metrics[team.ID][metricTeamDistinctRoleCount] = float64(len(distinct))
			metrics[team.ID][metricTeamRoleAssignedMembers] = float64(withRole)
			if metrics[team.ID][metricTeamMemberCount] <= 0 {
				metrics[team.ID][metricTeamMemberCount] = float64(len(members))
			}
		}
	}

	return metrics
}

func countTokens(content string) int {
	return len(strings.Fields(content))
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
