package installgateway

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"
)

const ytDlpUnpackedLimit = int64(512 << 20)

type ytDlpInstallReceipt struct {
	Package string `json:"package"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Wheel   string `json:"wheel"`
}

// installLocalYtDlpWheel installs one approved pure-Python wheel into a
// surface-owned directory. Its only network read is the exact wheel already
// sized and hashed by PlanDiskAdmission; dependency resolution and caches are
// disabled, and the caller holds the full disk lease across download, unpack,
// verification, and staging cleanup.
func installLocalYtDlpWheel(ctx context.Context, r Resolution) (output string, retErr error) {
	if r.PackageName != "yt-dlp" || !r.ExactVersion || r.DiskPythonExecutable == "" || r.DiskPythonWheelURL == "" || r.DiskPythonWheelSHA256 == "" || r.DiskPythonWheelBytes <= 0 || r.DiskPythonTarget == "" || r.DiskPythonStage == "" {
		return "", fmt.Errorf("surface-local yt-dlp install plan is incomplete")
	}
	if !pathWithin(r.SurfaceRoot, r.DiskPythonTarget) || !pathWithin(r.SurfaceRoot, r.DiskPythonStage) {
		return "", fmt.Errorf("surface-local yt-dlp install path escapes its governed surface")
	}
	if err := validateNoSymlinkPathComponents(r.SurfaceRoot, r.DiskPythonTarget); err != nil {
		return "", fmt.Errorf("surface-local yt-dlp target became unsafe: %w", err)
	}
	if err := validateNoSymlinkPathComponents(r.SurfaceRoot, r.DiskPythonStage); err != nil {
		return "", fmt.Errorf("surface-local yt-dlp staging path became unsafe: %w", err)
	}
	targetCreated, keepTarget, stageCreated := false, false, false
	defer func() {
		if stageCreated {
			if err := os.RemoveAll(r.DiskPythonStage); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("remove yt-dlp staging directory: %w", err))
			} else if _, err := os.Lstat(r.DiskPythonStage); !os.IsNotExist(err) {
				if err == nil {
					err = fmt.Errorf("staging directory still exists")
				}
				retErr = errors.Join(retErr, fmt.Errorf("verify yt-dlp staging cleanup: %w", err))
			}
		}
		if targetCreated && !keepTarget {
			if err := os.RemoveAll(r.DiskPythonTarget); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("remove incomplete yt-dlp target: %w", err))
			} else if _, err := os.Lstat(r.DiskPythonTarget); !os.IsNotExist(err) {
				if err == nil {
					err = fmt.Errorf("target directory still exists")
				}
				retErr = errors.Join(retErr, fmt.Errorf("verify incomplete yt-dlp target cleanup: %w", err))
			}
		}
	}()

	if err := os.MkdirAll(filepath.Dir(r.DiskPythonStage), 0o700); err != nil {
		return "", err
	}
	if err := os.Mkdir(r.DiskPythonStage, 0o700); err != nil {
		return "", fmt.Errorf("create private yt-dlp staging directory: %w", err)
	}
	stageCreated = true
	if _, err := os.Lstat(r.DiskPythonTarget); err == nil {
		return "", fmt.Errorf("yt-dlp target already exists; refusing replacement")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(r.DiskPythonTarget), 0o700); err != nil {
		return "", fmt.Errorf("create yt-dlp target parent: %w", err)
	}
	targetCreated = true
	wheelPath := filepath.Join(r.DiskPythonStage, "yt-dlp-2026.10.26-py3-none-any.whl")
	if err := downloadVerifiedWheel(ctx, r.DiskPythonWheelURL, r.DiskPythonWheelSHA256, r.DiskPythonWheelBytes, wheelPath); err != nil {
		return "", err
	}
	if err := validateWheelExpansion(wheelPath, ytDlpUnpackedLimit); err != nil {
		return "", err
	}
	args := []string{"-m", "pip", "install", "--disable-pip-version-check", "--no-input", "--no-deps", "--no-index", "--only-binary=:all:", "--no-cache-dir", "--no-warn-script-location", "--target", r.DiskPythonTarget, wheelPath}
	cmd := exec.CommandContext(ctx, r.DiskPythonExecutable, args...)
	cmd.Dir = r.SurfaceRoot
	cmd.WaitDelay = 5 * time.Second
	cmd.Env = overlayInstallEnvironment(os.Environ(), []string{
		"PIP_CONFIG_FILE=" + os.DevNull,
		"PIP_NO_CACHE_DIR=1",
		"PIP_DISABLE_PIP_VERSION_CHECK=1",
		"PYTHONDONTWRITEBYTECODE=1",
		"TMPDIR=" + r.DiskPythonStage,
		"TMP=" + r.DiskPythonStage,
		"TEMP=" + r.DiskPythonStage,
	})
	var diagnostic hostToolOutput
	cmd.Stdout, cmd.Stderr = &diagnostic, &diagnostic
	if err := cmd.Run(); err != nil {
		return diagnostic.String(), fmt.Errorf("install verified yt-dlp wheel into surface target: %w", err)
	}
	metadataPath, err := filepath.Glob(filepath.Join(r.DiskPythonTarget, "yt_dlp-*.dist-info", "METADATA"))
	if err != nil || len(metadataPath) != 1 {
		return diagnostic.String(), fmt.Errorf("verify installed yt-dlp wheel metadata: expected one dist-info METADATA")
	}
	metadata, err := readRegularBounded(metadataPath[0], 1<<20)
	if err != nil || !strings.Contains(string(metadata), "Name: yt-dlp\n") || !strings.Contains(string(metadata), "Version: 2026.10.26\n") {
		return diagnostic.String(), fmt.Errorf("installed yt-dlp wheel identity differs from the approved pin")
	}
	receipt := ytDlpInstallReceipt{Package: "yt-dlp", Version: "2026.10.26", SHA256: r.DiskPythonWheelSHA256, Wheel: filepath.Base(wheelPath)}
	receiptBytes, _ := json.MarshalIndent(receipt, "", "  ")
	if err := os.WriteFile(filepath.Join(r.DiskPythonTarget, "sda-install-receipt.json"), append(receiptBytes, '\n'), 0o600); err != nil {
		return diagnostic.String(), fmt.Errorf("write local yt-dlp installation receipt: %w", err)
	}
	targetCreated, keepTarget = true, true
	return diagnostic.String(), nil
}

func downloadVerifiedWheel(ctx context.Context, source, wantSHA string, wantBytes int64, target string) error {
	parsed, err := url.Parse(source)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "files.pythonhosted.org" || parsed.User != nil || wantBytes <= 0 || wantBytes > 64<<20 {
		return fmt.Errorf("refuse unbounded or untrusted yt-dlp wheel source")
	}
	downloadCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	request, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download approved yt-dlp wheel: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.ContentLength != wantBytes {
		return fmt.Errorf("yt-dlp wheel response size/status differs from bounded metadata")
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	count, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, wantBytes+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || count != wantBytes {
		_ = os.Remove(target)
		return fmt.Errorf("yt-dlp wheel download did not match its bounded size")
	}
	if hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(wantSHA) {
		_ = os.Remove(target)
		return fmt.Errorf("yt-dlp wheel SHA256 does not match PyPI metadata")
	}
	return nil
}

func validateWheelExpansion(path string, maxBytes int64) error {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("inspect downloaded yt-dlp wheel: %w", err)
	}
	defer archive.Close()
	if len(archive.File) == 0 || len(archive.File) > 10000 {
		return fmt.Errorf("yt-dlp wheel has an invalid or oversized entry count")
	}
	var expanded uint64
	for _, entry := range archive.File {
		clean := pathpkg.Clean(filepath.ToSlash(entry.Name))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || entry.Mode()&os.ModeSymlink != 0 || (!entry.Mode().IsRegular() && !entry.Mode().IsDir()) {
			return fmt.Errorf("yt-dlp wheel contains an unsafe filesystem entry")
		}
		if entry.UncompressedSize64 > uint64(maxBytes)-expanded {
			return fmt.Errorf("yt-dlp wheel expansion exceeds the %d-byte safety bound", maxBytes)
		}
		expanded += entry.UncompressedSize64
	}
	return nil
}

func pathWithin(root, candidate string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, candidateAbs)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// VerifyDiskAdmissionCleanup is called by the API owner before lease Complete.
// It proves the custom wheel staging directory is gone and that failed installs
// left no partial target behind.
func VerifyDiskAdmissionCleanup(r Resolution, successful bool) error {
	if r.PackageManager != "pip-local-wheel" {
		return nil
	}
	if _, err := os.Lstat(r.DiskPythonStage); !os.IsNotExist(err) {
		if err == nil {
			err = fmt.Errorf("yt-dlp staging directory remains")
		}
		return err
	}
	if !successful {
		if _, err := os.Lstat(r.DiskPythonTarget); !os.IsNotExist(err) {
			if err == nil {
				err = fmt.Errorf("partial yt-dlp target remains")
			}
			return err
		}
	}
	return nil
}
