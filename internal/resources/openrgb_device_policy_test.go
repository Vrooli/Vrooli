package resources

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func TestOpenRGBSelectedPolicyBindsUSBInterfaceAndPCIAdapter(t *testing.T) {
	root := projectRootForResourcesTest(t)
	data, err := os.ReadFile(filepath.Join(root, "resources", "openrgb", "config", "detectors-release_1.1.json"))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := decodeOpenRGBDetectorCatalog(data)
	if err != nil {
		t.Fatal(err)
	}
	policy := hostreqkit.OpenRGBLinuxPolicy{
		MachineID: strings.Repeat("a", 32), ArtifactSHA256: strings.Repeat("b", 64), DetectorSHA256: fmt.Sprintf("%x", sha256.Sum256(data)), RuleSHA256: strings.Repeat("d", 64), GroupID: 1500,
		Devices: []hostreqkit.OpenRGBLinuxDevice{
			{Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/0000:03:00.0/usb1/1-8/1-8:1.0", HardwareID: "0b05:19af:00"},
			{Detector: "ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC", Kind: "i2c", Topology: "pci0000:00/0000:01:00.0", HardwareID: "10de:2705:1043:8957", Adapter: "NVIDIA I2C adapter 1"},
		},
	}
	got, err := selectedOpenRGBDetectors(policy, catalog)
	if err != nil || !slices.Equal(got, []string{"ASUS Aura Motherboard", "ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC"}) {
		t.Fatalf("selected detectors = %v, err=%v", got, err)
	}
	if _, err := policy.ID(); err != nil {
		t.Fatalf("exact USB/PCI policy ID rejected: %v", err)
	}
}

func TestOpenRGBSelectedPolicyRejectsUnsafeOrUnqualifiedSelectors(t *testing.T) {
	root := projectRootForResourcesTest(t)
	data, err := os.ReadFile(filepath.Join(root, "resources", "openrgb", "config", "detectors-release_1.1.json"))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := decodeOpenRGBDetectorCatalog(data)
	if err != nil {
		t.Fatal(err)
	}
	base := hostreqkit.OpenRGBLinuxPolicy{MachineID: strings.Repeat("a", 32), ArtifactSHA256: strings.Repeat("b", 64), DetectorSHA256: fmt.Sprintf("%x", sha256.Sum256(data)), RuleSHA256: strings.Repeat("d", 64), GroupID: 1500}
	for _, tc := range []struct {
		name   string
		device hostreqkit.OpenRGBLinuxDevice
	}{
		{name: "ASUS USB ID cannot fall back to SMBus", device: hostreqkit.OpenRGBLinuxDevice{Detector: "ASUS Aura SMBus Motherboard", Kind: "i2c", Topology: "pci0000:00/0000:03:00.0", HardwareID: "0b05:19af:00", Adapter: "ASUS SMBus adapter"}},
		{name: "GPU cannot use dynamic adapter number", device: hostreqkit.OpenRGBLinuxDevice{Detector: "ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC", Kind: "i2c", Topology: "pci0000:00/0000:01:00.0", HardwareID: "10de:2705:1043:8957"}},
		{name: "write-capable ENE SMBus detector", device: hostreqkit.OpenRGBLinuxDevice{Detector: "ENE SMBus DRAM", Kind: "i2c", Topology: "pci0000:00/0000:03:00.0", HardwareID: "1234:5678:0001:0002", Adapter: "fixture adapter"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			policy := base
			policy.Devices = []hostreqkit.OpenRGBLinuxDevice{tc.device}
			if _, err := selectedOpenRGBDetectors(policy, catalog); err == nil {
				t.Fatal("unsafe or unqualified device selector was accepted")
			}
		})
	}
}

func TestOpenRGBPolicyStoreRejectsStaleArtifactDigest(t *testing.T) {
	root := projectRootForResourcesTest(t)
	controller := NewController(root, t.TempDir())
	policy := hostreqkit.OpenRGBLinuxPolicy{ArtifactSHA256: strings.Repeat("b", 64)}
	if _, err := controller.StoreOpenRGBDevicePolicy(policy); err == nil {
		t.Fatal("policy with stale artifact digest was stored")
	}
}
