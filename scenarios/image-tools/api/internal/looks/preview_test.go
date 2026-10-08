package looks

import (
	"bytes"
	"image/png"
	"slices"
	"testing"
)

// TestRenderPreviewDeterministicLook proves a film Look renders a real PNG with
// no deferred steps (every step runs on the in-process ops engine).
func TestRenderPreviewDeterministicLook(t *testing.T) {
	png0, deferred, err := RenderPreview(builtinByID(t, "noir"))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(deferred) != 0 {
		t.Errorf("a pure-deterministic Look should defer nothing, got %v", deferred)
	}
	img, err := png.Decode(bytes.NewReader(png0))
	if err != nil {
		t.Fatalf("preview is not a valid PNG: %v", err)
	}
	if img.Bounds().Dx() != previewSize || img.Bounds().Dy() != previewSize {
		t.Errorf("preview size = %v, want %dx%d", img.Bounds(), previewSize, previewSize)
	}
}

// TestRenderPreviewDefersAISteps proves a STYLE Look's AI step is reported as
// deferred and the preview falls back to the (valid) reference image.
func TestRenderPreviewDefersAISteps(t *testing.T) {
	png0, deferred, err := RenderPreview(builtinByID(t, "anime"))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !slices.Contains(deferred, "edit_instruct") {
		t.Errorf("expected edit_instruct deferred, got %v", deferred)
	}
	if _, err := png.Decode(bytes.NewReader(png0)); err != nil {
		t.Fatalf("preview is not a valid PNG: %v", err)
	}
}

// TestRenderPreviewChangesPixels proves a color-grade Look actually alters the
// reference (the preview differs from the unmodified reference image), so the
// preview reflects the Look rather than echoing the swatch.
func TestRenderPreviewChangesPixels(t *testing.T) {
	ref, err := referencePNG()
	if err != nil {
		t.Fatalf("reference: %v", err)
	}
	out, _, err := RenderPreview(builtinByID(t, "vivid-pop"))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if bytes.Equal(ref, out) {
		t.Error("a saturation/contrast Look should change the reference pixels")
	}
}
