package remotesessionprotection

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

type reservationTestEntry string

func (e reservationTestEntry) Name() string      { return string(e) }
func (e reservationTestEntry) IsDir() bool       { return false }
func (e reservationTestEntry) Type() os.FileMode { return 0 }
func (e reservationTestEntry) Info() (os.FileInfo, error) {
	return reservationTestFileInfo{mode: 0644}, nil
}

// The fake manager and filesystem observe the same definition set. Native
// properties are independently read from the modeled kernel after effects.
func installReservationManagerFixture(t *testing.T, files map[string]string) {
	t.Helper()
	roots := []string{"/etc/systemd/system.control", "/run/systemd/system.control", "/etc/systemd/system", "/run/systemd/system", "/usr/lib/systemd/system"}
	units := []string{"user.slice", "user-1000.slice"}
	dirs := map[string]bool{}
	for _, root := range roots {
		for p := root; ; p = filepath.Dir(p) {
			dirs[p] = true
			if filepath.Dir(p) == p {
				break
			}
		}
		for _, unit := range units {
			for _, d := range reservationDropinNames(unit) {
				dirs[filepath.Join(root, d)] = true
			}
		}
	}
	paths := func() []string {
		if files != nil {
			out := make([]string, 0, len(files))
			for p := range files {
				out = append(out, p)
			}
			return out
		}
		candidates := []string{ancestorSlicePath, desktopSlicePath}
		out := []string{}
		for _, p := range candidates {
			if _, err := hostreqkit.ReadFileFn(p); err == nil {
				out = append(out, p)
			}
		}
		return out
	}
	stat := reservationLstat
	reservationLstat = func(p string) (os.FileInfo, error) {
		if dirs[p] {
			return reservationTestFileInfo{mode: os.ModeDir | 0755}, nil
		}
		return stat(p)
	}
	reservationReadDir = func(dir string) ([]os.DirEntry, error) {
		if !dirs[dir] {
			return nil, os.ErrNotExist
		}
		out := []os.DirEntry{}
		for _, p := range paths() {
			if filepath.Dir(p) == dir {
				out = append(out, reservationTestEntry(filepath.Base(p)))
			}
		}
		return out, nil
	}
	output := hostreqkit.CombinedOutputFn
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name != "systemctl" || len(args) == 0 || args[0] != "show" {
			return output(name, args...)
		}
		if len(args) == 2 && args[1] == "--property=UnitPath" {
			return []byte("UnitPath=" + strings.Join(roots, " ") + "\n"), nil
		}
		if len(args) < 2 {
			return nil, fmt.Errorf("missing unit")
		}
		unit := args[1]
		if unit != "user.slice" && unit != "user-1000.slice" {
			return nil, fmt.Errorf("unrecognized unit")
		}
		cgroup := testUserCgroup
		if unit == "user-1000.slice" {
			cgroup += "/user-1000.slice"
		}
		read := func(p string) string {
			b, e := hostreqkit.ReadFileFn(p)
			if e != nil {
				return "0"
			}
			v := strings.TrimSpace(string(b))
			if v == "max" {
				return "infinity"
			}
			return v
		}
		fragment := ""
		for _, root := range roots {
			if _, err := hostreqkit.ReadFileFn(filepath.Join(root, unit)); err == nil {
				fragment = filepath.Join(root, unit)
				break
			}
		}
		dropins := []string{}
		for _, p := range paths() {
			for _, root := range roots {
				for _, dir := range reservationDropinNames(unit) {
					if filepath.Dir(p) == filepath.Join(root, dir) && strings.HasSuffix(p, ".conf") {
						dropins = append(dropins, p)
					}
				}
			}
		}
		sort.Strings(dropins)
		return []byte(fmt.Sprintf("Id=%s\nNames=%s\nLoadState=loaded\nFragmentPath=%s\nDropInPaths=%s\nMemoryMin=%s\nMemoryLow=%s\nControlGroup=%s\nNeedDaemonReload=yes\n", unit, unit, fragment, strings.Join(dropins, " "), read(cgroup+"/memory.min"), read(cgroup+"/memory.low"), strings.TrimPrefix(cgroup, "/sys/fs/cgroup"))), nil
	}
}

func TestReservationDefinitionsRejectLaterPersistentAndRuntimeResets(t *testing.T) {
	for _, runtime := range []bool{false, true} {
		t.Run(strconv.FormatBool(runtime), func(t *testing.T) {
			files, calls := reservationFixture(t)
			files[ancestorSlicePath] = ancestorSliceContent(4915, 6963)
			root := "/etc/systemd/system"
			if runtime {
				root = "/run/systemd/system"
			}
			external := root + "/user.slice.d/99-operator-memory.conf"
			files[external] = "[Slice]\nMemoryMin=0\nMemoryLow=0\n"
			status := newTestHandler().Inspect(linuxHost(), baseReq())
			if status.Applied {
				t.Fatal("later explicit zero erased by kernel/owned-file proof")
			}
			got, err := newTestHandler().Apply(linuxHost(), status, hostreqkit.EnsureOptions{SudoMode: "ask"})
			if err != nil || got.Applied || len(*calls) != 0 {
				t.Fatalf("conflicting operator intent caused effects: %+v %v %v", got, err, *calls)
			}
			if files[external] != "[Slice]\nMemoryMin=0\nMemoryLow=0\n" {
				t.Fatal("operator definition changed")
			}
		})
	}
}

func TestReservationDefinitionsConvergeStrongerInactiveOperatorFloors(t *testing.T) {
	files, calls := reservationFixture(t)
	files[ancestorSlicePath] = ancestorSliceContent(4915, 6963)
	external := "/etc/systemd/system/user.slice.d/99-operator-memory.conf"
	files[external] = "[Slice]\nMemoryMin=32G\nMemoryLow=40G\n"
	files[testUserCgroup+"/memory.min"] = "0"
	files[testUserCgroup+"/memory.low"] = "0"
	output := hostreqkit.CombinedOutputFn
	hostreqkit.CombinedOutputFn = func(n string, a ...string) ([]byte, error) {
		b, e := output(n, a...)
		if n == "systemctl" && len(a) > 1 && a[1] == "user.slice" {
			b = []byte(strings.ReplaceAll(string(b), " "+external, ""))
			b = []byte(strings.ReplaceAll(string(b), external+" ", ""))
		}
		return b, e
	}
	got, err := newTestHandler().Apply(linuxHost(), newTestHandler().Inspect(linuxHost(), baseReq()), hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil || !got.Applied {
		t.Fatalf("known stronger intent failed safe convergence: %+v %v calls=%v", got, err, *calls)
	}
	if files[testUserCgroup+"/memory.min"] != "34359738368" || files[testUserCgroup+"/memory.low"] != "42949672960" {
		t.Fatalf("stronger floors weakened: %v", *calls)
	}
	if files[external] != "[Slice]\nMemoryMin=32G\nMemoryLow=40G\n" {
		t.Fatal("operator settings overwritten")
	}
}

func TestReservationDefinitionLayerSpecificityAndSearchPathPrecedence(t *testing.T) {
	for _, mode := range []string{"generic nonmemory", "exact beats prefix", "etc beats vendor", "typewide reset"} {
		t.Run(mode, func(t *testing.T) {
			files, _ := reservationFixture(t)
			files[ancestorSlicePath] = ancestorSliceContent(4915, 6963)
			switch mode {
			case "generic nonmemory":
				files["/usr/lib/systemd/system/user-.slice.d/10-defaults.conf"] = "[Slice]\nTasksMax=33%\n"
			case "exact beats prefix":
				files["/usr/lib/systemd/system/user-.slice.d/99-policy.conf"] = "[Slice]\nMemoryMin=0\nMemoryLow=0\n"
				files["/etc/systemd/system/user-1000.slice.d/99-policy.conf"] = "[Slice]\nMemoryMin=8G\nMemoryLow=10G\n"
				files[testUserCgroup+"/user-1000.slice/memory.min"] = "8589934592"
				files[testUserCgroup+"/user-1000.slice/memory.low"] = "10737418240"
			case "etc beats vendor":
				files["/usr/lib/systemd/system/user.slice.d/99-policy.conf"] = "[Slice]\nMemoryMin=0\nMemoryLow=0\n"
				files["/etc/systemd/system/user.slice.d/99-policy.conf"] = "[Slice]\nMemoryMin=8G\nMemoryLow=10G\n"
				files[testUserCgroup+"/memory.min"] = "8589934592"
				files[testUserCgroup+"/memory.low"] = "10737418240"
			case "typewide reset":
				files["/etc/systemd/system/slice.d/99-all.conf"] = "[Slice]\nMemoryMin=0\nMemoryLow=0\n"
			}
			got := newTestHandler().Inspect(linuxHost(), baseReq())
			if got.Applied == (mode == "typewide reset") {
				t.Fatalf("wrong effective layer result mode=%s status=%+v", mode, got)
			}
		})
	}
}

func TestReservationDefinitionsRejectUnknownObservationsBeforeEffects(t *testing.T) {
	for _, mode := range []string{"percent memory", "include", "alias", "unreadable directory", "unbounded directory", "missing loaded dropin", "wrong manager cgroup", "empty manager field", "runtime-only floor"} {
		t.Run(mode, func(t *testing.T) {
			files, calls := reservationFixture(t)
			files[ancestorSlicePath] = ancestorSliceContent(4915, 6963)
			output := hostreqkit.CombinedOutputFn
			switch mode {
			case "percent memory":
				files["/etc/systemd/system/user.slice.d/99-operator.conf"] = "[Slice]\nMemoryMin=50%\n"
			case "include":
				files["/etc/systemd/system/user.slice.d/99-operator.conf"] = ".include /foreign.conf\n"
			case "alias", "wrong manager cgroup", "empty manager field", "missing loaded dropin":
				hostreqkit.CombinedOutputFn = func(n string, a ...string) ([]byte, error) {
					b, e := output(n, a...)
					if n == "systemctl" && len(a) > 1 && a[1] == "user.slice" {
						s := string(b)
						switch mode {
						case "alias":
							s = strings.Replace(s, "Names=user.slice", "Names=user.slice foreign.slice", 1)
						case "empty manager field":
							s = strings.Replace(s, "MemoryMin="+files[testUserCgroup+"/memory.min"], "MemoryMin=", 1)
						case "wrong manager cgroup":
							s = strings.Replace(s, "ControlGroup=/user.slice", "ControlGroup=/foreign.slice", 1)
						case "missing loaded dropin":
							s = strings.Replace(s, "DropInPaths=", "DropInPaths=/etc/systemd/system/user.slice.d/missing.conf ", 1)
						}
						b = []byte(s)
					}
					return b, e
				}
			case "unreadable directory":
				reservationReadDir = func(string) ([]os.DirEntry, error) { return nil, os.ErrPermission }
			case "unbounded directory":
				reservationReadDir = func(string) ([]os.DirEntry, error) {
					r := make([]os.DirEntry, 257)
					for i := range r {
						r[i] = reservationTestEntry(fmt.Sprintf("%d.conf", i))
					}
					return r, nil
				}
			case "runtime-only floor":
				delete(files, ancestorSlicePath)
				files["/run/systemd/system/user.slice.d/99-runtime.conf"] = "[Slice]\nMemoryMin=8G\nMemoryLow=10G\n"
				files["/etc/systemd/system/user.slice.d/99-operator.conf"] = "[Slice]\nMemoryMin=0\nMemoryLow=0\n"
			}
			got, err := newTestHandler().Apply(linuxHost(), newTestHandler().Inspect(linuxHost(), baseReq()), hostreqkit.EnsureOptions{SudoMode: "ask"})
			if err != nil || got.Applied || len(*calls) != 0 {
				t.Fatalf("unknown definitions caused effects mode=%s status=%+v error=%v calls=%v", mode, got, err, *calls)
			}
		})
	}
}

func TestReservationDefinitionRejectsRealPrivateParentSymlink(t *testing.T) {
	t.Cleanup(stubAll(t))
	root := t.TempDir()
	foreign := filepath.Join(root, "foreign")
	if err := os.Mkdir(foreign, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "owned")
	if err := os.Symlink(foreign, alias); err != nil {
		t.Fatal(err)
	}
	reservationLstat = os.Lstat
	hostreqkit.ReadFileFn = os.ReadFile
	_, _, write, err := reservationDefinition(filepath.Join(alias, "50-memory-protect.conf"), memoryAllocation{desktopMinMB: 4915, desktopLowMB: 6963})
	if err == nil || write {
		t.Fatal("foreign parent alias admitted for managed write")
	}
	entries, err := os.ReadDir(foreign)
	if err != nil || len(entries) != 0 {
		t.Fatal("qualification changed foreign directory")
	}
}
