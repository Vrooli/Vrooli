package ai

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	testutil "github.com/vrooli/cli-core/cliapptest"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
)

// TestSubmitCommandsCoverP0 asserts the CLI exposes one submit command per
// capability manifest that declares an `ai` CLI verb. The expected set is read
// from the manifests themselves (the single declaration the command table is
// generated from), so a new operation needs no edit here: the test fails only
// when the generated table and the manifests disagree, or a command has no handler.
func TestSubmitCommandsCoverP0(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "api", "internal", "capability", "manifests.seed.json"))
	if err != nil {
		t.Fatalf("read capability manifests: %v", err)
	}
	var seed struct {
		Manifests []struct {
			CLI *struct {
				Group string `json:"group"`
				Verb  string `json:"verb"`
			} `json:"cli"`
		} `json:"manifests"`
	}
	if err := json.Unmarshal(raw, &seed); err != nil {
		t.Fatalf("parse capability manifests: %v", err)
	}
	want := make([]string, 0)
	for _, m := range seed.Manifests {
		if m.CLI != nil && m.CLI.Group == "ai" {
			want = append(want, m.CLI.Verb)
		}
	}
	sort.Strings(want)
	if len(want) < 20 {
		t.Fatalf("read only %d ai verbs from the manifests; the read is broken", len(want))
	}
	h := newHandlers(nil) // submitCommands only captures h in closures; not invoked here
	got := make([]string, 0)
	for _, c := range h.submitCommands() {
		if c.RunCtx == nil {
			t.Errorf("command %q has no RunCtx handler", c.Name)
		}
		got = append(got, c.Name)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("submit commands = %v, manifests declare %v (run `make capability-gen`)", got, want)
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

func TestGenerate_NoAutoAcquireOptsOut(t *testing.T) {
	sent := make(chan *aiv1.AIParams, 2)
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		defer r.MultipartForm.RemoveAll()
		var p aiv1.AIParams
		if err := protojson.Unmarshal([]byte(r.FormValue("params")), &p); err != nil {
			t.Fatal(err)
		}
		sent <- &p
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"gen-job","modelId":"sd-1.5","tier":"local-cpu"}`))
	})
	core := testutil.NewTestApp(t, server)
	cmd := findCommand(t, newHandlers(core).submitCommands(), "generate")
	for _, flags := range []map[string]string{{"prompt": "a lighthouse"}, {"prompt": "a lighthouse", "no-auto-acquire": "true"}} {
		var output bytes.Buffer
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cmd.Args, Flags: flags, Core: core, Stdout: &output, Stderr: &output})
		if err := cmd.RunCtx(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if def := <-sent; def.AutoAcquire != nil {
		t.Fatalf("default submission set auto_acquire=%v; the service default must apply", def.GetAutoAcquire())
	}
	if opt := <-sent; opt.AutoAcquire == nil || opt.GetAutoAcquire() {
		t.Fatal("--no-auto-acquire did not send auto_acquire=false")
	}
}

// The routing flags travel as typed AIParams fields, unset unless given, so the
// deployment defaults (wait budget, spend cap, identity preservation) apply to a
// caller who says nothing.
func TestEdit_RoutingFlagsAreTypedAndUnsetByDefault(t *testing.T) {
	sent := make(chan *aiv1.AIParams, 3)
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		defer r.MultipartForm.RemoveAll()
		var p aiv1.AIParams
		if err := protojson.Unmarshal([]byte(r.FormValue("params")), &p); err != nil {
			t.Fatal(err)
		}
		sent <- &p
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"edit-job","modelId":"flux-2-klein-4b","tier":"local-gpu"}`))
	})
	core := testutil.NewTestApp(t, server)
	cmd := findCommand(t, newHandlers(core).submitCommands(), "edit")
	in := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(in, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, flags := range []map[string]string{
		{"prompt": "a hat"},
		{"prompt": "a hat", "max-wait": "90s", "max-cost": "$0.10", "edit-style": "Stylised", "no-preserve-identity": "true"},
		{"prompt": "a hat", "max-wait": "0", "max-cost": "0"},
	} {
		var output bytes.Buffer
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cmd.Args, Flags: flags, Positionals: map[string]string{"input": in}, Core: core, Stdout: &output, Stderr: &output})
		if err := cmd.RunCtx(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if def := <-sent; def.MaxWaitSeconds != nil || def.MaxCostUsd != nil || def.PreserveIdentity != nil || def.GetEditStyle() != "" {
		t.Fatalf("default submission must leave every routing knob unset: %s", protojson.Format(def))
	}
	set := <-sent
	if set.MaxWaitSeconds == nil || set.GetMaxWaitSeconds() != 90 || set.MaxCostUsd == nil || set.GetMaxCostUsd() != 0.10 || set.GetEditStyle() != "stylized" || set.PreserveIdentity == nil || set.GetPreserveIdentity() {
		t.Fatalf("flags not sent: %s", protojson.Format(set))
	}
	zero := <-sent
	if zero.MaxWaitSeconds == nil || zero.GetMaxWaitSeconds() != 0 || zero.MaxCostUsd == nil || zero.GetMaxCostUsd() != 0 {
		t.Fatalf("an explicit 0 means no limit / no spend and must be sent, not dropped: %s", protojson.Format(zero))
	}
}

func TestEdit_RejectsMalformedRoutingFlagsBeforeSubmitting(t *testing.T) {
	core := testutil.NewTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("a malformed routing flag must not reach the server")
	}))
	cmd := findCommand(t, newHandlers(core).submitCommands(), "edit")
	in := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(in, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, flags := range []map[string]string{
		{"max-wait": "soon"}, {"max-wait": "-5s"}, {"max-cost": "cheap"}, {"max-cost": "-1"}, {"edit-style": "pretty"},
	} {
		flags["prompt"] = "a hat"
		var output bytes.Buffer
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cmd.Args, Flags: flags, Positionals: map[string]string{"input": in}, Core: core, Stdout: &output, Stderr: &output})
		if err := cmd.RunCtx(ctx); err == nil {
			t.Fatalf("%v was accepted", flags)
		}
	}
}
