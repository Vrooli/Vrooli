// Package ai is the CLI's model-backed AI command surface. `ai list` mirrors the
// Connect AIService discovery RPC; the per-operation submit commands (generate,
// img2img, inpaint, …) drive the REST multipart submit edge
// (POST /api/v1/ai/{operation}) — image bytes can't ride a Connect call, so
// these are hand-built commands rather than manifest connect-rpc bindings.
//
// AI ops run asynchronously: submit returns a job id + ETA + the resolved
// model/tier, and the caller blocks ONCE on JobsService.WaitJob (no polling),
// mirroring the test-genie run-lifecycle philosophy.
package ai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
)

// GroupName is the manifest group name this package owns.
const GroupName = "ai"

// submitAI posts the optional input/mask images + protojson params to the REST
// submit edge and returns the parsed SubmitAIResponse. inputPath/maskPath may be
// empty (text_to_image needs neither).
func submitAI(core *cliapp.ScenarioApp, operation, inputPath, maskPath string, params *aiv1.AIParams) (*aiv1.SubmitAIResponse, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if inputPath != "" {
		if err := addFilePart(mw, "file", inputPath); err != nil {
			return nil, err
		}
	}
	if maskPath != "" {
		if err := addFilePart(mw, "mask", maskPath); err != nil {
			return nil, err
		}
	}
	if params != nil {
		raw, err := protojson.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		if err := mw.WriteField("params", string(raw)); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	url := strings.TrimRight(baseURL, "/") + "/api/v1/ai/" + operation
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("submit %s: %w", operation, err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("%s failed (%d): %s", operation, resp.StatusCode, strings.TrimSpace(string(out)))
	}
	parsed := &aiv1.SubmitAIResponse{}
	if err := protojson.Unmarshal(out, parsed); err != nil {
		return nil, fmt.Errorf("decode submit response: %w", err)
	}
	return parsed, nil
}

func addFilePart(mw *multipart.Writer, field, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s %q: %w", field, path, err)
	}
	fw, err := mw.CreateFormFile(field, filepath.Base(path))
	if err != nil {
		return err
	}
	_, err = fw.Write(data)
	return err
}

// waitAndDownload blocks once on JobsService.WaitJob, then downloads the result
// blob to outPath. It returns the terminal job for status reporting.
func waitAndDownload(core *cliapp.ScenarioApp, jobID, outPath string) (*jobsv1.Job, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := jobsconnect.NewJobsServiceClient(httpClient, baseURL)
	resp, err := client.WaitJob(context.Background(), connect.NewRequest(&jobsv1.WaitJobRequest{Id: jobID}))
	if err != nil {
		return nil, cliapp.WrapAPIError("wait job", err, nil)
	}
	job := resp.Msg.GetJob()
	if job.GetState() != jobsv1.JobState_JOB_STATE_SUCCEEDED {
		return job, fmt.Errorf("job %s %s: %s", jobID, stateName(job.GetState()), job.GetError())
	}
	if outPath != "" && job.GetResultRef() != "" {
		if err := downloadBlob(core, job.GetResultRef(), outPath); err != nil {
			return job, err
		}
	}
	return job, nil
}

func downloadBlob(core *cliapp.ScenarioApp, ref, outPath string) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	url := strings.TrimRight(baseURL, "/") + "/api/v1/blobs/" + ref
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download result: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download result failed (%d)", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write output %q: %w", outPath, err)
	}
	return nil
}

func stateName(s jobsv1.JobState) string {
	switch s {
	case jobsv1.JobState_JOB_STATE_SUCCEEDED:
		return "succeeded"
	case jobsv1.JobState_JOB_STATE_FAILED:
		return "failed"
	case jobsv1.JobState_JOB_STATE_CANCELED:
		return "canceled"
	case jobsv1.JobState_JOB_STATE_RUNNING:
		return "running"
	case jobsv1.JobState_JOB_STATE_QUEUED:
		return "queued"
	default:
		return "unknown"
	}
}
