package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	"image-tools/internal/fetch"
	"image-tools/internal/jobrunner"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/storage"
)

// Qualification uses already installed read-only weights/runtime, a temporary
// SQLite database and in-memory blobs. It never downloads, changes live model
// settings, starts another scenario or touches its jobs.
func TestSegment_InstalledMobileSAMDurableJob(t *testing.T) {
	root, source := os.Getenv("IMAGE_TOOLS_QUALIFY_MODEL_ROOT"), os.Getenv("IMAGE_TOOLS_QUALIFY_SOURCE")
	if root == "" || source == "" {
		t.Skip("opt-in installed-model qualification")
	}
	registry, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	providers := backends.New()
	python := filepath.Join(root, "pyenv", "bin", "python")
	sidecar, err := filepath.Abs("../../internal/sidecar/py")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PYTHONPATH", sidecar)
	if err := internalai.RegisterProviders(providers, nil, nil, python); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := providers.Close(); err != nil {
			t.Error(err)
		}
	})
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	engine, err := internalai.NewEngine(internalai.Deps{Registry: registry, Backends: providers, Store: store, ModelsRoot: root,
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 4}},
		ModelInstalled: func(id string) bool { return id == "mobilesam" }})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := jobrunner.New()
	for op, runner := range engine.BuildRunners() {
		dispatcher.Register(op, runner)
	}
	db := testdb.NewSQLite(t)
	if _, err := db.ExecContext(context.Background(), internaljobs.Schema()); err != nil {
		t.Fatal(err)
	}
	manager := internaljobs.New(db, internaljobs.Config{Runner: dispatcher.Run, CPUWorkers: 1})
	base, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := manager.Start(base); err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	deps := &Deps{Engine: engine, Store: store, Jobs: manager, Guard: storage.DefaultGuard()}
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/ai/{operation}", deps.submitHandler).Methods(http.MethodPost)
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "device.png")
	if err != nil {
		t.Fatal(err)
	}
	part.Write(data)
	mw.WriteField("params", `{"modelOverride":"mobilesam","segmentation":{"box":{"x":0.06,"y":0.02,"width":0.8,"height":0.96}}}`)
	mw.Close()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/segment", &body)
	request.Header.Set("Content-Type", mw.FormDataContentType())
	requestContext, disconnect := context.WithCancel(context.Background())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request.WithContext(requestContext))
	disconnect() // work belongs to the job manager
	if response.Code != http.StatusAccepted {
		t.Fatalf("submit=%d %s", response.Code, response.Body.String())
	}
	var accepted aiv1.SubmitAIResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.ModelId != "mobilesam" || accepted.Tier != "local-cpu" {
		t.Fatalf("wrong selection: %s", response.Body.String())
	}
	job, err := manager.Wait(base, accepted.JobId)
	if err != nil || job.State != internaljobs.StateSucceeded {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	r, _, err := store.Get(base, job.ResultRef)
	if err != nil {
		t.Fatal(err)
	}
	mask, err := io.ReadAll(r)
	r.Close()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(mask))
	if err != nil {
		t.Fatal(err)
	}
	original, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != original.Width || decoded.Bounds().Dy() != original.Height {
		t.Fatal("mask lost source viewport")
	}
	var selected int
	for y := 0; y < original.Height; y++ {
		for x := 0; x < original.Width; x++ {
			v, _, _, _ := decoded.At(x, y).RGBA()
			if v != 0 && v != 65535 {
				t.Fatal("mask is not binary")
			}
			if v > 0 {
				selected++
			}
		}
	}
	if selected == 0 || selected == original.Width*original.Height {
		t.Fatal("degenerate mask")
	}
	if out := os.Getenv("IMAGE_TOOLS_QUALIFY_OUTPUT"); out != "" {
		if err := os.WriteFile(out, mask, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("native durable job=%s model=%s tier=%s dimensions=%dx%d selected=%d sha256=%s", job.ID, accepted.ModelId, accepted.Tier, original.Width, original.Height, selected, fmt.Sprintf("%x", sha256.Sum256(mask)))
}

// TestSegment_ManagedInstallSAM2DurableJob exercises a pinned SAM 2.1 artifact
// through the production Installer and a real durable segmentation job. It is
// opt-in because model installation and CPU inference are substantial.
func TestSegment_ManagedInstallSAM2DurableJob(t *testing.T) {
	if os.Getenv("IMAGE_TOOLS_QUALIFY_SAM2") != "1" {
		t.Skip("opt-in SAM 2.1 managed-install qualification")
	}
	python := os.Getenv("IMAGE_TOOLS_QUALIFY_PYTHON")
	if python == "" {
		t.Fatal("IMAGE_TOOLS_QUALIFY_PYTHON must name the Image Tools private interpreter")
	}
	qualifyGPU := os.Getenv("IMAGE_TOOLS_QUALIFY_GPU") == "1"
	qualifyContention := os.Getenv("IMAGE_TOOLS_QUALIFY_CONTENTION") == "1"
	if qualifyContention && !qualifyGPU {
		t.Fatal("IMAGE_TOOLS_QUALIFY_CONTENTION requires IMAGE_TOOLS_QUALIFY_GPU=1")
	}
	modelID := os.Getenv("IMAGE_TOOLS_QUALIFY_MODEL")
	if modelID == "" {
		modelID = "sam-2.1-tiny"
	}
	registry, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	modelRoot := t.TempDir()
	db := testdb.NewSQLite(t)
	if _, err := db.ExecContext(context.Background(), models.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), internaljobs.Schema()); err != nil {
		t.Fatal(err)
	}
	installer := &models.Installer{
		Root: modelRoot, Reg: registry, State: models.NewInstallStore(db), Download: fetch.HTTPDownloader{},
	}
	installStarted := time.Now()
	lastInstallPercent := -1
	record, err := installer.Install(context.Background(), modelID, func(p int, message string) {
		if p != lastInstallPercent {
			t.Logf("model install %d%% %s", p, message)
			lastInstallPercent = p
		}
	})
	if err != nil {
		t.Fatalf("native SAM 2.1 model install: %v", err)
	}
	if record.SizeBytes <= 0 {
		t.Fatalf("installed payload too small: %d bytes", record.SizeBytes)
	}
	var configSHA256, weightsSHA256 string
	if modelID == "sam-2.1-tiny" || modelID == "sam-2.1-base-plus" {
		var weightsSize int64
		weightsSHA256, weightsSize, err = fetch.HashFile(filepath.Join(record.Path, "model.safetensors"))
		if err != nil {
			t.Fatalf("hash installed SAM2 weights: %v", err)
		}
		var expectedConfigSHA, expectedWeightsSHA string
		if modelID == "sam-2.1-tiny" {
			configSHA256, _, err = fetch.HashFile(filepath.Join(record.Path, "config.json"))
			if err != nil {
				t.Fatalf("hash installed SAM2 config: %v", err)
			}
		}
		for _, model := range registry.Models() {
			if model.ID != modelID {
				continue
			}
			for _, asset := range model.Source.Assets {
				switch asset.Filename {
				case "config.json":
					expectedConfigSHA = asset.SHA256
				case "model.safetensors":
					expectedWeightsSHA = asset.SHA256
				}
			}
		}
		if modelID == "sam-2.1-tiny" && !strings.EqualFold(configSHA256, expectedConfigSHA) || !strings.EqualFold(weightsSHA256, expectedWeightsSHA) {
			t.Fatalf("installed SAM2 asset hashes differ from pinned catalog: config=%s expected=%s weights=%s expected=%s",
				configSHA256, expectedConfigSHA, weightsSHA256, expectedWeightsSHA)
		}
		if modelID == "sam-2.1-tiny" {
			t.Logf("SAM2 installed asset hashes config.json=%s model.safetensors=%s (%d bytes)", configSHA256, weightsSHA256, weightsSize)
		} else {
			t.Logf("SAM2 installed asset hash model.safetensors=%s (%d bytes)", weightsSHA256, weightsSize)
		}
	}

	providers := backends.New()
	var sidecarMetrics string
	var sidecarPID int
	var sidecarExitCode int
	commandRunner := func(ctx context.Context, name string, args []string) error {
		cmd := exec.CommandContext(ctx, name, args...)
		out, runErr := cmd.CombinedOutput()
		if cmd.Process != nil {
			sidecarPID = cmd.Process.Pid
		}
		if cmd.ProcessState != nil {
			sidecarExitCode = cmd.ProcessState.ExitCode()
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, `"sam2_metrics"`) {
				sidecarMetrics = line
			}
		}
		if runErr != nil {
			return fmt.Errorf("%s %v: %w (%s)", name, args, runErr, string(out))
		}
		return nil
	}
	if err := internalai.RegisterProviders(providers, nil, commandRunner, python); err != nil {
		t.Fatalf("register providers: %v", err)
	}
	sidecar, err := filepath.Abs("../../internal/sidecar/py")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PYTHONPATH", sidecar)
	t.Cleanup(func() {
		if err := providers.Close(); err != nil {
			t.Error(err)
		}
	})
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	var hostProbe capabilities.Probe = capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 4}}
	var capacityBroker internalai.CapacityBroker
	if qualifyGPU {
		hostProbe = capabilities.NewCLIProbe()
		capacityBroker = &internalai.CLICapacityBroker{}
		host, probeErr := hostProbe.Inventory(context.Background())
		if probeErr != nil {
			t.Fatalf("native GPU inventory: %v", probeErr)
		}
		for _, gpu := range host.GPUs {
			free, known := gpu.VRAMFreeBytes()
			t.Logf("native GPU inventory index=%d uuid=%s name=%s free_bytes=%d free_known=%t", gpu.Index, gpu.UUID, gpu.Name, free, known)
		}
	}
	var blockerClaim internalai.CapacityLease
	if qualifyContention {
		host, probeErr := hostProbe.Inventory(context.Background())
		if probeErr != nil || len(host.GPUs) == 0 {
			t.Fatalf("fresh native GPU inventory for contention: host=%+v err=%v", host, probeErr)
		}
		blocker, claimErr := capacityBroker.Claim(context.Background(), "image-tools:qualification-contention", int64(9)<<30, internalai.CapacityClaimOptions{
			Priority: "service", AllowReclaim: false, GPUUUID: host.GPUs[0].UUID,
		})
		if claimErr != nil {
			t.Fatalf("reserve bounded no-reclaim contention window: %v", claimErr)
		}
		if blocker.DegradeToCPU || blocker.ClaimID == "" {
			t.Fatalf("no-reclaim contention reservation not granted: %+v", blocker)
		}
		blockerClaim = blocker
		t.Logf("temporary no-reclaim broker contention claim=%s reserved_bytes=%d", blocker.ClaimID, int64(9)<<30)
	}
	if blockerClaim.ClaimID != "" {
		defer func() {
			if err := capacityBroker.Release(context.Background(), blockerClaim); err != nil {
				t.Errorf("release temporary contention claim %s: %v", blockerClaim.ClaimID, err)
			}
		}()
	}
	engine, err := internalai.NewEngine(internalai.Deps{Registry: registry, Backends: providers, Store: store, ModelsRoot: modelRoot,
		Probe:    hostProbe,
		Capacity: capacityBroker,
		Enabled: func(context.Context) (models.EnabledFunc, error) {
			return func(id string) bool { return id == modelID }, nil
		},
		ModelInstalled: func(id string) bool { return id == modelID },
	})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := jobrunner.New()
	for op, runner := range engine.BuildRunners() {
		dispatcher.Register(op, runner)
	}
	manager := internaljobs.New(db, internaljobs.Config{Runner: dispatcher.Run, CPUWorkers: 1})
	base, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	if err := manager.Start(base); err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	deps := &Deps{Engine: engine, Store: store, Jobs: manager, Guard: storage.DefaultGuard()}
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/ai/{operation}", deps.submitHandler).Methods(http.MethodPost)
	source := os.Getenv("IMAGE_TOOLS_QUALIFY_SOURCE")
	if source == "" {
		source = "../../internal/ai/testdata/aquila-clean-512.png"
	}
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "thin-line-art.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	allowReclaim := os.Getenv("IMAGE_TOOLS_QUALIFY_ALLOW_RECLAIM") != "0"
	prompts := fmt.Sprintf(`{"modelOverride":%q,"allowReclaim":%t,"segmentation":{"points":[{"x":0.61,"y":0.56},{"x":0.02,"y":0.02,"negative":true}],"box":{"x":0.18,"y":0.18,"width":0.64,"height":0.68}}}`, modelID, allowReclaim)
	if os.Getenv("IMAGE_TOOLS_QUALIFY_SCENE") == "device" {
		prompts = fmt.Sprintf(`{"modelOverride":%q,"allowReclaim":%t,"segmentation":{"points":[{"x":0.46,"y":0.51},{"x":0.02,"y":0.02,"negative":true}],"box":{"x":0.06,"y":0.02,"width":0.8,"height":0.96}}}`, modelID, allowReclaim)
	}
	if err := mw.WriteField("params", prompts); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/segment", &body)
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
	if accepted.ModelId != modelID {
		t.Fatalf("wrong selection: %s", response.Body.String())
	}
	if qualifyContention && accepted.Tier != "local-gpu" {
		t.Fatalf("test host should plan GPU work before the capacity claim degrades it: %s", response.Body.String())
	}
	job, err := manager.Wait(base, accepted.JobId)
	if err != nil || job.State != internaljobs.StateSucceeded {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	if qualifyGPU {
		if accepted.Tier == "local-gpu" && !qualifyContention && !strings.Contains(sidecarMetrics, `"device": "cuda"`) {
			t.Fatalf("request selected GPU but sidecar did not report CUDA: %s", sidecarMetrics)
		}
		if accepted.Tier == "local-gpu" && qualifyContention && !strings.Contains(sidecarMetrics, `"device": "cpu"`) {
			t.Fatalf("capacity contention should degrade to CPU but sidecar metrics were: %s", sidecarMetrics)
		}
		if accepted.Tier == "local-cpu" && !strings.Contains(sidecarMetrics, `"device": "cpu"`) {
			t.Fatalf("request selected CPU but sidecar did not report CPU: %s", sidecarMetrics)
		}
		cmd := exec.CommandContext(base, "vrooli", "capacity", "list", "--json")
		output, listErr := cmd.Output()
		if listErr != nil {
			t.Fatalf("list native capacity claims: %v", listErr)
		}
		var claims struct {
			Claims []struct {
				OwnerID string `json:"owner_id"`
				Status  string `json:"status"`
			} `json:"claims"`
		}
		if err := json.Unmarshal(output, &claims); err != nil {
			t.Fatalf("decode native capacity claims: %v", err)
		}
		for _, claim := range claims.Claims {
			if claim.OwnerID == "image-tools:"+job.ID && (claim.Status == "granted" || claim.Status == "active") {
				t.Fatalf("capacity claim remains after completed job: %+v", claim)
			}
		}
	}
	r, _, err := store.Get(base, job.ResultRef)
	if err != nil {
		t.Fatal(err)
	}
	mask, err := io.ReadAll(r)
	r.Close()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(mask))
	if err != nil {
		t.Fatal(err)
	}
	original, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != original.Width || decoded.Bounds().Dy() != original.Height {
		t.Fatal("mask lost source viewport")
	}
	var selected int
	for y := 0; y < original.Height; y++ {
		for x := 0; x < original.Width; x++ {
			v, _, _, _ := decoded.At(x, y).RGBA()
			if v != 0 && v != 65535 {
				t.Fatal("mask is not binary")
			}
			if v > 0 {
				selected++
			}
		}
	}
	if selected == 0 || selected == original.Width*original.Height {
		t.Fatal("degenerate mask")
	}
	negativeX, negativeY := original.Width/50, original.Height/50
	if v, _, _, _ := decoded.At(negativeX, negativeY).RGBA(); v != 0 {
		t.Fatalf("negative prompt point remains selected at (%d,%d)", negativeX, negativeY)
	}
	if out := os.Getenv("IMAGE_TOOLS_QUALIFY_SAM2_OUTPUT"); out != "" {
		if err := os.WriteFile(out, mask, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if blockerClaim.ClaimID != "" {
		if err := capacityBroker.Release(base, blockerClaim); err != nil {
			t.Fatalf("release temporary contention claim %s: %v", blockerClaim.ClaimID, err)
		}
		blockerClaim = internalai.CapacityLease{}
	}
	jobMS := int64(0)
	if job.FinishedAt != nil {
		jobMS = job.FinishedAt.Sub(job.CreatedAt).Milliseconds()
	}
	t.Logf("SAM2 managed install bytes=%d primary_asset=config.json primary_asset_sha256=%s config_sha256=%s weights_sha256=%s install_ms=%d durable_job=%s model=%s tier=%s device=%s dimensions=%dx%d selected=%d mask_sha256=%x sidecar=%s sidecar_pid=%d sidecar_exit=%d job_ms=%d",
		record.SizeBytes, record.Checksum, configSHA256, weightsSHA256, time.Since(installStarted).Milliseconds(), job.ID, accepted.ModelId, accepted.Tier,
		map[bool]string{true: "cuda-or-cpu-from-broker", false: "cpu"}[qualifyGPU], original.Width, original.Height,
		selected, sha256.Sum256(mask), sidecarMetrics, sidecarPID, sidecarExitCode, jobMS)
}
