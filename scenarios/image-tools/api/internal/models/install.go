package models

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
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
	ErrInsufficientDisk = errors.New("models: insufficient disk space for download")
	// ErrChecksumMismatch is returned when a downloaded artifact's hash does not
	// match the pinned/seed checksum. The partial download is removed.
	ErrChecksumMismatch = errors.New("models: downloaded artifact checksum mismatch")
	// ErrNoDownloadSource is returned when a model has neither a download URL nor
	// a local path to install from.
	ErrNoDownloadSource = errors.New("models: model has no download source")
	// ErrModelNotFound is returned when an id is in neither the seed nor custom.
	ErrModelNotFound = errors.New("models: model not found")
	// ErrCustomShadowsSeed is returned when a custom entry reuses a seed id.
	ErrCustomShadowsSeed = errors.New("models: custom model id collides with a seed model")
	// ErrLocalPathMissing is returned when a custom local model points at a path
	// that does not exist.
	ErrLocalPathMissing = errors.New("models: custom model local path does not exist")
)

// diskMarginFraction reserves headroom above the model's approximate size so an
// install never fills the volume to the brim. Approximate sizes are rough, so the
// margin is deliberately generous.
const diskMarginFraction = 1.15

// minDiskMarginBytes is a floor on the required free space, so even a tiny model
// won't install onto a near-full disk.
const minDiskMarginBytes int64 = 256 << 20 // 256 MiB

// InstallJobOperation is the job.Operation name a model download runs under. The
// models handler submits a job with this op + an InstallPayload; main.go registers
// the matching runner on the jobrunner dispatcher.
const InstallJobOperation = "model_install"

// InstallPayload is the JSON body of an install job.
type InstallPayload struct {
	ModelID string `json:"model_id"`
}

// EstimateInstallSeconds is a rough ETA for downloading a model of the given
// approximate size, assuming a conservative throughput. Surfaced as the job ETA.
func EstimateInstallSeconds(sizeMBApprox int) int {
	const assumedMBPerSec = 15
	if sizeMBApprox <= 0 {
		return 0
	}
	if eta := sizeMBApprox / assumedMBPerSec; eta > 0 {
		return eta
	}
	return 1
}

// Downloader fetches a remote artifact to a local path. It is a seam so the
// engine is testable without network access; the production implementation is
// HTTPDownloader. emit reports byte progress (total may be -1 if unknown).
type Downloader interface {
	Download(ctx context.Context, rawURL, destPath string, emit func(done, total int64)) error
}

// DiskSpaceFunc reports the free bytes available at path's filesystem. The
// production implementation (diskAvail) wraps statfs; tests inject a fake.
type DiskSpaceFunc func(path string) (availBytes int64, err error)

// DefaultDiskAvail is the production DiskSpaceFunc (statfs-backed on unix, a
// large-value fallback elsewhere). Exported so main.go can wire it without a
// build-tagged reference.
func DefaultDiskAvail(path string) (int64, error) { return diskAvail(path) }

// InstallRecord is the persisted install state of one model.
type InstallRecord struct {
	ID          string
	Installed   bool
	Path        string
	Checksum    string
	SizeBytes   int64
	InstalledAt time.Time
}

// InstallStore persists model install state in the model_install table.
type InstallStore struct {
	db SQLExecutor
}

// NewInstallStore constructs the install-state store over db.
func NewInstallStore(db SQLExecutor) *InstallStore { return &InstallStore{db: db} }

const upsertInstallSQL = `
INSERT INTO model_install (id, installed, path, checksum, size_bytes, installed_at)
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
		return fmt.Errorf("models: persist install %q: %w", r.ID, err)
	}
	return nil
}

// Get returns the install record for id (ok=false when absent).
func (s *InstallStore) Get(ctx context.Context, id string) (InstallRecord, bool, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, installed, path, checksum, size_bytes, installed_at FROM model_install WHERE id = ?", id)
	r, err := scanInstall(row.Scan)
	if errors.Is(err, errNoInstallRow) {
		return InstallRecord{}, false, nil
	}
	if err != nil {
		return InstallRecord{}, false, fmt.Errorf("models: get install %q: %w", id, err)
	}
	return r, true, nil
}

// List returns every install record.
func (s *InstallStore) List(ctx context.Context) ([]InstallRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, installed, path, checksum, size_bytes, installed_at FROM model_install ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("models: list installs: %w", err)
	}
	defer rows.Close()

	var out []InstallRecord
	for rows.Next() {
		r, scanErr := scanInstall(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("models: scan install: %w", scanErr)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("models: iterate installs: %w", err)
	}
	return out, nil
}

// Delete removes the install record for id.
func (s *InstallStore) Delete(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM model_install WHERE id = ?", id); err != nil {
		return fmt.Errorf("models: delete install %q: %w", id, err)
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

// CustomStore persists operator-registered custom/local model entries.
type CustomStore struct {
	db SQLExecutor
}

// NewCustomStore constructs the custom-model store over db.
func NewCustomStore(db SQLExecutor) *CustomStore { return &CustomStore{db: db} }

// Add persists a custom model entry (upsert by id).
func (s *CustomStore) Add(ctx context.Context, m Model) error {
	raw, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("models: marshal custom %q: %w", m.ID, err)
	}
	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO custom_model (id, json) VALUES (?, ?) ON CONFLICT(id) DO UPDATE SET json=excluded.json",
		m.ID, string(raw)); err != nil {
		return fmt.Errorf("models: persist custom %q: %w", m.ID, err)
	}
	return nil
}

// List returns every custom model entry.
func (s *CustomStore) List(ctx context.Context) ([]Model, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT json FROM custom_model ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("models: list custom: %w", err)
	}
	defer rows.Close()

	var out []Model
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("models: scan custom: %w", err)
		}
		var m Model
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return nil, fmt.Errorf("models: unmarshal custom: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("models: iterate custom: %w", err)
	}
	return out, nil
}

// Delete removes a custom model entry.
func (s *CustomStore) Delete(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM custom_model WHERE id = ?", id); err != nil {
		return fmt.Errorf("models: delete custom %q: %w", id, err)
	}
	return nil
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
	// DiskAvail reports free space at Root (diskAvail in production).
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

// modelDir is the absolute directory model id's weights live under.
func (in *Installer) modelDir(id string) string {
	return filepath.Join(in.Root, "models", id)
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
// record AND the directory exists on disk).
func (in *Installer) Installed(ctx context.Context, id string) bool {
	if in.State == nil {
		return dirExists(in.modelDir(id))
	}
	rec, ok, err := in.State.Get(ctx, id)
	if err != nil || !ok || !rec.Installed {
		return false
	}
	return dirExists(rec.Path)
}

// Install downloads (or, for a custom local model, verifies) the weights for id,
// checks free disk space first, verifies/pins the checksum, and records the
// install. It is idempotent: an already-installed model returns its record
// without re-downloading. emit reports human-readable progress.
func (in *Installer) Install(ctx context.Context, id string, emit func(progress int, message string)) (InstallRecord, error) {
	if emit == nil {
		emit = func(int, string) {}
	}
	m, err := in.Resolve(ctx, id)
	if err != nil {
		return InstallRecord{}, err
	}

	// Idempotent: already installed on disk → no-op.
	if in.Installed(ctx, id) {
		emit(100, "already installed")
		if in.State != nil {
			if rec, ok, _ := in.State.Get(ctx, id); ok {
				return rec, nil
			}
		}
		return InstallRecord{ID: id, Installed: true, Path: in.modelDir(id)}, nil
	}

	// A custom local model is "installed" by reference — no download.
	if m.Source.LocalPath != "" {
		if !pathExists(m.Source.LocalPath) {
			return InstallRecord{}, fmt.Errorf("%w: %q", ErrLocalPathMissing, m.Source.LocalPath)
		}
		size, _ := pathSize(m.Source.LocalPath)
		rec := InstallRecord{
			ID: id, Installed: true, Path: m.Source.LocalPath,
			SizeBytes: size, InstalledAt: in.now(),
		}
		if in.State != nil {
			if err := in.State.Set(ctx, rec); err != nil {
				return InstallRecord{}, err
			}
		}
		emit(100, "registered local model")
		return rec, nil
	}

	// Resolve the asset list. Seed models declare Source.Assets (direct,
	// resolvable, artifact-validated weights). An operator-registered custom
	// model carries a single Source.DownloadURL, treated as one generic asset —
	// and now validated the same way, so a custom URL that points at a page is
	// rejected instead of being recorded as a model.
	assets, err := resolveAssets(m)
	if err != nil {
		return InstallRecord{}, err
	}

	// Disk-space awareness: refuse before downloading if there isn't room.
	required := requiredBytes(m.SizeMBApprox)
	if in.DiskAvail != nil {
		avail, derr := in.DiskAvail(in.Root)
		if derr != nil {
			return InstallRecord{}, fmt.Errorf("models: check disk space: %w", derr)
		}
		if avail < required {
			return InstallRecord{}, fmt.Errorf("%w: need ~%d MiB free at %s, have %d MiB",
				ErrInsufficientDisk, required>>20, in.Root, avail>>20)
		}
	}

	dir := in.modelDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return InstallRecord{}, fmt.Errorf("models: create model dir: %w", err)
	}

	primarySum, totalSize, err := in.downloadAssets(ctx, id, dir, assets, emit)
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

// resolveAssets returns the artifacts to download for a model: its declared
// Source.Assets when present, else a single generic asset synthesized from the
// custom-model Source.DownloadURL. A model with neither cannot be installed —
// its seed entry points only at a documentation page (the un-migrated case).
func resolveAssets(m Model) ([]Asset, error) {
	if len(m.Source.Assets) > 0 {
		out := make([]Asset, 0, len(m.Source.Assets))
		for i, a := range m.Source.Assets {
			if a.URL == "" {
				return nil, fmt.Errorf("models: %q asset %d has no url", m.ID, i)
			}
			if a.Filename == "" {
				a.Filename = artifactName(a.URL)
			}
			out = append(out, a)
		}
		return out, nil
	}
	if m.Source.DownloadURL != "" {
		// Carry the seed's published checksum (if any) onto the synthesized asset.
		return []Asset{{
			URL:      m.Source.DownloadURL,
			Filename: artifactName(m.Source.DownloadURL),
			Kind:     ArtifactGeneric,
			SHA256:   strings.TrimSpace(m.Source.Checksum.Value),
		}}, nil
	}
	return nil, fmt.Errorf("%w: %q (its source has no resolvable weight asset — only a documentation page)", ErrNoDownloadSource, m.ID)
}

// downloadAssets fetches, validates, and (where applicable) checksum-verifies
// every asset into dir. It returns the primary (first) asset's sha256 to pin and
// the total bytes written. Any failure leaves cleanup to the caller (the model
// dir is removed). Validation rejects HTML pages / wrong-format / truncated
// downloads BEFORE the install is recorded — the install-stub fix.
func (in *Installer) downloadAssets(ctx context.Context, id, dir string, assets []Asset, emit func(progress int, message string)) (primarySum string, totalSize int64, err error) {
	pinned := in.pinnedChecksum(ctx, id)
	n := len(assets)
	for i, a := range assets {
		dest := filepath.Join(dir, a.Filename)
		base := 5 + int(float64(i)/float64(n)*80)
		span := 80 / n
		emit(base, "downloading "+a.Filename)
		if dlErr := in.Download.Download(ctx, a.URL, dest, func(done, total int64) {
			if total > 0 {
				emit(base+int(float64(done)/float64(total)*float64(span)), "downloading "+a.Filename)
			}
		}); dlErr != nil {
			return "", 0, fmt.Errorf("models: download %q asset %q: %w", id, a.Filename, dlErr)
		}

		emit(base+span, "validating "+a.Filename)
		if vErr := validateArtifact(dest, a); vErr != nil {
			return "", 0, vErr
		}

		sum, size, hErr := hashFile(dest)
		if hErr != nil {
			return "", 0, fmt.Errorf("models: hash artifact %q: %w", a.Filename, hErr)
		}
		totalSize += size

		// Checksum precedence for the PRIMARY asset: a previously pinned value
		// wins; then the asset's published sha256. A mismatch fails the install.
		// Absence pins the freshly computed hash AFTER artifact validation passed
		// (never hand-written — see DECISIONS).
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

// Remove deletes a model's downloaded weights and clears its install record. A
// custom local model is unlinked (its referenced path is never deleted).
func (in *Installer) Remove(ctx context.Context, id string) error {
	m, err := in.Resolve(ctx, id)
	if err != nil {
		return err
	}
	if m.Source.LocalPath == "" {
		if rmErr := os.RemoveAll(in.modelDir(id)); rmErr != nil {
			return fmt.Errorf("models: remove weights %q: %w", id, rmErr)
		}
	}
	if in.State != nil {
		return in.State.Delete(ctx, id)
	}
	return nil
}

// AddCustom registers a custom/local model entry. The id must not collide with a
// seed model, and a declared local path must exist.
func (in *Installer) AddCustom(ctx context.Context, m Model) error {
	if in.Custom == nil {
		return errors.New("models: custom store unavailable")
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
	return in.Custom.Add(ctx, m)
}

// pinnedChecksum returns the previously pinned checksum for id, or "".
func (in *Installer) pinnedChecksum(ctx context.Context, id string) string {
	if in.State == nil {
		return ""
	}
	if rec, ok, err := in.State.Get(ctx, id); err == nil && ok {
		return rec.Checksum
	}
	return ""
}

// requiredBytes is the free space an install needs: the model's approximate size
// scaled by the safety margin, floored at minDiskMarginBytes.
func requiredBytes(sizeMBApprox int) int64 {
	base := int64(float64(int64(sizeMBApprox)<<20) * diskMarginFraction)
	if base < minDiskMarginBytes {
		return minDiskMarginBytes
	}
	return base
}

// artifactName derives a local filename from a download URL, defaulting to
// "model.bin" when the URL has no usable path component.
func artifactName(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err == nil {
		if base := path.Base(u.Path); base != "" && base != "/" && base != "." {
			return base
		}
	}
	return "model.bin"
}

func hashFile(p string) (sum string, size int64, err error) {
	f, err := os.Open(p) //nolint:gosec // path is engine-constructed under Root
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
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

// HTTPDownloader is the production Downloader: a streaming HTTP GET.
type HTTPDownloader struct {
	Client *http.Client
}

// Download streams rawURL to destPath, reporting byte progress.
func (d HTTPDownloader) Download(ctx context.Context, rawURL, destPath string, emit func(done, total int64)) error {
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %s", rawURL, resp.Status)
	}

	f, err := os.Create(destPath) //nolint:gosec // destPath is engine-constructed under Root
	if err != nil {
		return err
	}
	defer f.Close()

	total := resp.ContentLength
	src := &progressReader{r: resp.Body, total: total, emit: emit}
	if _, err := io.Copy(f, src); err != nil {
		return err
	}
	return nil
}

type progressReader struct {
	r     io.Reader
	done  int64
	total int64
	emit  func(done, total int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.done += int64(n)
	if pr.emit != nil {
		pr.emit(pr.done, pr.total)
	}
	return n, err
}
