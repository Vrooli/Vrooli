package quality

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
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
	"google.golang.org/protobuf/encoding/protojson"
)

func evaluateCommand(core *cliapp.ScenarioApp, ctx cliapp.RunContext) error {
	report, err := evaluate(core, ctx.Positional("manifest"), ctx.Flag("baseline"))
	if err != nil {
		return err
	}
	if path := ctx.Flag("out"); path != "" {
		b, err := protojson.MarshalOptions{Indent: "  "}.Marshal(report)
		if err != nil {
			return err
		}
		if err = os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
			return err
		}
	}
	lines := []string{fmt.Sprintf("%d cases; complete=%t; promotion eligible=%t", len(report.Cases), report.Complete, report.PromotionEligible), "job: " + report.JobId, "report: " + report.ResultRef}
	for _, v := range report.Violations {
		lines = append(lines, fmt.Sprintf("%s %s: %s", v.CaseId, v.Metric, v.Reason))
	}
	if err := cliapp.RenderProtoList(ctx, report, cliapp.ListReport{Summary: []string{fmt.Sprintf("Quality gate passed=%t (%s)", report.Passed, report.Operation)}, Results: lines}); err != nil {
		return err
	}
	if !report.Passed && !ctx.BoolFlag("report-only") {
		return fmt.Errorf("quality gate failed with %d violation(s); report retained at %s", len(report.Violations), report.ResultRef)
	}
	return nil
}

func evaluate(core *cliapp.ScenarioApp, manifestPath, baselinePath string) (*qualityv1.EvaluationReport, error) {
	body, contentType, err := buildUpload(manifestPath, baselinePath)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	defer os.Remove(body.Name())
	hc, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	base = strings.TrimRight(base, "/")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, base+"/api/v1/quality/evaluate", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	resp.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("quality submission (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	sub := &qualityv1.SubmitEvaluationResponse{}
	if err := protojson.Unmarshal(raw, sub); err != nil {
		return nil, err
	}
	if sub.JobId == "" {
		return nil, fmt.Errorf("quality submission omitted job identity")
	}
	finished, err := jobsconnect.NewJobsServiceClient(hc, base).WaitJob(context.Background(), connect.NewRequest(&jobsv1.WaitJobRequest{Id: sub.JobId}))
	if err != nil {
		return nil, fmt.Errorf("wait quality job %s: %w (resume with image-tools jobs wait %s)", sub.JobId, err, sub.JobId)
	}
	job := finished.Msg.GetJob()
	if job.GetState() != jobsv1.JobState_JOB_STATE_SUCCEEDED {
		return nil, fmt.Errorf("quality job %s: %s", sub.JobId, job.GetError())
	}
	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, base+"/api/v1/blobs/"+job.GetResultRef(), nil)
	if err != nil {
		return nil, err
	}
	resp, err = hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("read quality report: HTTP %d", resp.StatusCode)
	}
	raw, err = io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	report := &qualityv1.EvaluationReport{}
	if err = protojson.Unmarshal(raw, report); err != nil {
		return nil, err
	}
	return report, nil
}

// Use a bounded temporary upload spool rather than retaining a 64-image set
// twice in memory. Paths exist only on the client; API fields are opaque IDs.
func buildUpload(manifestPath, baselinePath string) (*os.File, string, error) {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, "", err
	}
	if len(raw) > 2<<20 {
		return nil, "", fmt.Errorf("manifest exceeds 2 MiB")
	}
	m := &qualityv1.EvaluationManifest{}
	if err = protojson.Unmarshal(raw, m); err != nil {
		return nil, "", err
	}
	spool, err := os.CreateTemp("", "image-quality-*.multipart")
	if err != nil {
		return nil, "", err
	}
	failed := true
	defer func() {
		if failed {
			spool.Close()
			os.Remove(spool.Name())
		}
	}()
	mw := multipart.NewWriter(spool)
	parts := map[string]string{}
	for _, c := range m.Cases {
		for _, field := range []*string{&c.Input, &c.Reference, &c.Output} {
			if *field == "" {
				continue
			}
			path := *field
			if !filepath.IsAbs(path) {
				path = filepath.Join(filepath.Dir(manifestPath), path)
			}
			path = filepath.Clean(path)
			if key, ok := parts[path]; ok {
				*field = key
				continue
			}
			f, err := os.Open(path)
			if err != nil {
				return nil, "", err
			}
			key := fmt.Sprintf("asset_%d", len(parts))
			part, err := mw.CreateFormFile(key, filepath.Base(path))
			if err != nil {
				f.Close()
				return nil, "", err
			}
			n, err := io.Copy(part, io.LimitReader(f, (32<<20)+1))
			f.Close()
			if err != nil {
				return nil, "", err
			}
			if n == 0 || n > 32<<20 {
				return nil, "", fmt.Errorf("image %s is empty or exceeds 32 MiB", path)
			}
			parts[path] = key
			*field = key
		}
	}
	encoded, err := protojson.Marshal(m)
	if err != nil {
		return nil, "", err
	}
	if err = mw.WriteField("manifest", string(encoded)); err != nil {
		return nil, "", err
	}
	if baselinePath != "" {
		b, err := os.ReadFile(baselinePath)
		if err != nil {
			return nil, "", err
		}
		if len(b) > 4<<20 {
			return nil, "", fmt.Errorf("baseline exceeds 4 MiB")
		}
		if err = mw.WriteField("baseline", string(bytes.TrimSpace(b))); err != nil {
			return nil, "", err
		}
	}
	if err = mw.Close(); err != nil {
		return nil, "", err
	}
	size, err := spool.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, "", err
	}
	if size > 256<<20 {
		return nil, "", fmt.Errorf("output set exceeds 256 MiB; split the corpus")
	}
	if _, err = spool.Seek(0, io.SeekStart); err != nil {
		return nil, "", err
	}
	failed = false
	return spool, mw.FormDataContentType(), nil
}
