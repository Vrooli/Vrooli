package instance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
)

// DefaultImageManifestPath is the recorded image manifest for the QEMU lane,
// relative to the scenario root. Operators record approved base images and
// their digests there; the provider never downloads an image on its own.
const DefaultImageManifestPath = "certification/lanes/qemu-images.json"

// ImageManifestEnv overrides DefaultImageManifestPath.
const ImageManifestEnv = "VROOLI_QEMU_IMAGE_MANIFEST"

// Host tool names and the service.json hostTools declaration that owns each.
// Installing any of them is the job of `vrooli setup`, the sole elevation
// boundary; this package only observes.
const (
	toolQEMUAMD64   = "qemu-system-x86_64"
	toolQEMUARM64   = "qemu-system-aarch64"
	toolQEMUImg     = "qemu-img"
	toolCloudLocal  = "cloud-localds"
	ownerQEMU       = "qemu"
	ownerCloudLocal = "cloud-localds"
	setupReference  = "vrooli setup --scenarios scenario-to-cloud"
)

// Architecture names use the Go/OCI vocabulary.
const (
	ArchAMD64 = "amd64"
	ArchARM64 = "arm64"
)

// ArchitectureState says how a guest architecture can execute on this host.
type ArchitectureState string

const (
	// ArchStateKVM runs with hardware acceleration.
	ArchStateKVM ArchitectureState = "kvm"
	// ArchStateTCG runs under software emulation (slow, but a real VM).
	ArchStateTCG ArchitectureState = "tcg"
	// ArchStateUnavailable cannot run: the emulator binary is absent.
	ArchStateUnavailable ArchitectureState = "unavailable"
)

// ReadinessState is the lane verdict.
type ReadinessState string

const (
	ReadinessReady       ReadinessState = "ready"
	ReadinessUnavailable ReadinessState = "unavailable"
)

// ToolReadiness reports one declared host tool.
type ToolReadiness struct {
	Name          string `json:"name"`
	Present       bool   `json:"present"`
	Path          string `json:"path,omitempty"`
	DeclaredOwner string `json:"declared_owner"`
}

// ImageRecord is one approved base image from the recorded manifest.
type ImageRecord struct {
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	SourceURL  string `json:"source_url,omitempty"`
	RecordedAt string `json:"recorded_at,omitempty"`
	// Present is whether the path is readable on this host.
	Present bool `json:"present"`
	// DigestState is "verified", "mismatch", "unverified" (present, hashing
	// not requested) or "missing".
	DigestState string `json:"digest_state"`
	Error       string `json:"error,omitempty"`
}

// ImageManifest is the on-disk shape of DefaultImageManifestPath.
type ImageManifest struct {
	SchemaVersion int           `json:"schema_version"`
	Images        []ImageRecord `json:"images"`
	Note          string        `json:"note,omitempty"`
}

// NextAction is the one operator action that moves the lane forward.
type NextAction struct {
	Owner     string `json:"owner"`
	Kind      string `json:"kind"`
	Reference string `json:"reference"`
	Label     string `json:"label"`
}

// LaneReadiness is the typed readiness report for the disposable QEMU lane.
// It is a report, never a silent failure: an absent tool is named with its
// declaring owner and the setup command that installs it.
type LaneReadiness struct {
	Provider         string                       `json:"provider"`
	State            ReadinessState               `json:"state"`
	Ready            bool                         `json:"ready"`
	HostArchitecture string                       `json:"host_architecture"`
	Tools            []ToolReadiness              `json:"tools"`
	KVMAvailable     bool                         `json:"kvm_available"`
	KVMDetail        string                       `json:"kvm_detail,omitempty"`
	ImageManifest    ManifestStatus               `json:"image_manifest"`
	Images           []ImageRecord                `json:"images"`
	Architectures    map[string]ArchitectureState `json:"architectures"`
	Limitations      []string                     `json:"limitations"`
	NextAction       NextAction                   `json:"next_action"`
}

// ManifestStatus reports where the image manifest was looked for.
type ManifestStatus struct {
	Path    string `json:"path"`
	Present bool   `json:"present"`
	Error   string `json:"error,omitempty"`
}

// Err returns a typed ErrProviderUnavailable naming the next action when the
// lane is not ready, and nil otherwise.
func (r LaneReadiness) Err() error {
	if r.Ready {
		return nil
	}
	return fmt.Errorf("%w: %s; next: %s", ErrProviderUnavailable, strings.Join(r.Limitations, "; "), r.NextAction.Reference)
}

// ReadinessOptions are the seams Readiness observes the host through.
type ReadinessOptions struct {
	LookPath          func(string) (string, error)
	KVMDevice         string
	HostArchitecture  string
	ImageManifestPath string
	// VerifyImages hashes every present image against its recorded sha256.
	// It is off by default because base images are gigabytes.
	VerifyImages bool
}

// Readiness observes the host and returns the lane report. It never returns
// an error: a host that cannot run the lane is a valid, explicitly not-ready
// answer.
func Readiness(ctx context.Context, opts ReadinessOptions) LaneReadiness {
	lookPath := opts.LookPath
	if lookPath == nil {
		lookPath = defaultLookPath
	}
	hostArch := opts.HostArchitecture
	if hostArch == "" {
		hostArch = goArchToOCI(runtime.GOARCH)
	}
	report := LaneReadiness{
		Provider:         (LocalQEMUProvider{}).Name(),
		State:            ReadinessUnavailable,
		HostArchitecture: hostArch,
		Architectures:    map[string]ArchitectureState{ArchAMD64: ArchStateUnavailable, ArchARM64: ArchStateUnavailable},
		Limitations:      []string{},
		Images:           []ImageRecord{},
	}
	tools := []struct{ name, owner string }{
		{toolQEMUAMD64, ownerQEMU}, {toolQEMUARM64, ownerQEMU}, {toolQEMUImg, ownerQEMU}, {toolCloudLocal, ownerCloudLocal},
	}
	present := map[string]bool{}
	for _, tool := range tools {
		entry := ToolReadiness{Name: tool.name, DeclaredOwner: tool.owner}
		if path, err := lookPath(tool.name); err == nil && path != "" {
			entry.Present, entry.Path = true, path
		}
		present[tool.name] = entry.Present
		report.Tools = append(report.Tools, entry)
	}

	kvmDevice := opts.KVMDevice
	if kvmDevice == "" {
		kvmDevice = "/dev/kvm"
	}
	report.KVMAvailable, report.KVMDetail = kvmUsable(kvmDevice)

	report.Architectures[ArchAMD64] = architectureState(present[toolQEMUAMD64], hostArch == ArchAMD64 && report.KVMAvailable)
	report.Architectures[ArchARM64] = architectureState(present[toolQEMUARM64], hostArch == ArchARM64 && report.KVMAvailable)

	manifestPath := opts.ImageManifestPath
	if manifestPath == "" {
		manifestPath = strings.TrimSpace(os.Getenv(ImageManifestEnv))
	}
	if manifestPath == "" {
		manifestPath = DefaultImageManifestPath
	}
	report.ImageManifest, report.Images = loadImages(ctx, manifestPath, opts.VerifyImages)

	missingTools := []string{}
	for _, tool := range report.Tools {
		if !tool.Present && tool.Name != toolQEMUARM64 {
			missingTools = append(missingTools, tool.Name+" (hostTools."+tool.DeclaredOwner+")")
		}
	}
	if len(missingTools) > 0 {
		report.Limitations = append(report.Limitations, "host tools absent (EXT-06): "+strings.Join(missingTools, ", "))
	}
	if !present[toolQEMUARM64] {
		report.Limitations = append(report.Limitations, "arm64 lane unavailable (EXT-05): "+toolQEMUARM64+" absent; arm64 needs TCG emulation on this host or an arm64 host")
	} else if report.Architectures[ArchARM64] == ArchStateTCG {
		report.Limitations = append(report.Limitations, "arm64 lane runs under TCG emulation on this "+hostArch+" host (slow; budgets from certification/budgets.json still apply)")
	}
	if !report.KVMAvailable {
		report.Limitations = append(report.Limitations, "KVM not usable: "+report.KVMDetail+"; the native architecture falls back to TCG")
	}
	usableImages := 0
	for _, image := range report.Images {
		if image.Present && image.DigestState != "mismatch" {
			usableImages++
		}
	}
	if len(report.Images) == 0 {
		report.Limitations = append(report.Limitations, "no approved base image recorded in "+report.ImageManifest.Path)
	} else if usableImages == 0 {
		report.Limitations = append(report.Limitations, "no recorded base image is readable with a matching digest")
	}

	report.NextAction = nextAction(len(missingTools) > 0, len(report.Images) == 0, usableImages == 0, report.ImageManifest.Path)
	report.Ready = len(missingTools) == 0 && usableImages > 0
	if report.Ready {
		report.State = ReadinessReady
	}
	sort.Strings(report.Limitations)
	return report
}

// Readiness reports the lane readiness through the provider's seams.
func (p LocalQEMUProvider) Readiness(ctx context.Context) LaneReadiness {
	return Readiness(ctx, ReadinessOptions{LookPath: p.LookPath, ImageManifestPath: p.ImageManifest})
}

func nextAction(toolsMissing, noImages, noUsableImage bool, manifestPath string) NextAction {
	switch {
	case toolsMissing:
		return NextAction{Owner: "operator", Kind: "host_setup", Reference: setupReference, Label: "Install the declared host tools qemu and cloud-localds through vrooli setup (the sole elevation boundary; EXT-06)"}
	case noImages:
		return NextAction{Owner: "operator", Kind: "record_image", Reference: manifestPath, Label: "Record an approved Ubuntu 24.04 base image path and sha256 per architecture in the image manifest"}
	case noUsableImage:
		return NextAction{Owner: "operator", Kind: "repair_image", Reference: manifestPath, Label: "A recorded base image is missing or its digest does not match; re-acquire it and re-record the digest"}
	default:
		return NextAction{Owner: "operator", Kind: "run_qualification", Reference: "program-runtime library run scenario-to-cloud.cloud-qemu-qualification", Label: "Run the governed QEMU qualification program"}
	}
}

func architectureState(emulatorPresent, accelerated bool) ArchitectureState {
	switch {
	case !emulatorPresent:
		return ArchStateUnavailable
	case accelerated:
		return ArchStateKVM
	default:
		return ArchStateTCG
	}
}

func kvmUsable(device string) (bool, string) {
	info, err := os.Stat(device)
	if err != nil {
		return false, device + " is absent"
	}
	if info.Mode()&os.ModeDevice == 0 {
		return false, device + " is not a device"
	}
	file, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return false, device + " exists but the service user cannot open it (not in the kvm group)"
		}
		return false, device + ": " + err.Error()
	}
	_ = file.Close()
	return true, ""
}

func loadImages(ctx context.Context, manifestPath string, verify bool) (ManifestStatus, []ImageRecord) {
	status := ManifestStatus{Path: manifestPath}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			status.Error = err.Error()
		}
		return status, []ImageRecord{}
	}
	status.Present = true
	var manifest ImageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		status.Error = "parse image manifest: " + err.Error()
		return status, []ImageRecord{}
	}
	if manifest.SchemaVersion != 1 {
		status.Error = fmt.Sprintf("unsupported image manifest schema_version %d", manifest.SchemaVersion)
		return status, []ImageRecord{}
	}
	images := make([]ImageRecord, 0, len(manifest.Images))
	for _, image := range manifest.Images {
		image.Present, image.DigestState, image.Error = false, "missing", ""
		if _, err := os.Stat(image.Path); err != nil {
			image.Error = err.Error()
			images = append(images, image)
			continue
		}
		image.Present = true
		image.DigestState = "unverified"
		if verify {
			if err := VerifyImageDigest(ctx, image.Path, image.SHA256); err != nil {
				image.DigestState = "mismatch"
				image.Error = err.Error()
			} else {
				image.DigestState = "verified"
			}
		}
		images = append(images, image)
	}
	return status, images
}

// VerifyImageDigest hashes path and compares it with the recorded sha256. It
// is the source-image preservation check the qualification program runs
// before creating an owned disk and again after destroying it (P20-A05).
func VerifyImageDigest(ctx context.Context, path, recorded string) error {
	want := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(recorded)), "sha256:")
	if len(want) != sha256.Size*2 {
		return fmt.Errorf("%w: recorded sha256 for %s is not a 64-hex digest", ErrInvalidRequest, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	buffer := make([]byte, 1<<20)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := file.Read(buffer)
		if n > 0 {
			hash.Write(buffer[:n])
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if got != want {
		return fmt.Errorf("image %s digest mismatch: recorded sha256:%s, observed sha256:%s", path, want, got)
	}
	return nil
}

func goArchToOCI(arch string) string {
	switch arch {
	case "amd64", "arm64":
		return arch
	case "x86_64":
		return ArchAMD64
	case "aarch64":
		return ArchARM64
	default:
		return arch
	}
}

func defaultLookPath(name string) (string, error) {
	return (LocalQEMUProvider{}).lookPath(name)
}
