// Package agentinstall contains shell-free host installation primitives shared
// by external-cli resources.
package agentinstall

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/config"
)

// Spec describes one user-owned upstream CLI installation.
type Spec struct {
	Binary       string
	BinDir       string
	DataDir      string
	Version      string
	NPM          string
	URLTemplate  string
	URLTemplates map[string]string
	ArchiveEntry string
}

// UnsupportedPlatformError is returned before any download or execution when
// a resource has not declared an artifact route for the requested target.
type UnsupportedPlatformError struct {
	OS   string
	Arch string
}

func (e UnsupportedPlatformError) Error() string {
	return fmt.Sprintf("unsupported installer platform %s/%s", e.OS, e.Arch)
}

// StatusArgs is the common status argument contract formerly implemented by
// status-args.sh.
type StatusArgs struct {
	Format  string
	Verbose bool
	Fast    bool
}

// ParseStatusArgs accepts the standard status flags and ignores unknown flags
// for compatibility with the former shell parser.
func ParseStatusArgs(args []string) StatusArgs {
	parsed := StatusArgs{Format: "text"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			parsed.Format = "json"
		case "--format":
			if i+1 < len(args) {
				parsed.Format = args[i+1]
				i++
			}
		case "--verbose", "-v":
			parsed.Verbose = true
		case "--fast":
			parsed.Fast = true
		}
	}
	return parsed
}

// WarnIfShadowed reports whether PATH resolves a binary outside the managed
// directory. The caller decides how to display the warning.
func WarnIfShadowed(binary, managedBinDir string, lookPath func(string) (string, error)) string {
	resolved, err := lookPath(binary)
	if err != nil || resolved == "" {
		return fmt.Sprintf("%s may not be on PATH — add %s so %q resolves", managedBinDir, managedBinDir, binary)
	}
	managed, err := filepath.Abs(managedBinDir)
	if err != nil {
		return ""
	}
	path, err := filepath.Abs(resolved)
	if err != nil || path == managed || strings.HasPrefix(path, managed+string(filepath.Separator)) {
		return ""
	}
	return fmt.Sprintf("another %q at %s precedes %s on PATH", binary, resolved, managedBinDir)
}

// BlockingSystemInstall returns a root-owned host-tool path that a managed
// user installation must not overwrite. An empty result means no blocker.
func BlockingSystemInstall(binary, managedBinDir string, lookPath func(string) (string, error), stat func(string) (os.FileInfo, error)) (string, error) {
	path, err := lookPath(binary)
	if err != nil {
		return "", nil
	}
	managedBinDir, err = filepath.Abs(managedBinDir)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if resolved == managedBinDir || strings.HasPrefix(resolved, managedBinDir+string(filepath.Separator)) {
		return "", nil
	}
	info, err := stat(filepath.Dir(resolved))
	if err != nil {
		return "", err
	}
	if info.Mode().Perm()&tuning.PermOwnerWrite == 0 {
		return resolved, nil
	}
	return "", nil
}

// InstalledVersion probes an already-installed binary without invoking a
// shell. It is exported so resource CLIs can keep discovery separate from
// authentication or service readiness.
func InstalledVersion(ctx context.Context, path string) (string, error) {
	cmd := shell.NewCommandContext(ctx, path, "--version")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Install performs an idempotent user-prefix install without a shell.
func Install(ctx context.Context, spec Spec) error {
	if strings.TrimSpace(spec.Binary) == "" || strings.TrimSpace(spec.BinDir) == "" {
		return fmt.Errorf("binary and bin directory are required")
	}
	if err := os.MkdirAll(spec.BinDir, tuning.PermDir); err != nil {
		return fmt.Errorf("create install directory: %w", err)
	}
	if spec.DataDir != "" {
		if err := os.MkdirAll(spec.DataDir, tuning.PermPrivateDir); err != nil {
			return fmt.Errorf("create data directory: %w", err)
		}
	}
	if blocker, err := BlockingSystemInstall(spec.Binary, spec.BinDir, exec.LookPath, os.Stat); err != nil {
		return err
	} else if blocker != "" {
		return fmt.Errorf("refusing to overwrite root-owned %s at %s", spec.Binary, blocker)
	}
	target := filepath.Join(spec.BinDir, spec.Binary)
	if runtime.GOOS == string(hostreqspec.PlatformWindows) {
		target += ".exe"
	}
	if spec.Version != "" {
		if current, err := InstalledVersion(ctx, target); err == nil && strings.Contains(current, spec.Version) {
			return nil
		}
	}
	if spec.NPM != "" {
		prefix := filepath.Dir(spec.BinDir)
		packageRef := spec.NPM
		if spec.Version != "" {
			packageRef += "@" + strings.TrimPrefix(spec.Version, "v")
		}
		cmd := shell.NewCommandContext(ctx, "npm", "install", "--prefix", prefix, "--no-fund", "--no-audit", packageRef)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("install %s: %w", spec.NPM, err)
		}
		return nil
	}
	url := strings.TrimSpace(os.Getenv(strings.ToUpper(spec.Binary) + "_ARTIFACT_URL"))
	if url == "" {
		var err error
		url, err = ResolveURL(spec, runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return err
		}
	}
	return download(ctx, url, target, spec.ArchiveEntry)
}

// ResolveURL selects and expands a platform-specific artifact route without
// touching the network. It is used by platform-gate tests and install.
func ResolveURL(spec Spec, goos, arch string) (string, error) {
	template := spec.URLTemplate
	if len(spec.URLTemplates) > 0 {
		candidate := spec.URLTemplates[goos]
		if candidate == "" {
			return "", UnsupportedPlatformError{OS: goos, Arch: arch}
		}
		template = candidate
	}
	if strings.TrimSpace(template) == "" {
		return "", UnsupportedPlatformError{OS: goos, Arch: arch}
	}
	return expandURL(template, spec.Version, goos, arch), nil
}

func expandURL(template, version, goos, arch string) string {
	value := strings.ReplaceAll(template, "${version}", strings.TrimPrefix(version, "v"))
	value = strings.ReplaceAll(value, "${os}", goos)
	value = strings.ReplaceAll(value, "${arch}", arch)
	platform := goos
	if platform == string(hostreqspec.PlatformDarwin) {
		platform = "macos"
	}
	return strings.ReplaceAll(value, "${platform}", platform)
}

func download(ctx context.Context, url, target, archiveEntry string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := (&http.Client{Timeout: tuning.AgentInstallDownloadTimeout()}).Do(request)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("download %s returned HTTP %d", url, response.StatusCode)
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".agent-install-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, response.Body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	switch {
	case strings.HasSuffix(url, ".tar.gz"), strings.HasSuffix(url, ".tgz"):
		return extractTarGz(tmpName, target, archiveEntry)
	case strings.HasSuffix(url, ".zip"):
		return extractZip(tmpName, target, archiveEntry)
	default:
		return writeDownloadedFile(target, tmpName)
	}
}

func writeDownloadedFile(target, source string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return config.WriteOwnedFileAtomic(target, data, info.Mode().Perm())
}

func extractTarGz(path, target, wanted string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer reader.Close()
	archive := tar.NewReader(reader)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if archiveNameMatches(header.Name, wanted) {
			return writeExecutable(target, archive)
		}
	}
	return fmt.Errorf("archive %s does not contain %s", path, wanted)
}

func extractZip(path, target, wanted string) error {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer archive.Close()
	for _, entry := range archive.File {
		if !archiveNameMatches(entry.Name, wanted) {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return err
		}
		err = writeExecutable(target, reader)
		_ = reader.Close()
		return err
	}
	return fmt.Errorf("archive %s does not contain %s", path, wanted)
}

func archiveNameMatches(name, wanted string) bool {
	if wanted == "" {
		return true
	}
	name, wanted = filepath.Base(name), filepath.Base(wanted)
	return name == wanted || strings.TrimSuffix(name, ".exe") == strings.TrimSuffix(wanted, ".exe")
}

func writeExecutable(target string, source io.Reader) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".agent-binary-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, source); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(tuning.PermDir); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return writeDownloadedFile(target, tmpName)
}
