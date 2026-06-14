package interfacegraph

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDriftDetectorDetectsAsymmetricDrift(t *testing.T) {
	root := t.TempDir()
	writeServiceConfig(t, root, "alpha", []string{"charlie"})
	writeServiceConfig(t, root, "bravo", nil)
	writeServiceConfig(t, root, "charlie", nil)

	builder := NewBuilder(
		fakeProtoClient{resp: &ProtoSurfaceResponse{Results: []ProtoSurfaceResult{
			{Scenario: "alpha", Surface: ProtoSurface{Scenario: "alpha"}},
			{Scenario: "bravo", Surface: ProtoSurface{Scenario: "bravo"}},
			{Scenario: "charlie", Surface: ProtoSurface{Scenario: "charlie"}},
		}}},
		fakeImportClient{resp: &ImportFactsResponse{Results: []ImportFactsResult{
			{
				Scenario: "alpha",
				Facts: []ImportFact{
					{ImportPath: "github.com/vrooli/vrooli/packages/proto/gen/go/bravo/v1/bravo"},
				},
			},
		}}},
	)
	detector := NewDriftDetector(builder, filepath.Join(root, "scenarios"))

	report, err := detector.Detect(context.Background(), BuildRequest{Scenarios: []string{"alpha", "bravo", "charlie"}})
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(report.Findings) != 2 {
		t.Fatalf("findings = %d, want 2: %#v", len(report.Findings), report.Findings)
	}

	var undeclared, declaredOnly *DriftFinding
	for i := range report.Findings {
		finding := &report.Findings[i]
		switch finding.Kind {
		case DriftUndeclaredUsed:
			undeclared = finding
		case DriftDeclaredWithoutProof:
			declaredOnly = finding
		}
	}
	if undeclared == nil || undeclared.Scenario != "alpha" || undeclared.Dependency != "bravo" || undeclared.Severity != SeverityWarning {
		t.Fatalf("undeclared finding = %#v", undeclared)
	}
	if len(undeclared.Evidence) == 0 || undeclared.Evidence[0].Source != EvidenceGoImport {
		t.Fatalf("undeclared evidence = %#v", undeclared.Evidence)
	}
	if declaredOnly == nil || declaredOnly.Scenario != "alpha" || declaredOnly.Dependency != "charlie" || declaredOnly.Severity != SeverityInfo {
		t.Fatalf("declared-only finding = %#v", declaredOnly)
	}
	if declaredOnly.ActualEvidence {
		t.Fatalf("declared-only ActualEvidence = true, want false")
	}
}

func writeServiceConfig(t *testing.T, root, scenario string, deps []string) {
	t.Helper()
	dir := filepath.Join(root, "scenarios", scenario, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir service dir: %v", err)
	}
	content := `{"name":"` + scenario + `","dependencies":{"scenarios":{`
	for i, dep := range deps {
		if i > 0 {
			content += ","
		}
		content += `"` + dep + `":{"enabled":true,"required":true}`
	}
	content += `}}}`
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write service config: %v", err)
	}
}
