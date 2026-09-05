package hostinventory

import "testing"

func TestAuditReportsEachFindingKind(t *testing.T) {
	report := AuditArtifacts(
		[]DeclaredArtifact{{Owner: "owner", Path: "/declared", ContentHash: "expected"}, {Owner: "owner", Path: "/missing"}},
		[]ObservedArtifact{{Path: "/declared", ContentHash: "actual"}, {Path: "/unmanaged"}},
	)
	seen := map[AuditKind]bool{}
	for _, finding := range report.Findings {
		seen[finding.Kind] = true
	}
	for _, kind := range []AuditKind{AuditContentDrift, AuditMissingDeclaration, AuditUndeclaredMutation} {
		if !seen[kind] {
			t.Fatalf("findings = %+v, missing %s", report.Findings, kind)
		}
	}
}
