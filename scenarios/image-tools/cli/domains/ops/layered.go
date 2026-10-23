package ops

import (
	"connectrpc.com/connect"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"net/http"
	"os"
	"strings"
)

func (h *handlers) layeredCommand(name, description string) cliapp.Command {
	return cliapp.Command{Name: name, Description: description, NeedsAPI: true, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{inputArg()}, Flags: []cliapp.Flag{outFlag(), {Name: "operation-id", Description: "Stable id for parallax retries; required for parallax"}, {Name: "params-file", Required: true, Description: "Typed OpParams JSON containing composite or parallax"}, {Name: "plate", Repeated: true, Description: "Prepared RGBA layer path, in declared parameter order (repeatable)"}}}, RunCtx: func(ctx cliapp.RunContext) error {
		raw, err := os.ReadFile(ctx.Flag("params-file"))
		if err != nil {
			return err
		}
		params := &opsv1.OpParams{}
		if err = protojson.Unmarshal(raw, params); err != nil {
			return fmt.Errorf("typed layered parameters: %w", err)
		}
		scene := params.GetComposite()
		if name == "parallax" {
			if params.GetParallax() == nil {
				return fmt.Errorf("params-file must contain parallax")
			}
			scene = params.GetParallax().GetScene()
		}
		if scene == nil {
			return fmt.Errorf("params-file must contain %s scene", name)
		}
		plates := ctx.FlagValues("plate")
		if len(plates) != len(scene.GetPlates()) {
			return fmt.Errorf("declared %d layers; supply exactly that many --plate paths", len(scene.GetPlates()))
		}
		if ctx.Flag("out") == "" {
			return fmt.Errorf("--out is required")
		}
		if name == "parallax" && ctx.Flag("operation-id") == "" {
			return fmt.Errorf("--operation-id is required; reuse it to recover the same job after an uncertain result")
		}
		result, err := runOpWithPlates(h.core, name, ctx.Positional("input"), "", extToFormat(ctx.Flag("out")), params, plates, ctx.Flag("operation-id"))
		if err != nil {
			return err
		}
		if result.format == "queued" {
			fmt.Fprintf(ctx.Stderr(), "Parallax job %s accepted. Waiting once; recover with image-tools jobs get/wait/download %s.\n", result.jobID, result.jobID)
			result, err = waitParallax(h.core, result.jobID)
			if err != nil {
				return err
			}
		}
		return writeOutput(ctx, result, ctx.Flag("out"))
	}}
}

func waitParallax(core *cliapp.ScenarioApp, id string) (runResult, error) {
	client, base := cliapp.NewConnectHTTPClient(core)
	waited, err := jobsconnect.NewJobsServiceClient(client, base).WaitJob(context.Background(), connect.NewRequest(&jobsv1.WaitJobRequest{Id: id}))
	if err != nil {
		return runResult{}, fmt.Errorf("wait job %s (the job remains owned by the server): %w", id, err)
	}
	job := waited.Msg.GetJob()
	if job.GetState() != jobsv1.JobState_JOB_STATE_SUCCEEDED {
		return runResult{}, fmt.Errorf("parallax job %s ended %s: %s", id, job.GetState(), job.GetError())
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, strings.TrimRight(base, "/")+"/api/v1/blobs/"+job.GetResultRef(), nil)
	if err != nil {
		return runResult{}, err
	}
	response, err := client.Do(req)
	if err != nil {
		return runResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return runResult{}, fmt.Errorf("download parallax job %s: HTTP %d", id, response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 128<<20+1))
	if err != nil {
		return runResult{}, err
	}
	if len(data) > 128<<20 {
		return runResult{}, fmt.Errorf("parallax result exceeds 128MiB")
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != job.GetResultMeta()["sha256"] {
		return runResult{}, fmt.Errorf("parallax result digest mismatch")
	}
	return runResult{body: data, jobID: id, format: "zip", width: job.GetResultMeta()["width"], height: job.GetResultMeta()["height"]}, nil
}
