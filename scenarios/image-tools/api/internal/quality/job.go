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
		meta["model_id"], meta["judge_model"] = report.JudgeModel, report.JudgeModel
		if chain, err := ParseJudgeChain(report.JudgeModel); err == nil {
			meta["capacity_ownership"] = ChainOwnership(chain)
		}
	}
	return jobs.Result{Ref: report.ResultRef, Meta: meta}, nil
}

// Admit reserves bounded metric work in the same scheduler as all image jobs.
// The judge's accelerator admission remains with AI Gateway/resource-ollama.
func Admit(ctx context.Context, job jobs.Job) (jobs.Admission, error) {
	ramBytes, cpuMilli := int64(1<<30), int64(1000)
	var task Task
	manifest := &qualityv1.EvaluationManifest{}
	parsed := json.Unmarshal(job.Payload, &task) == nil && protojson.Unmarshal(task.Manifest, manifest) == nil
	// LPIPS runs torch on CPU in the sidecar: a 12-megapixel pair is scored in
	// 1024-pixel tiles inside about 2 GiB and two cores.
	if parsed && wantsMetric(budgetMetricNames(manifest), MetricLPIPS) {
		ramBytes, cpuMilli = 2<<30, 2000
	}
	scope, err := sharedcapacity.AcquireScoped(ctx, sharedcapacity.Request{OwnerID: "image-tools:" + job.ID + ":quality", RAMBytes: ramBytes, CPUMilli: cpuMilli, Priority: string(job.Priority), NoReclaim: true})
	if err != nil {
		var unavailable *sharedcapacity.UnavailableError
		if errors.As(err, &unavailable) {
			err = fmt.Errorf("%w: %v", jobs.ErrCapacityBusy, err)
		}
		return jobs.Admission{Context: ctx}, err
	}
	meta := map[string]string{"ram_capacity_claim_bytes": strconv.FormatInt(ramBytes, 10), "cpu_capacity_milli": strconv.FormatInt(cpuMilli, 10), "judge_capacity_ownership": "resource:ollama"}
	// A manifest that names a hosted judge hands accelerator ownership to that
	// provider; recording the local resource would misattribute the work.
	if parsed && manifest.Judge != "disabled" {
		if chain, err := ParseJudgeChain(manifest.JudgeModel); err == nil {
			meta["judge_capacity_ownership"] = ChainOwnership(chain)
		}
	}
	return jobs.Admission{Context: scope.Context(), Release: scope.Release, ReleaseRetryable: true, Meta: meta}, nil
}
