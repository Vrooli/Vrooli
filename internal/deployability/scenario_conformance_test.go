package deployability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScenarioManifestConformanceRejectsReservedBuilder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service.json")
	if err := os.WriteFile(path, []byte(`{"components":{"api":{"build":{"kind":"cargo"},"run":{"argv":["./api"]}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := CheckScenarioManifest(path)
	if err != nil {
		t.Fatalf("CheckScenarioManifest: %v", err)
	}
	if len(findings) != 1 || findings[0].Rule != "known-builder-kind" || !strings.Contains(findings[0].Message, "reserved builder kind") {
		t.Fatalf("findings = %#v, want reserved builder finding", findings)
	}
}

func TestScenarioManifestRulesAreRegistered(t *testing.T) {
	rules := ScenarioManifestConformanceRules()
	if len(rules) != 4 {
		t.Fatalf("registered manifest rules = %d, want 4", len(rules))
	}
	for _, want := range []string{"known-builder-kind", "no-development-server", "production-ui-artifact", "NO_SHELL_ENTRYPOINT"} {
		found := false
		for _, rule := range rules {
			if rule.Name == want && rule.Check != nil {
				found = true
			}
		}
		if !found {
			t.Fatalf("registered rules missing %q: %#v", want, rules)
		}
	}
}

func TestScenarioManifestConformanceRejectsDevelopmentServer(t *testing.T) {
	path := filepath.Join("testdata", "production-ui-dev-server.json")
	findings, err := CheckScenarioManifest(path)
	if err != nil {
		t.Fatalf("CheckScenarioManifest: %v", err)
	}
	if len(findings) != 1 || findings[0].Rule != "no-development-server" {
		t.Fatalf("findings = %#v, want one no-development-server finding", findings)
	}
	if !strings.Contains(findings[0].Message, "pnpm run dev") {
		t.Fatalf("finding = %q, want offending argv", findings[0].Message)
	}
}

func TestScenarioManifestConformanceAcceptsStaticServer(t *testing.T) {
	path := filepath.Join("testdata", "production-ui-static-server.json")
	findings, err := CheckScenarioManifest(path)
	if err != nil {
		t.Fatalf("CheckScenarioManifest: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("static server findings = %#v, want none", findings)
	}
}

func TestLiveScenarioManifestConformance(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	report, err := CheckScenarioFleet(filepath.Clean(filepath.Join(workingDir, "..", "..")))
	if err != nil {
		t.Fatalf("CheckScenarioFleet: %v", err)
	}
	if report.ManifestCount < 120 {
		t.Fatalf("live manifest count = %d, want at least 120", report.ManifestCount)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("live fleet conformance findings = %#v", report.Findings)
	}
	t.Logf("production UI conformance passed for %d live scenario manifests", report.ManifestCount)
}
