package ai

import (
	"bytes"
	"context"
	"errors"
	"image"
	"os"
	"strings"
	"testing"

	"image-tools/internal/adapters"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/models"
	"image-tools/internal/posemap"
	"image-tools/internal/storage"

	"github.com/vrooli/api-core/blobstore"
)

// poseJob is a pose control over a stored image, with a pose reader that records
// the request it was given and the models the job held while it ran.
type poseJob struct {
	eng    *Engine
	req    posemap.Request
	read   bool
	held   []string
	freed  int
	params map[string]string
}

func newPoseJob(t *testing.T, source image.Image, withReader bool) *poseJob {
	t.Helper()
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	data, err := encodePNG(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), "blob/source.png", bytes.NewReader(data), "image/png"); err != nil {
		t.Fatalf("put: %v", err)
	}
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	job := &poseJob{params: map[string]string{}}
	deps := Deps{
		Registry: registry, Backends: backends.New(), Store: store, RightsEvaluator: allowRightsForTest,
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		ModelInstalled: func(string) bool { return true },
		AcquireModelUse: func(id string) (func(), error) {
			job.held = append(job.held, id)
			return func() { job.freed++ }, nil
		},
	}
	if withReader {
		deps.PosePreprocessor = func(_ context.Context, req posemap.Request) error {
			job.req, job.read = req, true
			return os.WriteFile(req.Output, []byte("synthetic OpenPose map"), 0o600)
		}
	}
	if job.eng, err = NewEngine(deps); err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return job
}

func (j *poseJob) materialize(t *testing.T, arch models.Architecture) ([]adapters.ResolvedAdapter, error) {
	t.Helper()
	pl := &Payload{InputKey: "blob/source.png", Params: j.params, Adapters: []adapters.ResolvedAdapter{
		{ID: "controlnet-openpose-test", Kind: adapters.KindControlNet, Architecture: arch, Preprocessor: adapters.PreprocessorPose, ConditioningImageKey: "blob/source.png"},
	}}
	return j.eng.materializeConditioningImages(context.Background(), t.TempDir(), "job-1", pl, "", func(int, string) {})
}

func TestPoseControlReadsTheWholeSourceAtThePoseShape(t *testing.T) {
	j := newPoseJob(t, chartPage(400, 800), true)
	resolved, err := j.materialize(t, models.ArchSDXL)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if !j.read || j.req.Box != nil || j.req.Prepare != "" {
		t.Fatalf("reader request = %+v (read %v), want the whole source with no preparation", j.req, j.read)
	}
	if j.req.Width != 704 || j.req.Height != 1472 {
		t.Errorf("canvas %dx%d, want the 1:2 shape at about a megapixel (704x1472)", j.req.Width, j.req.Height)
	}
	if j.params["width"] != "704" || j.params["height"] != "1472" {
		t.Errorf("the generation must run on the control's canvas, params = %v", j.params)
	}
	if resolved[0].ConditioningImageKey != j.req.Output {
		t.Errorf("control image %q is not the reader's output %q", resolved[0].ConditioningImageKey, j.req.Output)
	}
	if len(j.held) != 1 || j.held[0] != posemap.ModelID || j.freed != 1 {
		t.Errorf("pose model held %v, released %d times; want %s held and released once", j.held, j.freed, posemap.ModelID)
	}
}

func TestPoseControlReadsOneChartCellAndKeepsAGivenSize(t *testing.T) {
	page := chartPage(1100, 1100)
	for _, r := range []image.Rectangle{image.Rect(60, 40, 260, 400), image.Rect(420, 50, 640, 420)} {
		figure(page, r)
	}
	j := newPoseJob(t, page, true)
	j.params[ParamPoseCell], j.params[ParamPosePrepare] = "2", "line_art"
	j.params["width"], j.params["height"] = "832", "1216"
	if _, err := j.materialize(t, models.ArchSDXL); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	box := j.req.Box
	if box == nil || box[0] > 420 || box[1] > 50 || box[2] < 640 || box[3] < 420 || box[0] < 260 {
		t.Fatalf("cell 2 box = %v, want the second figure (420,50)-(640,420) and none of the first", box)
	}
	if j.req.Width != 832 || j.req.Height != 1216 || j.params["width"] != "832" {
		t.Errorf("a given size must stand, got %dx%d with params %v", j.req.Width, j.req.Height, j.params)
	}
	if j.req.Prepare != "line_art" {
		t.Errorf("prepare = %q, want line_art", j.req.Prepare)
	}
}

func TestPoseControlSizesOlderModelsSmaller(t *testing.T) {
	j := newPoseJob(t, chartPage(400, 400), true)
	if _, err := j.materialize(t, models.ArchSD15); err != nil {
		t.Fatal(err)
	}
	if j.req.Width != 512 || j.req.Height != 512 {
		t.Errorf("SD 1.5 canvas %dx%d, want 512x512", j.req.Width, j.req.Height)
	}
}

func TestPoseControlRefusesWhatItCannotRead(t *testing.T) {
	page := chartPage(600, 600)
	figure(page, image.Rect(100, 50, 400, 500))
	for name, c := range map[string]struct {
		cell   string
		reader bool
		want   string
	}{
		"a cell the chart lacks": {"4", true, "the chart has 1 cells"},
		"not a number":           {"top", true, "must be a chart cell number"},
		"no reader":              {"", false, "pose preprocessor is unavailable"},
	} {
		t.Run(name, func(t *testing.T) {
			j := newPoseJob(t, page, c.reader)
			if c.cell != "" {
				j.params[ParamPoseCell] = c.cell
			}
			if _, err := j.materialize(t, models.ArchSDXL); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error = %v, want %q", err, c.want)
			}
		})
	}
}

func TestPoseControlFailsWhenThePoseModelCannotBeHeld(t *testing.T) {
	j := newPoseJob(t, chartPage(200, 200), true)
	j.eng.deps.ModelInstalled = func(string) bool { return false }
	j.eng.deps.AcquireModel = nil
	_, err := j.materialize(t, models.ArchSDXL)
	if !errors.Is(err, ErrModelNotInstalled) || !strings.Contains(err.Error(), "pose reading") {
		t.Fatalf("error = %v, want ErrModelNotInstalled naming the pose reading", err)
	}
	if j.read {
		t.Fatal("the reader ran without its model")
	}
}
