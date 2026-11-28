package agentaccess

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	browserclosure "github.com/vrooli/vrooli/packages/browser-closure"
)

// BrowserRuntimeEvidence contains only stable, non-path runtime identifiers.
// Source and channel describe the verified installation mechanism; they do
// not assert who supplied or signed the archive.
type BrowserRuntimeEvidence struct {
	BundleVersion  string `json:"bundle_version"`
	BrowserVersion string `json:"browser_version"`
	ClosureSHA256  string `json:"closure_sha256"`
	Driver         string `json:"driver"`
	DriverVersion  string `json:"driver_version"`
	Source         string `json:"source"`
	Channel        string `json:"channel"`
}

type runtimeClosureManifest struct {
	SchemaVersion           string                         `json:"schema_version"`
	Platform                string                         `json:"platform"`
	Identity                browserclosure.RuntimeIdentity `json:"runtime_identity"`
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

type runtimeInstallReceipt struct {
	SchemaVersion string                         `json:"schema_version"`
	ArchiveSHA256 string                         `json:"archive_sha256"`
	Identity      browserclosure.RuntimeIdentity `json:"runtime_identity"`
}

var exactRuntimeDriverVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// verifiedBrowserRuntimeEvidence reads the existing packaged closure contract.
// With no closure-related environment it returns nil so non-packaged BAS runs
// remain usable without making an identity claim. A partial or mismatched
// contract fails closed. Local paths are consumed only for verification and
// are never returned in evidence.
func verifiedBrowserRuntimeEvidence() (*BrowserRuntimeEvidence, error) {
	rootValue := strings.TrimSpace(os.Getenv("PLAYWRIGHT_BROWSERS_PATH"))
	execValue := strings.TrimSpace(os.Getenv("BROWSER_EXECUTABLE_PATH"))
	driverValue := strings.TrimSpace(os.Getenv("BAS_SIDECAR_DRIVER_DIR"))
	if rootValue == "" && execValue == "" && driverValue == "" {
		return nil, nil
	}
	if rootValue == "" || execValue == "" || driverValue == "" {
		return nil, errors.New("packaged browser runtime provenance is incomplete")
	}
	root, err := filepath.Abs(rootValue)
	if err != nil {
		return nil, fmt.Errorf("resolve packaged browser closure: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve packaged browser closure: %w", err)
	}
	closure, err := readRuntimeClosure(root)
	if err != nil {
		return nil, err
	}
	identity := closure.Identity
	if err := validateRuntimeClosure(closure); err != nil {
		return nil, err
	}
	if err := validateRuntimeInstallReceipt(root, identity); err != nil {
		return nil, err
	}
	if err := verifyRuntimeExecutable(root, closure.Executable, execValue, closure.ExecutableSHA); err != nil {
		return nil, err
	}
	bundleRoot := filepath.Dir(filepath.Dir(root))
	if err := verifyRuntimeDriver(bundleRoot, driverValue, identity.DriverVersion); err != nil {
		return nil, err
	}
	return &BrowserRuntimeEvidence{
		BundleVersion: identity.BundleVersion, BrowserVersion: identity.BrowserVersion,
		ClosureSHA256: identity.SHA256, Driver: identity.Driver, DriverVersion: identity.DriverVersion,
		Source: "offline-install-receipt", Channel: "offline-transfer",
	}, nil
}

func readRuntimeClosure(root string) (runtimeClosureManifest, error) {
	data, err := os.ReadFile(filepath.Join(root, "browser-closure.json"))
	if err != nil {
		return runtimeClosureManifest{}, errors.New("packaged browser closure manifest is unavailable")
	}
	var closure runtimeClosureManifest
	if err := json.Unmarshal(data, &closure); err != nil {
		return runtimeClosureManifest{}, errors.New("packaged browser closure manifest is invalid")
	}
	return closure, nil
}

func validateRuntimeClosure(closure runtimeClosureManifest) error {
	identity := closure.Identity
	if closure.SchemaVersion != "v3" || closure.Platform != runtimeBrowserPlatform() || closure.SupportMatrix != "v3" || !validRuntimeTargetLifecycle(closure.Platform, closure.TargetLifecycleStatus) ||
		closure.LayoutStatus != "source-fixture-tested" || closure.ArtifactStatus != "unqualified-no-provisioned-artifact" || closure.CleanHostStatus != "unqualified-no-receipt" ||
		closure.OnlineAcquisitionStatus != "unsupported-no-governed-browser-source" || closure.OfflineTransferStatus != "source-fixture-tested-linux-x64" ||
		closure.OfflineTransferScope != "Deterministic archive and install fixtures run on linux-x64 only." || closure.OfflineArtifactStatus != "unqualified-no-provisioned-artifact" ||
		closure.OfflineCleanHostStatus != "unqualified-no-receipt" || identity.Validate() != nil || identity.Engine != "chromium" || identity.Driver != "rebrowser-playwright-core" ||
		closure.Executable == "" || filepath.IsAbs(closure.Executable) || filepath.Clean(closure.Executable) != closure.Executable ||
		strings.HasPrefix(closure.Executable, ".."+string(filepath.Separator)) || closure.Executable == ".." {
		return errors.New("packaged browser closure identity is invalid")
	}
	return nil
}

func validateRuntimeInstallReceipt(root string, identity browserclosure.RuntimeIdentity) error {
	receiptBytes, err := os.ReadFile(filepath.Join(root, ".offline-install-receipt.json"))
	if err != nil {
		return errors.New("verified offline browser install receipt is required")
	}
	var receipt runtimeInstallReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil || receipt.SchemaVersion != "v3" || !validRuntimeSHA256(receipt.ArchiveSHA256) || receipt.Identity != identity {
		return errors.New("offline browser install receipt does not match the closure identity")
	}
	digest, err := browserclosure.ComputeRuntimeDigest(root)
	if err != nil || digest != identity.SHA256 {
		return errors.New("packaged browser closure digest does not match its identity")
	}
	return nil
}

func verifyRuntimeExecutable(root, relativePath, configuredPath, expectedSHA string) error {
	resolvedExecutable, err := filepath.EvalSymlinks(filepath.Join(root, relativePath))
	if err != nil {
		return errors.New("packaged browser executable is unavailable")
	}
	rel, err := filepath.Rel(root, resolvedExecutable)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("packaged browser executable escapes the verified closure")
	}
	info, err := os.Stat(resolvedExecutable)
	if err != nil || !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0) {
		return errors.New("packaged browser executable is not executable")
	}
	requestedExecutable, err := filepath.Abs(configuredPath)
	if err != nil {
		return errors.New("configured browser executable is invalid")
	}
	requestedExecutable, err = filepath.EvalSymlinks(requestedExecutable)
	if err != nil || requestedExecutable != resolvedExecutable {
		return errors.New("configured browser executable does not match the verified closure")
	}
	file, err := os.Open(resolvedExecutable)
	if err != nil {
		return errors.New("packaged browser executable is unavailable")
	}
	h := sha256.New()
	_, copyErr := io.Copy(h, file)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || hex.EncodeToString(h.Sum(nil)) != expectedSHA {
		return errors.New("packaged browser executable checksum does not match the closure")
	}
	return nil
}

func verifyRuntimeDriver(bundleRoot, driverValue, expectedVersion string) error {
	driverPath := driverValue
	if !filepath.IsAbs(driverPath) {
		driverPath = filepath.Join(bundleRoot, driverPath)
	}
	driverRoot, err := filepath.Abs(driverPath)
	if err != nil {
		return errors.New("packaged browser driver path is invalid")
	}
	driverRoot, err = filepath.EvalSymlinks(driverRoot)
	if err != nil {
		return errors.New("packaged browser driver is unavailable")
	}
	relativeDriver, err := filepath.Rel(bundleRoot, driverRoot)
	if err != nil || relativeDriver == ".." || strings.HasPrefix(relativeDriver, ".."+string(filepath.Separator)) {
		return errors.New("packaged browser driver escapes the bundle")
	}
	pkgData, err := os.ReadFile(filepath.Join(driverRoot, "package.json"))
	if err != nil {
		return errors.New("packaged browser driver manifest is unavailable")
	}
	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if json.Unmarshal(pkgData, &pkg) != nil {
		return errors.New("packaged browser driver manifest is invalid")
	}
	driverVersion := strings.TrimSpace(pkg.Dependencies["rebrowser-playwright"])
	if !exactRuntimeDriverVersion.MatchString(driverVersion) || driverVersion != expectedVersion {
		return errors.New("packaged browser driver version does not match the closure identity")
	}
	return nil
}

func runtimeBrowserPlatform() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	if runtime.GOOS == "windows" {
		return "windows-" + arch
	}
	return runtime.GOOS + "-" + arch
}

func validRuntimeTargetLifecycle(platform string, lifecycle map[string]string) bool {
	expected := "unqualified-no-node"
	switch platform {
	case "linux-x64":
		expected = "unqualified-no-receipt"
	case "linux-arm64", "darwin-x64", "darwin-arm64", "windows-x64":
	default:
		return false
	}
	want := map[string]string{"runtime_launch": expected, "browser_task": expected, "web_console_pane": expected, "update_rollback": expected, "uninstall": expected}
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

func validRuntimeSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
