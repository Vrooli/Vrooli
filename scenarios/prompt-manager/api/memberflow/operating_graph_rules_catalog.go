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

type graphTopicCatalogPurposeDriftRule struct{}

func (r graphTopicCatalogPurposeDriftRule) ID() string {
	return "graph_topic_catalog_purpose_drift"
}

func (r graphTopicCatalogPurposeDriftRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupDocs
}

func (r graphTopicCatalogPurposeDriftRule) DefaultSeverity() Severity {
	return SeverityError
}

func (r graphTopicCatalogPurposeDriftRule) AppliesTo(mode string) bool {
	return mode == "contract"
}

func (r graphTopicCatalogPurposeDriftRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	if !ctx.Block.Docs.TopicCatalog.Present {
		return nil
	}
	builder := NewOperatingFindingBuilder(ctx, r)
	catalog := topicCatalogByQualifiedTopic(ctx.Runtime.Contracts[ctx.Block.Metadata.Team])
	var findings []OperatingGraphFinding
	for _, row := range ctx.Block.Docs.TopicCatalog.Rows {
		if row.Topic == "" {
			continue
		}
		entry, ok := catalog[qualifiedTopicKey(row.Qualifier, row.Topic)]
		if !ok {
			if operatingTopicCatalogStatusIsCurrent(row.StatusKind) {
				f := builder.base(ctx.Block.Source.Path, row.SourceLine, fmt.Sprintf("Topic Catalog row %q has no matching team.json::topicCatalog entry", displayQualifiedTopic(row.Qualifier, row.Topic)))
				f.Topic = row.Topic
				findings = append(findings, f)
			}
			continue
		}
		if normalizeCatalogPurpose(row.Purpose) == normalizeCatalogPurpose(entry.Purpose) {
			continue
		}
		f := builder.base(ctx.Block.Source.Path, row.SourceLine, fmt.Sprintf("Topic Catalog purpose for %q does not match team.json::topicCatalog", displayQualifiedTopic(row.Qualifier, row.Topic)))
		f.Topic = row.Topic
		findings = append(findings, f)
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

func topicCatalogByQualifiedTopic(contract *LoadedTeamContract) map[string]TopicCatalogEntry {
	out := map[string]TopicCatalogEntry{}
	if contract == nil {
		return out
	}
	for _, entry := range contract.TopicCatalog {
		prefix := strings.TrimSpace(entry.Prefix)
		status := ParseOperatingTopicCatalogStatus(entry.Status)
		qualifier := strings.TrimSpace(entry.Qualifier)
		if qualifier == "" {
			qualifier, _ = expectedTopicCatalogQualifier(status)
		}
		out[qualifiedTopicKey(qualifier, prefix)] = entry
	}
	return out
}

func normalizeCatalogPurpose(raw string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
}
