//go:build linux

package hostreqkit

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

const openRGBTestCredentials = "Uid:\t1000 1000 1000 1000\nGid:\t1000 1000 1000 1000\nGroups:\t1000 1500\nCapInh:\t0\nCapPrm:\t0\nCapEff:\t0\nCapAmb:\t0\n"

type openRGBPolicyFixture struct {
	probe  openRGBLinuxProbe
	policy OpenRGBLinuxPolicy
	check  OpenRGBLinuxCheck
	stats  map[string]unix.Stat_t
	access map[string]uint32
	acls   map[string]bool
}

func writeOpenRGBFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func openRGBDeviceFixture(t *testing.T, gpuMinor uint32) *openRGBPolicyFixture {
	t.Helper()
	root := t.TempDir()
	f := &openRGBPolicyFixture{
		stats: map[string]unix.Stat_t{}, access: map[string]uint32{}, acls: map[string]bool{},
		policy: OpenRGBLinuxPolicy{
			MachineID: strings.Repeat("a", 32), ArtifactSHA256: "d556c2273b0ef863dbee7ffdb438304b382add5d685c93c60758b42de3b0268d", DetectorSHA256: openRGBQualifiedDetectorCatalogSHA256, RuleSHA256: strings.Repeat("d", 64), GroupID: 1500,
			Devices: []OpenRGBLinuxDevice{
				{Detector: "ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC", Kind: "i2c", Topology: "pci0000:00/0000:01:00.0", HardwareID: "10de:2705:1043:8957", Adapter: "NVIDIA I2C adapter 1"},
				{Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/0000:03:00.0/usb1/1-8/1-8:1.0", HardwareID: "0b05:19af:00"},
			},
		},
	}
	id, err := f.policy.ID()
	if err != nil {
		t.Fatal(err)
	}
	f.check = OpenRGBLinuxCheck{PolicyID: id, ArtifactSHA256: f.policy.ArtifactSHA256, DetectorSHA256: f.policy.DetectorSHA256, EffectiveUID: 1000}
	f.probe = openRGBLinuxProbe{
		dev: filepath.Join(root, "dev"), sys: filepath.Join(root, "sys"), proc: filepath.Join(root, "proc"),
		machineID:     func() (string, error) { return f.policy.MachineID, nil },
		ruleDigest:    func() (string, error) { return f.policy.RuleSHA256, nil },
		accountGroups: func(int) ([]uint32, error) { return []uint32{1000, 1500}, nil },
		stat: func(path string, out *unix.Stat_t) error {
			if stat, ok := f.stats[filepath.Base(path)]; ok {
				*out = stat
				return nil
			}
			return unix.ENOENT
		},
		access: func(path string, mode uint32) error {
			if f.access[filepath.Base(path)]&mode == mode {
				return nil
			}
			return unix.EACCES
		},
		hasACL:      func(path string) (bool, error) { return f.acls[filepath.Base(path)], nil },
		incarnation: func(int) (string, error) { return "boot:123", nil },
	}
	for _, pid := range []string{"self", "42"} {
		writeOpenRGBFixture(t, filepath.Join(f.probe.proc, pid, "status"), openRGBTestCredentials)
	}
	writeOpenRGBFixture(t, filepath.Join(f.probe.proc, "self", "cgroup"), "0::/user.slice/vrooli-runtime-supervisor.service")
	writeOpenRGBFixture(t, filepath.Join(f.probe.proc, "42", "cgroup"), "0::/user.slice/vrooli-services.slice/openrgb.scope")
	gpu := filepath.Join(f.probe.sys, "devices", f.policy.Devices[0].Topology)
	for name, value := range map[string]string{"vendor": "0x10de", "device": "0x2705", "subsystem_vendor": "0x1043", "subsystem_device": "0x8957"} {
		writeOpenRGBFixture(t, filepath.Join(gpu, name), value)
	}
	gpuNode := fmt.Sprintf("i2c-%d", gpuMinor)
	writeOpenRGBFixture(t, filepath.Join(gpu, gpuNode, "name"), f.policy.Devices[0].Adapter)
	f.addNode(t, gpuNode, "i2c-dev", filepath.Join(gpu, gpuNode), 89, gpuMinor, true)
	other := filepath.Join(f.probe.sys, "devices", "pci0000:00", "0000:02:00.0", "i2c-9")
	writeOpenRGBFixture(t, filepath.Join(other, "name"), "Unrelated adapter")
	f.addNode(t, "i2c-9", "i2c-dev", other, 89, 9, false)
	iface := filepath.Join(f.probe.sys, "devices", f.policy.Devices[1].Topology)
	writeOpenRGBFixture(t, filepath.Join(filepath.Dir(iface), "idVendor"), "0b05")
	writeOpenRGBFixture(t, filepath.Join(filepath.Dir(iface), "idProduct"), "19af")
	writeOpenRGBFixture(t, filepath.Join(iface, "bInterfaceNumber"), "00")
	hid := filepath.Join(iface, "0003:0B05:19AF.0001")
	writeOpenRGBFixture(t, filepath.Join(hid, "uevent"), "")
	f.addNode(t, "hidraw3", "hidraw", hid, 240, 3, true)
	f.addNode(t, "port", "", "", 1, 4, false)
	return f
}

func (f *openRGBPolicyFixture) addNode(t *testing.T, node, class, sysDevice string, major, minor uint32, allowed bool) {
	t.Helper()
	writeOpenRGBFixture(t, filepath.Join(f.probe.dev, node), "fixture metadata only")
	stat := unix.Stat_t{Mode: unix.S_IFCHR | 0o660, Rdev: unix.Mkdev(major, minor)}
	if allowed {
		stat.Gid = f.policy.GroupID
		f.access[node] = unix.R_OK | unix.W_OK
	}
	f.stats[node] = stat
	if class != "" {
		root := filepath.Join(f.probe.sys, "class", class, node)
		writeOpenRGBFixture(t, filepath.Join(root, "dev"), fmt.Sprintf("%d:%d", major, minor))
		if err := os.Symlink(sysDevice, filepath.Join(root, "device")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOpenRGBLinuxPolicyResolvesRenumberedPhysicalDevices(t *testing.T) {
	var policyID string
	for _, minor := range []uint32{4, 11} {
		f := openRGBDeviceFixture(t, minor)
		if policyID != "" && policyID != f.check.PolicyID {
			t.Fatal("device renumbering changed the enrolled identity")
		}
		policyID = f.check.PolicyID
		got, err := f.probe.observe(t.Context(), f.policy, f.check)
		if err != nil || !got.StaticGrantConfigured || !got.CurrentScopeVerified || got.LiveCredentialsMatch || len(got.SelectedNodes) != 2 || filepath.Base(got.SelectedNodes[1]) != fmt.Sprintf("i2c-%d", minor) {
			t.Fatalf("scoped observed access = %+v, %v", got, err)
		}
	}
}

func TestOpenRGBLinuxPolicyRejectsWideOrUnknownPermissions(t *testing.T) {
	for _, test := range []struct {
		name  string
		alter func(*openRGBPolicyFixture)
		want  string
	}{
		{"selected seat ACL", func(f *openRGBPolicyFixture) { f.acls["hidraw3"] = true }, "static device group"},
		{"unknown access syscall", func(f *openRGBPolicyFixture) { f.probe.access = func(string, uint32) error { return unix.ENOSYS } }, "effective read/write access"},
		{"installed rule changed", func(f *openRGBPolicyFixture) { f.probe.ruleDigest = func() (string, error) { return "changed", nil } }, "static udev grant"},
		{"session-only group", func(f *openRGBPolicyFixture) {
			f.probe.accountGroups = func(int) ([]uint32, error) { return []uint32{1000}, nil }
		}, "persistent OpenRGB device group"},
		{"different host", func(f *openRGBPolicyFixture) { f.probe.machineID = func() (string, error) { return "other", nil } }, "machine identity"},
		{"wrong detector digest", func(f *openRGBPolicyFixture) { f.check.DetectorSHA256 = strings.Repeat("e", 64) }, "enrolled launch identity"},
		{"device number changed", func(f *openRGBPolicyFixture) {
			node := f.stats["i2c-4"]
			node.Rdev = unix.Mkdev(89, 6)
			f.stats["i2c-4"] = node
		}, "device number"},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := openRGBDeviceFixture(t, 4)
			test.alter(f)
			got, err := f.probe.observe(t.Context(), f.policy, f.check)
			if err == nil || !strings.Contains(err.Error(), test.want) || got.CurrentScopeVerified {
				t.Fatalf("incomplete policy looked qualified: %+v, %v", got, err)
			}
		})
	}
}

func TestOpenRGBLinuxPolicyVerifiesChildCredentialsWithoutClaimingChildAccess(t *testing.T) {
	f := openRGBDeviceFixture(t, 4)
	f.check.PID, f.check.Incarnation = 42, "boot:123"
	got, err := f.probe.observe(t.Context(), f.policy, f.check)
	if !errors.Is(err, ErrOpenRGBLiveDeviceAccessUnverified) || !strings.Contains(err.Error(), "different cgroup") || !got.CurrentScopeVerified || !got.LiveCredentialsMatch {
		t.Fatalf("live proof distinction = %+v, %v", got, err)
	}
	writeOpenRGBFixture(t, filepath.Join(f.probe.proc, "42", "cgroup"), "0::/user.slice/vrooli-runtime-supervisor.service")
	got, err = f.probe.observe(t.Context(), f.policy, f.check)
	if !errors.Is(err, ErrOpenRGBLiveDeviceAccessUnverified) || !got.LiveCredentialsMatch {
		t.Fatalf("matching cgroups alone cannot qualify child access: %+v, %v", got, err)
	}
	writeOpenRGBFixture(t, filepath.Join(f.probe.proc, "42", "status"), strings.Replace(openRGBTestCredentials, "CapEff:\t0", "CapEff:\t2", 1))
	got, err = f.probe.observe(t.Context(), f.policy, f.check)
	if err == nil || !strings.Contains(err.Error(), "child credentials differ") || got.LiveCredentialsMatch {
		t.Fatalf("privileged child = %+v, %v", got, err)
	}
	writeOpenRGBFixture(t, filepath.Join(f.probe.proc, "42", "status"), openRGBTestCredentials)
	observations := 0
	f.probe.incarnation = func(int) (string, error) {
		observations++
		if observations == 2 {
			return "boot:124", nil
		}
		return "boot:123", nil
	}
	got, err = f.probe.observe(t.Context(), f.policy, f.check)
	if err == nil || !strings.Contains(err.Error(), "incarnation changed") || got.LiveCredentialsMatch {
		t.Fatalf("PID reuse = %+v, %v", got, err)
	}
}

func TestOpenRGBLinuxPolicyRejectsNodeReplacementDuringAccessCheck(t *testing.T) {
	f := openRGBDeviceFixture(t, 4)
	access := f.probe.access
	f.probe.access = func(path string, mode uint32) error {
		if filepath.Base(path) == "i2c-4" {
			stat := f.stats["i2c-4"]
			stat.Ino++
			f.stats["i2c-4"] = stat
		}
		return access(path, mode)
	}
	got, err := f.probe.observe(t.Context(), f.policy, f.check)
	if err == nil || !strings.Contains(err.Error(), "changed during access observation") || got.CurrentScopeVerified {
		t.Fatalf("replaced node looked qualified: %+v, %v", got, err)
	}
}

func TestOpenRGBLinuxPolicyRejectsAccessibleUnselectedHIDNode(t *testing.T) {
	f := openRGBDeviceFixture(t, 4)
	unrelated := filepath.Join(f.probe.sys, "devices", "pci0000:00", "0000:04:00.0", "usb2", "2-3", "2-3:1.0", "0003:0B05:19AF.0002")
	writeOpenRGBFixture(t, filepath.Join(unrelated, "uevent"), "HID_ID=0003:00000B05:000019AF")
	f.addNode(t, "hidraw4", "hidraw", unrelated, 240, 4, true)

	got, err := f.probe.observe(t.Context(), f.policy, f.check)
	if err == nil || !strings.Contains(err.Error(), "unrelated device") || got.CurrentScopeVerified {
		t.Fatalf("accessible unrelated HID node looked qualified: %+v, %v", got, err)
	}
}

func TestOpenRGBLinuxPolicyPreservesUnrelatedAmbientAccess(t *testing.T) {
	f := openRGBDeviceFixture(t, 4)
	f.access["i2c-9"], f.access["port"] = unix.R_OK|unix.W_OK, unix.R_OK|unix.W_OK
	other := filepath.Join(f.probe.sys, "devices", "pci0000:00", "0000:04:00.0", "usb2", "2-3", "2-3:1.0", "0003:1A2B:3C4D.0001")
	writeOpenRGBFixture(t, filepath.Join(other, "uevent"), "HID_ID=0003:00001A2B:00003C4D")
	f.addNode(t, "hidraw4", "hidraw", other, 240, 4, true)
	if got, err := f.probe.observe(t.Context(), f.policy, f.check); err != nil || !got.CurrentScopeVerified {
		t.Fatalf("unrelated ambient rights blocked bounded supported detection: %+v, %v", got, err)
	}
}

func TestOpenRGBLinuxPolicyRequiresEveryMatchingGPUAdapterInEnrollment(t *testing.T) {
	f := openRGBDeviceFixture(t, 4)
	gpu := filepath.Join(f.probe.sys, "devices", f.policy.Devices[0].Topology)
	writeOpenRGBFixture(t, filepath.Join(gpu, "i2c-5", "name"), "NVIDIA I2C adapter 2")
	f.addNode(t, "i2c-5", "i2c-dev", filepath.Join(gpu, "i2c-5"), 89, 5, true)
	if _, err := f.probe.observe(t.Context(), f.policy, f.check); err == nil || !strings.Contains(err.Error(), "unenrolled matching route") {
		t.Fatalf("an unselected matching GPU adapter was accessible: %v", err)
	}
	selected := f.policy.Devices[0]
	selected.Adapter = "NVIDIA I2C adapter 2"
	f.policy.Devices = append(f.policy.Devices, selected)
	f.check.PolicyID, _ = f.policy.ID()
	if got, err := f.probe.observe(t.Context(), f.policy, f.check); err != nil || len(got.SelectedNodes) != 3 {
		t.Fatalf("whole-card route enrollment failed: %+v, %v", got, err)
	}
}

func TestOpenRGBLinuxPolicyVerifiesPersistentNumericOwnerWithoutGroupRefresh(t *testing.T) {
	f := openRGBDeviceFixture(t, 4)
	f.policy.OwnerUID, f.policy.GroupID = 1000, 0
	f.check.PolicyID, _ = f.policy.ID()
	f.probe.accountGroups = func(int) ([]uint32, error) { return nil, errors.New("group lookup must not be needed") }
	for _, name := range []string{"i2c-4", "hidraw3"} {
		stat := f.stats[name]
		stat.Uid, stat.Gid, stat.Mode = 1000, 0, unix.S_IFCHR|0o600
		f.stats[name] = stat
		f.acls[name] = true // Existing session ACLs do not remove persistent owner rights.
	}
	if got, err := f.probe.observe(t.Context(), f.policy, f.check); err != nil || !got.StaticGrantConfigured {
		t.Fatalf("static owner grant required a session/group refresh: %+v, %v", got, err)
	}
}

func TestOpenRGBLinuxPolicyRejectsUnqualifiedDetectorCatalogBeforeProbe(t *testing.T) {
	f := openRGBDeviceFixture(t, 7)
	f.policy.DetectorSHA256 = strings.Repeat("f", 64)
	f.check.DetectorSHA256 = f.policy.DetectorSHA256
	f.check.PolicyID, _ = f.policy.ID()
	f.probe.access = func(string, uint32) error {
		t.Fatal("unqualified catalog reached device-access observation")
		return nil
	}
	if _, err := f.probe.observe(t.Context(), f.policy, f.check); err == nil {
		t.Fatal("unknown detector catalog received supported footprint proof")
	}
}

func TestOpenRGBLinuxPolicyGenerationTracksSelectedDeviceBirth(t *testing.T) {
	f := openRGBDeviceFixture(t, 7)
	observe := func() string {
		t.Helper()
		evidence, err := f.probe.observe(t.Context(), f.policy, f.check)
		if err != nil {
			t.Fatal(err)
		}
		if !openRGBHex(evidence.DeviceGenerationSHA256, 64) {
			t.Fatal("missing selected kernel generation")
		}
		return evidence.DeviceGenerationSHA256
	}
	before := observe()
	unrelated := f.stats["i2c-9"]
	unrelated.Ino++
	f.stats["i2c-9"] = unrelated
	if after := observe(); after != before {
		t.Fatal("unrelated device replacement changed selected generation")
	}
	selected := f.stats["i2c-7"]
	selected.Ctim.Nsec++
	f.stats["i2c-7"] = selected
	if after := observe(); after != before {
		t.Fatal("incidental permission timestamp changed selected generation")
	}
	selected.Ino++
	f.stats["i2c-7"] = selected
	if after := observe(); after == before {
		t.Fatal("same-path selected device replacement reused prior generation")
	}
}
