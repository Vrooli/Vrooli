package ai

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	testdb "github.com/vrooli/api-core/databasetest"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	"google.golang.org/protobuf/encoding/protojson"
	internalai "image-tools/internal/ai"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/jobrunner"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/storage"
)

// TestSceneGeometry_InstalledMoGeDurableJob is an opt-in qualification of the
// HTTP admission, installed native provider, durable job runner, and ZIP blob.
func TestSceneGeometry_InstalledMoGeDurableJob(t *testing.T) {
	root := os.Getenv("IMAGE_TOOLS_MOGE_QUALIFY_ROOT")
	if root == "" {
		t.Skip("set IMAGE_TOOLS_MOGE_QUALIFY_ROOT to use the pinned installed MoGe model")
	}
	python := os.Getenv("IMAGE_TOOLS_MOGE_QUALIFY_PYTHON")
	if python == "" {
		t.Fatal("IMAGE_TOOLS_MOGE_QUALIFY_PYTHON must name the private Image Tools interpreter")
	}
	registry, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	providers := backends.New()
	if err := internalai.RegisterProviders(providers, nil, nil, python); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := providers.Close(); err != nil {
			t.Error(err)
		}
	})
	sidecar, err := filepath.Abs("../../internal/sidecar/py")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PYTHONPATH", sidecar)
	db := testdb.NewSQLite(t)
	if _, err := db.ExecContext(context.Background(), models.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), internaljobs.Schema()); err != nil {
		t.Fatal(err)
	}
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	engine, err := internalai.NewEngine(internalai.Deps{
		Registry: registry, Backends: providers, Store: store, ModelsRoot: root,
		Probe: capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 4}},
		Enabled: func(context.Context) (models.EnabledFunc, error) {
			return func(id string) bool { return id == "moge-2-vits-normal" }, nil
		},
		ModelInstalled: func(id string) bool { return id == "moge-2-vits-normal" },
	})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := jobrunner.New()
	for op, runner := range engine.BuildRunners() {
		dispatcher.Register(op, runner)
	}
	manager := internaljobs.New(db, internaljobs.Config{Runner: dispatcher.Run, CPUWorkers: 1})
	base, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := manager.Start(base); err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	deps := &Deps{Engine: engine, Store: store, Jobs: manager, Guard: storage.DefaultGuard()}
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/ai/{operation}", deps.submitHandler).Methods(http.MethodPost)
	source := os.Getenv("IMAGE_TOOLS_MOGE_QUALIFY_SOURCE")
	if source == "" {
		source = "../../internal/ai/testdata/aquila-clean-512.png"
	}
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "scene-input.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	params := `{"modelOverride":"moge-2-vits-normal"}`
	if roi := os.Getenv("IMAGE_TOOLS_MOGE_QUALIFY_SUPPORT_ROI"); roi != "" {
		params = fmt.Sprintf(`{"modelOverride":"moge-2-vits-normal","sceneGeometry":{"supportRoi":%s}}`, roi)
	}
	if err := mw.WriteField("params", params); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/scene_geometry", &body)
	request.Header.Set("Content-Type", mw.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("submit=%d %s", response.Code, response.Body.String())
	}
	var accepted aiv1.SubmitAIResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.ModelId != "moge-2-vits-normal" || accepted.Tier != "local-cpu" {
		t.Fatalf("wrong provider selection: %s", response.Body.String())
	}
	job, err := manager.Wait(base, accepted.JobId)
	if err != nil || job.State != internaljobs.StateSucceeded {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	r, mime, err := store.Get(base, job.ResultRef)
	if err != nil {
		t.Fatal(err)
	}
	archiveBytes, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		t.Fatal(err)
	}
	if mime != "application/zip" || len(archiveBytes) < 4 || !bytes.Equal(archiveBytes[:2], []byte("PK")) {
		t.Fatalf("job result is not an application/zip artifact: mime=%q bytes=%d", mime, len(archiveBytes))
	}
	archive, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		t.Fatal(err)
	}
	manifestFile := archive.File[len(archive.File)-1]
	if manifestFile.Name != "manifest.json" {
		t.Fatal("ZIP is missing its final manifest.json member")
	}
	manifestReader, err := manifestFile.Open()
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.NewDecoder(manifestReader).Decode(&manifest); err != nil {
		t.Fatal(err)
	}
	_ = manifestReader.Close()
	if manifest["format"] != "vrooli.image-tools.scene-geometry.v1" {
		t.Fatalf("unsupported result manifest: %v", manifest["format"])
	}
	if os.Getenv("IMAGE_TOOLS_MOGE_QUALIFY_SUPPORT_ROI") != "" {
		if manifest["support_plane"] == nil {
			t.Fatal("requested support ROI did not produce support_plane metadata")
		}
		foundRejectedMask := false
		for _, member := range archive.File {
			foundRejectedMask = foundRejectedMask || member.Name == "support_plane_rejected.npy"
		}
		if !foundRejectedMask {
			t.Fatal("requested support ROI did not produce its rejected-pixel mask")
		}
	}
	if out := os.Getenv("IMAGE_TOOLS_MOGE_QUALIFY_OUTPUT"); out != "" {
		if err := os.WriteFile(out, archiveBytes, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("MoGe durable job=%s model=%s tier=%s archive_bytes=%d zip_sha256=%x artifact=%s", job.ID, accepted.ModelId, accepted.Tier, len(archiveBytes), sha256.Sum256(archiveBytes), job.ResultRef)
	fmt.Fprintf(os.Stdout, "MOGE_DURABLE_JOB job=%s model=%s tier=%s archive_bytes=%d zip_sha256=%x artifact=%s\n", job.ID, accepted.ModelId, accepted.Tier, len(archiveBytes), sha256.Sum256(archiveBytes), job.ResultRef)
}
