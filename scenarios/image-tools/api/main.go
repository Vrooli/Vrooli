package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	internalai "image-tools/internal/ai"
	internalanalysis "image-tools/internal/analysis"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/clock"
	"image-tools/internal/jobrunner"
	internaljobs "image-tools/internal/jobs"
	internalmeasures "image-tools/internal/measures"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/modules"
	"image-tools/internal/safety"
	"image-tools/internal/server"
	"image-tools/internal/sidecar"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	aiH "image-tools/handlers/ai"
	analysisH "image-tools/handlers/analysis"
	diffH "image-tools/handlers/diff"
	healthH "image-tools/handlers/health"
	jobsH "image-tools/handlers/jobs"
	looksH "image-tools/handlers/looks"
	modelsH "image-tools/handlers/models"
	opsH "image-tools/handlers/ops"
	safetyH "image-tools/handlers/safety"
	selectionH "image-tools/handlers/selection"

	internalstorage "image-tools/internal/storage"
)

const (
	cpuWorkersEnv         = "IMAGE_TOOLS_CPU_WORKERS"
	installMBPerSecondEnv = "IMAGE_TOOLS_INSTALL_MB_PER_SECOND"
	defaultCPUWorkers     = 4
	minCPUWorkers         = 1
	maxCPUWorkers         = 32
	minInstallMBPerSecond = 1
	maxInstallMBPerSecond = 10000
)

type runtimeConfig struct {
	CPUWorkers         int
	InstallMBPerSecond int
}

func loadRuntimeConfig() (runtimeConfig, error) {
	cpuWorkers, err := intFromEnv(cpuWorkersEnv, defaultCPUWorkers, minCPUWorkers, maxCPUWorkers)
	if err != nil {
		return runtimeConfig{}, err
	}
	installMBPerSecond, err := intFromEnv(installMBPerSecondEnv, internalmodels.DefaultInstallMBPerSecond, minInstallMBPerSecond, maxInstallMBPerSecond)
	if err != nil {
		return runtimeConfig{}, err
	}
	return runtimeConfig{CPUWorkers: cpuWorkers, InstallMBPerSecond: installMBPerSecond}, nil
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

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
//
// The path scope is the variant-aware namespace (storage.ScenarioNamespace),
// not the bare slug: under a Baseline Modes shadow engagement the lifecycle
// injects VROOLI_STORAGE_NAMESPACE, so the shadow's SQLite file lands beside
// "<scenario>_shadow" and never shares live's database. Outside the lifecycle
// (local `go run`, tests) it falls back to the compile-time slug, so live paths
// are unchanged. This is why a generated scenario is shadow-safe with zero
// per-scenario work — see packages/api-core/storage/namespace.go.
//
// The pragmas mirror agent-inbox; tweak in lockstep with
// internal/testutil/db.NewSQLite so production and tests open files the
// same way.
func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("image-tools")
	if err != nil {
		return "", fmt.Errorf("resolve image-tools storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"image-tools.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve image-tools db path: %w", err)
	}
	return sqliteFileDSN(path)
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
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

	dsn, err := sqliteDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

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
	dispatcher := jobrunner.New()

	// Runtime measures (IMG-P0-012): the recorder captures op latency + queue-wait
	// + outcome for every finalized job via the Manager's OnComplete hook. Recorded
	// under the server-lifetime context so it is independent of any caller.
	measureRec := internalmeasures.NewRecorder(db)
	jobManager := internaljobs.New(db, internaljobs.Config{
		Runner:     dispatcher.Run,
		Clock:      clock.System{},
		CPUWorkers: runtimeCfg.CPUWorkers,
		OnComplete: func(job internaljobs.Job) {
			sample, trace := jobSampleAndTrace(job)
			if err := measureRec.Record(jobCtx, sample); err != nil {
				log.Printf("measures: observe job %s: %v", job.ID, err)
			}
			if err := measureRec.RecordJobTrace(jobCtx, trace); err != nil {
				log.Printf("measures: trace job %s: %v", job.ID, err)
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
	blobStore, err := internalstorage.New(blobNamespace)
	if err != nil {
		log.Fatalf("blob storage init failed: %v", err)
	}

	// Model weights live alongside the blob root (outside the repo): the blob
	// store roots at <dataDir>/blobs, so the models root is <dataDir>/models.
	// Model download-on-first-use (IMG-P0-007, Phase 4) populates it; until a
	// model is installed, modelInstalled returns false and AI ops refuse with an
	// actionable hint rather than launching a doomed job.
	modelsRoot := filepath.Dir(blobStore.Root())

	// Materialize the embedded Python sidecar (image_tools_sidecar) under the
	// data dir and put it on PYTHONPATH, so the onnxruntime backend's
	// `python3 -m image_tools_sidecar.<op>` invocations resolve regardless of the
	// process working directory. The runtime packages it imports (onnxruntime,
	// Pillow, numpy) are host provisioning — see docs/reference/backends.md.
	if sidecarPath, scErr := sidecar.EnsureOnPath(filepath.Join(modelsRoot, "sidecar")); scErr != nil {
		log.Printf("WARNING: sidecar materialize failed (%v); onnxruntime AI ops will be unavailable", scErr)
	} else {
		log.Printf("sidecar ready on PYTHONPATH: %s", sidecarPath)
	}

	// Model management (IMG-P0-007): the Installer owns checksummed downloads,
	// disk-space checks, removal, and custom/local entries. Install state is
	// DB-backed (model_install) so modelInstalled reflects a completed download,
	// not just a stray directory.
	installer := &internalmodels.Installer{
		Root:      modelsRoot,
		Reg:       registry,
		Custom:    internalmodels.NewCustomStore(db),
		State:     internalmodels.NewInstallStore(db),
		Download:  internalmodels.HTTPDownloader{},
		DiskAvail: internalmodels.DefaultDiskAvail,
	}
	modelInstalled := func(id string) bool {
		return installer.Installed(context.Background(), id)
	}

	// The model-download runner: a durable CPU-lane job that fetches + verifies a
	// model's weights, emitting progress. ModelsService.InstallModel submits it.
	dispatcher.Register(internalmodels.InstallJobOperation, func(jobCtx context.Context, job internaljobs.Job, emit func(progress int, message string)) (string, error) {
		var p internalmodels.InstallPayload
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return "", fmt.Errorf("decode install payload: %w", err)
		}
		rec, err := installer.Install(jobCtx, p.ModelID, emit)
		if err != nil {
			return "", err
		}
		return rec.Path, nil
	})

	// Backend provider registry: register the standalone AI backends and enforce
	// the headless tenet at boot (every AI op must have a non-ComfyUI provider).
	backendReg := backends.New()
	if err := internalai.RegisterProviders(backendReg, nil, nil); err != nil {
		log.Fatalf("ai provider registration failed: %v", err)
	}
	if err := internalanalysis.RegisterBackendProviders(backendReg); err != nil {
		log.Fatalf("analysis provider registration failed: %v", err)
	}
	if err := backendReg.Validate(); err != nil {
		log.Fatalf("backend invariant failed: %v", err)
	}

	// Model enabled-state overlay (SQLite over the seed defaults), re-read per
	// selection so runtime enable/disable changes take effect immediately.
	modelStore := internalmodels.NewStore(db)
	enabled := func(ctx context.Context) (internalmodels.EnabledFunc, error) {
		overlay, lErr := modelStore.LoadOverlay(ctx)
		if lErr != nil {
			return nil, lErr
		}
		return registry.EnabledWithOverlay(overlay), nil
	}

	// Per-operation default-model pins (settings surface): selection applies a
	// pin when a request gives no explicit override.
	opDefaults := internalmodels.NewOpDefaultStore(db)

	// Analysis service (OCR / NSFW / probe). Its NSFW path also backs the AI
	// generation auto-scan hook.
	analysisService, err := internalanalysis.NewService(internalanalysis.Config{
		ModelInstalled: modelInstalled,
		ModelsRoot:     modelsRoot,
		Logger:         log.Default(),
	})
	if err != nil {
		log.Fatalf("analysis service init failed: %v", err)
	}

	// AI engine: probe → hardware-fit model select → backend select → execute →
	// persist, with the optional NSFW auto-scan on generated output.
	aiEngine, err := internalai.NewEngine(internalai.Deps{
		Registry:        registry,
		Backends:        backendReg,
		Probe:           probe,
		Store:           blobStore,
		Enabled:         enabled,
		DefaultOverride: opDefaults.Get,
		ModelInstalled:  modelInstalled,
		ModelsRoot:      modelsRoot,
		AutoScan:        analysisService.ScanNSFW,
		Logger:          log.Default(),
	})
	if err != nil {
		log.Fatalf("ai engine init failed: %v", err)
	}
	// Register the AI op runners on the dispatcher (the jobs Manager fans its
	// single Runner out to these per-operation handlers).
	for op, run := range aiEngine.BuildRunners() {
		dispatcher.Register(op, run)
	}

	// Responsible-Use deployment gate (IMG-P1-015): the tier is chosen at deploy
	// time (IMAGE_TOOLS_DEPLOYMENT_TIER), defaulting to local/unrestricted. The
	// gate the AI submit edge enforces bundles the resolved policy, the consent
	// audit log, and the public-tier abuse throttle.
	deployTier := safety.ResolveTier()
	safetyGate := safety.NewGate(deployTier, safety.NewConsentLog(db.Primary()))

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		healthH.Module(db, "image-tools-api", "1.0.0"),
		aiH.Module(aiEngine, registry, blobStore, jobManager, safetyGate, log.Default()),
		analysisH.Module(analysisService, jobManager, log.Default()),
		diffH.Module(blobStore, jobManager, log.Default()),
		jobsH.Module(jobManager, log.Default()),
		looksH.Module(db, blobStore, log.Default()),
		modelsH.Module(db, registry, probe, installer, backendReg, jobManager, func(sizeMBApprox int) int {
			return internalmodels.EstimateInstallSecondsAt(sizeMBApprox, runtimeCfg.InstallMBPerSecond)
		}, log.Default()),
		opsH.Module(blobStore, jobManager, log.Default()),
		safetyH.Module(deployTier),
		selectionH.Module(blobStore, jobManager, log.Default()),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)

	// No /measures substrate is mounted yet: image operations declare their
	// op-latency / throughput / queue-wait measures as they land (Phase 2+),
	// and the measures-go serve registry is wired here when the first measure
	// exists. Mounting an empty substrate now would be dead scaffolding.

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
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
