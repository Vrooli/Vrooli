package ai

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"image-tools/internal/storage"
)

// TestGeneration_InvokesBackendAndPersistsOutput is the IMG-P0-002 unit: a
// generation op selects its standalone backend, passes the prompt params
// through, and persists the produced output (retrievable by the returned ref).
func TestGeneration_InvokesBackendAndPersistsOutput(t *testing.T) {
	fp := &fakeProvider{out: []byte("generated-image")}
	eng, store, modelID := newTestEngine(t, "text_to_image", fp)

	ref, err := runJob(t, eng, "text_to_image", Payload{
		Operation: "text_to_image",
		ModelID:   modelID,
		Params:    map[string]string{"prompt": "a red bicycle", "width": "512"},
	})
	if err != nil {
		t.Fatalf("run text_to_image: %v", err)
	}

	if fp.execN != 1 {
		t.Errorf("provider Execute calls = %d, want 1", fp.execN)
	}
	if got := fp.lastReq.Model.ID; got != modelID {
		t.Errorf("backend got model %q, want %q", got, modelID)
	}
	if got := fp.lastReq.Params["prompt"]; got != "a red bicycle" {
		t.Errorf("backend got prompt %q, want %q", got, "a red bicycle")
	}
	if ref == "" {
		t.Fatal("expected a non-empty output ref")
	}
	assertBlob(t, store, ref, "generated-image")
}

// TestGeneration_ImageToImageMaterializesInput proves an edit op materializes its
// stored input blob to a file the backend receives.
func TestGeneration_ImageToImageMaterializesInput(t *testing.T) {
	fp := &fakeProvider{}
	eng, store, modelID := newTestEngine(t, "image_to_image", fp)
	storeInput(t, store, "input/x.png", []byte("source-pixels"))

	_, err := runJob(t, eng, "image_to_image", Payload{
		Operation: "image_to_image",
		ModelID:   modelID,
		InputKey:  "input/x.png",
		Params:    map[string]string{"prompt": "watercolor", "strength": "0.6"},
	})
	if err != nil {
		t.Fatalf("run image_to_image: %v", err)
	}
	if len(fp.lastReq.InputKeys) != 1 || fp.lastReq.InputKeys[0] == "" {
		t.Fatalf("expected one materialized input path, got %v", fp.lastReq.InputKeys)
	}
}

// TestGeneration_EditInstructMaterializesInput proves the instruction-edit op
// (identity-preserving, prompt-only, no mask) materializes its single input blob
// and threads the natural-language instruction through to the backend.
func TestGeneration_EditInstructMaterializesInput(t *testing.T) {
	fp := &fakeProvider{out: []byte("edited-image")}
	eng, store, modelID := newTestEngine(t, "edit_instruct", fp)
	storeInput(t, store, "input/x.png", []byte("source-pixels"))

	ref, err := runJob(t, eng, "edit_instruct", Payload{
		Operation: "edit_instruct",
		ModelID:   modelID,
		InputKey:  "input/x.png",
		Params:    map[string]string{"prompt": "add a winter coat"},
	})
	if err != nil {
		t.Fatalf("run edit_instruct: %v", err)
	}
	if len(fp.lastReq.InputKeys) != 1 || fp.lastReq.InputKeys[0] == "" {
		t.Fatalf("expected one materialized input path, got %v", fp.lastReq.InputKeys)
	}
	if got := fp.lastReq.Params["prompt"]; got != "add a winter coat" {
		t.Errorf("backend got instruction %q, want %q", got, "add a winter coat")
	}
	assertBlob(t, store, ref, "edited-image")
}

// TestGeneration_InpaintRequiresMask proves the mask input is materialized as the
// second input for mask-driven ops.
func TestGeneration_InpaintRequiresMask(t *testing.T) {
	fp := &fakeProvider{}
	eng, store, modelID := newTestEngine(t, "inpaint", fp)
	storeInput(t, store, "input/x.png", []byte("source"))
	storeInput(t, store, "mask/m.png", []byte("mask"))

	_, err := runJob(t, eng, "inpaint", Payload{
		Operation: "inpaint",
		ModelID:   modelID,
		InputKey:  "input/x.png",
		MaskKey:   "mask/m.png",
		Params:    map[string]string{"prompt": "sky"},
	})
	if err != nil {
		t.Fatalf("run inpaint: %v", err)
	}
	if len(fp.lastReq.InputKeys) != 2 {
		t.Fatalf("expected image+mask inputs, got %v", fp.lastReq.InputKeys)
	}
}

// TestGeneration_Variations produces N distinct outputs in one job.
func TestGeneration_Variations(t *testing.T) {
	fp := &fakeProvider{}
	eng, _, modelID := newTestEngine(t, "text_to_image", fp)

	_, err := runJob(t, eng, "text_to_image", Payload{
		Operation:  "text_to_image",
		ModelID:    modelID,
		Variations: 3,
		Params:     map[string]string{"prompt": "x", "seed": "100"},
	})
	if err != nil {
		t.Fatalf("run with variations: %v", err)
	}
	if fp.execN != 3 {
		t.Errorf("Execute calls = %d, want 3 (one per variation)", fp.execN)
	}
	// The third variation's seed is base+2.
	if got := fp.lastReq.Params["seed"]; got != "102" {
		t.Errorf("last variation seed = %q, want 102", got)
	}
}

// TestGeneration_AutoScanHook fires the NSFW scanner on the output when requested.
func TestGeneration_AutoScanHook(t *testing.T) {
	fp := &fakeProvider{}
	eng, _, modelID := newTestEngine(t, "text_to_image", fp)
	scanned := false
	eng.deps.AutoScan = func(context.Context, []byte) (ScanVerdict, error) {
		scanned = true
		return ScanVerdict{NSFW: true, Score: 0.91}, nil
	}

	_, err := runJob(t, eng, "text_to_image", Payload{
		Operation:    "text_to_image",
		ModelID:      modelID,
		AutoScanNSFW: true,
		Params:       map[string]string{"prompt": "x"},
	})
	if err != nil {
		t.Fatalf("run with auto-scan: %v", err)
	}
	if !scanned {
		t.Error("expected the NSFW auto-scan hook to fire")
	}
}

func assertBlob(t *testing.T, store *storage.Store, ref, want string) {
	t.Helper()
	rc, _, err := store.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("get blob %q: %v", ref, err)
	}
	defer func() { _ = rc.Close() }()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read blob: %v", err)
	}
	if string(got) != want {
		t.Errorf("blob %q = %q, want %q", ref, string(got), want)
	}
}

// recordingStore records which artifacts the engine refuses to publish.
type recordingStore struct {
	*storage.Store
	deleted []string
}

func (s *recordingStore) Delete(ctx context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return s.Store.Delete(ctx, key)
}

func TestReviewOutputHardRefusalFailsJobAndDeletesArtifacts(t *testing.T) {
	fp := &fakeProvider{}
	eng, store, modelID := newTestEngine(t, "text_to_image", fp)
	rec := &recordingStore{Store: store}
	eng.deps.Store = rec
	reviewed := 0
	eng.deps.ReviewOutput = func(context.Context, []byte) (ScanVerdict, error) {
		reviewed++
		return ScanVerdict{HardRefusal: "classifier: minor + sexual"}, nil
	}

	_, err := runJob(t, eng, "text_to_image", Payload{Operation: "text_to_image", ModelID: modelID, Params: map[string]string{"prompt": "x"}})
	if err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("err = %v, want a hard refusal", err)
	}
	if reviewed != 1 {
		t.Fatalf("reviewed %d outputs, want 1", reviewed)
	}
	if len(rec.deleted) != 1 {
		t.Fatalf("deleted %v, want the refused artifact removed before the job failed", rec.deleted)
	}
}

func TestReviewOutputUnavailableFailsClosed(t *testing.T) {
	fp := &fakeProvider{}
	eng, store, modelID := newTestEngine(t, "text_to_image", fp)
	rec := &recordingStore{Store: store}
	eng.deps.Store = rec
	eng.deps.ReviewOutput = func(context.Context, []byte) (ScanVerdict, error) {
		return ScanVerdict{}, errors.New("classifier crashed")
	}

	_, err := runJob(t, eng, "text_to_image", Payload{Operation: "text_to_image", ModelID: modelID, Params: map[string]string{"prompt": "x"}})
	if err == nil || !strings.Contains(err.Error(), "review failed") {
		t.Fatalf("err = %v, want the review failure surfaced", err)
	}
	if len(rec.deleted) != 1 {
		t.Fatalf("deleted %v, want the unreviewed artifact removed", rec.deleted)
	}
}
