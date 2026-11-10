// Package agentinstall contains shell-free host installation primitives shared
// by external-cli resources.
package agentinstall

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/cli-core/upstreamcheck"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	scopedPackageParts = 2
	// archiveEntryMaxBytes bounds one extracted executable; the largest agent
	// binary today is about 210 MiB.
	archiveEntryMaxBytes = 2 << 30
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
	// Acquisition is the manifest-owned artifact contract. When set, it takes
	// precedence over the legacy URL/NPM fields and is checksum-verified.
	Acquisition  *binaryfetch.Acquisition
	ResourceName string
	// UpstreamSourceKind and UpstreamSourceID identify the published release
	// channel `update` resolves. Empty defaults to npm and the package named by
	// the acquisition contract.
	UpstreamSourceKind string
	UpstreamSourceID   string
}

func acquisitionFromSourceManifest(name string) (*binaryfetch.Acquisition, error) {
	return acquisitionFromSourceManifestWithDeps(name, defaultPlatformDeps())
}

func acquisitionFromSourceManifestWithDeps(name string, platform platformDeps) (*binaryfetch.Acquisition, error) {
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}
	// RESOURCE_ROOT is injected by the resource controller for the exact
	// manifest being operated on. Prefer it over embedded/checkout roots so a
	// source-checkout install cannot accidentally resolve an older installed
	// resource manifest from a different working tree.
	roots := []string{os.Getenv("RESOURCE_ROOT"), os.Getenv("VROOLI_CLI_SOURCE_ROOT"), os.Getenv("VROOLI_RESOURCE_ROOT"), os.Getenv("VROOLI_ROOT"), filepath.Join(cliutil.ResolveRepoRoot(), "resources", name), "."}
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		for _, path := range []string{filepath.Join(root, "resource.json"), filepath.Join(filepath.Dir(root), "resource.json"), filepath.Join(root, name, "resource.json")} {
			manifest, err := platform.withDefaults().loadManifest(path)
			if err != nil {
				continue
			}
			if manifest.Name != name {
				continue
			}
			if manifest.Acquisition != nil {
				return manifest.Acquisition, nil
			}
		}
	}
	return nil, nil
}

// PinnedVersion reads the acquisition owner instead of maintaining a second
// resource-CLI constant. It does not select or update the installed executable.
func (spec Spec) PinnedVersion() (string, error) {
	acquisition := spec.Acquisition
	if acquisition == nil {
		var err error
		acquisition, err = acquisitionFromSourceManifest(spec.ResourceName)
		if err != nil {
			return "", err
		}
	}
	if acquisition == nil {
		return spec.Version, nil
	}
	target, err := acquisition.Resolve(binaryfetch.Facts{"os": runtime.GOOS, "arch": runtime.GOARCH})
	if err != nil {
		return "", err
	}
	return target.Version, nil
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
	return blockingSystemInstall(binary, managedBinDir, lookPath, stat, defaultPlatformDeps())
}

func blockingSystemInstall(binary, managedBinDir string, lookPath func(string) (string, error), stat func(string) (os.FileInfo, error), platform platformDeps) (string, error) {
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
	if info.Mode().Perm()&platform.withDefaults().permOwnerWrite == 0 {
		return resolved, nil
	}
	return "", nil
}

// InstalledVersion probes an already-installed binary without invoking a
// shell. It is exported so resource CLIs can keep discovery separate from
// authentication or service readiness.
func InstalledVersion(ctx context.Context, path string) (string, error) {
	return installedVersion(ctx, path, defaultPlatformDeps())
}

func installedVersion(ctx context.Context, path string, platform platformDeps) (string, error) {
	out, err := platform.withDefaults().commandCombined(ctx, path, "--version")
	return string(out), err
}

// Install performs an idempotent user-prefix install without a shell.
func Install(ctx context.Context, spec Spec) error {
	installErr := install(ctx, spec, defaultPlatformDeps())
	// Even a failed install can leave root-created files behind, so the hand
	// back runs regardless of the outcome.
	target := filepath.Join(spec.BinDir, spec.Binary)
	if runtime.GOOS == "windows" {
		target += ".exe"
	}
	return errors.Join(installErr, handBackToBinDirOwner(spec.BinDir, target, spec.DataDir))
}

func install(ctx context.Context, spec Spec, platform platformDeps) error {
	platform = platform.withDefaults()
	if strings.TrimSpace(spec.Binary) == "" || strings.TrimSpace(spec.BinDir) == "" {
		return fmt.Errorf("binary and bin directory are required")
	}
	if err := platform.mkdirAll(spec.BinDir, platform.permDir); err != nil {
		return fmt.Errorf("create install directory: %w", err)
	}
	if spec.DataDir != "" {
		if err := platform.mkdirAll(spec.DataDir, platform.permPrivate); err != nil {
			return fmt.Errorf("create data directory: %w", err)
		}
	}
	if blocker, err := blockingSystemInstall(spec.Binary, spec.BinDir, exec.LookPath, os.Stat, platform); err != nil {
		return err
	} else if blocker != "" {
		return fmt.Errorf("refusing to overwrite root-owned %s at %s", spec.Binary, blocker)
	}
	target := filepath.Join(spec.BinDir, spec.Binary)
	if platform.windows {
		target += ".exe"
	}
	var resolved *binaryfetch.AcquisitionTarget
	pin := spec.Version
	if spec.Acquisition != nil {
		acquired, err := spec.Acquisition.Resolve(binaryfetch.Facts{"os": platform.goos, "arch": platform.goarch})
		if err != nil {
			return err
		}
		if acquired.Unsupported != "" {
			return &UnsupportedPlatformError{OS: platform.goos, Arch: platform.goarch}
		}
		resolved, pin = &acquired, acquired.Version
	}
	if pin != "" {
		if current, err := installedVersion(ctx, target, platform); err == nil && satisfiesPin(current, pin) {
			return nil
		}
	}
	if resolved != nil {
		artifact := *resolved
		if artifact.Package != "" {
			packageURL := npmTarballURL(artifact.Package, artifact.Version)
			if err := binaryfetch.VerifyURL(ctx, packageURL, artifact.SHA256); err != nil {
				return fmt.Errorf("verify npm package %s: %w", artifact.Package, err)
			}
			return installNPMWithDeps(ctx, spec, artifact.Package+"@"+artifact.Version, platform)
		}
		_, err := binaryfetch.Fetch(ctx, binaryfetch.Target{Name: spec.Binary, URL: artifact.URL, SHA256: artifact.SHA256, Archive: artifact.Archive, BinPath: artifact.BinPath, Mode: artifact.Mode}, spec.BinDir, nil)
		return err
	}
	if spec.NPM != "" {
		packageRef := spec.NPM
		if spec.Version != "" {
			packageRef += "@" + strings.TrimPrefix(spec.Version, "v")
		}
		return installNPMWithDeps(ctx, spec, packageRef, platform)
	}
	url := strings.TrimSpace(os.Getenv(strings.ToUpper(spec.Binary) + "_ARTIFACT_URL"))
	if url == "" {
		var err error
		url, err = ResolveURL(spec, platform.goos, platform.goarch)
		if err != nil {
			return err
		}
	}
	return downloadWithDeps(ctx, url, target, spec.ArchiveEntry, platform)
}

// satisfiesPin reports whether an installed copy already meets a declared pin.
// A pin is the version a resource was reviewed against, not a ceiling: an
// installed release that is newer (typically after `resource update`) must not
// be reinstalled as the older pin, which the dependency approval no longer
// covers and which would be a downgrade.
func satisfiesPin(installedOutput, pin string) bool {
	if strings.Contains(installedOutput, pin) {
		return true
	}
	installed := upstreamcheck.ExtractSemver(installedOutput)
	want := upstreamcheck.ExtractSemver(pin)
	if installed == "" || want == "" {
		return false
	}
	status := upstreamcheck.Compare(installed, want)
	return status == upstreamcheck.StatusUpToDate || status == upstreamcheck.StatusAhead
}

// installNPM delegates all package mutation to the dependency-governance owner.
// A successful CLI exit alone is insufficient: blocked/failed JSON is refused.
func installNPM(ctx context.Context, spec Spec, packageRef string) error {
	return installNPMWithDeps(ctx, spec, packageRef, defaultPlatformDeps())
}

func installNPMWithDeps(ctx context.Context, spec Spec, packageRef string, platform platformDeps) error {
	platform = platform.withDefaults()
	if spec.ResourceName == "" {
		return fmt.Errorf("npm installation requires a resource owner")
	}
	home, err := platform.homeDir()
	if err != nil {
		return err
	}
	if filepath.Clean(spec.BinDir) != filepath.Join(home, ".local", "bin") {
		return fmt.Errorf("governed harness installation requires the owner user prefix")
	}
	output, commandErr := platform.commandOutput(ctx, "scenario-dependency-analyzer", "deps", "install", "npm/"+packageRef, "--scenario", spec.ResourceName, "--surface", "host-tool", "--exact", "--apply", "--json")
	var result governancev1.InstallDependencyResponse
	if err := protojson.Unmarshal(output, &result); err != nil {
		if commandErr != nil {
			return fmt.Errorf("governed install %s: %w", packageRef, commandErr)
		}
		return fmt.Errorf("invalid governed install receipt: %w", err)
	}
	prefix, err := npmInstallPrefix(home, spec.ResourceName, packageRef[strings.LastIndex(packageRef, "@")+1:], platform)
	if err != nil {
		return err
	}
	expectedManifest := filepath.Join(cliutil.ResolveRepoRoot(), "resources", spec.ResourceName, "resource.json")
	if result.Blocked || !result.Installed || result.DryRun || result.SourceUpdated || result.Verdict != "approved" || result.PackageManager != "npm" || result.ManifestPath != expectedManifest || result.Command != cliutil.DisplayCommand(NPMInstallArgs(prefix, packageRef)) {
		return fmt.Errorf("governed install %s did not return a matching approved runtime receipt: %s", packageRef, result.Message)
	}

	if commandErr != nil {
		return fmt.Errorf("governed install %s: %w", packageRef, commandErr)
	}
	return nil
}

// NPMInstallPrefix places each exact harness version in its resource-owned
// data tree, isolated from the previously selected prefix and other tools.
func NPMInstallPrefix(home, resource, version string) (string, error) {
	return npmInstallPrefix(home, resource, version, defaultPlatformDeps())
}

func npmInstallPrefix(home, resource, version string, platform platformDeps) (string, error) {
	if version == "" || filepath.Base(version) != version || version == "." || version == ".." {
		return "", fmt.Errorf("invalid harness version")
	}
	dataDir, err := platform.withDefaults().resourceDataDir(home, resource)
	if err != nil {
		return "", err
	}
	platform = platform.withDefaults()
	return filepath.Join(dataDir, "cli", version+"-"+platform.goos+"-"+platform.goarch), nil
}

// NPMInstallArgs is the exact script-disabled harness command shared with SDA.
// This only builds argv; package mutation remains inside the SDA gateway.
func NPMInstallArgs(prefix, packageRef string) []string {
	return []string{"npm", "install", "--prefix", prefix, "--ignore-scripts", "--no-audit", "--no-fund", "--save-exact", packageRef}
}

// LinkNPMBinary exposes the owner-qualified npm executable at the install location
// the resource declares. `npm install --prefix P` links executables under
// P/node_modules/.bin, never P/bin, so the install used to report success while
// the binary that the resource status, the PATH, and a node's capability probe
// look for never appeared — a remote "Install" that changed nothing.
//
// Only a missing path or an existing symlink is replaced; a real file at the
// target belongs to someone else and is refused, not overwritten.
func LinkNPMBinary(source, binary, binDir string) error {
	return linkNPMBinaryWithDeps(source, binary, binDir, defaultPlatformDeps())
}

func linkNPMBinaryWithDeps(source, binary, binDir string, platform platformDeps) error {
	platform = platform.withDefaults()
	if platform.windows {
		return nil
	}
	if err := platform.mkdirAll(binDir, platform.permDir); err != nil {
		return err
	}
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("npm finished but produced no %s executable at %s: %w", binary, source, err)
	}
	target := filepath.Join(binDir, binary)
	if info, err := os.Lstat(target); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("refusing to replace %s: it is a file this installer did not create", target)
		}
		if current, readErr := os.Readlink(target); readErr == nil && current == source {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect %s: %w", target, err)
	}
	return publishLinkWithDeps(source, target, platform)
}

// publishLink atomically points target at source, so a launch racing this sees
// the old link or the new one and never a missing command.
func publishLink(source, target string) error {
	return publishLinkWithDeps(source, target, defaultPlatformDeps())
}

func publishLinkWithDeps(source, target string, platform platformDeps) error {
	temporary, err := os.CreateTemp(filepath.Dir(target), ".harness-link-")
	if err != nil {
		return err
	}
	name := temporary.Name()
	if err := temporary.Close(); err != nil {
		return err
	}
	defer os.Remove(name)
	if err := os.Remove(name); err != nil {
		return err
	}
	if err := os.Symlink(source, name); err != nil {
		return err
	}
	if err := platform.withDefaults().commitOwnedPath(name, target); err != nil {
		return fmt.Errorf("publish harness link: %w", err)
	}

	return nil
}

func npmTarballURL(packageName, version string) string {
	name := strings.TrimSpace(packageName)
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", scopedPackageParts)
		if len(parts) == scopedPackageParts {
			return "https://registry.npmjs.org/" + name + "/-" + "/" + parts[1] + "-" + version + ".tgz"
		}
	}
	return "https://registry.npmjs.org/" + name + "/-/" + name + "-" + version + ".tgz"
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
	if platform == "darwin" {
		platform = "macos"
	}
	return strings.ReplaceAll(value, "${platform}", platform)
}

func download(ctx context.Context, url, target, archiveEntry string) error {
	return downloadWithDeps(ctx, url, target, archiveEntry, defaultPlatformDeps())
}

func downloadWithDeps(ctx context.Context, url, target, archiveEntry string, platform platformDeps) error {
	platform = platform.withDefaults()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := (&http.Client{Timeout: platform.downloadTimeout}).Do(request)
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
		return extractTarGzWithDeps(tmpName, target, archiveEntry, platform)
	case strings.HasSuffix(url, ".zip"):
		return extractZipWithDeps(tmpName, target, archiveEntry, platform)
	default:
		return writeDownloadedFileWithDeps(target, tmpName, platform)
	}
}

func writeDownloadedFile(target, source string) error {
	return writeDownloadedFileWithDeps(target, source, defaultPlatformDeps())
}

func writeDownloadedFileWithDeps(target, source string, platform platformDeps) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return platform.withDefaults().writeOwnedFile(target, data, info.Mode().Perm())
}

func extractTarGz(path, target, wanted string) error {
	return extractTarGzWithDeps(path, target, wanted, defaultPlatformDeps())
}

func extractTarGzWithDeps(path, target, wanted string, platform platformDeps) error {
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
		// Only a regular file can be the executable; a link or directory of the
		// same name would extract as an empty file.
		if header.Typeflag != tar.TypeReg || !archiveNameMatches(header.Name, wanted) {
			continue
		}
		if header.Size > archiveEntryMaxBytes {
			return fmt.Errorf("archive entry %s is %d bytes, over the %d limit", header.Name, header.Size, archiveEntryMaxBytes)
		}
		return writeExecutableWithDeps(target, io.LimitReader(archive, header.Size), platform)
	}
	return fmt.Errorf("archive %s does not contain %s", path, wanted)
}

func extractZip(path, target, wanted string) error {
	return extractZipWithDeps(path, target, wanted, defaultPlatformDeps())
}

func extractZipWithDeps(path, target, wanted string, platform platformDeps) error {
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
		err = writeExecutableWithDeps(target, reader, platform)
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
	return writeExecutableWithDeps(target, source, defaultPlatformDeps())
}

func writeExecutableWithDeps(target string, source io.Reader, platform platformDeps) error {
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
	if err := tmp.Chmod(platform.withDefaults().permDir); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return writeDownloadedFileWithDeps(target, tmpName, platform)
}
