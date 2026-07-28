package memberflow

import "fmt"

type graphDocsUnknownActorRule struct{}

func (r graphDocsUnknownActorRule) ID() string                { return "graph_docs_unknown_actor" }
func (r graphDocsUnknownActorRule) Group() RuleGroup          { return OperatingRuleGroupDocs }
func (r graphDocsUnknownActorRule) DefaultSeverity() Severity { return SeverityError }
func (r graphDocsUnknownActorRule) AppliesTo(ctx RuleContext) bool {
	return string(ctx.Block.Metadata.Mode) == "contract"
}

func (r graphDocsUnknownActorRule) Check(ctx RuleContext) []OperatingGraphFinding {
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

func (r graphTopicCatalogWriterDriftRule) Group() RuleGroup {
	return OperatingRuleGroupDocs
}
func (r graphTopicCatalogWriterDriftRule) DefaultSeverity() Severity { return SeverityError }
func (r graphTopicCatalogWriterDriftRule) AppliesTo(ctx RuleContext) bool {
	return string(ctx.Block.Metadata.Mode) == "contract"
}

func (r graphTopicCatalogWriterDriftRule) Check(ctx RuleContext) []OperatingGraphFinding {
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

func (r graphTopicCatalogReaderDriftRule) Group() RuleGroup {
	return OperatingRuleGroupDocs
}
func (r graphTopicCatalogReaderDriftRule) DefaultSeverity() Severity { return SeverityError }
func (r graphTopicCatalogReaderDriftRule) AppliesTo(ctx RuleContext) bool {
	return string(ctx.Block.Metadata.Mode) == "contract"
}

func (r graphTopicCatalogReaderDriftRule) Check(ctx RuleContext) []OperatingGraphFinding {
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

func (r graphTopicCatalogActorUnsupportedRule) Group() RuleGroup {
	return OperatingRuleGroupDocs
}

func (r graphTopicCatalogActorUnsupportedRule) DefaultSeverity() Severity {
	return SeverityWarning
}

func (r graphTopicCatalogActorUnsupportedRule) AppliesTo(ctx RuleContext) bool {
	return string(ctx.Block.Metadata.Mode) == "contract"
}

func (r graphTopicCatalogActorUnsupportedRule) Check(ctx RuleContext) []OperatingGraphFinding {
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

func validateOperatingActorRef(ctx RuleContext, builder OperatingFindingBuilder, ref OperatingActorReference, line int) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	resolver := NewOperatingActorResolver(ctx.Block.Metadata, ctx.Block.Graph)
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
		case OperatingActorKindProcess:
			continue
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

type catalogActorParityExpectation struct {
	Relationship OperatingRelationship
	Actor        OperatingActorReference
	Enforceable  bool
	Reason       string
}

func topicCatalogWriterExpectations(ctx RuleContext, row OperatingTopicCatalogRow) []catalogActorParityExpectation {
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

func topicCatalogReaderExpectations(ctx RuleContext, row OperatingTopicCatalogRow) []catalogActorParityExpectation {
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

func topicCatalogActorExpectations(ctx RuleContext, resolver DefaultOperatingActorResolver, row OperatingTopicCatalogRow, ref OperatingActorReference, writer bool) []catalogActorParityExpectation {
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

func topicCatalogConcreteActorExpectation(ctx RuleContext, row OperatingTopicCatalogRow, actor OperatingActorReference, writer bool) catalogActorParityExpectation {
	expectation := catalogActorParityExpectation{Actor: actor, Enforceable: true}
	if writer && actor.Kind == OperatingActorKindExternal && row.StatusKind == OperatingTopicStatusLiveSystem {
		expectation.Enforceable = false
		return expectation
	}
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
		case OperatingActorKindProcess:
			expectation.Enforceable = false
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
		case OperatingActorKindProcess:
			expectation.Enforceable = false
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
