package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func validateNativeOpenRGBPolicyManifest(p hostreqkit.OpenRGBNativePolicy, m ResourceManifest, root string) error {
	if m.Name != openRGBResourceName || m.ManagedService == nil || m.ManagedService.Config == nil || m.ManagedService.Artifact.Version != openRGBAppImageVersion {
		return errors.New("native OpenRGB policy requires the exact resource release contract")
	}
	if _, err := p.ID(); err != nil {
		return err
	}
	if err := hostreqkit.ValidateOpenRGBNativeDetectorScope(p); err != nil {
		return err
	}
	if p.Platform != runtime.GOOS {
		return errors.New("native HID policy platform differs from this host")
	}
	artifact, artifactErr := m.ManagedService.Artifact.ForCurrentPlatform()
	profile, profileErr := openRGBReleaseProfileFor(runtime.GOOS, runtime.GOARCH)
	if artifactErr != nil || profileErr != nil || artifact.SHA256 != profile.EnvelopeSHA256 || !profile.acceptsArtifactSHA256(p.ArtifactSHA256) {
		return errors.New("native HID policy artifact is not the pinned platform release")
	}
	data, err := readManagedUpdateConfigFile(filepath.Join(root, "resources", openRGBResourceName), m.ManagedService.Config.ContentFile)
	if err != nil {
		return err
	}
	catalog, err := decodeOpenRGBDetectorCatalog(data)
	if err != nil {
		return err
	}
	if _, err = (OpenRGBDevicePolicyRecord{NativePolicy: &p}).selectedDetectors(catalog); err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != p.DetectorSHA256 {
		return errors.New("native HID policy detector catalog is stale")
	}
	return nil
}

func (c *Controller) StoreOpenRGBNativeDevicePolicy(p hostreqkit.OpenRGBNativePolicy) (string, error) {
	if c == nil || c.Root == "" || c.Home == "" {
		return "", errors.New("native HID policy requires resource owner root and home")
	}
	if err := c.ValidateOpenRGBNativeResourceStorage(context.Background()); err != nil {
		return "", err
	}
	m, err := c.managedServiceArtifactManifest(openRGBResourceName)
	if err != nil {
		return "", err
	}
	if err = validateNativeOpenRGBPolicyManifest(p, m, c.Root); err != nil {
		return "", err
	}
	account, err := c.openRGBNativeRuntimeAccount(context.Background())
	if err != nil || account != p.Account {
		return "", errors.Join(errors.New("native HID enrollment differs from the actual operator"), err)
	}
	machine, err := hostreqkit.NativeOpenRGBMachineIdentity()
	if err != nil || machine != p.MachineID {
		return "", errors.Join(errors.New("native HID enrollment differs from this machine"), err)
	}
	return c.writeOpenRGBNativeDevicePolicy(p)
}

func (c *Controller) LoadOpenRGBNativeDevicePolicy() (hostreqkit.OpenRGBNativePolicy, error) {
	if c == nil {
		return hostreqkit.OpenRGBNativePolicy{}, errors.New("OpenRGB resource controller is unavailable")
	}
	m, err := c.managedServiceArtifactManifest(openRGBResourceName)
	if err != nil {
		return hostreqkit.OpenRGBNativePolicy{}, err
	}
	record, err := c.loadOpenRGBDevicePolicy(m)
	if err != nil {
		return hostreqkit.OpenRGBNativePolicy{}, err
	}
	if record == nil || record.NativePolicy == nil {
		return hostreqkit.OpenRGBNativePolicy{}, os.ErrNotExist
	}
	return *record.NativePolicy, nil
}

func (c *Controller) writeOpenRGBNativeDevicePolicy(p hostreqkit.OpenRGBNativePolicy) (string, error) {
	id, err := p.ID()
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(OpenRGBDevicePolicyRecord{SchemaVersion: openRGBNativeDevicePolicySchemaVersion, NativePolicy: &p})
	if err != nil {
		return "", err
	}
	if len(data) > openRGBDevicePolicyMaxBytes {
		return "", errors.New("native HID policy exceeds size limit")
	}
	path, err := openRGBDevicePolicyPath(c.Home)
	if err != nil {
		return "", err
	}
	if err = ensureOpenRGBDevicePolicyDirectory(filepath.Dir(path), true); err != nil {
		return "", err
	}
	release, err := acquireOpenRGBDevicePolicyPublicationLock(context.Background(), path)
	if err != nil {
		return "", err
	}
	defer release()
	if err = writeResourceOwnedFileDurable(path, data, 0o600); err != nil {
		return "", err
	}
	return id, confirmOpenRGBPublicationDurability(filepath.Dir(path))
}
