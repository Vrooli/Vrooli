package binaryfetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Target is the fetch spec for one binary on one os/arch. It mirrors the
// data-declared tool.json source target, decoupled from the platform structs so
// this package stays standard-library-only.
type Target struct {
	// Name is the installed binary's final filename (the command name).
	Name string
	// URL is the download location of the binary or archive.
	URL string
	// SHA256 is the mandatory lower/upper-hex digest of the downloaded bytes
	// (the archive when archived, else the raw binary).
	SHA256 string
	// Archive is "tar.gz", "zip", or "" / "none" for a raw binary.
	Archive string
	// Layout is "file" (default) or "dir". It is informational for Fetch (which
	// always installs a single file) and consulted by callers to choose FetchDir.
	Layout string
	// BinPath is the path of the binary inside the archive. When empty, the
	// archive must contain exactly one regular file. For FetchDir it is the entry
	// binary's path relative to the extracted opt dir.
	BinPath string
	// Mode is the octal file mode for the placed binary; defaults to 0755.
	Mode string
}

// Stage names the phase reported through ProgressFunc.
type Stage string

const (
	StageDownloading Stage = "downloading"
	StageVerifying   Stage = "verifying"
	StageExtracting  Stage = "extracting"
	StageInstalling  Stage = "installing"
)

// Progress is a point-in-time update emitted during a Fetch.
type Progress struct {
	Stage         Stage
	BytesComplete int64
	BytesTotal    int64 // 0 when the server did not advertise Content-Length
}

// ProgressFunc receives progress updates. It may be nil.
type ProgressFunc func(Progress)

func (p ProgressFunc) emit(progress Progress) {
	if p != nil {
		p(progress)
	}
}

// HTTPClient is the client used for downloads; overridable in tests.
var HTTPClient = http.DefaultClient

// Fetch downloads the target into destDir and returns the absolute path of the
// installed binary. It is context-cancelable, verifies the SHA-256, rejects HTML
// pages and truncated downloads, extracts archives, and atomically places the
// executable. destDir is created if absent.
func Fetch(ctx context.Context, spec Target, destDir string, onProgress ProgressFunc) (string, error) {
	if strings.TrimSpace(spec.URL) == "" {
		return "", fmt.Errorf("binaryfetch: target URL is required")
	}
	if strings.TrimSpace(spec.SHA256) == "" {
		return "", fmt.Errorf("binaryfetch: target SHA256 is required (checksums are mandatory for executables)")
	}
	binName := strings.TrimSpace(spec.Name)
	if binName == "" {
		return "", fmt.Errorf("binaryfetch: target Name is required")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("binaryfetch: create dest dir: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "binaryfetch-*")
	if err != nil {
		return "", fmt.Errorf("binaryfetch: temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	downloadPath := filepath.Join(tmpDir, "download")
	if err := download(ctx, spec.URL, downloadPath, onProgress); err != nil {
		return "", err
	}

	onProgress.emit(Progress{Stage: StageVerifying})
	if err := verifyChecksum(downloadPath, spec.SHA256); err != nil {
		return "", err
	}

	// Resolve the binary bytes (raw download or extracted from archive) into a
	// staging file in tmpDir.
	staged := filepath.Join(tmpDir, "staged")
	format := normalizeArchive(spec.Archive)
	if format == "" {
		// Raw binary: the download IS the binary. Guard it before placing.
		if err := ValidateNotHTMLAndSized(downloadPath, binName, 0); err != nil {
			return "", err
		}
		staged = downloadPath
	} else {
		onProgress.emit(Progress{Stage: StageExtracting})
		if err := extractBinary(downloadPath, format, spec.BinPath, staged); err != nil {
			return "", err
		}
		if err := ValidateNotHTMLAndSized(staged, binName, 0); err != nil {
			return "", err
		}
	}

	onProgress.emit(Progress{Stage: StageInstalling})
	finalPath := filepath.Join(destDir, binName)
	if err := atomicPlace(staged, finalPath, fileMode(spec.Mode)); err != nil {
		return "", err
	}
	return finalPath, nil
}

// FetchDir downloads + sha256-verifies the target archive and extracts its
// ENTIRE tree into optDir (cleaning optDir first so reinstalls are idempotent).
// It validates that the entry binary at <optDir>/<spec.BinPath> exists, is not an
// HTML page, and clears the size floor, chmods it +x, and returns its absolute
// path. The archive is preserved as-is (relative paths intact); BinPath is
// resolved relative to the archive root. spec.BinPath is required.
func FetchDir(ctx context.Context, spec Target, optDir string, onProgress ProgressFunc) (string, error) {
	if strings.TrimSpace(spec.URL) == "" {
		return "", fmt.Errorf("binaryfetch: target URL is required")
	}
	if strings.TrimSpace(spec.SHA256) == "" {
		return "", fmt.Errorf("binaryfetch: target SHA256 is required (checksums are mandatory for executables)")
	}
	binPath := strings.TrimSpace(spec.BinPath)
	if binPath == "" {
		return "", fmt.Errorf("binaryfetch: target BinPath is required for a dir-layout install")
	}
	format := normalizeArchive(spec.Archive)
	if format == "" {
		return "", fmt.Errorf("binaryfetch: dir-layout install requires an archive (got none)")
	}

	tmpDir, err := os.MkdirTemp("", "binaryfetch-*")
	if err != nil {
		return "", fmt.Errorf("binaryfetch: temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	downloadPath := filepath.Join(tmpDir, "download")
	if err := download(ctx, spec.URL, downloadPath, onProgress); err != nil {
		return "", err
	}

	onProgress.emit(Progress{Stage: StageVerifying})
	if err := verifyChecksum(downloadPath, spec.SHA256); err != nil {
		return "", err
	}

	onProgress.emit(Progress{Stage: StageExtracting})
	// Extract into a fresh staging dir, validate the entry binary, then swap the
	// staging dir into optDir so a failed extraction never leaves a partial opt.
	stageDir := filepath.Join(tmpDir, "extracted")
	if err := extractAll(downloadPath, format, stageDir); err != nil {
		return "", err
	}
	entry := filepath.Join(stageDir, filepath.Clean(binPath))
	if !strings.HasPrefix(entry, filepath.Clean(stageDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("binaryfetch: binPath %q escapes the archive root", binPath)
	}
	if info, statErr := os.Stat(entry); statErr != nil || info.IsDir() {
		return "", fmt.Errorf("%w: entry %q not found in archive", ErrNoBinaryInArchive, binPath)
	}
	if err := ValidateNotHTMLAndSized(entry, filepath.Base(binPath), 0); err != nil {
		return "", err
	}
	if err := os.Chmod(entry, fileMode(spec.Mode)); err != nil {
		return "", fmt.Errorf("binaryfetch: chmod entry binary: %w", err)
	}

	onProgress.emit(Progress{Stage: StageInstalling})
	if err := os.RemoveAll(optDir); err != nil {
		return "", fmt.Errorf("binaryfetch: clean opt dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(optDir), 0o755); err != nil {
		return "", fmt.Errorf("binaryfetch: create opt parent: %w", err)
	}
	if err := os.Rename(stageDir, optDir); err != nil {
		return "", fmt.Errorf("binaryfetch: place opt dir: %w", err)
	}
	return filepath.Join(optDir, filepath.Clean(binPath)), nil
}

func download(ctx context.Context, url, destPath string, onProgress ProgressFunc) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("binaryfetch: build request: %w", err)
	}
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("binaryfetch: download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("binaryfetch: download %s: unexpected status %s", url, resp.Status)
	}

	out, err := os.Create(destPath) //nolint:gosec // destPath is under a controlled temp dir
	if err != nil {
		return fmt.Errorf("binaryfetch: create download file: %w", err)
	}
	defer out.Close()

	total := resp.ContentLength
	onProgress.emit(Progress{Stage: StageDownloading, BytesTotal: max64(total, 0)})
	pw := &progressWriter{onProgress: onProgress, total: total}
	if _, err := io.Copy(out, io.TeeReader(resp.Body, pw)); err != nil {
		return fmt.Errorf("binaryfetch: stream download: %w", err)
	}
	return nil
}

type progressWriter struct {
	onProgress ProgressFunc
	total      int64
	written    int64
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.written += int64(n)
	w.onProgress.emit(Progress{Stage: StageDownloading, BytesComplete: w.written, BytesTotal: max64(w.total, 0)})
	return n, nil
}

func verifyChecksum(path, want string) error {
	f, err := os.Open(path) //nolint:gosec // path is under a controlled temp dir
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("binaryfetch: hash download: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, strings.TrimSpace(want)) {
		return fmt.Errorf("%w: got %s, want %s", ErrChecksumMismatch, got, strings.ToLower(strings.TrimSpace(want)))
	}
	return nil
}

// atomicPlace copies src to a sibling temp of dst, then renames it into place so
// a concurrent reader never sees a half-written binary.
func atomicPlace(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src) //nolint:gosec // src is under a controlled temp dir
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("binaryfetch: create install temp: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return fmt.Errorf("binaryfetch: copy into place: %w", err)
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	// Ensure the mode survives umask (OpenFile mode is masked by umask).
	if err := os.Chmod(tmp, mode); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("binaryfetch: chmod: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("binaryfetch: atomic rename: %w", err)
	}
	return nil
}

func fileMode(mode string) os.FileMode {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return 0o755
	}
	parsed, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return 0o755
	}
	return os.FileMode(parsed)
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
