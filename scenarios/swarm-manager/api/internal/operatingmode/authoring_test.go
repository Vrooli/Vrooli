package operatingmode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScaffoldValidateSimulateRoundTrip proves the self-serve authoring loop:
// scaffold writes a valid mode folder, ValidateModeDraft loads it clean from
// disk, and SimulateModeDraft walks its real guards to the terminal phase —
// all before the mode is registered and with no restart.
func TestScaffoldValidateSimulateRoundTrip(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	result, err := svc.ScaffoldMode(ScaffoldRequest{ID: "demo-mode", Label: "Demo Mode"})
	if err != nil {
		t.Fatalf("ScaffoldMode: %v", err)
	}
	if result.Mode != "demo-mode" {
		t.Fatalf("scaffold mode = %q, want demo-mode", result.Mode)
	}
	for _, rel := range []string{"mode.json", "example-runs/happy-path.json"} {
		if _, err := os.Stat(filepath.Join(root, "modes", "demo-mode", rel)); err != nil {
			t.Fatalf("expected scaffolded file %q: %v", rel, err)
		}
	}

	report, err := svc.ValidateModeDraft("demo-mode")
	if err != nil {
		t.Fatalf("ValidateModeDraft: %v", err)
	}
	if !report.OK {
		t.Fatalf("scaffolded mode failed validation: %+v", report)
	}
	if report.PhaseCount != 3 {
		t.Fatalf("phase count = %d, want 3", report.PhaseCount)
	}
	if report.ExampleRuns != 1 {
		t.Fatalf("example runs = %d, want 1", report.ExampleRuns)
	}

	sim, err := svc.SimulateModeDraft(context.Background(), "demo-mode", "")
	if err != nil {
		t.Fatalf("SimulateModeDraft: %v", err)
	}
	gotPath := make([]string, 0, len(sim.Trace))
	for _, step := range sim.Trace {
		gotPath = append(gotPath, step.Phase)
	}
	want := []string{"execute", "review", "reconcile"}
	if len(gotPath) != len(want) {
		t.Fatalf("simulated path = %v, want %v", gotPath, want)
	}
	for i := range want {
		if gotPath[i] != want[i] {
			t.Fatalf("simulated path = %v, want %v", gotPath, want)
		}
	}
	if sim.ActivePreset != happyPathPresetID {
		t.Fatalf("active preset = %q, want %q", sim.ActivePreset, happyPathPresetID)
	}
}

// TestScaffoldRefusesExistingWithoutForce guards an author from silently
// clobbering a mode they are editing; force overwrites deliberately.
func TestScaffoldRefusesExistingWithoutForce(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	if _, err := svc.ScaffoldMode(ScaffoldRequest{ID: "demo-mode"}); err != nil {
		t.Fatalf("first scaffold: %v", err)
	}
	if _, err := svc.ScaffoldMode(ScaffoldRequest{ID: "demo-mode"}); err == nil {
		t.Fatalf("expected second scaffold to refuse an existing folder")
	}
	if _, err := svc.ScaffoldMode(ScaffoldRequest{ID: "demo-mode", Force: true}); err != nil {
		t.Fatalf("forced scaffold: %v", err)
	}
}

func TestScaffoldRejectsInvalidID(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	for _, bad := range []string{"", "Demo Mode", "demo_mode", "-demo", "demo-"} {
		if _, err := svc.ScaffoldMode(ScaffoldRequest{ID: bad}); err == nil {
			t.Fatalf("expected scaffold to reject id %q", bad)
		}
	}
}

// TestValidateReportsUncoveredBranches proves the branch-coverage signal: the
// blank template's happy-path example-run walks review→reconcile (verdict
// accepted) but never the review→execute rework branch, so validate reports
// that single uncovered branch while still reporting the mode valid.
func TestValidateReportsUncoveredBranches(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	if _, err := svc.ScaffoldMode(ScaffoldRequest{ID: "cover-mode"}); err != nil {
		t.Fatalf("ScaffoldMode: %v", err)
	}
	report, err := svc.ValidateModeDraft("cover-mode")
	if err != nil {
		t.Fatalf("ValidateModeDraft: %v", err)
	}
	if !report.OK {
		t.Fatalf("mode should be valid: %+v", report)
	}
	if len(report.UncoveredBranches) != 1 {
		t.Fatalf("uncovered branches = %v, want exactly the review→execute rework branch", report.UncoveredBranches)
	}
	if !strings.Contains(report.UncoveredBranches[0], "review") || !strings.Contains(report.UncoveredBranches[0], "execute") {
		t.Fatalf("uncovered branch = %q, want the review→execute branch", report.UncoveredBranches[0])
	}
}

// TestScaffoldStartFromClonesExistingMode proves the reuse-first path: scaffold
// --start-from clones an existing mode under a new identity, re-homes its
// id-derived fields, re-targets its example-runs, and produces a folder that
// loads and simulates exactly like the source's shape.
func TestScaffoldStartFromClonesExistingMode(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	if _, err := svc.ScaffoldMode(ScaffoldRequest{ID: "src-mode", Label: "Source Mode"}); err != nil {
		t.Fatalf("scaffold source: %v", err)
	}
	result, err := svc.ScaffoldMode(ScaffoldRequest{ID: "clone-mode", Label: "Clone Mode", StartFrom: "src-mode"})
	if err != nil {
		t.Fatalf("scaffold --start-from: %v", err)
	}
	if len(result.CreatedFiles) < 2 {
		t.Fatalf("clone created files = %v, want mode.json + example-run(s)", result.CreatedFiles)
	}

	// The clone re-homes identity and id-derived fields.
	raw, err := os.ReadFile(filepath.Join(root, "modes", "clone-mode", "mode.json"))
	if err != nil {
		t.Fatalf("read clone mode.json: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode clone mode.json: %v", err)
	}
	if doc["id"] != "clone-mode" || doc["label"] != "Clone Mode" {
		t.Fatalf("clone identity = id:%v label:%v, want clone-mode / Clone Mode", doc["id"], doc["label"])
	}
	if prompt, ok := doc["prompt"].(map[string]any); ok {
		if prompt["catalog_prefix"] != "swarm-manager-clone-mode" {
			t.Fatalf("clone catalog_prefix = %v, want swarm-manager-clone-mode", prompt["catalog_prefix"])
		}
	}

	// The clone example-run is re-targeted at the new mode and still walks.
	exRaw, err := os.ReadFile(filepath.Join(root, "modes", "clone-mode", "example-runs", "happy-path.json"))
	if err != nil {
		t.Fatalf("read clone example-run: %v", err)
	}
	var ex map[string]any
	if err := json.Unmarshal(exRaw, &ex); err != nil {
		t.Fatalf("decode clone example-run: %v", err)
	}
	if ex["mode"] != "clone-mode" {
		t.Fatalf("clone example-run mode = %v, want clone-mode", ex["mode"])
	}

	report, err := svc.ValidateModeDraft("clone-mode")
	if err != nil {
		t.Fatalf("ValidateModeDraft(clone): %v", err)
	}
	if !report.OK {
		t.Fatalf("clone failed validation: %+v", report)
	}
}

// TestScaffoldStartFromUnknownMode fails cleanly rather than writing a broken
// folder when the source mode does not exist.
func TestScaffoldStartFromUnknownMode(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	if _, err := svc.ScaffoldMode(ScaffoldRequest{ID: "clone-mode", StartFrom: "does-not-exist"}); err == nil {
		t.Fatalf("expected clone from unknown mode to fail")
	}
	if _, err := os.Stat(filepath.Join(root, "modes", "clone-mode")); !os.IsNotExist(err) {
		t.Fatalf("expected no clone folder written on failure")
	}
}

// TestRehomeCloneModeRewritesArtifactPaths locks the re-homing contract: a
// clone's identity, id-derived fields, AND every artifact path anchored under
// the source's artifact root move to the new mode's root — a phase artifact left
// pointing at the source root would be rejected as outside the mode root.
func TestRehomeCloneModeRewritesArtifactPaths(t *testing.T) {
	src := []byte(`{
  "kind": "operating-mode",
  "id": "src",
  "label": "Src",
  "description": "source",
  "prompt": { "catalog_prefix": "swarm-manager-src" },
  "artifact": { "root": "modes/src" },
  "metrics": { "event_source": "src" },
  "phase_graph": {
    "phases": [
      { "id": "execute", "output_artifacts": [ { "path": "modes/src/findings.md" } ] }
    ]
  }
}`)
	out, err := rehomeCloneMode(src, "dst", "Dst", "dest")
	if err != nil {
		t.Fatalf("rehomeCloneMode: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc["id"] != "dst" || doc["label"] != "Dst" {
		t.Fatalf("identity not re-homed: %v / %v", doc["id"], doc["label"])
	}
	if got := doc["artifact"].(map[string]any)["root"]; got != "modes/dst" {
		t.Fatalf("artifact root = %v, want modes/dst", got)
	}
	if got := doc["metrics"].(map[string]any)["event_source"]; got != "dst" {
		t.Fatalf("event_source = %v, want dst", got)
	}
	phases := doc["phase_graph"].(map[string]any)["phases"].([]any)
	artifacts := phases[0].(map[string]any)["output_artifacts"].([]any)
	if got := artifacts[0].(map[string]any)["path"]; got != "modes/dst/findings.md" {
		t.Fatalf("phase artifact path = %v, want modes/dst/findings.md", got)
	}
}

// TestValidateModeDraftMissing returns a typed not-found report rather than a
// transport error, so the CLI prints a clean diagnostic.
func TestValidateModeDraftMissing(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	report, err := svc.ValidateModeDraft("nope")
	if err != nil {
		t.Fatalf("ValidateModeDraft: %v", err)
	}
	if report.OK {
		t.Fatalf("expected missing mode to be invalid, got %+v", report)
	}
}
