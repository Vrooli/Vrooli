package memberflow

// TopicRuleCatalog is the explicit operator-facing metadata for the topic
// validator family. Adding a topic rule requires adding an entry here; the
// catalog-enforced registry then rejects omissions and stale entries.
func TopicRuleCatalog() (RuleCatalog, error) {
	entry := func(id string, severity Severity, description string) RuleCatalogEntry {
		return RuleCatalogEntry{ID: id, Group: OperatingRuleGroupTopic, Severity: severity, Description: description, Actuator: "Route the declaration or runtime-state correction through the owning team"}
	}
	return NewRuleCatalog(
		entry("conflicting_drain", SeverityError, "A required intake is drained by incompatible member contracts."),
		entry("orphan_output", SeverityWarning, "A declared output has no declared consumer."),
		entry("orphan_input", SeverityError, "An intake has no declared producer."),
		entry("unread_required", SeverityError, "A required read is not consumed by its member contract."),
		entry("wildcard_source_misuse", SeverityWarning, "A wildcard source makes a topic-flow declaration ambiguous."),
		entry("missing_taxonomy", SeverityError, "A topic declaration omits the required taxonomy."),
		entry("unknown_taxonomy", SeverityError, "A topic declaration names an unregistered taxonomy."),
		entry("missing_destination_schema", SeverityWarning, "A destination topic has no declared storage schema."),
		entry("dangling_por_sink", SeverityError, "A topic flow points to a missing plan-of-record sink."),
		entry("dangling_evidence_decision", SeverityError, "A topic flow references a decision that does not exist."),
		entry("team_role_member_drift", SeverityError, "A team role and its member declaration disagree."),
		entry("actual_writer_undeclared", SeverityError, "Runtime attribution shows a member wrote outside its declared outputs."),
		entry("attribution_malformed", SeverityError, "Runtime attribution cannot be safely interpreted."),
		entry("prose_topic_leak", SeverityWarning, "Operator prose references an undeclared or impermissible topic."),
		entry("member_doc_unreadable", SeverityError, "A required member document cannot be read."),
		entry("member_doc_file_missing", SeverityError, "A required member document is absent."),
		entry("member_doc_section_alias", SeverityError, "A member document uses an ambiguous section alias."),
		entry("member_doc_section_duplicate", SeverityError, "A member document defines a required section more than once."),
		entry("member_doc_section_missing", SeverityError, "A member document omits a required section."),
		entry("member_doc_section_recommended", SeverityWarning, "A member document omits a recommended section."),
		entry("loop_kind_missing", SeverityWarning, "A declared loop has no loop-kind classification."),
		entry("loop_kind_invalid", SeverityError, "A declared loop uses an invalid loop-kind classification."),
		entry("loop_kind_intake_mismatch", SeverityError, "A loop-kind declaration disagrees with its intake contract."),
		entry("sweep_without_ledger", SeverityError, "A sweep loop has no evidence ledger."),
		entry("ledger_shape_invalid", SeverityError, "A sweep evidence ledger has an invalid shape."),
		entry("sweep_population_missing", SeverityWarning, "A sweep loop does not declare its population."),
	)
}
