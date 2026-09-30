// DOC: docs/concepts/GRAPH.md#health-scoring
package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/scenariocli"
	"github.com/vrooli/envkit-go"
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
	"outgoing-edges":            outgoingEdgesScore,
	"incoming-edges":            incomingEdgesScore,
	"code-usage":                codeUsageScore,
	"recent-activity":           recentActivityScore,
	"skill-content-length":      skillContentLengthScore,
	"agent-context-load":        agentContextLoadScore,
	"team-member-count-balance": teamMemberCountBalanceScore,
	"team-role-coverage":        teamRoleCoverageScore,
	"action-contract":           actionContractScore,
	"action-command":            actionCommandScore,
	"action-examples":           actionExamplesScore,
	"action-owner":              actionOwnerScore,
}

const (
	severityInfo     = "info"
	severityWarning  = "warning"
	severityCritical = "critical"

	metricSkillContentTokens      = "skill-content-tokens"
	metricAgentContextTokens      = "agent-context-tokens"
	metricTeamMemberCount         = "team-member-count"
	metricTeamDistinctRoleCount   = "team-distinct-role-count"
	metricTeamRoleAssignedMembers = "team-members-with-role-count"
	metricActionContractValid     = "action-contract-valid"
	metricActionCommandDeclared   = "action-command-declared"
	metricActionExamples          = "action-examples"
	metricActionOwnerDeclared     = "action-owner-declared"
)

type scoreWithDiagnostics struct {
	value    float64
	messages []HealthMessage
}

type rankedMessage struct {
	message HealthMessage
	impact  float64
}

// ScoreAll computes health scores for all nodes using the provided score functions.
// Edge counts are pre-computed in O(E) to avoid quadratic per-node scanning.
func ScoreAll(g Graph, fns []ScoreFn) []HealthScore {
	ec := buildEdgeCounts(g)
	// Build optimized versions of the edge-based score functions.
	type scoreFnOptimized struct {
		Name   string
		Weight float64
		Fn     func(nodeID string) float64
	}
	optimized := make([]scoreFnOptimized, len(fns))
	for i, fn := range fns {
		fn := fn // capture
		switch fn.Name {
		case "outgoing-edges":
			optimized[i] = scoreFnOptimized{fn.Name, fn.Weight, func(nodeID string) float64 {
				return math.Min(float64(ec.outgoing[nodeID])/5.0, 1.0)
			}}
		case "incoming-edges":
			optimized[i] = scoreFnOptimized{fn.Name, fn.Weight, func(nodeID string) float64 {
				return math.Min(float64(ec.incoming[nodeID])/5.0, 1.0)
			}}
		default:
			optimized[i] = scoreFnOptimized{fn.Name, fn.Weight, func(nodeID string) float64 {
				return fn.Fn(nodeID, g)
			}}
		}
	}

	scores := make([]HealthScore, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		hs := HealthScore{
			NodeID:  n.ID,
			Factors: make(map[string]float64, len(fns)),
		}
		var total float64
		var weightSum float64
		for _, fn := range optimized {
			val := fn.Fn(n.ID)
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
// Edge counts are pre-computed in O(E) to avoid quadratic per-node scanning.
func ScoreAllWithConfig(g Graph, cfg HealthConfig) []HealthScore {
	ec := buildEdgeCounts(g)
	scores := make([]HealthScore, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		weights := weightsForNodeType(n.Type, cfg)
		hs := HealthScore{
			NodeID: n.ID,
			Factors: map[string]float64{
				"outgoing-edges":            0,
				"incoming-edges":            0,
				"code-usage":                0,
				"recent-activity":           0,
				"skill-content-length":      0,
				"agent-context-load":        0,
				"team-member-count-balance": 0,
				"team-role-coverage":        0,
				"action-contract":           0,
				"action-command":            0,
				"action-examples":           0,
				"action-owner":              0,
			},
		}
		score, factors, messages := scoreWithWeightsEC(n.ID, g, weights, ec)
		hs.Score = score
		for k, v := range factors {
			hs.Factors[k] = v
		}
		hs.Messages = messages
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
		if e.Kind != EdgeCodeUsage && e.Kind != EdgeActionCommand {
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
		if e.Category == CodeScenarioCLI || e.Kind == EdgeActionCommand {
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
			externalScore := normalizeScenarioScore(cfg.ExternalToolScore)
			scoreByNodeID[n.ID] = HealthScore{
				NodeID: n.ID,
				Score:  externalScore,
				Factors: map[string]float64{
					portabilityFactor: externalScore,
				},
				Messages: []HealthMessage{
					{
						Key:            "cli.external-tool",
						Severity:       severityWarning,
						Factor:         portabilityFactor,
						Summary:        "External CLI detected",
						Detail:         fmt.Sprintf("Command %q is treated as non-scenario tooling.", command),
						Recommendation: "Wrap this workflow in a scenario CLI to improve portability.",
						MetricValue:    externalScore,
						Target:         "Scenario CLI usage",
					},
				},
			}
			continue
		}

		// Scenario CLI: resolve scenario health via scenario-completeness-scoring.
		scenario := command
		score, ok := resolvedScenarioScores[scenario]
		if !ok {
			score = normalizeScenarioScore(cfg.ScenarioFallbackScore)
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
			Messages: []HealthMessage{
				{
					Key:            "cli.scenario-completeness",
					Severity:       severityInfo,
					Factor:         scenarioCompletenessFactor,
					Summary:        "Scenario CLI health applied",
					Detail:         fmt.Sprintf("CLI %q uses scenario completeness score.", scenario),
					Recommendation: "Improve scenario completeness to raise this CLI node score.",
					MetricValue:    score,
					Target:         ">= 0.80",
				},
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
	timeout          time.Duration
	resolveExec      func(context.Context) (string, error)
	resolveExecOnce  sync.Once
	resolvedExecPath string
	resolvedExecErr  error
}

func NewScenarioCompletenessCLIProvider(timeout time.Duration) *ScenarioCompletenessCLIProvider {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &ScenarioCompletenessCLIProvider{
		timeout: timeout,
		resolveExec: func(ctx context.Context) (string, error) {
			return scenariocli.ResolveExecutableFromRepoRootContext(ctx, "scenario-completeness-scoring")
		},
	}
}

func (p *ScenarioCompletenessCLIProvider) ScenarioScore(ctx context.Context, scenario string) (float64, error) {
	callCtx := ctx
	cancel := func() {}
	if p.timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, p.timeout)
	}
	defer cancel()

	executable, err := p.resolveExecutable(callCtx)
	if err != nil {
		return 0.0, err
	}

	cmd := exec.CommandContext(callCtx, executable, "score", scenario, "--json")
	cmd.Env = []string(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.ForeignScenario, nil))
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

func (p *ScenarioCompletenessCLIProvider) resolveExecutable(ctx context.Context) (string, error) {
	p.resolveExecOnce.Do(func() {
		if p.resolveExec == nil {
			p.resolvedExecErr = fmt.Errorf("scenario completeness CLI resolver is not configured")
			return
		}
		p.resolvedExecPath, p.resolvedExecErr = p.resolveExec(ctx)
	})
	return p.resolvedExecPath, p.resolvedExecErr
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
	case NodeAction:
		return cfg.Action
	default:
		// CLI scores are overridden by ApplyCLIHealthPolicyWithConfig, but
		// we still compute a baseline for consistency.
		return cfg.Skill
	}
}

func evaluateFactorWithMessages(factorName, nodeID string, g Graph, value float64) scoreWithDiagnostics {
	diag := scoreWithDiagnostics{value: value}
	nodeType := nodeTypeByID(g, nodeID)
	if nodeType == "" {
		return diag
	}

	switch factorName {
	case "outgoing-edges":
		if value < 0.2 {
			diag.messages = append(diag.messages, HealthMessage{
				Key:            "factor.outgoing-edges.low",
				Severity:       severityWarning,
				Factor:         factorName,
				Summary:        "Low outbound connectivity",
				Detail:         "This node has very few outgoing references.",
				Recommendation: "Add explicit links to dependent skills, agents, or tools.",
				MetricValue:    value,
				Target:         ">= 0.60",
			})
		}
	case "incoming-edges":
		if value < 0.2 {
			diag.messages = append(diag.messages, HealthMessage{
				Key:            "factor.incoming-edges.low",
				Severity:       severityWarning,
				Factor:         factorName,
				Summary:        "Low inbound discoverability",
				Detail:         "Few nodes reference this node.",
				Recommendation: "Cross-reference this node from related teams, agents, or skills.",
				MetricValue:    value,
				Target:         ">= 0.60",
			})
		}
	case "code-usage":
		if value <= 0.1 {
			diag.messages = append(diag.messages, HealthMessage{
				Key:            "factor.code-usage.external",
				Severity:       severityWarning,
				Factor:         factorName,
				Summary:        "External tooling dependency",
				Detail:         "Detected external CLI or script usage.",
				Recommendation: "Prefer Vrooli scenario CLIs for reproducibility and orchestration.",
				MetricValue:    value,
				Target:         "1.00",
			})
		}
	case "skill-content-length":
		if nodeType == NodeSkill {
			tokens := metricValue(g, nodeID, metricSkillContentTokens)
			if value < 0.5 {
				diag.messages = append(diag.messages, HealthMessage{
					Key:            "skill.content-length.high",
					Severity:       severityWarning,
					Factor:         factorName,
					Summary:        "Skill content is oversized",
					Detail:         fmt.Sprintf("Skill content is about %.0f tokens.", tokens),
					Recommendation: "Split the skill into smaller focused skills and keep task-critical instructions in the primary file.",
					MetricValue:    tokens,
					Target:         "150-1800 tokens",
				})
			}
		}
	case "agent-context-load":
		if nodeType == NodeAgent {
			tokens := metricValue(g, nodeID, metricAgentContextTokens)
			if value < 0.5 {
				diag.messages = append(diag.messages, HealthMessage{
					Key:            "agent.context-load.high",
					Severity:       severityWarning,
					Factor:         factorName,
					Summary:        "Agent context load is high",
					Detail:         fmt.Sprintf("Agent markdown payload is about %.0f tokens.", tokens),
					Recommendation: "Move reference-heavy content to shared docs and keep per-agent files concise.",
					MetricValue:    tokens,
					Target:         "<= 3500 tokens",
				})
			}
		}
	case "team-member-count-balance":
		if nodeType == NodeTeam {
			members := metricValue(g, nodeID, metricTeamMemberCount)
			if value < 0.5 {
				diag.messages = append(diag.messages, HealthMessage{
					Key:            "team.member-count.imbalanced",
					Severity:       severityWarning,
					Factor:         factorName,
					Summary:        "Team size is imbalanced",
					Detail:         fmt.Sprintf("Team currently has %.0f members.", members),
					Recommendation: "Aim for a balanced team size where collaboration and specialization are both practical.",
					MetricValue:    members,
					Target:         "3-8 members",
				})
			}
		}
	case "team-role-coverage":
		if nodeType == NodeTeam {
			distinctRoles := metricValue(g, nodeID, metricTeamDistinctRoleCount)
			withRole := metricValue(g, nodeID, metricTeamRoleAssignedMembers)
			members := math.Max(metricValue(g, nodeID, metricTeamMemberCount), 1)
			coverage := withRole / members
			if value < 0.5 {
				diag.messages = append(diag.messages, HealthMessage{
					Key:            "team.role-coverage.low",
					Severity:       severityWarning,
					Factor:         factorName,
					Summary:        "Role coverage is weak",
					Detail:         fmt.Sprintf("Distinct roles: %.0f, members with role assignments: %.0f (%.0f%%).", distinctRoles, withRole, coverage*100),
					Recommendation: "Define clearer role assignments and ensure each member has explicit role coverage.",
					MetricValue:    coverage,
					Target:         ">= 0.80 assignment coverage and >= 2 distinct roles",
				})
			}
		}
	case "action-contract":
		if nodeType == NodeAction && value < 1.0 {
			diag.messages = append(diag.messages, HealthMessage{
				Key:            "action.contract.invalid",
				Severity:       severityCritical,
				Factor:         factorName,
				Summary:        "Action contract did not validate",
				Detail:         "The Action graph node was present without a valid persisted contract marker.",
				Recommendation: "Run `prompt-manager action validate <id>` and fix the Action contract before using it.",
				MetricValue:    value,
				Target:         "1.00",
			})
		}
	case "action-command":
		if nodeType == NodeAction && value < 1.0 {
			diag.messages = append(diag.messages, HealthMessage{
				Key:            "action.command.missing",
				Severity:       severityCritical,
				Factor:         factorName,
				Summary:        "Action command is missing",
				Detail:         "Actions should wrap exactly one argv-shaped controlled CLI command.",
				Recommendation: "Declare a static command argv in action.json.",
				MetricValue:    value,
				Target:         "1.00",
			})
		}
	case "action-examples":
		if nodeType == NodeAction && value < 1.0 {
			diag.messages = append(diag.messages, HealthMessage{
				Key:            "action.examples.missing",
				Severity:       severityWarning,
				Factor:         factorName,
				Summary:        "Action examples are missing",
				Detail:         "Examples help agents call the Action with valid typed input.",
				Recommendation: "Add at least one example input payload to action.json.",
				MetricValue:    value,
				Target:         "1.00",
			})
		}
	case "action-owner":
		if nodeType == NodeAction && value < 1.0 {
			diag.messages = append(diag.messages, HealthMessage{
				Key:            "action.owner.missing",
				Severity:       severityWarning,
				Factor:         factorName,
				Summary:        "Action owner is missing",
				Detail:         "Actions need an explicit project, scenario, resource, team, or agent owner.",
				Recommendation: "Declare the owning Vrooli surface in action.json.",
				MetricValue:    value,
				Target:         "1.00",
			})
		}
	}

	return diag
}

// factorEvaluatorsEC are O(1) versions of edge-based evaluators that use
// pre-computed edge counts instead of scanning all edges per node.
var factorEvaluatorsEC = map[string]func(nodeID string, g Graph, ec edgeCounts) float64{
	"outgoing-edges": func(nodeID string, _ Graph, ec edgeCounts) float64 {
		return math.Min(float64(ec.outgoing[nodeID])/5.0, 1.0)
	},
	"incoming-edges": func(nodeID string, _ Graph, ec edgeCounts) float64 {
		return math.Min(float64(ec.incoming[nodeID])/5.0, 1.0)
	},
}

// scoreWithWeightsEC is the optimized variant that uses pre-computed edge counts.
func scoreWithWeightsEC(nodeID string, g Graph, weights HealthWeights, ec edgeCounts) (float64, map[string]float64, []HealthMessage) {
	weightByFactor := map[string]float64{
		"outgoing-edges":            weights.OutgoingEdges,
		"incoming-edges":            weights.IncomingEdges,
		"code-usage":                weights.CodeUsage,
		"recent-activity":           weights.RecentActivity,
		"skill-content-length":      weights.SkillContentLength,
		"agent-context-load":        weights.AgentContextLoad,
		"team-member-count-balance": weights.TeamMemberCountBalance,
		"team-role-coverage":        weights.TeamRoleCoverage,
		"action-contract":           weights.ActionContract,
		"action-command":            weights.ActionCommand,
		"action-examples":           weights.ActionExamples,
		"action-owner":              weights.ActionOwner,
	}

	factors := make(map[string]float64, len(weightByFactor))
	var ranked []rankedMessage
	var total float64
	var weightSum float64
	factorNames := make([]string, 0, len(weightByFactor))
	for factorName := range weightByFactor {
		factorNames = append(factorNames, factorName)
	}
	sort.Strings(factorNames)
	for _, factorName := range factorNames {
		weight := weightByFactor[factorName]
		// Use O(1) edge-count evaluator when available, else fall back to O(E) scan.
		var val float64
		if ecEval, ok := factorEvaluatorsEC[factorName]; ok {
			val = ecEval(nodeID, g, ec)
		} else if evaluator, ok := factorEvaluators[factorName]; ok {
			val = evaluator(nodeID, g)
		}
		result := evaluateFactorWithMessages(factorName, nodeID, g, val)
		factors[factorName] = result.value
		if weight <= 0 {
			continue
		}
		if len(result.messages) > 0 {
			impact := weight * (1 - result.value)
			for _, msg := range result.messages {
				ranked = append(ranked, rankedMessage{
					message: msg,
					impact:  impact,
				})
			}
		}
		total += result.value * weight
		weightSum += weight
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].impact != ranked[j].impact {
			return ranked[i].impact > ranked[j].impact
		}
		left := severityRank(ranked[i].message.Severity)
		right := severityRank(ranked[j].message.Severity)
		if left != right {
			return left > right
		}
		return ranked[i].message.Factor < ranked[j].message.Factor
	})
	messages := make([]HealthMessage, 0, len(ranked))
	for _, r := range ranked {
		messages = append(messages, r.message)
	}
	if weightSum <= 0 {
		return 0, factors, messages
	}
	return total / weightSum, factors, messages
}

func scoreWithWeights(nodeID string, g Graph, weights HealthWeights) (float64, map[string]float64, []HealthMessage) {
	weightByFactor := map[string]float64{
		"outgoing-edges":            weights.OutgoingEdges,
		"incoming-edges":            weights.IncomingEdges,
		"code-usage":                weights.CodeUsage,
		"recent-activity":           weights.RecentActivity,
		"skill-content-length":      weights.SkillContentLength,
		"agent-context-load":        weights.AgentContextLoad,
		"team-member-count-balance": weights.TeamMemberCountBalance,
		"team-role-coverage":        weights.TeamRoleCoverage,
		"action-contract":           weights.ActionContract,
		"action-command":            weights.ActionCommand,
		"action-examples":           weights.ActionExamples,
		"action-owner":              weights.ActionOwner,
	}

	factors := make(map[string]float64, len(weightByFactor))
	var ranked []rankedMessage
	var total float64
	var weightSum float64
	factorNames := make([]string, 0, len(weightByFactor))
	for factorName := range weightByFactor {
		factorNames = append(factorNames, factorName)
	}
	sort.Strings(factorNames)
	for _, factorName := range factorNames {
		weight := weightByFactor[factorName]
		evaluator, ok := factorEvaluators[factorName]
		if !ok {
			factors[factorName] = 0
			continue
		}
		result := evaluateFactorWithMessages(factorName, nodeID, g, evaluator(nodeID, g))
		factors[factorName] = result.value
		if weight <= 0 {
			continue
		}
		if len(result.messages) > 0 {
			impact := weight * (1 - result.value)
			for _, msg := range result.messages {
				ranked = append(ranked, rankedMessage{
					message: msg,
					impact:  impact,
				})
			}
		}
		total += result.value * weight
		weightSum += weight
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].impact != ranked[j].impact {
			return ranked[i].impact > ranked[j].impact
		}
		left := severityRank(ranked[i].message.Severity)
		right := severityRank(ranked[j].message.Severity)
		if left != right {
			return left > right
		}
		return ranked[i].message.Factor < ranked[j].message.Factor
	})
	messages := make([]HealthMessage, 0, len(ranked))
	for _, r := range ranked {
		messages = append(messages, r.message)
	}
	if weightSum <= 0 {
		return 0, factors, messages
	}
	return total / weightSum, factors, messages
}

func severityRank(severity string) int {
	switch severity {
	case severityCritical:
		return 3
	case severityWarning:
		return 2
	default:
		return 1
	}
}

func normalizeScenarioScore(score float64) float64 {
	// A failed or incomplete external score must never leak NaN/Inf into the
	// graph contract. Proto/JSON encoders either omit those values or emit a
	// non-portable representation, which makes the entire health response
	// unusable to strict clients.
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0.0
	}
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

// edgeCounts pre-computes outgoing and incoming edge counts for all nodes
// in O(E) time, avoiding O(N*E) per-node scans.
type edgeCounts struct {
	outgoing map[string]int
	incoming map[string]int
}

func buildEdgeCounts(g Graph) edgeCounts {
	ec := edgeCounts{
		outgoing: make(map[string]int, len(g.Nodes)),
		incoming: make(map[string]int, len(g.Nodes)),
	}
	for _, e := range g.Edges {
		ec.outgoing[e.From]++
		ec.incoming[e.To]++
	}
	return ec
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

func nodeTypeByID(g Graph, nodeID string) NodeType {
	for _, n := range g.Nodes {
		if n.ID == nodeID {
			return n.Type
		}
	}
	return ""
}

func metricValue(g Graph, nodeID, key string) float64 {
	if g.NodeMetrics == nil {
		return 0
	}
	nodeMetrics, ok := g.NodeMetrics[nodeID]
	if !ok {
		return 0
	}
	return nodeMetrics[key]
}

func skillContentLengthScore(nodeID string, g Graph) float64 {
	if nodeTypeByID(g, nodeID) != NodeSkill {
		return 0.5
	}
	tokens := metricValue(g, nodeID, metricSkillContentTokens)
	if tokens <= 0 {
		return 0.5
	}
	// 150-1800 tokens is ideal; penalties ramp up outside this range.
	if tokens < 150 {
		return 0.5 + (tokens/150.0)*0.5
	}
	if tokens <= 1800 {
		return 1.0
	}
	if tokens >= 4000 {
		return 0.0
	}
	return 1.0 - ((tokens - 1800) / (4000 - 1800))
}

func agentContextLoadScore(nodeID string, g Graph) float64 {
	if nodeTypeByID(g, nodeID) != NodeAgent {
		return 0.5
	}
	tokens := metricValue(g, nodeID, metricAgentContextTokens)
	if tokens <= 0 {
		return 0.5
	}
	// <=3500 tokens is ideal, with decay to 0 at 12000 tokens.
	if tokens <= 3500 {
		return 1.0
	}
	if tokens >= 12000 {
		return 0.0
	}
	return 1.0 - ((tokens - 3500) / (12000 - 3500))
}

func teamMemberCountBalanceScore(nodeID string, g Graph) float64 {
	if nodeTypeByID(g, nodeID) != NodeTeam {
		return 0.5
	}
	members := metricValue(g, nodeID, metricTeamMemberCount)
	if members <= 0 {
		return 0.0
	}
	switch {
	case members == 1:
		return 0.25
	case members == 2:
		return 0.60
	case members >= 3 && members <= 8:
		return 1.0
	case members <= 14:
		return 1.0 - ((members-8)/(14-8))*(1.0-0.4)
	case members >= 20:
		return 0.0
	default:
		return 0.4 - ((members-14)/(20-14))*(0.4-0.0)
	}
}

func teamRoleCoverageScore(nodeID string, g Graph) float64 {
	if nodeTypeByID(g, nodeID) != NodeTeam {
		return 0.5
	}
	members := metricValue(g, nodeID, metricTeamMemberCount)
	if members <= 0 {
		return 0.0
	}
	distinctRoles := metricValue(g, nodeID, metricTeamDistinctRoleCount)
	membersWithRole := metricValue(g, nodeID, metricTeamRoleAssignedMembers)

	var roleVarietyScore float64
	switch {
	case distinctRoles <= 0:
		roleVarietyScore = 0.0
	case distinctRoles == 1:
		roleVarietyScore = 0.3
	case distinctRoles == 2:
		roleVarietyScore = 0.7
	case distinctRoles <= 6:
		roleVarietyScore = 1.0
	case distinctRoles >= 12:
		roleVarietyScore = 0.5
	default:
		roleVarietyScore = 1.0 - ((distinctRoles-6)/(12-6))*(1.0-0.5)
	}

	assignCoverage := membersWithRole / members
	if assignCoverage < 0 {
		assignCoverage = 0
	}
	if assignCoverage > 1 {
		assignCoverage = 1
	}

	return (roleVarietyScore * 0.5) + (assignCoverage * 0.5)
}

func actionContractScore(nodeID string, g Graph) float64 {
	if nodeTypeByID(g, nodeID) != NodeAction {
		return 0.5
	}
	return binaryMetricScore(g, nodeID, metricActionContractValid)
}

func actionCommandScore(nodeID string, g Graph) float64 {
	if nodeTypeByID(g, nodeID) != NodeAction {
		return 0.5
	}
	if binaryMetricScore(g, nodeID, metricActionCommandDeclared) == 1 {
		return 1
	}
	for _, e := range g.Edges {
		if e.From == nodeID && e.Kind == EdgeActionCommand {
			return 1
		}
	}
	return 0
}

func actionExamplesScore(nodeID string, g Graph) float64 {
	if nodeTypeByID(g, nodeID) != NodeAction {
		return 0.5
	}
	return binaryMetricScore(g, nodeID, metricActionExamples)
}

func actionOwnerScore(nodeID string, g Graph) float64 {
	if nodeTypeByID(g, nodeID) != NodeAction {
		return 0.5
	}
	return binaryMetricScore(g, nodeID, metricActionOwnerDeclared)
}

func binaryMetricScore(g Graph, nodeID, key string) float64 {
	if metricValue(g, nodeID, key) > 0 {
		return 1
	}
	return 0
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
