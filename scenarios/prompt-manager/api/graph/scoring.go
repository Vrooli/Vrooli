// DOC: docs/concepts/GRAPH.md#health-scoring
package graph

import (
	"context"
	"encoding/json"
	"math"
	"os/exec"
	"strings"
	"time"
)

// ScoreFn computes a named scoring factor for a node within the graph context.
type ScoreFn struct {
	Name   string
	Weight float64
	Fn     func(nodeID string, g Graph) float64
}

// ScenarioHealthProvider resolves normalized (0-1) health for scenario CLIs.
type ScenarioHealthProvider interface {
	ScenarioScore(ctx context.Context, scenario string) (float64, error)
}

var factorEvaluators = map[string]func(nodeID string, g Graph) float64{
	"outgoing-edges":  outgoingEdgesScore,
	"incoming-edges":  incomingEdgesScore,
	"code-usage":      codeUsageScore,
	"recent-activity": recentActivityScore,
}

// ScoreAll computes health scores for all nodes using the provided score functions.
func ScoreAll(g Graph, fns []ScoreFn) []HealthScore {
	scores := make([]HealthScore, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		hs := HealthScore{
			NodeID:  n.ID,
			Factors: make(map[string]float64, len(fns)),
		}
		var total float64
		var weightSum float64
		for _, fn := range fns {
			val := fn.Fn(n.ID, g)
			hs.Factors[fn.Name] = val
			total += val * fn.Weight
			weightSum += fn.Weight
		}
		if weightSum > 0 {
			hs.Score = total / weightSum
		}
		scores = append(scores, hs)
	}
	return scores
}

// ScoreAllWithConfig computes health scores for all nodes using per-entity weights.
func ScoreAllWithConfig(g Graph, cfg HealthConfig) []HealthScore {
	scores := make([]HealthScore, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		weights := weightsForNodeType(n.Type, cfg)
		hs := HealthScore{
			NodeID: n.ID,
			Factors: map[string]float64{
				"outgoing-edges":  0,
				"incoming-edges":  0,
				"code-usage":      0,
				"recent-activity": 0,
			},
		}
		score, factors := scoreWithWeights(n.ID, g, weights)
		hs.Score = score
		for k, v := range factors {
			hs.Factors[k] = v
		}
		scores = append(scores, hs)
	}
	return scores
}

// ApplyCLIHealthPolicy rewrites CLI node health semantics:
//   - non-vrooli tools: score 0.0 (portability penalty)
//   - "vrooli": neutral (no score row emitted)
//   - scenario CLIs: score from ScenarioHealthProvider (if available)
func ApplyCLIHealthPolicy(ctx context.Context, g Graph, scores []HealthScore, provider ScenarioHealthProvider) []HealthScore {
	return ApplyCLIHealthPolicyWithConfig(ctx, g, scores, provider, DefaultHealthConfig().CLI)
}

// ApplyCLIHealthPolicyWithConfig rewrites CLI node health semantics using explicit policy config.
func ApplyCLIHealthPolicyWithConfig(ctx context.Context, g Graph, scores []HealthScore, provider ScenarioHealthProvider, cfg CLIHealthConfig) []HealthScore {
	if len(g.Nodes) == 0 {
		return scores
	}

	nodeByID := make(map[string]Node, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeByID[n.ID] = n
	}
	scoreByNodeID := make(map[string]HealthScore, len(scores))
	for _, hs := range scores {
		scoreByNodeID[hs.NodeID] = hs
	}

	type cliUsageSummary struct {
		command       string
		isScenarioCLI bool
	}
	usageByNodeID := make(map[string]cliUsageSummary)
	for _, e := range g.Edges {
		if e.Kind != EdgeCodeUsage {
			continue
		}
		targetNode, ok := nodeByID[e.To]
		if !ok || targetNode.Type != NodeCLI {
			continue
		}
		current := usageByNodeID[e.To]
		cmd := strings.TrimPrefix(e.To, "cli:")
		if e.Command != "" {
			cmd = e.Command
		}
		if cmd != "" {
			current.command = cmd
		}
		if e.Category == CodeScenarioCLI {
			current.isScenarioCLI = true
		}
		usageByNodeID[e.To] = current
	}

	resolvedScenarioScores := make(map[string]float64)
	const portabilityFactor = "cli-portability"
	const scenarioCompletenessFactor = "scenario-completeness"
	neutralCommands := make(map[string]bool, len(cfg.NeutralCommands))
	for _, cmd := range cfg.NeutralCommands {
		neutralCommands[strings.TrimSpace(cmd)] = true
	}

	for _, n := range g.Nodes {
		if n.Type != NodeCLI {
			continue
		}
		usage := usageByNodeID[n.ID]
		command := usage.command
		if command == "" {
			command = strings.TrimPrefix(n.ID, "cli:")
		}
		command = strings.TrimSpace(command)

		// Neutral commands are unscored and omitted from graph health rows.
		if neutralCommands[command] {
			delete(scoreByNodeID, n.ID)
			continue
		}

		// Penalize non-scenario tools.
		if !usage.isScenarioCLI {
			scoreByNodeID[n.ID] = HealthScore{
				NodeID: n.ID,
				Score:  cfg.ExternalToolScore,
				Factors: map[string]float64{
					portabilityFactor: cfg.ExternalToolScore,
				},
			}
			continue
		}

		// Scenario CLI: resolve scenario health via scenario-completeness-scoring.
		scenario := command
		score, ok := resolvedScenarioScores[scenario]
		if !ok {
			score = cfg.ScenarioFallbackScore
			if provider != nil {
				if resolved, err := provider.ScenarioScore(ctx, scenario); err == nil {
					score = normalizeScenarioScore(resolved)
				}
			}
			resolvedScenarioScores[scenario] = score
		}
		scoreByNodeID[n.ID] = HealthScore{
			NodeID: n.ID,
			Score:  score,
			Factors: map[string]float64{
				scenarioCompletenessFactor: score,
			},
		}
	}

	merged := make([]HealthScore, 0, len(scoreByNodeID))
	for _, n := range g.Nodes {
		hs, ok := scoreByNodeID[n.ID]
		if !ok {
			continue
		}
		merged = append(merged, hs)
	}
	return merged
}

type scenarioCompletenessJSON struct {
	Score float64 `json:"score"`
}

// ScenarioCompletenessCLIProvider resolves scores by invoking:
// scenario-completeness-scoring score <scenario> --json
type ScenarioCompletenessCLIProvider struct {
	timeout time.Duration
}

func NewScenarioCompletenessCLIProvider(timeout time.Duration) *ScenarioCompletenessCLIProvider {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &ScenarioCompletenessCLIProvider{timeout: timeout}
}

func (p *ScenarioCompletenessCLIProvider) ScenarioScore(ctx context.Context, scenario string) (float64, error) {
	callCtx := ctx
	cancel := func() {}
	if p.timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, p.timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(callCtx, "scenario-completeness-scoring", "score", scenario, "--json")
	output, err := cmd.Output()
	if err != nil {
		return 0.0, err
	}

	var parsed scenarioCompletenessJSON
	if err := json.Unmarshal(output, &parsed); err != nil {
		return 0.0, err
	}
	return parsed.Score, nil
}

// DefaultScoreFns returns the default set of scoring functions.
func DefaultScoreFns() []ScoreFn {
	return []ScoreFn{
		{Name: "outgoing-edges", Weight: 1.0, Fn: outgoingEdgesScore},
		{Name: "incoming-edges", Weight: 1.0, Fn: incomingEdgesScore},
		{Name: "code-usage", Weight: 0.5, Fn: codeUsageScore},
		{Name: "recent-activity", Weight: 0.5, Fn: recentActivityScore},
	}
}

func weightsForNodeType(nodeType NodeType, cfg HealthConfig) HealthWeights {
	switch nodeType {
	case NodeTeam:
		return cfg.Team
	case NodeAgent:
		return cfg.Agent
	case NodeSkill:
		return cfg.Skill
	default:
		// CLI scores are overridden by ApplyCLIHealthPolicyWithConfig, but
		// we still compute a baseline for consistency.
		return cfg.Skill
	}
}

func scoreWithWeights(nodeID string, g Graph, weights HealthWeights) (float64, map[string]float64) {
	weightByFactor := map[string]float64{
		"outgoing-edges":  weights.OutgoingEdges,
		"incoming-edges":  weights.IncomingEdges,
		"code-usage":      weights.CodeUsage,
		"recent-activity": weights.RecentActivity,
	}

	factors := make(map[string]float64, len(weightByFactor))
	var total float64
	var weightSum float64
	for factorName, weight := range weightByFactor {
		evaluator, ok := factorEvaluators[factorName]
		if !ok {
			factors[factorName] = 0
			continue
		}
		value := evaluator(nodeID, g)
		factors[factorName] = value
		if weight <= 0 {
			continue
		}
		total += value * weight
		weightSum += weight
	}
	if weightSum <= 0 {
		return 0, factors
	}
	return total / weightSum, factors
}

func normalizeScenarioScore(score float64) float64 {
	if score > 1.0 {
		score = score / 100.0
	}
	if score < 0 {
		return 0.0
	}
	if score > 1 {
		return 1.0
	}
	return score
}

// outgoingEdgesScore: higher is better, capped at 1.0.
func outgoingEdgesScore(nodeID string, g Graph) float64 {
	count := 0
	for _, e := range g.Edges {
		if e.From == nodeID {
			count++
		}
	}
	// Normalize: 5+ outgoing edges is "full score"
	return math.Min(float64(count)/5.0, 1.0)
}

// incomingEdgesScore: higher is better (node is well-referenced).
func incomingEdgesScore(nodeID string, g Graph) float64 {
	count := 0
	for _, e := range g.Edges {
		if e.To == nodeID {
			count++
		}
	}
	return math.Min(float64(count)/5.0, 1.0)
}

// codeUsageScore rewards Vrooli-only tool use and penalizes external tools.
//
//	1.0 — only Vrooli CLI usage (incentivized)
//	0.5 — no tool usage detected (neutral)
//	0.1 — has external tool or script usage (penalty)
func codeUsageScore(nodeID string, g Graph) float64 {
	hasVrooli := false
	hasExternal := false
	for _, e := range g.Edges {
		if e.From == nodeID && e.Kind == EdgeCodeUsage {
			switch e.Category {
			case CodeScenarioCLI:
				hasVrooli = true
			case CodeExternalTool, CodeScript:
				hasExternal = true
			}
		}
	}
	if hasExternal {
		return 0.1
	}
	if hasVrooli {
		return 1.0
	}
	return 0.5
}

// recentActivityScore: based on the node's updatedAt timestamp.
// Nodes updated within 7 days get 1.0, decaying over 90 days to 0.
func recentActivityScore(nodeID string, g Graph) float64 {
	for _, n := range g.Nodes {
		if n.ID != nodeID {
			continue
		}
		// Nodes don't carry timestamps directly in the graph; return 0.5 as neutral.
		return 0.5
	}
	return 0.0
}

// RecentActivityScoreFromTimestamp computes the recency score from a timestamp string.
func RecentActivityScoreFromTimestamp(updatedAt string) float64 {
	t, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return 0.0
	}
	daysSince := time.Since(t).Hours() / 24.0
	if daysSince <= 7 {
		return 1.0
	}
	if daysSince >= 90 {
		return 0.0
	}
	// Linear decay from 1.0 at 7 days to 0.0 at 90 days
	return 1.0 - (daysSince-7)/(90-7)
}
