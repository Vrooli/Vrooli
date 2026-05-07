package memberflow

type OperatingGraphRuleGroup string

const (
	OperatingRuleGroupEntity       OperatingGraphRuleGroup = "entity"
	OperatingRuleGroupEdgeTruth    OperatingGraphRuleGroup = "edge_truth"
	OperatingRuleGroupCompleteness OperatingGraphRuleGroup = "completeness"
	OperatingRuleGroupPrompt       OperatingGraphRuleGroup = "prompt"
)

type OperatingGraphRule interface {
	ID() string
	Group() OperatingGraphRuleGroup
	DefaultSeverity() Severity
	AppliesTo(mode string) bool
	Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding
}

type OperatingGraphRuleContext struct {
	Block   OperatingGraphBlock
	Runtime OperatingGraphRuntime
	Index   OperatingGraphContractIndex
	Matcher OperatingRelationshipMatcher
}

func DefaultOperatingGraphRules() []OperatingGraphRule {
	return []OperatingGraphRule{
		graphUntypedNodeRule{},
		graphUnknownNodeKindRule{},
		graphUnknownMemberRule{},
		graphUnknownDecisionRule{},
		graphUnknownTeamRule{},
		graphUnknownPORRule{},
		graphTopicUnresolvedRule{},
		graphFutureTopicLiveEdgeRule{},
		graphUnsupportedEdgeSemanticsRule{},
		graphEdgeUnbackedRule{},
		graphDeclaredMemberMissingRule{},
		graphDeclaredIntakeMissingRule{},
		graphDeclaredRequiredReadMissingRule{},
		graphDeclaredEvidenceMissingRule{},
		graphDeclaredOutputMissingRule{},
		graphDeclaredDecisionOwnedMissingRule{},
		graphDeclaredDecisionConsumedMissingRule{},
		graphDeclaredCapabilityGapMissingRule{},
		graphDeclaredExternalProducerMissingRule{},
		graphDeclaredCrossTeamOutputMissingRule{},
		graphPromptTopicContractMissingRule{},
		graphPromptTopicContractSourceMismatchRule{},
	}
}

func ValidateOperatingGraphs(blocks []OperatingGraphBlock, runtime OperatingGraphRuntime, teamFilter, idFilter string) OperatingGraphValidationResult {
	result := OperatingGraphValidationResult{Findings: []OperatingGraphFinding{}}
	for _, block := range filterOperatingGraphBlocks(blocks, teamFilter, idFilter) {
		if block.Metadata.Mode == "explanatory" {
			continue
		}
		ctx := NewOperatingGraphContractContext(block, runtime)
		ruleCtx := OperatingGraphRuleContext(ctx)
		for _, rule := range DefaultOperatingGraphRules() {
			if rule.AppliesTo(block.Metadata.Mode) {
				addOperatingFindings(&result, rule.Check(ruleCtx))
			}
		}
	}
	sortOperatingFindings(result.Findings)
	return result
}
