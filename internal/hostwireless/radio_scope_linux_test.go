//go:build linux

package hostwireless

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"slices"
	"testing"
)

func TestRadioBluetoothControllerUsesExactUSBInterfaceComponent(t *testing.T) {
	f := radioTestFixture(t)
	for _, name := range radioModuleNames(NobleBluetoothProfile) {
		f.loaded[name] = "candidate"
	}
	for name, link := range map[string]string{"hci0": "../../../11-6/11-6:1.0", "hci1": "../../../1-6/1-6:1.0"} {
		if err := f.target.root.MkdirAll("sys/class/bluetooth/"+name, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := f.target.root.Symlink(link, "sys/class/bluetooth/"+name+"/device"); err != nil {
			t.Fatal(err)
		}
	}
	r, err := observeWirelessRadio(context.Background(), f.owner)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Controllers, []string{"hci1"}) {
		t.Fatalf("attributed sibling USB bus to selected controller: %v", r.Controllers)
	}
	if selectedUSBInterfaceLink("../../../11-6:1.0", "1-6") {
		t.Fatal("substring matched a different USB bus")
	}
}

func TestRadioPresentUnqualifiedSiblingBlocksSharedDriverUnload(t *testing.T) {
	f := radioTestFixture(t)
	if err := f.target.root.MkdirAll("sys/bus/usb/devices/2-7:1.0", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := f.target.root.Symlink("../../../../drivers/btusb", "sys/bus/usb/devices/2-7:1.0/driver"); err != nil {
		t.Fatal(err)
	}
	if err := unloadRadioCohort(context.Background(), f.owner); err == nil {
		t.Fatal("present unreadable modalias escaped shared-driver scope")
	}
	if len(f.calls) != 0 {
		t.Fatal("unloaded driver before complete consumer inventory")
	}
}

func TestWirelessJournalReadOnlyOpenHasNoStateCreation(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if owner, err := openAdmittedDurableWireless(context.Background(), root, nil, true); err == nil {
		owner.close()
		t.Fatal("missing root journal accepted")
	}
	if _, err = root.Stat("var"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("read-only status created a state directory", err)
	}
}

func TestWirelessNamespaceReadOnlyCurrentDistribution(t *testing.T) {
	if os.Getenv("VROOLI_WIRELESS_RADIO_OBSERVE_LIVE") != "1" {
		t.Skip("explicit read-only native observation required")
	}
	root, err := os.OpenRoot("/")
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	paths, err := admitWirelessRootPaths(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if paths["lib"] != "usr/lib" {
		t.Fatal("current usr merge was not explicitly resolved")
	}
	for _, module := range []string{"btusb", "btmtk"} {
		identity := observeProvider(context.Background(), nobleABI, module)
		if identity.Issue != "" {
			t.Fatal(identity.Issue)
		}
		data, err := os.ReadFile(identity.Path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := moduleELF(context.Background(), data, identity.Path)
		if err != nil {
			t.Fatal(err)
		}
		match, err := moduleLoadedMatches(root, module, file)
		if err != nil || !match {
			t.Fatalf("native stock module identity mismatch %s: %v", module, err)
		}
	}
	if _, err = root.Stat(path.Join(wirelessJournalDir, "active.json")); err == nil {
		t.Log("existing live journal present; observation did not open its mutation owner")
	}
}

func TestPassiveBlueZPeerEvidencePreservesSelectedControllerScope(t *testing.T) {
	raw := []byte(`{"type":"a{oa{sa{sv}}}","data":[{"/org/bluez/hci1":{"org.bluez.Adapter1":{"Powered":{"type":"b","data":true}}},"/org/bluez/hci1/dev_AA_BB_CC_DD_EE_FF":{"org.bluez.Device1":{"Paired":{"type":"b","data":true},"Connected":{"type":"b","data":true}}}}]}`)
	if verified, err := blueZPeerEvidence(raw, []string{"hci0"}); err != nil || verified {
		t.Fatalf("unrelated controller verified selected radio: verified=%t err=%v", verified, err)
	}
	if verified, err := blueZPeerEvidence(raw, []string{"hci1"}); err != nil || !verified {
		t.Fatalf("selected powered connected peer lost: verified=%t err=%v", verified, err)
	}
	if verified, err := blueZPeerEvidence([]byte(`{"type":"a{oa{sa{sv}}}","data":[{}]}`), []string{"hci0"}); err != nil || verified {
		t.Fatalf("empty controller inventory became function proof: verified=%t err=%v", verified, err)
	}
}
