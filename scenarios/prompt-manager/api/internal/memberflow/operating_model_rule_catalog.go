package memberflow

// OperatingModelRuleCatalog is deliberately authored rather than inferred
// from rule identifiers. It is the operator-facing contract for model
// validation: every failure names both the violated invariant and its repair
// surface before the rule can enter the shared registry.
func OperatingModelRuleCatalog() (RuleCatalog, error) {
	entry := func(id string, group RuleGroup, description string) RuleCatalogEntry {
		return RuleCatalogEntry{ID: id, Group: group, Severity: SeverityError, Kind: KindDeclaration, Description: description, Actuator: "Correct the operating-model document or route a team-owned content change through its work item type"}
	}
	return NewRuleCatalog(
		entry("operating_model_required_section_missing", OperatingModelRuleGroupStructure, "The operating model omits a required section."),
		entry("operating_model_duplicate_section", OperatingModelRuleGroupStructure, "The operating model defines a required section more than once."),
		entry("operating_model_external_inputs_table_missing", OperatingModelRuleGroupExternalInput, "The operating model omits its External Inputs table."),
		entry("operating_model_external_inputs_header_drift", OperatingModelRuleGroupExternalInput, "The External Inputs table header does not match the contract."),
		entry("operating_model_external_inputs_empty", OperatingModelRuleGroupExternalInput, "The External Inputs table has no declared inputs."),
		entry("operating_model_external_inputs_row_incomplete", OperatingModelRuleGroupExternalInput, "An External Inputs table row omits required information."),
		entry("operating_model_external_inputs_producer_unbacked", OperatingModelRuleGroupExternalInput, "An external input names a producer unsupported by the runtime contract."),
		entry("operating_model_external_inputs_entry_unbacked", OperatingModelRuleGroupExternalInput, "An external input entry is unsupported by the runtime contract."),
		entry("operating_model_external_inputs_drainer_unbacked", OperatingModelRuleGroupExternalInput, "An external input drainer is unsupported by the runtime contract."),
		entry("operating_model_outputs_table_missing", OperatingModelRuleGroupOutput, "The operating model omits its Outputs table."),
		entry("operating_model_outputs_header_drift", OperatingModelRuleGroupOutput, "The Outputs table header does not match the contract."),
		entry("operating_model_outputs_empty", OperatingModelRuleGroupOutput, "The Outputs table has no declared outputs."),
		entry("operating_model_outputs_row_incomplete", OperatingModelRuleGroupOutput, "An Outputs table row omits required information."),
		entry("operating_model_outputs_surface_unbacked", OperatingModelRuleGroupOutput, "An output surface is unsupported by the runtime contract."),
		entry("operating_model_outputs_consumer_unbacked", OperatingModelRuleGroupOutput, "An output consumer is unsupported by the runtime contract."),
		entry("operating_model_feedback_steps_missing", OperatingModelRuleGroupFeedback, "The operating model omits feedback steps."),
		entry("operating_model_feedback_step_unanchored", OperatingModelRuleGroupFeedback, "A feedback step has no traceable operating-model anchor."),
		entry("operating_model_feedback_reference_unbacked", OperatingModelRuleGroupFeedback, "A feedback reference is unsupported by the runtime contract."),
		entry("operating_model_gaps_items_missing", OperatingModelRuleGroupGap, "The operating model omits declared implementation gaps."),
		entry("operating_model_gap_item_unanchored", OperatingModelRuleGroupGap, "An implementation-gap item has no traceable anchor."),
		entry("operating_model_adoption_command_missing", OperatingModelRuleGroupAdoption, "The operating model omits an adoption command."),
		entry("operating_model_plan_of_record_missing", OperatingModelRuleGroupDiscoverability, "The operating model is missing its plan-of-record reference."),
		entry("operating_model_readme_link_missing", OperatingModelRuleGroupDiscoverability, "The operating model is not linked from its README."),
	)
}
