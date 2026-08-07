package targetpack

import (
	"os"
	"path/filepath"
	"testing"

	"structure-health/internal/rules"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func codes(findings []rules.Finding) map[string]bool {
	out := map[string]bool{}
	for _, finding := range findings {
		out[finding.Code] = true
	}
	return out
}

func TestResourcePackPositive(t *testing.T) {
	root := t.TempDir()
	write(t, root, "resource.json", `{"name":"demo","runtime":{"image":"example/demo@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},"health_checks":[{"kind":"readiness"}]}`)
	got := codes(Evaluate("resource", root, "demo"))
	if len(got) != 0 {
		t.Fatalf("conforming resource findings = %v", got)
	}
}

func TestResourcePackNegativeRules(t *testing.T) {
	root := t.TempDir()
	write(t, root, "resource.json", `{"name":"demo","runtime":{"image":"example/demo:latest"},"health_checks":[{"type":"http"}]}`)
	write(t, root, "legacy.sh", "#!/bin/sh\n")
	write(t, root, "docker/compose.yaml", "services:\n  demo:\n    image: example/demo:latest\n")
	got := codes(Evaluate("resource", root, "demo"))
	for _, want := range []string{"RESOURCE_HEALTH_KIND_MISSING", "RESOURCE_SHELL_FORBIDDEN", "RESOURCE_IMAGE_UNPINNED"} {
		if !got[want] {
			t.Fatalf("missing %s in %v", want, got)
		}
	}
}

func TestResourcePackRejectsInvalidManifest(t *testing.T) {
	root := t.TempDir()
	write(t, root, "resource.json", "{")
	if got := codes(Evaluate("resource", root, "demo")); !got["RESOURCE_MANIFEST_INVALID"] {
		t.Fatalf("expected invalid resource manifest, got %v", got)
	}
}

func TestToolPackPositiveAndNegative(t *testing.T) {
	root := t.TempDir()
	write(t, root, "tool.json", `{"name":"demo","description":"demo tool","commands":["demo"],"versionArgs":["--version"],"bundling":"host-required"}`)
	if got := Evaluate("tool", root, "demo"); len(got) != 0 {
		t.Fatalf("conforming tool findings = %v", got)
	}
	write(t, root, "tool.json", `{"name":"other","description":"","commands":[],"versionArgs":[],"bundling":""}`)
	got := codes(Evaluate("tool", root, "demo"))
	if !got["TOOL_NAME_MISMATCH"] || !got["TOOL_MANIFEST_INVALID"] {
		t.Fatalf("expected tool identity and manifest findings, got %v", got)
	}
}

func TestToolPackRequiresDeclaredHandlerFile(t *testing.T) {
	root := t.TempDir()
	write(t, root, "tool.json", `{"name":"demo","description":"demo","commands":["demo"],"versionArgs":["--version"],"bundling":"host-required","handler":"demo"}`)
	got := codes(Evaluate("tool", root, "demo"))
	if !got["TOOL_HANDLER_MISSING"] {
		t.Fatalf("expected missing handler, got %v", got)
	}
}

func TestSafeguardPackPositiveAndNegative(t *testing.T) {
	root := t.TempDir()
	write(t, root, "safeguard.json", `{"name":"demo","description":"demo safeguard","handler":"demo","privilege":"user","bundling":"prohibited","deployment":{"profiles":{}}}`)
	write(t, root, "handler.go", "package demo\n")
	if got := Evaluate("safeguard", root, "demo"); len(got) != 0 {
		t.Fatalf("conforming safeguard findings = %v", got)
	}
	write(t, root, "safeguard.json", `{"name":"other","description":"","handler":"demo","privilege":"","bundling":"","deployment":null}`)
	got := codes(Evaluate("safeguard", root, "demo"))
	if !got["SAFEGUARD_NAME_MISMATCH"] || !got["SAFEGUARD_MANIFEST_INVALID"] {
		t.Fatalf("expected safeguard identity and manifest findings, got %v", got)
	}
}

func TestSafeguardPackRequiresDeclaredHandlerFile(t *testing.T) {
	root := t.TempDir()
	write(t, root, "safeguard.json", `{"name":"demo","description":"demo","handler":"demo","privilege":"user","bundling":"prohibited","deployment":{"profiles":{}}}`)
	got := codes(Evaluate("safeguard", root, "demo"))
	if !got["SAFEGUARD_HANDLER_MISSING"] {
		t.Fatalf("expected missing handler, got %v", got)
	}
}

func TestTargetPackDoesNotLeakAcrossKinds(t *testing.T) {
	root := t.TempDir()
	write(t, root, "resource.json", `{"name":"demo"}`)
	if got := Evaluate("unknown", root, "demo"); len(got) != 0 {
		t.Fatalf("unknown kind findings = %v", got)
	}
}

func TestPackagePackPositiveAndNegative(t *testing.T) {
	root := t.TempDir()
	write(t, root, ".vrooli/package.json", `{"$schema":"schemas/package.schema.json","version":"1.0.0","package":{"name":"demo","kind":"go_runtime","module_identifiers":["example/demo"],"adoption":{"scenario_adoptable":true,"allowed_consumers":[],"adoption_modes":[]},"lifecycle":{},"refresh":{"strategy":"none","restart_running_consumers":false}}}`)
	write(t, root, "README.md", "# demo\n")
	write(t, root, "go.mod", "module example/demo\n\ngo 1.25\n")
	if got := Evaluate("package", root, "demo"); len(got) != 0 {
		t.Fatalf("conforming package findings = %v", got)
	}
	write(t, root, ".vrooli/package.json", `{"$schema":"schemas/package.schema.json","version":"1.0.0","package":{"name":"other","kind":"go_runtime","module_identifiers":["example/other"]}}`)
	got := codes(Evaluate("package", root, "demo"))
	for _, want := range []string{"PACKAGE_NAME_MISMATCH", "PACKAGE_MANIFEST_INVALID", "PACKAGE_MODULE_PATH_MISMATCH"} {
		if !got[want] {
			t.Fatalf("missing %s in %v", want, got)
		}
	}
}

func TestPackagePackRequiresManifest(t *testing.T) {
	if got := codes(Evaluate("package", t.TempDir(), "demo")); !got["PACKAGE_MANIFEST_MISSING"] {
		t.Fatalf("expected missing package manifest, got %v", got)
	}
}

func TestControlPlanePackPositiveAndNegative(t *testing.T) {
	root := t.TempDir()
	write(t, root, "main.go", "package main\n")
	if got := Evaluate("control-plane", root, "internal"); len(got) != 0 {
		t.Fatalf("conforming control-plane findings = %v", got)
	}
	if got := codes(Evaluate("control-plane", t.TempDir(), "internal")); !got["CONTROL_PLANE_LAYOUT_MISSING"] {
		t.Fatalf("expected control-plane layout finding, got %v", got)
	}
}

func TestDocsPackPositiveAndNegative(t *testing.T) {
	root := t.TempDir()
	write(t, root, "manifest.json", `{"version":"1.0.0","title":"Docs","sections":[{"id":"start"}]}`)
	write(t, root, "README.md", "# Docs\n")
	if got := Evaluate("docs", root, "docs"); len(got) != 0 {
		t.Fatalf("conforming docs findings = %v", got)
	}
	write(t, root, "manifest.json", `{"title":"Docs"}`)
	got := codes(Evaluate("docs", root, "docs"))
	if !got["DOCS_MANIFEST_INVALID"] {
		t.Fatalf("expected docs manifest finding, got %v", got)
	}
}

func TestTeamPackPositiveAndNegative(t *testing.T) {
	root := t.TempDir()
	write(t, root, "manifest.json", `{"contract":{"team":"demo-team"},"sections":[{"id":"start"}]}`)
	write(t, root, "README.md", "# Team\n")
	if got := Evaluate("team", root, "demo-team"); len(got) != 0 {
		t.Fatalf("conforming team findings = %v", got)
	}
	write(t, root, "manifest.json", `{"contract":{"team":"other"},"sections":[]}`)
	got := codes(Evaluate("team", root, "demo-team"))
	if !got["TEAM_OWNER_MISMATCH"] {
		t.Fatalf("expected team owner finding, got %v", got)
	}
}
