package hostwireless

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func TestSelectNobleHost(t *testing.T) {
	const alias = "pci:v000014C3d00007927sv00001A3Bsd00006113bc02sc80i00"
	for _, tc := range []struct {
		name, distro, abi, alias, want string
		probe                          hostinventory.IntegrityProbeState
		observed                       bool
	}{
		{"observed", "noble", nobleABI, alias, "eligible", hostinventory.IntegrityProbeOK, true},
		{"other release", "noble", "7.0.0-42-generic", alias, "unsupported", hostinventory.IntegrityProbeOK, true},
		{"other distribution", "resolute", nobleABI, alias, "unsupported", hostinventory.IntegrityProbeOK, true},
		{"missing device", "noble", nobleABI, "", "unsupported", hostinventory.IntegrityProbeOK, false},
		{"failed probe", "noble", nobleABI, "", "unverified", hostinventory.IntegrityProbeFailed, false},
		{"missing alias", "noble", nobleABI, "", "unverified", hostinventory.IntegrityProbeOK, true},
		{"contradictory alias", "noble", nobleABI, "pci:v000014C3d00007925", "unverified", hostinventory.IntegrityProbeOK, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inventory := hostinventory.HostInventory{
				Kernel:      hostinventory.KernelInfo{Release: tc.abi},
				ProbeStatus: map[string]hostinventory.IntegrityProbeState{"pciDevices": tc.probe},
			}
			if tc.observed {
				inventory.Devices = []hostinventory.DeviceInfo{{ID: "fixture", BusType: "pci", VendorID: "14c3", DeviceID: "7927", Modalias: tc.alias}}
			}
			var r Receipt
			selectNobleHost(&r, []byte("ID=ubuntu\nVERSION_CODENAME="+tc.distro), inventory)
			if r.State != tc.want || r.ActivationEligible {
				t.Fatalf("unexpected receipt %+v", r)
			}
		})
	}
}

func TestUnknownProfileIsReadOnlyAndCannotApply(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, apply := range []bool{false, true} {
		r, err := Prepare(context.Background(), "other", apply)
		if (err != nil) != apply || r.State != "unsupported" || r.ArtifactDirectory != "" || r.ActivationEligible {
			t.Fatalf("apply %v receipt %+v error %v", apply, r, err)
		}
	}
}

func TestBluetoothRequiresExactObservedHCIInterface(t *testing.T) {
	const alias = "usb:v0489pE13Ad0100dcEFdsc02dp01icE0isc01ip01in00"
	for _, tc := range []struct {
		name, id, alias, want string
	}{
		{"matching HCI", "e13a", alias, "eligible"},
		{"neighbor product", "e139", alias, "unsupported"},
		{"other interface", "e13a", strings.Replace(alias, "in00", "in01", 1), "unverified"},
		{"missing function alias", "e13a", "", "unverified"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inventory := hostinventory.HostInventory{
				Kernel:      hostinventory.KernelInfo{Release: nobleABI},
				ProbeStatus: map[string]hostinventory.IntegrityProbeState{"usbDevices": hostinventory.IntegrityProbeOK},
				Devices: []hostinventory.DeviceInfo{{ID: "usb:fixture", BusType: "usb", VendorID: "0489", DeviceID: tc.id,
					Functions: []hostinventory.DeviceFunction{{ID: "fixture:1.0", Modalias: tc.alias}}}},
			}
			r := Receipt{Profile: NobleBluetoothProfile}
			selectNobleHost(&r, []byte("ID=ubuntu\nVERSION_CODENAME=noble"), inventory)
			if r.State != tc.want || r.ActivationEligible || len(r.PCIDevices) != 0 {
				t.Fatalf("unexpected receipt %+v", r)
			}
		})
	}
}

func TestBluetoothQuirkIsTheOnlyChange(t *testing.T) {
	stock := make([]byte, 64)
	stock[0] = 7
	entry := [32]byte{0x03, 0, 0x89, 0x04, 0x3a, 0xe1, 26: 0x30}
	candidate := append(append(append([]byte{}, stock[:32]...), entry[:]...), stock[32:]...)
	if err := compareBluetoothQuirk(stock, candidate); err != nil {
		t.Fatal(err)
	}
	candidate[0] ^= 1
	if err := compareBluetoothQuirk(stock, candidate); err == nil {
		t.Fatal("accepted an unrelated quirk change")
	}
	candidate[0] ^= 1
	candidate[32+26] = 0x10
	if err := compareBluetoothQuirk(stock, candidate); err == nil {
		t.Fatal("accepted altered MediaTek/wideband flags")
	}
}

func TestPublishCandidateHashesExactBytes(t *testing.T) {
	root := t.TempDir()
	bundle, path := filepath.Join(root, "bundle"), filepath.Join(root, "published")
	if err := os.Mkdir(bundle, 0700); err != nil {
		t.Fatal(err)
	}
	const name = "mt7925e.ko"
	data := []byte("unsigned module")
	writeFixture(t, filepath.Join(bundle, name), string(data))
	r, err := publishCandidate(context.Background(), Receipt{CopiedTreeRollback: true}, bundle, path)
	if err != nil || r.ArtifactSHA256[name] != digest(data) || r.ActivationEligible {
		t.Fatalf("receipt %+v error %v", r, err)
	}
	content, err := os.ReadFile(filepath.Join(path, "receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored Receipt
	if err := json.Unmarshal(content, &stored); err != nil || stored.ArtifactSHA256[name] != digest(data) || !stored.CopiedTreeRollback {
		t.Fatalf("stored receipt %+v error %v", stored, err)
	}
	if _, err := os.Stat(bundle); !os.IsNotExist(err) {
		t.Fatalf("temporary bundle remains: %v", err)
	}
}

func TestCopiedTreeBudgetDoesNotFollowBuildLink(t *testing.T) {
	tree := t.TempDir()
	external := t.TempDir()
	writeFixture(t, filepath.Join(external, "header"), strings.Repeat("x", 64<<10))
	writeFixture(t, filepath.Join(tree, "module"), "module")
	if err := os.Symlink(external, filepath.Join(tree, "build")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	got, err := copiedTreeBudget(context.Background(), tree)
	if err != nil || got != 3*8192+int64(len("module")) {
		t.Fatalf("tree budget %d error %v", got, err)
	}
}

func TestVerifyABIRejectsChangedExportsAndImports(t *testing.T) {
	const current = "0x00000002 old_export drivers/mt7925e EXPORT_SYMBOL_GPL NS\n"
	const global = "0x00000001 core_export drivers/mt76 EXPORT_SYMBOL\n"
	const extension = "0x00000003 mt792x_dma_alloc_queues drivers/mt792x-lib EXPORT_SYMBOL_GPL\n" +
		"0x00000004 mt7925_mcu_set_dbdc drivers/mt7925-common EXPORT_SYMBOL_GPL\n"
	for _, tc := range []struct {
		name, exports, imports, wantErr string
		stock                           bool
	}{
		{"stock", current, "0x00000001 core_export\n", "", true},
		{"candidate", current + extension, "0x00000003 mt792x_dma_alloc_queues\n", "", false},
		{"wrong CRC", strings.Replace(current, "00000002", "00000003", 1), "0x00000001 core_export\n", "old_export changed", true},
		{"wrong namespace", strings.Replace(current, " NS", " OTHER", 1), "0x00000001 core_export\n", "old_export changed", false},
		{"bad import", current + extension, "0x00000002 core_export\n", "incompatible import", false},
		{"unexpected provider", current + global, "0x00000001 core_export\n", "unexpected export provider", false},
		{"missing extension", current, "0x00000001 core_export\n", "missing expected new export", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			stock := filepath.Join(dir, "stock")
			writeFixture(t, stock, global+current)
			writeFixture(t, filepath.Join(dir, "Module.symvers"), tc.exports)
			for _, name := range nobleModules {
				writeFixture(t, filepath.Join(dir, name+".modversions"), tc.imports)
			}
			var exports map[string]string
			if !tc.stock {
				exports = preparationProfiles[NobleProfile].newExports
			}
			r, err := verifyABI(stock, dir, nobleModules, exports)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("got %v, want %q", err, tc.wantErr)
				}
			} else if err != nil || r.StockExports != 1 || r.Imports != len(nobleModules) {
				t.Fatalf("proof %+v error %v", r, err)
			}
		})
	}
}

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
