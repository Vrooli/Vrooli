package operatingmode

import (
	"context"
	"os"
	"path/filepath"
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
