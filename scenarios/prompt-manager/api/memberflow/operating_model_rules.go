package memberflow

type OperatingModelRuleGroup string

const (
	OperatingModelRuleGroupStructure       OperatingModelRuleGroup = "structure"
	OperatingModelRuleGroupDecision        OperatingModelRuleGroup = "decision"
	OperatingModelRuleGroupExternalInput   OperatingModelRuleGroup = "external_input"
	OperatingModelRuleGroupOutput          OperatingModelRuleGroup = "output"
	OperatingModelRuleGroupFeedback        OperatingModelRuleGroup = "feedback"
	OperatingModelRuleGroupGap             OperatingModelRuleGroup = "gap"
	OperatingModelRuleGroupAdoption        OperatingModelRuleGroup = "adoption"
	OperatingModelRuleGroupDiscoverability OperatingModelRuleGroup = "discoverability"
)

type OperatingModelRule interface {
	ID() string
	Group() OperatingModelRuleGroup
	DefaultSeverity() Severity
	AppliesTo(model OperatingModelDocument) bool
	Check(ctx OperatingModelRuleContext) []OperatingGraphFinding
}

type OperatingModelRuleContext struct {
	Model          OperatingModelDocument
	Runtime        OperatingGraphRuntime
	ReferenceIndex OperatingModelReferenceIndex
}

type operatingModelRule struct {
	id    string
	group OperatingModelRuleGroup
	check func(OperatingModelRuleContext) []OperatingGraphFinding
}

func (rule operatingModelRule) ID() string {
	return rule.id
}

func (rule operatingModelRule) Group() OperatingModelRuleGroup {
	return rule.group
}

func (rule operatingModelRule) DefaultSeverity() Severity {
	return SeverityError
}

func (rule operatingModelRule) AppliesTo(model OperatingModelDocument) bool {
	return operatingModelPrimaryGraphMode(model) == OperatingGraphModeContract
}

func (rule operatingModelRule) Check(ctx OperatingModelRuleContext) []OperatingGraphFinding {
	if rule.check == nil {
		return nil
	}
	return rule.check(ctx)
}

func DefaultOperatingModelRules() []OperatingModelRule {
	return []OperatingModelRule{
		operatingModelRule{id: "operating_model_required_section_missing", group: OperatingModelRuleGroupStructure, check: checkOperatingModelRequiredSectionMissing},
		operatingModelRule{id: "operating_model_duplicate_section", group: OperatingModelRuleGroupStructure, check: checkOperatingModelDuplicateSection},
		operatingModelFilteredRule("operating_model_decisions_header_drift", OperatingModelRuleGroupDecision, validateOperatingModelDecisions),
		operatingModelFilteredRule("operating_model_decisions_empty", OperatingModelRuleGroupDecision, validateOperatingModelDecisions),
		operatingModelFilteredRule("operating_model_decisions_row_incomplete", OperatingModelRuleGroupDecision, validateOperatingModelDecisions),
		operatingModelFilteredRule("operating_model_decisions_effect_weak", OperatingModelRuleGroupDecision, validateOperatingModelDecisions),
		operatingModelContextualFilteredRule("operating_model_external_inputs_table_missing", OperatingModelRuleGroupExternalInput, validateOperatingModelExternalInputs),
		operatingModelContextualFilteredRule("operating_model_external_inputs_header_drift", OperatingModelRuleGroupExternalInput, validateOperatingModelExternalInputs),
		operatingModelContextualFilteredRule("operating_model_external_inputs_empty", OperatingModelRuleGroupExternalInput, validateOperatingModelExternalInputs),
		operatingModelContextualFilteredRule("operating_model_external_inputs_row_incomplete", OperatingModelRuleGroupExternalInput, validateOperatingModelExternalInputs),
		operatingModelContextualFilteredRule("operating_model_external_inputs_producer_unbacked", OperatingModelRuleGroupExternalInput, validateOperatingModelExternalInputs),
		operatingModelContextualFilteredRule("operating_model_external_inputs_entry_unbacked", OperatingModelRuleGroupExternalInput, validateOperatingModelExternalInputs),
		operatingModelContextualFilteredRule("operating_model_external_inputs_drainer_unbacked", OperatingModelRuleGroupExternalInput, validateOperatingModelExternalInputs),
		operatingModelContextualFilteredRule("operating_model_outputs_table_missing", OperatingModelRuleGroupOutput, validateOperatingModelOutputs),
		operatingModelContextualFilteredRule("operating_model_outputs_header_drift", OperatingModelRuleGroupOutput, validateOperatingModelOutputs),
		operatingModelContextualFilteredRule("operating_model_outputs_empty", OperatingModelRuleGroupOutput, validateOperatingModelOutputs),
		operatingModelContextualFilteredRule("operating_model_outputs_row_incomplete", OperatingModelRuleGroupOutput, validateOperatingModelOutputs),
		operatingModelContextualFilteredRule("operating_model_outputs_surface_unbacked", OperatingModelRuleGroupOutput, validateOperatingModelOutputs),
		operatingModelContextualFilteredRule("operating_model_outputs_consumer_unbacked", OperatingModelRuleGroupOutput, validateOperatingModelOutputs),
		operatingModelContextualFilteredRule("operating_model_feedback_steps_missing", OperatingModelRuleGroupFeedback, validateOperatingModelFeedback),
		operatingModelContextualFilteredRule("operating_model_feedback_step_unanchored", OperatingModelRuleGroupFeedback, validateOperatingModelFeedback),
		operatingModelContextualFilteredRule("operating_model_feedback_reference_unbacked", OperatingModelRuleGroupFeedback, validateOperatingModelFeedback),
		operatingModelFilteredRule("operating_model_gaps_items_missing", OperatingModelRuleGroupGap, validateOperatingModelGapItems),
		operatingModelFilteredRule("operating_model_gap_item_unanchored", OperatingModelRuleGroupGap, validateOperatingModelGapItems),
		operatingModelFilteredRule("operating_model_gap_item_target_state_missing", OperatingModelRuleGroupGap, validateOperatingModelGapItems),
		operatingModelFilteredRule("operating_model_adoption_command_missing", OperatingModelRuleGroupAdoption, validateOperatingModelAdoptionCommands),
		operatingModelFilteredRuleWithRuntime("operating_model_plan_of_record_missing", OperatingModelRuleGroupDiscoverability, validateOperatingModelDiscoverability),
		operatingModelFilteredRuleWithRuntime("operating_model_readme_link_missing", OperatingModelRuleGroupDiscoverability, validateOperatingModelDiscoverability),
	}
}

func operatingModelFilteredRule(id string, group OperatingModelRuleGroup, check func(OperatingModelDocument) []OperatingGraphFinding) OperatingModelRule {
	return operatingModelRule{
		id:    id,
		group: group,
		check: func(ctx OperatingModelRuleContext) []OperatingGraphFinding {
			return operatingModelFindingsForRule(check(ctx.Model), id)
		},
	}
}

func operatingModelContextualFilteredRule(id string, group OperatingModelRuleGroup, check func(OperatingModelRuleContext) []OperatingGraphFinding) OperatingModelRule {
	return operatingModelRule{
		id:    id,
		group: group,
		check: func(ctx OperatingModelRuleContext) []OperatingGraphFinding {
			return operatingModelFindingsForRule(check(ctx), id)
		},
	}
}

func operatingModelFilteredRuleWithRuntime(id string, group OperatingModelRuleGroup, check func(OperatingModelDocument, OperatingGraphRuntime) []OperatingGraphFinding) OperatingModelRule {
	return operatingModelRule{
		id:    id,
		group: group,
		check: func(ctx OperatingModelRuleContext) []OperatingGraphFinding {
			return operatingModelFindingsForRule(check(ctx.Model, ctx.Runtime), id)
		},
	}
}

func operatingModelFindingsForRule(findings []OperatingGraphFinding, ruleID string) []OperatingGraphFinding {
	var out []OperatingGraphFinding
	for _, finding := range findings {
		if finding.Rule == ruleID {
			out = append(out, finding)
		}
	}
	return out
}
