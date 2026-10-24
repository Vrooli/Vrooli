package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	ErrInsufficientDisk = errors.New("adapters: insufficient disk space for download")
	// ErrChecksumMismatch is returned when a downloaded artifact's hash does not
	// match the pinned/seed checksum. The partial download is removed.
	ErrChecksumMismatch = errors.New("adapters: downloaded artifact checksum mismatch")
	// ErrNoDownloadSource is returned when an adapter has no resolvable source.
	ErrNoDownloadSource = errors.New("adapters: adapter has no download source")
	// ErrAdapterNotFound is returned when an id is in neither the seed nor custom.
	ErrAdapterNotFound = errors.New("adapters: adapter not found")
	// ErrCustomShadowsSeed is returned when a custom entry reuses a seed id.
	ErrCustomShadowsSeed = errors.New("adapters: custom adapter id collides with a seed adapter")
	// ErrLocalPathMissing is returned when a custom local adapter points at a path
	// that does not exist.
	ErrLocalPathMissing = errors.New("adapters: custom adapter local path does not exist")
	// ErrRepoRevisionUnpinned is returned when a repo-source adapter has no pinned
	// commit SHA (required for a reproducible install + tree-manifest hash).
	ErrRepoRevisionUnpinned = errors.New("adapters: repo source has no pinned revision")
)

const diskMarginFraction = 1.15

const minDiskMarginBytes int64 = 256 << 20 // 256 MiB

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

// InstallRecord is the persisted install state of one adapter.
type InstallRecord struct {
	ID          string
	Installed   bool
	Path        string
	Checksum    string
	SizeBytes   int64
	InstalledAt time.Time
}

// InstallStore persists adapter install state in the adapter_install table.
type InstallStore struct {
	db SQLExecutor
}

// NewInstallStore constructs the install-state store over db.
func NewInstallStore(db SQLExecutor) *InstallStore { return &InstallStore{db: db} }

const upsertInstallSQL = `
INSERT INTO adapter_install (id, installed, path, checksum, size_bytes, installed_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  installed=excluded.installed, path=excluded.path, checksum=excluded.checksum,
  size_bytes=excluded.size_bytes, installed_at=excluded.installed_at`

// Set persists an install record (upsert by id).
func (s *InstallStore) Set(ctx context.Context, r InstallRecord) error {
	at := ""
	if !r.InstalledAt.IsZero() {
		at = r.InstalledAt.UTC().Format(time.RFC3339)
	}
	if _, err := s.db.ExecContext(ctx, upsertInstallSQL,
		r.ID, boolToInt(r.Installed), r.Path, r.Checksum, r.SizeBytes, at); err != nil {
		return fmt.Errorf("adapters: persist install %q: %w", r.ID, err)
	}
	return nil
}

// Get returns the install record for id (ok=false when absent).
func (s *InstallStore) Get(ctx context.Context, id string) (InstallRecord, bool, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, installed, path, checksum, size_bytes, installed_at FROM adapter_install WHERE id = ?", id)
	r, err := scanInstall(row.Scan)
	if errors.Is(err, errNoInstallRow) {
		return InstallRecord{}, false, nil
	}
	if err != nil {
		return InstallRecord{}, false, fmt.Errorf("adapters: get install %q: %w", id, err)
	}
	return r, true, nil
}

// List returns every install record.
func (s *InstallStore) List(ctx context.Context) ([]InstallRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, installed, path, checksum, size_bytes, installed_at FROM adapter_install ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("adapters: list installs: %w", err)
	}
	defer rows.Close()

	var out []InstallRecord
	for rows.Next() {
		r, scanErr := scanInstall(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("adapters: scan install: %w", scanErr)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("adapters: iterate installs: %w", err)
	}
	return out, nil
}

// Delete removes the install record for id.
func (s *InstallStore) Delete(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM adapter_install WHERE id = ?", id); err != nil {
		return fmt.Errorf("adapters: delete install %q: %w", id, err)
	}
	return nil
}

var errNoInstallRow = errors.New("no install row")

func scanInstall(scan func(...any) error) (InstallRecord, error) {
	var (
		r  InstallRecord
		in int
		at string
	)
	if err := scan(&r.ID, &in, &r.Path, &r.Checksum, &r.SizeBytes, &at); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return InstallRecord{}, errNoInstallRow
		}
		return InstallRecord{}, err
	}
	r.Installed = in != 0
	if at != "" {
		if t, err := time.Parse(time.RFC3339, at); err == nil {
			r.InstalledAt = t
		}
	}
	return r, nil
}

// CustomStore persists operator-registered / imported custom adapter entries.
type CustomStore struct {
	db SQLExecutor
}

// NewCustomStore constructs the custom-adapter store over db.
func NewCustomStore(db SQLExecutor) *CustomStore { return &CustomStore{db: db} }

// Add persists a custom adapter entry (upsert by id).
func (s *CustomStore) Add(ctx context.Context, a Adapter) error {
	raw, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("adapters: marshal custom %q: %w", a.ID, err)
	}
	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO custom_adapter (id, json) VALUES (?, ?) ON CONFLICT(id) DO UPDATE SET json=excluded.json",
		a.ID, string(raw)); err != nil {
		return fmt.Errorf("adapters: persist custom %q: %w", a.ID, err)
	}
	return nil
}

// List returns every custom adapter entry.
func (s *CustomStore) List(ctx context.Context) ([]Adapter, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT json FROM custom_adapter ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("adapters: list custom: %w", err)
	}
	defer rows.Close()

	var out []Adapter
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("adapters: scan custom: %w", err)
		}
		var a Adapter
		if err := json.Unmarshal([]byte(raw), &a); err != nil {
			return nil, fmt.Errorf("adapters: unmarshal custom: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("adapters: iterate custom: %w", err)
	}
	return out, nil
}

// Delete removes a custom adapter entry.
func (s *CustomStore) Delete(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM custom_adapter WHERE id = ?", id); err != nil {
		return fmt.Errorf("adapters: delete custom %q: %w", id, err)
	}
	return nil
}

// Installer downloads, verifies, removes, and registers adapter weights.
type Installer struct {
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
	return dirExists(rec.Path)
}

// Install downloads (or, for a local adapter, verifies) the weights for id,
// checks free disk space first, verifies/pins the checksum, and records the
// install. It is idempotent: an already-installed adapter returns its record.
func (in *Installer) Install(ctx context.Context, id string, emit func(progress int, message string)) (InstallRecord, error) {
	if emit == nil {
		emit = func(int, string) {}
	}
	a, err := in.Resolve(ctx, id)
	if err != nil {
		return InstallRecord{}, err
	}

	if in.Installed(ctx, id) {
		var rec InstallRecord
		if in.State != nil {
			if got, ok, _ := in.State.Get(ctx, id); ok {
				rec = got
			}
		}
		if rec.Path == "" {
			rec = InstallRecord{ID: id, Installed: true, Path: in.adapterDir(id)}
		}
		emit(100, "already installed")
		return rec, nil
	}

	// A local adapter is "installed" by reference — no download.
	if a.Source.LocalPath != "" {
		if !pathExists(a.Source.LocalPath) {
			return InstallRecord{}, fmt.Errorf("%w: %q", ErrLocalPathMissing, a.Source.LocalPath)
		}
		size, _ := pathSize(a.Source.LocalPath)
		rec := InstallRecord{
			ID: id, Installed: true, Path: a.Source.LocalPath,
			SizeBytes: size, InstalledAt: in.now(),
		}
		if in.State != nil {
			if err := in.State.Set(ctx, rec); err != nil {
				return InstallRecord{}, err
			}
		}
		emit(100, "registered local adapter")
		return rec, nil
	}

	if a.Source.HasRepo() {
		return in.installRepo(ctx, id, a, emit)
	}

	if len(a.Source.Assets) == 0 {
		return InstallRecord{}, fmt.Errorf("%w: %q (no assets, repo, or local_path)", ErrNoDownloadSource, id)
	}

	required := requiredBytes(a.SizeMBApprox)
	if in.DiskAvail != nil {
		avail, derr := in.DiskAvail(in.Root)
		if derr != nil {
			return InstallRecord{}, fmt.Errorf("adapters: check disk space: %w", derr)
		}
		if avail < required {
			return InstallRecord{}, fmt.Errorf("%w: need ~%d MiB free at %s, have %d MiB",
				ErrInsufficientDisk, required>>20, in.Root, avail>>20)
		}
	}

	dir := in.adapterDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return InstallRecord{}, fmt.Errorf("adapters: create adapter dir: %w", err)
	}

	primarySum, totalSize, err := in.downloadAssets(ctx, id, dir, a.Source.Assets, emit)
	if err != nil {
		_ = os.RemoveAll(dir)
		return InstallRecord{}, err
	}

	rec := InstallRecord{
		ID: id, Installed: true, Path: dir, Checksum: primarySum,
		SizeBytes: totalSize, InstalledAt: in.now(),
	}
	if in.State != nil {
		if err := in.State.Set(ctx, rec); err != nil {
			return InstallRecord{}, err
		}
	}
	emit(100, "installed")
	return rec, nil
}

// downloadAssets fetches, validates, and checksum-verifies every asset into dir.
func (in *Installer) downloadAssets(ctx context.Context, id, dir string, assets []fetch.Asset, emit func(progress int, message string)) (primarySum string, totalSize int64, err error) {
	pinned := in.pinnedChecksum(ctx, id)
	n := len(assets)
	for i, a := range assets {
		if strings.TrimSpace(a.URL) == "" {
			return "", 0, fmt.Errorf("adapters: %q asset %d has no url", id, i)
		}
		dest := filepath.Join(dir, a.Filename)
		base := 5 + int(float64(i)/float64(n)*80)
		span := 80 / n
		emit(base, "downloading "+a.Filename)
		if dlErr := in.Download.Download(ctx, a.URL, dest, func(done, total int64) {
			if total > 0 {
				emit(base+int(float64(done)/float64(total)*float64(span)), "downloading "+a.Filename)
			}
		}); dlErr != nil {
			return "", 0, fmt.Errorf("adapters: download %q asset %q: %w", id, a.Filename, dlErr)
		}

		emit(base+span, "validating "+a.Filename)
		if vErr := fetch.ValidateArtifact(dest, a); vErr != nil {
			return "", 0, vErr
		}

		sum, size, hErr := fetch.HashFile(dest)
		if hErr != nil {
			return "", 0, fmt.Errorf("adapters: hash artifact %q: %w", a.Filename, hErr)
		}
		totalSize += size

		expected := strings.TrimSpace(a.SHA256)
		if i == 0 && pinned != "" {
			expected = pinned
		}
		if expected != "" && !strings.EqualFold(expected, sum) {
			return "", 0, fmt.Errorf("%w: %q asset %q expected %s got %s", ErrChecksumMismatch, id, a.Filename, expected, sum)
		}
		if i == 0 {
			primarySum = sum
		}
	}
	return primarySum, totalSize, nil
}

// installRepo fetches a multi-file repo snapshot at its pinned revision.
func (in *Installer) installRepo(ctx context.Context, id string, a Adapter, emit func(progress int, message string)) (InstallRecord, error) {
	if in.RepoFetch == nil {
		return InstallRecord{}, fmt.Errorf("%w: %q is a repo source but no RepoFetcher is configured", ErrNoDownloadSource, id)
	}
	if strings.TrimSpace(a.Source.Repo.Revision) == "" {
		return InstallRecord{}, fmt.Errorf("%w: %q", ErrRepoRevisionUnpinned, id)
	}

	required := requiredBytes(a.SizeMBApprox)
	if in.DiskAvail != nil {
		avail, derr := in.DiskAvail(in.Root)
		if derr != nil {
			return InstallRecord{}, fmt.Errorf("adapters: check disk space: %w", derr)
		}
		if avail < required {
			return InstallRecord{}, fmt.Errorf("%w: need ~%d MiB free at %s, have %d MiB",
				ErrInsufficientDisk, required>>20, in.Root, avail>>20)
		}
	}

	dir := in.adapterDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return InstallRecord{}, fmt.Errorf("adapters: create adapter dir: %w", err)
	}

	emit(5, "fetching "+a.Source.Repo.RepoID+"@"+fetch.ShortRev(a.Source.Repo.Revision))
	if err := in.RepoFetch.Snapshot(ctx, a.Source.Repo, dir, func(done, total int64) {
		if total > 0 {
			emit(5+int(float64(done)/float64(total)*80), "fetching "+a.Source.Repo.RepoID)
		}
	}); err != nil {
		_ = os.RemoveAll(dir)
		return InstallRecord{}, fmt.Errorf("adapters: fetch repo %q (%s): %w", id, a.Source.Repo.RepoID, err)
	}

	emit(88, "verifying snapshot")
	manifest, totalSize, err := fetch.TreeManifestHash(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return InstallRecord{}, fmt.Errorf("adapters: hash repo snapshot %q: %w", id, err)
	}

	expected := strings.TrimSpace(a.Source.Checksum.Value)
	if pinned := in.pinnedChecksum(ctx, id); pinned != "" {
		expected = pinned
	}
	if expected != "" && !strings.EqualFold(expected, manifest) {
		_ = os.RemoveAll(dir)
		return InstallRecord{}, fmt.Errorf("%w: %q repo snapshot expected %s got %s", ErrChecksumMismatch, id, expected, manifest)
	}

	rec := InstallRecord{
		ID: id, Installed: true, Path: dir, Checksum: manifest,
		SizeBytes: totalSize, InstalledAt: in.now(),
	}
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

func requiredBytes(sizeMBApprox int) int64 {
	base := int64(float64(int64(sizeMBApprox)<<20) * diskMarginFraction)
	if base < minDiskMarginBytes {
		return minDiskMarginBytes
	}
	return base
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func pathSize(p string) (int64, error) {
	info, err := os.Stat(p)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
