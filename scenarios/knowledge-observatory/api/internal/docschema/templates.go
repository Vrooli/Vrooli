package docschema

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed templates/*.md
var templateFS embed.FS

// TemplateInfo holds metadata about a doc template.
type TemplateInfo struct {
	Filename string
	Purpose  string
}

// templateDocTypes maps doc types that have templates to their metadata.
var templateDocTypes = map[DocType]TemplateInfo{
	DocTypeSeams:           {"SEAMS.md", "Integration boundaries, responsibility zones, decision points, testability"},
	DocTypeProblems:        {"PROBLEMS.md", "Known issues, tech debt, test gaps, UX issues, cleanup history"},
	DocTypeProgress:        {"PROGRESS.md", "Development history, completed milestones"},
	DocTypeInvariants:      {"INVARIANTS.md", "System contracts that must never be violated"},
	DocTypeAssumptions:     {"ASSUMPTIONS.md", "Implicit beliefs not yet validated"},
	DocTypeErrorSemantics:  {"ERROR-SEMANTICS.md", "Error categories, recovery paths, user messaging"},
	DocTypeSecurityPosture: {"SECURITY-POSTURE.md", "Security hardening status by category"},
	DocTypeTemporalFlows:   {"TEMPORAL-FLOWS.md", "Async patterns, race conditions, checkpoint flows"},
	DocTypeCoherenceNotes:  {"COHERENCE-NOTES.md", "React state patterns, duplication, styling audit"},
	DocTypeExperienceAudit: {"EXPERIENCE-AUDIT.md", "Persona mapping, friction analysis, navigation"},
}

// TemplateForDocType returns the template content for the given doc type.
func TemplateForDocType(dt DocType) (string, error) {
	info, ok := templateDocTypes[dt]
	if !ok {
		return "", fmt.Errorf("no template for doc type: %s", dt)
	}
	data, err := templateFS.ReadFile("templates/" + info.Filename)
	if err != nil {
		return "", fmt.Errorf("reading template %s: %w", info.Filename, err)
	}
	return string(data), nil
}

// TemplatePurpose returns the purpose description for a doc type's template,
// or empty string if the doc type has no template.
func TemplatePurpose(dt DocType) string {
	if info, ok := templateDocTypes[dt]; ok {
		return info.Purpose
	}
	return ""
}

// HasTemplate reports whether the given doc type has a template.
func HasTemplate(dt DocType) bool {
	_, ok := templateDocTypes[dt]
	return ok
}

// ListTemplateDocTypes returns all doc types that have templates, sorted alphabetically.
func ListTemplateDocTypes() []DocType {
	types := make([]DocType, 0, len(templateDocTypes))
	for dt := range templateDocTypes {
		types = append(types, dt)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i] < types[j]
	})
	return types
}

// TemplateFilename returns the expected filename for a doc type's template,
// or empty string if the doc type has no template.
func (dt DocType) TemplateFilename() string {
	if info, ok := templateDocTypes[dt]; ok {
		return info.Filename
	}
	// Derive from expected path as fallback.
	ep := dt.ExpectedPath()
	if ep == "" {
		return ""
	}
	parts := strings.Split(ep, "/")
	return parts[len(parts)-1]
}
