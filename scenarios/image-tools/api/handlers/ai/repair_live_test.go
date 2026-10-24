package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"image/png"
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
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection"
	"google.golang.org/protobuf/encoding/protojson"
	internalai "image-tools/internal/ai"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/fetch"
	"image-tools/internal/jobrunner"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/storage"
)

// TestRepair_ManagedLaMaDurableJob qualifies real local model bytes and queue
// ownership. Image quality is reviewed separately on the retained output.
func TestRepair_ManagedLaMaDurableJob(t *testing.T) {
	if os.Getenv("IMAGE_TOOLS_QUALIFY_LAMA") != "1" {
		t.Skip("opt-in managed model download and repair")
	}
	python, source, mask := os.Getenv("IMAGE_TOOLS_QUALIFY_PYTHON"), os.Getenv("IMAGE_TOOLS_QUALIFY_SOURCE"), os.Getenv("IMAGE_TOOLS_QUALIFY_MASK")
	if python == "" || source == "" || mask == "" {
		t.Fatal("qualification requires private interpreter, explicit source and removal mask")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	registry, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	db := testdb.NewSQLite(t)
	for _, schema := range []string{models.Schema(), internaljobs.Schema()} {
		if _, err := db.ExecContext(ctx, schema); err != nil {
			t.Fatal(err)
		}
	}
	root := t.TempDir()
	state := models.NewInstallStore(db)
	installer := models.Installer{Root: root, Reg: registry, State: state, Download: fetch.HTTPDownloader{}}
	started := time.Now()
	installed, err := installer.Install(ctx, "big-lama", nil)
	if err != nil {
		t.Fatal(err)
	}
	if installed.Checksum != "1faef5301d78db7dda502fe59966957ec4b79dd64e16f03ed96913c7a4eb68d6" {
		t.Fatalf("checkpoint digest: %s", installed.Checksum)
	}
	providers := backends.New()
	if err := internalai.RegisterProviders(providers, nil, nil, python); err != nil {
		t.Fatal(err)
	}
	defer providers.Close()
	sidecar, err := filepath.Abs("../../internal/sidecar/py")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PYTHONPATH", sidecar)
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	engine, err := internalai.NewEngine(internalai.Deps{
		Registry: registry, Backends: providers, Store: store, ModelsRoot: root,
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 4}},
		ModelInstalled: func(id string) bool { return id == "big-lama" && installed.Installed },
		Enabled: func(context.Context) (models.EnabledFunc, error) {
			return func(id string) bool { return id == "big-lama" }, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := jobrunner.New()
	for op, runner := range engine.BuildRunners() {
		dispatcher.Register(op, runner)
	}
	manager := internaljobs.New(db, internaljobs.Config{Runner: dispatcher.Run, CPUWorkers: 1})
	if err := manager.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	router := mux.NewRouter()
	deps := Deps{Engine: engine, Store: store, Jobs: manager, Guard: storage.DefaultGuard()}
	router.HandleFunc("/api/v1/ai/{operation}", deps.submitHandler).Methods(http.MethodPost)
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for _, input := range []struct{ name, path string }{{"file", source}, {"mask", mask}} {
		data, err := os.ReadFile(input.path)
		if err != nil {
			t.Fatal(err)
		}
		part, err := form.CreateFormFile(input.name, filepath.Base(input.path))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	params, err := protojson.Marshal(&aiv1.AIParams{
		ModelOverride: "big-lama",
		BackgroundRepair: &aiv1.BackgroundRepairParams{
			MaskExpansionPixels: 16,
			RemovalRegions:      []*selectionv1.Box{{X: 0.08, Y: 0.88, Width: 0.84, Height: 0.11}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := form.WriteField("params", string(params)); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/object_removal", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("admission=%d %s", response.Code, response.Body.String())
	}
	var accepted aiv1.SubmitAIResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.ModelId != "big-lama" || accepted.Tier != "local-cpu" {
		t.Fatalf("selection: %s", response.Body.String())
	}
	job, err := manager.Wait(ctx, accepted.JobId)
	if err != nil || job.State != internaljobs.StateSucceeded {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	reader, mime, err := store.Get(ctx, job.ResultRef)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if mime != "image/png" {
		t.Fatalf("repair MIME=%s", mime)
	}
	if out := os.Getenv("IMAGE_TOOLS_QUALIFY_OUTPUT"); out != "" {
		if err := os.WriteFile(out, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	meta, _ := json.Marshal(job.Meta)
	provenance, err := pngText(data, "vrooli_repair_provenance")
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		EffectiveMaskPNGBase64 string  `json:"effective_mask_png_base64"`
		EffectiveMaskSHA256    string  `json:"effective_mask_png_sha256"`
		RepairElapsedMS        float64 `json:"repair_elapsed_ms"`
		ProcessPeakRSSBytes    int64   `json:"process_peak_rss_bytes"`
		TiledInference         bool    `json:"tiled_inference"`
		TileCrops              []any   `json:"tile_crops"`
	}
	if err := json.Unmarshal([]byte(provenance), &receipt); err != nil {
		t.Fatal(err)
	}
	effectiveMask, err := base64.StdEncoding.DecodeString(receipt.EffectiveMaskPNGBase64)
	if err != nil || len(effectiveMask) == 0 {
		t.Fatalf("effective mask payload missing or invalid: %v", err)
	}
	if out := os.Getenv("IMAGE_TOOLS_QUALIFY_EFFECTIVE_MASK_OUTPUT"); out != "" {
		if err := os.WriteFile(out, effectiveMask, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("native model=big-lama checkpoint_sha256=%s installed_bytes=%d durable_job=%s tier=%s output=%dx%d output_sha256=%x elapsed_ms=%d repair_ms=%.3f sidecar_peak_rss_bytes=%d effective_mask_sha256=%s effective_mask_bytes=%d tiled=%t tile_count=%d result_meta=%s", installed.Checksum, installed.SizeBytes, job.ID, accepted.Tier, decoded.Bounds().Dx(), decoded.Bounds().Dy(), sha256.Sum256(data), time.Since(started).Milliseconds(), receipt.RepairElapsedMS, receipt.ProcessPeakRSSBytes, receipt.EffectiveMaskSHA256, len(effectiveMask), receipt.TiledInference, len(receipt.TileCrops), meta)
}

func pngText(data []byte, key string) (string, error) {
	if len(data) < 8 || !bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return "", io.ErrUnexpectedEOF
	}
	for offset := 8; offset+12 <= len(data); {
		size := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		if size < 0 || offset+12+size > len(data) {
			return "", io.ErrUnexpectedEOF
		}
		kind := string(data[offset+4 : offset+8])
		chunk := data[offset+8 : offset+8+size]
		if kind == "tEXt" {
			if split := bytes.IndexByte(chunk, 0); split >= 0 && string(chunk[:split]) == key {
				return string(chunk[split+1:]), nil
			}
		}
		offset += size + 12
		if kind == "IEND" {
			break
		}
	}
	return "", os.ErrNotExist
}
