package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	internaladapters "image-tools/internal/adapters"
	internalai "image-tools/internal/ai"
	internalanalysis "image-tools/internal/analysis"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	internalcleanup "image-tools/internal/cleanup"
	"image-tools/internal/fetch"
	internalhosttool "image-tools/internal/hosttool"
	"image-tools/internal/jobrunner"
	internaljobs "image-tools/internal/jobs"
	internalmeasures "image-tools/internal/measures"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/modules"
	"image-tools/internal/provision"
	"image-tools/internal/pydeps"
	internalquality "image-tools/internal/quality"
	"image-tools/internal/safety"
	"image-tools/internal/server"
	"image-tools/internal/sidecar"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/pyenv-go"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	_ "modernc.org/sqlite"

	adaptersH "image-tools/handlers/adapters"
	aiH "image-tools/handlers/ai"
	analysisH "image-tools/handlers/analysis"
	diffH "image-tools/handlers/diff"
	healthH "image-tools/handlers/health"
	jobsH "image-tools/handlers/jobs"
	looksH "image-tools/handlers/looks"
	measuresH "image-tools/handlers/measures"
	modelsH "image-tools/handlers/models"
	opsH "image-tools/handlers/ops"
	preparationH "image-tools/handlers/preparation"
	qualityH "image-tools/handlers/quality"
	safetyH "image-tools/handlers/safety"
	selectionH "image-tools/handlers/selection"

	internalstorage "image-tools/internal/storage"
)

const (
	cpuWorkersEnv         = "IMAGE_TOOLS_CPU_WORKERS"
	networkWorkersEnv     = "IMAGE_TOOLS_NETWORK_WORKERS"
	installMBPerSecondEnv = "IMAGE_TOOLS_INSTALL_MB_PER_SECOND"
	defaultCPUWorkers     = 4
	minCPUWorkers         = 1
	maxCPUWorkers         = 32
	defaultNetworkWorkers = 4
	minNetworkWorkers     = 1
	maxNetworkWorkers     = 32
	minInstallMBPerSecond = 1
	maxInstallMBPerSecond = 10000
)

type runtimeConfig struct {
	CPUWorkers         int
	NetworkWorkers     int
	InstallMBPerSecond int
}

func loadRuntimeConfig() (runtimeConfig, error) {
	cpuWorkers, err := intFromEnv(cpuWorkersEnv, defaultCPUWorkers, minCPUWorkers, maxCPUWorkers)
	if err != nil {
		return runtimeConfig{}, err
	}
	networkWorkers, err := intFromEnv(networkWorkersEnv, defaultNetworkWorkers, minNetworkWorkers, maxNetworkWorkers)
	if err != nil {
		return runtimeConfig{}, err
	}
	installMBPerSecond, err := intFromEnv(installMBPerSecondEnv, internalmodels.DefaultInstallMBPerSecond, minInstallMBPerSecond, maxInstallMBPerSecond)
	if err != nil {
		return runtimeConfig{}, err
	}
	return runtimeConfig{CPUWorkers: cpuWorkers, NetworkWorkers: networkWorkers, InstallMBPerSecond: installMBPerSecond}, nil
}

func intFromEnv(name string, def, min, max int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	if v < min || v > max {
		return 0, fmt.Errorf("%s must be between %d and %d", name, min, max)
	}
	return v, nil
}

// jobDurationMS is a finalized job's run wall-clock (StartedAt → FinishedAt) in
// milliseconds, or 0 when timing is incomplete.
func jobDurationMS(j internaljobs.Job) int64 {
	if j.StartedAt == nil || j.FinishedAt == nil {
		return 0
	}
	return j.FinishedAt.Sub(*j.StartedAt).Milliseconds()
}

// jobQueueMS is a job's queue wait (CreatedAt → StartedAt) in milliseconds.
func jobQueueMS(j internaljobs.Job) int64 {
	if j.StartedAt == nil {
		return 0
	}
	return j.StartedAt.Sub(j.CreatedAt).Milliseconds()
}

func jobSampleAndTrace(j internaljobs.Job) (internalmeasures.Sample, internalmeasures.JobTrace) {
	durationMS := jobDurationMS(j)
	queueMS := jobQueueMS(j)
	sample := internalmeasures.Sample{
		Operation:   j.Operation,
		State:       string(j.State),
		DurationMS:  durationMS,
		QueueWaitMS: queueMS,
	}
	trace := internalmeasures.JobTrace{
		JobID:       j.ID,
		Operation:   j.Operation,
		Lane:        string(j.Lane),
		State:       string(j.State),
		DurationMS:  durationMS,
		QueueWaitMS: queueMS,
		ResultRef:   j.ResultRef,
		Error:       j.Error,
	}

	var payload internalai.Payload
	if err := json.Unmarshal(j.Payload, &payload); err == nil {
		if payload.Operation != "" {
			sample.Operation = payload.Operation
			trace.Operation = payload.Operation
		}
		sample.ModelID = payload.ModelID
		sample.Tier = payload.Tier
		sample.FallbackUsed = payload.GPU && payload.Tier != "" && payload.Tier != backends.TierLocalGPU.String()
		trace.ModelID = payload.ModelID
		trace.Backend = payload.Backend
		trace.Tier = payload.Tier
	}

	return sample, trace
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "image-tools"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "image-tools",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	configureTestPoolSchemas(db)

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	// Load the validated model registry once (its seed-integrity invariants are
	// asserted here, so a bad seed fails loud at boot) and share it with the
	// models domain. Host hardware facts come through the root vrooli CLI.
	registry, err := internalmodels.Load()
	if err != nil {
		log.Fatalf("model registry load failed: %v", err)
	}
	// The conditioning-adapter catalog (LoRA / ControlNet / IP-Adapter) is the
	// sibling of the model registry; its seed-integrity invariants are likewise
	// asserted at boot so a bad seed fails loud here.
	adapterRegistry, err := internaladapters.Load()
	if err != nil {
		log.Fatalf("adapter registry load failed: %v", err)
	}
	runtimeCfg, err := loadRuntimeConfig()
	if err != nil {
		log.Fatalf("runtime configuration failed: %v", err)
	}
	probe := capabilities.NewCLIProbe()

	// The durable job Manager runs under a server-lifetime context so a client
	// disconnect never destroys in-flight work. Operation domains register their
	// execution handlers on the dispatcher; in Phase 1 none are registered yet,
	// so a submitted job for an unknown op fails cleanly (and nothing submits).
	jobCtx, cancelJobs := context.WithCancel(context.Background())
	dispatcher := jobrunner.New(internalai.AdmitCPUWork)

	// Runtime measures (IMG-P0-012): the recorder captures op latency + queue-wait
	// + outcome for every finalized job via the Manager's OnComplete hook. Recorded
	// under the server-lifetime context so it is independent of any caller.
	measureRec := internalmeasures.NewRecorder(db)
	jobManager := internaljobs.New(db, internaljobs.Config{
		Runner:         dispatcher.Run,
		Admit:          dispatcher.Admit,
		Clock:          schedule.System(),
		CPUWorkers:     runtimeCfg.CPUWorkers,
		NetworkWorkers: runtimeCfg.NetworkWorkers,
		OnComplete: func(job internaljobs.Job) {
			sample, trace := jobSampleAndTrace(job)
			if err := measureRec.Record(jobCtx, sample); err != nil {
				log.Printf("measures: observe job %s: %v", job.ID, err)
			}
			if err := measureRec.RecordJobTrace(jobCtx, trace); err != nil {
				log.Printf("measures: trace job %s: %v", job.ID, err)
			}
			if err := measureRec.RecordModelJobProfiles(jobCtx, sample, job.ID, job.Meta); err != nil {
				log.Printf("measures: model profiles for job %s: %v", job.ID, err)
			}
		},
	})
	if err := jobManager.Start(jobCtx); err != nil {
		cancelJobs()
		log.Fatalf("job manager start failed: %v", err)
	}

	// Image blobs persist under the api-core storage substrate (outside the
	// repo, shadow/variant aware), addressed by opaque keys — pixels never go
	// in SQLite. The ops domain stores op inputs/outputs here.
	blobNamespace, err := storage.ScenarioNamespace("image-tools")
	if err != nil {
		log.Fatalf("resolve image-tools storage namespace: %v", err)
	}
	primaryFileRoots, err := internalstorage.ResolvePaths(blobNamespace)
	if err != nil {
		log.Fatalf("file storage roots init failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)
	blobStore, err := internalstorage.NewWithRoutedRoots(primaryFileRoots, fileRoots)
	if err != nil {
		log.Fatalf("blob storage init failed: %v", err)
	}
	// Owner retention for managed job outputs (storage entry job_outputs,
	// pruner "custom"): storage-manager defers this entry to the owner and
	// reads the enforcement receipt the loop writes. The budget comes from
	// .vrooli/service.json, discovered from the working directory.
	if err := internalcleanup.StartOutputRetention(jobCtx, internalcleanup.RetentionConfig{
		Jobs:         jobManager,
		Store:        blobStore,
		StartupDelay: 2 * time.Minute,
	}); err != nil {
		log.Printf("WARNING: output retention disabled: %v", err)
	}

	// Model weights live alongside the blob root (outside the repo): the blob
	// store roots at <dataDir>/blobs, so the models root is <dataDir>/models.
	// Model download-on-first-use (IMG-P0-007, Phase 4) populates it; until a
	// model is installed, modelInstalled returns false and AI ops refuse with an
	// actionable hint rather than launching a doomed job.
	modelsRoot := filepath.Dir(blobStore.Root())

	// Materialize the embedded Python sidecar (image_tools_sidecar) under the
	// cache dir and put it on PYTHONPATH, so the onnxruntime backend's
	// `python3 -m image_tools_sidecar.<op>` invocations resolve regardless of the
	// process working directory. The runtime packages it imports (onnxruntime,
	// Pillow, numpy) are host provisioning — see docs/reference/backends.md.
	runtimeRoot := primaryFileRoots.CacheDir
	var sidecarPath string
	if p, scErr := sidecar.EnsureOnPath(filepath.Join(runtimeRoot, "sidecar")); scErr != nil {
		log.Printf("WARNING: sidecar materialize failed (%v); onnxruntime AI ops will be unavailable", scErr)
	} else {
		sidecarPath = p
		log.Printf("sidecar ready on PYTHONPATH: %s", sidecarPath)
	}

	// Runtime paths are stable before installation, so providers become usable
	// immediately when provisioning finishes without an API restart.
	backendEnsurer := internalhosttool.NewEnsurer()
	venvDir := filepath.Join(runtimeRoot, "pyenv")
	pythonInterpreter := pyenv.InterpreterPath(venvDir)
	runtimeSerial := make(chan struct{}, 1)
	ensureRuntime := func(ctx context.Context, emit func(int, string)) error {
		select {
		case runtimeSerial <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}
		defer func() { <-runtimeSerial }()
		if _, err := exec.LookPath("uv"); err != nil {
			status, err := backendEnsurer.Ensure(ctx, "uv")
			if err != nil {
				return err
			}
			if !status.GetOk() {
				return fmt.Errorf("provision uv: %s", strings.Join(status.GetNotes(), "; "))
			}
		}
		lockPath, err := pydeps.Materialize(runtimeRoot)
		if err != nil {
			return err
		}
		emit(0, "preparing locked Python runtime")
		_, err = pyenv.Ensure(ctx, pyenv.Spec{VenvDir: venvDir, LockFile: lockPath}, nil)
		return err
	}
	if _, err := exec.LookPath("uv"); err == nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			if err := ensureRuntime(ctx, func(int, string) {}); err != nil {
				log.Printf("Python runtime needs preparation: %v; run image-tools provision --wait", err)
			}
		}()
	}

	// Model management (IMG-P0-007): the Installer owns checksummed downloads,
	// disk-space checks, removal, and custom/local entries. Install state is
	// DB-backed (model_install) so modelInstalled reflects a completed download,
	// not just a stray directory. The Smoke gate runs an install-time load-smoke
	// (fetch → env → smoke) through the venv interpreter so an installed-but-broken
	// model fails the install instead of surfacing as a late job crash.
	modelUseGate := internalmodels.NewModelUseGate(modelsRoot)
	imageCapacity := &internalai.CLICapacityBroker{}
	installer := &internalmodels.Installer{
		Root:          modelsRoot,
		Reg:           registry,
		Custom:        internalmodels.NewCustomStore(db),
		State:         internalmodels.NewInstallStore(db),
		UseGate:       modelUseGate,
		Download:      fetch.HTTPDownloader{},
		RepoFetch:     &fetch.HFSnapshotFetcher{Python: pythonInterpreter},
		DiskAvail:     fetch.DefaultDiskAvail,
		DiskAdmission: imageCapacity,
		Smoke: &internalmodels.SmokeConfig{
			Python:     pythonInterpreter,
			PythonPath: sidecarPath,
			LockHash:   pydeps.LockHash(),
			Acquire:    sharedcapacity.AcquireScoped,
		},
		Build: &internalmodels.BuildConfig{
			Python:        pythonInterpreter,
			PythonPath:    sidecarPath,
			ReferenceONNX: filepath.Join(modelsRoot, "models", "big-lama", "lama_fp32.onnx"),
			Operations:    internalmodels.NewPlatformBuildOperationOwner(),
			Capacity: func(ctx context.Context) (internalmodels.BuildCapacityBroker, error) {
				return sharedcapacity.NewBroker(ctx, "")
			},
		},
	}
	modelInstalled := func(id string) bool {
		return installer.Installed(context.Background(), id)
	}
	modelRuntimeReady := hostModelRuntimeReadiness(installer)

	// The model-download runner: a durable CPU-lane job that fetches + verifies a
	// model's weights, emitting progress. ModelsService.InstallModel submits it.
	dispatcher.Register(internalmodels.InstallJobOperation, func(jobCtx context.Context, job internaljobs.Job, emit func(progress int, message string)) (internaljobs.Result, error) {
		var p internalmodels.InstallPayload
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return internaljobs.Result{}, fmt.Errorf("decode install payload: %w", err)
		}
		rec, err := installer.InstallForJob(jobCtx, p.ModelID, job.ID, emit)
		if err != nil {
			return internaljobs.Result{}, err
		}
		return internaljobs.Result{Ref: rec.Path}, nil
	})

	// Adapter management mirrors model management: the Installer owns checksummed
	// downloads, disk-space checks, removal, and custom/imported entries, sharing
	// the same fetch spine and data root (<dataDir>/adapters/<id>). Install state
	// is DB-backed (adapter_install).
	adapterUseGate := &sharedcapacity.UseGate{Root: filepath.Join(modelsRoot, "adapters")}
	adapterInstaller := &internaladapters.Installer{
		UseGate:       adapterUseGate,
		Root:          modelsRoot,
		Reg:           adapterRegistry,
		Custom:        internaladapters.NewCustomStore(db),
		State:         internaladapters.NewInstallStore(db),
		Download:      fetch.HTTPDownloader{},
		RepoFetch:     &fetch.HFSnapshotFetcher{Python: pythonInterpreter},
		DiskAvail:     fetch.DefaultDiskAvail,
		DiskAdmission: imageCapacity,
	}

	// Conditioning seams the AI engine resolver uses to validate an adapter stack
	// against the chosen model (decision C2/C5): a merged (seed + custom) catalog
	// lookup, the enabled-state overlay, and the install check. The same adapter
	// root the installer uses (<dataDir>/adapters/<id>) is threaded so resolved
	// adapters carry their on-disk dir.
	adapterStore := internaladapters.NewStore(db)
	adapterByID := func(id string) (internaladapters.Adapter, bool) {
		a, err := adapterInstaller.Resolve(context.Background(), id)
		return a, err == nil
	}
	adapterEnabled := func(ctx context.Context) (func(id string) bool, error) {
		overlay, lErr := adapterStore.LoadOverlay(ctx)
		if lErr != nil {
			return nil, lErr
		}
		return func(id string) bool {
			if a, ok := adapterByID(id); ok {
				return internaladapters.EffectiveEnabled(a, overlay)
			}
			return false
		}, nil
	}
	adapterInstalled := func(id string) bool {
		return adapterInstaller.Installed(context.Background(), id)
	}

	// The adapter-download runner: a durable CPU-lane job that fetches + verifies
	// an adapter's weights, emitting progress. AdaptersService.InstallAdapter
	// submits it.
	dispatcher.Register(internaladapters.InstallJobOperation, func(jobCtx context.Context, job internaljobs.Job, emit func(progress int, message string)) (internaljobs.Result, error) {
		var p internaladapters.InstallPayload
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return internaljobs.Result{}, fmt.Errorf("decode adapter install payload: %w", err)
		}
		rec, err := adapterInstaller.InstallForJob(jobCtx, p.AdapterID, job.ID, emit)
		if err != nil {
			return internaljobs.Result{}, err
		}
		return internaljobs.Result{Ref: rec.Path}, nil
	})

	opsH.RegisterParallaxRunner(dispatcher, blobStore)

	// On-demand host-tool backend provisioning. The durable job shells
	// `vrooli host install <tool> --json` (the same compose-don't-reinvent path
	// capabilities uses for host inventory) so a missing generative backend can
	// be installed from the product with live progress and client-cancel safety.
	dispatcher.Register(internalhosttool.EnsureJobOperation, func(jobCtx context.Context, job internaljobs.Job, emit func(progress int, message string)) (internaljobs.Result, error) {
		var p internalhosttool.EnsurePayload
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return internaljobs.Result{}, fmt.Errorf("decode ensure payload: %w", err)
		}
		emit(5, "downloading "+p.Tool)
		st, err := backendEnsurer.Ensure(jobCtx, p.Tool)
		if err != nil {
			return internaljobs.Result{}, fmt.Errorf("ensure backend %q: %w", p.Tool, err)
		}
		if !st.GetOk() {
			return internaljobs.Result{}, fmt.Errorf("ensure backend %q: %s", p.Tool, strings.Join(st.GetNotes(), "; "))
		}
		emit(100, st.GetExecutionState())
		return internaljobs.Result{Ref: st.GetCommand()}, nil
	})

	// Backend provider registry: register the standalone AI backends and enforce
	// the headless tenet at boot (every AI op must have a non-ComfyUI provider).
	backendReg := backends.New()
	if err := internalai.RegisterProviders(backendReg, nil, nil, pythonInterpreter); err != nil {
		log.Fatalf("ai provider registration failed: %v", err)
	}
	if err := internalanalysis.RegisterBackendProviders(backendReg); err != nil {
		log.Fatalf("analysis provider registration failed: %v", err)
	}
	if err := backendReg.Validate(); err != nil {
		log.Fatalf("backend invariant failed: %v", err)
	}

	installer.BackendReadiness = func(ctx context.Context, model internalmodels.Model) error {
		for _, op := range model.Operations {
			available := false
			for _, provider := range backendReg.Providers(op) {
				if provider.Name() == model.Backend && provider.Available(ctx) {
					available = true
					break
				}
			}
			if !available {
				return fmt.Errorf("backend %s cannot currently serve %s", model.Backend, op)
			}
		}
		return nil
	}
	provisioner := &provision.Service{
		Installer: installer, EnsureRuntime: ensureRuntime,
		RuntimeNeeded: func(model internalmodels.Model) bool {
			tool, _ := internalai.HostToolForProvider(model.Backend)
			return model.Source.HasRepo() || tool == "uv" || model.Backend == "python-sidecar" || model.Backend == "sam2-transformers"
		},
		EnsureBackend: func(ctx context.Context, model internalmodels.Model) error {
			tool, ok := internalai.HostToolForProvider(model.Backend)
			if !ok || tool == "uv" {
				return nil
			}
			status, err := backendEnsurer.Ensure(ctx, tool)
			if err != nil {
				return err
			}
			if !status.GetOk() {
				return fmt.Errorf("provision %s: %s", tool, strings.Join(status.GetNotes(), "; "))
			}
			return nil
		},
	}
	dispatcher.Register(provision.Operation, provisioner.Run)

	// Model enabled-state overlay (SQLite over the seed defaults), re-read per
	// selection so runtime enable/disable changes take effect immediately.
	modelStore := internalmodels.NewStore(db)
	enabled := modelStore.LoadOverlay

	// Per-operation default-model pins (settings surface): selection applies a
	// pin when a request gives no explicit override.
	opDefaults := internalmodels.NewOpDefaultStore(db)

	// Analysis service (OCR / NSFW / probe). Its NSFW path also backs the AI
	// generation auto-scan hook.
	analysisService, err := internalanalysis.NewService(internalanalysis.Config{
		AcquireCapacity: sharedcapacity.AcquireScoped,
		ModelReady:      installer.RuntimeReadiness,
		ModelRAMBytes: func(id string) (int64, error) {
			model, ok := registry.ByID(id)
			if !ok || model.Hardware.MinRAMGB <= 0 {
				return 0, fmt.Errorf("analysis model %s has no RAM budget", id)
			}
			return int64(model.Hardware.MinRAMGB) << 30, nil
		},
		ModelInstalled:  modelInstalled,
		AcquireModelUse: modelUseGate.Acquire,
		ModelsRoot:      modelsRoot,
		PythonProgram:   pythonInterpreter,
		Logger:          log.Default(),
	})
	if err != nil {
		log.Fatalf("analysis service init failed: %v", err)
	}

	for _, op := range internalanalysis.List() {
		dispatcher.RegisterAdmission(op.Name, analysisService.Admit)
	}
	qualityRunner := &internalquality.Runner{Evaluator: &internalquality.Evaluator{Judge: internalquality.NewGatewayJudge()}, Store: blobStore}
	dispatcher.Register(internalquality.Operation, qualityRunner.Run)
	dispatcher.RegisterAdmission(internalquality.Operation, internalquality.Admit)

	// AI engine: probe → hardware-fit model select → backend select → execute →
	// persist, with the optional NSFW auto-scan on generated output.
	aiEngine, err := internalai.NewEngine(internalai.Deps{
		Registry:              registry,
		Backends:              backendReg,
		Probe:                 probe,
		Store:                 blobStore,
		Enabled:               enabled,
		DefaultOverride:       opDefaults.Get,
		ModelInstalled:        modelInstalled,
		ModelRuntimeReadiness: modelRuntimeReady,
		AcquireModelUse:       modelUseGate.Acquire,
		CustomModels:          installer.Custom.List,
		ModelsRoot:            modelsRoot,
		AdapterByID:           adapterByID,
		AdapterEnabled:        adapterEnabled,
		AdapterInstalled:      adapterInstalled,
		AcquireAdapterUse:     adapterUseGate.Acquire,
		PosePreprocessor:      analysisService.PoseMap,
		AdaptersRoot:          modelsRoot,
		AutoScan:              analysisService.ScanNSFW,
		Capacity:              imageCapacity,
		Disk:                  imageCapacity,
		Memory:                internalai.AcquireLocalRAM,
		CPU:                   internalai.AcquireLocalCPU,
		Logger:                log.Default(),
	})
	if err != nil {
		log.Fatalf("ai engine init failed: %v", err)
	}
	// Register the AI op runners on the dispatcher (the jobs Manager fans its
	// single Runner out to these per-operation handlers).
	for op, run := range aiEngine.BuildRunners() {
		dispatcher.Register(op, run)
		dispatcher.RegisterAdmission(op, aiEngine.Admit)
	}

	// Responsible-Use deployment gate (IMG-P1-015): the tier is chosen at deploy
	// time (IMAGE_TOOLS_DEPLOYMENT_TIER), defaulting to local/unrestricted. The
	// gate the AI submit edge enforces bundles the resolved policy, the consent
	// audit log, and the public-tier abuse throttle.
	deployTier := safety.ResolveTier()
	safetyGate := safety.NewGate(deployTier, safety.NewConsentLog(db.Primary()))

	// Non-critical health summary: an enabled model whose install-time load-smoke
	// FAILED (installed but not runnable) degrades /health so the standard probe
	// surfaces it; the transient env-not-provisioned state (venv still building)
	// is intentionally not a degrade.
	healthModelRuntime := func(ctx context.Context) error {
		catalog := registry.Models()
		enabled := make([]internalmodels.Model, 0, len(catalog))
		for _, model := range catalog {
			if internalmodels.EffectiveEnabled(model, nil) {
				enabled = append(enabled, model)
			}
		}
		runtime, err := installer.DoctorRuntimeReport(ctx, enabled, nil)
		if err != nil {
			return fmt.Errorf("enabled model lifecycle unavailable: %w", err)
		}
		var failed []string
		for _, f := range runtime.Findings {
			if f.Code == "enabled_model_smoke_failed" {
				failed = append(failed, f.ModelID)
			}
		}
		if len(failed) > 0 {
			return fmt.Errorf("%d enabled model(s) failed load-smoke (installed but not runnable): %s",
				len(failed), strings.Join(failed, ", "))
		}
		return nil
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "image-tools-api", "1.0.0", healthModelRuntime),
		adaptersH.Module(db, adapterRegistry, registry, adapterInstaller, jobManager, func(sizeMBApprox int) int {
			return internaladapters.EstimateInstallSeconds(sizeMBApprox)
		}, log.Default()),
		aiH.Module(aiEngine, registry, blobStore, jobManager, safetyGate, log.Default()),
		analysisH.Module(analysisService, blobStore, jobManager, log.Default()),
		internalcleanup.Module(internalcleanup.Deps{Jobs: jobManager, Store: blobStore}),
		diffH.Module(blobStore, jobManager, log.Default()),
		jobsH.Module(jobManager, log.Default()),
		looksH.Module(db, blobStore, log.Default()),
		measuresH.Module(measureRec),
		preparationH.Module(db, blobStore),
		qualityH.Module(blobStore, jobManager),
		modelsH.Module(db, registry, probe, installer, backendReg, jobManager, func(sizeMBApprox int) int {
			return internalmodels.EstimateInstallSecondsAt(sizeMBApprox, runtimeCfg.InstallMBPerSecond)
		}, backendEnsurer, modelsH.AdapterSeams{
			ByID:      adapterByID,
			Enabled:   adapterEnabled,
			Installed: adapterInstalled,
		}, log.Default()),
		opsH.Module(blobStore, jobManager, imageCapacity, log.Default()),
		safetyH.Module(deployTier),
		selectionH.Module(blobStore, jobManager, log.Default()),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool and file roots without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error {
			// Stop the job manager (waits for in-flight jobs to observe
			// cancellation) and release its server-lifetime context before
			// closing the database.
			jobManager.Close()
			cancelJobs()
			if err := backendReg.Close(); err != nil {
				log.Printf("backend cleanup: %v", err)
			}
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// configureTestPoolSchemas applies the same domain-owned schema providers to
// each leased database before routed requests can reach it.
func configureTestPoolSchemas(db *database.RoutedDB) {
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return database.EnsureSchemas(ctx, pool, modules.AllSchemas()...)
	})
}

// hostModelRuntimeReadiness reads lifecycle evidence from the host catalog.
// The model weights, their smoke verdicts and runtime readiness are shared
// host artifacts; a routed test database must isolate user/job rows without
// making installed host models appear absent during a test-mode request.
func hostModelRuntimeReadiness(installer *internalmodels.Installer) func(context.Context, string) (bool, string) {
	return func(_ context.Context, id string) (bool, string) {
		return installer.RuntimeReadiness(context.Background(), id)
	}
}
