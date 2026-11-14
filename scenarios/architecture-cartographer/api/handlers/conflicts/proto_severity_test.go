package conflicts

import (
	"testing"

	"architecture-cartographer/internal/conflicts"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
)

func TestSeverity(t *testing.T) {
	for _, test := range []struct {
		name string
		in   conflicts.Severity
		want sharedv1.Severity
	}{
		{name: "info", in: conflicts.SeverityInfo, want: sharedv1.Severity_SEVERITY_INFO},
		{name: "warn", in: conflicts.SeverityWarn, want: sharedv1.Severity_SEVERITY_WARN},
		{name: "error", in: conflicts.SeverityError, want: sharedv1.Severity_SEVERITY_ERROR},
		{name: "blocker", in: conflicts.SeverityBlocker, want: sharedv1.Severity_SEVERITY_BLOCKER},
		{name: "unspecified", in: conflicts.SeverityUnspecified, want: sharedv1.Severity_SEVERITY_UNSPECIFIED},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := SeverityToProto(test.in); got != test.want {
				t.Fatalf("Severity(%v) = %v, want %v", test.in, got, test.want)
			}
		})
	}
}

func TestFindingClassToProto(t *testing.T) {
	for _, test := range []struct {
		name string
		in   conflicts.FindingClass
		want sharedv1.FindingClass
	}{
		{name: "deterministic", in: conflicts.FindingClassDeterministic, want: sharedv1.FindingClass_FINDING_CLASS_DETERMINISTIC},
		{name: "heuristic", in: conflicts.FindingClassHeuristic, want: sharedv1.FindingClass_FINDING_CLASS_HEURISTIC},
		{name: "unspecified", in: conflicts.FindingClassUnspecified, want: sharedv1.FindingClass_FINDING_CLASS_UNSPECIFIED},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := FindingClassToProto(test.in); got != test.want {
				t.Fatalf("FindingClassToProto(%v) = %v, want %v", test.in, got, test.want)
			}
		})
	}
}
