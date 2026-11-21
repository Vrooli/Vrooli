package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/lightingruntime"
)

const (
	openRGBDevicePolicyFilename = "device-policy.json"
	openRGBDevicePolicyMaxBytes = 16 << 10
)

// OpenRGBDevicePolicyRecord is an explicit machine-local selection. Each
// enabled detector is bound to a physical selector; it is never inferred from
// device enumeration at startup.
type OpenRGBDevicePolicyRecord struct {
	SchemaVersion int                             `json:"schema_version"`
	Policy        hostreqkit.OpenRGBLinuxPolicy   `json:"policy"`
	NativePolicy  *hostreqkit.OpenRGBNativePolicy `json:"native_policy,omitempty"`
}

// StoreOpenRGBDevicePolicy stores an already qualified, explicit selection.
// It does not probe hardware or modify host permissions. The policy ID binds
// exact detector names, USB interfaces or PCI topology/adapter selectors, and
// the static permission grant evidence supplied by its owning host service.
func (c *Controller) StoreOpenRGBDevicePolicy(policy hostreqkit.OpenRGBLinuxPolicy) (string, error) {
	if c == nil || strings.TrimSpace(c.Root) == "" || strings.TrimSpace(c.Home) == "" {
		return "", errors.New("OpenRGB device policy requires a resource root and operator home")
	}
	manifest, err := c.managedServiceArtifactManifest(openRGBResourceName)
	if err != nil {
		return "", err
	}
	artifactSHA, ok := manifest.ManagedService.Artifact.SHA256ByPlatform[runtimePlatformKey()]
	if !ok || artifactSHA != policy.ArtifactSHA256 {
		return "", errors.New("OpenRGB device policy is not bound to the pinned local artifact")
	}
	asset, err := readManagedUpdateConfigFile(filepath.Join(c.Root, "resources", openRGBResourceName), manifest.ManagedService.Config.ContentFile)
	if err != nil {
		return "", fmt.Errorf("read pinned OpenRGB detector catalog: %w", err)
	}
	detectors, err := decodeOpenRGBDetectorCatalog(asset)
	if err != nil {
		return "", err
	}
	detectorDigest := fmt.Sprintf("%x", sha256.Sum256(asset))
	if detectorDigest != policy.DetectorSHA256 {
		return "", errors.New("OpenRGB device policy is not bound to the pinned detector catalog")
	}
	if _, err := selectedOpenRGBDetectors(policy, detectors); err != nil {
		return "", err
	}
	return c.writeOpenRGBDevicePolicyRecord(policy)
}

// Prior-release rollback is allowed only for an exact source-qualified scope;
// it changes selected metadata, never the installed physical permission grant.
func (c *Controller) storeOpenRGBDevicePolicyRecord(policy hostreqkit.OpenRGBLinuxPolicy) (string, error) {
	if err := hostreqkit.ValidateOpenRGBLinuxDetectorScope(policy); err != nil {
		return "", err
	}
	return c.writeOpenRGBDevicePolicyRecord(policy)
}
func (c *Controller) writeOpenRGBDevicePolicyRecord(policy hostreqkit.OpenRGBLinuxPolicy) (string, error) {
	id, err := policy.ID()
	if err != nil {
		return "", err
	}
	record := OpenRGBDevicePolicyRecord{SchemaVersion: openRGBDevicePolicySchemaVersion, Policy: policy}
	data, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	if len(data) > openRGBDevicePolicyMaxBytes {
		return "", errors.New("OpenRGB device policy exceeds the size limit")
	}
	path, err := openRGBDevicePolicyPath(c.Home)
	if err != nil {
		return "", err
	}
	if err := ensureOpenRGBDevicePolicyDirectory(filepath.Dir(path), true); err != nil {
		return "", err
	}
	lock, err := acquireOpenRGBDevicePolicyPublicationLock(context.Background(), path)
	if err != nil {
		return "", err
	}
	defer lock()
	if err := writeResourceOwnedFileDurable(path, data, 0o600); err != nil {
		return "", fmt.Errorf("store machine-local OpenRGB device policy: %w", err)
	}
	if err := confirmOpenRGBPublicationDurability(filepath.Dir(path)); err != nil {
		return "", err
	}
	return id, nil
}

// LoadOpenRGBDevicePolicy returns the exact selected policy for hostreqkit's
// permission verifier. A missing policy is an error for an explicit setup,
// while ordinary config generation treats its absence as all detectors off.
func (c *Controller) LoadOpenRGBDevicePolicy() (hostreqkit.OpenRGBLinuxPolicy, error) {
	if c == nil {
		return hostreqkit.OpenRGBLinuxPolicy{}, errors.New("OpenRGB resource controller is unavailable")
	}
	manifest, err := c.managedServiceArtifactManifest(openRGBResourceName)
	if err != nil {
		return hostreqkit.OpenRGBLinuxPolicy{}, err
	}
	record, err := c.loadOpenRGBDevicePolicy(manifest)
	if err != nil {
		return hostreqkit.OpenRGBLinuxPolicy{}, err
	}
	if record == nil {
		return hostreqkit.OpenRGBLinuxPolicy{}, os.ErrNotExist
	}
	if record.NativePolicy != nil {
		return hostreqkit.OpenRGBLinuxPolicy{}, errors.New("native HID enrollment is not a Linux udev policy")
	}
	return record.Policy, nil
}

func (c *Controller) readOpenRGBDevicePolicyRecord() (*OpenRGBDevicePolicyRecord, error) {
	if c == nil || strings.TrimSpace(c.Root) == "" || strings.TrimSpace(c.Home) == "" {
		return nil, nil
	}
	path, err := openRGBDevicePolicyPath(c.Home)
	if err != nil {
		return nil, err
	}
	if err := ensureOpenRGBDevicePolicyDirectory(filepath.Dir(path), false); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect machine-local OpenRGB device policy: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > openRGBDevicePolicyMaxBytes {
		return nil, errors.New("machine-local OpenRGB device policy has unsafe type, permissions, or size")
	}
	if err := verifyManagedUpdatePrivatePath(path, info); err != nil {
		return nil, err
	}
	data, err := readManagedUpdatePrivateFile(path, openRGBDevicePolicyMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("read machine-local OpenRGB device policy: %w", err)
	}
	var record OpenRGBDevicePolicyRecord
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return nil, errors.Join(errors.New("machine-local OpenRGB device policy is invalid"), err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	nativeField := false
	for key := range fields {
		if strings.EqualFold(key, "native_policy") {
			nativeField = true
		}
	}
	if record.SchemaVersion == openRGBDevicePolicySchemaVersion {
		if nativeField {
			return nil, errors.New("Linux v1 OpenRGB policy cannot contain native fields")
		}
	} else if record.SchemaVersion == openRGBNativeDevicePolicySchemaVersion {
		if err := validateManagedUpdateJSON(data); err != nil {
			return nil, err
		}
		if record.NativePolicy == nil || !reflect.DeepEqual(record.Policy, hostreqkit.OpenRGBLinuxPolicy{}) {
			return nil, errors.New("native v2 OpenRGB policy requires exactly one native policy")
		}
		if _, err := record.NativePolicy.ID(); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("unknown OpenRGB device policy version")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, errors.Join(errors.New("machine-local OpenRGB device policy has trailing data"), err)
	}
	return &record, nil
}

func (c *Controller) loadOpenRGBDevicePolicy(manifest ResourceManifest) (*OpenRGBDevicePolicyRecord, error) {
	record, err := c.readOpenRGBDevicePolicyRecord()
	if err != nil || record == nil {
		return record, err
	}
	if record.NativePolicy != nil {
		if err := validateNativeOpenRGBPolicyManifest(*record.NativePolicy, manifest, c.Root); err != nil {
			return nil, err
		}
		return record, nil
	}
	asset, err := readManagedUpdateConfigFile(filepath.Join(c.Root, "resources", openRGBResourceName), manifest.ManagedService.Config.ContentFile)
	if err != nil {
		return nil, err
	}
	detectors, err := decodeOpenRGBDetectorCatalog(asset)
	if err != nil {
		return nil, err
	}
	if record.Policy.DetectorSHA256 != fmt.Sprintf("%x", sha256.Sum256(asset)) {
		return nil, errors.New("machine-local OpenRGB policy detector catalog digest is stale")
	}
	artifactSHA, ok := manifest.ManagedService.Artifact.SHA256ByPlatform[runtimePlatformKey()]
	if !ok || record.Policy.ArtifactSHA256 != artifactSHA {
		return nil, errors.New("machine-local OpenRGB policy artifact digest is stale")
	}
	if _, err := selectedOpenRGBDetectors(record.Policy, detectors); err != nil {
		return nil, err
	}
	return record, nil
}

func (c *Controller) selectedOpenRGBDetectors(manifest ResourceManifest) ([]string, string, error) {
	record, err := c.loadOpenRGBDevicePolicy(manifest)
	if err != nil || record == nil {
		return nil, "", err
	}
	asset, err := readManagedUpdateConfigFile(filepath.Join(c.Root, "resources", openRGBResourceName), manifest.ManagedService.Config.ContentFile)
	if err != nil {
		return nil, "", err
	}
	detectors, err := decodeOpenRGBDetectorCatalog(asset)
	if err != nil {
		return nil, "", err
	}
	names, err := record.selectedDetectors(detectors)
	if err != nil {
		return nil, "", err
	}
	id, err := record.ID()
	return names, id, err
}

func openRGBDevicePolicyPath(home string) (string, error) {
	if !filepath.IsAbs(home) || filepath.Clean(home) != home {
		return "", errors.New("OpenRGB device policy requires a canonical absolute operator home")
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		root, err := OpenRGBNativeResourceRoot()
		if err != nil {
			return "", err
		}
		return filepath.Join(root, "state", "host-lighting", openRGBDevicePolicyFilename), nil
	}
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", err
	}
	return filepath.Join(state, "host-lighting", openRGBDevicePolicyFilename), nil
}

func ensureOpenRGBDevicePolicyDirectory(path string, create bool) error {
	if runtime.GOOS != "linux" {
		if create {
			p, err := resourcePrivatePathPolicy(path)
			if err != nil {
				return err
			}
			return config.EnsurePrivateDirectory(path, p)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return errors.New("native OpenRGB policy parent is not a directory")
		}
		return verifyManagedUpdatePrivatePath(path, info)
	}
	if err := lightingruntime.EnsureStateDirectory(path, create); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return errors.New("machine-local OpenRGB policy directory must be a private, real directory")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(path) {
		return errors.Join(errors.New("machine-local OpenRGB policy directory resolves through a symlink"), err)
	}
	return nil
}

func confirmOpenRGBPublicationDurability(path string) error {
	// Native Windows publication uses write-through MoveFileEx in the private
	// writer; a POSIX directory fsync is neither available nor substituted.
	if runtime.GOOS == "windows" {
		return nil
	}
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open OpenRGB publication directory for durability: %w", err)
	}
	if err := errors.Join(dir.Sync(), dir.Close()); err != nil {
		return fmt.Errorf("confirm durable OpenRGB publication: %w", err)
	}
	return nil
}

func decodeOpenRGBDetectorCatalog(data []byte) (map[string]bool, error) {
	var config struct {
		Detectors struct {
			Detectors map[string]bool `json:"detectors"`
		} `json:"Detectors"`
	}
	if len(data) == 0 || json.Unmarshal(data, &config) != nil || len(config.Detectors.Detectors) != 1948 {
		return nil, errors.New("pinned OpenRGB detector catalog is invalid")
	}
	for name, enabled := range config.Detectors.Detectors {
		if strings.TrimSpace(name) == "" || enabled {
			return nil, errors.New("pinned OpenRGB detector catalog must be host-neutral and all disabled")
		}
	}
	return config.Detectors.Detectors, nil
}

func selectedOpenRGBDetectors(policy hostreqkit.OpenRGBLinuxPolicy, catalog map[string]bool) ([]string, error) {
	if _, err := policy.ID(); err != nil {
		return nil, err
	}
	selected := make(map[string]struct{}, len(policy.Devices))
	for _, device := range policy.Devices {
		if _, ok := catalog[device.Detector]; !ok {
			return nil, fmt.Errorf("OpenRGB policy selects unknown detector %q", device.Detector)
		}
		if device.Detector == "ENE SMBus DRAM" {
			return nil, errors.New("ENE SMBus DRAM detector is write/remap-capable and cannot be selected")
		}
		if strings.HasPrefix(device.HardwareID, "0b05:19af:") && (device.Kind != "hidraw" || device.Detector != "ASUS Aura Motherboard") {
			return nil, errors.New("ASUS 0b05:19af must use its exact USB HID detector; SMBus fallback requires separate qualified identity")
		}
		if strings.HasPrefix(device.Detector, "ASUS TUF GeForce RTX") && (device.Kind != "i2c" || device.Adapter == "" || !strings.HasPrefix(device.Topology, "pci")) {
			return nil, errors.New("GPU detector selection requires exact PCI topology and adapter identity")
		}
		selected[device.Detector] = struct{}{}
	}
	if len(selected) == 0 {
		return nil, errors.New("OpenRGB machine-local policy selects no detectors")
	}
	names := make([]string, 0, len(selected))
	for name := range selected {
		names = append(names, name)
	}
	slices.Sort(names)
	return names, nil
}

func runtimePlatformKey() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}
