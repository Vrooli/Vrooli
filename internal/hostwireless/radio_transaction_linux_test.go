//go:build linux

package hostwireless

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"testing"
)

const radioFixtureBoot = "11111111-1111-4111-8111-111111111111"

type radioFixture struct {
	owner       *durableWireless
	target      *CopiedActivation
	loaded      map[string]string
	calls       []string
	failRestore bool
	cancelLoad  context.CancelFunc
	unloadCheck func() error
}

func radioTestFixture(t *testing.T) *radioFixture {
	t.Helper()
	plan, baseline, signed, cert := indexedFixture(t, NobleBluetoothProfile)
	route := "Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT\nenp1s0 00000000 0101A8C0 0003 0 0 100 00000000 0 0 0\n"
	identity := identityFixture(baseline, "/proc/net/route", route)
	for i, file := range plan.Preserved {
		if file.Path == identity.Path {
			plan.Preserved[i] = identity
		}
	}
	indexed, err := newIndexedActivation(context.Background(), baseline, plan, fixtureIndexes)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = indexed.close() })
	if _, err = indexed.stage(context.Background(), signed, cert); err != nil {
		t.Fatal(err)
	}
	owner, target := durableTargetFixture(t, indexed)
	f := &radioFixture{owner: owner, target: target, loaded: map[string]string{}}
	f.prepare(t, NobleBluetoothProfile, "usb:1-6")
	f.write(t, "etc/machine-id", strings.Repeat("a", 32))
	owner.admission = &wirelessRootAdmission{} // Exercise the bounded namespace snapshot.
	ctx := &wirelessJournalContext{OperatorUID: uint32(os.Geteuid()), Device: "usb:1-6", BootID: radioFixtureBoot, MachineID: strings.Repeat("a", 32), EthernetRoutes: ethernetRouteLines([]byte(route), plan.WiredInterfaces), Recovery: RecoveryProofRequest{TargetABI: nobleABI, RecoveryRelease: "7.0.0-29-generic", MenuID: "advanced>stock", RequiredEthernetModules: []string{"atlantic"}, RequiredNVIDIAModules: []string{"nvidia"}}}
	ctx.StageModules, err = moduleSnapshots(target.root, NobleBluetoothProfile)
	if err != nil {
		t.Fatal(err)
	}
	owner.context = ctx
	if err = owner.begin(context.Background(), indexed, cert); err != nil {
		t.Fatal(err)
	}
	if err = owner.apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	owner.radio = f.system()
	return f
}

func (f *radioFixture) write(t *testing.T, name, value string) {
	t.Helper()
	if err := f.target.root.MkdirAll(path.Dir(name), 0700); err != nil {
		t.Fatal(err)
	}
	if err := f.target.root.WriteFile(name, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
}

func (f *radioFixture) prepare(t *testing.T, profile, id string) {
	t.Helper()
	f.write(t, "proc/sys/kernel/random/boot_id", radioFixtureBoot)
	f.write(t, "sys/bus/usb/devices/1-6:1.0/modalias", "usb:v0489pE13Ad0100dcEFdsc02dp01icE0isc01ip01in00")
	for _, name := range []string{"nvidia", "atlantic", "bluetooth"} {
		if err := f.target.root.MkdirAll("sys/module/"+name+"/holders", 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range radioModuleNames(profile) {
		f.loaded[name] = "stock"
		if err := f.target.root.MkdirAll("sys/module/"+name+"/holders", 0700); err != nil {
			t.Fatal(err)
		}
		f.write(t, "sys/module/"+name+"/parameters/test_flag", "N")
	}
	_ = f.target.root.Symlink("../../../../drivers/btusb", "sys/bus/usb/devices/1-6:1.0/driver")
}

func (f *radioFixture) system() *wirelessRadioSystem {
	return &wirelessRadioSystem{
		identity: func(_ context.Context, _ *durableWireless, name string) (bool, bool, bool, error) {
			kind := f.loaded[name]
			return kind != "", kind == "candidate", kind == "stock", nil
		},
		aliases: func(context.Context, *durableWireless) ([]string, error) { return []string{"usb:v0489pE13A*"}, nil },
		unload: func(ctx context.Context, name string) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if !slices.Contains(radioModuleNames(f.owner.intent.Plan.Profile), name) {
				return errors.New("attempted unrelated unload")
			}
			if f.unloadCheck != nil {
				if err := f.unloadCheck(); err != nil {
					return err
				}
			}
			f.calls = append(f.calls, "unload:"+name)
			delete(f.loaded, name)
			_ = f.target.root.RemoveAll("sys/module/" + name)
			if name == "btusb" {
				_ = f.target.root.Remove("sys/bus/usb/devices/1-6:1.0/driver")
				_ = f.target.root.RemoveAll("sys/class/bluetooth/hci0")
			}
			return nil
		},
		command: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			f.calls = append(f.calls, name+" "+strings.Join(args, " "))
			if slices.Contains(args, "--show-depends") {
				return []byte("insmod /lib/modules/" + nobleABI + "/kernel/btmtk.ko\ninsmod /lib/modules/" + nobleABI + "/kernel/btusb.ko\n"), nil
			}
			if name != "/usr/sbin/modprobe" {
				return nil, errors.New("unexpected fixture command")
			}
			separator := slices.Index(args, "--")
			if separator < 0 || separator+1 >= len(args) {
				return nil, errors.New("missing closed module argument")
			}
			module := args[separator+1]
			kind := "candidate"
			if f.owner.phase == "rolled_back" {
				kind = "stock"
				if f.failRestore {
					f.failRestore = false
					return nil, errors.New("interrupted stock restoration")
				}
			}
			if f.cancelLoad != nil && kind == "candidate" {
				f.cancelLoad()
				f.cancelLoad = nil
				return nil, context.Canceled
			}
			names := []string{module}
			if kind == "candidate" {
				names = radioModuleNames(f.owner.intent.Plan.Profile)
			}
			for _, loaded := range names {
				f.loaded[loaded] = kind
				_ = f.target.root.MkdirAll("sys/module/"+loaded+"/holders", 0700)
				_ = f.target.root.MkdirAll("sys/module/"+loaded+"/parameters", 0700)
			}
			for _, arg := range args[separator+2:] {
				key, value, ok := strings.Cut(arg, "=")
				if ok {
					_ = f.target.root.WriteFile("sys/module/"+module+"/parameters/"+key, []byte(value), 0600)
				}
			}
			_ = f.target.root.Symlink("../../../../drivers/btusb", "sys/bus/usb/devices/1-6:1.0/driver")
			_ = f.target.root.MkdirAll("sys/class/bluetooth/hci0", 0700)
			_ = f.target.root.Symlink("../../../1-6:1.0", "sys/class/bluetooth/hci0/device")
			return nil, nil
		},
	}
}

func TestRadioActivationRefreshesActualStockSnapshot(t *testing.T) {
	f := radioTestFixture(t)
	f.write(t, "sys/module/btusb/parameters/test_flag", "Y") // Drift after file-stage, before owned activation.
	if err := activateWirelessRadio(context.Background(), f.owner); err != nil {
		t.Fatal(err)
	}
	record, err := f.owner.readRadioRecord()
	if err != nil {
		t.Fatal(err)
	}
	if record.Before[1].Parameters["test_flag"] != "Y" || record.BootID != radioFixtureBoot {
		t.Fatal("activation retained stale stage snapshot")
	}
	if err = rollbackWirelessRadio(context.Background(), f.owner); err != nil {
		t.Fatal(err)
	}
	value, err := radioRead(f.target.root, "sys/module/btusb/parameters/test_flag", 512)
	if err != nil || string(value) != "Y" {
		t.Fatal("fresh original parameters not restored", err)
	}
	for _, name := range []string{"nvidia", "atlantic", "bluetooth"} {
		if _, err = f.target.root.Stat("sys/module/" + name); err != nil {
			t.Fatal("unrelated module removed", name)
		}
	}
}

func TestRadioActivationRejectsChangedBootBeforeIntent(t *testing.T) {
	f := radioTestFixture(t)
	f.write(t, "proc/sys/kernel/random/boot_id", "22222222-2222-4222-8222-222222222222")
	if err := activateWirelessRadio(context.Background(), f.owner); err == nil {
		t.Fatal("stale boot admitted")
	}
	if _, err := f.target.root.Stat(f.owner.statePath("radio.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("radio intent written before boot qualification")
	}
	if len(f.calls) != 0 {
		t.Fatal("device command executed before boot qualification")
	}
}

func TestRadioRollbackUnloadsPartialCandidateWithoutRadioRecord(t *testing.T) {
	f := radioTestFixture(t)
	f.loaded["btusb"] = "candidate"
	durable := false
	f.owner.fault = func(point string) error {
		if point == "after-sync:"+f.owner.statePath("radio.json") {
			durable = true
		}
		return nil
	}
	f.unloadCheck = func() error {
		if !durable {
			return errors.New("radio intent not durable before unload")
		}
		r, err := f.owner.readRadioRecord()
		if err != nil {
			return err
		}
		if r.Phase != radioPhaseRestoring {
			return errors.New("wrong durable recovery phase")
		}
		return nil
	}
	if err := rollbackWirelessRadio(context.Background(), f.owner); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(f.calls, "unload:btusb") {
		t.Fatal("partial candidate survived file rollback")
	}
	for _, module := range radioModuleNames(NobleBluetoothProfile) {
		if f.loaded[module] != "stock" {
			t.Fatal("stock runtime not restored", module)
		}
	}
	if f.owner.phase != "rolled_back" {
		t.Fatal("file rollback incomplete")
	}
}

func radioCommittedPeerFixture(t *testing.T) (*radioFixture, string) {
	t.Helper()
	ctx := context.Background()
	wifi, baseline, wifiSigned, wifiCert := indexedFixture(t, NobleProfile)
	bt, _, btSigned, btCert := indexedFixture(t, NobleBluetoothProfile)
	route := "Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT\nenp1s0 00000000 0101A8C0 0003 0 0 100 00000000 0 0 0\n"
	identity := identityFixture(baseline, "/proc/net/route", route)
	for i, file := range wifi.Preserved {
		if file.Path == identity.Path {
			wifi.Preserved[i] = identity
		}
	}
	dependency := dependencyFixture(wifi, false) + dependencyFixture(bt, false)
	identity = identityFixture(baseline, "/"+abiTree+"/modules.dep", dependency)
	for i, file := range wifi.Preserved {
		if file.Path == identity.Path {
			wifi.Preserved[i] = identity
		}
	}
	indexer := func(ctx context.Context, copied *CopiedActivation, plan ActivationPlan) error {
		updated := dependency
		for i, original := range plan.Originals {
			updated = strings.ReplaceAll(updated, strings.TrimPrefix(original.Path, "/"+abiTree+"/"), strings.TrimPrefix(plan.OverlayDirectory, abiTree+"/")+"/"+plan.Candidates[i].Path)
		}
		for _, name := range depmodIndexes {
			data := "combined " + name
			if name == "modules.dep" {
				data = updated
			}
			if err := copied.root.WriteFile(path.Join(abiTree, name), []byte(data), 0600); err != nil {
				return err
			}
		}
		return ctx.Err()
	}
	wifiIndex, err := newIndexedActivation(ctx, baseline, wifi, indexer)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = wifiIndex.close() })
	if _, err = wifiIndex.stage(ctx, wifiSigned, wifiCert); err != nil {
		t.Fatal(err)
	}
	owner, target := durableTargetFixture(t, wifiIndex)
	f := &radioFixture{owner: owner, target: target, loaded: map[string]string{}}
	f.prepare(t, NobleBluetoothProfile, "usb:1-6")
	owner.admission = &wirelessRootAdmission{}
	owner.context = &wirelessJournalContext{OperatorUID: uint32(os.Geteuid()), Device: "pci:0000:09:00.0", BootID: radioFixtureBoot, EthernetRoutes: ethernetRouteLines([]byte(route), wifi.WiredInterfaces)}
	if err = owner.begin(ctx, wifiIndex, wifiCert); err != nil {
		t.Fatal(err)
	}
	if err = owner.apply(ctx); err != nil {
		t.Fatal(err)
	}
	parent := owner.intent.ID
	data, err := owner.read(abiTree+"/modules.dep", maxArtifactBytes)
	if err != nil {
		t.Fatal(err)
	}
	dependency = string(data)
	for i, file := range bt.Preserved {
		got, e := owner.identity(strings.TrimPrefix(file.Path, "/"))
		if e != nil || got.Absent {
			t.Fatal("missing combined baseline", file.Path, e)
		}
		bt.Preserved[i].SHA256 = got.SHA256
	}
	btIndex, err := newIndexedActivation(ctx, target.root.FS(), bt, indexer)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = btIndex.close() })
	if _, err = btIndex.stage(ctx, btSigned, btCert); err != nil {
		t.Fatal(err)
	}
	stock, err := moduleSnapshots(target.root, NobleBluetoothProfile)
	if err != nil {
		t.Fatal(err)
	}
	owner.context = &wirelessJournalContext{OperatorUID: uint32(os.Geteuid()), Device: "usb:1-6", BootID: radioFixtureBoot, EthernetRoutes: ethernetRouteLines([]byte(route), bt.WiredInterfaces), StageModules: stock}
	if err = owner.begin(ctx, btIndex, btCert); err != nil {
		t.Fatal(err)
	}
	if err = owner.apply(ctx); err != nil {
		t.Fatal(err)
	}
	owner.radio = f.system()
	return f, parent
}

func TestRadioPeerParentWaitsForDurableStockRuntimeAndReopen(t *testing.T) {
	for _, afterStockRecord := range []bool{false, true} {
		t.Run(map[bool]string{false: "stock_load_interrupted", true: "stock_record_durable_before_parent"}[afterStockRecord], func(t *testing.T) {
			f, parent := radioCommittedPeerFixture(t)
			child := f.owner.intent.ID
			if err := activateWirelessRadio(context.Background(), f.owner); err != nil {
				t.Fatal(err)
			}
			if afterStockRecord {
				f.owner.fault = func(point string) error {
					if point == "before-rename:"+path.Join(wirelessJournalDir, "active.json") {
						return errors.New("parent publication interrupted")
					}
					return nil
				}
			} else {
				f.failRestore = true
			}
			if err := rollbackWirelessRadio(context.Background(), f.owner); err == nil {
				t.Fatal("stock restoration interruption not exercised")
			}
			activeBytes, err := f.owner.readState(path.Join(wirelessJournalDir, "active.json"), 4096)
			if err != nil {
				t.Fatal(err)
			}
			var active durableActive
			if err = strictWirelessJSON(activeBytes, &active); err != nil {
				t.Fatal(err)
			}
			if active.ID != child {
				t.Fatal("published parent selected before durable runtime restoration")
			}
			f.owner.close()
			reopened, err := openDurableWireless(context.Background(), f.target.root)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.close()
			f.owner = reopened
			reopened.radio = f.system()
			if err = rollbackWirelessRadio(context.Background(), reopened); err != nil {
				t.Fatal(err)
			}
			record, err := reopened.readRadioRecord()
			if err != nil || record.Phase != radioPhaseRestored {
				t.Fatal("durable stock restoration missing", err)
			}
			activeBytes, err = reopened.readState(path.Join(wirelessJournalDir, "active.json"), 4096)
			if err != nil {
				t.Fatal(err)
			}
			if err = strictWirelessJSON(activeBytes, &active); err != nil {
				t.Fatal(err)
			}
			if active.ID != parent {
				t.Fatal("committed peer parent not published after durable stock runtime")
			}
		})
	}
}

func TestRadioCancellationCompensatesWithFreshContext(t *testing.T) {
	f := radioTestFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.cancelLoad = cancel
	if err := activateWirelessRadio(ctx, f.owner); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
	if f.owner.phase != "rolled_back" {
		t.Fatal("fresh-context compensation did not restore files")
	}
	record, err := f.owner.readRadioRecord()
	if err != nil || record.Phase != radioPhaseRestored {
		t.Fatal("runtime compensation incomplete", err)
	}
}

func TestRadioUnknownIdentityDoesNotUnloadOrUndoFiles(t *testing.T) {
	f := radioTestFixture(t)
	f.loaded["btusb"] = "foreign"
	if err := rollbackWirelessRadio(context.Background(), f.owner); err == nil {
		t.Fatal("unknown code admitted to rollback")
	}
	if f.owner.phase != "published" || len(f.calls) != 0 {
		t.Fatal("unknown runtime caused effects")
	}
}

func TestRadioRecordRetainsExactLineage(t *testing.T) {
	f := radioTestFixture(t)
	if err := activateWirelessRadio(context.Background(), f.owner); err != nil {
		t.Fatal(err)
	}
	r, err := f.owner.readRadioRecord()
	if err != nil {
		t.Fatal(err)
	}
	r.IntentSHA256 = strings.Repeat("f", 64)
	if err = f.owner.saveRadioRecord(r); err != nil {
		t.Fatal(err)
	}
	if err = rollbackWirelessRadio(context.Background(), f.owner); err == nil {
		t.Fatal("foreign radio journal lineage accepted")
	}
}
