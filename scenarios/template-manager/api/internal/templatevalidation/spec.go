package templatevalidation

import "github.com/vrooli/maturity-go/assessment"

func MaturitySpec() *assessment.Spec {
	levels := []assessment.Level{
		{
			ID:            "L0",
			Name:          "Template provenance absent",
			Description:   "The scenario has no trustworthy record of the template lineage that produced or adopted it.",
			EntryCriteria: []string{"A scenario root exists and can be inspected."},
			ExitCriteria:  []string{"The scenario declares generation.template.id and generation.template.version in .vrooli/service.json."},
			StatusLabel:   "Unavailable",
			NextUnlock:    "Stamp adopted provenance for the latest default template.",
		},
		{
			ID:            "L1",
			Name:          "Template lineage known",
			Description:   "The scenario has a readable template provenance block.",
			EntryCriteria: []string{"Template provenance identifies a known template and version."},
			ExitCriteria:  []string{"Orientation state is available or the scenario is finalized."},
			StatusLabel:   "Foundation",
			NextUnlock:    "Make orientation standing machine-readable.",
		},
		{
			ID:            "L2",
			Name:          "Orientation standing known",
			Description:   "Template orientation standing can be evaluated without running deep validation.",
			EntryCriteria: []string{"Orientation state is readable when the scenario is not finalized."},
			ExitCriteria:  []string{"Template version and manifest hashes are current enough to bound drift and migration lag."},
			StatusLabel:   "Ready",
			NextUnlock:    "Resolve version lag and static drift evidence.",
		},
		{
			ID:            "L3",
			Name:          "Template standing current",
			Description:   "The scenario is at the current template version with bounded static drift.",
			EntryCriteria: []string{"Template version lag and static manifest drift are absent."},
			ExitCriteria:  []string{"Inherited template debt is resolved or intentionally accepted."},
			StatusLabel:   "Hardened",
			NextUnlock:    "Retire inherited template debt.",
		},
		{
			ID:                "L4",
			Name:              "Template standing clean",
			Description:       "The scenario carries current provenance, honest orientation state, bounded drift, and no inherited template debt.",
			EntryCriteria:     []string{"All lower rungs are satisfied and inherited template debt is clean."},
			ExitCriteria:      []string{"Template standing remains clean after new template releases."},
			StatusLabel:       "Complete",
			CapabilitySummary: "Template standing is self-describing and ready for fleet governance.",
		},
	}
	capabilities := []assessment.CapabilitySpec{
		capability("provenance", "Generation Provenance", "Template identity, version, and adoption metadata are present.", levels),
		capability("orientation", "Orientation Standing", "Generated initialization gates are visible and machine-readable.", levels),
		capability("drift_and_migration", "Drift and Migration", "Template version lag and static drift are bounded.", levels),
		capability("inherited_debt", "Inherited Debt", "Known template-origin defects are tracked and resolved.", levels),
	}
	return &assessment.Spec{
		Provider:     Provider,
		Phase:        Phase,
		Version:      "1.0.0",
		Levels:       levels,
		Capabilities: capabilities,
		Findings: map[string]assessment.FindingMapping{
			CodeProvenanceMissing: {
				CapabilityID:        "provenance",
				LocalLevelImpact:    "L0",
				GlobalImpact:        assessment.ImpactCapabilityGap,
				Dimension:           "operational-targets",
				SeverityDefault:     "SEVERITY_ERROR",
				CleanRequirement:    "required",
				RecommendedSkillIDs: []string{"scenario-generation"},
				FixClass:            assessment.FixClassAuto,
				FixerStatus:         assessment.FixerStatusImplemented,
			},
			CodeTemplateUnknown: {
				CapabilityID:        "provenance",
				LocalLevelImpact:    "L1",
				GlobalImpact:        assessment.ImpactCapabilityGap,
				Dimension:           "operational-targets",
				SeverityDefault:     "SEVERITY_ERROR",
				CleanRequirement:    "required",
				RecommendedSkillIDs: []string{"scenario-generation"},
				FixClass:            assessment.FixClassManual,
				FixReason:           "Unknown template lineage requires operator review before selecting a source template.",
			},
			CodeOrientationStateMissing: {
				CapabilityID:        "orientation",
				LocalLevelImpact:    "L1",
				GlobalImpact:        assessment.ImpactCapabilityGap,
				Dimension:           "operational-targets",
				SeverityDefault:     "SEVERITY_WARNING",
				CleanRequirement:    "required",
				RecommendedSkillIDs: []string{"scenario-generation"},
				FixClass:            assessment.FixClassManual,
				FixReason:           "Orientation completion depends on scenario-specific initialization work.",
			},
			CodeTemplateVersionLag: {
				CapabilityID:        "drift_and_migration",
				LocalLevelImpact:    "L2",
				GlobalImpact:        assessment.ImpactCapabilityGap,
				Dimension:           "operational-targets",
				SeverityDefault:     "SEVERITY_WARNING",
				CleanRequirement:    "advisory",
				RecommendedSkillIDs: []string{"scenario-generation"},
				FixClass:            assessment.FixClassManual,
				FixReason:           "Template migrations require reading changelog entries and applying scenario-specific changes.",
			},
			CodeTemplateManifestDrift: {
				CapabilityID:        "drift_and_migration",
				LocalLevelImpact:    "L2",
				GlobalImpact:        assessment.ImpactCapabilityGap,
				Dimension:           "operational-targets",
				SeverityDefault:     "SEVERITY_WARNING",
				CleanRequirement:    "advisory",
				RecommendedSkillIDs: []string{"scenario-generation"},
				FixClass:            assessment.FixClassManual,
				FixReason:           "Static drift must be reconciled against the current template and target scenario intent.",
			},
			CodeInheritedDebtOutstanding: {
				CapabilityID:        "inherited_debt",
				LocalLevelImpact:    "L3",
				GlobalImpact:        assessment.ImpactCapabilityGap,
				Dimension:           "operational-targets",
				SeverityDefault:     "SEVERITY_WARNING",
				CleanRequirement:    "advisory",
				RecommendedSkillIDs: []string{"test"},
				FixClass:            assessment.FixClassManual,
				FixReason:           "Inherited debt resolution depends on the concrete defect and target scenario surface.",
			},
		},
		Fallback: assessment.FallbackPolicy{
			CapabilityID:     "provenance",
			LocalLevelImpact: "L0",
			GlobalImpact:     assessment.ImpactUnknown,
			Dimension:        "operational-targets",
			SeverityDefault:  "SEVERITY_WARNING",
			CleanRequirement: "advisory",
		},
	}
}

func capability(id, label, description string, levels []assessment.Level) assessment.CapabilitySpec {
	return assessment.CapabilitySpec{
		ID:          id,
		Label:       label,
		Description: description,
		Levels:      append([]assessment.Level(nil), levels...),
	}
}
