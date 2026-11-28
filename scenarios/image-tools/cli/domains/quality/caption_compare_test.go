package quality

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
)

type fakeCaptionRuntime struct {
	submits     int
	failAt      int
	model       string
	outputs     map[string][]byte
	prompts     []string
	cancel      context.CancelFunc
	cancelCount int
}

func (f *fakeCaptionRuntime) preflight(context.Context, string) (captionModelReady, error) {
	return captionModelReady{}, errors.New("preflight is not part of this task-set test")
}

func (f *fakeCaptionRuntime) submit(_ context.Context, _ string, prompt, model string) (*aiv1.SubmitAIResponse, error) {
	f.submits++
	f.prompts = append(f.prompts, prompt)
	if f.failAt != 0 && f.submits == f.failAt {
		return nil, errors.New("shared admission denied")
	}
	id := "job-" + string(rune('0'+f.submits))
	ref := "result-" + id
	data, _ := json.Marshal(captionOutput{Caption: "A synthetic scene.", Backend: "fake-cpu", Model: model})
	if f.outputs == nil {
		f.outputs = map[string][]byte{}
	}
	f.outputs[ref] = data
	if f.cancel != nil {
		f.cancel()
	}
	return &aiv1.SubmitAIResponse{JobId: id, ModelId: model, Tier: "local-cpu"}, nil
}

func (f *fakeCaptionRuntime) wait(ctx context.Context, id string) (*jobsv1.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &jobsv1.Job{Id: id, State: jobsv1.JobState_JOB_STATE_SUCCEEDED, ResultRef: "result-" + id, ResultMeta: map[string]string{"model": f.model, "backend": "fake-cpu", "tier": "local-cpu"}}, nil
}

func (f *fakeCaptionRuntime) cancelAndWait(_ context.Context, id string) (*jobsv1.Job, error) {
	f.cancelCount++
	return &jobsv1.Job{Id: id, State: jobsv1.JobState_JOB_STATE_CANCELED}, nil
}

func (f *fakeCaptionRuntime) fetch(_ context.Context, ref string) ([]byte, error) {
	return f.outputs[ref], nil
}

func (*fakeCaptionRuntime) evaluate(context.Context, string, string) (*qualityv1.EvaluationReport, error) {
	return nil, errors.New("evaluation is not part of this task-set test")
}

func frozenCaptionCorpusPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "api", "internal", "quality", "testdata", "caption-v1", "corpus.json")
	absolute, err := filepath.Abs(path)
	require.NoError(t, err)
	return absolute
}

func TestLoadCaptionCorpusRequiresFrozenManifestAndVerifiesAllSources(t *testing.T) {
	path := frozenCaptionCorpusPath(t)
	corpus, raw, _, err := loadCaptionCorpus(path)
	require.NoError(t, err)
	require.Len(t, corpus.Images, 5)
	require.NotEmpty(t, raw)
}

func TestCaptionTaskSetRunsSeriallyAndBuildsJudgeBoundManifest(t *testing.T) {
	corpus, _, corpusDir, err := loadCaptionCorpus(frozenCaptionCorpusPath(t))
	require.NoError(t, err)
	model := captionModelReady{ID: "smolvlm-256m", Revision: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Support: "native"}
	runtime := &fakeCaptionRuntime{model: model.ID}
	run := &captionModelRun{}
	runPath := filepath.Join(t.TempDir(), "run.json")
	require.NoError(t, os.WriteFile(runPath, []byte("{}\n"), 0o600))
	manifest, err := runCaptionTaskSet(context.Background(), runtime, corpus, corpusDir, model, "visual-question-answering", run, runPath)
	require.NoError(t, err)
	require.Equal(t, 5, runtime.submits)
	require.Len(t, manifest.Cases, 5)
	require.Equal(t, "required", manifest.Judge)
	require.Equal(t, captionJudge, manifest.JudgeModel)
	require.Equal(t, "What shape and color is above the gray bar?", runtime.prompts[0])
	require.Equal(t, "A red oval.", manifest.Cases[0].ReferenceText)
	require.Equal(t, "A synthetic scene.", manifest.Cases[0].OutputText)
	require.Equal(t, "smolvlm-256m", manifest.CandidateModel)
	require.Equal(t, model.Revision, manifest.CandidateRevision)
	require.Len(t, run.Tasks, 5)
}

func TestCaptionTaskSetRetainsAdmissionFailureWithoutRetry(t *testing.T) {
	corpus, _, corpusDir, err := loadCaptionCorpus(frozenCaptionCorpusPath(t))
	require.NoError(t, err)
	model := captionModelReady{ID: "smolvlm-256m", Revision: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	runtime := &fakeCaptionRuntime{model: model.ID, failAt: 1}
	run := &captionModelRun{}
	runPath := filepath.Join(t.TempDir(), "run.json")
	require.NoError(t, os.WriteFile(runPath, []byte("{}\n"), 0o600))
	manifest, err := runCaptionTaskSet(context.Background(), runtime, corpus, corpusDir, model, "caption", run, runPath)
	require.ErrorContains(t, err, "shared admission denied")
	require.Nil(t, manifest)
	require.Equal(t, 1, runtime.submits, "a refused job must not be retried")
	require.Len(t, run.Tasks, 1)
	require.Equal(t, "submit-failed", run.Tasks[0].State)
	require.Contains(t, run.Tasks[0].Error, "shared admission denied")
}

func TestCaptionTaskSetCancelsAcceptedJobAndWaitsForTerminalCleanup(t *testing.T) {
	corpus, _, corpusDir, err := loadCaptionCorpus(frozenCaptionCorpusPath(t))
	require.NoError(t, err)
	model := captionModelReady{ID: "smolvlm-256m", Revision: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runtime := &fakeCaptionRuntime{model: model.ID, cancel: cancel}
	run := &captionModelRun{}
	runPath := filepath.Join(t.TempDir(), "run.json")
	require.NoError(t, os.WriteFile(runPath, []byte("{}\n"), 0o600))
	_, err = runCaptionTaskSet(ctx, runtime, corpus, corpusDir, model, "caption", run, runPath)
	require.ErrorContains(t, err, "canceled-cleaned")
	require.Equal(t, 1, runtime.submits, "cancellation stops before submitting another case")
	require.Equal(t, 1, runtime.cancelCount, "accepted job receives one owner cancellation request")
	require.Len(t, run.Tasks, 1)
	require.Equal(t, "canceled-cleaned", run.Tasks[0].State)
}
