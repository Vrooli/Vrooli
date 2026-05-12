package scenariohandlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	scenariocli "github.com/vrooli/vrooli/internal/cli/scenariocli"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

// Test the analysis pure-function end-to-end: build a fake recorded
// provenance, point it at a real template, then mutate the template and
// confirm we report drift.
func TestAnalyzeDriftForScenarioDetectsContentDrift(t *testing.T) {
	_, info := mustLoadRepoTemplate(t, "react-vite")
	manifestSha, contentSha, err := computeTemplateHashes(info)
	if err != nil {
		t.Fatalf("computeTemplateHashes: %v", err)
	}

	// Snapshot the recorded hashes into a fake scenario manifest, then point
	// loadTemplate at a mutated copy by overriding the repo root.
	tmpDir := copyTemplateDirToTemp(t, info.Path)

	// Build a fake repo layout with templates/scenarios/<name>/ pointing at
	// our tmpDir copy. loadTemplate(root, name) joins templates/scenarios.
	fakeRoot := t.TempDir()
	tmplBase := filepath.Join(fakeRoot, "templates", "scenarios", "react-vite")
	if err := os.MkdirAll(filepath.Dir(tmplBase), 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	if err := os.Rename(tmpDir, tmplBase); err != nil {
		t.Fatalf("place template tree: %v", err)
	}
	tmpDir = tmplBase

	sc := scenariomodel.Scenario{
		Slug: "drift-probe",
		Path: t.TempDir(),
		Manifest: scenariomodel.ServiceManifest{
			Generation: &scenariomodel.GenerationMetadata{
				Template:    scenariomodel.GenerationTemplate{ID: "react-vite", Version: info.Manifest.Version},
				ManifestSha: manifestSha,
				ContentSha:  contentSha,
			},
		},
	}

	// 1) No drift baseline.
	report := analyzeDriftForScenario(fakeRoot, sc, false)
	if report.Drifted() {
		t.Fatalf("baseline reported drift: %+v", report)
	}
	if report.Status != scenariocli.TemplateDriftStatusOK {
		t.Fatalf("baseline status = %q, want %q", report.Status, scenariocli.TemplateDriftStatusOK)
	}

	// 2) Mutate the template — append to README.md or any inherited file.
	target := filepath.Join(tmpDir, "README.md")
	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Skip("template has no README.md to mutate")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if err := os.WriteFile(target, append(data, []byte("\n// drift probe\n")...), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	report = analyzeDriftForScenario(fakeRoot, sc, false)
	if !report.ContentDrifted {
		t.Fatalf("expected ContentDrifted after mutation, got %+v", report)
	}
	if report.ManifestDrifted {
		t.Fatalf("manifest should not drift on content-only edit, got %+v", report)
	}
	if report.Status != scenariocli.TemplateDriftStatusDrifted {
		t.Fatalf("status = %q, want %q", report.Status, scenariocli.TemplateDriftStatusDrifted)
	}
}

func TestAnalyzeDriftForScenarioMissingProvenance(t *testing.T) {
	sc := scenariomodel.Scenario{Slug: "no-provenance", Manifest: scenariomodel.ServiceManifest{}}
	report := analyzeDriftForScenario("", sc, false)
	if report.Status != scenariocli.TemplateDriftStatusNoProvenance {
		t.Fatalf("status = %q, want %q", report.Status, scenariocli.TemplateDriftStatusNoProvenance)
	}
}

func TestAnalyzeDriftForScenarioMissingHashes(t *testing.T) {
	_, info := mustLoadRepoTemplate(t, "react-vite")
	tmpDir := copyTemplateDirToTemp(t, info.Path)
	fakeRoot := t.TempDir()
	tmplBase := filepath.Join(fakeRoot, "templates", "scenarios", "react-vite")
	if err := os.MkdirAll(filepath.Dir(tmplBase), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink(tmpDir, tmplBase); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// Provenance present but no hashes — pre-drift-tracking scenarios.
	sc := scenariomodel.Scenario{
		Slug: "old-scenario",
		Manifest: scenariomodel.ServiceManifest{
			Generation: &scenariomodel.GenerationMetadata{
				Template: scenariomodel.GenerationTemplate{ID: "react-vite", Version: "0.0.1"},
			},
		},
	}
	report := analyzeDriftForScenario(fakeRoot, sc, false)
	if report.Status != scenariocli.TemplateDriftStatusMissingHashes {
		t.Fatalf("status = %q, want %q", report.Status, scenariocli.TemplateDriftStatusMissingHashes)
	}
}

func TestAnalyzeDriftForScenarioTemplateGone(t *testing.T) {
	sc := scenariomodel.Scenario{
		Slug: "phantom",
		Manifest: scenariomodel.ServiceManifest{
			Generation: &scenariomodel.GenerationMetadata{
				Template:    scenariomodel.GenerationTemplate{ID: "this-template-does-not-exist", Version: "1"},
				ManifestSha: "abc",
				ContentSha:  "def",
			},
		},
	}
	report := analyzeDriftForScenario(t.TempDir(), sc, false)
	if report.Status != scenariocli.TemplateDriftStatusTemplateGone {
		t.Fatalf("status = %q, want %q", report.Status, scenariocli.TemplateDriftStatusTemplateGone)
	}
}

// Integration test: generate from a real template, confirm the recorded
// provenance carries non-empty hashes.
func TestGenerationRecordsTemplateHashes(t *testing.T) {
	_, info := mustLoadRepoTemplate(t, "react-vite")

	// Drive buildGenerationProvenance directly. Coverage of full-end-to-end
	// generation already exists in TestBuildTemplateValuesAndCopyTemplate...;
	// here we just want to confirm the hashes are populated in the result
	// struct that's written to .vrooli/service.json.
	prov := buildGenerationProvenance(info, scenariocli.ResolvedDesign{}, time.Unix(1_700_000_000, 0).UTC())
	if prov.ManifestSha == "" {
		t.Fatalf("provenance.ManifestSha empty")
	}
	if prov.ContentSha == "" {
		t.Fatalf("provenance.ContentSha empty")
	}

	// Round-trip through the service-manifest injector to make sure the
	// fields actually land in JSON on disk.
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	stub := scenariomodel.ServiceManifest{Service: scenariomodel.ServiceMetadata{Name: "drift-probe"}}
	stubBytes, _ := json.MarshalIndent(stub, "", "  ")
	if err := os.WriteFile(filepath.Join(dest, ".vrooli", "service.json"), stubBytes, 0o644); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	if err := injectScenarioProvenance(dest, prov); err != nil {
		t.Fatalf("inject: %v", err)
	}
	on, err := os.ReadFile(filepath.Join(dest, ".vrooli", "service.json"))
	if err != nil {
		t.Fatalf("read service.json: %v", err)
	}
	var out scenariomodel.ServiceManifest
	if err := json.Unmarshal(on, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Generation == nil {
		t.Fatalf("generation block missing after inject")
	}
	if out.Generation.ManifestSha == "" || out.Generation.ContentSha == "" {
		t.Fatalf("hashes missing in persisted manifest: %+v", out.Generation)
	}
}
