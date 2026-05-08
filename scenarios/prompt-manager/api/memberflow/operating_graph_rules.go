package memberflow

type OperatingGraphRuleGroup string

const (
	OperatingRuleGroupEntity       OperatingGraphRuleGroup = "entity"
	OperatingRuleGroupEdgeTruth    OperatingGraphRuleGroup = "edge_truth"
	OperatingRuleGroupCompleteness OperatingGraphRuleGroup = "completeness"
	OperatingRuleGroupPrompt       OperatingGraphRuleGroup = "prompt"
	OperatingRuleGroupDocs         OperatingGraphRuleGroup = "docs"
	OperatingRuleGroupCoherence    OperatingGraphRuleGroup = "coherence"
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
	registry := DefaultOperatingRelationshipRegistry()
	rules := []OperatingGraphRule{
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
	}
	rules = append(rules, graphDeclaredRuntimeRelationshipMissingRules(registry)...)
	rules = append(rules,
		graphTopicCatalogMissingRule{},
		graphTopicCatalogInvalidTopicRule{},
		graphTopicCatalogDriftRule{},
		graphTopicCatalogUnknownStatusRule{},
		graphTopicCatalogStatusQualifierDriftRule{},
		graphTopicCatalogLiveStatusUnbackedRule{},
		graphTopicCatalogTransitionalWithoutTargetRule{},
		graphTopicCatalogPurposeDriftRule{},
		graphDocsUnknownActorRule{},
		graphTopicCatalogWriterDriftRule{},
		graphTopicCatalogReaderDriftRule{},
		graphTopicCatalogActorUnsupportedRule{},
		graphDecisionsTableMissingRule{},
		graphDecisionsTableDriftRule{},
		graphDecisionsTableOwnerDriftRule{},
		graphPromptTopicContractMissingRule{},
		graphPromptTopicContractSourceMismatchRule{},
		graphPromptTopicContractContentMismatchRule{},
	)
	// Coherence rules intentionally start after completeness rules.
	// Completeness proves docs, graph, runtime config, and prompts agree.
	// Coherence will later prove the agreed graph is operationally plausible
	// (for example: live topics have producers/consumers, queues drain, and
	// terminal topics are explicit). Keep that future rule family separate from
	// docs-table and relationship-completeness rules.
	return rules
}

func ValidateOperatingGraphs(blocks []OperatingGraphBlock, runtime OperatingGraphRuntime, teamFilter, idFilter string) OperatingGraphValidationResult {
	result := OperatingGraphValidationResult{Findings: []OperatingGraphFinding{}}
	for _, block := range filterOperatingGraphBlocks(blocks, teamFilter, idFilter) {
		if block.Metadata.Mode == OperatingGraphModeExplanatory {
			continue
		}
		ctx := NewOperatingGraphContractContext(block, runtime)
		ruleCtx := OperatingGraphRuleContext(ctx)
		for _, rule := range DefaultOperatingGraphRules() {
			if rule.AppliesTo(string(block.Metadata.Mode)) {
				addOperatingFindings(&result, rule.Check(ruleCtx))
			}
		}
	}
	sortOperatingFindings(result.Findings)
	return result
}
