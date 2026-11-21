package conflicts

import (
	"architecture-cartographer/internal/conflicts"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
)

// Severity maps conflict-domain severity to the shared API enum.
func SeverityToProto(value conflicts.Severity) sharedv1.Severity {
	switch value {
	case conflicts.SeverityInfo:
		return sharedv1.Severity_SEVERITY_INFO
	case conflicts.SeverityWarn:
		return sharedv1.Severity_SEVERITY_WARN
	case conflicts.SeverityError:
		return sharedv1.Severity_SEVERITY_ERROR
	case conflicts.SeverityBlocker:
		return sharedv1.Severity_SEVERITY_BLOCKER
	default:
		return sharedv1.Severity_SEVERITY_UNSPECIFIED
	}
}

// FindingClassToProto maps conflict-domain classification to the shared API enum.
func FindingClassToProto(value conflicts.FindingClass) sharedv1.FindingClass {
	switch value {
	case conflicts.FindingClassDeterministic:
		return sharedv1.FindingClass_FINDING_CLASS_DETERMINISTIC
	case conflicts.FindingClassHeuristic:
		return sharedv1.FindingClass_FINDING_CLASS_HEURISTIC
	default:
		return sharedv1.FindingClass_FINDING_CLASS_UNSPECIFIED
	}
}
