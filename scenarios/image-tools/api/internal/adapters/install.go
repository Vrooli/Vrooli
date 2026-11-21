package adapters

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/catalogstate"
	"image-tools/internal/fetch"
)

// =============================================================================
// Adapter management: opt-in checksummed downloads, removal, and custom/imported
// entries — the runtime side of the adapter catalog. It reuses the shared
// internal/fetch spine (Downloader / RepoFetcher / DiskSpaceFunc / artifact
// validation / checksum pinning) so models and adapters share one audited
// install path (decision CC1). The seed stays the read-only baseline; install +
// custom state live in SQLite (adapter_install, custom_adapter) and on disk
// under <Root>/adapters/<id>.
// =============================================================================

// Install-engine errors. Callers (handlers/CLI) map these to actionable codes.
var (
	// ErrInsufficientDisk is returned when the adapter's approximate size (plus a
	// safety margin) exceeds the free space at the adapters root.
	ErrInsufficientDisk = fetch.ErrInsufficientDisk
	// ErrChecksumMismatch is returned when a downloaded artifact's hash does not
	// match the pinned/seed checksum. The partial download is removed.
	ErrChecksumMismatch = fetch.ErrChecksumMismatch
	// ErrNoDownloadSource is returned when an adapter has no resolvable source.
	ErrNoDownloadSource = fetch.ErrNoDownloadSource
	// ErrAdapterNotFound is returned when an id is in neither the seed nor custom.
	ErrAdapterNotFound = errors.New("adapters: adapter not found")
	// ErrCustomShadowsSeed is returned when a custom entry reuses a seed id.
	ErrCustomShadowsSeed = errors.New("adapters: custom adapter id collides with a seed adapter")
	// ErrLocalPathMissing is returned when a custom local adapter points at a path
	// that does not exist.
	ErrLocalPathMissing = fetch.ErrLocalPathMissing
	// ErrRepoRevisionUnpinned is returned when a repo-source adapter has no pinned
	// commit SHA (required for a reproducible install + tree-manifest hash).
	ErrRepoRevisionUnpinned = fetch.ErrRepoRevisionUnpinned
)

// InstallJobOperation is the job.Operation name an adapter download runs under.
const InstallJobOperation = "adapter_install"

// InstallPayload is the JSON body of an adapter install job.
type InstallPayload struct {
	AdapterID string `json:"adapter_id"`
}

// DefaultInstallMBPerSecond is the conservative throughput used for ETAs.
const DefaultInstallMBPerSecond = 15

// EstimateInstallSeconds is a rough ETA for downloading an adapter of the given
// approximate size, assuming a conservative throughput.
func EstimateInstallSeconds(sizeMBApprox int) int {
	if sizeMBApprox <= 0 {
		return 0
	}
	if eta := sizeMBApprox / DefaultInstallMBPerSecond; eta > 0 {
		return eta
	}
	return 1
}

// The install I/O seams (Downloader, RepoFetcher, DiskSpaceFunc) are the shared
// fetch spine, re-exported under the Installer's field types.
type (
	// Downloader fetches a remote artifact to a local path. See fetch.Downloader.
	Downloader = fetch.Downloader
	// RepoFetcher fetches a whole multi-file repo snapshot. See fetch.RepoFetcher.
	RepoFetcher = fetch.RepoFetcher
	// DiskSpaceFunc reports free bytes at a path's filesystem. See fetch.
	DiskSpaceFunc = fetch.DiskSpaceFunc
)

type InstallRecord = catalogstate.InstallRecord

type InstallStore = catalogstate.Installs

func NewInstallStore(db SQLExecutor) *InstallStore {
	return catalogstate.NewInstalls(db, catalogstate.Adapters)
}

type CustomStore = catalogstate.Custom[Adapter]

func NewCustomStore(db SQLExecutor) *CustomStore {
	return catalogstate.NewCustom(db, catalogstate.Adapters, func(value Adapter) string { return value.ID })
}

// Installer downloads, verifies, removes, and registers adapter weights.
type Installer struct {
	UseGate *capacity.UseGate
	// Root is the absolute directory adapter weights live under (per adapter at
	// <Root>/adapters/<id>).
	Root string
	// Reg is the read-only seed catalog.
	Reg *Registry
	// Custom resolves operator-registered / imported entries (may be nil).
	Custom *CustomStore
	// State persists install state.
	State *InstallStore
	// Download fetches remote artifacts (fetch.HTTPDownloader in production).
	Download Downloader
	// RepoFetch fetches multi-file repo snapshots (fetch.HFSnapshotFetcher).
	RepoFetch RepoFetcher
	// DiskAvail reports free space at Root (fetch.DefaultDiskAvail).
	DiskAvail DiskSpaceFunc
	// DiskAdmission reserves the complete remote install across every write filesystem.
	DiskAdmission capacity.DiskJobAdmitter
	// Now sources timestamps (defaults to time.Now).
	Now func() time.Time
}

func (in *Installer) now() time.Time {
	if in.Now != nil {
		return in.Now()
	}
	return time.Now()
}

// adapterDir is the absolute directory adapter id's weights live under.
func (in *Installer) adapterDir(id string) string {
	return filepath.Join(in.Root, "adapters", id)
}

// Resolve returns the adapter entry for id from the seed or the custom store.
func (in *Installer) Resolve(ctx context.Context, id string) (Adapter, error) {
	if in.Reg != nil {
		if a, ok := in.Reg.ByID(id); ok {
			return a, nil
		}
	}
	if in.Custom != nil {
		customs, err := in.Custom.List(ctx)
		if err != nil {
			return Adapter{}, err
		}
		for _, a := range customs {
			if a.ID == id {
				return a, nil
			}
		}
	}
	return Adapter{}, fmt.Errorf("%w: %q", ErrAdapterNotFound, id)
}

// Installed reports whether an adapter's weights are present (a completed install
// record AND the directory exists on disk).
func (in *Installer) Installed(ctx context.Context, id string) bool {
	if in.State == nil {
		return dirExists(in.adapterDir(id))
	}
	rec, ok, err := in.State.Get(ctx, id)
	if err != nil || !ok || !rec.Installed {
		return false
	}
	return pathExists(rec.Path)
}

// Install downloads (or, for a local adapter, verifies) the weights for id,
// checks free disk space first, verifies/pins the checksum, and records the
// install. It is idempotent: an already-installed adapter returns its record.
func (in *Installer) useGate() *capacity.UseGate {
	if in.UseGate != nil {
		return in.UseGate
	}
	return &capacity.UseGate{Root: filepath.Join(in.Root, "adapters")}
}

func (in *Installer) Install(ctx context.Context, id string, emit func(int, string)) (InstallRecord, error) {
	return in.InstallForJob(ctx, id, "", emit)
}

// InstallForJob binds the admission receipt to the durable adapter install job.
func (in *Installer) InstallForJob(ctx context.Context, id, jobID string, emit func(int, string)) (rec InstallRecord, installErr error) {
	release, err := in.useGate().AcquireExclusive(id)
	if err != nil {
		return InstallRecord{}, err
	}
	defer release()
	if emit == nil {
		emit = func(int, string) {}
	}
	a, err := in.Resolve(ctx, id)
	if err != nil {
		return InstallRecord{}, err
	}
	acquirer := fetch.Acquirer{Root: in.Root, Download: in.Download, RepoFetch: in.RepoFetch, DiskAvail: in.DiskAvail}
	if in.Installed(ctx, id) {
		rec := InstallRecord{ID: id, Installed: true, Path: in.adapterDir(id)}
		if in.State != nil {
			stored, _, readErr := in.State.Get(ctx, id)
			if readErr != nil {
				return InstallRecord{}, readErr
			}
			rec = stored
		}
		if acquirer.ValidateInstalled(a.Source, rec.Path, rec.Checksum) == nil {
			return rec, nil
		}
	}
	var diskProgress func(context.Context, string, int64, int64) error
	if jobID == "" {
		jobID = fmt.Sprintf("%s-%d", id, in.now().UnixNano())
	}
	if a.Source.LocalPath == "" {
		if in.DiskAdmission == nil {
			return InstallRecord{}, fmt.Errorf("adapters: shared host disk admission is unavailable; refusing acquisition")
		}
		request, planErr := fetch.PlanInstallDiskJob(ctx, "image-tools:adapter-install", jobID, in.Root, a.SizeMBApprox, a.Source, 1)
		if planErr != nil {
			return InstallRecord{}, fmt.Errorf("adapters: disk peak planning failed before acquisition: %w", planErr)
		}
		acquirer.RepoFetch = fetch.PinHuggingFaceCache(acquirer.RepoFetch, fetch.HuggingFaceCachePath(request))
		lease, reserveErr := in.DiskAdmission.ReserveDiskJob(ctx, request)
		if reserveErr != nil {
			return InstallRecord{}, fmt.Errorf("adapters: reserve complete install disk peak: %w", reserveErr)
		}
		if lease == nil {
			return InstallRecord{}, fmt.Errorf("adapters: shared host disk admission returned no lease")
		}
		growthGuard, growthErr := fetch.NewDiskGrowthGuard(lease, request, 1, fetch.InstallOwnerCeilingBytes)
		if growthErr != nil {
			return InstallRecord{}, fmt.Errorf("adapters: create bounded disk growth observer: %w", growthErr)
		}
		diskProgress = fetch.DiskGrowthObserver(growthGuard)
		acquirer.DiskProgressObserver = diskProgress
		ctx, guard := fetch.MaintainDiskLease(ctx, lease, 45*time.Second)
		defer func() {
			if heartbeatErr := guard.Stop(); heartbeatErr != nil {
				installErr = errors.Join(installErr, heartbeatErr)
				return
			}
			if errors.Is(installErr, fetch.ErrCleanupUnverified) {
				installErr = errors.Join(installErr, fmt.Errorf("adapters: disk lease %s remains active because staging cleanup is unverified", lease.Reservation().ID))
				return
			}
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			receipt, completeErr := lease.Complete(cleanup)
			if completeErr != nil {
				installErr = errors.Join(installErr, fmt.Errorf("adapters: complete disk lease %s after install stopped: %w", receipt.ID, completeErr))
			}
		}()
	}
	acquired, err := acquirer.Acquire(ctx, a.Source, in.adapterDir(id), in.pinnedChecksum(ctx, id), a.SizeMBApprox, emit)
	if err != nil {
		return InstallRecord{}, fmt.Errorf("adapters: acquire %q: %w", id, err)
	}
	rec = InstallRecord{ID: id, Installed: true, Path: acquired.Path, Checksum: acquired.Checksum, SizeBytes: acquired.SizeBytes, InstalledAt: in.now()}
	if in.State != nil {
		if err := in.State.Set(ctx, rec); err != nil {
			return InstallRecord{}, err
		}
	}
	emit(100, "installed")
	return rec, nil
}

// Remove deletes an adapter's downloaded weights and clears its install record.
// A local adapter is unlinked (its referenced path is never deleted).
func (in *Installer) Remove(ctx context.Context, id string) error {
	release, err := in.useGate().AcquireExclusive(id)
	if err != nil {
		return err
	}
	defer release()
	a, err := in.Resolve(ctx, id)
	if err != nil {
		return err
	}
	if a.Source.LocalPath == "" {
		if rmErr := os.RemoveAll(in.adapterDir(id)); rmErr != nil {
			return fmt.Errorf("adapters: remove weights %q: %w", id, rmErr)
		}
	}
	if in.State != nil {
		return in.State.Delete(ctx, id)
	}
	return nil
}

// AddCustom registers a custom/imported adapter entry. The id must not collide
// with a seed adapter, and a declared local path must exist.
func (in *Installer) AddCustom(ctx context.Context, a Adapter) error {
	if in.Custom == nil {
		return errors.New("adapters: custom store unavailable")
	}
	if a.ID == "" {
		return errors.New("adapters: custom adapter id is required")
	}
	if in.Reg != nil {
		if _, ok := in.Reg.ByID(a.ID); ok {
			return fmt.Errorf("%w: %q", ErrCustomShadowsSeed, a.ID)
		}
	}
	if a.Source.LocalPath != "" && !pathExists(a.Source.LocalPath) {
		return fmt.Errorf("%w: %q", ErrLocalPathMissing, a.Source.LocalPath)
	}
	return in.Custom.Add(ctx, a)
}

func (in *Installer) pinnedChecksum(ctx context.Context, id string) string {
	if in.State == nil {
		return ""
	}
	if rec, ok, err := in.State.Get(ctx, id); err == nil && ok {
		return rec.Checksum
	}
	return ""
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
