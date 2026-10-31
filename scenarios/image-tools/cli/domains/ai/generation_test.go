package ai

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"testing"

	testutil "github.com/vrooli/cli-core/cliapptest"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
)

// TestSubmitCommandsCoverP0 asserts the CLI exposes one submit command per
// AI op (the P0 generation/enhancement ops + the Phase-4 breadth ops:
// outpaint / background-replace / colorize / depth — the headless surface
// IMG-P0-002/003 + IMG-P1-001/002 require).
func TestSubmitCommandsCoverP0(t *testing.T) {
	h := newHandlers(nil) // submitCommands only captures h in closures; not invoked here
	want := []string{"background-replace", "bg-removal", "colorize", "denoise", "depth", "edit", "generate", "geometry", "image-to-object", "img2img", "inpaint", "naturalize", "normal-map", "object-removal", "outpaint", "segment", "upscale"}
	got := make([]string, 0)
	for _, c := range h.submitCommands() {
		if c.RunCtx == nil {
			t.Errorf("command %q has no RunCtx handler", c.Name)
		}
		got = append(got, c.Name)
	}
	sort.Strings(got)
	if len(got) != len(want) {
		t.Fatalf("submit commands = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("submit commands = %v, want %v", got, want)
		}
	}
}

// fakeAIServer responds to the AI submit edge with a canned SubmitAIResponse.
func fakeAIServer(t *testing.T) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/ai/", func(w http.ResponseWriter, _ *http.Request) {
		resp := &aiv1.SubmitAIResponse{JobId: "job-9", ModelId: "sd-1.5", Tier: "local-cpu", EstimatedSeconds: 30}
		raw, _ := protojson.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(raw)
	})
	return mux
}

func TestSegment_SubmitsExplicitNegativePointAndBox(t *testing.T) {
	const operationID = "4c3369eb-7fe6-4837-96bb-7a22f5e7baf6"
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ai/segment" {
			t.Errorf("wrong operation: %s", r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") != operationID {
			t.Errorf("retry identity lost: %q", r.Header.Get("Idempotency-Key"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		defer r.MultipartForm.RemoveAll()
		var p aiv1.AIParams
		if err := protojson.Unmarshal([]byte(r.FormValue("params")), &p); err != nil {
			t.Fatal(err)
		}
		if len(p.GetSegmentation().GetPoints()) != 2 || !p.Segmentation.Points[1].Negative || p.Segmentation.Box.Width != 0.6 {
			t.Errorf("prompt lost: %s", r.FormValue("params"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"jobId":"sam-job","modelId":"mobilesam","tier":"local-cpu"}`))
	})
	core := testutil.NewTestApp(t, server)
	cmd := findCommand(t, newHandlers(core).submitCommands(), "segment")
	in := filepath.Join(t.TempDir(), "device.png")
	if err := os.WriteFile(in, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:      cmd.Args,
		Flags:       map[string]string{"operation-id": operationID, "selection": `{"points":[{"x":0.7,"y":0.4},{"x":0.1,"y":0.2,"negative":true}],"box":{"x":0.2,"y":0.1,"width":0.6,"height":0.8}}`},
		Positionals: map[string]string{"input": in}, Core: core, Stdout: &output, Stderr: &output,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte("sam-job")) {
		t.Fatalf("missing reattach identity: %s", output.String())
	}
}

func TestGeometry_SubmitsTypedSupportROI(t *testing.T) {
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ai/scene_geometry" {
			t.Errorf("wrong operation: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		defer r.MultipartForm.RemoveAll()
		var p aiv1.AIParams
		if err := protojson.Unmarshal([]byte(r.FormValue("params")), &p); err != nil {
			t.Fatal(err)
		}
		roi := p.GetSceneGeometry().GetSupportRoi()
		if roi == nil || roi.GetY() != 0.85 || roi.GetHeight() != 0.15 || roi.GetWidth() != 1 {
			t.Errorf("support ROI lost: %s", r.FormValue("params"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"geometry-job","modelId":"moge-2-vits-normal","tier":"local-cpu"}`))
	})
	core := testutil.NewTestApp(t, server)
	cmd := findCommand(t, newHandlers(core).submitCommands(), "geometry")
	in := filepath.Join(t.TempDir(), "scene.png")
	if err := os.WriteFile(in, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cmd.Args, Flags: map[string]string{"support-roi": `{"supportRoi":{"x":0,"y":0.85,"width":1,"height":0.15}}`},
		Positionals: map[string]string{"input": in}, Core: core, Stdout: &output, Stderr: &output,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte("geometry-job")) {
		t.Fatalf("missing geometry job identity: %s", output.String())
	}
}

func TestInpaint_SubmitsTypedMaskCrop(t *testing.T) {
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		defer r.MultipartForm.RemoveAll()
		var p aiv1.AIParams
		if err := protojson.Unmarshal([]byte(r.FormValue("params")), &p); err != nil {
			t.Fatal(err)
		}
		crop := p.GetInpaint()
		if crop == nil || !crop.GetCropToMask() || crop.GetContextPixels() != 96 || crop.GetMaxEdgePx() != 1024 || !crop.GetNeutralizeMaskedPixels() {
			t.Errorf("inpaint crop controls lost: %s", r.FormValue("params"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"inpaint-job","modelId":"sdxl-inpainting-0.1","tier":"local-gpu"}`))
	})
	core := testutil.NewTestApp(t, server)
	cmd := findCommand(t, newHandlers(core).submitCommands(), "inpaint")
	in := filepath.Join(t.TempDir(), "photo.png")
	mask := filepath.Join(t.TempDir(), "mask.png")
	_ = os.WriteFile(in, []byte("input"), 0o600)
	_ = os.WriteFile(mask, []byte("mask"), 0o600)
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:      cmd.Args,
		Flags:       map[string]string{"mask": mask, "crop-to-mask": "true", "crop-context-pixels": "96", "crop-max-edge-px": "1024", "neutralize-masked-pixels": "true", "prompt": "empty tabletop"},
		Positionals: map[string]string{"input": in}, Core: core, Stdout: &output, Stderr: &output,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestObjectRemoval_SubmitsExplicitMaskAdditions(t *testing.T) {
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		defer r.MultipartForm.RemoveAll()
		var p aiv1.AIParams
		if err := protojson.Unmarshal([]byte(r.FormValue("params")), &p); err != nil {
			t.Fatal(err)
		}
		repair := p.GetBackgroundRepair()
		if repair == nil || repair.GetMaskExpansionPixels() != 12 || len(repair.GetRemovalRegions()) != 1 || repair.RemovalRegions[0].GetY() != .8 {
			t.Errorf("repair controls lost: %s", r.FormValue("params"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"repair-job","modelId":"big-lama","tier":"local-cpu"}`))
	})
	core := testutil.NewTestApp(t, server)
	cmd := findCommand(t, newHandlers(core).submitCommands(), "object-removal")
	in := filepath.Join(t.TempDir(), "device.png")
	mask := filepath.Join(t.TempDir(), "mask.png")
	_ = os.WriteFile(in, []byte("input"), 0o600)
	_ = os.WriteFile(mask, []byte("mask"), 0o600)
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:      cmd.Args,
		Flags:       map[string]string{"mask": mask, "mask-expansion": "12", "remove-region": `{"x":0.1,"y":0.8,"width":0.3,"height":0.1}`},
		Positionals: map[string]string{"input": in}, Core: core, Stdout: &output, Stderr: &output,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatal(err)
	}
}

// TestGenerate_SubmitsAsyncJob is the IMG-P0-002 integration: the generate
// command submits to the REST edge and reports the durable job id + reattach
// hint (no polling).
func TestGenerate_SubmitsAsyncJob(t *testing.T) {
	core := testutil.NewTestApp(t, fakeAIServer(t))
	h := newHandlers(core)
	cmd := findCommand(t, h.submitCommands(), "generate")

	var buf bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cmd.Args,
		Flags:  map[string]string{"prompt": "a red bicycle"},
		Core:   core,
		Stdout: &buf,
		Stderr: &buf,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if out := buf.String(); !bytes.Contains([]byte(out), []byte("job-9")) {
		t.Errorf("expected the job id in output, got: %s", out)
	}
}

// TestEditInstruct_SubmitsAsyncJob is the IMG-P1-014 integration: the edit
// command (instruction-edit) takes an input image + a natural-language
// instruction and submits to the REST edge, reporting the durable job id.
func TestEditInstruct_SubmitsAsyncJob(t *testing.T) {
	core := testutil.NewTestApp(t, fakeAIServer(t))
	h := newHandlers(core)
	cmd := findCommand(t, h.submitCommands(), "edit")

	input := filepath.Join(t.TempDir(), "photo.png")
	if err := os.WriteFile(input, []byte("\x89PNG\r\n\x1a\nsource"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var buf bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:      cmd.Args,
		Positionals: map[string]string{"input": input},
		Flags:       map[string]string{"prompt": "make it look like winter"},
		Core:        core,
		Stdout:      &buf,
		Stderr:      &buf,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if out := buf.String(); !bytes.Contains([]byte(out), []byte("job-9")) {
		t.Errorf("expected the job id in output, got: %s", out)
	}
}

func findCommand(t *testing.T, cmds []cliapp.Command, name string) cliapp.Command {
	t.Helper()
	for _, c := range cmds {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("command %q not found", name)
	return cliapp.Command{}
}
