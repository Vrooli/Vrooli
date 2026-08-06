package memberflow

// findingRule is implemented by a rule whose findings carry the richer member
// and prefix identity the topic pass reports. Registration rejects a topic-pass
// rule that cannot produce one.
type findingRule interface {
	CheckFindings(ctx RuleContext) []Finding
}

// topicCheck adapts a topic-validation function to the shared rule contract.
//
// There is one registration per check function. Ten registrations previously
// backed three functions — ruleMemberDocSections ran six times per pass and
// ruleUnknownTaxonomy and ruleActualWriterUndeclared twice each — and every one
// of them discarded the findings whose Rule field did not match its own id.
// That filter is deleted: a check already sets the rule id on each finding it
// produces, so execution attributes by the id the check chose.
type topicCheck struct {
	// id is the registration identity. For a multi-id check it is the first
	// declared id, which keeps registry lookup keyed on something stable.
	id       string
	emits    []string
	severity Severity
	check    func([]MemberTopics, ValidationOptions) []Finding
}

func (r topicCheck) ID() string                 { return r.id }
func (r topicCheck) Group() RuleGroup           { return OperatingRuleGroupTopic }
func (r topicCheck) DefaultSeverity() Severity  { return r.severity }
func (r topicCheck) AppliesTo(RuleContext) bool { return true }
func (r topicCheck) Emits() []string            { return r.emits }

func (r topicCheck) Check(ctx RuleContext) []OperatingGraphFinding {
	return r.CheckFindings(ctx)
}

func (r topicCheck) CheckFindings(ctx RuleContext) []Finding {
	return r.check(ctx.Members, ctx.Options)
}

// members adapts a check that reads only member declarations.
func members(fn func([]MemberTopics) []Finding) func([]MemberTopics, ValidationOptions) []Finding {
	return func(m []MemberTopics, _ ValidationOptions) []Finding { return fn(m) }
}

// options adapts a check that reads only validation options.
func options(fn func(ValidationOptions) []Finding) func([]MemberTopics, ValidationOptions) []Finding {
	return func(_ []MemberTopics, o ValidationOptions) []Finding { return fn(o) }
}

func DefaultTopicRules() []Rule {
	one := func(id string, severity Severity, check func([]MemberTopics, ValidationOptions) []Finding) Rule {
		return topicCheck{id: id, emits: []string{id}, severity: severity, check: check}
	}
	return []Rule{
		one("conflicting_drain", SeverityError, members(ruleConflictingDrain)),
		one("orphan_output", SeverityWarning, ruleOrphanOutput),
		one("orphan_input", SeverityError, members(ruleOrphanInput)),
		one("unread_required", SeverityError, ruleUnreadRequired),
		one("wildcard_source_misuse", SeverityWarning, members(ruleWildcardSourceMisuse)),
		// One function, two ids: a topic whose taxonomy is absent and one whose
		// taxonomy is unknown are found by the same scan.
		topicCheck{
			id:       "missing_taxonomy",
			emits:    []string{"missing_taxonomy", "unknown_taxonomy"},
			severity: SeverityError,
			check:    ruleUnknownTaxonomy,
		},
		one("missing_destination_schema", SeverityWarning, ruleMissingDestinationSchema),
		one("dangling_por_sink", SeverityError, ruleDanglingPORSink),
		one("dangling_evidence_decision", SeverityError, ruleDanglingEvidenceDecision),
		one("team_role_member_drift", SeverityError, options(ruleTeamRoleMemberDrift)),
		// One function, two ids: an undeclared writer and a malformed
		// attribution record are both read out of the same knowledge log.
		topicCheck{
			id:       "actual_writer_undeclared",
			emits:    []string{"actual_writer_undeclared", "attribution_malformed"},
			severity: SeverityError,
			check:    ruleActualWriterUndeclared,
		},
		one("prose_topic_leak", SeverityWarning, ruleProseTopicLeak),
		// One function, six ids. This ran six times per validation pass, each
		// run discarding five sixths of its own output.
		topicCheck{
			id: "member_doc_unreadable",
			emits: []string{
				"member_doc_unreadable",
				"member_doc_file_missing",
				"member_doc_section_alias",
				"member_doc_section_duplicate",
				"member_doc_section_missing",
				"member_doc_section_recommended",
			},
			severity: SeverityError,
			check:    ruleMemberDocSections,
		},
		one("loop_kind_missing", SeverityWarning, members(ruleLoopKindMissing)),
		one("loop_kind_invalid", SeverityError, members(ruleLoopKindInvalid)),
		one("loop_kind_intake_mismatch", SeverityError, members(ruleLoopKindIntakeMismatch)),
		one("sweep_without_ledger", SeverityError, members(ruleSweepWithoutLedger)),
		one("ledger_shape_invalid", SeverityError, members(ruleLedgerShapeInvalid)),
		one("sweep_population_missing", SeverityWarning, members(ruleSweepPopulationMissing)),
		// Runtime enrichment. These were appended to the finding list after the
		// registry had run — a post-hoc injection path that let five rule ids
		// reach output with no catalog entry behind them. They register here so
		// the catalog covers them; the enrichment call sites still supply the
		// live query, and AppliesTo keeps them inert when it is absent.
		runtimeEnrichmentRule{
			id:    "stalled_drain",
			emits: []string{"stalled_drain", "piling_inbox", "drain_status_unavailable"},
		},
		runtimeEnrichmentRule{
			id:    "topic_key_prefix_mismatch",
			emits: []string{"topic_key_prefix_mismatch", "topic_key_query_unavailable"},
		},
	}
}

// runtimeEnrichmentRule claims the ids produced by the drain-status and
// key-prefix enrichment passes.
//
// Those passes need a live KnowledgeQuery that the pure-Go validation path does
// not have, so they run at the call site that owns the query and append their
// findings afterwards. Registering them here does not move that execution; it
// makes their ids catalogued, which is what stops five rule ids from reaching
// an operator with no description, no severity contract, and no documentation
// row. Phase 6 gates on Kind, and all five are Kind=runtime.
type runtimeEnrichmentRule struct {
	id    string
	emits []string
}

func (r runtimeEnrichmentRule) ID() string                              { return r.id }
func (runtimeEnrichmentRule) Group() RuleGroup                          { return OperatingRuleGroupTopic }
func (runtimeEnrichmentRule) DefaultSeverity() Severity                 { return SeverityWarning }
func (r runtimeEnrichmentRule) Emits() []string                         { return r.emits }
func (runtimeEnrichmentRule) AppliesTo(RuleContext) bool                { return false }
func (runtimeEnrichmentRule) Check(RuleContext) []OperatingGraphFinding { return nil }
func (runtimeEnrichmentRule) CheckFindings(RuleContext) []Finding       { return nil }
