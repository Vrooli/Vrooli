package fetch

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	ErrInsufficientDisk     = errors.New("fetch: insufficient disk space for download")
	ErrChecksumMismatch     = errors.New("fetch: downloaded artifact checksum mismatch")
	ErrNoDownloadSource     = errors.New("fetch: no download source")
	ErrLocalPathMissing     = errors.New("fetch: local artifact path does not exist")
	ErrRepoRevisionUnpinned = errors.New("fetch: repository revision must be pinned")
)

// Source is shared by weight and conditioning catalogs. Documentation URLs are
// never download targets; DownloadURL is an explicit custom artifact source.
type Source struct {
	DownloadURL  string   `json:"download_url"`
	SourceRepo   string   `json:"source_repo"`
	DocsURL      string   `json:"docs_url"`
	UpdateSource string   `json:"update_source"`
	LocalPath    string   `json:"local_path,omitempty"`
	Assets       []Asset  `json:"assets,omitempty"`
	Repo         RepoSpec `json:"repo,omitempty"`
	Manual       bool     `json:"manual,omitempty"`
	Checksum     Checksum `json:"checksum"`
}

type Checksum struct {
	Algo   string `json:"algo"`
	Value  string `json:"value"`
	Status string `json:"status"`
}

func (s Source) HasRepo() bool { return strings.TrimSpace(s.Repo.RepoID) != "" }
func (s Source) HasFetchStrategy() bool {
	return strings.TrimSpace(s.LocalPath) != "" || s.HasRepo() || len(s.Assets) > 0
}

// Acquirer owns validation and publication, while each catalog owns its
// lifecycle and durable record. Callers hold an exclusive collection lease.
type Acquirer struct {
	Root      string
	Download  Downloader
	RepoFetch RepoFetcher
	DiskAvail DiskSpaceFunc
}

type Acquisition struct {
	Path, Checksum string
	SizeBytes      int64
}

func (a Acquirer) Acquire(ctx context.Context, source Source, destination, pinned string, sizeMB int, emit func(int, string)) (out Acquisition, err error) {
	if emit == nil {
		emit = func(int, string) {}
	}
	if source.LocalPath != "" {
		info, err := os.Stat(source.LocalPath)
		if err != nil {
			return out, fmt.Errorf("%w: %s: %w", ErrLocalPathMissing, source.LocalPath, err)
		}
		out.Path = source.LocalPath
		if info.IsDir() {
			out.Checksum, out.SizeBytes, err = TreeManifestHash(out.Path)
		} else {
			out.Checksum, out.SizeBytes, err = HashFile(out.Path)
		}
		return out, err
	}
	if a.DiskAvail != nil {
		available, err := a.DiskAvail(a.Root)
		if err != nil {
			return out, fmt.Errorf("fetch: check disk space: %w", err)
		}
		if required := RequiredBytes(sizeMB); available < required {
			return out, fmt.Errorf("%w: need ~%d MiB free at %s, have %d MiB", ErrInsufficientDisk, required>>20, a.Root, available>>20)
		}
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return out, err
	}
	stage, err := os.MkdirTemp(filepath.Dir(destination), ".acquiring-*")
	if err != nil {
		return out, err
	}
	defer os.RemoveAll(stage)
	if source.HasRepo() {
		out.Checksum, out.SizeBytes, err = a.repo(ctx, source, stage, pinned, emit)
	} else {
		out.Checksum, out.SizeBytes, err = a.assets(ctx, source, stage, pinned, emit)
	}
	if err != nil {
		return out, err
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	// A failed replacement cannot destroy an existing installation. Both renames
	// stay on this filesystem; interrupted prior artifacts remain recoverable.
	previous := stage + ".previous"
	if err := os.Rename(destination, previous); err != nil && !errors.Is(err, os.ErrNotExist) {
		return out, err
	}
	if err := os.Rename(stage, destination); err != nil {
		if _, statErr := os.Stat(previous); statErr == nil {
			err = errors.Join(err, os.Rename(previous, destination))
		}
		return out, err
	}
	out.Path = destination
	return out, os.RemoveAll(previous)
}

// ValidateInstalled rechecks an existing installation against the current
// catalog manifest. The persisted checksum is a fallback pin for artifacts
// whose upstream catalog does not publish one; a newly declared checksum is
// authoritative when the catalog changes.
func (a Acquirer) ValidateInstalled(source Source, installed, pinned string) error {
	if source.LocalPath != "" {
		path := source.LocalPath
		var sum string
		var err error
		if info, statErr := os.Stat(path); statErr != nil {
			return fmt.Errorf("fetch: inspect local artifact: %w", statErr)
		} else if info.IsDir() {
			sum, _, err = TreeManifestHash(path)
		} else {
			sum, _, err = HashFile(path)
		}
		if err != nil {
			return err
		}
		return verifyChecksum(firstNonEmpty(source.Checksum.Value, pinned), sum)
	}
	if source.HasRepo() {
		sum, _, err := TreeManifestHash(installed)
		if err != nil {
			return err
		}
		return verifyChecksum(firstNonEmpty(source.Checksum.Value, pinned), sum)
	}
	assets, err := source.ResolvedAssets()
	if err != nil {
		return err
	}
	for i, asset := range assets {
		path := filepath.Join(installed, asset.Filename)
		if err := ValidateArtifact(path, asset); err != nil {
			return fmt.Errorf("fetch: installed artifact %q failed validation: %w", asset.Filename, err)
		}
		sum, _, err := HashFile(path)
		if err != nil {
			return err
		}
		expected := asset.SHA256
		if i == 0 && expected == "" {
			expected = firstNonEmpty(source.Checksum.Value, pinned)
		}
		if err := verifyChecksum(expected, sum); err != nil {
			return fmt.Errorf("fetch: installed artifact %q failed checksum validation: %w", asset.Filename, err)
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (a Acquirer) repo(ctx context.Context, source Source, dir, pinned string, emit func(int, string)) (string, int64, error) {
	if a.RepoFetch == nil {
		return "", 0, ErrNoDownloadSource
	}
	if strings.TrimSpace(source.Repo.Revision) == "" {
		return "", 0, ErrRepoRevisionUnpinned
	}
	emit(5, "fetching "+source.Repo.RepoID+"@"+ShortRev(source.Repo.Revision))
	if err := a.RepoFetch.Snapshot(ctx, source.Repo, dir, func(done, total int64) {
		if total > 0 {
			emit(5+int(80*done/total), "fetching "+source.Repo.RepoID)
		}
	}); err != nil {
		return "", 0, err
	}
	emit(88, "verifying snapshot")
	sum, size, err := TreeManifestHash(dir)
	if err != nil {
		return "", 0, err
	}
	expected := firstNonEmpty(source.Checksum.Value, pinned)
	return sum, size, verifyChecksum(expected, sum)
}

func (a Acquirer) assets(ctx context.Context, source Source, dir, pinned string, emit func(int, string)) (primary string, total int64, err error) {
	assets, err := source.ResolvedAssets()
	if err != nil {
		return "", 0, err
	}
	if a.Download == nil {
		return "", 0, ErrNoDownloadSource
	}
	for i, asset := range assets {
		if !filepath.IsLocal(asset.Filename) || strings.Contains(asset.Filename, "\\") || asset.Filename == "." {
			return "", 0, fmt.Errorf("fetch: invalid artifact filename %q", asset.Filename)
		}
		dest := filepath.Join(dir, asset.Filename)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return "", 0, err
		}
		base, span := 5+80*i/len(assets), 80/len(assets)
		emit(base, "downloading "+asset.Filename)
		if err := a.Download.Download(ctx, asset.URL, dest, func(done, total int64) {
			if total > 0 {
				emit(base+int(int64(span)*done/total), "downloading "+asset.Filename)
			}
		}); err != nil {
			return "", 0, err
		}
		emit(base+span, "validating "+asset.Filename)
		if err := ValidateArtifact(dest, asset); err != nil {
			return "", 0, err
		}
		sum, size, err := HashFile(dest)
		if err != nil {
			return "", 0, err
		}
		expected := asset.SHA256
		if i == 0 && expected == "" {
			primary = sum
			expected = firstNonEmpty(source.Checksum.Value, pinned)
		} else if i == 0 {
			primary = sum
		}
		if err := verifyChecksum(expected, sum); err != nil {
			return "", 0, err
		}
		total += size
	}
	return primary, total, nil
}

func (s Source) ResolvedAssets() ([]Asset, error) {
	assets := append([]Asset(nil), s.Assets...)
	if len(assets) == 0 && s.DownloadURL != "" {
		assets = []Asset{{URL: s.DownloadURL, SHA256: s.Checksum.Value}}
	}
	if len(assets) == 0 {
		return nil, ErrNoDownloadSource
	}
	for i := range assets {
		if strings.TrimSpace(assets[i].URL) == "" {
			return nil, fmt.Errorf("fetch: asset %d has no url", i)
		}
		if assets[i].Filename == "" {
			u, err := url.Parse(assets[i].URL)
			if err != nil {
				return nil, err
			}
			assets[i].Filename = path.Base(u.Path)
		}
	}
	return assets, nil
}

func RequiredBytes(sizeMB int) int64 { return max(256<<20, int64(float64(int64(sizeMB)<<20)*1.15)) }

func verifyChecksum(expected, actual string) error {
	if expected = strings.TrimSpace(expected); expected != "" && !strings.EqualFold(expected, actual) {
		return fmt.Errorf("%w: expected %s got %s", ErrChecksumMismatch, expected, actual)
	}
	return nil
}
