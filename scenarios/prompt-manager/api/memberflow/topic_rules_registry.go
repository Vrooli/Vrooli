package memberflow

// topicRule adapts the topic-validation result shape to the shared rule
// registry. Validate converts the shared representation back before exposing
// it, preserving the topic validator's public Finding contract.
type topicRule struct {
	id       string
	severity Severity
	check    func([]MemberTopics, ValidationOptions) []Finding
}

type findingRule interface {
	Rule
	CheckFindings(RuleContext) []Finding
}

func (r topicRule) ID() string                 { return r.id }
func (r topicRule) Group() RuleGroup           { return OperatingRuleGroupTopic }
func (r topicRule) DefaultSeverity() Severity  { return r.severity }
func (r topicRule) AppliesTo(RuleContext) bool { return true }
func (r topicRule) Check(ctx RuleContext) []OperatingGraphFinding {
	return findingsToOperatingFindings(r.CheckFindings(ctx))
}

func (r topicRule) CheckFindings(ctx RuleContext) []Finding {
	return topicFindingsForRule(r.id, r.check(ctx.Members, ctx.Options))
}

func topicFindingsForRule(id string, findings []Finding) []Finding {
	out := make([]Finding, 0, len(findings))
	for _, finding := range findings {
		if finding.Rule == id {
			out = append(out, finding)
		}
	}
	return out
}

func findingsToOperatingFindings(findings []Finding) []OperatingGraphFinding {
	out := make([]OperatingGraphFinding, 0, len(findings))
	for _, finding := range findings {
		out = append(out, OperatingGraphFinding{
			Rule:     finding.Rule,
			Severity: string(finding.Severity),
			Member:   finding.Member.String(),
			Topic:    finding.Prefix,
			Detail:   finding.Detail,
		})
	}
	return out
}

func DefaultTopicRules() []Rule {
	return []Rule{
		topicRule{id: "conflicting_drain", severity: SeverityError, check: func(m []MemberTopics, _ ValidationOptions) []Finding { return ruleConflictingDrain(m) }},
		topicRule{id: "orphan_output", severity: SeverityWarning, check: ruleOrphanOutput},
		topicRule{id: "orphan_input", severity: SeverityError, check: func(m []MemberTopics, _ ValidationOptions) []Finding { return ruleOrphanInput(m) }},
		topicRule{id: "unread_required", severity: SeverityError, check: ruleUnreadRequired},
		topicRule{id: "wildcard_source_misuse", severity: SeverityWarning, check: func(m []MemberTopics, _ ValidationOptions) []Finding { return ruleWildcardSourceMisuse(m) }},
		topicRule{id: "missing_taxonomy", severity: SeverityError, check: ruleUnknownTaxonomy},
		topicRule{id: "unknown_taxonomy", severity: SeverityError, check: ruleUnknownTaxonomy},
		topicRule{id: "missing_destination_schema", severity: SeverityWarning, check: ruleMissingDestinationSchema},
		topicRule{id: "dangling_por_sink", severity: SeverityError, check: ruleDanglingPORSink},
		topicRule{id: "dangling_evidence_decision", severity: SeverityError, check: ruleDanglingEvidenceDecision},
		topicRule{id: "team_role_member_drift", severity: SeverityError, check: func(_ []MemberTopics, o ValidationOptions) []Finding { return ruleTeamRoleMemberDrift(o) }},
		topicRule{id: "actual_writer_undeclared", severity: SeverityError, check: ruleActualWriterUndeclared},
		topicRule{id: "attribution_malformed", severity: SeverityError, check: ruleActualWriterUndeclared},
		topicRule{id: "prose_topic_leak", severity: SeverityWarning, check: ruleProseTopicLeak},
		topicRule{id: "member_doc_unreadable", severity: SeverityError, check: ruleMemberDocSections},
		topicRule{id: "member_doc_file_missing", severity: SeverityError, check: ruleMemberDocSections},
		topicRule{id: "member_doc_section_alias", severity: SeverityError, check: ruleMemberDocSections},
		topicRule{id: "member_doc_section_duplicate", severity: SeverityError, check: ruleMemberDocSections},
		topicRule{id: "member_doc_section_missing", severity: SeverityError, check: ruleMemberDocSections},
		topicRule{id: "member_doc_section_recommended", severity: SeverityWarning, check: ruleMemberDocSections},
		topicRule{id: "loop_kind_missing", severity: SeverityWarning, check: func(m []MemberTopics, _ ValidationOptions) []Finding { return ruleLoopKindMissing(m) }},
		topicRule{id: "loop_kind_invalid", severity: SeverityError, check: func(m []MemberTopics, _ ValidationOptions) []Finding { return ruleLoopKindInvalid(m) }},
		topicRule{id: "loop_kind_intake_mismatch", severity: SeverityError, check: func(m []MemberTopics, _ ValidationOptions) []Finding { return ruleLoopKindIntakeMismatch(m) }},
		topicRule{id: "sweep_without_ledger", severity: SeverityError, check: func(m []MemberTopics, _ ValidationOptions) []Finding { return ruleSweepWithoutLedger(m) }},
		topicRule{id: "ledger_shape_invalid", severity: SeverityError, check: func(m []MemberTopics, _ ValidationOptions) []Finding { return ruleLedgerShapeInvalid(m) }},
		topicRule{id: "sweep_population_missing", severity: SeverityWarning, check: func(m []MemberTopics, _ ValidationOptions) []Finding { return ruleSweepPopulationMissing(m) }},
	}
}
