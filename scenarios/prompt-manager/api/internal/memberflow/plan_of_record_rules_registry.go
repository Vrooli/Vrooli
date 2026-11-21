package memberflow

// planOfRecordCheck is the single registry entry for the plan-of-record family.
//
// There were nineteen registrations, one per rule id, and each executed no
// check at all: it filtered the findings ValidateOperatingModels had already
// computed down to the ones matching its own id. Nineteen registrations, one
// real check, and eighteen redundant passes over the same slice. Now the family
// registers once and declares the ids it can produce.
type planOfRecordCheck struct {
	id       string
	emits    []string
	severity Severity
}

func (r planOfRecordCheck) ID() string                { return r.id }
func (planOfRecordCheck) Group() RuleGroup            { return OperatingRuleGroupPlanOfRecord }
func (r planOfRecordCheck) DefaultSeverity() Severity { return r.severity }
func (r planOfRecordCheck) Emits() []string           { return r.emits }
func (planOfRecordCheck) AppliesTo(ctx RuleContext) bool {
	return ctx.ModelContext != nil
}

// Check returns the plan-of-record findings as computed. Each already carries
// the rule id its check set, so there is nothing to filter or re-attribute.
func (r planOfRecordCheck) Check(ctx RuleContext) []OperatingGraphFinding {
	return ctx.PlanOfRecordFindings
}

func DefaultPlanOfRecordRules() []Rule {
	ids := []string{
		"por_discovery_failed",
		"por_manifest_unreadable",
		"por_manifest_invalid",
		"por_manifest_kind_unknown",
		"por_manifest_schema_unknown",
		"por_manifest_team_mismatch",
		"por_required_section_missing",
		"por_required_document_missing",
		"por_package_required_file_missing",
		"por_package_entries_missing",
		"por_manifest_duplicate_section",
		"por_manifest_path_invalid",
		"por_manifest_duplicate_document",
		"por_manifest_duplicate_package",
		"por_document_unreadable",
		"por_required_heading_missing",
		"por_required_link_missing",
		"por_unregistered_document",
		"por_notebook_surface",
	}
	return []Rule{planOfRecordCheck{id: ids[0], emits: ids, severity: SeverityError}}
}

func PlanOfRecordRuleCatalog() (RuleCatalog, error) {
	entry := func(id string, severity Severity, description string) RuleCatalogEntry {
		return RuleCatalogEntry{ID: id, Group: OperatingRuleGroupPlanOfRecord, Severity: severity, Kind: KindDeclaration, Description: description, Actuator: "Correct the plan-of-record manifest or route content changes through the owning team work item type"}
	}
	return NewRuleCatalog(
		entry("por_discovery_failed", SeverityError, "Plan-of-record discovery could not complete."),
		entry("por_manifest_unreadable", SeverityError, "A plan-of-record manifest cannot be read."),
		entry("por_manifest_invalid", SeverityError, "A plan-of-record manifest is malformed."),
		entry("por_manifest_kind_unknown", SeverityError, "A plan-of-record manifest names an unknown kind."),
		entry("por_manifest_schema_unknown", SeverityError, "A plan-of-record manifest names an unknown schema."),
		entry("por_manifest_team_mismatch", SeverityError, "A plan-of-record manifest belongs to a different team."),
		entry("por_required_section_missing", SeverityError, "A plan-of-record omits a required section."),
		entry("por_required_document_missing", SeverityError, "A plan-of-record omits a required document."),
		entry("por_package_required_file_missing", SeverityError, "A declared plan-of-record package omits a required file."),
		entry("por_package_entries_missing", SeverityError, "A declared plan-of-record package has no entries."),
		entry("por_manifest_duplicate_section", SeverityError, "A plan-of-record manifest duplicates a section."),
		entry("por_manifest_path_invalid", SeverityError, "A plan-of-record manifest contains an invalid path."),
		entry("por_manifest_duplicate_document", SeverityError, "A plan-of-record manifest duplicates a document."),
		entry("por_manifest_duplicate_package", SeverityError, "A plan-of-record manifest duplicates a package."),
		entry("por_document_unreadable", SeverityError, "A registered plan-of-record document cannot be read."),
		entry("por_required_heading_missing", SeverityError, "A plan-of-record document omits a required heading."),
		entry("por_required_link_missing", SeverityError, "A plan-of-record document omits a required link."),
		entry("por_unregistered_document", SeverityWarning, "A plan-of-record document is not registered in its manifest."),
		entry("por_notebook_surface", SeverityError, "A plan-of-record exposes a prohibited notebook surface."),
	)
}
