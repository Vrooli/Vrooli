// Package assets owns signed, bounded asset provisioning through project setup.
// Plans are scenario-owned declarations; execution and capacity policy stay in
// the canonical setup layer so image acquisition never becomes a private host
// installer.
package assets

import (
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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const planName = "setup-assets.json"
const metadataGrowthAllowance = uint64(64 * 1024)
const assetPlanVersion = 1
const aptPackageManager = "apt"
const imageManifestSchemaVersion = 1
const imageReceiptVersion = 1

type Plan struct {
	Version            int        `json:"version"`
	OutputDirectory    string     `json:"output_directory"`
	ImageManifest      string     `json:"image_manifest"`
	ReceiptPath        string     `json:"receipt_path"`
	ChecksumURL        string     `json:"checksum_url"`
	SignatureURL       string     `json:"signature_url"`
	KeyringPath        string     `json:"keyring_path"`
	MinimumFreeBytes   uint64     `json:"minimum_free_bytes"`
	MaximumGrowthBytes uint64     `json:"maximum_growth_bytes"`
	MaximumImageBytes  uint64     `json:"maximum_image_bytes"`
	HostTools          []ToolPlan `json:"host_tools"`
	Images             []Image    `json:"images"`
}

type ToolPlan struct {
	Name           string   `json:"name"`
	PackageManager string   `json:"package_manager"`
	Package        string   `json:"package"`
	Command        []string `json:"command"`
}

type Image struct {
	FileName  string `json:"file_name"`
	SHA256    string `json:"sha256"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	SourceURL string `json:"source_url"`
}

type ImageRecord struct {
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	SourceURL  string `json:"source_url"`
	RecordedAt string `json:"recorded_at"`
}

type ImageManifest struct {
	SchemaVersion int           `json:"schema_version"`
	Note          string        `json:"note,omitempty"`
	Images        []ImageRecord `json:"images"`
}

type Receipt struct {
	Version                int           `json:"version"`
	Plan                   string        `json:"plan"`
	RecordedAt             string        `json:"recorded_at"`
	ChecksumSignature      string        `json:"checksum_signature"`
	ChecksumURL            string        `json:"checksum_url"`
	ChecksumFileSHA256     string        `json:"checksum_file_sha256"`
	SignatureFileSHA256    string        `json:"signature_file_sha256"`
	Images                 []ImageRecord `json:"images"`
	HostTools              []ToolPlan    `json:"host_tools"`
	DestinationFilesystem  string        `json:"destination_filesystem"`
	DestinationDevice      uint64        `json:"destination_device_id"`
	StagingDevice          uint64        `json:"staging_device_id"`
	FreeBytesBefore        uint64        `json:"free_bytes_before"`
	FreeBytesAfterImages   uint64        `json:"free_bytes_after_images_before_receipt"`
	StagingFreeBytesBefore uint64        `json:"staging_free_bytes_before"`
	HostToolGrowthBytes    uint64        `json:"host_tool_growth_bytes"`
	ImageStagingBytes      uint64        `json:"image_staging_bytes"`
	ImageFinalBytes        uint64        `json:"image_final_bytes"`
	PeakGrowthBytes        uint64        `json:"peak_growth_bytes"`
	ReserveBytes           uint64        `json:"reserve_bytes"`
	GrowthLimitBytes       uint64        `json:"growth_limit_bytes"`
}

type Inspection struct {
	PlanPath             string      `json:"plan_path"`
	HostTools            []ToolPlan  `json:"host_tools"`
	Destination          string      `json:"destination"`
	Images               []ImagePlan `json:"images"`
	ChecksumSignature    string      `json:"checksum_signature"`
	Keyring              string      `json:"keyring"`
	DestinationFreeBytes uint64      `json:"destination_free_bytes"`
	GrowthHeadroomBytes  uint64      `json:"growth_headroom_above_reserve_bytes"`
	DestinationDevice    uint64      `json:"destination_device_id"`
	StagingFreeBytes     uint64      `json:"staging_free_bytes"`
	StagingDevice        uint64      `json:"staging_device_id"`
	SameFilesystem       bool        `json:"same_filesystem_for_staging_and_output"`
	PeakGrowthBytes      uint64      `json:"peak_growth_bytes"`
	FinalGrowthBytes     uint64      `json:"final_growth_bytes"`
	ReserveBytes         uint64      `json:"reserve_bytes"`
	GrowthLimitBytes     uint64      `json:"growth_limit_bytes"`
	Effects              []string    `json:"effects"`
	Stops                []string    `json:"stop_conditions"`
	AlreadyReady         bool        `json:"already_ready"`
}

type ImagePlan struct {
	FileName    string `json:"file_name"`
	SourceURL   string `json:"source_url"`
	SHA256      string `json:"sha256"`
	SizeBytes   uint64 `json:"size_bytes"`
	Destination string `json:"destination"`
	State       string `json:"state"`
}

func LoadScenarioPlan(root, scenarioName string) (Plan, string, bool, error) {
	path := filepath.Join(root, "scenarios", scenarioName, ".vrooli", planName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Plan{}, path, false, nil
	}
	if err != nil {
		return Plan{}, path, false, fmt.Errorf("read asset plan %s: %w", path, err)
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return Plan{}, path, true, fmt.Errorf("decode asset plan %s: %w", path, err)
	}
	if err := plan.Validate(); err != nil {
		return Plan{}, path, true, fmt.Errorf("validate asset plan %s: %w", path, err)
	}
	return plan, path, true, nil
}

func (p Plan) Validate() error {
	if p.Version != assetPlanVersion || len(p.Images) == 0 {
		return fmt.Errorf("asset plan version 1 and at least one image are required")
	}
	if !filepath.IsAbs(p.OutputDirectory) || !filepath.IsAbs(p.ReceiptPath) || strings.TrimSpace(p.ImageManifest) == "" {
		return fmt.Errorf("asset output directory and receipt must be absolute; image manifest must be repo-relative")
	}
	if filepath.Clean(p.OutputDirectory) != "/var/lib/vm" || filepath.Clean(p.ReceiptPath) != "/var/lib/vm/qemu-image-setup-receipt.json" || filepath.Clean(p.ImageManifest) != filepath.Join("scenarios", "scenario-to-cloud", "certification", "lanes", "qemu-images.json") {
		return fmt.Errorf("A2 image destinations must remain /var/lib/vm and the scenario-to-cloud QEMU manifest")
	}
	if p.MinimumFreeBytes == 0 || p.MaximumGrowthBytes == 0 || p.MaximumImageBytes == 0 {
		return fmt.Errorf("asset free-space, growth, and per-image limits are required")
	}
	if p.MinimumFreeBytes < 120_000_000_000 || p.MaximumGrowthBytes > 25_000_000_000 || p.MaximumImageBytes > 734_003_200 {
		return fmt.Errorf("A2 asset plan cannot lower the 120 GB reserve or widen the 25 GB / 700 MiB growth limits")
	}
	if len(p.HostTools) != 3 {
		return fmt.Errorf("A2 plan must contain exactly the three approved host tools")
	}
	toolNames := map[string]bool{}
	for _, tool := range p.HostTools {
		if tool.Name == "" || tool.PackageManager != aptPackageManager || tool.Package == "" || toolNames[tool.Name] {
			return fmt.Errorf("A2 host tools must be unique, exact apt package plans")
		}
		toolNames[tool.Name] = true
		if strings.Join(tool.Command, " ") != "apt-get install -y "+tool.Package {
			return fmt.Errorf("host tool %s command must be apt-get install -y %s", tool.Name, tool.Package)
		}
	}
	for name, pkg := range map[string]string{"qemu": "qemu-system-x86", "qemu-arm": "qemu-system-arm", "cloud-localds": "cloud-image-utils"} {
		found := false
		for _, tool := range p.HostTools {
			if tool.Name == name && tool.Package == pkg {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("A2 host tool plan is missing exact owner %s → %s", name, pkg)
		}
	}
	if !strings.HasPrefix(p.ChecksumURL, "https://cloud-images.ubuntu.com/") || !strings.HasPrefix(p.SignatureURL, "https://cloud-images.ubuntu.com/") {
		return fmt.Errorf("checksum metadata must use Canonical's HTTPS cloud-images.ubuntu.com origin")
	}
	if !strings.HasSuffix(p.ChecksumURL, "/SHA256SUMS") || p.SignatureURL != p.ChecksumURL+".gpg" {
		return fmt.Errorf("checksum URL must name SHA256SUMS and its matching .gpg signature")
	}
	const release = "https://cloud-images.ubuntu.com/releases/noble/release-20261017/"
	if p.ChecksumURL != release+"SHA256SUMS" || p.KeyringPath != "/usr/share/keyrings/ubuntu-cloudimage-keyring.gpg" {
		return fmt.Errorf("A2 checksum release and Canonical keyring must match the approved packet")
	}
	if !filepath.IsAbs(p.KeyringPath) {
		return fmt.Errorf("Canonical keyring path must be absolute")
	}
	if err := rejectSymlinkComponents(p.OutputDirectory); err != nil {
		return fmt.Errorf("unsafe image output path: %w", err)
	}
	seen := map[string]bool{}
	for _, image := range p.Images {
		if filepath.Base(image.FileName) != image.FileName || image.FileName == "." || image.FileName == ".." || seen[image.FileName] {
			return fmt.Errorf("image filenames must be unique basenames")
		}
		seen[image.FileName] = true
		if len(image.SHA256) != sha256.Size*2 {
			return fmt.Errorf("image %s must declare a SHA-256 digest", image.FileName)
		}
		if _, err := hex.DecodeString(image.SHA256); err != nil {
			return fmt.Errorf("image %s has invalid SHA-256: %w", image.FileName, err)
		}
		parsed, err := url.Parse(image.SourceURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host != "cloud-images.ubuntu.com" || filepath.Base(parsed.Path) != image.FileName {
			return fmt.Errorf("image %s must use its exact Canonical HTTPS release URL", image.FileName)
		}
		checksum, _ := url.Parse(p.ChecksumURL)
		if filepath.Dir(parsed.Path) != filepath.Dir(checksum.Path) {
			return fmt.Errorf("image %s and signed SHA256SUMS must come from the same pinned release directory", image.FileName)
		}
		if image.SourceURL != release+image.FileName {
			return fmt.Errorf("image %s must use the packet-pinned Canonical release URL", image.FileName)
		}
	}
	if len(p.Images) != 2 {
		return fmt.Errorf("A2 image plan must contain exactly the two approved pinned images")
	}
	pinnedImages := map[string]Image{
		"ubuntu-24.04-server-cloudimg-amd64.img": {FileName: "ubuntu-24.04-server-cloudimg-amd64.img", SHA256: "04e8f2461a16a44e78a370284471f29616cdd9baf1893df25807b71f985b6c94", OS: "ubuntu-24.04", Arch: "amd64"},
		"ubuntu-24.04-server-cloudimg-arm64.img": {FileName: "ubuntu-24.04-server-cloudimg-arm64.img", SHA256: "50e24e1fb79e89116e939d90aa7738f7af48c442fa1bc2510131e0b39c50348e", OS: "ubuntu-24.04", Arch: "arm64"},
	}
	for _, image := range p.Images {
		pinned, ok := pinnedImages[image.FileName]
		if !ok || !strings.EqualFold(image.SHA256, pinned.SHA256) || image.OS != pinned.OS || image.Arch != pinned.Arch {
			return fmt.Errorf("A2 image %s does not match the approved Ubuntu 24.04 digest and architecture", image.FileName)
		}
	}
	return nil
}

func InspectScenarioPlan(root, scenarioName string) (Inspection, bool, error) {
	plan, path, found, err := LoadScenarioPlan(root, scenarioName)
	if err != nil || !found {
		return Inspection{}, found, err
	}
	inspection, err := Inspect(context.Background(), root, plan, path)
	return inspection, true, err
}

func PreviewScenarioPlan(root, scenarioName string, stdout io.Writer) error {
	inspection, found, err := InspectScenarioPlan(root, scenarioName)
	if err != nil || !found {
		return err
	}
	data, _ := json.MarshalIndent(inspection, "", "  ")
	_, _ = fmt.Fprintf(stdout, "[INFO]    Scenario asset plan (review only):\n%s\n", data)
	for _, stop := range inspection.Stops {
		return fmt.Errorf("asset plan stop condition: %s", stop)
	}
	return nil
}

func ApplyScenarioPlan(root, scenarioName, sudoMode string, baseline Inspection, stdout, stderr io.Writer) error {
	plan, path, found, err := LoadScenarioPlan(root, scenarioName)
	if err != nil || !found {
		return err
	}
	release, err := artifactlock.Acquire("scenario-setup-assets:" + scenarioName)
	if err != nil {
		return fmt.Errorf("acquire scenario asset setup lock: %w", err)
	}
	defer release()
	inspection, err := Inspect(context.Background(), root, plan, path)
	if err != nil {
		return err
	}
	if baseline.DestinationDevice != inspection.DestinationDevice {
		return fmt.Errorf("output filesystem changed after setup review; refusing to provision images")
	}
	if baseline.StagingDevice != inspection.StagingDevice {
		return fmt.Errorf("staging filesystem changed after setup review; refusing to provision images")
	}
	packageGrowth, currentDestinationFree, err := measurePackageGrowth(plan, baseline, inspection)
	if err != nil {
		return err
	}
	if packageGrowth > plan.MaximumGrowthBytes {
		return fmt.Errorf("stop: A2 host-tool installation consumed %d bytes; cap is %d", packageGrowth, plan.MaximumGrowthBytes)
	}
	if currentDestinationFree < plan.MinimumFreeBytes {
		return fmt.Errorf("stop: output filesystem has %d free bytes, below the %d-byte reserve", currentDestinationFree, plan.MinimumFreeBytes)
	}
	if inspection.AlreadyReady {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Canonical image manifest and pinned images are already verified; no asset changes are needed.")
		return nil
	}
	if len(inspection.Stops) > 0 {
		return fmt.Errorf("asset plan stop condition: %s", strings.Join(inspection.Stops, "; "))
	}
	for _, reviewed := range baseline.Images {
		for _, current := range inspection.Images {
			if reviewed.FileName == current.FileName && reviewed.SizeBytes != current.SizeBytes {
				return fmt.Errorf("image %s changed size after dry-run review: %d -> %d", reviewed.FileName, reviewed.SizeBytes, current.SizeBytes)
			}
		}
	}
	return apply(context.Background(), root, plan, path, sudoMode, baseline, inspection, packageGrowth, stdout, stderr)
}

func measurePackageGrowth(plan Plan, baseline, current Inspection) (uint64, uint64, error) {
	destBase, err := nearestExistingDirectory(plan.OutputDirectory)
	if err != nil {
		return 0, 0, err
	}
	stageBase := os.TempDir()
	destFree, err := freeBytes(destBase)
	if err != nil {
		return 0, 0, err
	}
	stageFree, err := freeBytes(stageBase)
	if err != nil {
		return 0, 0, err
	}
	destDevice, err := filesystemDevice(destBase)
	if err != nil {
		return 0, 0, err
	}
	stageDevice, err := filesystemDevice(stageBase)
	if err != nil {
		return 0, 0, err
	}
	if destDevice != current.DestinationDevice || stageDevice != current.StagingDevice {
		return 0, 0, fmt.Errorf("filesystem identity changed during A2 setup")
	}
	growth := positiveDelta(baseline.DestinationFreeBytes, destFree)
	if destDevice != stageDevice {
		growth += positiveDelta(baseline.StagingFreeBytes, stageFree)
	}
	return growth, destFree, nil
}

func positiveDelta(before, after uint64) uint64 {
	if before <= after {
		return 0
	}
	return before - after
}

func Inspect(ctx context.Context, root string, plan Plan, planPath string) (Inspection, error) {
	inspection := Inspection{
		PlanPath: planPath, HostTools: append([]ToolPlan(nil), plan.HostTools...), Destination: plan.OutputDirectory,
		ChecksumSignature: "verify SHA256SUMS.gpg with gpgv and Canonical Ubuntu cloud-image keyring",
		Keyring:           plan.KeyringPath, ReserveBytes: plan.MinimumFreeBytes,
		GrowthLimitBytes: plan.MaximumGrowthBytes,
		Effects:          []string{"Install only missing declared images under " + plan.OutputDirectory + " with mode 0444", "Write the verified repository image manifest " + filepath.Join(root, plan.ImageManifest), "Write durable receipt " + plan.ReceiptPath},
		Stops:            []string{},
	}
	if _, err := exec.LookPath("gpgv"); err != nil {
		inspection.Stops = append(inspection.Stops, "gpgv is unavailable")
	}
	if info, err := os.Stat(plan.KeyringPath); err != nil || info.IsDir() {
		inspection.Stops = append(inspection.Stops, "Canonical Ubuntu cloud-image keyring is unavailable at "+plan.KeyringPath)
	}
	stageRoot := os.TempDir()
	stageDev, err := filesystemDevice(stageRoot)
	if err != nil {
		return inspection, fmt.Errorf("inspect staging filesystem %s: %w", stageRoot, err)
	}
	destBase, err := nearestExistingDirectory(plan.OutputDirectory)
	if err != nil {
		return inspection, err
	}
	destDev, err := filesystemDevice(destBase)
	if err != nil {
		return inspection, fmt.Errorf("inspect output filesystem %s: %w", destBase, err)
	}
	destFree, err := freeBytes(destBase)
	if err != nil {
		return inspection, err
	}
	stageFree, err := freeBytes(stageRoot)
	if err != nil {
		return inspection, err
	}
	inspection.DestinationFreeBytes = destFree
	if destFree > plan.MinimumFreeBytes {
		inspection.GrowthHeadroomBytes = destFree - plan.MinimumFreeBytes
	}
	inspection.DestinationDevice = destDev
	inspection.StagingFreeBytes = stageFree
	inspection.StagingDevice = stageDev
	inspection.SameFilesystem = stageDev == destDev
	manifestPath := filepath.Join(root, plan.ImageManifest)
	manifest, manifestErr := readManifestSafely(manifestPath)
	if manifestErr != nil {
		inspection.Stops = append(inspection.Stops, "existing image manifest cannot be read: "+manifestErr.Error())
	}
	missingBytes := uint64(0)
	allPresent := manifestErr == nil && manifestMatches(plan, manifest)
	receipt, receiptErr := readReceiptSafely(plan.ReceiptPath)
	if receiptErr != nil || !receiptMatches(plan, receipt) {
		allPresent = false
		if !errors.Is(receiptErr, os.ErrNotExist) {
			inspection.Stops = append(inspection.Stops, "existing durable receipt is invalid for the pinned asset plan; refusing to overwrite it")
		}
	} else {
		inspection.ChecksumSignature = receipt.ChecksumSignature
	}
	for _, image := range plan.Images {
		destination := filepath.Join(plan.OutputDirectory, image.FileName)
		entry := ImagePlan{FileName: image.FileName, SourceURL: image.SourceURL, SHA256: image.SHA256, Destination: destination, State: "would_download_verify_and_install"}
		existing, statErr := os.Lstat(destination)
		if statErr == nil {
			if existing.Mode()&os.ModeSymlink != 0 || !existing.Mode().IsRegular() {
				inspection.Stops = append(inspection.Stops, "existing destination is not a regular image file: "+destination)
				inspection.Images = append(inspection.Images, entry)
				continue
			}
			digest, hashErr := fileSHA256(destination)
			if hashErr != nil || !strings.EqualFold(digest, image.SHA256) {
				inspection.Stops = append(inspection.Stops, "existing destination image is not the pinned digest; refusing replacement: "+destination)
			} else {
				entry.State = "already_present_verified"
				if existing.Mode().Perm() != 0444 {
					inspection.Stops = append(inspection.Stops, "existing pinned image is not read-only (0444): "+destination)
				}
				if existing.Size() >= 0 {
					entry.SizeBytes = uint64(existing.Size())
				}
			}
		} else if !errors.Is(statErr, os.ErrNotExist) {
			inspection.Stops = append(inspection.Stops, "cannot inspect destination image "+destination+": "+statErr.Error())
		} else {
			size, sizeErr := remoteSize(ctx, image.SourceURL)
			if sizeErr != nil {
				inspection.Stops = append(inspection.Stops, "cannot verify Canonical image size for "+image.FileName+": "+sizeErr.Error())
				continue
			}
			entry.SizeBytes = size
			if size > plan.MaximumImageBytes {
				inspection.Stops = append(inspection.Stops, fmt.Sprintf("image %s is %d bytes, over the per-image cap %d", image.FileName, size, plan.MaximumImageBytes))
			}
			missingBytes += size
			allPresent = false
		}
		inspection.Images = append(inspection.Images, entry)
	}
	if !allPresent {
		if checksum, signature, _, _, sigErr := fetchAndVerifySums(ctx, plan); sigErr != nil {
			inspection.Stops = append(inspection.Stops, "signed Canonical checksum metadata verification failed: "+sigErr.Error())
		} else {
			inspection.ChecksumSignature = "verified by gpgv; signing fingerprint " + signature
			for _, image := range plan.Images {
				if got := checksum[image.FileName]; !strings.EqualFold(got, image.SHA256) {
					inspection.Stops = append(inspection.Stops, "signed SHA256SUMS does not contain the pinned digest for "+image.FileName)
				}
			}
		}
	}
	inspection.Effects = append(inspection.Effects, "Verify the signed checksum list with gpgv before either image is accepted; verify both fixed SHA-256 digests before copy")
	inspection.Effects = append(inspection.Effects, "Image peak counts one complete staging copy plus destination copy when both paths share a filesystem, plus a 64 KiB receipt/manifest allowance")
	stagePeak := missingBytes
	outputPeak := missingBytes + metadataGrowthAllowance
	if stageDev == destDev {
		outputPeak += missingBytes
	}
	inspection.FinalGrowthBytes = missingBytes + metadataGrowthAllowance
	inspection.PeakGrowthBytes = stagePeak + missingBytes + metadataGrowthAllowance
	if inspection.PeakGrowthBytes > plan.MaximumGrowthBytes {
		inspection.Stops = append(inspection.Stops, fmt.Sprintf("A2 peak growth %d bytes exceeds the %d-byte cap", inspection.PeakGrowthBytes, plan.MaximumGrowthBytes))
	}
	if destFree < outputPeak || destFree-outputPeak < plan.MinimumFreeBytes {
		inspection.Stops = append(inspection.Stops, fmt.Sprintf("output filesystem would have less than the required %d free bytes after the %d-byte peak", plan.MinimumFreeBytes, outputPeak))
	}
	if stageFree < stagePeak {
		inspection.Stops = append(inspection.Stops, fmt.Sprintf("staging filesystem has %d free bytes; %d are needed for verified download staging", stageFree, stagePeak))
	}
	if manifestErr == nil && !manifestMatches(plan, manifest) && len(manifest.Images) > 0 {
		inspection.Stops = append(inspection.Stops, "existing qemu image manifest contains a different image set; refusing to overwrite it")
	}
	inspection.AlreadyReady = allPresent && manifestErr == nil && manifestMatches(plan, manifest) && receiptErr == nil && receiptMatches(plan, receipt) && len(inspection.Stops) == 0
	return inspection, nil
}

// ValidateInitialHeadroom reserves the entire approved A2 growth budget before
// any package installation begins. Apply later measures actual package growth
// and combines it with the image staging/copy peak before writing images.
func ValidateInitialHeadroom(plan Plan, inspection Inspection) error {
	if inspection.DestinationFreeBytes < plan.MinimumFreeBytes {
		return fmt.Errorf("output filesystem has %d free bytes; A2 requires at least %d", inspection.DestinationFreeBytes, plan.MinimumFreeBytes)
	}
	if inspection.GrowthHeadroomBytes < plan.MaximumGrowthBytes {
		return fmt.Errorf("output filesystem has %d bytes of headroom above the reserve; A2 requires the full %d-byte growth budget before setup", inspection.GrowthHeadroomBytes, plan.MaximumGrowthBytes)
	}
	return nil
}

func ValidateToolPreview(plan Plan, report hostreqkit.Report) error {
	if report.Host.OS != "linux" || (report.Host.PackageManager != "apt" && report.Host.PackageManager != "apt-get") {
		return fmt.Errorf("A2 setup supports only the reviewed Linux/apt host package path")
	}
	actual := make(map[string]string, len(report.Tools))
	for _, tool := range report.Tools {
		actual[tool.Name] = tool.PackageName
	}
	if len(actual) != len(plan.HostTools) {
		return fmt.Errorf("scenario host requirements contain %d tools; A2 approves exactly %d", len(actual), len(plan.HostTools))
	}
	if len(report.Safeguards) != 0 {
		return fmt.Errorf("scenario setup includes %d unapproved safeguard changes", len(report.Safeguards))
	}
	for _, expected := range plan.HostTools {
		if got := actual[expected.Name]; got != expected.Package {
			return fmt.Errorf("host tool %s package preview changed: expected %s, got %s", expected.Name, expected.Package, got)
		}
	}
	return nil
}

func remoteSize(ctx context.Context, source string) (uint64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Range", "bytes=0-0")
	client := secureClient()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("range probe returned HTTP %d", resp.StatusCode)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2))
	parts := strings.Split(resp.Header.Get("Content-Range"), "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("range probe returned no total content length")
	}
	return strconv.ParseUint(parts[1], 10, 64)
}

func secureClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" || req.URL.Host != "cloud-images.ubuntu.com" {
			return fmt.Errorf("refusing off-origin redirect to %s", req.URL)
		}
		if len(via) >= 4 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}}
}

func fetchAndVerifySums(ctx context.Context, plan Plan) (map[string]string, string, string, string, error) {
	tmp, err := os.MkdirTemp("", "vrooli-setup-assets-metadata-")
	if err != nil {
		return nil, "", "", "", err
	}
	defer os.RemoveAll(tmp)
	sumsPath, sigPath := filepath.Join(tmp, "SHA256SUMS"), filepath.Join(tmp, "SHA256SUMS.gpg")
	if err := downloadBounded(ctx, plan.ChecksumURL, sumsPath, 1<<20); err != nil {
		return nil, "", "", "", err
	}
	if err := downloadBounded(ctx, plan.SignatureURL, sigPath, 64<<10); err != nil {
		return nil, "", "", "", err
	}
	out, err := exec.CommandContext(ctx, "gpgv", "--status-fd=1", "--keyring", plan.KeyringPath, sigPath, sumsPath).CombinedOutput()
	if err != nil {
		return nil, "", "", "", fmt.Errorf("gpgv: %s: %w", strings.TrimSpace(string(out)), err)
	}
	fingerprint := ""
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "[GNUPG:]" && fields[1] == "VALIDSIG" {
			fingerprint = strings.ToUpper(fields[2])
			break
		}
	}
	if fingerprint == "" {
		return nil, "", "", "", fmt.Errorf("gpgv returned success without a VALIDSIG fingerprint")
	}
	data, err := os.ReadFile(sumsPath)
	if err != nil {
		return nil, "", "", "", err
	}
	signatureData, err := os.ReadFile(sigPath)
	if err != nil {
		return nil, "", "", "", err
	}
	sumDigest := sha256.Sum256(data)
	sigDigest := sha256.Sum256(signatureData)
	checksums := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		checksums[filepath.Base(name)] = strings.ToLower(fields[0])
	}
	return checksums, fingerprint, hex.EncodeToString(sumDigest[:]), hex.EncodeToString(sigDigest[:]), nil
}

func downloadBounded(ctx context.Context, source, destination string, max uint64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	resp, err := secureClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned HTTP %d", source, resp.StatusCode)
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(resp.Body, int64(max)+1))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if uint64(written) > max {
		return fmt.Errorf("download exceeded %d-byte limit", max)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validFingerprint(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func readManifest(path string) (ImageManifest, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ImageManifest{SchemaVersion: imageManifestSchemaVersion}, nil
	}
	if err != nil {
		return ImageManifest{}, err
	}
	var manifest ImageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ImageManifest{}, err
	}
	if manifest.SchemaVersion != imageManifestSchemaVersion {
		return ImageManifest{}, fmt.Errorf("unsupported manifest schema version %d", manifest.SchemaVersion)
	}
	return manifest, nil
}

func readManifestSafely(path string) (ImageManifest, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return ImageManifest{SchemaVersion: imageManifestSchemaVersion}, nil
	}
	if err != nil {
		return ImageManifest{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return ImageManifest{}, fmt.Errorf("manifest is not a regular file")
	}
	return readManifest(path)
}

func readReceipt(path string) (Receipt, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Receipt{}, err
	}
	var receipt Receipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return Receipt{}, err
	}
	if receipt.Version != imageReceiptVersion {
		return Receipt{}, fmt.Errorf("unsupported image receipt version %d", receipt.Version)
	}
	return receipt, nil
}

func readReceiptSafely(path string) (Receipt, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Receipt{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Receipt{}, fmt.Errorf("receipt is not a regular file")
	}
	return readReceipt(path)
}

func receiptMatches(plan Plan, receipt Receipt) bool {
	fingerprint := strings.TrimPrefix(receipt.ChecksumSignature, "gpgv-validsig:")
	if receipt.Version != imageReceiptVersion || receipt.ChecksumURL != plan.ChecksumURL || !validFingerprint(fingerprint) || !validSHA256(receipt.ChecksumFileSHA256) || !validSHA256(receipt.SignatureFileSHA256) || receipt.GrowthLimitBytes > plan.MaximumGrowthBytes || receipt.ReserveBytes < plan.MinimumFreeBytes {
		return false
	}
	manifest := ImageManifest{SchemaVersion: imageManifestSchemaVersion, Images: receipt.Images}
	return manifestMatches(plan, manifest)
}

func manifestMatches(plan Plan, manifest ImageManifest) bool {
	if len(manifest.Images) != len(plan.Images) {
		return false
	}
	for _, image := range plan.Images {
		found := false
		for _, record := range manifest.Images {
			if filepath.Clean(record.Path) == filepath.Join(plan.OutputDirectory, image.FileName) && strings.EqualFold(record.SHA256, image.SHA256) && record.OS == image.OS && record.Arch == image.Arch && record.SourceURL == image.SourceURL {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func apply(ctx context.Context, root string, plan Plan, planPath, sudoMode string, baseline, initial Inspection, packageGrowth uint64, stdout, stderr io.Writer) error {
	stage, err := os.MkdirTemp("", "vrooli-setup-assets-")
	if err != nil {
		return fmt.Errorf("create verified asset staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	checksums, fingerprint, checksumDigest, signatureDigest, err := fetchAndVerifySums(ctx, plan)
	if err != nil {
		return fmt.Errorf("verify signed Canonical SHA256SUMS: %w", err)
	}
	for _, image := range plan.Images {
		if got := checksums[image.FileName]; !strings.EqualFold(got, image.SHA256) {
			return fmt.Errorf("signed SHA256SUMS expected %s for %s, got %s", image.SHA256, image.FileName, got)
		}
	}
	imageRecords := make([]ImageRecord, 0, len(plan.Images))
	var stagedBytes uint64
	for _, image := range plan.Images {
		destination := filepath.Join(plan.OutputDirectory, image.FileName)
		if _, err := os.Stat(destination); err == nil {
			digest, hashErr := fileSHA256(destination)
			if hashErr != nil || !strings.EqualFold(digest, image.SHA256) {
				return fmt.Errorf("destination exists with a non-pinned digest; refusing to replace %s", destination)
			}
			imageRecords = append(imageRecords, ImageRecord{Path: destination, SHA256: image.SHA256, OS: image.OS, Arch: image.Arch, SourceURL: image.SourceURL, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano)})
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect image destination %s: %w", destination, err)
		}
		stagePath := filepath.Join(stage, image.FileName)
		if err := downloadBounded(ctx, image.SourceURL, stagePath, plan.MaximumImageBytes); err != nil {
			return fmt.Errorf("download pinned image %s: %w", image.FileName, err)
		}
		info, err := os.Stat(stagePath)
		if err != nil {
			return err
		}
		if info.Size() <= 0 || uint64(info.Size()) > plan.MaximumImageBytes {
			return fmt.Errorf("image %s size %d exceeds the declared per-image limit", image.FileName, info.Size())
		}
		if expected, ok := findPlannedSize(initial, image.FileName); ok && uint64(info.Size()) != expected {
			return fmt.Errorf("image %s changed size after review: planned %d, received %d", image.FileName, expected, info.Size())
		}
		digest, err := fileSHA256(stagePath)
		if err != nil {
			return err
		}
		if !strings.EqualFold(digest, image.SHA256) {
			return fmt.Errorf("image %s SHA-256 mismatch: expected %s, got %s", image.FileName, image.SHA256, digest)
		}
		stagedBytes += uint64(info.Size())
		imageRecords = append(imageRecords, ImageRecord{Path: destination, SHA256: image.SHA256, OS: image.OS, Arch: image.Arch, SourceURL: image.SourceURL, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	}
	if stagedBytes == 0 && initial.AlreadyReady {
		return nil
	}
	// Downloads and final copies coexist until verification and durable receipts
	// complete. Recheck actual free capacity after staging, on the destination
	// filesystem itself, before creating or copying any output image.
	destBase, err := nearestExistingDirectory(plan.OutputDirectory)
	if err != nil {
		return err
	}
	freeNow, err := freeBytes(destBase)
	if err != nil {
		return err
	}
	stageRoot := os.TempDir()
	destDev, err := filesystemDevice(destBase)
	if err != nil {
		return err
	}
	stageDev, err := filesystemDevice(stageRoot)
	if err != nil {
		return err
	}
	if destDev != baseline.DestinationDevice || stageDev != baseline.StagingDevice {
		return fmt.Errorf("stop: output or staging filesystem identity changed after review")
	}
	outputCopyBytes := stagedBytes + metadataGrowthAllowance
	if freeNow < outputCopyBytes || freeNow-outputCopyBytes < plan.MinimumFreeBytes {
		return fmt.Errorf("stop: destination filesystem would fall below %d free bytes during peak copy", plan.MinimumFreeBytes)
	}
	peakGrowth := packageGrowth + stagedBytes + stagedBytes + metadataGrowthAllowance
	if peakGrowth > plan.MaximumGrowthBytes {
		return fmt.Errorf("stop: A2 peak growth %d exceeds cap %d", peakGrowth, plan.MaximumGrowthBytes)
	}
	_, _ = fmt.Fprintf(stdout, "[INFO]    Signed checksum and image digests verified; installing %d pinned image(s) to %s.\n", len(imageRecords), plan.OutputDirectory)
	if _, err := os.Stat(plan.OutputDirectory); errors.Is(err, os.ErrNotExist) {
		if err := runPrivileged(ctx, sudoMode, stdout, stderr, "install", "-d", "-m", "0755", plan.OutputDirectory); err != nil {
			return fmt.Errorf("create image destination: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("inspect image destination: %w", err)
	}
	for _, image := range plan.Images {
		stagePath := filepath.Join(stage, image.FileName)
		if _, err := os.Stat(stagePath); errors.Is(err, os.ErrNotExist) {
			continue
		}
		destination := filepath.Join(plan.OutputDirectory, image.FileName)
		if err := runPrivileged(ctx, sudoMode, stdout, stderr, "cp", "--no-clobber", "--preserve=mode", "--", stagePath, destination); err != nil {
			return fmt.Errorf("install image %s: %w", image.FileName, err)
		}
		if err := runPrivileged(ctx, sudoMode, stdout, stderr, "chmod", "0444", destination); err != nil {
			return err
		}
		digest, err := fileSHA256(destination)
		if err != nil || !strings.EqualFold(digest, image.SHA256) {
			return fmt.Errorf("installed image %s did not retain its verified digest", destination)
		}
	}
	freeAfter, err := freeBytes(plan.OutputDirectory)
	if err != nil {
		return err
	}
	if freeAfter < plan.MinimumFreeBytes {
		return fmt.Errorf("stop: output filesystem has %d free bytes after image writes, below the %d-byte reserve", freeAfter, plan.MinimumFreeBytes)
	}
	actualPeak := positiveDelta(baseline.DestinationFreeBytes, freeAfter)
	if baseline.DestinationDevice != baseline.StagingDevice {
		stageFreeAfter, stageErr := freeBytes(os.TempDir())
		if stageErr != nil {
			return stageErr
		}
		actualPeak += positiveDelta(baseline.StagingFreeBytes, stageFreeAfter)
	}
	actualPeak += metadataGrowthAllowance
	if actualPeak > peakGrowth {
		peakGrowth = actualPeak
	}
	if peakGrowth > plan.MaximumGrowthBytes {
		return fmt.Errorf("stop: observed A2 peak growth %d exceeds cap %d", peakGrowth, plan.MaximumGrowthBytes)
	}
	receipt := Receipt{Version: imageReceiptVersion, Plan: planPath, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), ChecksumSignature: "gpgv-validsig:" + fingerprint, ChecksumURL: plan.ChecksumURL, ChecksumFileSHA256: checksumDigest, SignatureFileSHA256: signatureDigest, Images: imageRecords, HostTools: plan.HostTools, DestinationFilesystem: plan.OutputDirectory, DestinationDevice: baseline.DestinationDevice, StagingDevice: baseline.StagingDevice, FreeBytesBefore: baseline.DestinationFreeBytes, FreeBytesAfterImages: freeAfter, StagingFreeBytesBefore: baseline.StagingFreeBytes, HostToolGrowthBytes: packageGrowth, ImageStagingBytes: stagedBytes, ImageFinalBytes: stagedBytes, PeakGrowthBytes: peakGrowth, ReserveBytes: plan.MinimumFreeBytes, GrowthLimitBytes: plan.MaximumGrowthBytes}
	receiptBytes, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	if err := writePrivilegedAtomic(ctx, sudoMode, plan.ReceiptPath, receiptBytes, stdout, stderr); err != nil {
		return fmt.Errorf("write durable image receipt: %w", err)
	}
	manifest := ImageManifest{SchemaVersion: imageManifestSchemaVersion, Note: "Canonical signed cloud images, pinned by SHA-256 for the scenario-to-cloud QEMU proving ground.", Images: imageRecords}
	manifestPath := filepath.Join(root, plan.ImageManifest)
	if err := writeManifestAtomic(manifestPath, manifest); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "[INFO]    Verified image manifest written to %s; durable receipt written to %s.\n", manifestPath, plan.ReceiptPath)
	return nil
}

func findPlannedSize(inspection Inspection, name string) (uint64, bool) {
	for _, image := range inspection.Images {
		if image.FileName == name {
			return image.SizeBytes, true
		}
	}
	return 0, false
}

func runPrivileged(ctx context.Context, mode string, stdout, stderr io.Writer, name string, args ...string) error {
	cmdName, cmdArgs, err := hostreqkit.WithSudo(mode, name, args)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	return cmd.Run()
}

func writePrivilegedAtomic(ctx context.Context, mode, destination string, data []byte, stdout, stderr io.Writer) error {
	stage, err := os.CreateTemp("", "vrooli-image-receipt-")
	if err != nil {
		return err
	}
	stageName := stage.Name()
	defer os.Remove(stageName)
	if _, err := stage.Write(data); err != nil {
		_ = stage.Close()
		return err
	}
	if err := stage.Sync(); err != nil {
		_ = stage.Close()
		return err
	}
	if err := stage.Close(); err != nil {
		return err
	}
	directory := filepath.Dir(destination)
	temporaryDestination := filepath.Join(directory, "."+filepath.Base(destination)+".tmp-"+filepath.Base(stageName))
	if err := runPrivileged(ctx, mode, stdout, stderr, "install", "-m", "0444", stageName, temporaryDestination); err != nil {
		return err
	}
	if err := runPrivileged(ctx, mode, stdout, stderr, "mv", "-fT", "--", temporaryDestination, destination); err != nil {
		return err
	}
	return nil
}

func writeManifestAtomic(path string, manifest ImageManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".qemu-images-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
