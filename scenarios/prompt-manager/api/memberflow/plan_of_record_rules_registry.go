package memberflow

// planOfRecordValidationRule is the registry entry for the plan-of-record
// family. Its existing validators already produce OperatingGraphFinding, so
// no lossy projection is needed at the shared-rule boundary.
type planOfRecordValidationRule struct {
	id       string
	severity Severity
}

func (r planOfRecordValidationRule) ID() string                { return r.id }
func (planOfRecordValidationRule) Group() RuleGroup            { return OperatingRuleGroupPlanOfRecord }
func (r planOfRecordValidationRule) DefaultSeverity() Severity { return r.severity }
func (planOfRecordValidationRule) AppliesTo(ctx RuleContext) bool {
	return ctx.ModelContext != nil
}

func (r planOfRecordValidationRule) Check(ctx RuleContext) []OperatingGraphFinding {
	out := make([]OperatingGraphFinding, 0, len(ctx.PlanOfRecordFindings))
	for _, finding := range ctx.PlanOfRecordFindings {
		if finding.Rule == r.id {
			out = append(out, finding)
		}
	}
	return out
}

func DefaultPlanOfRecordRules() []Rule {
	ids := []struct {
		id       string
		severity Severity
	}{
		{"por_discovery_failed", SeverityError},
		{"por_manifest_unreadable", SeverityError},
		{"por_manifest_invalid", SeverityError},
		{"por_manifest_kind_unknown", SeverityError},
		{"por_manifest_schema_unknown", SeverityError},
		{"por_manifest_team_mismatch", SeverityError},
		{"por_required_section_missing", SeverityError},
		{"por_required_document_missing", SeverityError},
		{"por_package_required_file_missing", SeverityError},
		{"por_package_entries_missing", SeverityError},
		{"por_manifest_duplicate_section", SeverityError},
		{"por_manifest_path_invalid", SeverityError},
		{"por_manifest_duplicate_document", SeverityError},
		{"por_manifest_duplicate_package", SeverityError},
		{"por_document_unreadable", SeverityError},
		{"por_required_heading_missing", SeverityError},
		{"por_required_link_missing", SeverityError},
		{"por_unregistered_document", SeverityWarning},
		{"por_notebook_surface", SeverityError},
	}
	rules := make([]Rule, 0, len(ids))
	for _, id := range ids {
		rules = append(rules, planOfRecordValidationRule{id: id.id, severity: id.severity})
	}
	return rules
}

func PlanOfRecordRuleCatalog() (RuleCatalog, error) {
	entry := func(id string, severity Severity, description string) RuleCatalogEntry {
		return RuleCatalogEntry{ID: id, Group: OperatingRuleGroupPlanOfRecord, Severity: severity, Description: description, Actuator: "Correct the plan-of-record manifest or route content changes through the owning team decision context"}
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
