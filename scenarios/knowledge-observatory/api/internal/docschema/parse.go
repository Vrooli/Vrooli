package docschema

import (
	"fmt"
	"strings"
)

// ParseDocType normalizes a string into a DocType.
func ParseDocType(value string) (DocType, error) {
	clean := strings.TrimSpace(strings.ToLower(value))
	switch clean {
	case "problems", "problem":
		return DocTypeProblems, nil
	case "progress":
		return DocTypeProgress, nil
	case "seams", "seam":
		return DocTypeSeams, nil
	case "invariants", "invariant":
		return DocTypeInvariants, nil
	case "assumptions", "assumption":
		return DocTypeAssumptions, nil
	case "error-semantics", "error_semantics", "errorsemantics":
		return DocTypeErrorSemantics, nil
	case "security-posture", "security_posture", "securityposture":
		return DocTypeSecurityPosture, nil
	case "temporal-flows", "temporal_flows", "temporalflows":
		return DocTypeTemporalFlows, nil
	case "coherence-notes", "coherence_notes", "coherencenotes":
		return DocTypeCoherenceNotes, nil
	case "experience-audit", "experience_audit", "experienceaudit":
		return DocTypeExperienceAudit, nil
	case "quickstart":
		return DocTypeQuickstart, nil
	case "architecture":
		return DocTypeArchitecture, nil
	case "glossary":
		return DocTypeGlossary, nil
	case "prd":
		return DocTypePRD, nil
	case "readme":
		return DocTypeReadme, nil
	case "manifest":
		return DocTypeManifest, nil
	default:
		return "", fmt.Errorf("unknown doc type: %s", value)
	}
}
