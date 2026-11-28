package ai

import (
	"os"
	"path/filepath"
	"testing"
)

const testReceipt = `{"execution_device":"cuda","placement":"staged_text_encoder_then_denoiser:sequential_cpu_offload","gpu_peak_allocated_bytes":2106183680,"gpu_peak_reserved_bytes":2231369728}`

// stampReceipt writes a PNG with the sidecar's receipt chunk, the way PIL does.
func stampReceipt(t *testing.T, path, keyword, text string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, withPNGTextChunks(data, []pngChunk{newPNGTextChunk(keyword, text)}), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReceiptMetaReadsThePeakTheSidecarMeasured(t *testing.T) {
	path := writeFrameImage(t, t.TempDir(), "out.png", 64, 64)
	stampReceipt(t, path, "vrooli_masked_edit_provenance", testReceipt)
	got := receiptMeta(path)
	want := map[string]string{
		"inference_device": "cuda", "inference_placement": "staged_text_encoder_then_denoiser:sequential_cpu_offload",
		"inference_cuda_peak_allocated_bytes": "2106183680", "inference_cuda_peak_reserved_bytes": "2231369728",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %q, want %q (all: %v)", key, got[key], value, got)
		}
	}
}

func TestReceiptMetaIsEmptyWithoutAReceipt(t *testing.T) {
	dir := t.TempDir()
	plain := writeFrameImage(t, dir, "plain.png", 32, 32)
	other := writeFrameImage(t, dir, "other.png", 32, 32)
	stampReceipt(t, other, "Comment", "a text chunk that is not a receipt")
	notPNG := filepath.Join(dir, "x.jpg")
	if err := os.WriteFile(notPNG, []byte("\xff\xd8\xff not a png"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{plain, other, notPNG, filepath.Join(dir, "missing.png")} {
		if got := receiptMeta(path); len(got) != 0 {
			t.Errorf("%s: unexpected meta %v", filepath.Base(path), got)
		}
	}
}

// Resampling an edit back to the source frame rewrites the pixels, and the
// receipt the sidecar stamped must survive it.
func TestFrameMatchKeepsTheSidecarReceipt(t *testing.T) {
	dir := t.TempDir()
	source := writeFrameImage(t, dir, "source.png", 1152, 1536)
	out := writeFrameImage(t, dir, "out.png", 880, 1168)
	stampReceipt(t, out, "vrooli_diffusers_provenance", testReceipt)
	policy, rewritten, err := matchSourceFrame("edit_instruct", source, out, nil)
	if err != nil || policy != framePolicyResample || !rewritten {
		t.Fatalf("policy=%q rewritten=%t err=%v", policy, rewritten, err)
	}
	if w, h := frameSize(t, out); w != 1152 || h != 1536 {
		t.Fatalf("size = %dx%d", w, h)
	}
	if got := receiptMeta(out)["inference_cuda_peak_allocated_bytes"]; got != "2106183680" {
		t.Fatalf("the receipt did not survive the frame match: peak = %q", got)
	}
}
