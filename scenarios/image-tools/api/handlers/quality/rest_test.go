package quality

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
	"google.golang.org/protobuf/encoding/protojson"
	"image-tools/internal/jobs"
	internalquality "image-tools/internal/quality"
	"image-tools/internal/storage"
)

type queue struct{ specs []jobs.Spec }

func (q *queue) Submit(_ context.Context, s jobs.Spec) (jobs.Job, error) {
	q.specs = append(q.specs, s)
	return jobs.Job{ID: "quality-job", Operation: s.Operation, Payload: s.Payload}, nil
}
func upload(t *testing.T, manifest string, assets map[string][]byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	require.NoError(t, mw.WriteField("manifest", manifest))
	for k, v := range assets {
		w, err := mw.CreateFormFile(k, "image.png")
		require.NoError(t, err)
		_, err = w.Write(v)
		require.NoError(t, err)
	}
	require.NoError(t, mw.Close())
	r := httptest.NewRequest(http.MethodPost, "/api/v1/quality/evaluate", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	return r
}

func TestEvaluationMultipartRetainsInputsAndRunsDurableReport(t *testing.T) {
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	q := &queue{}
	h := &Deps{Store: store, Jobs: q}
	im := image.NewNRGBA(image.Rect(0, 0, 24, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 24; x++ {
			im.SetNRGBA(x, y, color.NRGBA{uint8(10 * x), uint8(15 * y), 80, 255})
		}
	}
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, im))
	raw := `{"schema_version":1,"name":"fixture","operation":"resize","profile":"image","dataset_id":"owned","candidate_model":"builtin","candidate_revision":"v1","judge":"disabled","cases":[{"id":"one","input":"same","reference":"same","output":"same"}],"budgets":[{"metric":"reference.ssim","minimum":0.999}]}`
	w := httptest.NewRecorder()
	h.evaluate(w, upload(t, raw, map[string][]byte{"same": data.Bytes()}))
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	require.Len(t, q.specs, 1)
	require.Equal(t, jobs.PriorityBatch, q.specs[0].Priority)
	var task internalquality.Task
	require.NoError(t, json.Unmarshal(q.specs[0].Payload, &task))
	m := &qualityv1.EvaluationManifest{}
	require.NoError(t, protojson.Unmarshal(task.Manifest, m))
	require.Contains(t, m.Cases[0].Input, "inputs/quality/")
	require.Equal(t, m.Cases[0].Input, m.Cases[0].Reference)
	result, err := (&internalquality.Runner{Evaluator: &internalquality.Evaluator{}, Store: store}).Run(context.Background(), jobs.Job{ID: "quality-job", Payload: q.specs[0].Payload}, func(int, string) {})
	require.NoError(t, err)
	require.Equal(t, "outputs/quality-job/quality.json", result.Ref)
	rc, _, err := store.Get(context.Background(), result.Ref)
	require.NoError(t, err)
	defer rc.Close()
	var reportData bytes.Buffer
	_, err = reportData.ReadFrom(rc)
	require.NoError(t, err)
	report := &qualityv1.EvaluationReport{}
	require.NoError(t, protojson.Unmarshal(reportData.Bytes(), report))
	require.True(t, report.Passed)
	require.NoError(t, internalquality.VerifyReport(report))
	require.Equal(t, "quality-job", report.JobId)
}

func TestEvaluationDoesNotReadServerFilesystemPaths(t *testing.T) {
	q := &queue{}
	h := &Deps{Store: storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir()), Jobs: q}
	raw := `{"schema_version":1,"operation":"resize","profile":"image","dataset_id":"owned","candidate_model":"builtin","candidate_revision":"v1","judge":"disabled","cases":[{"id":"one","reference":"/etc/passwd","output":"/etc/passwd"}],"budgets":[{"metric":"reference.ssim","minimum":0.99}]}`
	w := httptest.NewRecorder()
	h.evaluate(w, upload(t, raw, nil))
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "missing image part")
	require.Empty(t, q.specs)
}
