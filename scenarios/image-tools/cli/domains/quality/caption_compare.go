package quality

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models/models_v1connect"
	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	captionDatasetID = "image-tools-caption-v1"
	captionJudge     = "qwen3-vl:4b"
	captionCorpusSHA = "a514bbcbd4c0ca5e38519cb249772ce6ce40bd0374a09e28e56dc64aaec4a6a5"
)

var captionModelIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,100}$`)

type captionCorpusFile struct {
	SchemaVersion int                  `json:"schema_version"`
	DatasetID     string               `json:"dataset_id"`
	Safety        string               `json:"safety"`
	Images        []captionCorpusImage `json:"images"`
}

type captionCorpusImage struct {
	ID         string `json:"id"`
	Input      string `json:"input"`
	SHA256     string `json:"sha256"`
	Difficulty string `json:"difficulty"`
	Caption    string `json:"caption"`
	Question   string `json:"question"`
	Answer     string `json:"answer"`
}

type captionModelReady struct {
	ID, Name, Revision, Support string
	Resolution                  *modelsv1.Resolution
}

type captionTaskReceipt struct {
	CaseID       string `json:"case_id"`
	Task         string `json:"task"`
	JobID        string `json:"job_id,omitempty"`
	State        string `json:"state"`
	Model        string `json:"model,omitempty"`
	Backend      string `json:"backend,omitempty"`
	Tier         string `json:"tier,omitempty"`
	ResultRef    string `json:"result_ref,omitempty"`
	OutputSHA256 string `json:"output_sha256,omitempty"`
	Error        string `json:"error,omitempty"`
}

type captionModelRun struct {
	SchemaVersion int                  `json:"schema_version"`
	DatasetID     string               `json:"dataset_id"`
	CorpusSHA256  string               `json:"corpus_sha256"`
	ModelID       string               `json:"model_id"`
	ModelRevision string               `json:"model_revision"`
	ResolvedTier  string               `json:"resolved_tier"`
	Support       string               `json:"support"`
	StartedAt     string               `json:"started_at"`
	FinishedAt    string               `json:"finished_at,omitempty"`
	State         string               `json:"state"`
	Tasks         []captionTaskReceipt `json:"tasks"`
	Failure       string               `json:"failure,omitempty"`
	Limits        []string             `json:"limits"`
}

type captionOutput struct {
	Caption string `json:"caption"`
	Backend string `json:"backend"`
	Model   string `json:"model"`
}

// captionRuntime is the owner seam for API interactions. Tests replace it with
// a fake so no validation test can submit work to the host job service.
type captionRuntime interface {
	preflight(context.Context, string) (captionModelReady, error)
	submit(context.Context, string, string, string) (*aiv1.SubmitAIResponse, error)
	wait(context.Context, string) (*jobsv1.Job, error)
	cancelAndWait(context.Context, string) (*jobsv1.Job, error)
	fetch(context.Context, string) ([]byte, error)
	evaluate(context.Context, string, string) (*qualityv1.EvaluationReport, error)
}

type coreCaptionRuntime struct{ core *cliapp.ScenarioApp }

func (r coreCaptionRuntime) preflight(ctx context.Context, id string) (captionModelReady, error) {
	return preflightCaptionModel(ctx, r.core, id)
}
func (r coreCaptionRuntime) submit(ctx context.Context, input, prompt, model string) (*aiv1.SubmitAIResponse, error) {
	return submitCaptionCPU(ctx, r.core, input, prompt, model)
}
func (r coreCaptionRuntime) wait(ctx context.Context, id string) (*jobsv1.Job, error) {
	return waitCaptionJob(ctx, r.core, id)
}
func (r coreCaptionRuntime) cancelAndWait(ctx context.Context, id string) (*jobsv1.Job, error) {
	return cancelCaptionJob(ctx, r.core, id)
}
func (r coreCaptionRuntime) fetch(ctx context.Context, ref string) ([]byte, error) {
	return fetchCaptionResult(ctx, r.core, ref)
}
func (r coreCaptionRuntime) evaluate(ctx context.Context, manifest, baseline string) (*qualityv1.EvaluationReport, error) {
	return evaluate(ctx, r.core, manifest, baseline)
}

func captionCompareCommand(core *cliapp.ScenarioApp, ctx cliapp.RunContext) error {
	commandCtx, stopOnSignal := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopOnSignal()
	models := strings.Split(strings.TrimSpace(ctx.Flag("models")), ",")
	if len(models) != 2 {
		return fmt.Errorf("--models requires exactly two comma-separated model ids ordered baseline,candidate")
	}
	for i := range models {
		models[i] = strings.TrimSpace(models[i])
		if !captionModelIDPattern.MatchString(models[i]) {
			return fmt.Errorf("invalid model id %q", models[i])
		}
	}
	if models[0] == models[1] {
		return fmt.Errorf("baseline and candidate model ids must differ")
	}
	corpusPath := ctx.Positional("corpus")
	outDir := ctx.Flag("out-dir")
	if outDir == "" {
		return fmt.Errorf("--out-dir is required; use a new or empty directory so completed evidence is immutable")
	}
	corpus, corpusBytes, corpusDir, err := loadCaptionCorpus(corpusPath)
	if err != nil {
		return err
	}
	// Read-only preflight for both models happens before creating job evidence or
	// submitting anything. It does not install, warm, reserve or execute a model.
	ready := make([]captionModelReady, len(models))
	runtime := coreCaptionRuntime{core: core}
	for i, id := range models {
		ready[i], err = runtime.preflight(commandCtx, id)
		if err != nil {
			return fmt.Errorf("preflight %s: %w", id, err)
		}
	}
	if err := prepareCaptionOutputDir(outDir); err != nil {
		return err
	}
	corpusSum := sha256.Sum256(corpusBytes)
	corpusHash := hex.EncodeToString(corpusSum[:])
	manifests := make([]map[string]string, len(models))
	for i, model := range ready {
		run := captionModelRun{
			SchemaVersion: 1, DatasetID: captionDatasetID, CorpusSHA256: corpusHash,
			ModelID: model.ID, ModelRevision: model.Revision, ResolvedTier: "local-cpu",
			Support: model.Support, StartedAt: time.Now().UTC().Format(time.RFC3339Nano), State: "running",
			Tasks: []captionTaskReceipt{}, Limits: []string{"Only the frozen five-image synthetic corpus is covered; it contains no client uploads, private media, or real-person likenesses.", "Caption jobs are forced to local CPU, local-only, serialized and non-reclaiming; each still requires a fresh successful whole-job admission.", "The local judge is a model opinion, not calibrated human preference; promotion applies only to this corpus and evaluator version."},
		}
		runPath := filepath.Join(outDir, model.ID+"-run.json")
		if err := writeJSONExclusive(runPath, &run); err != nil {
			return err
		}
		manifests[i] = map[string]string{}
		for _, task := range []string{"caption", "visual-question-answering"} {
			manifest, failure := runCaptionTaskSet(commandCtx, runtime, corpus, corpusDir, model, task, &run, runPath)
			if failure != nil {
				run.State = "failed"
				run.Failure = failure.Error()
				run.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
				_ = replaceJSON(runPath, &run)
				return fmt.Errorf("caption comparison stopped at %s/%s; no retry was submitted; receipt %s: %w", model.ID, task, runPath, failure)
			}
			path := filepath.Join(outDir, model.ID+"-"+task+".manifest.json")
			if err := writeProtoExclusive(path, manifest); err != nil {
				run.State = "failed"
				run.Failure = err.Error()
				run.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
				_ = replaceJSON(runPath, &run)
				return err
			}
			manifests[i][task] = path
		}
		run.State = "outputs-complete"
		run.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := replaceJSON(runPath, &run); err != nil {
			return err
		}
	}

	// The first model's immutable reports establish the paired baselines. The
	// second model's reports are scored against those exact report files.
	reports := make([]map[string]*qualityv1.EvaluationReport, len(models))
	for i := range models {
		reports[i] = map[string]*qualityv1.EvaluationReport{}
		for _, task := range []string{"caption", "visual-question-answering"} {
			baselinePath := ""
			if i == 1 {
				baselinePath = filepath.Join(outDir, models[0]+"-"+task+".report.json")
			}
			report, err := runtime.evaluate(commandCtx, manifests[i][task], baselinePath)
			if err != nil {
				return fmt.Errorf("evaluate %s/%s: %w", models[i], task, err)
			}
			reports[i][task] = report
			reportPath := filepath.Join(outDir, models[i]+"-"+task+".report.json")
			if err := writeProtoExclusive(reportPath, report); err != nil {
				return err
			}
		}
	}
	rows := []string{}
	eligible := true
	for _, task := range []string{"caption", "visual-question-answering"} {
		base, challenger := reports[0][task], reports[1][task]
		rows = append(rows, fmt.Sprintf("%s: baseline passed=%t complete=%t; candidate passed=%t promotion_eligible=%t", task, base.GetPassed(), base.GetComplete(), challenger.GetPassed(), challenger.GetPromotionEligible()))
		eligible = eligible && challenger.GetPromotionEligible()
	}
	if err := ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("C07 paired CPU corpus run: %s → %s; promotion eligible=%t", models[0], models[1], eligible), "Evidence directory: " + outDir}, ResultsHeading: "Caption and VQA gates", Results: rows}); err != nil {
		return err
	}
	if !eligible {
		return fmt.Errorf("candidate is not promotion-eligible in both caption and VQA; reports retained in %s", outDir)
	}
	return nil
}

func loadCaptionCorpus(path string) (*captionCorpusFile, []byte, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, "", fmt.Errorf("read C07 corpus %q: %w", path, err)
	}
	var corpus captionCorpusFile
	if err := json.Unmarshal(raw, &corpus); err != nil {
		return nil, nil, "", fmt.Errorf("decode C07 corpus: %w", err)
	}
	if corpus.SchemaVersion != 1 || corpus.DatasetID != captionDatasetID || len(corpus.Images) != 5 {
		return nil, nil, "", fmt.Errorf("C07 requires schema 1 dataset %q with exactly five frozen cases", captionDatasetID)
	}
	corpusSum := sha256.Sum256(raw)
	if !strings.EqualFold(hex.EncodeToString(corpusSum[:]), captionCorpusSHA) {
		return nil, nil, "", fmt.Errorf("C07 corpus manifest hash differs from the frozen v1 identity")
	}
	if !strings.Contains(strings.ToLower(corpus.Safety), "no client uploads") || !strings.Contains(strings.ToLower(corpus.Safety), "real-person likenesses") {
		return nil, nil, "", fmt.Errorf("C07 corpus does not declare the required synthetic-only safety boundary")
	}
	dir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, nil, "", err
	}
	seen := map[string]bool{}
	for _, item := range corpus.Images {
		if item.ID == "" || seen[item.ID] || item.Caption == "" || item.Question == "" || item.Answer == "" || len(item.SHA256) != 64 {
			return nil, nil, "", fmt.Errorf("C07 corpus has an incomplete or duplicate case %q", item.ID)
		}
		seen[item.ID] = true
		input := item.Input
		if !filepath.IsAbs(input) {
			input = filepath.Join(dir, input)
		}
		input, err = filepath.Abs(filepath.Clean(input))
		if err != nil {
			return nil, nil, "", err
		}
		data, err := os.ReadFile(input)
		if err != nil {
			return nil, nil, "", fmt.Errorf("case %s input %q: %w", item.ID, input, err)
		}
		sum := sha256.Sum256(data)
		if !strings.EqualFold(hex.EncodeToString(sum[:]), item.SHA256) {
			return nil, nil, "", fmt.Errorf("case %s input SHA-256 mismatch", item.ID)
		}
		item.Input = input
	}
	return &corpus, raw, dir, nil
}

func preflightCaptionModel(ctx context.Context, core *cliapp.ScenarioApp, modelID string) (captionModelReady, error) {
	hc, base := cliapp.NewConnectHTTPClient(core)
	models := modelsconnect.NewModelsServiceClient(hc, base)
	got, err := models.GetModel(ctx, connect.NewRequest(&modelsv1.GetModelRequest{Id: modelID}))
	if err != nil {
		return captionModelReady{}, cliapp.WrapAPIError("get caption model", err, nil)
	}
	model := got.Msg.GetModel()
	if model == nil || model.GetId() != modelID || !model.GetInstall().GetInstalled() {
		return captionModelReady{}, fmt.Errorf("model is not installed; this evaluator never installs or downloads models")
	}
	revision := strings.TrimSpace(model.GetInstall().GetChecksum())
	if len(revision) != 64 {
		return captionModelReady{}, fmt.Errorf("installed model has no pinned SHA-256 identity; refusing an unbound comparison")
	}
	if _, err := hex.DecodeString(revision); err != nil {
		return captionModelReady{}, fmt.Errorf("installed model checksum is not hexadecimal: %w", err)
	}
	response, err := models.ExplainResolution(ctx, connect.NewRequest(&modelsv1.ExplainResolutionRequest{Operation: "caption", ModelId: modelID, FallbackPolicy: "local_only", ForceCpu: true}))
	if err != nil {
		return captionModelReady{}, cliapp.WrapAPIError("preflight forced-CPU caption resolution", err, nil)
	}
	resolution := response.Msg.GetResolution()
	if resolution == nil || resolution.GetModelId() != modelID || resolution.GetTier() != "local-cpu" {
		return captionModelReady{}, fmt.Errorf("forced-CPU resolution did not select %s/local-cpu", modelID)
	}
	return captionModelReady{ID: modelID, Name: model.GetName(), Revision: "sha256:" + strings.ToLower(revision), Support: resolution.GetSupport(), Resolution: resolution}, nil
}

func prepareCaptionOutputDir(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("output path %q is not a directory", path)
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		if len(entries) != 0 {
			return fmt.Errorf("output directory %q is not empty; refusing to overwrite immutable run evidence", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(path, 0o755)
}

func runCaptionTaskSet(ctx context.Context, runtime captionRuntime, corpus *captionCorpusFile, corpusDir string, model captionModelReady, task string, run *captionModelRun, runPath string) (*qualityv1.EvaluationManifest, error) {
	manifest := newCaptionManifest(model, task)
	for _, item := range corpus.Images {
		prompt, reference := "Describe the visible image accurately.", item.Caption
		if task == "visual-question-answering" {
			prompt, reference = item.Question, item.Answer
		}
		input := item.Input
		if !filepath.IsAbs(input) {
			input = filepath.Join(corpusDir, input)
		}
		input, _ = filepath.Abs(input)
		started := time.Now()
		receipt := captionTaskReceipt{CaseID: item.ID, Task: task, State: "submitting"}
		response, err := runtime.submit(ctx, input, prompt, model.ID)
		if err != nil {
			receipt.State, receipt.Error = "submit-failed", err.Error()
			run.Tasks = append(run.Tasks, receipt)
			_ = replaceJSON(runPath, run)
			return nil, fmt.Errorf("%s/%s submit after %s: %w", task, item.ID, time.Since(started).Round(time.Millisecond), err)
		}
		receipt.JobID, receipt.State, receipt.ResultRef = response.GetJobId(), "accepted", ""
		_ = replaceJSON(runPath, run)
		job, err := runtime.wait(ctx, response.GetJobId())
		if err != nil {
			if ctx.Err() != nil {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				canceled, cancelErr := runtime.cancelAndWait(cleanupCtx, response.GetJobId())
				cancel()
				receipt.JobID = response.GetJobId()
				receipt.State = "cancel-cleanup-failed"
				if cancelErr == nil && canceled != nil && canceled.GetState() == jobsv1.JobState_JOB_STATE_CANCELED {
					receipt.State = "canceled-cleaned"
				} else if cancelErr == nil && jobTerminal(canceled) {
					receipt.State = "cancel-terminal-" + strings.ToLower(canceled.GetState().String())
				}
				receipt.Error = errors.Join(err, cancelErr).Error()
				run.Tasks = append(run.Tasks, receipt)
				_ = replaceJSON(runPath, run)
				return nil, fmt.Errorf("%s/%s job %s interrupted; cancellation cleanup %s: %w", task, item.ID, response.GetJobId(), receipt.State, errors.Join(err, cancelErr))
			}
			receipt.State, receipt.Error = "wait-failed", err.Error()
			run.Tasks = append(run.Tasks, receipt)
			_ = replaceJSON(runPath, run)
			return nil, fmt.Errorf("%s/%s job %s failed without retry: %w", task, item.ID, response.GetJobId(), err)
		}
		receipt.JobID, receipt.ResultRef = job.GetId(), job.GetResultRef()
		receipt.Model, receipt.Backend, receipt.Tier = job.GetResultMeta()["model"], job.GetResultMeta()["backend"], job.GetResultMeta()["tier"]
		if receipt.Tier != "local-cpu" || receipt.Model != model.ID {
			receipt.State, receipt.Error = "execution-mismatch", fmt.Sprintf("result metadata model=%q tier=%q; required %q/local-cpu", receipt.Model, receipt.Tier, model.ID)
			run.Tasks = append(run.Tasks, receipt)
			_ = replaceJSON(runPath, run)
			return nil, fmt.Errorf("%s/%s: %s", task, item.ID, receipt.Error)
		}
		out, err := runtime.fetch(ctx, job.GetResultRef())
		if err != nil {
			receipt.State, receipt.Error = "result-fetch-failed", err.Error()
			run.Tasks = append(run.Tasks, receipt)
			_ = replaceJSON(runPath, run)
			return nil, fmt.Errorf("%s/%s: %w", task, item.ID, err)
		}
		var decoded captionOutput
		if err := json.Unmarshal(out, &decoded); err != nil {
			receipt.State, receipt.Error = "result-decode-failed", err.Error()
			run.Tasks = append(run.Tasks, receipt)
			_ = replaceJSON(runPath, run)
			return nil, fmt.Errorf("%s/%s: decode caption result: %w", task, item.ID, err)
		}
		if strings.TrimSpace(decoded.Caption) == "" || decoded.Model != model.ID || decoded.Backend == "" {
			receipt.State, receipt.Error = "result-identity-failed", "caption result text/backend/model identity is incomplete or mismatched"
			run.Tasks = append(run.Tasks, receipt)
			_ = replaceJSON(runPath, run)
			return nil, fmt.Errorf("%s/%s: %s", task, item.ID, receipt.Error)
		}
		decoded.Caption = strings.TrimSpace(decoded.Caption)
		sum := sha256.Sum256([]byte(decoded.Caption))
		receipt.OutputSHA256, receipt.State = hex.EncodeToString(sum[:]), "succeeded"
		receipt.Backend = decoded.Backend
		run.Tasks = append(run.Tasks, receipt)
		manifest.Cases = append(manifest.Cases, &qualityv1.EvaluationCase{Id: item.ID, Input: input, InputSha256: item.SHA256, ReferenceText: reference, OutputText: decoded.Caption, Instruction: prompt, Cohort: item.Difficulty})
		if err := replaceJSON(runPath, run); err != nil {
			return nil, fmt.Errorf("persist task receipt: %w", err)
		}
	}
	return manifest, nil
}

func newCaptionManifest(model captionModelReady, task string) *qualityv1.EvaluationManifest {
	return &qualityv1.EvaluationManifest{
		SchemaVersion: 1, Name: "caption-v1-" + task, Operation: "caption", Profile: "text", DatasetId: captionDatasetID,
		CandidateModel: model.ID, CandidateRevision: model.Revision, Judge: "required", JudgeModel: captionJudge,
		Budgets: []*qualityv1.MetricBudget{
			{Metric: "text.similarity", Scope: "case", Minimum: captionFloat64(.35)},
			{Metric: "judge.fidelity", Scope: "case", Minimum: captionFloat64(.70)},
			{Metric: "judge.instruction", Scope: "case", Minimum: captionFloat64(.75)},
			{Metric: "judge.artifacts", Scope: "case", Minimum: captionFloat64(.80)},
			{Metric: "judge.fidelity", Scope: "mean", Minimum: captionFloat64(.80)},
			{Metric: "judge.fidelity", Scope: "mean", MinImprovement: captionFloat64(.01)},
			{Metric: "judge.instruction", Scope: "mean", Minimum: captionFloat64(.80)},
			{Metric: "judge.artifacts", Scope: "mean", Minimum: captionFloat64(.80)},
		},
	}
}

func submitCaptionCPU(ctx context.Context, core *cliapp.ScenarioApp, input, prompt, modelID string) (*aiv1.SubmitAIResponse, error) {
	params := &aiv1.AIParams{Prompt: prompt, ModelOverride: modelID, ForceCpu: true, FallbackPolicy: "local_only", Priority: "batch", AllowReclaim: boolPointer(false)}
	encoded, err := protojson.Marshal(params)
	if err != nil {
		return nil, err
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	f, err := os.Open(input)
	if err != nil {
		return nil, err
	}
	part, err := mw.CreateFormFile("file", filepath.Base(input))
	if err == nil {
		_, err = io.Copy(part, io.LimitReader(f, (32<<20)+1))
	}
	f.Close()
	if err != nil {
		return nil, err
	}
	if err := mw.WriteField("params", string(encoded)); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	hc, base := cliapp.NewConnectHTTPClient(core)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/api/v1/ai/caption", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 4<<20 {
		return nil, fmt.Errorf("caption submit response exceeds 4 MiB")
	}
	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("caption submit returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var submitted aiv1.SubmitAIResponse
	if err := protojson.Unmarshal(raw, &submitted); err != nil {
		return nil, err
	}
	if submitted.GetJobId() == "" || submitted.GetModelId() != modelID || submitted.GetTier() != "local-cpu" {
		return nil, fmt.Errorf("caption submit identity/tier mismatch: job=%q model=%q tier=%q", submitted.GetJobId(), submitted.GetModelId(), submitted.GetTier())
	}
	return &submitted, nil
}

func waitCaptionJob(ctx context.Context, core *cliapp.ScenarioApp, id string) (*jobsv1.Job, error) {
	hc, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	resp, err := jobsconnect.NewJobsServiceClient(hc, base).WaitJob(ctx, connect.NewRequest(&jobsv1.WaitJobRequest{Id: id}))
	if err != nil {
		return nil, fmt.Errorf("wait once for job %s: %w", id, err)
	}
	job := resp.Msg.GetJob()
	if job == nil || job.GetState() != jobsv1.JobState_JOB_STATE_SUCCEEDED {
		if job == nil {
			return nil, fmt.Errorf("job %s wait returned no job", id)
		}
		return job, fmt.Errorf("job %s %s: %s", id, job.GetState().String(), job.GetError())
	}
	if job.GetResultRef() == "" {
		return job, fmt.Errorf("job %s succeeded without result_ref", id)
	}
	return job, nil
}

func cancelCaptionJob(ctx context.Context, core *cliapp.ScenarioApp, id string) (*jobsv1.Job, error) {
	hc, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	client := jobsconnect.NewJobsServiceClient(hc, base)
	resp, err := client.CancelJob(ctx, connect.NewRequest(&jobsv1.CancelJobRequest{Id: id}))
	if err != nil {
		return nil, fmt.Errorf("request cancellation for job %s: %w", id, err)
	}
	job := resp.Msg.GetJob()
	if job == nil {
		return nil, fmt.Errorf("cancel job %s returned no job", id)
	}
	if jobTerminal(job) {
		return job, nil
	}
	return waitCaptionJob(ctx, core, id)
}

func jobTerminal(job *jobsv1.Job) bool {
	if job == nil {
		return false
	}
	switch job.GetState() {
	case jobsv1.JobState_JOB_STATE_SUCCEEDED, jobsv1.JobState_JOB_STATE_FAILED, jobsv1.JobState_JOB_STATE_CANCELED:
		return true
	default:
		return false
	}
}

func fetchCaptionResult(ctx context.Context, core *cliapp.ScenarioApp, ref string) ([]byte, error) {
	hc, base := cliapp.NewConnectHTTPClient(core)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/api/v1/blobs/"+ref, nil)
	if err != nil {
		return nil, err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("caption result fetch returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > 1<<20 {
		return nil, fmt.Errorf("caption result is empty or exceeds 1 MiB")
	}
	return data, nil
}

func writeJSONExclusive(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeExclusive(path, append(data, '\n'))
}

func replaceJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func writeProtoExclusive(path string, value proto.Message) error {
	data, err := protojson.MarshalOptions{Indent: "  "}.Marshal(value)
	if err != nil {
		return err
	}
	return writeExclusive(path, append(data, '\n'))
}

func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create immutable evidence %q: %w", path, err)
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func boolPointer(value bool) *bool { return &value }

func captionFloat64(value float64) *float64 { return &value }
