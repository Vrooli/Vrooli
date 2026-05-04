package docschema

import (
	"path"
	"regexp"
)

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
	DocTypePerfAudit       DocType = "perf-audit"
)

// perfAuditFilenamePattern matches `<YYYY-MM-DD>-<kebab-slug>.md`. The leading
// date keeps the directory chronologically sortable; the slug is free-form
// kebab-case with dots/digits allowed for version-y suffixes.
var perfAuditFilenamePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-[a-z0-9][a-z0-9.-]*\.md$`)

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
	case DocTypePerfAudit:
		// Many-file-per-directory doc type. ExpectedDir() + FilenamePattern()
		// are the canonical placement contract; ExpectedPath stays empty.
		return ""
	default:
		return ""
	}
}

// ExpectedDir returns the canonical directory for a doc type. For most types
// this is path.Dir(ExpectedPath()). For directory-pattern types like
// DocTypePerfAudit it returns the parent directory whose contents match
// FilenamePattern.
func (dt DocType) ExpectedDir() string {
	if dt == DocTypePerfAudit {
		return "docs/perf"
	}
	ep := dt.ExpectedPath()
	if ep == "" {
		return ""
	}
	dir := path.Dir(ep)
	if dir == "." {
		return ""
	}
	return dir
}

// FilenamePattern returns the regex a filename must match within ExpectedDir
// for directory-pattern doc types. Returns nil for fixed-path doc types
// (where ExpectedPath is the canonical placement).
func (dt DocType) FilenamePattern() *regexp.Regexp {
	if dt == DocTypePerfAudit {
		return perfAuditFilenamePattern
	}
	return nil
}

// ExpectedPathDisplay returns a human-readable "where files of this type
// belong" string. Fixed-path types return their ExpectedPath verbatim;
// directory-pattern types return `<ExpectedDir>/<filename-pattern-hint>`.
// Use this for CLI/HTTP responses so directory-pattern types still show a
// useful placement string.
func (dt DocType) ExpectedPathDisplay() string {
	if ep := dt.ExpectedPath(); ep != "" {
		return ep
	}
	if dt == DocTypePerfAudit {
		return "docs/perf/<YYYY-MM-DD>-<slug>.md"
	}
	if dir := dt.ExpectedDir(); dir != "" {
		return dir + "/"
	}
	return ""
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
		// DocTypePerfAudit is intentionally excluded from infrastructure
		// validation: it is many-file-per-directory and validated through
		// AuditPerfDocs (frontmatter + per-component table) rather than
		// through expected-path placement.
	},
}
