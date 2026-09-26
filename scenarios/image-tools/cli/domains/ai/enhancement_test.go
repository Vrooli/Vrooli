package ai

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	testutil "github.com/vrooli/cli-core/cliapptest"

	"github.com/vrooli/cli-core/cliapp"
)

// TestUpscale_SubmitsWithInput is the IMG-P0-003 integration: an enhancement
// command reads its input image and submits the async job.
func TestUpscale_SubmitsWithInput(t *testing.T) {
	core := testutil.NewTestApp(t, fakeAIServer(t))
	h := newHandlers(core)
	cmd := findCommand(t, h.submitCommands(), "upscale")

	in := filepath.Join(t.TempDir(), "in.png")
	if err := os.WriteFile(in, []byte("fake-png-bytes"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var buf bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:      cmd.Args,
		Flags:       map[string]string{"scale": "4"},
		Positionals: map[string]string{"input": in},
		Core:        core,
		Stdout:      &buf,
		Stderr:      &buf,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatalf("upscale: %v", err)
	}
	if out := buf.String(); !bytes.Contains([]byte(out), []byte("job-9")) {
		t.Errorf("expected job id in output, got: %s", out)
	}
}

// TestEnhancement_MissingInputErrors proves the input positional is required.
func TestEnhancement_MissingInputErrors(t *testing.T) {
	core := testutil.NewTestApp(t, fakeAIServer(t))
	h := newHandlers(core)
	cmd := findCommand(t, h.submitCommands(), "denoise")
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cmd.Args,
		Core:   core,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	})
	if err := cmd.RunCtx(ctx); err == nil {
		t.Fatal("expected an error when the input image is missing")
	}
}

// TestNaturalize_SubmitsWithRealismAndFaceAware proves the naturalize command
// reads its input image and threads the --realism / --face-aware flags into the
// submitted params (IMG-P1-011).
func TestNaturalize_SubmitsWithRealismAndFaceAware(t *testing.T) {
	core := testutil.NewTestApp(t, fakeAIServer(t))
	h := newHandlers(core)
	cmd := findCommand(t, h.submitCommands(), "naturalize")

	in := filepath.Join(t.TempDir(), "in.png")
	if err := os.WriteFile(in, []byte("fake-png-bytes"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var buf bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:      cmd.Args,
		Flags:       map[string]string{"realism": "0.8", "face-aware": "true"},
		Positionals: map[string]string{"input": in},
		Core:        core,
		Stdout:      &buf,
		Stderr:      &buf,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatalf("naturalize: %v", err)
	}
	if out := buf.String(); !bytes.Contains([]byte(out), []byte("job-9")) {
		t.Errorf("expected job id in output, got: %s", out)
	}
}

// TestNaturalize_BuildParams pins the flag → AIParams mapping for the knobs.
func TestNaturalize_BuildParams(t *testing.T) {
	h := newHandlers(nil)
	cmd := findCommand(t, h.submitCommands(), "naturalize")
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cmd.Args,
		Flags:  map[string]string{"realism": "0.6", "face-aware": "true"},
	})
	p := buildParams(ctx)
	if p.GetRealism() != 0.6 {
		t.Errorf("realism = %v, want 0.6", p.GetRealism())
	}
	if !p.GetFaceAware() {
		t.Error("face_aware should be true")
	}
}
