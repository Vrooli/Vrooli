package dependencyhealth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIntegrationConformanceAcceptsDeclaredRegistryContract(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "consumer")
	dependency := filepath.Join(root, "audio-tools")
	if err := os.MkdirAll(filepath.Join(target, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "api", "internal", "capabilities"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dependency, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"dependencies":{"scenarios":{"audio-tools":{"required":false,"degraded_behavior":"Voice controls remain unavailable until audio-tools recovers."}}}}`)
	if err := os.WriteFile(filepath.Join(target, ".vrooli", "service.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	registry := []byte(`package capabilities
type Checker interface { Check() }
type Def struct { Description string; DependencySlug string; ActionKind string }
var Known = []Def{{Description: "shared voice capability", ActionKind: "scenario_start"}}
var audioTools = "audio-tools"
`)
	if err := os.WriteFile(filepath.Join(target, "api", "internal", "capabilities", "registry.go"), registry, 0o644); err != nil {
		t.Fatal(err)
	}

	h := &connectHandler{scenariosDir: func() string { return root }}
	section, findings, _ := h.evaluateIntegrationConformance(withScenarioPath(context.Background(), target), "consumer")
	if len(findings) != 0 {
		t.Fatalf("expected conformance pass, got %v", findings)
	}
	if section.GetStatus() != "pass" {
		t.Fatalf("expected pass section, got %q", section.GetStatus())
	}
}

func TestIntegrationConformanceReportsMissingRegistryEntry(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "consumer")
	dependency := filepath.Join(root, "audio-tools")
	if err := os.MkdirAll(filepath.Join(target, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "api", "internal", "capabilities"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dependency, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"dependencies":{"scenarios":{"audio-tools":{"required":false,"degraded_behavior":"Voice controls remain unavailable until audio-tools recovers."}}}}`)
	if err := os.WriteFile(filepath.Join(target, ".vrooli", "service.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "api", "internal", "capabilities", "registry.go"), []byte("package capabilities\n// audio-tools is intentionally absent from this registry.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := &connectHandler{scenariosDir: func() string { return root }}
	_, findings, _ := h.evaluateIntegrationConformance(withScenarioPath(context.Background(), target), "consumer")
	if len(findings) != 1 || findings[0].GetRuleId() != "INTEGRATION_REGISTRY_ENTRY_MISSING" {
		t.Fatalf("expected missing-entry finding, got %v", findings)
	}
}

func TestIntegrationConformanceIsNotApplicableWithoutScenarioDependencies(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "standalone")
	if err := os.MkdirAll(filepath.Join(target, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, ".vrooli", "service.json"), []byte(`{"dependencies":{"scenarios":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	h := &connectHandler{scenariosDir: func() string { return root }}
	section, findings, degraded := h.evaluateIntegrationConformance(withScenarioPath(context.Background(), target), "standalone")
	if len(findings) != 0 || len(degraded) != 0 {
		t.Fatalf("expected no findings or degraded dependencies, got findings=%v degraded=%v", findings, degraded)
	}
	if section.GetStatus() != "not_applicable" {
		t.Fatalf("status = %q, want not_applicable", section.GetStatus())
	}
}
