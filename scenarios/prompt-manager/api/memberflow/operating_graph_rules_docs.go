package memberflow

import (
	"fmt"
	"strings"
)

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
		if node.Kind != OperatingGraphNodeKindTopic || node.Qualifier == OperatingGraphQualifierOld || node.Qualifier == OperatingGraphQualifierExternal {
			continue
		}
		graphTopics[qualifiedTopicKey(string(node.Qualifier), node.Value)] = node
	}
	var findings []OperatingGraphFinding
	for key, node := range graphTopics {
		if _, ok := rows[key]; ok {
			continue
		}
		f := builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("graph topic %q is missing from the Topic Catalog table", displayQualifiedTopic(string(node.Qualifier), node.Value)))
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

type graphTopicCatalogUnknownStatusRule struct{}

func (r graphTopicCatalogUnknownStatusRule) ID() string { return "graph_topic_catalog_unknown_status" }
func (r graphTopicCatalogUnknownStatusRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}
func (r graphTopicCatalogUnknownStatusRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphTopicCatalogUnknownStatusRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphTopicCatalogUnknownStatusRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, row := range ctx.Block.Docs.TopicCatalog.Rows {
		if row.StatusKind != OperatingTopicStatusUnknown {
			continue
		}
		f := builder.base(ctx.Block.Source.Path, row.SourceLine, fmt.Sprintf("Topic Catalog row %q uses unknown status %q", displayQualifiedTopic(row.Qualifier, row.Topic), row.Status))
		f.Topic = row.Topic
		findings = append(findings, f)
	}
	return findings
}

type graphTopicCatalogStatusQualifierDriftRule struct{}

func (r graphTopicCatalogStatusQualifierDriftRule) ID() string {
	return "graph_topic_catalog_status_qualifier_drift"
}

func (r graphTopicCatalogStatusQualifierDriftRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}

func (r graphTopicCatalogStatusQualifierDriftRule) DefaultSeverity() Severity {
	return SeverityError
}

func (r graphTopicCatalogStatusQualifierDriftRule) AppliesTo(mode string) bool {
	return mode == "contract"
}

func (r graphTopicCatalogStatusQualifierDriftRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, row := range ctx.Block.Docs.TopicCatalog.Rows {
		wantQualifier, ok := expectedTopicCatalogQualifier(row.StatusKind)
		if !ok || row.Topic == "" || row.Qualifier == wantQualifier {
			continue
		}
		f := builder.base(ctx.Block.Source.Path, row.SourceLine, fmt.Sprintf("Topic Catalog status %q expects %s, got %s", row.Status, displayQualifiedTopic(wantQualifier, row.Topic), displayQualifiedTopic(row.Qualifier, row.Topic)))
		f.Topic = row.Topic
		findings = append(findings, f)
	}
	return findings
}

type graphTopicCatalogLiveStatusUnbackedRule struct{}

func (r graphTopicCatalogLiveStatusUnbackedRule) ID() string {
	return "graph_topic_catalog_live_status_unbacked"
}

func (r graphTopicCatalogLiveStatusUnbackedRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}
func (r graphTopicCatalogLiveStatusUnbackedRule) DefaultSeverity() Severity { return SeverityError }
func (r graphTopicCatalogLiveStatusUnbackedRule) AppliesTo(mode string) bool {
	return mode == "contract"
}

func (r graphTopicCatalogLiveStatusUnbackedRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, row := range ctx.Block.Docs.TopicCatalog.Rows {
		if row.Topic == "" || !operatingTopicCatalogStatusIsCurrent(row.StatusKind) || catalogGraphTopicExists(ctx.Block, row.Qualifier, row.Topic) {
			continue
		}
		f := builder.base(ctx.Block.Source.Path, row.SourceLine, fmt.Sprintf("Topic Catalog row %q is marked %q but has no matching live graph topic node", displayQualifiedTopic(row.Qualifier, row.Topic), row.Status))
		f.Topic = row.Topic
		findings = append(findings, f)
	}
	return findings
}

type graphTopicCatalogTransitionalWithoutTargetRule struct{}

func (r graphTopicCatalogTransitionalWithoutTargetRule) ID() string {
	return "graph_topic_catalog_transitional_without_target"
}

func (r graphTopicCatalogTransitionalWithoutTargetRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}

func (r graphTopicCatalogTransitionalWithoutTargetRule) DefaultSeverity() Severity {
	return SeverityWarning
}

func (r graphTopicCatalogTransitionalWithoutTargetRule) AppliesTo(mode string) bool {
	return mode == "contract"
}

func (r graphTopicCatalogTransitionalWithoutTargetRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, row := range ctx.Block.Docs.TopicCatalog.Rows {
		if row.StatusKind != OperatingTopicStatusLiveTransitional || catalogRowReferencesFutureTopic(ctx.Block, row) {
			continue
		}
		f := builder.base(ctx.Block.Source.Path, row.SourceLine, fmt.Sprintf("Topic Catalog row %q is live transitional but does not reference a future replacement topic", displayQualifiedTopic(row.Qualifier, row.Topic)))
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

type graphTopicCatalogWriterDriftRule struct{}

func (r graphTopicCatalogWriterDriftRule) ID() string {
	return "graph_topic_catalog_writer_drift"
}

func (r graphTopicCatalogWriterDriftRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}
func (r graphTopicCatalogWriterDriftRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphTopicCatalogWriterDriftRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphTopicCatalogWriterDriftRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, row := range ctx.Block.Docs.TopicCatalog.Rows {
		for _, expectation := range topicCatalogWriterExpectations(ctx, row) {
			if !expectation.Enforceable {
				continue
			}
			if ctx.Index.GraphHasRelationship(expectation.Relationship) && ctx.Index.RuntimeHasRelationship(expectation.Relationship, ctx.Matcher) {
				continue
			}
			f := builder.base(ctx.Block.Source.Path, row.SourceLine, catalogActorParityDetail("writer", row.Topic, expectation))
			f.Member = expectation.Relationship.Member
			f.Topic = row.Topic
			findings = append(findings, f)
		}
	}
	return findings
}

type graphTopicCatalogReaderDriftRule struct{}

func (r graphTopicCatalogReaderDriftRule) ID() string {
	return "graph_topic_catalog_reader_drift"
}

func (r graphTopicCatalogReaderDriftRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}
func (r graphTopicCatalogReaderDriftRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphTopicCatalogReaderDriftRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphTopicCatalogReaderDriftRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, row := range ctx.Block.Docs.TopicCatalog.Rows {
		for _, expectation := range topicCatalogReaderExpectations(ctx, row) {
			if !expectation.Enforceable {
				continue
			}
			if ctx.Index.GraphHasRelationship(expectation.Relationship) && ctx.Index.RuntimeHasRelationship(expectation.Relationship, ctx.Matcher) {
				continue
			}
			f := builder.base(ctx.Block.Source.Path, row.SourceLine, catalogActorParityDetail("reader", row.Topic, expectation))
			if row.StatusKind == OperatingTopicStatusLiveUnderConsumed {
				f.Severity = string(SeverityWarning)
			}
			f.Member = expectation.Relationship.Member
			f.Topic = row.Topic
			findings = append(findings, f)
		}
	}
	return findings
}

type graphTopicCatalogActorUnsupportedRule struct{}

func (r graphTopicCatalogActorUnsupportedRule) ID() string {
	return "graph_topic_catalog_actor_unsupported"
}

func (r graphTopicCatalogActorUnsupportedRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}

func (r graphTopicCatalogActorUnsupportedRule) DefaultSeverity() Severity {
	return SeverityWarning
}
func (r graphTopicCatalogActorUnsupportedRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphTopicCatalogActorUnsupportedRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, row := range ctx.Block.Docs.TopicCatalog.Rows {
		for _, expectation := range append(topicCatalogWriterExpectations(ctx, row), topicCatalogReaderExpectations(ctx, row)...) {
			if expectation.Enforceable || expectation.Reason == "" {
				continue
			}
			f := builder.base(ctx.Block.Source.Path, row.SourceLine, expectation.Reason)
			f.Topic = row.Topic
			findings = append(findings, f)
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

func validateOperatingActorRef(ctx OperatingGraphRuleContext, builder OperatingFindingBuilder, ref OperatingActorReference, line int) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	resolver := NewOperatingActorResolver(ctx.Block.Metadata)
	for _, expanded := range resolver.Expand(ctx.Block.Metadata.Team, ctx.Runtime, ref) {
		switch expanded.Kind {
		case OperatingActorKindMember:
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
		case OperatingActorKindTeam:
			if _, ok := ctx.Runtime.Contracts[expanded.Value]; !ok {
				f := builder.base(ctx.Block.Source.Path, line, fmt.Sprintf("actor reference %q resolves to unknown team %q", ref.Raw, expanded.Value))
				findings = append(findings, f)
			}
		case OperatingActorKindGroup:
			f := builder.base(ctx.Block.Source.Path, line, fmt.Sprintf("actor group %q is not declared for this operating graph", expanded.Value))
			findings = append(findings, f)
		case OperatingActorKindUnknown:
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

type catalogActorParityExpectation struct {
	Relationship OperatingRelationship
	Actor        OperatingActorReference
	Enforceable  bool
	Reason       string
}

func topicCatalogWriterExpectations(ctx OperatingGraphRuleContext, row OperatingTopicCatalogRow) []catalogActorParityExpectation {
	if row.Topic == "" || !operatingTopicCatalogStatusIsCurrent(row.StatusKind) {
		return nil
	}
	resolver := NewOperatingActorResolver(ctx.Block.Metadata, ctx.Block.Graph)
	var out []catalogActorParityExpectation
	for _, ref := range row.Writers {
		out = append(out, topicCatalogActorExpectations(ctx, resolver, row, ref, true)...)
	}
	return out
}

func topicCatalogReaderExpectations(ctx OperatingGraphRuleContext, row OperatingTopicCatalogRow) []catalogActorParityExpectation {
	if row.Topic == "" || !operatingTopicCatalogStatusIsCurrent(row.StatusKind) {
		return nil
	}
	resolver := NewOperatingActorResolver(ctx.Block.Metadata, ctx.Block.Graph)
	var out []catalogActorParityExpectation
	for _, ref := range row.Readers {
		out = append(out, topicCatalogActorExpectations(ctx, resolver, row, ref, false)...)
	}
	return out
}

func topicCatalogActorExpectations(ctx OperatingGraphRuleContext, resolver DefaultOperatingActorResolver, row OperatingTopicCatalogRow, ref OperatingActorReference, writer bool) []catalogActorParityExpectation {
	expanded := resolver.Expand(ctx.Block.Metadata.Team, ctx.Runtime, ref)
	if len(expanded) == 0 {
		return nil
	}
	out := make([]catalogActorParityExpectation, 0, len(expanded))
	for _, actor := range expanded {
		out = append(out, topicCatalogConcreteActorExpectation(ctx, row, actor, writer))
	}
	return out
}

func topicCatalogConcreteActorExpectation(ctx OperatingGraphRuleContext, row OperatingTopicCatalogRow, actor OperatingActorReference, writer bool) catalogActorParityExpectation {
	expectation := catalogActorParityExpectation{Actor: actor, Enforceable: true}
	rel := OperatingRelationship{Team: ctx.Block.Metadata.Team, Topic: row.Topic}
	if writer {
		switch actor.Kind {
		case OperatingActorKindMember:
			rel.Kind = operatingRelTopicOutput
			rel.Member = actor.Value
		case OperatingActorKindExternal:
			rel.Kind = operatingRelExternalProducerIntake
			rel.External = actor.Value
		case OperatingActorKindTeam:
			rel.Kind = operatingRelCrossTeamOutput
			rel.TargetTeam = actor.Value
		default:
			expectation.Enforceable = false
			expectation.Reason = fmt.Sprintf("Topic Catalog writer %q for %q is not enforceable as a graph/runtime relationship", actor.Raw, row.Topic)
		}
	} else {
		switch actor.Kind {
		case OperatingActorKindMember:
			rel.Kind = operatingRelTopicRead
			rel.Member = actor.Value
		case OperatingActorKindTeam:
			rel.Kind = operatingRelCrossTeamOutput
			rel.TargetTeam = actor.Value
		case OperatingActorKindExternal:
			expectation.Enforceable = false
			expectation.Reason = fmt.Sprintf("Topic Catalog reader %q for %q is external; external topic readers are not modeled by the operating graph runtime contract", actor.Raw, row.Topic)
		default:
			expectation.Enforceable = false
			expectation.Reason = fmt.Sprintf("Topic Catalog reader %q for %q is not enforceable as a graph/runtime relationship", actor.Raw, row.Topic)
		}
	}
	expectation.Relationship = rel
	return expectation
}

func catalogActorParityDetail(role, topic string, expectation catalogActorParityExpectation) string {
	statement := catalogActorRelationshipStatement(expectation.Relationship)
	if statement == "" {
		statement = fmt.Sprintf("%s %q", role, expectation.Actor.Raw)
	}
	return fmt.Sprintf("Topic Catalog %s %q for %q is not backed by graph/runtime relationship %s", role, expectation.Actor.Raw, topic, statement)
}

func catalogActorRelationshipStatement(rel OperatingRelationship) string {
	switch rel.Kind {
	case operatingRelTopicOutput:
		return fmt.Sprintf("member:%s -> topic:%s", rel.Member, rel.Topic)
	case operatingRelTopicRead:
		return fmt.Sprintf("topic:%s -> member:%s", rel.Topic, rel.Member)
	case operatingRelExternalProducerIntake:
		return fmt.Sprintf("external:%s -> topic:%s", rel.External, rel.Topic)
	case operatingRelCrossTeamOutput:
		return fmt.Sprintf("topic:%s -> team:%s", rel.Topic, rel.TargetTeam)
	default:
		return ""
	}
}

func catalogGraphTopicExists(block OperatingGraphBlock, qualifier, topic string) bool {
	for _, node := range block.Graph.Nodes {
		if node.Kind == OperatingGraphNodeKindTopic && string(node.Qualifier) == qualifier && node.Value == topic {
			return true
		}
	}
	return false
}

func catalogRowReferencesFutureTopic(block OperatingGraphBlock, row OperatingTopicCatalogRow) bool {
	if strings.Contains(row.Purpose, "topic[future]:") {
		return true
	}
	for _, node := range block.Graph.Nodes {
		if node.Kind == OperatingGraphNodeKindTopic && node.Qualifier == OperatingGraphQualifierFuture && topicsOverlap(node.Value, row.Topic) {
			return true
		}
	}
	return false
}
