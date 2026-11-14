//go:build linux

package hostreqkit

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func permissionRequest() OpenRGBPermissionRequest {
	return OpenRGBPermissionRequest{
		ArtifactSHA256: "d556c2273b0ef863dbee7ffdb438304b382add5d685c93c60758b42de3b0268d", DetectorSHA256: openRGBQualifiedDetectorCatalogSHA256, OwnerUID: 1000,
		Devices: []OpenRGBLinuxDevice{{Detector: "ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC", Kind: "i2c", Topology: "pci0000:00/0000:00:01.0/0000:01:00.0", HardwareID: "10de:2705:1043:8957", Adapter: "NVIDIA i2c adapter 0 at 1:00.0"}, {Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/0000:00:14.0/usb1/1-7/1-7:1.0", HardwareID: "0b05:19af:00"}},
	}
}

type permissionFixture struct {
	e           openRGBPermissionExecutor
	now         time.Time
	nodes       []OpenRGBLinuxNode
	snapshots   map[OpenRGBLinuxDevice]openRGBNodeSnapshot
	calls       [][]string
	failTrigger bool
	failResolve bool
	restoreFail bool
}

func newPermissionFixture(t *testing.T) *permissionFixture {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	r := permissionRequest()
	f := &permissionFixture{now: time.Date(2026, 11, 11, 3, 0, 0, 0, time.UTC), snapshots: map[OpenRGBLinuxDevice]openRGBNodeSnapshot{}}
	for i, d := range r.Devices {
		n := OpenRGBLinuxNode{Selector: d, Node: filepath.Join(root, "dev", []string{"i2c-7", "hidraw3"}[i]), SysPath: filepath.Join("/sys/devices", d.Topology, []string{"i2c-7/i2c-dev/i2c-7", "0003:0B05:19AF.0001/hidraw/hidraw3"}[i]), MajorMinor: []string{"89:7", "240:3"}[i]}
		f.nodes = append(f.nodes, n)
		f.snapshots[d] = openRGBNodeSnapshot{Selector: d, UID: 0, GID: 51, Mode: 0o660, ACL: permissionReadbackACL(6)}
	}
	f.e = openRGBPermissionExecutor{rule: filepath.Join(root, "etc/udev/rules.d/68-vrooli-openrgb.rules"), state: filepath.Join(root, "var/lib/vrooli/openrgb-device-access"), boundary: root, rootUID: uint32(os.Geteuid()), isRoot: func() bool { return true }, operatorUID: func() (uint32, error) { return 1000, nil }, now: func() time.Time { return f.now }, machine: func() (string, error) { return strings.Repeat("a", 32), nil }}
	f.e.resolve = func(ctx context.Context, ds []OpenRGBLinuxDevice) ([]OpenRGBLinuxNode, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if f.failResolve {
			return nil, errors.New("fixture topology unavailable")
		}
		var out []OpenRGBLinuxNode
		for _, d := range ds {
			i := slices.IndexFunc(f.nodes, func(n OpenRGBLinuxNode) bool { return n.Selector == d })
			if i < 0 {
				return nil, errors.New("fixture selected node missing")
			}
			out = append(out, f.nodes[i])
		}
		return out, nil
	}
	f.e.resolvePresent = func(ctx context.Context, ds []OpenRGBLinuxDevice) ([]OpenRGBLinuxNode, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var out []OpenRGBLinuxNode
		for _, d := range ds {
			for _, n := range f.nodes {
				if n.Selector == d {
					out = append(out, n)
				}
			}
		}
		return out, nil
	}
	f.e.snapshot = func(n OpenRGBLinuxNode) (openRGBNodeSnapshot, error) { return f.snapshots[n.Selector], nil }
	f.e.restore = func(_ context.Context, n OpenRGBLinuxNode, s openRGBNodeSnapshot) error {
		if f.restoreFail {
			return errors.New("fixture restore failed")
		}
		f.snapshots[n.Selector] = s
		return nil
	}
	f.e.run = func(ctx context.Context, args ...string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		f.calls = append(f.calls, slices.Clone(args))
		if args[0] == "trigger" {
			if f.failTrigger {
				return errors.New("fixture targeted trigger failed")
			}
			for _, n := range f.nodes {
				if n.SysPath == args[len(args)-1] {
					s := f.snapshots[n.Selector]
					s.UID = 1000
					s.Mode = 0o600
					s.ACL = permissionReadbackACL(0)
					f.snapshots[n.Selector] = s
				}
			}
		}
		return nil
	}
	return f
}

func (f *permissionFixture) plan(t *testing.T) OpenRGBPermissionPlan {
	t.Helper()
	p, err := f.e.plan(context.Background(), permissionRequest())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOpenRGBPermissionPlanBindsExactLiteralPhysicalRules(t *testing.T) {
	f := newPermissionFixture(t)
	p := f.plan(t)
	if p.Policy.OwnerUID != 1000 || p.Policy.GroupID != 0 || p.Policy.RuleSHA256 != openRGBPermissionDigest([]byte(p.RuleContent)) || !p.ExpiresAt.Equal(p.CreatedAt.Add(15*time.Minute)) {
		t.Fatalf("bad plan: %+v", p)
	}
	for _, want := range []string{`OWNER:="1000", MODE:="0600"`, `ATTRS{subsystem_device}=="0x8957"`, `ATTRS{name}=="NVIDIA i2c adapter 0 at 1:00.0"`, `ATTRS{bInterfaceNumber}=="00"`, `ENV{VROOLI_OPENRGB_ROUTE}=""`} {
		if !strings.Contains(p.RuleContent, want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Contains(p.RuleContent, "i2c-7") || strings.Contains(p.RuleContent, "hidraw3") || strings.Contains(p.RuleContent, "GROUP") || len(f.calls) > 0 {
		t.Fatalf("plan pinned transient node or performed effect: %+v", p)
	}
	// Node renumbering does not alter physical authority, unlike any rule/UID edit.
	p.Nodes[0].Node = "/dev/i2c-999"
	if err := f.e.validatePlan(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	p.RuleContent += "OWNER=\"0\"\n"
	if f.e.validatePlan(context.Background(), p) == nil {
		t.Fatal("modified rule admitted")
	}
}

func TestOpenRGBPermissionPlanRejectsUnqualifiedOrPatternSelectors(t *testing.T) {
	for _, change := range []func(*OpenRGBPermissionRequest){func(r *OpenRGBPermissionRequest) { r.OwnerUID = 0 }, func(r *OpenRGBPermissionRequest) { r.OwnerUID = 1001 }, func(r *OpenRGBPermissionRequest) { r.ArtifactSHA256 = strings.Repeat("c", 64) }, func(r *OpenRGBPermissionRequest) { r.Devices[0].Adapter = "NVIDIA*" }, func(r *OpenRGBPermissionRequest) { r.Devices[0].Topology += "/../other" }, func(r *OpenRGBPermissionRequest) { r.Devices[1].HardwareID = "0b05:ffff:00" }} {
		f := newPermissionFixture(t)
		r := permissionRequest()
		change(&r)
		if _, err := f.e.plan(context.Background(), r); err == nil {
			t.Fatal("unqualified selection admitted")
		}
		if len(f.calls) > 0 {
			t.Fatal("invalid plan executed commands")
		}
	}
}

func TestOpenRGBPermissionApplyRollbackAndIdempotentRetry(t *testing.T) {
	f := newPermissionFixture(t)
	p := f.plan(t)
	original := map[OpenRGBLinuxDevice]openRGBNodeSnapshot{}
	for d, s := range f.snapshots {
		original[d] = s
	}
	// Resolve the physical route after renumbering rather than invoking the old path.
	f.nodes[0].Node = filepath.Join(filepath.Dir(f.nodes[0].Node), "i2c-17")
	f.nodes[0].SysPath = strings.ReplaceAll(f.nodes[0].SysPath, "i2c-7", "i2c-17")
	f.nodes[0].MajorMinor = "89:17"
	got, err := f.e.apply(context.Background(), p)
	if err != nil || !got.Applied || got.Pending {
		t.Fatalf("apply=%+v err=%v", got, err)
	}
	if got.Nodes[0] != f.nodes[0].Node || len(f.calls) != 3 || !slices.Equal(f.calls[0], []string{"control", "--reload-rules"}) {
		t.Fatalf("wrong actions: %+v %+v", got, f.calls)
	}
	for _, cmd := range f.calls[1:] {
		if len(cmd) != 4 || cmd[0] != "trigger" || cmd[1] != "--action=change" || cmd[2] != "--settle" || !strings.HasPrefix(cmd[3], "/sys/devices/") {
			t.Fatalf("unbounded trigger: %v", cmd)
		}
	}
	if _, err = f.e.apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	got, err = f.e.rollback(context.Background(), p.ID)
	if err != nil || !got.RolledBack || got.Pending {
		t.Fatalf("rollback=%+v %v", got, err)
	}
	if _, err = os.Lstat(f.e.rule); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("initial absent rule not restored: %v", err)
	}
	for d, want := range original {
		actual := f.snapshots[d]
		if actual.UID != want.UID || actual.GID != want.GID || actual.Mode != want.Mode || !slices.Equal(actual.ACL, want.ACL) {
			t.Fatalf("node snapshot not restored: %+v", actual)
		}
	}
	if _, err = f.e.rollback(context.Background(), p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.e.apply(context.Background(), p); err == nil {
		t.Fatal("rolled-back plan reused")
	}
}

func TestOpenRGBPermissionForeignFileAndSymlinksPreserved(t *testing.T) {
	for _, kind := range []string{"foreign", "empty foreign", "symlink", "ancestor-symlink"} {
		t.Run(kind, func(t *testing.T) {
			f := newPermissionFixture(t)
			if err := os.MkdirAll(filepath.Dir(f.e.rule), 0o700); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "foreign":
				if err := os.WriteFile(f.e.rule, []byte("operator-owned content\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			case "empty foreign":
				if err := os.WriteFile(f.e.rule, []byte{}, 0o644); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(filepath.Join(f.e.boundary, "foreign"), f.e.rule); err != nil {
					t.Fatal(err)
				}
			case "ancestor-symlink":
				dir := filepath.Dir(f.e.rule)
				if err := os.Remove(dir); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(f.e.boundary, dir); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := f.e.plan(context.Background(), permissionRequest()); err == nil {
				t.Fatal("foreign path admitted")
			}
			if len(f.calls) > 0 {
				t.Fatal("foreign path activated")
			}
		})
	}
}

func TestOpenRGBPermissionFailureExpiryAndCancellationRemainTruthful(t *testing.T) {
	t.Run("expiry", func(t *testing.T) {
		f := newPermissionFixture(t)
		p := f.plan(t)
		f.now = p.ExpiresAt
		if got, err := f.e.apply(context.Background(), p); err == nil || got.Applied || len(f.calls) > 0 {
			t.Fatalf("expired plan applied: %+v %v", got, err)
		}
	})
	t.Run("canceled before effects", func(t *testing.T) {
		f := newPermissionFixture(t)
		p := f.plan(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := f.e.apply(ctx, p); !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if len(f.calls) > 0 {
			t.Fatal("canceled plan executed")
		}
	})
	t.Run("targeted activation retry after expiry", func(t *testing.T) {
		f := newPermissionFixture(t)
		p := f.plan(t)
		f.failTrigger = true
		got, err := f.e.apply(context.Background(), p)
		if err == nil || !got.Pending || got.Applied {
			t.Fatalf("false application: %+v %v", got, err)
		}
		f.failTrigger = false
		f.now = p.ExpiresAt.Add(time.Minute)
		got, err = f.e.apply(context.Background(), p)
		if err != nil || !got.Applied {
			t.Fatalf("durable pending retry failed: %+v %v", got, err)
		}
	})
	t.Run("rollback retry", func(t *testing.T) {
		f := newPermissionFixture(t)
		p := f.plan(t)
		if _, err := f.e.apply(context.Background(), p); err != nil {
			t.Fatal(err)
		}
		f.restoreFail = true
		got, err := f.e.rollback(context.Background(), p.ID)
		if err == nil || !got.Pending || got.RolledBack {
			t.Fatalf("false rollback: %+v %v", got, err)
		}
		if _, applyErr := f.e.apply(context.Background(), p); applyErr == nil {
			t.Fatal("stale apply reversed pending rollback")
		}
		f.restoreFail = false
		got, err = f.e.rollback(context.Background(), p.ID)
		if err != nil || !got.RolledBack {
			t.Fatalf("rollback retry failed: %+v %v", got, err)
		}
	})
	t.Run("foreign replacement", func(t *testing.T) {
		f := newPermissionFixture(t)
		p := f.plan(t)
		if _, err := f.e.apply(context.Background(), p); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f.e.rule, []byte("later foreign rule\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := f.e.rollback(context.Background(), p.ID); err == nil {
			t.Fatal("foreign replacement overwritten")
		}
		b, _ := os.ReadFile(f.e.rule)
		if string(b) != "later foreign rule\n" {
			t.Fatal("foreign replacement changed")
		}
	})
}

func TestOpenRGBPermissionPlanRequiresElevatedExecutorAndSerializes(t *testing.T) {
	f := newPermissionFixture(t)
	p := f.plan(t)
	f.e.isRoot = func() bool { return false }
	if _, err := f.e.apply(context.Background(), p); !errors.Is(err, ErrElevationRequired) {
		t.Fatal(err)
	}
	f.e.isRoot = func() bool { return true }
	release, err := f.e.lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan error, 1)
	go func() { defer wg.Done(); _, err := f.e.apply(ctx, p); done <- err }()
	cancel()
	wg.Wait()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(f.calls) > 0 {
		t.Fatal("canceled queued operation executed")
	}
}

func TestOpenRGBPermissionUdevSyntax(t *testing.T) {
	// verify parses only these scratch rules; unlike test/trigger it does not
	// simulate or dispatch an event and receives no sysfs or device argument.
	binary, err := exec.LookPath("udevadm")
	if err != nil {
		t.Skip("native udev rule syntax verifier unavailable")
	}
	f := newPermissionFixture(t)
	p := f.plan(t)
	path := filepath.Join(t.TempDir(), "68-vrooli-openrgb.rules")
	if err = os.WriteFile(path, []byte(p.RuleContent), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.CommandContext(t.Context(), binary, "verify", path).CombinedOutput()
	if err != nil {
		t.Fatalf("udev syntax: %v: %s", err, output)
	}
}

func TestOpenRGBPermissionReplacementRevokesRemovedRoutesAndRollbackRestoresPrior(t *testing.T) {
	f := newPermissionFixture(t)
	p1 := f.plan(t)
	if _, err := f.e.apply(context.Background(), p1); err != nil {
		t.Fatal(err)
	}
	r := permissionRequest()
	r.Devices = r.Devices[:1]
	f.now = f.now.Add(time.Minute)
	p2, err := f.e.plan(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	result, err := f.e.apply(context.Background(), p2)
	if err != nil || !result.Applied {
		t.Fatalf("replace=%+v %v", result, err)
	}
	hid := permissionRequest().Devices[1]
	if actual := f.snapshots[hid]; actual.UID != 0 || actual.Mode != 0o660 || !slices.Equal(actual.ACL, permissionReadbackACL(6)) {
		t.Fatalf("deselected route retains grant: %+v", actual)
	}
	if result.ActivePlanID != p2.ID || result.ActivePolicy == nil {
		t.Fatal("new active authority missing")
	}
	result, err = f.e.rollback(context.Background(), p2.ID)
	if err != nil || !result.RolledBack || result.ActivePlanID != p1.ID || result.ActivePolicy == nil {
		t.Fatalf("restore prior=%+v %v", result, err)
	}
	if f.snapshots[hid].UID != 1000 || f.snapshots[hid].Mode != 0o600 {
		t.Fatal("prior selected grant not restored")
	}
	if _, err = f.e.rollback(context.Background(), p1.ID); err != nil {
		t.Fatal(err)
	}
	if f.snapshots[hid].UID != 0 {
		t.Fatal("original baseline lost across replacement")
	}
}

func TestOpenRGBPermissionReplacementRetainsOriginalBaselineAcrossSeveralSelections(t *testing.T) {
	f := newPermissionFixture(t)
	full := permissionRequest()
	gpu := full
	gpu.Devices = slices.Clone(full.Devices[:1])
	p1, err := f.e.plan(context.Background(), gpu)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.e.apply(context.Background(), p1); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Minute)
	p2, err := f.e.plan(context.Background(), full)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.e.apply(context.Background(), p2); err != nil {
		t.Fatal(err)
	}
	hidOnly := full
	hidOnly.Devices = slices.Clone(full.Devices[1:])
	f.now = f.now.Add(time.Minute)
	p3, err := f.e.plan(context.Background(), hidOnly)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.e.apply(context.Background(), p3); err != nil {
		t.Fatal(err)
	}
	if got := f.snapshots[full.Devices[0]]; got.UID != 0 || got.Mode != 0o660 {
		t.Fatalf("original GPU baseline replaced by intermediate grant: %+v", got)
	}
}

func TestOpenRGBPermissionAbsentControllerCannotBlockPersistentRevocation(t *testing.T) {
	f := newPermissionFixture(t)
	p := f.plan(t)
	if _, err := f.e.apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	f.nodes = f.nodes[:1]
	got, err := f.e.rollback(context.Background(), p.ID)
	if err != nil || !got.RolledBack || got.ActivePolicy != nil {
		t.Fatalf("absent controller blocked rollback: %+v %v", got, err)
	}
	if _, err = os.Lstat(f.e.rule); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("persistent rule remains")
	}
	f.nodes = nil
	if _, err = f.e.rollback(context.Background(), p.ID); err != nil {
		t.Fatalf("completed rollback lost idempotence after unplug: %v", err)
	}
}

func TestOpenRGBPermissionIdenticalSuccessorCannotBeUndoneByStaleTransaction(t *testing.T) {
	f := newPermissionFixture(t)
	p1 := f.plan(t)
	if _, err := f.e.apply(context.Background(), p1); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Minute)
	p2 := f.plan(t)
	if p1.ID == p2.ID || p1.RuleContent != p2.RuleContent {
		t.Fatal("fixture has no distinct identical-content successor")
	}
	if _, err := f.e.apply(context.Background(), p2); err != nil {
		t.Fatal(err)
	}
	before := len(f.calls)
	if _, err := f.e.rollback(context.Background(), p1.ID); err == nil {
		t.Fatal("stale rollback revoked successor")
	}
	if _, err := f.e.apply(context.Background(), p1); err == nil {
		t.Fatal("stale apply replaced successor")
	}
	if len(f.calls) != before {
		t.Fatal("stale transaction dispatched effects")
	}
	id, err := f.e.currentID()
	if err != nil || id != p2.ID {
		t.Fatal("successor authority changed")
	}
}

func TestOpenRGBPermissionARMSoCHIDPlanApplyRollback(t *testing.T) {
	if !linuxOpenRGBPermissionArchitectureSupported("arm64") {
		t.Fatal("qualified Linux ARM release cannot enroll")
	}
	f := newPermissionFixture(t)
	r := permissionRequest()
	r.ArtifactSHA256 = "0fd6806c265f9dbd9b0a8847cf4b4cb172c86baf2e39f52f869f8c5325c6751b"
	d := r.Devices[1]
	previous := f.snapshots[d]
	d.Topology = "platform/soc/fe980000.usb/usb1/1-2/1-2:1.0"
	r.Devices = []OpenRGBLinuxDevice{d}
	n := f.nodes[1]
	n.Selector = d
	n.SysPath = filepath.Join("/sys/devices", d.Topology, "0003:0B05:19AF.0001/hidraw/hidraw3")
	f.nodes = []OpenRGBLinuxNode{n}
	previous.Selector = d
	f.snapshots = map[OpenRGBLinuxDevice]openRGBNodeSnapshot{d: previous}
	plan, err := f.e.plan(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.RuleContent, "/devices/platform/soc/fe980000.usb/") || strings.Contains(plan.RuleContent, "pci0000:") {
		t.Fatal("ARM physical route was replaced with an x86 topology")
	}
	result, err := f.e.apply(t.Context(), plan)
	if err != nil || !result.Applied {
		t.Fatalf("ARM permission apply: %+v %v", result, err)
	}
	if len(f.calls) != 2 || f.calls[1][len(f.calls[1])-1] != n.SysPath {
		t.Fatalf("ARM apply did not target only the selected node: %v", f.calls)
	}
	result, err = f.e.rollback(t.Context(), plan.ID)
	if err != nil || !result.RolledBack {
		t.Fatalf("ARM permission rollback: %+v %v", result, err)
	}
	restored := f.snapshots[d]
	if restored.UID != previous.UID || restored.GID != previous.GID || restored.Mode != previous.Mode || string(restored.ACL) != string(previous.ACL) {
		t.Fatal("ARM rollback lost original node authority")
	}
}
func TestOpenRGBPermissionArchitecturesRemainBounded(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		if !linuxOpenRGBPermissionArchitectureSupported(arch) {
			t.Fatal(arch)
		}
	}
	for _, arch := range []string{"arm", "386", "ppc64", "mips", "unknown"} {
		if linuxOpenRGBPermissionArchitectureSupported(arch) {
			t.Fatal("unqualified release ABI admitted", arch)
		}
	}
}

func TestOpenRGBPermissionARMQualificationIsExactArtifactAndHIDClass(t *testing.T) {
	request := permissionRequest()
	for _, artifact := range []string{openRGBLinuxARM64EnvelopeSHA256, openRGBLinuxARM64PayloadSHA256} {
		policy := OpenRGBLinuxPolicy{ArtifactSHA256: artifact, DetectorSHA256: openRGBQualifiedDetectorCatalogSHA256, Devices: request.Devices[1:]}
		if err := ValidateOpenRGBLinuxDetectorScope(policy); err != nil {
			t.Fatal("exact ARM HID release rejected", err)
		}
		policy.Devices = request.Devices
		if err := ValidateOpenRGBLinuxDetectorScope(policy); err == nil {
			t.Fatal("ARM GPU/I2C footprint inferred from x86 qualification")
		}
	}
	policy := OpenRGBLinuxPolicy{ArtifactSHA256: strings.Repeat("a", 64), DetectorSHA256: openRGBQualifiedDetectorCatalogSHA256, Devices: request.Devices[1:]}
	if err := ValidateOpenRGBLinuxDetectorScope(policy); err == nil {
		t.Fatal("arbitrary ARM executable was qualified")
	}
	for _, artifact := range []string{openRGBLinuxAMD64EnvelopeSHA256, openRGBLinuxAMD64PayloadSHA256} {
		policy.ArtifactSHA256 = artifact
		policy.Devices = request.Devices
		if err := ValidateOpenRGBLinuxDetectorScope(policy); err != nil {
			t.Fatal("exact amd64 release scope changed", err)
		}
	}
}
