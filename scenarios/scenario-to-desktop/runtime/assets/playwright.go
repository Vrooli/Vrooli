package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	browserclosure "github.com/vrooli/vrooli/packages/browser-closure"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/ports"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/telemetry"
)

type browserClosure struct {
	SchemaVersion           string                         `json:"schema_version"`
	Platform                string                         `json:"platform"`
	HostPlatform            string                         `json:"playwright_host_platform"`
	Identity                browserclosure.RuntimeIdentity `json:"runtime_identity"`
	HeadlessRev             string                         `json:"headless_shell_revision"`
	FFmpegRev               string                         `json:"ffmpeg_revision"`
	Executable              string                         `json:"executable_path"`
	ExecutableSHA           string                         `json:"executable_sha256"`
	SupportMatrix           string                         `json:"support_matrix_schema"`
	LayoutStatus            string                         `json:"bundle_layout_status"`
	ArtifactStatus          string                         `json:"browser_artifact_status"`
	CleanHostStatus         string                         `json:"clean_host_status"`
	OnlineAcquisitionStatus string                         `json:"online_acquisition_status"`
	OfflineTransferStatus   string                         `json:"offline_transfer_status"`
	OfflineTransferScope    string                         `json:"offline_transfer_scope"`
	OfflineArtifactStatus   string                         `json:"offline_browser_artifact_status"`
	OfflineCleanHostStatus  string                         `json:"offline_clean_host_status"`
	TargetLifecycleStatus   map[string]string              `json:"target_lifecycle_status"`
}

type offlineBrowserInstallReceipt struct {
	SchemaVersion string                         `json:"schema_version"`
	ArchiveSHA256 string                         `json:"archive_sha256"`
	Identity      browserclosure.RuntimeIdentity `json:"runtime_identity"`
}

var exactDriverVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

type PlaywrightConfig struct {
	BundlePath string
	FS         infra.FileSystem
	EnvReader  infra.EnvReader
	Ports      ports.Allocator
	Telemetry  telemetry.Recorder
}

// ApplyPlaywrightConventions binds packaged Playwright services to the exact
// executable named by their staged closure manifest. It never consults a user
// cache, system browser, or Electron fallback.
func ApplyPlaywrightConventions(cfg PlaywrightConfig, svc manifest.Service, env map[string]string) error {
	if !ServiceUsesPlaywright(svc) {
		return nil
	}
	if _, ok := env["PLAYWRIGHT_DRIVER_PORT"]; !ok {
		if port, err := cfg.Ports.Resolve(svc.ID, "http"); err == nil {
			env["PLAYWRIGHT_DRIVER_PORT"] = strconv.Itoa(port)
		}
	}
	if _, ok := env["PLAYWRIGHT_DRIVER_URL"]; !ok {
		if port, err := cfg.Ports.Resolve(svc.ID, "http"); err == nil {
			env["PLAYWRIGHT_DRIVER_URL"] = fmt.Sprintf("http://127.0.0.1:%d", port)
		}
	}
	if _, ok := env["ENGINE"]; !ok {
		env["ENGINE"] = "playwright"
	}

	browserDir := strings.TrimSpace(env["PLAYWRIGHT_BROWSERS_PATH"])
	if browserDir == "" {
		// A standalone explicit executable remains supported for existing non-bundle callers.
		// It is never filled from ambient environment or searched for.
		return validateExplicitExecutable(cfg, svc, env)
	}
	browserDir = manifest.ResolvePath(cfg.BundlePath, browserDir)
	info, err := cfg.FS.Stat(browserDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("Playwright browser closure directory missing for %s: %s", svc.ID, browserDir)
	}
	manifestPath := filepath.Join(browserDir, "browser-closure.json")
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("Playwright browser closure manifest missing for %s: %s", svc.ID, manifestPath)
	}
	var closure browserClosure
	if err := json.Unmarshal(contents, &closure); err != nil {
		return fmt.Errorf("invalid Playwright browser closure manifest for %s: %w", svc.ID, err)
	}
	expectedDriverVersion, err := packagedDriverVersion(cfg.BundlePath, svc)
	if err != nil || closure.SchemaVersion != "v3" || closure.Platform != browserPlatform() || !compatiblePlaywrightHost(closure.Platform, closure.HostPlatform) || closure.Identity.Validate() != nil || closure.Identity.Engine != "chromium" || closure.Identity.Driver != "rebrowser-playwright-core" || closure.Identity.DriverVersion != expectedDriverVersion || closure.HeadlessRev != closure.Identity.BrowserVersion || closure.FFmpegRev == "" || closure.SupportMatrix != "v3" || !validTargetLifecycleStatus(closure.Platform, closure.TargetLifecycleStatus) || closure.LayoutStatus != "source-fixture-tested" || closure.ArtifactStatus != "unqualified-no-provisioned-artifact" || closure.CleanHostStatus != "unqualified-no-receipt" || closure.OnlineAcquisitionStatus != "unsupported-no-governed-browser-source" || closure.OfflineTransferStatus != "source-fixture-tested-linux-x64" || closure.OfflineTransferScope != "Deterministic archive and install fixtures run on linux-x64 only." || closure.OfflineArtifactStatus != "unqualified-no-provisioned-artifact" || closure.OfflineCleanHostStatus != "unqualified-no-receipt" {
		return fmt.Errorf("Playwright browser closure metadata mismatch for %s", svc.ID)
	}
	if err := validateOfflineBrowserInstallReceipt(browserDir, closure.Identity); err != nil {
		return fmt.Errorf("Playwright offline install receipt mismatch for %s: %w", svc.ID, err)
	}
	if closure.Executable == "" || filepath.IsAbs(closure.Executable) || filepath.Clean(closure.Executable) != closure.Executable || strings.HasPrefix(closure.Executable, ".."+string(filepath.Separator)) || closure.Executable == ".." {
		return fmt.Errorf("Playwright closure executable path is not a safe relative path for %s", svc.ID)
	}
	root, err := filepath.EvalSymlinks(browserDir)
	if err != nil {
		return fmt.Errorf("resolve Playwright closure root for %s: %w", svc.ID, err)
	}
	executable := filepath.Join(browserDir, closure.Executable)
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return fmt.Errorf("Playwright closure executable missing for %s: %s", svc.ID, executable)
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("Playwright closure executable escapes package for %s", svc.ID)
	}
	st, err := os.Stat(resolved)
	if err != nil || !st.Mode().IsRegular() || (runtime.GOOS != "windows" && st.Mode().Perm()&0o111 == 0) {
		return fmt.Errorf("Playwright closure executable is not executable for %s: %s", svc.ID, resolved)
	}
	digest, err := browserclosure.ComputeRuntimeDigest(browserDir)
	if err != nil || digest != closure.Identity.SHA256 {
		return fmt.Errorf("Playwright browser runtime bundle digest mismatch for %s", svc.ID)
	}
	f, err := os.Open(resolved)
	if err != nil {
		return fmt.Errorf("open Playwright executable for checksum: %w", err)
	}
	h := sha256.New()
	_, copyErr := io.Copy(h, f)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if hex.EncodeToString(h.Sum(nil)) != closure.ExecutableSHA {
		return fmt.Errorf("Playwright closure executable checksum mismatch for %s", svc.ID)
	}
	if supplied := strings.TrimSpace(env["BROWSER_EXECUTABLE_PATH"]); supplied != "" {
		if manifest.ResolvePath(cfg.BundlePath, supplied) != resolved {
			return fmt.Errorf("BROWSER_EXECUTABLE_PATH conflicts with packaged closure for %s", svc.ID)
		}
	}
	env["PLAYWRIGHT_BROWSERS_PATH"] = browserDir
	env["BROWSER_EXECUTABLE_PATH"] = resolved
	return nil
}

func validTargetLifecycleStatus(platform string, lifecycle map[string]string) bool {
	expected := "unqualified-no-node"
	switch platform {
	case "linux-x64":
		expected = "unqualified-no-receipt"
	case "linux-arm64", "darwin-x64", "darwin-arm64", "windows-x64":
	default:
		return false
	}
	want := map[string]string{
		"runtime_launch": expected, "browser_task": expected, "web_console_pane": expected,
		"update_rollback": expected, "uninstall": expected,
	}
	if len(lifecycle) != len(want) {
		return false
	}
	for name, status := range want {
		if lifecycle[name] != status {
			return false
		}
	}
	return true
}

func packagedDriverVersion(bundlePath string, svc manifest.Service) (string, error) {
	driverDir := strings.TrimSpace(svc.Env["BAS_SIDECAR_DRIVER_DIR"])
	if driverDir == "" {
		return "", fmt.Errorf("service must declare its packaged BAS driver directory")
	}
	driverRoot := manifest.ResolvePath(bundlePath, driverDir)
	root, err := filepath.EvalSymlinks(bundlePath)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(driverRoot)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("packaged BAS driver directory escapes bundle")
	}
	data, err := os.ReadFile(filepath.Join(resolved, "package.json"))
	if err != nil {
		return "", err
	}
	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", err
	}
	version := strings.TrimSpace(pkg.Dependencies["rebrowser-playwright"])
	if !exactDriverVersion.MatchString(version) {
		return "", fmt.Errorf("packaged BAS driver must pin rebrowser-playwright to an exact version")
	}
	return version, nil
}

func validateOfflineBrowserInstallReceipt(browserDir string, identity browserclosure.RuntimeIdentity) error {
	path := filepath.Join(browserDir, ".offline-install-receipt.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("verified browser install receipt is required: %w", err)
	}
	var receipt offlineBrowserInstallReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return err
	}
	if receipt.SchemaVersion != "v3" || !validBrowserSHA256(receipt.ArchiveSHA256) || receipt.Identity.Validate() != nil || receipt.Identity != identity {
		return fmt.Errorf("receipt does not attest the exact staged runtime identity")
	}
	return nil
}

func validBrowserSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func browserPlatform() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	if runtime.GOOS == "windows" {
		return "windows-" + arch
	}
	return runtime.GOOS + "-" + arch
}

func compatiblePlaywrightHost(platform, host string) bool {
	switch {
	case platform == "windows-x64":
		return host == "win64"
	case platform == "darwin-x64":
		return strings.HasPrefix(host, "mac") && !strings.HasSuffix(host, "-arm64")
	case platform == "darwin-arm64":
		return strings.HasPrefix(host, "mac") && strings.HasSuffix(host, "-arm64")
	case strings.HasPrefix(platform, "linux-"):
		return strings.HasSuffix(host, "-"+strings.TrimPrefix(platform, "linux-"))
	default:
		return false
	}
}

func validateExplicitExecutable(cfg PlaywrightConfig, svc manifest.Service, env map[string]string) error {
	p := strings.TrimSpace(env["BROWSER_EXECUTABLE_PATH"])
	if p == "" {
		return fmt.Errorf("Playwright service %s requires PLAYWRIGHT_BROWSERS_PATH or an explicit BROWSER_EXECUTABLE_PATH", svc.ID)
	}
	p = manifest.ResolvePath(cfg.BundlePath, p)
	info, err := cfg.FS.Stat(p)
	if err != nil || !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0) {
		return fmt.Errorf("explicit browser executable missing for %s: %s", svc.ID, p)
	}
	env["BROWSER_EXECUTABLE_PATH"] = p
	return nil
}

func ServiceUsesPlaywright(svc manifest.Service) bool {
	if strings.Contains(strings.ToLower(svc.ID), "playwright") {
		return true
	}
	for k := range svc.Env {
		if k == "BROWSER_EXECUTABLE_PATH" || strings.HasPrefix(k, "PLAYWRIGHT_") {
			return true
		}
	}
	return false
}
