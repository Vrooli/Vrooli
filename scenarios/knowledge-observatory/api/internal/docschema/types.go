package docschema

// DOC: ../../docs/plans/knowledge-observatory-documentation-hub-expansion.md#phase-0-foundation--documentation-structure-standards-week-1

// DocType represents a known documentation file type.
type DocType string

const (
	DocTypeProblems        DocType = "problems"
	DocTypeProgress        DocType = "progress"
	DocTypeSeams           DocType = "seams"
	DocTypeInvariants      DocType = "invariants"
	DocTypeAssumptions     DocType = "assumptions"
	DocTypeErrorSemantics  DocType = "error-semantics"
	DocTypeSecurityPosture DocType = "security-posture"
	DocTypeTemporalFlows   DocType = "temporal-flows"
	DocTypeCoherenceNotes  DocType = "coherence-notes"
	DocTypeExperienceAudit DocType = "experience-audit"
	DocTypeQuickstart      DocType = "quickstart"
	DocTypeArchitecture    DocType = "architecture"
	DocTypeGlossary        DocType = "glossary"
	DocTypePRD             DocType = "prd"
	DocTypeReadme          DocType = "readme"
	DocTypeManifest        DocType = "manifest"
)

// ExpectedPath returns the canonical path for a doc type relative to scenario root.
func (dt DocType) ExpectedPath() string {
	switch dt {
	case DocTypeProblems:
		return "docs/internal/PROBLEMS.md"
	case DocTypeProgress:
		return "docs/internal/PROGRESS.md"
	case DocTypeSeams:
		return "docs/internal/SEAMS.md"
	case DocTypeInvariants:
		return "docs/internal/INVARIANTS.md"
	case DocTypeAssumptions:
		return "docs/internal/ASSUMPTIONS.md"
	case DocTypeErrorSemantics:
		return "docs/internal/ERROR-SEMANTICS.md"
	case DocTypeSecurityPosture:
		return "docs/internal/SECURITY-POSTURE.md"
	case DocTypeTemporalFlows:
		return "docs/internal/TEMPORAL-FLOWS.md"
	case DocTypeCoherenceNotes:
		return "docs/internal/COHERENCE-NOTES.md"
	case DocTypeExperienceAudit:
		return "docs/internal/EXPERIENCE-AUDIT.md"
	case DocTypeQuickstart:
		return "docs/QUICKSTART.md"
	case DocTypeArchitecture:
		return "docs/concepts/ARCHITECTURE.md"
	case DocTypeGlossary:
		return "docs/concepts/GLOSSARY.md"
	case DocTypePRD:
		return "PRD.md"
	case DocTypeManifest:
		return "docs/manifest.json"
	case DocTypeReadme:
		return "README.md"
	default:
		return ""
	}
}

// ScenarioDocStructure defines the expected documentation layout.
type ScenarioDocStructure struct {
	Required    []DocType
	Optional    []DocType
	CustomPaths map[string]string // scenario-specific overrides keyed by DocType
}

var StandardScenarioStructure = ScenarioDocStructure{
	Required: []DocType{DocTypeReadme},
	Optional: []DocType{
		DocTypeProblems,
		DocTypeProgress,
		DocTypeSeams,
		DocTypeInvariants,
		DocTypeAssumptions,
		DocTypeErrorSemantics,
		DocTypeSecurityPosture,
		DocTypeTemporalFlows,
		DocTypeCoherenceNotes,
		DocTypeExperienceAudit,
		DocTypeQuickstart,
		DocTypeArchitecture,
		DocTypeGlossary,
		DocTypePRD,
		DocTypeManifest,
	},
}
