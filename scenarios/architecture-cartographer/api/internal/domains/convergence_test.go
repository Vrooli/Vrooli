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
