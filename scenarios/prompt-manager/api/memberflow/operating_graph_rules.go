package memberflow

const (
	OperatingRuleGroupEntity       RuleGroup = "entity"
	OperatingRuleGroupEdgeTruth    RuleGroup = "edge_truth"
	OperatingRuleGroupCompleteness RuleGroup = "completeness"
	OperatingRuleGroupPrompt       RuleGroup = "prompt"
	OperatingRuleGroupDocs         RuleGroup = "docs"
	OperatingRuleGroupCoherence    RuleGroup = "coherence"
	OperatingRuleGroupTopic        RuleGroup = "topic"
	OperatingRuleGroupPlanOfRecord RuleGroup = "plan_of_record"
)

type Rule interface {
	ID() string
	Group() RuleGroup
	DefaultSeverity() Severity
	AppliesTo(ctx RuleContext) bool
	Check(ctx RuleContext) []OperatingGraphFinding
}

// RuleContext is the single execution context for every validation rule.
// A graph rule reads OperatingGraphRuleContext; a model rule reads
// OperatingModelRuleContext. Embedding keeps both surfaces explicit while
// allowing one registry and one execution contract.
type RuleContext struct {
	OperatingGraphRuleContext
	ModelContext         *OperatingModelRuleContext
	ObjectiveInput       *ObjectiveValidationInput
	PlanOfRecordFindings []OperatingGraphFinding
	Members              []MemberTopics
	Options              ValidationOptions
}

// GraphBlock reports the graph block and whether one was supplied. The block is
// one optional input among several rather than an embedded requirement: a check
// that reads no graph — the topic, plan-of-record, and objective families — must
// be registrable without one. Requiring it is why those families grew adapter
// types that executed nothing or went around the registry entirely.
func (c RuleContext) GraphBlock() (OperatingGraphBlock, bool) {
	if c.Block.Source.Path == "" {
		return OperatingGraphBlock{}, false
	}
	return c.Block, true
}

type OperatingGraphRuleContext struct {
	Block   OperatingGraphBlock
	Runtime OperatingGraphRuntime
	Index   OperatingGraphContractIndex
	Matcher OperatingRelationshipMatcher
}

func DefaultOperatingGraphRules() []Rule {
	rules := []Rule{
		graphUntypedNodeRule{},
		graphUnknownNodeKindRule{},
		graphNodeShapeConventionDriftRule{},
		graphUnknownMemberRule{},
		graphUnknownDecisionRule{},
		graphUnknownTeamRule{},
		graphUnknownPORRule{},
		graphFutureTopicLiveEdgeRule{},
		graphUnsupportedEdgeSemanticsRule{},
	}
	rules = append(rules,
		graphTopicCatalogMissingRule{},
		graphTopicCatalogInvalidTopicRule{},
		graphTopicCatalogDriftRule{},
		graphTopicCatalogUnknownStatusRule{},
		graphTopicCatalogStatusQualifierDriftRule{},
		graphTopicCatalogLiveStatusUnbackedRule{},
		graphTopicCatalogTransitionalWithoutTargetRule{},
		graphTopicCatalogPurposeDriftRule{},
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
	registry, err := DefaultRuleRegistry()
	if err != nil {
		addOperatingFindings(&result, []OperatingGraphFinding{{Rule: "rule_registry_invalid", Severity: SeverityError, Detail: err.Error()}})
		return result
	}
	for _, block := range filterOperatingGraphBlocks(blocks, teamFilter, idFilter) {
		if block.Metadata.Mode == OperatingGraphModeExplanatory {
			continue
		}
		ctx := NewOperatingGraphContractContext(block, runtime)
		ruleCtx := RuleContext{OperatingGraphRuleContext: OperatingGraphRuleContext(ctx)}
		for _, rule := range registry.RulesForPass(RulePassGraph) {
			if rule.AppliesTo(ruleCtx) {
				addOperatingFindings(&result, rule.Check(ruleCtx))
			}
		}
	}
	sortOperatingFindings(result.Findings)
	return result
}
