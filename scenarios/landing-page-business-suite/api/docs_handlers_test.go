package main

import (
	"path/filepath"
	"testing"
)

func TestGetDocsRoot_PrefersExplicitOverride(t *testing.T) {
	t.Setenv("SCENARIO_ROOT", "/ignored/scenario")
	t.Setenv("DOCS_ROOT", "custom-docs")

	got := getDocsRoot()
	want, err := filepath.Abs("custom-docs")
	if err != nil {
		t.Fatalf("resolve expected docs root: %v", err)
	}
	if got != want {
		t.Fatalf("expected explicit docs root %q, got %q", want, got)
	}
}

func TestGetDocsRoot_DerivesFromScenarioRoot(t *testing.T) {
	t.Setenv("DOCS_ROOT", "")
	t.Setenv("SCENARIO_ROOT", "/srv/landing-page-business-suite")

	if got, want := getDocsRoot(), "/srv/landing-page-business-suite/docs"; got != want {
		t.Fatalf("expected scenario docs root %q, got %q", want, got)
	}
}

func TestGetDocsRoot_DefaultIsAbsolute(t *testing.T) {
	t.Setenv("DOCS_ROOT", "")
	t.Setenv("SCENARIO_ROOT", "")

	got := getDocsRoot()
	if !filepath.IsAbs(got) {
		t.Fatalf("expected default docs root to be absolute, got %q", got)
	}
	if filepath.Base(got) != "docs" {
		t.Fatalf("expected default docs directory, got %q", got)
	}
}

func TestDocsConnectDependencies_InstallsEveryBoundaryAdapter(t *testing.T) {
	deps := docsConnectDependencies()
	if deps.DocsRoot == nil || deps.Log == nil {
		t.Fatal("expected document Connect composition to install every boundary adapter")
	}
}
