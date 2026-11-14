package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/catalogstate"
	"image-tools/internal/fetch"
)

// =============================================================================
// Model management (IMG-P0-007): opt-in checksummed downloads, disk-space
// awareness, removal, and custom/local entries. The seed catalog stays the
// read-only baseline; install + custom state live in SQLite (model_install,
// custom_model) and on disk under <Root>/models/<id>.
// =============================================================================

// Install-engine errors. Callers (handlers/CLI) map these to actionable codes.
var (
	// ErrInsufficientDisk is returned when the model's approximate size (plus a
	// safety margin) exceeds the free space at the models root.
	ErrInsufficientDisk = fetch.ErrInsufficientDisk
	// ErrChecksumMismatch is returned when a downloaded artifact's hash does not
	// match the pinned/seed checksum. The partial download is removed.
	ErrChecksumMismatch = fetch.ErrChecksumMismatch
	// ErrNoDownloadSource is returned when a model has neither a download URL nor
	// a local path to install from.
	ErrNoDownloadSource = fetch.ErrNoDownloadSource
	// ErrModelNotFound is returned when an id is in neither the seed nor custom.
	ErrModelNotFound = errors.New("models: model not found")
	// ErrCustomShadowsSeed is returned when a custom entry reuses a seed id.
	ErrCustomShadowsSeed = errors.New("models: custom model id collides with a seed model")
	// ErrLocalPathMissing is returned when a custom local model points at a path
	// that does not exist.
	ErrLocalPathMissing = fetch.ErrLocalPathMissing
	// ErrManualInstall is returned when a weight-backed model is honestly marked
	// source.manual: it has no auto-fetchable artifact and must be installed
	// manually per its docs_url (the installer never GETs a documentation page).
	ErrManualInstall = errors.New("models: model requires manual installation")
)

// InstallJobOperation is the job.Operation name a model download runs under. The
// models handler submits a job with this op + an InstallPayload; main.go registers
// the matching runner on the jobrunner dispatcher.
const InstallJobOperation = "model_install"

// InstallPayload is the JSON body of an install job.
type InstallPayload struct {
	ModelID string `json:"model_id"`
}

// DefaultInstallMBPerSecond is the conservative throughput used for install
// ETAs when the operator has not configured a host-specific value.
const DefaultInstallMBPerSecond = 15

// EstimateInstallSeconds is a rough ETA for downloading a model of the given
// approximate size, assuming a conservative throughput. Surfaced as the job ETA.
func EstimateInstallSeconds(sizeMBApprox int) int {
	return EstimateInstallSecondsAt(sizeMBApprox, DefaultInstallMBPerSecond)
}

// EstimateInstallSecondsAt is EstimateInstallSeconds with an explicit measured
// throughput. Non-positive throughput falls back to the default.
func EstimateInstallSecondsAt(sizeMBApprox, mbPerSecond int) int {
	if mbPerSecond <= 0 {
		mbPerSecond = DefaultInstallMBPerSecond
	}
	if sizeMBApprox <= 0 {
		return 0
	}
	if eta := sizeMBApprox / mbPerSecond; eta > 0 {
		return eta
	}
	return 1
}

// The install I/O seams the Installer depends on (Downloader, RepoFetcher,
// DiskSpaceFunc) live in the shared internal/fetch spine so the model and adapter
// catalogs share one audited install path. They are re-exported here under the
// Installer's field types; the wiring (main.go) supplies the concrete fetch
// implementations (fetch.HTTPDownloader / fetch.HFSnapshotFetcher /
// fetch.DefaultDiskAvail).
type (
	// Downloader fetches a remote artifact to a local path. See fetch.Downloader.
	Downloader = fetch.Downloader
	// RepoFetcher fetches a whole multi-file repo snapshot. See fetch.RepoFetcher.
	RepoFetcher = fetch.RepoFetcher
	// DiskSpaceFunc reports free bytes at a path's filesystem. See fetch.
	DiskSpaceFunc = fetch.DiskSpaceFunc
)

type InstallRecord = catalogstate.InstallRecord

type InstallStore struct {
	*catalogstate.Installs
	db SQLExecutor
}

func NewInstallStore(db SQLExecutor) *InstallStore {
	return &InstallStore{Installs: catalogstate.NewInstalls(db, catalogstate.Models), db: db}
}

type CustomStore = catalogstate.Custom[Model]

func NewCustomStore(db SQLExecutor) *CustomStore {
	return catalogstate.NewCustom(db, catalogstate.Models, func(value Model) string { return value.ID })
}

// Installer downloads, verifies, removes, and registers model weights. It is the
// runtime side of the model registry: the seed/custom entries describe a model,
// the Installer makes it present on disk and records that fact.
type Installer struct {
	// Root is the absolute directory model weights live under (per model at
	// <Root>/models/<id>) — the same root the AI engine reads (main.go).
	Root string
	// Reg is the read-only seed catalog.
	Reg *Registry
	// Custom resolves operator-registered local entries (may be nil).
	Custom *CustomStore
	// State persists install state.
	State *InstallStore
	// Download fetches remote artifacts (HTTPDownloader in production).
	Download Downloader
	// RepoFetch fetches multi-file repo snapshots (HFSnapshotFetcher in
	// production). nil disables repo-source installs (they fail with a clear error).
	RepoFetch RepoFetcher
	// DiskAvail reports free space at Root (diskAvail in production).
	DiskAvail DiskSpaceFunc
	// DiskAdmission is the shared host owner used for full model job peaks.
	// A nil owner refuses weight acquisition; snapshot checks are not admission.
	DiskAdmission sharedcapacity.DiskJobAdmitter
	// Now sources timestamps (defaults to time.Now).
	Now func() time.Time
	// Smoke, when set, runs an install-time load-smoke after a successful fetch
	// (fetch → env → smoke), so an installed-but-broken model fails the install
	// instead of surfacing as a late job crash. nil disables the gate.
	Smoke *SmokeConfig
	// BackendReadiness verifies this model's declared operation providers. The
	// process adapter owns its probes; the lifecycle owns their readiness result.
	BackendReadiness func(context.Context, Model) error
	// Build, when set, executes only the closed built-in exporter selected by a
	// validated seed BuildProfile. Export assets are already installed and hash
	// checked before this hook can run.
	Build *BuildConfig
	// UseGate is shared with inference services so model files cannot be
	// installed or removed while an operation is using them.
	UseGate *ModelUseGate
	// removeAll is the filesystem seam for retiring model directories. Tests use
	// it to represent interruption after the directory has been atomically moved
	// out of the path readers resolve.
	removeAll       func(string) error
	gateMu          sync.Mutex
	activeLifecycle sync.Map // model ID -> current owned preparation stage
}

func (in *Installer) useGate() *ModelUseGate {
	in.gateMu.Lock()
	defer in.gateMu.Unlock()
	if in.UseGate == nil {
		in.UseGate = NewModelUseGate(in.Root)
	}
	return in.UseGate
}

func (in *Installer) now() time.Time {
	if in.Now != nil {
		return in.Now()
	}
	return time.Now()
}

// modelDir is the absolute directory model id's weights live under.
func (in *Installer) modelDir(id string) string {
	return filepath.Join(in.Root, "models", id)
}

func (in *Installer) removalDir(id string) string {
	sum := sha256.Sum256([]byte(id))
	return filepath.Join(in.Root, ".image-tools-model-removals", hex.EncodeToString(sum[:]))
}

func (in *Installer) removePath(path string) error {
	if in.removeAll != nil {
		return in.removeAll(path)
	}
	return os.RemoveAll(path)
}

// Resolve returns the model entry for id from the seed or, failing that, the
// custom store.
func (in *Installer) Resolve(ctx context.Context, id string) (Model, error) {
	if m, ok := in.Reg.ByID(id); ok {
		return m, nil
	}
	if in.Custom != nil {
		customs, err := in.Custom.List(ctx)
		if err != nil {
			return Model{}, err
		}
		for _, m := range customs {
			if m.ID == id {
				return m, nil
			}
		}
	}
	return Model{}, fmt.Errorf("%w: %q", ErrModelNotFound, id)
}

// Installed reports whether a model's weights are present (a completed install
// record AND the directory exists on disk). A builtin (weightless) model has
// nothing to download and is always installed.
func (in *Installer) Installed(ctx context.Context, id string) bool {
	_, ok := in.installedRecord(ctx, id)
	return ok
}

// installedRecord resolves the durable install claim once for lifecycle and
// readiness callers. A claim is valid only while its managed artifact remains
// present; weightless builtins do not need a filesystem record.
func (in *Installer) installedRecord(ctx context.Context, id string) (InstallRecord, bool) {
	if in.Reg != nil {
		if m, ok := in.Reg.ByID(id); ok && !m.RequiresWeights() {
			return InstallRecord{ID: id, Installed: true, Path: in.modelDir(id)}, true
		}
	}
	if in.State == nil {
		path := in.modelDir(id)
		return InstallRecord{ID: id, Installed: true, Path: path}, dirExists(path)
	}
	rec, ok, err := in.State.Get(ctx, id)
	if err != nil || !ok || !rec.Installed {
		return InstallRecord{}, false
	}
	return rec, pathExists(rec.Path)
}

// Install downloads (or, for a custom local model, verifies) the weights for id,
// checks free disk space first, verifies/pins the checksum, and records the
// install. It is idempotent: an already-installed model returns its record
// without re-downloading. emit reports human-readable progress.
func (in *Installer) Install(ctx context.Context, id string, emit func(progress int, message string)) (InstallRecord, error) {
	return in.install(ctx, id, "", emit)
}

// InstallForJob is the durable-job entry point. jobID is the persisted model
// installation job identity used to bind a native build operation before it is
// dispatched.
func (in *Installer) InstallForJob(ctx context.Context, id, jobID string, emit func(progress int, message string)) (InstallRecord, error) {
	if strings.TrimSpace(jobID) == "" {
		return InstallRecord{}, errors.New("models: install job identity is required")
	}
	return in.install(ctx, id, jobID, emit)
}

func (in *Installer) install(ctx context.Context, id, jobID string, emit func(int, string)) (rec InstallRecord, installErr error) {
	release, err := in.useGate().AcquireExclusive(id)
	if err != nil {
		return rec, err
	}
	defer release()
	if emit == nil {
		emit = func(int, string) {}
	}
	m, err := in.Resolve(ctx, id)
	if err != nil {
		return rec, err
	}
	if m.CapabilityLabels.CommercialUse != CommercialUseYes {
		return rec, fmt.Errorf("models: %q cannot be installed until its commercial rights are verified", id)
	}
	if jobID == "" {
		jobID = fmt.Sprintf("%s-%d", id, in.now().UnixNano())
	}
	var diskLease sharedcapacity.DiskJobLease
	var diskGuard *fetch.DiskLeaseGuard
	repoFetch := in.RepoFetch
	var diskProgress func(context.Context, string, int64, int64) error
	if m.RequiresWeights() && m.Source.LocalPath == "" {
		if in.DiskAdmission == nil {
			return rec, fmt.Errorf("models: shared host disk admission is unavailable; refusing acquisition")
		}
		buildExpansion := 1
		if m.BuildProfile != nil {
			buildExpansion = 2
		}
		request, planErr := fetch.PlanInstallDiskJob(ctx, "image-tools:model-install", jobID, in.Root, m.SizeMBApprox, m.Source, buildExpansion)
		if planErr != nil {
			return rec, fmt.Errorf("models: disk peak planning failed before acquisition: %w", planErr)
		}
		repoFetch = fetch.PinHuggingFaceCache(repoFetch, fetch.HuggingFaceCachePath(request))
		diskLease, err = in.DiskAdmission.ReserveDiskJob(ctx, request)
		if err != nil {
			return rec, fmt.Errorf("models: reserve complete install disk peak: %w", err)
		}
		if diskLease == nil {
			return rec, fmt.Errorf("models: shared host disk admission returned no lease")
		}
		growthGuard, growthErr := fetch.NewDiskGrowthGuard(diskLease, request, buildExpansion, fetch.InstallOwnerCeilingBytes)
		if growthErr != nil {
			return rec, fmt.Errorf("models: create bounded disk growth observer: %w", growthErr)
		}
		diskProgress = fetch.DiskGrowthObserver(growthGuard)
		ctx, diskGuard = fetch.MaintainDiskLease(ctx, diskLease, 45*time.Second)
		defer func() {
			heartbeatErr := diskGuard.Stop()
			if heartbeatErr != nil {
				installErr = errors.Join(installErr, heartbeatErr)
				return // preserve the lease for owner-verified reconciliation
			}
			if errors.Is(installErr, fetch.ErrCleanupUnverified) {
				installErr = errors.Join(installErr, fmt.Errorf("models: disk lease %s remains active because acquisition cleanup is unverified", diskLease.Reservation().ID))
				return
			}
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			receipt, completeErr := diskLease.Complete(cleanup)
			if completeErr != nil {
				installErr = errors.Join(installErr, fmt.Errorf("models: complete disk lease %s after install stopped: %w", receipt.ID, completeErr))
			}
		}()
	}
	current, err := in.lifecycleForModel(ctx, m)
	if err != nil {
		return rec, err
	}
	if err := in.advance(ctx, &current, ModelInstalling, "installing"); err != nil {
		return rec, err
	}
	defer func() {
		defer in.activeLifecycle.Delete(id)
		if installErr == nil {
			return
		}
		current.Detail = installErr.Error()
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		installErr = errors.Join(installErr, in.advance(cleanup, &current, ModelFailed, "install_failed"))
	}()

	rec, err = in.acquireArtifacts(ctx, m, repoFetch, diskProgress, emit)
	if err != nil {
		return rec, err
	}
	if err := in.advance(ctx, &current, ModelInstalled, "installed"); err != nil {
		return rec, err
	}
	modelHash := rec.Checksum
	if m.BuildProfile != nil {
		if err := in.advance(ctx, &current, ModelBuilding, "build_not_ready"); err != nil {
			return rec, err
		}
		modelHash, err = in.ensureBuild(ctx, m, rec.Path, jobID, emit)
		if err != nil {
			return rec, err
		}
	}
	if in.BackendReadiness != nil {
		if err := in.BackendReadiness(ctx, m); err != nil {
			current.Detail = err.Error()
			if err := in.advance(ctx, &current, ModelInstalled, "needs_backend"); err != nil {
				return rec, err
			}
			return rec, nil
		}
	}
	if _, _, applicable := probeArgsFor(m, rec.Path, false); applicable && in.Smoke != nil && fileExists(in.Smoke.Python) {
		if err := in.advance(ctx, &current, ModelSmoking, "smoke_pending"); err != nil {
			return rec, err
		}
	}
	if err := in.ensureSmoke(ctx, m, rec.Path, modelHash, emit); err != nil {
		return rec, err
	}
	final := in.observeLifecycle(ctx, m, current, &rec)
	current.ArtifactHash, current.RuntimeHash = final.ArtifactHash, final.RuntimeHash
	current.Detail = final.Detail
	if err := in.advance(ctx, &current, final.State, final.Reason); err != nil {
		return rec, err
	}
	emit(100, string(final.State))
	return rec, nil
}

// acquireArtifacts has one publication boundary for downloaded, repo and local
// artifacts. Every path then passes the same build/smoke/lifecycle completion.
func (in *Installer) acquireArtifacts(ctx context.Context, m Model, repoFetch fetch.RepoFetcher, diskProgress func(context.Context, string, int64, int64) error, emit func(int, string)) (InstallRecord, error) {
	id := m.ID
	acquirer := fetch.Acquirer{Root: in.Root, Download: in.Download, RepoFetch: repoFetch, DiskAvail: in.DiskAvail, DiskProgressObserver: diskProgress}
	if !m.RequiresWeights() {
		rec, _ := in.installedRecord(ctx, id)
		return rec, nil
	}
	var previous InstallRecord
	var hasPrevious bool
	if in.State != nil {
		var err error
		previous, hasPrevious, err = in.State.Get(ctx, id)
		if err != nil {
			return InstallRecord{}, err
		}
	} else {
		previous, hasPrevious = in.installedRecord(ctx, id)
	}
	if hasPrevious && previous.Installed && pathExists(previous.Path) {
		// Installed is a durable claim, not current proof: source manifests can
		// change and files can be truncated or replaced after the first install.
		// Revalidate before taking the idempotent path so bad old artifacts are
		// reacquired through the same staged, atomic publication boundary.
		if acquirer.ValidateInstalled(m.Source, previous.Path, previous.Checksum) == nil && (m.BuildProfile == nil || buildInputsReady(m.BuildProfile, previous.Path)) {
			return previous, nil
		}
	}
	if m.RequiresWeights() && m.Source.Manual && m.Source.LocalPath == "" {
		return InstallRecord{}, fmt.Errorf("%w: %q must be installed manually per its docs_url", ErrManualInstall, id)
	}
	pinnedChecksum := ""
	if hasPrevious {
		pinnedChecksum = previous.Checksum
	}
	acquired, err := acquirer.Acquire(ctx, m.Source, in.modelDir(id), pinnedChecksum, m.SizeMBApprox, emit)
	if err != nil {
		return InstallRecord{}, fmt.Errorf("models: acquire %q: %w", id, err)
	}
	rec := InstallRecord{ID: id, Installed: true, Path: acquired.Path, Checksum: acquired.Checksum, SizeBytes: acquired.SizeBytes, InstalledAt: in.now()}

	if in.State != nil {
		if err := in.State.Set(ctx, rec); err != nil {
			return rec, err
		}
	}
	return rec, nil
}

// Remove deletes a model's downloaded weights and clears its install record. A
// custom local model is unlinked (its referenced path is never deleted).
func (in *Installer) Remove(ctx context.Context, id string) error {
	release, err := in.useGate().AcquireExclusive(id)
	if err != nil {
		return err
	}
	defer release()
	m, err := in.Resolve(ctx, id)
	if err != nil {
		return err
	}
	if m.Source.LocalPath == "" {
		modelDir := in.modelDir(id)
		// Rename the complete tree out of the serving path before recursive
		// deletion. Rename is atomic within this parent directory, so a crash
		// leaves either the intact installed tree or no tree at the path checked
		// by Installed; a partially cleaned quarantine can never be served.
		quarantineDir := in.removalDir(id)
		if err := os.MkdirAll(filepath.Dir(quarantineDir), 0o755); err != nil {
			return fmt.Errorf("models: create removal quarantine for %q: %w", id, err)
		}
		if err := in.removeExistingPath(quarantineDir); err != nil {
			return fmt.Errorf("models: clean interrupted removal %q: %w", id, err)
		}
		if err := os.Rename(modelDir, quarantineDir); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("models: stage weights removal %q: %w", id, err)
		}
		if err := in.removeExistingPath(quarantineDir); err != nil {
			return fmt.Errorf("models: remove weights %q: %w", id, err)
		}
	}
	if in.State != nil {
		return in.State.Delete(ctx, id)
	}
	return nil
}

func (in *Installer) removeExistingPath(path string) error {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return in.removePath(path)
}

// AddCustom registers a custom/local model entry. The id must not collide with a
// seed model, and a declared local path must exist.
func (in *Installer) AddCustom(ctx context.Context, m Model) error {
	if in.Custom == nil {
		return errors.New("models: custom store unavailable")
	}
	if in.Reg == nil {
		return errors.New("models: registry unavailable")
	}
	if m.ID == "" {
		return errors.New("models: custom model id is required")
	}
	if _, ok := in.Reg.ByID(m.ID); ok {
		return fmt.Errorf("%w: %q", ErrCustomShadowsSeed, m.ID)
	}
	if m.Source.LocalPath != "" && !pathExists(m.Source.LocalPath) {
		return fmt.Errorf("%w: %q", ErrLocalPathMissing, m.Source.LocalPath)
	}
	if strings.TrimSpace(m.CapabilityLabels.License) == "" {
		return errors.New("models: custom model license is required")
	}
	if m.CapabilityLabels.CommercialUse != CommercialUseYes {
		return errors.New("models: custom models must be explicitly commercial-clean")
	}
	if _, err := in.Reg.WithCustom([]Model{m}); err != nil {
		return fmt.Errorf("models: invalid custom model: %w", err)
	}
	return in.Custom.Add(ctx, m)
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
