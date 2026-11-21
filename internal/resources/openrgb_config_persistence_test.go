package resources

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func TestOpenRGBConfigSeedsPinnedPolicyOnFirstLaunch(t *testing.T) {
	root, home, manifest, env, path := openRGBConfigFixture(t)
	controller := &Controller{Root: root, Home: home}
	if err := writeManagedServiceConfig(controller, manifest, env); err != nil {
		t.Fatalf("seed OpenRGB config: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read seeded config: %v", err)
	}
	assertOpenRGBDetectorPolicy(t, got)
	assertOpenRGBDynamicDevicesDisabled(t, got)
	policyPath, err := openRGBDevicePolicyPath(home)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(policyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("default config path created or probed a machine selection: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 600", info.Mode().Perm())
	}
}

func TestOpenRGBConfigEnablesOnlyDetectorsInExplicitMachineLocalPolicy(t *testing.T) {
	root, home, manifest, env, configPath := openRGBConfigFixture(t)
	controller := &Controller{Root: root, Home: home}
	assetPath := filepath.Join(root, "resources", openRGBResourceName, filepath.FromSlash(manifest.ManagedService.Config.ContentFile))
	asset, err := os.ReadFile(assetPath)
	if err != nil {
		t.Fatal(err)
	}
	policy := hostreqkit.OpenRGBLinuxPolicy{
		MachineID: strings.Repeat("a", 32), ArtifactSHA256: manifest.ManagedService.Artifact.SHA256ByPlatform["linux-amd64"], DetectorSHA256: updateTestDigest(string(asset)),
		RuleSHA256: strings.Repeat("d", 64), GroupID: 1500,
		Devices: []hostreqkit.OpenRGBLinuxDevice{
			{Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/0000:03:00.0/usb1/1-8/1-8:1.0", HardwareID: "0b05:19af:00"},
			{Detector: "ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC", Kind: "i2c", Topology: "pci0000:00/0000:01:00.0", HardwareID: "10de:2705:1043:8957", Adapter: "NVIDIA I2C adapter 1"},
		},
	}
	policyID, err := controller.StoreOpenRGBDevicePolicy(policy)
	if err != nil {
		t.Fatalf("store explicit machine-local selector: %v", err)
	}
	loaded, err := controller.LoadOpenRGBDevicePolicy()
	if err != nil {
		t.Fatal(err)
	}
	loadedID, err := loaded.ID()
	if err != nil || loadedID != policyID {
		t.Fatalf("loaded policy identity = %q, %v; want %q", loadedID, err, policyID)
	}
	policyPath, err := openRGBDevicePolicyPath(home)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(policyPath)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("machine-local policy mode = %v, err=%v; want 0600", info, err)
	}
	dirInfo, err := os.Stat(filepath.Dir(policyPath))
	if err != nil || dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("machine-local policy directory mode = %v, err=%v; want 0700", dirInfo, err)
	}
	if err := writeManagedServiceConfig(controller, manifest, env); err != nil {
		t.Fatalf("render explicit selected detector config: %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Detectors struct {
			Detectors map[string]bool `json:"detectors"`
		} `json:"Detectors"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	for name, enabled := range got.Detectors.Detectors {
		want := name == "ASUS Aura Motherboard" || name == "ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC"
		if enabled != want {
			t.Errorf("detector %q enabled=%t, want %t from exact machine-local selectors", name, enabled, want)
		}
	}
}

func TestOpenRGBConfigFirstLaunchFailureLeavesNoPartialFileAndCanRecover(t *testing.T) {
	root, home, manifest, env, path := openRGBConfigFixture(t)
	values := managedServiceEnvValues(env)
	ambientPath := filepath.Join(home, ".config", "OpenRGB", "OpenRGB.json")
	ambientBefore := []byte(`{"UserInterface":{"language":"ambient-preserved"}}`)
	if err := os.MkdirAll(filepath.Dir(ambientPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ambientPath, ambientBefore, 0o600); err != nil {
		t.Fatal(err)
	}
	pinned, err := os.ReadFile(filepath.Join(root, "resources", "openrgb", filepath.FromSlash(manifest.ManagedService.Config.ContentFile)))
	if err != nil {
		t.Fatal(err)
	}
	manifest.ManagedService.Config.ContentFile = ""
	manifest.ManagedService.Config.Content = "{invalid pinned config"
	if err := writeManagedServiceConfig(&Controller{Root: root, Home: home}, manifest, env); err == nil {
		t.Fatal("expected invalid first-launch seed to fail")
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("failed first launch left a config artifact: stat err=%v", err)
	}
	manifest.ManagedService.Config.Content = string(pinned)
	if err := writeManagedServiceConfig(&Controller{Root: root, Home: home}, manifest, env); err != nil {
		t.Fatalf("first launch after repairing the seed did not recover: %v", err)
	}
	seeded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	assertOpenRGBDetectorPolicy(t, seeded)
	assertOpenRGBDynamicDevicesDisabled(t, seeded)
	ambientAfter, err := os.ReadFile(ambientPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ambientAfter, ambientBefore) {
		t.Fatalf("first-launch failure/recovery changed ambient config: got %q, want %q", ambientAfter, ambientBefore)
	}
	if got := filepath.Clean(values["XDG_CONFIG_HOME"]); got != filepath.Clean(values["RESOURCE_CONFIG_DIR"]) {
		t.Fatalf("constructor config root = %q, resource config root = %q", got, values["RESOURCE_CONFIG_DIR"])
	}
}

func TestOpenRGBConfigPreservesUserSettingsAndReappliesPinnedDetectors(t *testing.T) {
	root, home, manifest, env, path := openRGBConfigFixture(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	userConfig := []byte(`{
  "UserInterface": {"language": "fr", "theme": {"accent": "purple"}},
  "Server": {"default_port": 12345, "default_host": "127.0.0.2"},
  "Detectors": {
    "hid_safe_mode": true,
    "initial_detection_delay_ms": 950,
    "detectors": {"unlisted detector": true, "ASUS Aura SMBus Motherboard": false}
  },
  "QMKOpenRGBDevices": {"devices": [{"name":"extra HID","usb_vid":"1234","usb_pid":"5678"}], "leds_per_update": 7, "delay": 4},
  "QMKVialRGBDevices": {"devices": [{"name":"extra Vial HID","usb_vid":"1234","usb_pid":"5678"}]}
}`)
	if err := os.WriteFile(path, userConfig, 0o600); err != nil {
		t.Fatal(err)
	}
	for restart := 1; restart <= 2; restart++ {
		if err := writeManagedServiceConfig(&Controller{Root: root, Home: home}, manifest, env); err != nil {
			t.Fatalf("config update on managed-service start %d: %v", restart, err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(got, &config); err != nil {
		t.Fatalf("decode resulting config: %v", err)
	}
	var userInterface map[string]any
	if err := json.Unmarshal(config["UserInterface"], &userInterface); err != nil {
		t.Fatal(err)
	}
	if userInterface["language"] != "fr" {
		t.Fatalf("language setting = %#v, want preserved value fr", userInterface["language"])
	}
	if theme, ok := userInterface["theme"].(map[string]any); !ok || theme["accent"] != "purple" {
		t.Fatalf("nested theme setting was not preserved: %#v", userInterface["theme"])
	}
	var server map[string]any
	if err := json.Unmarshal(config["Server"], &server); err != nil {
		t.Fatal(err)
	}
	if server["default_port"] != float64(12345) || server["default_host"] != "127.0.0.2" {
		t.Fatalf("server settings were not preserved: %#v", server)
	}
	var detectorSettings map[string]json.RawMessage
	if err := json.Unmarshal(config["Detectors"], &detectorSettings); err != nil {
		t.Fatal(err)
	}
	var safeMode bool
	if err := json.Unmarshal(detectorSettings["hid_safe_mode"], &safeMode); err != nil || !safeMode {
		t.Fatalf("non-detector HID safety setting was not preserved: value=%t err=%v", safeMode, err)
	}
	var detectionDelay int
	if err := json.Unmarshal(detectorSettings["initial_detection_delay_ms"], &detectionDelay); err != nil || detectionDelay != 950 {
		t.Fatalf("non-detector delay setting was not preserved: value=%d err=%v", detectionDelay, err)
	}
	var qmkSettings map[string]json.RawMessage
	if err := json.Unmarshal(config["QMKOpenRGBDevices"], &qmkSettings); err != nil {
		t.Fatal(err)
	}
	var qmkDevices []json.RawMessage
	if err := json.Unmarshal(qmkSettings["devices"], &qmkDevices); err != nil || len(qmkDevices) != 0 {
		t.Fatalf("QMK dynamic HID registrations remain enabled: devices=%d err=%v", len(qmkDevices), err)
	}
	var delay, ledsPerUpdate int
	if err := json.Unmarshal(qmkSettings["delay"], &delay); err != nil || delay != 4 {
		t.Fatalf("QMK delay tuning was not preserved: value=%d err=%v", delay, err)
	}
	if err := json.Unmarshal(qmkSettings["leds_per_update"], &ledsPerUpdate); err != nil || ledsPerUpdate != 7 {
		t.Fatalf("QMK update tuning was not preserved: value=%d err=%v", ledsPerUpdate, err)
	}
	var vialSettings map[string]json.RawMessage
	if err := json.Unmarshal(config["QMKVialRGBDevices"], &vialSettings); err != nil {
		t.Fatal(err)
	}
	var vialDevices []json.RawMessage
	if err := json.Unmarshal(vialSettings["devices"], &vialDevices); err != nil || len(vialDevices) != 0 {
		t.Fatalf("VialRGB dynamic HID registrations remain enabled: devices=%d err=%v", len(vialDevices), err)
	}
	assertOpenRGBDetectorPolicy(t, got)
}

func TestOpenRGBConfigFailureLeavesExistingBytesUntouched(t *testing.T) {
	root, home, manifest, env, path := openRGBConfigFixture(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	before := []byte("{this is the user's recoverable config")
	if err := os.WriteFile(path, before, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeManagedServiceConfig(&Controller{Root: root, Home: home}, manifest, env); err == nil {
		t.Fatal("expected malformed existing config to be rejected")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("failed config update changed existing bytes: got %q, want %q", after, before)
	}
	if err := os.WriteFile(path, []byte(`{"UserInterface":{"language":"repaired"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeManagedServiceConfig(&Controller{Root: root, Home: home}, manifest, env); err != nil {
		t.Fatalf("retry after repairing config: %v", err)
	}
	recovered, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var recoveredConfig map[string]json.RawMessage
	if err := json.Unmarshal(recovered, &recoveredConfig); err != nil {
		t.Fatal(err)
	}
	var ui map[string]any
	if err := json.Unmarshal(recoveredConfig["UserInterface"], &ui); err != nil || ui["language"] != "repaired" {
		t.Fatalf("repaired user setting not preserved after recovery: %#v, err=%v", ui, err)
	}
	assertOpenRGBDetectorPolicy(t, recovered)
}

func TestOpenRGBConfigRejectsSymlinkedDirectoryOutsideResourceRoot(t *testing.T) {
	root, home, manifest, env, path := openRGBConfigFixture(t)
	values := managedServiceEnvValues(env)
	ambient := filepath.Join(home, "ambient-openrgb")
	if err := os.MkdirAll(ambient, 0o700); err != nil {
		t.Fatal(err)
	}
	ambientConfig := filepath.Join(ambient, "OpenRGB.json")
	before := []byte(`{"UserInterface":{"language":"keep"}}`)
	if err := os.WriteFile(ambientConfig, before, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(values["RESOURCE_CONFIG_DIR"], 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(ambient, filepath.Dir(path)); err != nil {
		t.Fatal(err)
	}
	if err := writeManagedServiceConfig(&Controller{Root: root, Home: home}, manifest, env); err == nil {
		t.Fatal("expected config directory symlink escape to be rejected")
	}
	after, err := os.ReadFile(ambientConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("ambient config changed through resource path: got %q, want %q", after, before)
	}
}

func TestOpenRGBConfigRejectsResourceRootAliasedToAmbientXDGConfig(t *testing.T) {
	root, home, manifest, env, path := openRGBConfigFixture(t)
	values := managedServiceEnvValues(env)
	ambientRoot := filepath.Join(home, ".config")
	ambientConfig := filepath.Join(ambientRoot, "OpenRGB", "OpenRGB.json")
	before := []byte(`{"UserInterface":{"language":"keep"}}`)
	if err := os.MkdirAll(filepath.Dir(ambientConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ambientConfig, before, 0o600); err != nil {
		t.Fatal(err)
	}
	resourceConfigRoot := values["RESOURCE_CONFIG_DIR"]
	if err := os.MkdirAll(filepath.Dir(resourceConfigRoot), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(ambientRoot, resourceConfigRoot); err != nil {
		t.Fatal(err)
	}
	if err := writeManagedServiceConfig(&Controller{Root: root, Home: home}, manifest, env); err == nil {
		t.Fatalf("expected resource config root %q to be rejected for aliasing ambient config %q", path, ambientConfig)
	}
	after, err := os.ReadFile(ambientConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("ambient config changed through aliased resource root: got %q, want %q", after, before)
	}
}

func openRGBConfigFixture(t *testing.T) (root, home string, manifest ResourceManifest, env []string, path string) {
	t.Helper()
	root = projectRootForResourcesTest(t)
	home = t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(home, ".vrooli"))
	loaded, err := manifestpkg.Load(filepath.Join(root, "resources", "openrgb", "resource.json"))
	if err != nil {
		t.Fatalf("load OpenRGB manifest: %v", err)
	}
	manifest = loaded
	env = resourceEnvForResource(root, home, "openrgb")
	env = renderManagedServiceEnvironment(env, manifest.ManagedService.Environment)
	values := managedServiceEnvValues(env)
	path = filepath.Join(values["RESOURCE_CONFIG_DIR"], filepath.FromSlash(manifest.ManagedService.Config.Path))
	return
}

func assertOpenRGBDetectorPolicy(t *testing.T, data []byte) {
	t.Helper()
	var got struct {
		Detectors struct {
			Detectors map[string]bool `json:"detectors"`
		} `json:"Detectors"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode OpenRGB detector config: %v", err)
	}
	asset, err := os.ReadFile(filepath.Join("..", "..", "resources", "openrgb", "config", "detectors-release_1.1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var want struct {
		Detectors struct {
			Detectors map[string]bool `json:"detectors"`
		} `json:"Detectors"`
	}
	if err := json.Unmarshal(asset, &want); err != nil {
		t.Fatal(err)
	}
	if len(got.Detectors.Detectors) != len(want.Detectors.Detectors) {
		t.Fatalf("detector count = %d, want pinned count %d", len(got.Detectors.Detectors), len(want.Detectors.Detectors))
	}
	for name, enabled := range want.Detectors.Detectors {
		value, ok := got.Detectors.Detectors[name]
		if !ok || value != enabled {
			t.Errorf("detector %q = (%t, present=%t), want pinned value %t", name, value, ok, enabled)
		}
	}
}

func assertOpenRGBDynamicDevicesDisabled(t *testing.T, data []byte) {
	t.Helper()
	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("decode OpenRGB config: %v", err)
	}
	for _, key := range []string{"QMKOpenRGBDevices", "QMKVialRGBDevices"} {
		var settings map[string]json.RawMessage
		if err := json.Unmarshal(config[key], &settings); err != nil || settings == nil {
			t.Errorf("dynamic detector settings %q missing or invalid: err=%v", key, err)
			continue
		}
		var devices []json.RawMessage
		if err := json.Unmarshal(settings["devices"], &devices); err != nil || len(devices) != 0 {
			t.Errorf("dynamic detector devices %q = %d; want explicit empty array (err=%v)", key, len(devices), err)
		}
	}
}
