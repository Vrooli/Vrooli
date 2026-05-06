package memberflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type graphUntypedNodeRule struct{}

func (r graphUntypedNodeRule) ID() string                     { return "graph_untyped_node" }
func (r graphUntypedNodeRule) Group() OperatingGraphRuleGroup { return OperatingRuleGroupEntity }
func (r graphUntypedNodeRule) DefaultSeverity() Severity      { return SeverityError }
func (r graphUntypedNodeRule) AppliesTo(mode string) bool     { return mode != "explanatory" }
func (r graphUntypedNodeRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind == "" {
			findings = append(findings, builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("node %q lacks a typed machine label", node.ID)))
		}
	}
	return findings
}

type graphUnknownNodeKindRule struct{}

func (r graphUnknownNodeKindRule) ID() string                     { return "graph_unknown_node_kind" }
func (r graphUnknownNodeKindRule) Group() OperatingGraphRuleGroup { return OperatingRuleGroupEntity }
func (r graphUnknownNodeKindRule) DefaultSeverity() Severity      { return SeverityError }
func (r graphUnknownNodeKindRule) AppliesTo(mode string) bool     { return mode != "explanatory" }
func (r graphUnknownNodeKindRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind == "" {
			continue
		}
		switch node.Kind {
		case "member", "decision", "team", "por", "topic", "external", "process", "future":
		default:
			findings = append(findings, builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("node kind %q is not supported", node.Kind)))
		}
	}
	return findings
}

type graphUnknownMemberRule struct{}

func (r graphUnknownMemberRule) ID() string                     { return "graph_unknown_member" }
func (r graphUnknownMemberRule) Group() OperatingGraphRuleGroup { return OperatingRuleGroupEntity }
func (r graphUnknownMemberRule) DefaultSeverity() Severity      { return SeverityError }
func (r graphUnknownMemberRule) AppliesTo(mode string) bool     { return mode != "explanatory" }
func (r graphUnknownMemberRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	contract := ctx.Runtime.Contracts[ctx.Block.Metadata.Team]
	var findings []OperatingGraphFinding
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind != "member" {
			continue
		}
		if contract == nil || contract.Contract == nil {
			findings = append(findings, builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("member %q cannot be resolved because team contract is unavailable", node.Value)))
			continue
		}
		if _, ok := contract.Contract.Members[node.Value]; !ok {
			findings = append(findings, builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("member %q is not declared in %s/team.json", node.Value, ctx.Block.Metadata.Team)))
		}
	}
	return findings
}

type graphUnknownDecisionRule struct{}

func (r graphUnknownDecisionRule) ID() string                     { return "graph_unknown_decision" }
func (r graphUnknownDecisionRule) Group() OperatingGraphRuleGroup { return OperatingRuleGroupEntity }
func (r graphUnknownDecisionRule) DefaultSeverity() Severity      { return SeverityError }
func (r graphUnknownDecisionRule) AppliesTo(mode string) bool     { return mode != "explanatory" }
func (r graphUnknownDecisionRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind != "decision" {
			continue
		}
		if ctx.Runtime.Contracts.HasTeamDecisionContext(ctx.Block.Metadata.Team, node.Value) {
			continue
		}
		teams := ctx.Runtime.Contracts.TeamsForDecisionContext(node.Value)
		detail := fmt.Sprintf("decision context %q is not declared in %s/team.json", node.Value, ctx.Block.Metadata.Team)
		if len(teams) > 0 {
			detail = fmt.Sprintf("decision context %q is declared by %s, but team-scoped graph %q must declare it in %s/team.json", node.Value, strings.Join(teams, ", "), ctx.Block.Metadata.ID, ctx.Block.Metadata.Team)
		}
		findings = append(findings, builder.WithNode(ctx.Block.Source.Path, node, detail))
	}
	return findings
}

type graphUnknownTeamRule struct{}

func (r graphUnknownTeamRule) ID() string                     { return "graph_unknown_team" }
func (r graphUnknownTeamRule) Group() OperatingGraphRuleGroup { return OperatingRuleGroupEntity }
func (r graphUnknownTeamRule) DefaultSeverity() Severity      { return SeverityWarning }
func (r graphUnknownTeamRule) AppliesTo(mode string) bool     { return mode != "explanatory" }
func (r graphUnknownTeamRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind == "team" {
			if _, ok := ctx.Runtime.Contracts[node.Value]; !ok {
				findings = append(findings, builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("team %q is not declared in the team registry", node.Value)))
			}
		}
	}
	return findings
}

type graphUnknownPORRule struct{}

func (r graphUnknownPORRule) ID() string                     { return "graph_unknown_por" }
func (r graphUnknownPORRule) Group() OperatingGraphRuleGroup { return OperatingRuleGroupEntity }
func (r graphUnknownPORRule) DefaultSeverity() Severity      { return SeverityError }
func (r graphUnknownPORRule) AppliesTo(mode string) bool     { return mode != "explanatory" }
func (r graphUnknownPORRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind == "por" && (node.Value == "" || !operatingGraphFileExists(filepath.Join(ctx.Runtime.RepoRoot, node.Value))) {
			findings = append(findings, builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("plan-of-record path %q does not exist", node.Value)))
		}
	}
	return findings
}

type graphTopicUnresolvedRule struct{}

func (r graphTopicUnresolvedRule) ID() string                     { return "graph_topic_unresolved" }
func (r graphTopicUnresolvedRule) Group() OperatingGraphRuleGroup { return OperatingRuleGroupEntity }
func (r graphTopicUnresolvedRule) DefaultSeverity() Severity      { return SeverityError }
func (r graphTopicUnresolvedRule) AppliesTo(mode string) bool     { return mode != "explanatory" }
func (r graphTopicUnresolvedRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind != "topic" || node.Qualifier == "future" || node.Qualifier == "old" || node.Qualifier == "external" {
			continue
		}
		if !ctx.Runtime.topicDeclared(ctx.Block.Metadata.Team, node.Value) {
			findings = append(findings, builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("live topic %q is not declared by any %s member topics.json", node.Value, ctx.Block.Metadata.Team)))
		}
	}
	return findings
}

func operatingGraphFileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
