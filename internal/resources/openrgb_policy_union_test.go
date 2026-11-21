package resources

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func nativeResourcePolicyFixture(t *testing.T) (hostreqkit.OpenRGBNativePolicy, []byte, []byte) {
	t.Helper()
	catalog, err := os.ReadFile(filepath.Join(projectRootForResourcesTest(t), "resources", "openrgb", "config", "detectors-release_1.1.json"))
	if err != nil {
		t.Fatal(err)
	}
	config, err := mergeOpenRGBConfigContentWithSelection(nil, string(catalog), []string{"ASUS Aura Motherboard"})
	if err != nil {
		t.Fatal(err)
	}
	settings, err := enforceNativeOpenRGBSettings([]byte(config))
	if err != nil {
		t.Fatal(err)
	}
	return hostreqkit.OpenRGBNativePolicy{Platform: "darwin", Account: "uid:501", MachineID: strings.Repeat("a", 64), ArtifactSHA256: strings.Repeat("b", 64), DetectorSHA256: fmt.Sprintf("%x", sha256.Sum256(catalog)), Devices: []hostreqkit.OpenRGBNativeDevice{{Detector: "ASUS Aura Motherboard", Kind: "hid", StableID: "IOService:/usb/port8/interface0", HardwareID: "0b05:19af", UsagePage: 0xff72, Usage: 0xa1}}}, catalog, settings
}

func TestOpenRGBNativePolicyFreezeBindsAccountAndExactDetectorConfig(t *testing.T) {
	p, catalog, config := nativeResourcePolicyFixture(t)
	binding, err := freezeOpenRGBNativeExecPolicy(p, p.ArtifactSHA256, catalog, config)
	if err != nil {
		t.Fatal(err)
	}
	if binding.Platform != p.Platform || binding.Account != p.Account || !binding.valid() {
		t.Fatal("native account binding lost")
	}
	other := p
	other.Account = "uid:502"
	different, err := freezeOpenRGBNativeExecPolicy(other, p.ArtifactSHA256, catalog, config)
	if err != nil || different.ID == binding.ID {
		t.Fatal("runtime account substitution retains policy identity")
	}
	unsafe, err := mergeOpenRGBConfigContentWithSelection(nil, string(catalog), []string{"ASUS Aura Motherboard", "ENE SMBus DRAM"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = freezeOpenRGBNativeExecPolicy(p, p.ArtifactSHA256, catalog, []byte(unsafe)); err == nil {
		t.Fatal("privileged detector configuration accepted")
	}
	role := openRGBFrozenUpdateRole{NativePolicy: &p}
	role.Policy.MachineID = "ambiguous-linux"
	if _, _, _, err = role.policyIdentity(); err == nil {
		t.Fatal("ambiguous frozen Linux/native union admitted")
	}
}

func TestOpenRGBNativeHIDPolicyRequiresOneShotDetection(t *testing.T) {
	p, catalog, settings := nativeResourcePolicyFixture(t)
	for _, value := range []string{"false", "null", "absent"} {
		var root map[string]json.RawMessage
		if err := json.Unmarshal(settings, &root); err != nil {
			t.Fatal(err)
		}
		var detectors map[string]json.RawMessage
		if err := json.Unmarshal(root["Detectors"], &detectors); err != nil {
			t.Fatal(err)
		}
		if value == "absent" {
			delete(detectors, "hid_safe_mode")
		} else {
			detectors["hid_safe_mode"] = json.RawMessage(value)
		}
		root["Detectors"], _ = json.Marshal(detectors)
		unsafe, _ := json.Marshal(root)
		if _, err := freezeOpenRGBNativeExecPolicy(p, p.ArtifactSHA256, catalog, unsafe); err == nil {
			t.Fatal("upstream hotplug path admitted without one-shot safe HID flag")
		}
		safe, err := enforceNativeOpenRGBSettings(unsafe)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = freezeOpenRGBNativeExecPolicy(p, p.ArtifactSHA256, catalog, safe); err != nil {
			t.Fatal("one-shot native settings rejected", err)
		}
	}
}

func TestOpenRGBNativePolicyRecordRoundTripAndLegacyVersionSeparation(t *testing.T) {
	p, _, _ := nativeResourcePolicyFixture(t)
	c := NewController(projectRootForResourcesTest(t), t.TempDir())
	id, err := c.writeOpenRGBNativeDevicePolicy(p)
	if err != nil {
		t.Fatal(err)
	}
	r, err := c.readOpenRGBDevicePolicyRecord()
	if err != nil || r == nil || r.SchemaVersion != 2 || r.NativePolicy == nil {
		t.Fatalf("native record: %+v %v", r, err)
	}
	actual, err := r.ID()
	if err != nil || actual != id {
		t.Fatal("native policy changed through persistence")
	}
	path, err := openRGBDevicePolicyPath(c.Home)
	if err != nil {
		t.Fatal(err)
	}
	for _, wire := range []string{
		`{"schema_version":1,"policy":{},"native_policy":null}`,
		`{"schema_version":1,"policy":{},"Native_Policy":null}`,
		`{"schema_version":1,"policy":{},"NATIVE_POLICY":null}`,
		`{"schema_version":1,"policy":{},"native_policy":null,"Native_Policy":null}`,
		`{"schema_version":2,"policy":{},"native_policy":null}`,
		`{"schema_version":3,"policy":{}}`,
	} {
		if err = os.WriteFile(path, []byte(wire), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err = c.readOpenRGBDevicePolicyRecord(); err == nil {
			t.Fatalf("invalid union/version admitted: %s", wire)
		}
	}
	// Legacy v1 wire shape remains exactly its prior public field set.
	data, err := json.Marshal(OpenRGBDevicePolicyRecord{SchemaVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "native_policy") {
		t.Fatal("Linux v1 wire shape changed")
	}
}

func TestOpenRGBNativeSetupPlanBindsRuntimePrincipalSeparatelyFromOperator(t *testing.T) {
	p, _, _ := nativeResourcePolicyFixture(t)
	stored := openRGBDeviceSetupStoredPlan{OwnerAccount: "uid:1000", NativePolicy: &p, StateDir: "private-resource-state", DeviceIDs: []string{"selected"}}
	id, err := nativeOpenRGBSetupPlanID(stored)
	if err != nil {
		t.Fatal(err)
	}
	stored.Plan.ID = id
	again, err := nativeOpenRGBSetupPlanID(stored)
	if err != nil || id != again {
		t.Fatal("stored plan identity changed when named")
	}
	copy := stored
	copy.OwnerAccount = "uid:1001"
	different, err := nativeOpenRGBSetupPlanID(copy)
	if err != nil || different == id {
		t.Fatal("operator substitution keeps plan identity")
	}
	copy = stored
	previous := p
	copy.PreviousNativePolicy = &previous
	if _, err = nativeOpenRGBSetupPlanID(copy); err == nil {
		t.Fatal("previous policy missing corresponding active plan admitted")
	}
}

func TestNativeTerminalPolicyEnablesOnlyItsRegisteredDetector(t *testing.T) {
	p, catalog, motherboardSettings := nativeResourcePolicyFixture(t)
	p.Devices = []hostreqkit.OpenRGBNativeDevice{{Detector: "ASUS ROG AURA Terminal", Kind: "hid", StableID: "external", HardwareID: "0b05:1889"}}
	terminalConfig, err := mergeOpenRGBConfigContentWithSelection(nil, string(catalog), []string{"ASUS ROG AURA Terminal"})
	if err != nil {
		t.Fatal(err)
	}
	settings, err := enforceNativeOpenRGBSettings([]byte(terminalConfig))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = freezeOpenRGBNativeExecPolicy(p, p.ArtifactSHA256, catalog, settings); err != nil {
		t.Fatal("selected external settings refused", err)
	}
	if _, err = freezeOpenRGBNativeExecPolicy(p, p.ArtifactSHA256, catalog, motherboardSettings); err == nil {
		t.Fatal("external policy accepted motherboard detector settings")
	}
	r := OpenRGBDevicePolicyRecord{NativePolicy: &p}
	got, err := r.selectedDetectors(map[string]bool{"ASUS Aura Motherboard": false, "ASUS ROG AURA Terminal": false})
	if err != nil || len(got) != 1 || got[0] != "ASUS ROG AURA Terminal" {
		t.Fatalf("external selection enabled wrong detector: %v %v", got, err)
	}
	if _, err = r.selectedDetectors(map[string]bool{"ASUS Aura Motherboard": false}); err == nil {
		t.Fatal("missing pinned registration accepted")
	}
}
