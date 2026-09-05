package providerconformance

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func topCapabilityLevel(descriptor map[string]any) map[string]any {
	caps := descriptor["maturity"].(map[string]any)["capabilities"].([]any)
	levels := caps[0].(map[string]any)["levels"].([]any)
	return levels[len(levels)-1].(map[string]any)
}

func firstCapabilityLevel(descriptor map[string]any) map[string]any {
	caps := descriptor["maturity"].(map[string]any)["capabilities"].([]any)
	levels := caps[0].(map[string]any)["levels"].([]any)
	return levels[0].(map[string]any)
}

func requireSeverity(t *testing.T, f Finding, want Severity) {
	t.Helper()
	if f.Severity != want {
		t.Fatalf("finding %s severity = %q, want %q", f.Code, f.Severity, want)
	}
}

func requireNoCode(t *testing.T, report Report, code string) {
	t.Helper()
	for _, f := range report.Findings {
		if f.Code == code {
			t.Fatalf("finding %s should be absent; got findings %v", code, findingCodes(report))
		}
	}
}

func TestContractChecksSilentWhenConformant(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", nil)
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	for _, code := range []string{CodeDocsSkeletonIncomplete, CodeNorthStarMissing, CodeLadderIncomplete} {
		requireNoCode(t, report, code)
	}
}

func TestNorthStarMissingGates(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", func(d map[string]any) {
		delete(topCapabilityLevel(d), "capability_summary")
	})
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireSeverity(t, requireCode(t, report, CodeNorthStarMissing), SeverityError)
	// Gating: a north-star gap now fails the phase.
	if report.Summary.Status() != "failed" {
		t.Fatalf("north-star gap must gate; status = %q", report.Summary.Status())
	}
}

func TestLadderIncompleteGates(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", func(d map[string]any) {
		delete(firstCapabilityLevel(d), "next_unlock")
	})
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireSeverity(t, requireCode(t, report, CodeLadderIncomplete), SeverityError)
	if report.Summary.Status() != "failed" {
		t.Fatalf("ladder gap must gate; status = %q", report.Summary.Status())
	}
}

func TestUngatedRungIsAdvisoryDuringFleetRemediation(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", func(d map[string]any) {
		maturity := d["maturity"].(map[string]any)
		maturity["findings"] = map[string]any{}
	})
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireSeverity(t, requireCode(t, report, CodeRungUngated), SeverityWarning)
	if report.Summary.Status() != "passed" {
		t.Fatalf("ungated rung is advisory during fleet remediation; status = %q", report.Summary.Status())
	}
}

func TestDocsSkeletonIncompleteGates(t *testing.T) {
	repoRoot, scenarioDir := fixtureRepo(t, "demo-provider", nil)
	// Overwrite the doc with one missing the required headings.
	if err := os.WriteFile(filepath.Join(scenarioDir, "README.md"), []byte("# fixture\n\n## Overview\nno skeleton here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireSeverity(t, requireCode(t, report, CodeDocsSkeletonIncomplete), SeverityError)
	if report.Summary.Status() != "failed" {
		t.Fatalf("doc-skeleton gap must gate; status = %q", report.Summary.Status())
	}
}

// TestBrokenFixtureIsGated proves the Phase Capability Contract is now gating: a
// deliberately-broken provider descriptor (no north star, incomplete ladder,
// non-skeleton doc) fails provider-conformance.
func TestBrokenFixtureIsGated(t *testing.T) {
	repoRoot, scenarioDir := fixtureRepo(t, "demo-provider", func(d map[string]any) {
		delete(topCapabilityLevel(d), "capability_summary")
		delete(firstCapabilityLevel(d), "next_unlock")
	})
	if err := os.WriteFile(filepath.Join(scenarioDir, "README.md"), []byte("# no skeleton\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	for _, code := range []string{CodeDocsSkeletonIncomplete, CodeNorthStarMissing, CodeLadderIncomplete} {
		requireCode(t, report, code)
	}
	if report.Summary.Status() != "failed" {
		t.Fatalf("a broken contract must gate (fail), got status %q", report.Summary.Status())
	}
	if report.Summary.Errors == 0 {
		t.Fatal("gating checks must raise errors")
	}
}
