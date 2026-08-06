package memberflow

// OperatingGraphRuleCatalog is the authored identity and remediation surface
// for graph validation. Completeness IDs are repeated intentionally: the
// relationship registry owns execution mechanics, while this catalog owns the
// operator contract and prevents a new executable rule from being anonymous.
func OperatingGraphRuleCatalog() (RuleCatalog, error) {
	entry := func(id string, group RuleGroup, severity Severity, description string) RuleCatalogEntry {
		return RuleCatalogEntry{ID: id, Group: group, Severity: severity, Kind: KindDeclaration, Description: description, Actuator: "Correct the declared graph or the supporting runtime contract"}
	}
	return NewRuleCatalog(
		entry("graph_untyped_node", OperatingRuleGroupEntity, SeverityError, "A graph node has no recognized type."),
		entry("graph_unknown_node_kind", OperatingRuleGroupEntity, SeverityError, "A graph node uses an unknown type."),
		entry("graph_node_shape_convention_drift", OperatingRuleGroupEntity, SeverityWarning, "A graph node shape conflicts with its declared type."),
		entry("graph_unknown_member", OperatingRuleGroupEntity, SeverityError, "A graph references an unknown member."),
		entry("graph_unknown_decision", OperatingRuleGroupEntity, SeverityError, "A graph references an unknown decision."),
		entry("graph_unknown_team", OperatingRuleGroupEntity, SeverityWarning, "A graph references an unknown team."),
		entry("graph_unknown_por", OperatingRuleGroupEntity, SeverityError, "A graph references an unknown plan-of-record surface."),
		entry("graph_future_topic_live_edge", OperatingRuleGroupEdgeTruth, SeverityWarning, "A live edge targets a future topic."),
		entry("graph_unsupported_edge_semantics", OperatingRuleGroupEdgeTruth, SeverityError, "A graph edge has unsupported semantics."),
		entry("graph_topic_catalog_missing", OperatingRuleGroupDocs, SeverityError, "The graph document omits its topic catalog."),
		entry("graph_topic_catalog_invalid_topic", OperatingRuleGroupDocs, SeverityError, "The topic catalog contains an invalid topic token."),
		entry("graph_topic_catalog_drift", OperatingRuleGroupDocs, SeverityError, "The topic catalog disagrees with the graph."),
		entry("graph_topic_catalog_unknown_status", OperatingRuleGroupDocs, SeverityError, "The topic catalog uses an unknown status."),
		entry("graph_topic_catalog_status_qualifier_drift", OperatingRuleGroupDocs, SeverityError, "A topic status conflicts with its qualifier."),
		entry("graph_topic_catalog_live_status_unbacked", OperatingRuleGroupDocs, SeverityError, "A live topic status has no graph evidence."),
		entry("graph_topic_catalog_transitional_without_target", OperatingRuleGroupDocs, SeverityWarning, "A transitional topic has no target state."),
		entry("graph_topic_catalog_purpose_drift", OperatingRuleGroupDocs, SeverityError, "A topic catalog purpose disagrees with its graph role."),
		entry("graph_decisions_table_missing", OperatingRuleGroupDocs, SeverityError, "The graph document omits its Decisions table."),
		entry("graph_decisions_table_drift", OperatingRuleGroupDocs, SeverityError, "The Decisions table disagrees with the graph."),
		entry("graph_decisions_table_owner_drift", OperatingRuleGroupDocs, SeverityError, "A Decisions table owner disagrees with the graph."),
		entry("graph_prompt_topic_contract_missing", OperatingRuleGroupPrompt, SeverityError, "A member lacks its generated topic-contract prompt section."),
		entry("graph_prompt_topic_contract_source_mismatch", OperatingRuleGroupPrompt, SeverityError, "A prompt topic-contract source disagrees with the declaration."),
		entry("graph_prompt_topic_contract_content_mismatch", OperatingRuleGroupPrompt, SeverityError, "A prompt topic-contract content differs from the declaration render."),
	)
}
