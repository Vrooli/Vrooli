package memberflow

import "fmt"

type graphPromptTopicContractMissingRule struct{}

func (r graphPromptTopicContractMissingRule) ID() string {
	return "graph_prompt_topic_contract_missing"
}

func (r graphPromptTopicContractMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupPrompt
}
func (r graphPromptTopicContractMissingRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphPromptTopicContractMissingRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphPromptTopicContractMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind != "member" {
			continue
		}
		if _, ok := topicContractPromptSection(ctx.Runtime, ctx.Block.Metadata.Team, node.Value); ok {
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

func (r graphPromptTopicContractSourceMismatchRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupPrompt
}

func (r graphPromptTopicContractSourceMismatchRule) DefaultSeverity() Severity {
	return SeverityError
}

func (r graphPromptTopicContractSourceMismatchRule) AppliesTo(mode string) bool {
	return mode == "contract"
}

func (r graphPromptTopicContractSourceMismatchRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	var findings []OperatingGraphFinding
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind != "member" {
			continue
		}
		section, ok := topicContractPromptSection(ctx.Runtime, ctx.Block.Metadata.Team, node.Value)
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
