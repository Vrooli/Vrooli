package memberflow

import "fmt"

type graphDecisionsTableMissingRule struct{}

func (r graphDecisionsTableMissingRule) ID() string { return "graph_decisions_table_missing" }
func (r graphDecisionsTableMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}
func (r graphDecisionsTableMissingRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphDecisionsTableMissingRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphDecisionsTableMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	if ctx.Block.Docs.Decisions.Present {
		return nil
	}
	builder := NewOperatingFindingBuilder(ctx, r)
	return []OperatingGraphFinding{builder.base(ctx.Block.Source.Path, ctx.Block.Source.FenceLine, "contract graph source is missing a ## Decisions table")}
}

type graphDecisionsTableDriftRule struct{}

func (r graphDecisionsTableDriftRule) ID() string { return "graph_decisions_table_drift" }
func (r graphDecisionsTableDriftRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}
func (r graphDecisionsTableDriftRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphDecisionsTableDriftRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphDecisionsTableDriftRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	if !ctx.Block.Docs.Decisions.Present {
		return nil
	}
	builder := NewOperatingFindingBuilder(ctx, r)
	rows := map[string]OperatingDecisionRow{}
	for _, row := range ctx.Block.Docs.Decisions.Rows {
		if row.Decision != "" {
			rows[row.Decision] = row
		}
	}
	graphDecisions := map[string]OperatingGraphNode{}
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind == "decision" {
			graphDecisions[node.Value] = node
		}
	}
	var findings []OperatingGraphFinding
	for decision, node := range graphDecisions {
		if _, ok := rows[decision]; ok {
			continue
		}
		f := builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("graph decision %q is missing from the Decisions table", decision))
		f.Decision = decision
		findings = append(findings, f)
	}
	for decision, row := range rows {
		if _, ok := graphDecisions[decision]; ok {
			continue
		}
		f := builder.base(ctx.Block.Source.Path, row.SourceLine, fmt.Sprintf("Decisions table row %q is missing from the contract graph", decision))
		f.Decision = decision
		findings = append(findings, f)
	}
	return findings
}

type graphDecisionsTableOwnerDriftRule struct{}

func (r graphDecisionsTableOwnerDriftRule) ID() string {
	return "graph_decisions_table_owner_drift"
}

func (r graphDecisionsTableOwnerDriftRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}
func (r graphDecisionsTableOwnerDriftRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphDecisionsTableOwnerDriftRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphDecisionsTableOwnerDriftRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	resolver := NewOperatingActorResolver(ctx.Block.Metadata)
	var findings []OperatingGraphFinding
	for _, row := range ctx.Block.Docs.Decisions.Rows {
		for _, ref := range row.Owners {
			for _, expanded := range resolver.Expand(ctx.Block.Metadata.Team, ctx.Runtime, ref) {
				if expanded.Kind != OperatingActorKindMember {
					continue
				}
				rel := OperatingRelationship{
					Kind:     operatingRelDecisionOwned,
					Team:     ctx.Block.Metadata.Team,
					Member:   expanded.Value,
					Decision: row.Decision,
				}
				if ctx.Index.GraphHasRelationship(rel) || graphHasCapabilityGapOwner(ctx, rel) {
					continue
				}
				f := builder.base(ctx.Block.Source.Path, row.SourceLine, fmt.Sprintf("Decisions table owner %q is not shown as an owner of decision %q in the contract graph", expanded.Value, row.Decision))
				f.Member = expanded.Value
				f.Decision = row.Decision
				findings = append(findings, f)
			}
		}
	}
	return findings
}

func graphHasCapabilityGapOwner(ctx OperatingGraphRuleContext, rel OperatingRelationship) bool {
	if rel.Decision != "capability-gap" {
		return false
	}
	rel.Kind = operatingRelCapabilityGapRaised
	return ctx.Index.GraphHasRelationship(rel)
}
