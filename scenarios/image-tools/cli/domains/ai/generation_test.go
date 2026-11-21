package ai

import (
	"bytes"
	"io"
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
// AI-catalog operation. The command table is generated from the capability
// manifests (`commands.generated.go`), so this guards the end-to-end wiring:
// every manifest with a cli spec becomes a runnable command, and the three
// restoration ops that previously had no headless verb are now reachable.
func TestSubmitCommandsCoverP0(t *testing.T) {
	h := newHandlers(nil) // submitCommands only captures h in closures; not invoked here
	want := []string{"background-replace", "bg-removal", "caption", "colorize", "deblur", "denoise", "depth", "edit", "face-restore", "face-swap", "garment-replace", "generate", "geometry", "image-to-object", "img2img", "inpaint", "naturalize", "normal-map", "object-removal", "old-photo-restore", "outpaint", "segment", "upscale"}
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
		seg, _ := p.GetOpParams().AsMap()["segmentation"].(map[string]any)
		points, _ := seg["points"].([]any)
		box, _ := seg["box"].(map[string]any)
		if len(points) != 2 || points[1].(map[string]any)["negative"] != true || box["width"] != 0.6 {
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

func TestEdit_SubmitsForceCPU(t *testing.T) {
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ai/edit_instruct" {
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
		if !p.GetForceCpu() {
			t.Errorf("force_cpu was not sent: %s", r.FormValue("params"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"cpu-edit-job","modelId":"instruct-pix2pix","tier":"local-cpu"}`))
	})
	core := testutil.NewTestApp(t, server)
	cmd := findCommand(t, newHandlers(core).submitCommands(), "edit")
	in := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(in, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cmd.Args, Flags: map[string]string{"force-cpu": "true", "prompt": "Recolor the stars coral"},
		Positionals: map[string]string{"input": in}, Core: core, Stdout: &output, Stderr: &output,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte("cpu-edit-job")) {
		t.Fatalf("missing job identity: %s", output.String())
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
		roi, _ := p.GetOpParams().AsMap()["support_roi"].(map[string]any)
		if roi == nil || roi["y"] != 0.85 || roi["height"] != 0.15 || roi["width"] != 1.0 {
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
		if !p.GetForceCpu() {
			t.Errorf("force_cpu was not sent: %s", r.FormValue("params"))
		}
		for field, want := range map[string]string{"file": "input", "mask": "mask"} {
			part, _, err := r.FormFile(field)
			if err != nil {
				t.Fatalf("missing %s multipart part: %v", field, err)
			}
			data, readErr := io.ReadAll(part)
			_ = part.Close()
			if readErr != nil || string(data) != want {
				t.Errorf("%s bytes=%q, want %q (read err %v)", field, data, want, readErr)
			}
		}
		crop := p.GetOpParams().AsMap()
		if crop["inpaint_crop_to_mask"] != true || crop["inpaint_context_pixels"] != float64(96) || crop["inpaint_max_edge_px"] != float64(1024) || crop["inpaint_neutralize_masked_pixels"] != true {
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
		Flags:       map[string]string{"mask": mask, "crop-to-mask": "true", "crop-context-pixels": "96", "crop-max-edge-px": "1024", "neutralize-masked-pixels": "true", "prompt": "empty tabletop", "force-cpu": "true"},
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
		repair, _ := p.GetOpParams().AsMap()["background_repair"].(map[string]any)
		regions, _ := repair["removalRegions"].([]any)
		if repair == nil || repair["maskExpansionPixels"] != 12.0 || len(regions) != 1 || regions[0].(map[string]any)["y"] != 0.8 {
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
