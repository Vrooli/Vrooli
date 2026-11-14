package remotesessionprotection

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const testAncestorPath = "/etc/systemd/system/user.slice.d/90-vrooli-remote-session-protection.conf"
const testUserCgroup = "/sys/fs/cgroup/user.slice"

// This fixture models acknowledged manager effects and kernel observations;
// no systemd, cgroup, swap or file installation is executed on the host.
func reservationFixture(t *testing.T) (map[string]string, *[]string) {
	t.Helper()
	t.Cleanup(stubAll(t))
	alloc := memoryAllocation{desktopMinMB: 4915, desktopLowMB: 6963}
	files := map[string]string{sysctlPath: sysctlContent, systemdPath: unitContent, logindPath: logindContent,
		procMeminfo: fakeMeminfo(32, 32), desktopSlicePath: desktopSliceContent(alloc), workloadSlicePath: workloadSliceContent()}
	for _, unit := range []string{"", "/user-1000.slice"} {
		files[testUserCgroup+unit+"/memory.min"] = strconv.Itoa(alloc.desktopMinMB * bytesPerMiB)
		files[testUserCgroup+unit+"/memory.low"] = strconv.Itoa(alloc.desktopLowMB * bytesPerMiB)
	}
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if s, ok := files[path]; ok {
			return []byte(s), nil
		}
		return nil, os.ErrNotExist
	}
	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "systemctl", "install", "mkdir", "sudo":
			return "/usr/bin/" + name, nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" && len(args) > 0 && args[0] == "list-unit-files" {
			return []byte("gdm3.service enabled\n"), nil
		}
		return nil, fmt.Errorf("unexpected read command: %s", name)
	}
	var calls []string
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if name == "sudo" {
			name = args[0]
			args = args[1:]
		}
		switch name {
		case "mkdir":
			return nil
		case "install":
			b, err := os.ReadFile(args[len(args)-2])
			if err != nil {
				return err
			}
			files[args[len(args)-1]] = string(b)
			return nil
		case "systemctl":
			if len(args) != 5 || args[0] != "set-property" || args[1] != "--runtime" {
				return fmt.Errorf("unexpected effect: %v", args)
			}
			dir := testUserCgroup
			if args[2] == "user-1000.slice" {
				dir += "/user-1000.slice"
			} else if args[2] != "user.slice" {
				return fmt.Errorf("unexpected unit: %s", args[2])
			}
			for _, prop := range args[3:] {
				k, v, ok := strings.Cut(prop, "=")
				if !ok {
					return fmt.Errorf("bad property")
				}
				attr := "memory.min"
				if k == "MemoryLow" {
					attr = "memory.low"
				}
				files[dir+"/"+attr] = v
			}
			return nil
		}
		return fmt.Errorf("unexpected effect: %s %v", name, args)
	}
	installReservationManagerFixture(t, files)
	return files, &calls
}

func TestInspectDesktopReservationRequiresEffectiveAncestors(t *testing.T) {
	for _, mode := range []string{"zero ancestor", "missing ancestor", "invalid ancestor", "unreadable ancestor", "zero child"} {
		t.Run(mode, func(t *testing.T) {
			files, _ := reservationFixture(t)
			// Preserve the old managed file while independently challenging its live proof.
			switch mode {
			case "zero ancestor":
				files[testUserCgroup+"/memory.min"] = "0"
			case "missing ancestor":
				delete(files, testUserCgroup+"/memory.low")
			case "invalid ancestor":
				files[testUserCgroup+"/memory.low"] = "garbage"
			case "zero child":
				files[testUserCgroup+"/user-1000.slice/memory.low"] = "0"
			case "unreadable ancestor":
				read := hostreqkit.ReadFileFn
				hostreqkit.ReadFileFn = func(p string) ([]byte, error) {
					if p == testUserCgroup+"/memory.min" {
						return nil, os.ErrPermission
					}
					return read(p)
				}
			}
			status := newTestHandler().Inspect(linuxHost(), baseReq())
			if status.Applied {
				t.Fatalf("unverified reservation reported applied: %v", status.Notes)
			}
		})
	}
}

func TestApplyDesktopReservationOnlyConvergesAndVerifies(t *testing.T) {
	files, calls := reservationFixture(t)
	files[testUserCgroup+"/memory.min"] = "0"
	files[testUserCgroup+"/memory.low"] = "0"
	before := files[desktopSlicePath]
	status := newTestHandler().Inspect(linuxHost(), baseReq())
	got, err := newTestHandler().Apply(linuxHost(), status, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil || !got.Applied {
		t.Fatalf("reservation convergence failed: %+v %v", got, err)
	}
	if files[desktopSlicePath] != before {
		t.Fatal("sufficient existing child definition changed")
	}
	if files[testAncestorPath] == "" {
		t.Fatal("ancestor boot definition missing")
	}
	for _, call := range *calls {
		for _, forbidden := range []string{"sysctl", "daemon-reload", "docker", "swap", "logind", "user@", "restart", "stop"} {
			if strings.Contains(call, forbidden) {
				t.Fatalf("reservation-only repair widened effects: %s", call)
			}
		}
	}
	if !newTestHandler().Inspect(linuxHost(), baseReq()).Applied {
		t.Fatal("convergence did not establish both file and live proof")
	}
	n := len(*calls)
	again := newTestHandler().Inspect(linuxHost(), baseReq())
	got, err = newTestHandler().Apply(linuxHost(), again, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil || !got.Applied || len(*calls) != n {
		t.Fatal("verified repair was not idempotent")
	}
}

func TestDesktopReservationPreservesStrongerAndUnknownSettings(t *testing.T) {
	for _, mode := range []string{"stronger", "unknown child", "unknown ancestor"} {
		t.Run(mode, func(t *testing.T) {
			files, calls := reservationFixture(t)
			if mode == "stronger" {
				files[desktopSlicePath] = desktopSliceContent(memoryAllocation{desktopMinMB: 8192, desktopLowMB: 10240})
				for _, u := range []string{"", "/user-1000.slice"} {
					files[testUserCgroup+u+"/memory.min"] = "12884901888"
					files[testUserCgroup+u+"/memory.low"] = "17179869184"
				}
			} else if mode == "unknown child" {
				files[desktopSlicePath] = "[Slice]\nMemoryMin=12G\n# operator-owned\n"
			} else {
				files[testAncestorPath] = "[Slice]\nMemoryLow=infinity\n# operator-owned\n"
			}
			beforeChild, beforeAncestor := files[desktopSlicePath], files[testAncestorPath]
			got, err := newTestHandler().Apply(linuxHost(), newTestHandler().Inspect(linuxHost(), baseReq()), hostreqkit.EnsureOptions{SudoMode: "ask"})
			if mode == "stronger" {
				if err != nil || !got.Applied {
					t.Fatalf("stronger allocation rejected: %+v %v", got, err)
				}
				if files[desktopSlicePath] != beforeChild {
					t.Fatal("stronger persistent definition lowered")
				}
				if files[testUserCgroup+"/memory.min"] != "12884901888" {
					t.Fatal("stronger effective allocation lowered")
				}
			} else {
				if got.Applied {
					t.Fatal("unknown owned path reported applied")
				}
				if len(*calls) != 0 {
					t.Fatalf("unknown definition caused effects before ownership qualification: %v", *calls)
				}
				if files[desktopSlicePath] != beforeChild || files[testAncestorPath] != beforeAncestor {
					t.Fatal("unknown operator settings overwritten")
				}
			}
		})
	}
}

func TestDesktopReservationAcknowledgementsAndPostconditions(t *testing.T) {
	for _, mode := range []string{"command failure", "lost acknowledgement", "false success", "readback failure", "persistent lost acknowledgement", "persistent false success"} {
		t.Run(mode, func(t *testing.T) {
			files, calls := reservationFixture(t)
			files[testUserCgroup+"/memory.min"] = "0"
			run := hostreqkit.RunCommandFn
			hostreqkit.RunCommandFn = func(name string, args []string, o hostreqkit.EnsureOptions) error {
				joined := name + " " + strings.Join(args, " ")
				if strings.Contains(joined, "set-property") {
					switch mode {
					case "command failure":
						*calls = append(*calls, joined)
						return fmt.Errorf("injected manager refusal")
					case "false success":
						*calls = append(*calls, joined)
						return nil
					case "readback failure":
						err := run(name, args, o)
						delete(files, testUserCgroup+"/memory.low")
						return err
					case "lost acknowledgement":
						if err := run(name, args, o); err != nil {
							return err
						}
						return fmt.Errorf("injected lost reply")
					}
				}
				if strings.Contains(joined, "install ") && strings.HasSuffix(joined, testAncestorPath) {
					switch mode {
					case "persistent false success":
						*calls = append(*calls, joined)
						return nil
					case "persistent lost acknowledgement":
						if err := run(name, args, o); err != nil {
							return err
						}
						return fmt.Errorf("injected lost reply")
					}
				}
				return run(name, args, o)
			}
			got, err := newTestHandler().Apply(linuxHost(), newTestHandler().Inspect(linuxHost(), baseReq()), hostreqkit.EnsureOptions{SudoMode: "ask"})
			success := mode == "lost acknowledgement" || mode == "persistent lost acknowledgement"
			if got.Applied != success {
				t.Fatalf("postcondition truth lost mode=%s status=%+v error=%v", mode, got, err)
			}
		})
	}
}

// Existing setup command tests also need an actual effective-reservation
// observation and exact installed-file readback; a nil command reply alone is
// deliberately insufficient. Failure and live-convergence edges are above.
func modelReservationReadback(t *testing.T) {
	t.Helper()
	alloc, err := calculateMemory()
	if err != nil {
		t.Fatal(err)
	}
	read, run := hostreqkit.ReadFileFn, hostreqkit.RunCommandFn
	files := map[string]string{}
	for _, unit := range []string{"", "/user-1000.slice"} {
		files[testUserCgroup+unit+"/memory.min"] = strconv.Itoa(alloc.desktopMinMB * bytesPerMiB)
		files[testUserCgroup+unit+"/memory.low"] = strconv.Itoa(alloc.desktopLowMB * bytesPerMiB)
	}
	hostreqkit.ReadFileFn = func(p string) ([]byte, error) {
		if v, ok := files[p]; ok {
			return []byte(v), nil
		}
		return read(p)
	}
	hostreqkit.RunCommandFn = func(name string, args []string, o hostreqkit.EnsureOptions) error {
		err := run(name, args, o)
		if err != nil {
			return err
		}
		if name == "sudo" {
			name = args[0]
			args = args[1:]
		}
		if name == "install" && len(args) >= 4 {
			path := args[len(args)-1]
			if path == testAncestorPath || path == desktopSlicePath {
				data, err := os.ReadFile(args[len(args)-2])
				if err != nil {
					return err
				}
				files[path] = string(data)
			}
		}
		return nil
	}

	installReservationManagerFixture(t, nil)
}

func TestInspectMemoryInventoryCannotProveProtectionWhenUnreadable(t *testing.T) {
	files, _ := reservationFixture(t)
	delete(files, procMeminfo)
	status := newTestHandler().Inspect(linuxHost(), baseReq())
	if status.Applied {
		t.Fatal("unreadable required memory inventory reported applied")
	}
}

func TestEnsureSwapReportsRequiredCommandFailures(t *testing.T) {
	for _, mode := range []string{"swapoff", "persistent install", "persistent append", "swapon"} {
		t.Run(mode, func(t *testing.T) {
			_, _ = reservationFixture(t)
			read := hostreqkit.ReadFileFn
			hostreqkit.ReadFileFn = func(p string) ([]byte, error) {
				if p == procMeminfo {
					return []byte(fakeMeminfo(16, 4)), nil
				}
				if p == "/proc/swaps" {
					return []byte("Filename Type Size Used Priority\n/swapfile file 4194304 0 -2\n"), nil
				}
				if p == remoteSessionFstab {
					return []byte("# no managed swap\n"), nil
				}
				return read(p)
			}
			hostreqkit.RunCommandFn = func(name string, args []string, o hostreqkit.EnsureOptions) error {
				call := name + " " + strings.Join(args, " ")
				fail := (mode == "swapoff" && strings.Contains(call, "swapoff")) || (mode == "persistent install" && strings.Contains(call, "fstab.vrooli-swap") && strings.Contains(call, "install")) || (mode == "persistent append" && strings.Contains(call, "cat /etc/fstab.vrooli-swap")) || (mode == "swapon" && strings.Contains(call, "swapon"))
				if fail {
					return fmt.Errorf("injected %s refusal", mode)
				}
				return nil
			}
			hostreqkit.LookPathFn = func(n string) (string, error) { return "/usr/bin/" + n, nil }
			if err := ensureSwap(memoryAllocation{targetSwapGB: 16}, hostreqkit.EnsureOptions{SudoMode: "ask"}); err == nil {
				t.Fatal("required swap failure was ignored")
			}
		})
	}
}

type reservationTestFileInfo struct {
	size int64
	mode os.FileMode
}

func (f reservationTestFileInfo) Name() string       { return "reservation" }
func (f reservationTestFileInfo) Size() int64        { return f.size }
func (f reservationTestFileInfo) Mode() os.FileMode  { return f.mode }
func (f reservationTestFileInfo) ModTime() time.Time { return time.Time{} }
func (f reservationTestFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f reservationTestFileInfo) Sys() any           { return nil }

func TestDesktopReservationRefusesNonRegularDefinitions(t *testing.T) {
	for _, kind := range []os.FileMode{os.ModeSymlink, os.ModeDir, os.ModeNamedPipe} {
		t.Run(kind.String(), func(t *testing.T) {
			_, calls := reservationFixture(t)
			old := reservationLstat
			reservationLstat = func(p string) (os.FileInfo, error) {
				if p == desktopSlicePath {
					return reservationTestFileInfo{mode: kind}, nil
				}
				return old(p)
			}
			got, err := newTestHandler().Apply(linuxHost(), newTestHandler().Inspect(linuxHost(), baseReq()), hostreqkit.EnsureOptions{SudoMode: "ask"})
			if err != nil || got.Applied || len(*calls) != 0 {
				t.Fatalf("unqualified path caused effects: %+v %v %v", got, err, *calls)
			}
		})
	}
}
