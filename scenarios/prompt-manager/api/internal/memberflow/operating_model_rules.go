package memberflow

const (
	OperatingModelRuleGroupStructure       RuleGroup = "structure"
	OperatingModelRuleGroupExternalInput   RuleGroup = "external_input"
	OperatingModelRuleGroupOutput          RuleGroup = "output"
	OperatingModelRuleGroupFeedback        RuleGroup = "feedback"
	OperatingModelRuleGroupGap             RuleGroup = "gap"
	OperatingModelRuleGroupAdoption        RuleGroup = "adoption"
	OperatingModelRuleGroupDiscoverability RuleGroup = "discoverability"
)

type OperatingModelRuleContext struct {
	Model          OperatingModelDocument
	Runtime        OperatingGraphRuntime
	ReferenceIndex OperatingModelReferenceIndex
}

// operatingModelRule is one registration per check function. Twenty-six
// registrations previously backed seven functions: each re-ran the whole check
// and then discarded every finding whose Rule field was not its own id, so a
// seven-id function ran seven times per model and threw away six sevenths of
// its output each time.
type operatingModelRule struct {
	id    string
	emits []string
	group RuleGroup
	check func(OperatingModelRuleContext) []OperatingGraphFinding
}

func (rule operatingModelRule) Emits() []string {
	if len(rule.emits) > 0 {
		return rule.emits
	}
	return []string{rule.id}
}

func (rule operatingModelRule) ID() string {
	return rule.id
}

func (rule operatingModelRule) Group() RuleGroup {
	return rule.group
}

func (rule operatingModelRule) DefaultSeverity() Severity {
	return SeverityError
}

func (rule operatingModelRule) AppliesTo(ctx RuleContext) bool {
	return ctx.ModelContext != nil && operatingModelPrimaryGraphMode(ctx.ModelContext.Model) == OperatingGraphModeContract
}

func (rule operatingModelRule) Check(ctx RuleContext) []OperatingGraphFinding {
	if rule.check == nil {
		return nil
	}
	if ctx.ModelContext == nil {
		return nil
	}
	return rule.check(*ctx.ModelContext)
}

func DefaultOperatingModelRules() []Rule {
	// model adapts a check that reads only the document.
	model := func(group RuleGroup, check func(OperatingModelDocument) []OperatingGraphFinding, ids ...string) Rule {
		return operatingModelRule{id: ids[0], emits: ids, group: group, check: func(ctx OperatingModelRuleContext) []OperatingGraphFinding {
			return check(ctx.Model)
		}}
	}
	// contextual adapts a check that also reads the reference index.
	contextual := func(group RuleGroup, check func(OperatingModelRuleContext) []OperatingGraphFinding, ids ...string) Rule {
		return operatingModelRule{id: ids[0], emits: ids, group: group, check: check}
	}
	// withRuntime adapts a check that also reads runtime declarations.
	withRuntime := func(group RuleGroup, check func(OperatingModelDocument, OperatingGraphRuntime) []OperatingGraphFinding, ids ...string) Rule {
		return operatingModelRule{id: ids[0], emits: ids, group: group, check: func(ctx OperatingModelRuleContext) []OperatingGraphFinding {
			return check(ctx.Model, ctx.Runtime)
		}}
	}
	return []Rule{
		operatingModelRule{id: "operating_model_required_section_missing", group: OperatingModelRuleGroupStructure, check: checkOperatingModelRequiredSectionMissing},
		operatingModelRule{id: "operating_model_duplicate_section", group: OperatingModelRuleGroupStructure, check: checkOperatingModelDuplicateSection},
		contextual(OperatingModelRuleGroupExternalInput, validateOperatingModelExternalInputs,
			"operating_model_external_inputs_table_missing",
			"operating_model_external_inputs_header_drift",
			"operating_model_external_inputs_empty",
			"operating_model_external_inputs_row_incomplete",
			"operating_model_external_inputs_producer_unbacked",
			"operating_model_external_inputs_entry_unbacked",
			"operating_model_external_inputs_drainer_unbacked"),
		contextual(OperatingModelRuleGroupOutput, validateOperatingModelOutputs,
			"operating_model_outputs_table_missing",
			"operating_model_outputs_header_drift",
			"operating_model_outputs_empty",
			"operating_model_outputs_row_incomplete",
			"operating_model_outputs_surface_unbacked",
			"operating_model_outputs_consumer_unbacked"),
		contextual(OperatingModelRuleGroupFeedback, validateOperatingModelFeedback,
			"operating_model_feedback_steps_missing",
			"operating_model_feedback_step_unanchored",
			"operating_model_feedback_reference_unbacked"),
		model(OperatingModelRuleGroupGap, validateOperatingModelGapItems,
			"operating_model_gaps_items_missing",
			"operating_model_gap_item_unanchored"),
		model(OperatingModelRuleGroupAdoption, validateOperatingModelAdoptionCommands,
			"operating_model_adoption_command_missing"),
		withRuntime(OperatingModelRuleGroupDiscoverability, validateOperatingModelDiscoverability,
			"operating_model_plan_of_record_missing",
			"operating_model_readme_link_missing"),
	}
}
