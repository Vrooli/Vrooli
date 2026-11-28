package ai

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	testutil "github.com/vrooli/cli-core/cliapptest"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestCaptionSubmitsMultipartImageAndPrompt(t *testing.T) {
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ai/caption" {
			t.Errorf("wrong caption route: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		defer r.MultipartForm.RemoveAll()
		if _, _, err := r.FormFile("file"); err != nil {
			t.Errorf("caption image missing: %v", err)
		}
		var params aiv1.AIParams
		if err := protojson.Unmarshal([]byte(r.FormValue("params")), &params); err != nil {
			t.Fatal(err)
		}
		if params.GetPrompt() != "Describe the red shape." {
			t.Errorf("caption prompt=%q", params.GetPrompt())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"caption-job","modelId":"smolvlm-256m","tier":"local-cpu"}`))
	})
	core := testutil.NewTestApp(t, server)
	cmd := findCommand(t, newHandlers(core).submitCommands(), "caption")
	input := filepath.Join(t.TempDir(), "synthetic.png")
	if err := os.WriteFile(input, []byte("synthetic image bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cmd.Args,
		Flags: map[string]string{
			"prompt": "Describe the red shape.", "fallback-policy": "local_only",
		},
		Positionals: map[string]string{"input": input}, Core: core, Stdout: &output, Stderr: &output,
	})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte("caption-job")) {
		t.Fatalf("caption submit did not return its durable job identity: %s", output.String())
	}
}

func TestParseCaptionOutput(t *testing.T) {
	got, err := parseCaptionOutput([]byte(`{"caption":"  A red circle.  ","backend":"smolvlm","model":"smolvlm-256m"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Caption != "A red circle." || got.Backend != "smolvlm" || got.Model != "smolvlm-256m" {
		t.Fatalf("parsed caption = %#v", got)
	}
	for _, input := range []string{`{`, `{"caption":"  "}`} {
		if _, err := parseCaptionOutput([]byte(input)); err == nil {
			t.Errorf("parseCaptionOutput(%q) succeeded; expected invalid result error", input)
		}
	}
}

type captionJobsService struct {
	jobsconnect.UnimplementedJobsServiceHandler
}

func (captionJobsService) WaitJob(context.Context, *connect.Request[jobsv1.WaitJobRequest]) (*connect.Response[jobsv1.WaitJobResponse], error) {
	return connect.NewResponse(&jobsv1.WaitJobResponse{Job: &jobsv1.Job{
		Id: "caption-job", State: jobsv1.JobState_JOB_STATE_SUCCEEDED, ResultRef: "caption-result",
	}}), nil
}

func TestCaptionWaitRendersDurableResult(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/ai/caption", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("caption method=%s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"caption-job","modelId":"smolvlm-256m","tier":"local-cpu"}`))
	})
	path, handler := jobsconnect.NewJobsServiceHandler(captionJobsService{})
	mux.Handle(path, handler)
	mux.HandleFunc("/api/v1/blobs/caption-result", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("blob method=%s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"caption":"A red circle.","backend":"smolvlm","model":"smolvlm-256m"}`))
	})
	core := testutil.NewTestApp(t, mux)
	cmd := findCommand(t, newHandlers(core).submitCommands(), "caption")
	input := filepath.Join(t.TempDir(), "synthetic.png")
	if err := os.WriteFile(input, []byte("synthetic image bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name        string
		json        bool
		writeOutput bool
		want        string
	}{
		{name: "text", want: "A red circle."},
		{name: "json", json: true, want: `"caption": "A red circle."`},
		{name: "write-caption-text", writeOutput: true, want: "A red circle."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			flags := map[string]string{"fallback-policy": "local_only"}
			var outputPath string
			if tc.writeOutput {
				outputPath = filepath.Join(t.TempDir(), "caption.png")
				flags["out"] = outputPath
			}
			ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
				Schema: cmd.Args, BoolFlags: map[string]bool{"wait": true},
				Flags:       flags,
				Positionals: map[string]string{"input": input}, Core: core, JSON: tc.json,
				Stdout: &output, Stderr: &output,
			})
			if err := cmd.RunCtx(ctx); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), tc.want) {
				t.Fatalf("caption result missing from output: want %q, got %s", tc.want, output.String())
			}
			if tc.json {
				for _, field := range []string{`"backend": "smolvlm"`, `"model": "smolvlm-256m"`} {
					if !strings.Contains(output.String(), field) {
						t.Errorf("caption metadata missing from JSON output: want %q, got %s", field, output.String())
					}
				}
			}
			if tc.writeOutput {
				writtenPath := filepath.Join(filepath.Dir(outputPath), "caption.txt")
				data, err := os.ReadFile(writtenPath)
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != "A red circle." {
					t.Fatalf("caption file = %q, want plain caption text", data)
				}
				if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
					t.Fatalf("image-looking output path unexpectedly exists, stat error=%v", err)
				}
				if !strings.Contains(output.String(), writtenPath) {
					t.Fatalf("actual .txt path missing from output: %s", output.String())
				}
			}
		})
	}
}
