package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// Exercise the persisted policy through the ordinary config constructor, rather
// than proving only the catalog-selection helper. No device is opened here.
func TestOpenRGBExternalSelectionReplacesPriorDetectorThroughConfigWriter(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the persisted Linux selector route is exercised on Linux")
	}
	root, home, manifest, env, configPath := openRGBConfigFixture(t)
	c := &Controller{Root: root, Home: home}
	asset, err := os.ReadFile(filepath.Join(root, "resources", openRGBResourceName, filepath.FromSlash(manifest.ManagedService.Config.ContentFile)))
	if err != nil {
		t.Fatal(err)
	}
	var pinned struct {
		Detectors struct {
			Detectors map[string]bool `json:"detectors"`
		} `json:"Detectors"`
	}
	if err := json.Unmarshal(asset, &pinned); err != nil {
		t.Fatal(err)
	}
	p := hostreqkit.OpenRGBLinuxPolicy{
		MachineID: strings.Repeat("a", 32), ArtifactSHA256: manifest.ManagedService.Artifact.SHA256ByPlatform[runtimePlatformKey()],
		DetectorSHA256: updateTestDigest(string(asset)), RuleSHA256: strings.Repeat("d", 64), GroupID: 1500,
		Devices: []hostreqkit.OpenRGBLinuxDevice{{Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/usb1/1-8/1-8:1.0", HardwareID: "0b05:19af:00"}},
	}
	assertOnly := func(want string) []byte {
		t.Helper()
		if err := writeManagedServiceConfig(c, manifest, env); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			Detectors struct {
				Detectors map[string]bool `json:"detectors"`
			} `json:"Detectors"`
		}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if len(decoded.Detectors.Detectors) != len(pinned.Detectors.Detectors) || !decoded.Detectors.Detectors[want] {
			t.Fatalf("selected detector %q absent from pinned catalog", want)
		}
		for name := range pinned.Detectors.Detectors {
			enabled, exists := decoded.Detectors.Detectors[name]
			if !exists || enabled != (name == want) {
				t.Fatalf("detector %q enabled=%t after selection of %q", name, enabled, want)
			}
		}
		assertOpenRGBDynamicDevicesDisabled(t, data)
		return data
	}
	if _, err := c.StoreOpenRGBDevicePolicy(p); err != nil {
		t.Fatal(err)
	}
	assertOnly("ASUS Aura Motherboard")
	p.Devices = []hostreqkit.OpenRGBLinuxDevice{{Detector: "ASUS ROG AURA Terminal", Kind: "hidraw", Topology: "pci0000:00/usb1/1-5/1-5:1.0", HardwareID: "0b05:1889:00"}}
	if _, err := c.StoreOpenRGBDevicePolicy(p); err != nil {
		t.Fatal(err)
	}
	before := assertOnly("ASUS ROG AURA Terminal")
	// An unqualified policy must fail before replacing the already valid file.
	p.DetectorSHA256 = strings.Repeat("f", 64)
	if _, err := c.StoreOpenRGBDevicePolicy(p); err == nil {
		t.Fatal("stale detector catalog admitted")
	}
	after := assertOnly("ASUS ROG AURA Terminal")
	if string(before) != string(after) {
		t.Fatal("rejected policy changed persisted detector configuration")
	}
}
