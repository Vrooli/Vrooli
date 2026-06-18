package ai

import (
	"bytes"
	"net/http"
	"sort"
	"testing"

	"image-tools/cli/internal/testutil"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
)

// TestSubmitCommandsCoverP0 asserts the CLI exposes one submit command per
// Phase-3 AI op (the headless surface IMG-P0-002/003 require).
func TestSubmitCommandsCoverP0(t *testing.T) {
	h := newHandlers(nil) // submitCommands only captures h in closures; not invoked here
	want := []string{"bg-removal", "denoise", "generate", "img2img", "inpaint", "naturalize", "object-removal", "upscale"}
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
