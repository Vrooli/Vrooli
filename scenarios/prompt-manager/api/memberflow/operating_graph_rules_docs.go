package memberflow

import "fmt"

type graphTopicCatalogMissingRule struct{}

func (r graphTopicCatalogMissingRule) ID() string { return "graph_topic_catalog_missing" }
func (r graphTopicCatalogMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}
func (r graphTopicCatalogMissingRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphTopicCatalogMissingRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphTopicCatalogMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	if ctx.Block.Docs.TopicCatalog.Present {
		return nil
	}
	builder := NewOperatingFindingBuilder(ctx, r)
	return []OperatingGraphFinding{builder.base(ctx.Block.Source.Path, ctx.Block.Source.FenceLine, "contract graph source is missing a ## Topic Catalog table")}
}

type graphTopicCatalogInvalidTopicRule struct{}

func (r graphTopicCatalogInvalidTopicRule) ID() string { return "graph_topic_catalog_invalid_topic" }
func (r graphTopicCatalogInvalidTopicRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}
func (r graphTopicCatalogInvalidTopicRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphTopicCatalogInvalidTopicRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphTopicCatalogInvalidTopicRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, row := range ctx.Block.Docs.TopicCatalog.Rows {
		if row.Topic != "" {
			continue
		}
		f := builder.base(ctx.Block.Source.Path, row.SourceLine, fmt.Sprintf("topic catalog row uses invalid topic token %q", row.RawTopic))
		findings = append(findings, f)
	}
	return findings
}

type graphTopicCatalogDriftRule struct{}

func (r graphTopicCatalogDriftRule) ID() string                     { return "graph_topic_catalog_drift" }
func (r graphTopicCatalogDriftRule) Group() OperatingGraphRuleGroup { return OperatingRuleGroupDocs }
func (r graphTopicCatalogDriftRule) DefaultSeverity() Severity      { return SeverityError }
func (r graphTopicCatalogDriftRule) AppliesTo(mode string) bool     { return mode == "contract" }
func (r graphTopicCatalogDriftRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	if !ctx.Block.Docs.TopicCatalog.Present {
		return nil
	}
	builder := NewOperatingFindingBuilder(ctx, r)
	rows := map[string]OperatingTopicCatalogRow{}
	for _, row := range ctx.Block.Docs.TopicCatalog.Rows {
		if row.Topic != "" {
			rows[qualifiedTopicKey(row.Qualifier, row.Topic)] = row
		}
	}
	graphTopics := map[string]OperatingGraphNode{}
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind != "topic" || node.Qualifier == "old" || node.Qualifier == "external" {
			continue
		}
		graphTopics[qualifiedTopicKey(node.Qualifier, node.Value)] = node
	}
	var findings []OperatingGraphFinding
	for key, node := range graphTopics {
		if _, ok := rows[key]; ok {
			continue
		}
		f := builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("graph topic %q is missing from the Topic Catalog table", displayQualifiedTopic(node.Qualifier, node.Value)))
		f.Topic = node.Value
		findings = append(findings, f)
	}
	for key, row := range rows {
		if _, ok := graphTopics[key]; ok {
			continue
		}
		f := builder.base(ctx.Block.Source.Path, row.SourceLine, fmt.Sprintf("Topic Catalog row %q is missing from the contract graph", displayQualifiedTopic(row.Qualifier, row.Topic)))
		f.Topic = row.Topic
		findings = append(findings, f)
	}
	return findings
}

type graphDocsUnknownActorRule struct{}

func (r graphDocsUnknownActorRule) ID() string                     { return "graph_docs_unknown_actor" }
func (r graphDocsUnknownActorRule) Group() OperatingGraphRuleGroup { return OperatingRuleGroupDocs }
func (r graphDocsUnknownActorRule) DefaultSeverity() Severity      { return SeverityError }
func (r graphDocsUnknownActorRule) AppliesTo(mode string) bool     { return mode == "contract" }
func (r graphDocsUnknownActorRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, row := range ctx.Block.Docs.TopicCatalog.Rows {
		for _, ref := range append(append([]OperatingActorReference{}, row.Writers...), row.Readers...) {
			findings = append(findings, validateOperatingActorRef(ctx, builder, ref, row.SourceLine)...)
		}
	}
	for _, row := range ctx.Block.Docs.Decisions.Rows {
		for _, ref := range row.Owners {
			findings = append(findings, validateOperatingActorRef(ctx, builder, ref, row.SourceLine)...)
		}
	}
	return findings
}

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
	var findings []OperatingGraphFinding
	for _, row := range ctx.Block.Docs.Decisions.Rows {
		for _, ref := range row.Owners {
			for _, expanded := range expandOperatingActorReference(ref) {
				if expanded.Kind != "member" {
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

func validateOperatingActorRef(ctx OperatingGraphRuleContext, builder OperatingFindingBuilder, ref OperatingActorReference, line int) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	for _, expanded := range expandOperatingActorReference(ref) {
		switch expanded.Kind {
		case "member":
			contract := ctx.Runtime.Contracts[ctx.Block.Metadata.Team]
			if contract == nil || contract.Contract == nil {
				f := builder.base(ctx.Block.Source.Path, line, fmt.Sprintf("actor reference %q cannot be resolved because team contract is unavailable", ref.Raw))
				f.Member = expanded.Value
				findings = append(findings, f)
				continue
			}
			if _, ok := contract.Contract.Members[expanded.Value]; !ok {
				f := builder.base(ctx.Block.Source.Path, line, fmt.Sprintf("actor reference %q resolves to unknown member %q", ref.Raw, expanded.Value))
				f.Member = expanded.Value
				findings = append(findings, f)
			}
		case "team":
			if _, ok := ctx.Runtime.Contracts[expanded.Value]; !ok {
				f := builder.base(ctx.Block.Source.Path, line, fmt.Sprintf("actor reference %q resolves to unknown team %q", ref.Raw, expanded.Value))
				findings = append(findings, f)
			}
		case "unknown":
			f := builder.base(ctx.Block.Source.Path, line, fmt.Sprintf("actor reference %q is not recognized; use a typed actor such as member:researcher, external:operator, team:monetization, or a supported group", ref.Raw))
			findings = append(findings, f)
		}
	}
	return findings
}

func qualifiedTopicKey(qualifier, topic string) string {
	return qualifier + "\x00" + topic
}

func displayQualifiedTopic(qualifier, topic string) string {
	if qualifier == "" {
		return "topic:" + topic
	}
	return "topic[" + qualifier + "]:" + topic
}
