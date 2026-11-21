// Package protolabel owns the CLI's shared labels for protobuf enums.
package protolabel

import (
	domainsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
)

// Archetype returns the canonical lowercase label, preserving an unrecognized
// declared label for forward-compatible CLI output.
func Archetype(a *domainsv1.DomainArchetype) string {
	switch a.GetArchetype() {
	case domainsv1.Archetype_ARCHETYPE_REPORTING:
		return "reporting"
	case domainsv1.Archetype_ARCHETYPE_SERVICE:
		return "service"
	case domainsv1.Archetype_ARCHETYPE_MUTATION:
		return "mutation"
	case domainsv1.Archetype_ARCHETYPE_CLASSIFICATION:
		return "classification"
	case domainsv1.Archetype_ARCHETYPE_ORCHESTRATION:
		return "orchestration"
	case domainsv1.Archetype_ARCHETYPE_SCORING:
		return "scoring"
	case domainsv1.Archetype_ARCHETYPE_QUERY:
		return "query"
	default:
		return a.GetDeclaredLabel()
	}
}

// FindingClass returns the stable CLI label for a conflict classification.
func FindingClass(c sharedv1.FindingClass) string {
	switch c {
	case sharedv1.FindingClass_FINDING_CLASS_DETERMINISTIC:
		return "deterministic"
	case sharedv1.FindingClass_FINDING_CLASS_HEURISTIC:
		return "heuristic"
	default:
		return "unspecified"
	}
}
