package ai

import "testing"

// TestEnhancement_OpsRunHeadless is the IMG-P0-003 unit: each enhancement op
// (upscale, background_removal, denoise) selects its CPU-capable standalone
// backend, materializes its input, and persists the output — proving the
// vertical works without a GPU (the fake host has none).
func TestEnhancement_OpsRunHeadless(t *testing.T) {
	for _, op := range []string{"upscale", "background_removal", "denoise"} {
		t.Run(op, func(t *testing.T) {
			fp := &fakeProvider{out: []byte(op + "-output")}
			eng, store, modelID := newTestEngine(t, op, fp)
			storeInput(t, store, "input/x.png", []byte("noisy-pixels"))

			ref, err := runJob(t, eng, op, Payload{
				Operation: op,
				ModelID:   modelID,
				InputKey:  "input/x.png",
				Params:    map[string]string{"scale": "4"},
			})
			if err != nil {
				t.Fatalf("run %s: %v", op, err)
			}
			if fp.execN != 1 {
				t.Errorf("%s: Execute calls = %d, want 1", op, fp.execN)
			}
			if len(fp.lastReq.InputKeys) != 1 {
				t.Errorf("%s: expected one materialized input, got %v", op, fp.lastReq.InputKeys)
			}
			// The fake host has no GPU, so the run must be CPU-tier.
			if fp.lastReq.GPU {
				t.Errorf("%s: expected CPU run on a GPU-less host", op)
			}
			assertBlob(t, store, ref, op+"-output")
		})
	}
}

// TestEnhancement_BackendFailurePropagates ensures a backend error fails the job.
func TestEnhancement_BackendFailurePropagates(t *testing.T) {
	fp := &fakeProvider{failWith: errBackend}
	eng, store, modelID := newTestEngine(t, "upscale", fp)
	storeInput(t, store, "input/x.png", []byte("pixels"))

	_, err := runJob(t, eng, "upscale", Payload{
		Operation: "upscale",
		ModelID:   modelID,
		InputKey:  "input/x.png",
	})
	if err == nil {
		t.Fatal("expected the backend failure to propagate")
	}
}

var errBackend = &backendErr{}

type backendErr struct{}

func (*backendErr) Error() string { return "backend exploded" }
