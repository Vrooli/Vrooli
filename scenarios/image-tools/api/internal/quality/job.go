package quality

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
	"google.golang.org/protobuf/encoding/protojson"
	"image-tools/internal/jobs"
)

type BlobStore interface {
	Put(context.Context, string, io.Reader, string) error
	Get(context.Context, string) (io.ReadCloser, string, error)
}
type Task struct {
	Manifest json.RawMessage `json:"manifest"`
	Baseline json.RawMessage `json:"baseline,omitempty"`
}
type Runner struct {
	Evaluator *Evaluator
	Store     BlobStore
}

func (r *Runner) Run(ctx context.Context, job jobs.Job, emit func(int, string)) (jobs.Result, error) {
	var task Task
	if err := json.Unmarshal(job.Payload, &task); err != nil {
		return jobs.Result{}, err
	}
	m := &qualityv1.EvaluationManifest{}
	if err := protojson.Unmarshal(task.Manifest, m); err != nil {
		return jobs.Result{}, err
	}
	var baseline *qualityv1.EvaluationReport
	if len(task.Baseline) > 0 {
		baseline = &qualityv1.EvaluationReport{}
		if err := protojson.Unmarshal(task.Baseline, baseline); err != nil {
			return jobs.Result{}, err
		}
	}
	report, err := r.Evaluator.Evaluate(ctx, m, func(ctx context.Context, key string) ([]byte, error) {
		rc, _, err := r.Store.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, (32<<20)+1))
	}, baseline, emit)
	if err != nil {
		return jobs.Result{}, err
	}
	report.JobId = job.ID
	report.ResultRef = "outputs/" + job.ID + "/quality.json"
	SealReport(report)
	b, err := protojson.Marshal(report)
	if err != nil {
		return jobs.Result{}, err
	}
	if err = r.Store.Put(ctx, report.ResultRef, bytes.NewReader(b), "application/json"); err != nil {
		return jobs.Result{}, err
	}
	meta := map[string]string{"quality_passed": strconv.FormatBool(report.Passed), "quality_complete": strconv.FormatBool(report.Complete), "quality_suite_sha256": report.SuiteSha256, "quality_report_sha256": report.ReportSha256, "evaluated_model": m.CandidateModel}
	if m.Judge != "disabled" {
		meta["model_id"], meta["judge_model"], meta["capacity_ownership"] = report.JudgeModel, report.JudgeModel, "resource:ollama"
	}
	return jobs.Result{Ref: report.ResultRef, Meta: meta}, nil
}

// Admit reserves bounded metric work in the same scheduler as all image jobs.
// The judge's accelerator admission remains with AI Gateway/resource-ollama.
func Admit(ctx context.Context, job jobs.Job) (jobs.Admission, error) {
	scope, err := sharedcapacity.AcquireScoped(ctx, sharedcapacity.Request{OwnerID: "image-tools:" + job.ID + ":quality", RAMBytes: 1 << 30, CPUMilli: 1000, Priority: string(job.Priority), NoReclaim: true})
	if err != nil {
		var unavailable *sharedcapacity.UnavailableError
		if errors.As(err, &unavailable) {
			err = fmt.Errorf("%w: %v", jobs.ErrCapacityBusy, err)
		}
		return jobs.Admission{Context: ctx}, err
	}
	return jobs.Admission{Context: scope.Context(), Release: scope.Release, Meta: map[string]string{"ram_capacity_claim_bytes": "1073741824", "cpu_capacity_milli": "1000", "judge_capacity_ownership": "resource:ollama"}}, nil
}
