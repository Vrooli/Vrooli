package looks

import (
	"testing"

	"image-tools/internal/ops"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

// TestBuiltinLooksValid asserts every seeded Look is structurally valid, ids are
// unique, and each step's engine tag matches the catalog it names — a bad seed
// edit fails loud here rather than at runtime.
func TestBuiltinLooksValid(t *testing.T) {
	seen := map[string]bool{}
	for _, look := range BuiltinLooks() {
		if !look.GetBuiltin() {
			t.Errorf("seed Look %q must have builtin=true", look.GetId())
		}
		if seen[look.GetId()] {
			t.Errorf("duplicate built-in Look id %q", look.GetId())
		}
		seen[look.GetId()] = true
		if err := Validate(look); err != nil {
			t.Errorf("built-in Look %q invalid: %v", look.GetId(), err)
		}
	}
}

// TestBuiltinDeterministicLooksRenderableInProcess proves the film/camera Looks
// are pure-deterministic (every step is a real internal/ops op), so their
// preview renders with no model — the headless guarantee the picker relies on.
func TestBuiltinDeterministicLooksRenderableInProcess(t *testing.T) {
	deterministicKinds := map[looksv1.LookKind]bool{
		looksv1.LookKind_LOOK_KIND_FILM:   true,
		looksv1.LookKind_LOOK_KIND_CAMERA: true,
	}
	for _, look := range BuiltinLooks() {
		if !deterministicKinds[look.GetKind()] {
			continue
		}
		for _, step := range look.GetSteps() {
			if step.GetKind() != looksv1.StepKind_STEP_KIND_DETERMINISTIC || !ops.Has(step.GetOperation()) {
				t.Errorf("Look %q (kind %v) step %q must be a deterministic ops op for a no-model preview", look.GetId(), look.GetKind(), step.GetOperation())
			}
		}
	}
}

// TestScreenshotCleanupLookRunsHeadless is the composition proof for a
// cross-step chain: a trimmed screenshot is produced by a deterministic step and
// handed to the next step through the image port, and only the model-backed
// step is deferred. It runs in-process with no API, no model, and no network.
func TestScreenshotCleanupLookRunsHeadless(t *testing.T) {
	var look *looksv1.Look
	for _, candidate := range BuiltinLooks() {
		if candidate.GetId() == "screenshot-cleanup" {
			look = candidate
		}
	}
	if look == nil {
		t.Fatal("the screenshot-cleanup Look is not seeded")
	}
	png, deferred, err := RenderPreview(look)
	if err != nil {
		t.Fatalf("RenderPreview: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("preview produced no image")
	}
	if len(deferred) != 1 || deferred[0] != "upscale" {
		t.Fatalf("deferred = %v, want only [upscale]", deferred)
	}
}
