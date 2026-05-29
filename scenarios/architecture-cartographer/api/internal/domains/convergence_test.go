package domains

import (
	"testing"
	"time"
)

// buildMapWith constructs a resolved map from per-source name lists for
// convergence testing.
func buildMapWith(t *testing.T, doc, folders, cli, ui []string) DerivedDomainMap {
	t.Helper()
	mk := func(src Source, names []string) Extraction {
		ex := Extraction{Source: src}
		for _, n := range names {
			ex.Domains = append(ex.Domains, ExtractedDomain{Name: n, Paths: []string{n + "/"}})
		}
		return ex
	}
	exts := []Extraction{
		mk(SourceDomainsDoc, doc),
		mk(SourceAPIFolders, folders),
		mk(SourceCLIGroups, cli),
		mk(SourceUIFeatures, ui),
	}
	m, err := Resolve("x", exts, time.Time{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	return m
}

func findingsByKind(fs []ConvergenceFinding) map[string][]string {
	out := map[string][]string{}
	for _, f := range fs {
		out[f.Kind] = append(out[f.Kind], f.Domain)
	}
	return out
}

func TestConvergence_FullAgreement(t *testing.T) {
	m := buildMapWith(t, []string{"graph", "conflicts"}, []string{"graph", "conflicts"}, []string{"graph", "conflicts"}, []string{"graph", "conflicts"})
	if got := Convergence(m); len(got) != 0 {
		t.Fatalf("expected no findings on full agreement, got %+v", got)
	}
}

func TestConvergence_MissingImplementationAndCLI(t *testing.T) {
	// "conflicts" is declared in the doc but missing from folders and CLI.
	m := buildMapWith(t, []string{"graph", "conflicts"}, []string{"graph"}, []string{"graph"}, nil)
	byKind := findingsByKind(Convergence(m))
	if got := byKind[FindingMissingImplementation]; len(got) != 1 || got[0] != "conflicts" {
		t.Fatalf("missing_implementation = %v, want [conflicts]", got)
	}
	if got := byKind[FindingMissingCLIGroup]; len(got) != 1 || got[0] != "conflicts" {
		t.Fatalf("missing_cli_group = %v, want [conflicts]", got)
	}
}

func TestConvergence_HealthDomainNoCLIIsNotFP(t *testing.T) {
	// Regression for L5-readiness Phase 7: a scenario that declares a
	// "health" domain inherits the cli-core built-in group; missing it
	// from the per-scenario manifest must NOT produce missing_cli_group.
	m := buildMapWith(t,
		[]string{"graph", "health"},
		[]string{"graph", "health"},
		[]string{"graph"}, // no "health" in cli manifest, on purpose
		nil)
	byKind := findingsByKind(Convergence(m))
	if got := byKind[FindingMissingCLIGroup]; len(got) != 0 {
		t.Fatalf("cli-core builtin domain must not raise missing_cli_group: %v", got)
	}
}

func TestConvergence_UndeclaredFolder(t *testing.T) {
	// "rogue" folder exists but the doc never declared it.
	m := buildMapWith(t, []string{"graph"}, []string{"graph", "rogue"}, []string{"graph"}, nil)
	byKind := findingsByKind(Convergence(m))
	if got := byKind[FindingUndeclaredFolder]; len(got) != 1 || got[0] != "rogue" {
		t.Fatalf("undeclared_folder = %v, want [rogue]", got)
	}
}

func TestConvergence_UIFeatureNoDomainIsAdvisory(t *testing.T) {
	// "widgets" UI feature maps to no declared domain.
	m := buildMapWith(t, []string{"graph"}, []string{"graph"}, []string{"graph"}, []string{"graph", "widgets"})
	var found *ConvergenceFinding
	for i := range Convergence(m) {
		f := Convergence(m)[i]
		if f.Kind == FindingUIFeatureNoDomain {
			found = &f
		}
	}
	if found == nil || found.Domain != "widgets" {
		t.Fatalf("expected ui_feature_no_domain for widgets, got %+v", Convergence(m))
	}
	if found.Severity != ConvergenceInfo {
		t.Fatalf("UI finding must be advisory (info), got %q", found.Severity)
	}
}

func TestConvergence_NoSurfaceNoFalsePositive(t *testing.T) {
	// Docs-only scenario: no folders/cli/ui declared. Must not flag every
	// domain as missing.
	m := buildMapWith(t, []string{"graph", "conflicts"}, nil, nil, nil)
	if got := Convergence(m); len(got) != 0 {
		t.Fatalf("docs-only scenario must produce no findings, got %+v", got)
	}
}

func TestConvergence_RollupMergesInfoFindings(t *testing.T) {
	// Three domains declared in the doc, only one in CLI groups. The CLI
	// gap is info-severity for each missing domain; with count >= 2 the
	// three rows must collapse into one rolled-up finding.
	m := buildMapWith(t,
		[]string{"alpha", "beta", "gamma", "delta"},
		[]string{"alpha", "beta", "gamma", "delta"},
		[]string{"alpha"},
		nil)
	var cliFindings []ConvergenceFinding
	for _, f := range Convergence(m) {
		if f.Kind == FindingMissingCLIGroup {
			cliFindings = append(cliFindings, f)
		}
	}
	if len(cliFindings) != 1 {
		t.Fatalf("expected 1 rolled-up missing_cli_group finding, got %d (%+v)", len(cliFindings), cliFindings)
	}
	f := cliFindings[0]
	if f.Domain != "" {
		t.Fatalf("rolled-up finding must have empty Domain, got %q", f.Domain)
	}
	want := []string{"beta", "delta", "gamma"}
	if len(f.RolledUpDomains) != len(want) {
		t.Fatalf("RolledUpDomains = %v, want %v", f.RolledUpDomains, want)
	}
	for i, n := range want {
		if f.RolledUpDomains[i] != n {
			t.Fatalf("RolledUpDomains[%d] = %q, want %q", i, f.RolledUpDomains[i], n)
		}
	}
}

func TestConvergence_AuthorityFallbackOnLowConfidence(t *testing.T) {
	// No DOMAINS.md → authority falls back to api folders, confidence=low.
	m := buildMapWith(t, nil, []string{"graph"}, nil, nil)
	if m.AuthorityConfidence != ConfidenceLow {
		t.Fatalf("authority confidence = %q, want low", m.AuthorityConfidence)
	}
	var fallback *ConvergenceFinding
	for i, f := range Convergence(m) {
		if f.Kind == FindingAuthorityFallback {
			f := Convergence(m)[i]
			fallback = &f
		}
	}
	if fallback == nil {
		t.Fatalf("expected authority_fallback finding, got %+v", Convergence(m))
	}
	if fallback.Severity != ConvergenceInfo {
		t.Fatalf("authority_fallback severity = %q, want info", fallback.Severity)
	}
}

func TestConvergence_HighConfidenceWhenDomainsDocAuthority(t *testing.T) {
	m := buildMapWith(t, []string{"graph"}, []string{"graph"}, []string{"graph"}, nil)
	if m.AuthorityConfidence != ConfidenceHigh {
		t.Fatalf("authority confidence = %q, want high", m.AuthorityConfidence)
	}
	for _, f := range Convergence(m) {
		if f.Kind == FindingAuthorityFallback {
			t.Fatalf("high-confidence map must NOT emit authority_fallback, got %+v", f)
		}
	}
}
