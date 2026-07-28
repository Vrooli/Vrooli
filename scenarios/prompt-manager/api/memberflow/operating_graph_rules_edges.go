package memberflow

import "fmt"

type graphFutureTopicLiveEdgeRule struct{}

func (r graphFutureTopicLiveEdgeRule) ID() string { return "graph_future_topic_live_edge" }
func (r graphFutureTopicLiveEdgeRule) Group() RuleGroup {
	return OperatingRuleGroupEdgeTruth
}
func (r graphFutureTopicLiveEdgeRule) DefaultSeverity() Severity { return SeverityWarning }
func (r graphFutureTopicLiveEdgeRule) AppliesTo(ctx RuleContext) bool {
	return string(ctx.Block.Metadata.Mode) != "explanatory"
}

func (r graphFutureTopicLiveEdgeRule) Check(ctx RuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, edge := range ctx.Block.Graph.Edges {
		from, fok := ctx.Index.NodesByID[edge.From]
		to, tok := ctx.Index.NodesByID[edge.To]
		if !fok || !tok || from.Kind == "" || to.Kind == "" {
			continue
		}
		if from.Kind == "process" || to.Kind == "process" || from.Kind == "future" || to.Kind == "future" {
			continue
		}
		if from.Kind == "topic" && from.Qualifier == "future" {
			f := builder.WithEdge(ctx.Block.Source.Path, edge, fmt.Sprintf("future topic %q is used as an active edge source", from.Value))
			f.Topic = from.Value
			findings = append(findings, f)
		}
		if to.Kind == "topic" && to.Qualifier == "future" {
			f := builder.WithEdge(ctx.Block.Source.Path, edge, fmt.Sprintf("future topic %q is used as an active edge target", to.Value))
			f.Topic = to.Value
			findings = append(findings, f)
		}
	}
	return findings
}

type graphUnsupportedEdgeSemanticsRule struct{}

func (r graphUnsupportedEdgeSemanticsRule) ID() string { return "graph_unsupported_edge_semantics" }
func (r graphUnsupportedEdgeSemanticsRule) Group() RuleGroup {
	return OperatingRuleGroupEdgeTruth
}
func (r graphUnsupportedEdgeSemanticsRule) DefaultSeverity() Severity { return SeverityError }
func (r graphUnsupportedEdgeSemanticsRule) AppliesTo(ctx RuleContext) bool {
	return string(ctx.Block.Metadata.Mode) != "explanatory"
}

func (r graphUnsupportedEdgeSemanticsRule) Check(ctx RuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, edge := range ctx.Block.Graph.Edges {
		from, fok := ctx.Index.NodesByID[edge.From]
		to, tok := ctx.Index.NodesByID[edge.To]
		if !fok || !tok || !operatingGraphEdgeActionable(from, to) {
			continue
		}
		if _, ok := DefaultOperatingRelationshipRegistry().RelationshipFromEdge(ctx.Block.Metadata.Team, OperatingSourceRef{Path: ctx.Block.Source.Path, Line: edge.SourceLine}, from, to); ok {
			continue
		}
		findings = append(findings, builder.WithEdge(ctx.Block.Source.Path, edge, fmt.Sprintf("edge %s:%s -> %s:%s does not map to a supported operating relationship", from.Kind, from.Value, to.Kind, to.Value)))
	}
	return findings
}

type graphEdgeUnbackedRule struct{}

func (r graphEdgeUnbackedRule) ID() string                { return "graph_edge_unbacked" }
func (r graphEdgeUnbackedRule) Group() RuleGroup          { return OperatingRuleGroupEdgeTruth }
func (r graphEdgeUnbackedRule) DefaultSeverity() Severity { return SeverityError }
func (r graphEdgeUnbackedRule) AppliesTo(ctx RuleContext) bool {
	return string(ctx.Block.Metadata.Mode) != "explanatory"
}

func (r graphEdgeUnbackedRule) Check(ctx RuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, edge := range ctx.Block.Graph.Edges {
		from, fok := ctx.Index.NodesByID[edge.From]
		to, tok := ctx.Index.NodesByID[edge.To]
		if !fok || !tok || from.Kind == "" || to.Kind == "" {
			continue
		}
		if !operatingGraphEdgeActionable(from, to) {
			continue
		}
		rel, ok := ctx.Index.RelationshipForEdge(edge)
		if !ok || ctx.Matcher.GraphBackedByRuntime(rel, ctx.Index.RuntimeRelationships) {
			continue
		}
		findings = append(findings, builder.WithEdge(ctx.Block.Source.Path, edge, fmt.Sprintf("edge %s:%s -> %s:%s is not backed by runtime declarations", from.Kind, from.Value, to.Kind, to.Value)))
	}
	return findings
}
