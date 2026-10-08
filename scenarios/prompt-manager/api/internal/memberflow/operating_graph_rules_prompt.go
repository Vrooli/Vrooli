package memberflow

import (
	"fmt"
	"strings"
)

type graphPromptTopicContractMissingRule struct{}

func (r graphPromptTopicContractMissingRule) ID() string {
	return "graph_prompt_topic_contract_missing"
}

func (r graphPromptTopicContractMissingRule) Group() RuleGroup {
	return OperatingRuleGroupPrompt
}
func (r graphPromptTopicContractMissingRule) DefaultSeverity() Severity { return SeverityError }
func (r graphPromptTopicContractMissingRule) AppliesTo(ctx RuleContext) bool {
	return string(ctx.Block.Metadata.Mode) == "contract"
}

func (r graphPromptTopicContractMissingRule) Check(ctx RuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind != OperatingGraphNodeKindMember {
			continue
		}
		if hasDerivedOnlyTopicContractPromptSection(ctx.Runtime, ctx.Block.Metadata.Team, node.Value) {
			continue
		}
		if _, ok := liveTopicContractPromptSection(ctx.Runtime, ctx.Block.Metadata.Team, node.Value); ok {
			continue
		}
		f := builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("member %q is missing generated topic-contract prompt section", node.Value))
		f.Member = node.Value
		findings = append(findings, f)
	}
	return findings
}

type graphPromptTopicContractSourceMismatchRule struct{}

func (r graphPromptTopicContractSourceMismatchRule) ID() string {
	return "graph_prompt_topic_contract_source_mismatch"
}

type graphPromptTopicContractContentMismatchRule struct{}

func (r graphPromptTopicContractContentMismatchRule) ID() string {
	return "graph_prompt_topic_contract_content_mismatch"
}

func (r graphPromptTopicContractContentMismatchRule) Group() RuleGroup {
	return OperatingRuleGroupPrompt
}

func (r graphPromptTopicContractContentMismatchRule) DefaultSeverity() Severity {
	return SeverityError
}

func (r graphPromptTopicContractContentMismatchRule) AppliesTo(ctx RuleContext) bool {
	return string(ctx.Block.Metadata.Mode) == string(OperatingGraphModeContract)
}

func (r graphPromptTopicContractContentMismatchRule) Check(ctx RuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind != OperatingGraphNodeKindMember {
			continue
		}
		section, ok := liveTopicContractPromptSection(ctx.Runtime, ctx.Block.Metadata.Team, node.Value)
		if !ok || strings.TrimSpace(section.Content) == "" {
			continue
		}
		expected, ok := expectedTopicContractContent(ctx.Runtime, ctx.Block.Metadata.Team, node.Value)
		if !ok || normalizePromptSectionContent(section.Content) == normalizePromptSectionContent(expected) {
			continue
		}
		f := builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("member %q topic-contract prompt content differs from topics.json render", node.Value))
		f.Member = node.Value
		findings = append(findings, f)
	}
	return findings
}

func (r graphPromptTopicContractSourceMismatchRule) Group() RuleGroup {
	return OperatingRuleGroupPrompt
}

func (r graphPromptTopicContractSourceMismatchRule) DefaultSeverity() Severity {
	return SeverityError
}

func (r graphPromptTopicContractSourceMismatchRule) AppliesTo(ctx RuleContext) bool {
	return string(ctx.Block.Metadata.Mode) == "contract"
}

func (r graphPromptTopicContractSourceMismatchRule) Check(ctx RuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind != OperatingGraphNodeKindMember {
			continue
		}
		if hasDerivedOnlyTopicContractPromptSection(ctx.Runtime, ctx.Block.Metadata.Team, node.Value) {
			continue
		}
		section, ok := liveTopicContractPromptSection(ctx.Runtime, ctx.Block.Metadata.Team, node.Value)
		if !ok {
			continue
		}
		want := expectedTopicContractSourcePath(ctx.Block.Metadata.Team, node.Value)
		if section.SourcePath == want {
			continue
		}
		f := builder.WithNode(ctx.Block.Source.Path, node, fmt.Sprintf("member %q topic-contract prompt section source is %q, want %q", node.Value, section.SourcePath, want))
		f.Member = node.Value
		f.Path = section.SourcePath
		findings = append(findings, f)
	}
	return findings
}

func topicContractPromptSection(runtime OperatingGraphRuntime, team, member string) (OperatingGraphPromptSection, bool) {
	for _, section := range runtime.PromptSections[MemberRef{Team: team, Member: member}] {
		if section.Kind == operatingGraphPromptSectionKindTopicContract {
			return section, true
		}
	}
	return OperatingGraphPromptSection{}, false
}

func liveTopicContractPromptSection(runtime OperatingGraphRuntime, team, member string) (OperatingGraphPromptSection, bool) {
	for _, section := range runtime.PromptSections[MemberRef{Team: team, Member: member}] {
		if section.Kind == operatingGraphPromptSectionKindTopicContract && promptSectionIsLive(section) {
			return section, true
		}
	}
	return OperatingGraphPromptSection{}, false
}

func hasDerivedOnlyTopicContractPromptSection(runtime OperatingGraphRuntime, team, member string) bool {
	hasDerived := false
	for _, section := range runtime.PromptSections[MemberRef{Team: team, Member: member}] {
		if section.Kind != operatingGraphPromptSectionKindTopicContract {
			continue
		}
		if promptSectionIsLive(section) {
			return false
		}
		if section.SourceKind == OperatingGraphPromptSectionSourceDerived {
			hasDerived = true
		}
	}
	return hasDerived
}

func promptSectionIsLive(section OperatingGraphPromptSection) bool {
	return section.SourceKind == "" || section.SourceKind == OperatingGraphPromptSectionSourceLive
}

func normalizePromptSectionContent(content string) string {
	return strings.TrimSpace(strings.ReplaceAll(content, "\r\n", "\n"))
}
