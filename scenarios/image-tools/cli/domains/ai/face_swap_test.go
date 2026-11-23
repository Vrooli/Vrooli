package ai

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	testutil "github.com/vrooli/cli-core/cliapptest"
	"google.golang.org/protobuf/encoding/protojson"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
)

func writeImage(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFaceSwap_SendsTheReferenceFaceOptionsAndConsent(t *testing.T) {
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ai/face_swap" {
			t.Errorf("wrong operation: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		defer r.MultipartForm.RemoveAll()
		for part, want := range map[string]string{"file": "target-bytes", "reference": "face-bytes"} {
			f, _, err := r.FormFile(part)
			if err != nil {
				t.Fatalf("missing %s part: %v", part, err)
			}
			got, _ := io.ReadAll(f)
			if string(got) != want {
				t.Errorf("%s part = %q, want %q", part, got, want)
			}
		}
		var p aiv1.AIParams
		if err := protojson.Unmarshal([]byte(r.FormValue("params")), &p); err != nil {
			t.Fatal(err)
		}
		if !p.GetConsentAffirmed() {
			t.Error("--consent must reach the server as consent_affirmed")
		}
		got := p.GetOpParams().AsMap()
		if got["all_faces"] != true || got["occlusion"] != false || got["target_face"] != float64(1) || got["source_face"] != float64(2) || got["color_match"] != 0.5 {
			t.Errorf("face swap options lost: %v", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"swap-job","modelId":"hyperswap-1a-256","tier":"local-gpu"}`))
	})
	core := testutil.NewTestApp(t, server)
	cmd := findCommand(t, newHandlers(core).submitCommands(), "face-swap")
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cmd.Args,
		Flags: map[string]string{
			"reference": writeImage(t, "face.png", "face-bytes"), "consent": "true", "all-faces": "true", "no-occlusion": "true",
			"target-face": "1", "source-face": "2", "color-match": "0.5",
		},
		Positionals: map[string]string{"input": writeImage(t, "target.png", "target-bytes")},
		Core:        core, Stdout: &output, Stderr: &output,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte("swap-job")) {
		t.Fatalf("missing job id: %s", output.String())
	}
}

func TestFaceSwap_RefusesToRunWithoutTheReferenceOrWithBadOptions(t *testing.T) {
	core := testutil.NewTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("nothing may be submitted for an invalid request")
	}))
	cmd := findCommand(t, newHandlers(core).submitCommands(), "face-swap")
	target := writeImage(t, "target.png", "t")
	for name, flags := range map[string]map[string]string{
		"no reference":      {},
		"bad face index":    {"reference": writeImage(t, "f.png", "f"), "target-face": "-1"},
		"bad colour match":  {"reference": writeImage(t, "f.png", "f"), "color-match": "2"},
		"non-numeric index": {"reference": writeImage(t, "f.png", "f"), "source-face": "left"},
	} {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
				Schema: cmd.Args, Flags: flags, Positionals: map[string]string{"input": target},
				Core: core, Stdout: &output, Stderr: &output,
			})
			if err := cmd.RunCtx(ctx); err == nil {
				t.Fatal("expected a usage error")
			}
		})
	}
}

func TestFaceSwap_ExtentFlagsBecomeOpParams(t *testing.T) {
	var got map[string]any
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		var p aiv1.AIParams
		if err := protojson.Unmarshal([]byte(r.FormValue("params")), &p); err != nil {
			t.Fatal(err)
		}
		got = p.GetOpParams().AsMap()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"swap-job","modelId":"hyperswap-1a-256","tier":"local-gpu"}`))
	})
	core := testutil.NewTestApp(t, server)
	cmd := findCommand(t, newHandlers(core).submitCommands(), "face-swap")
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cmd.Args,
		Flags: map[string]string{
			"reference": writeImage(t, "face.png", "face-bytes"),
			"extent":    "full-look", "match": "skin, hair-style, hair-colour", "face-strength": "0.6", "skin-target": "Body",
		},
		Positionals: map[string]string{"input": writeImage(t, "target.png", "target-bytes")},
		Core:        core, Stdout: &output, Stderr: &output,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatal(err)
	}
	stages, _ := got["match"].([]any)
	if got["extent"] != "full_look" || len(stages) != 3 || stages[0] != "skin" || stages[1] != "hair_style" || stages[2] != "hair_colour" || got["face_strength"] != 0.6 || got["skin_target"] != "body" {
		t.Fatalf("extent options lost or not normalised: %v", got)
	}
}

func TestFaceSwap_RejectsAFaceStrengthOutsideTheRange(t *testing.T) {
	core := testutil.NewTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("nothing may be submitted for an invalid request")
	}))
	cmd := findCommand(t, newHandlers(core).submitCommands(), "face-swap")
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:      cmd.Args,
		Flags:       map[string]string{"reference": writeImage(t, "f.png", "f"), "face-strength": "1.5"},
		Positionals: map[string]string{"input": writeImage(t, "target.png", "t")},
		Core:        core, Stdout: &output, Stderr: &output,
	})
	if err := cmd.RunCtx(ctx); err == nil {
		t.Fatal("expected a usage error")
	}
}

func TestMaskComposite_GrowAndFeatherFlagsBecomeOpParams(t *testing.T) {
	var got map[string]any
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		var p aiv1.AIParams
		if err := protojson.Unmarshal([]byte(r.FormValue("params")), &p); err != nil {
			t.Fatal(err)
		}
		got = p.GetOpParams().AsMap()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"composite-job","modelId":"mask-composite-lab-v1","tier":"local-cpu"}`))
	})
	core := testutil.NewTestApp(t, server)
	cmd := findCommand(t, newHandlers(core).submitCommands(), "mask-composite")
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cmd.Args,
		Flags: map[string]string{
			"mask": writeImage(t, "mask.png", "m"), "reference": writeImage(t, "ref.png", "r"), "grow-px": "8", "feather-px": "3",
		},
		Positionals: map[string]string{"input": writeImage(t, "in.png", "i")},
		Core:        core, Stdout: &output, Stderr: &output,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatal(err)
	}
	if got["grow_px"] != float64(8) || got["feather_px"] != float64(3) {
		t.Fatalf("composite options lost: %v", got)
	}
}
