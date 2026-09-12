//go:build linux

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testSlice = "vrooli-test.slice"

// userManagerReachable reports whether the user manager answers; the tests
// that create real scopes need the session bus of the executing user.
func userManagerReachable(t *testing.T) {
	t.Helper()
	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		t.Setenv("XDG_RUNTIME_DIR", fmt.Sprintf("/run/user/%d", os.Getuid()))
	}
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+os.Getenv("XDG_RUNTIME_DIR")+"/bus")
	}
	if err := exec.Command("systemctl", "--user", "show", "-p", "Version", "--value").Run(); err != nil {
		t.Skipf("user manager unreachable: %v", err)
	}
}

func startContainedSleep(t *testing.T, scope string) *Contained {
	t.Helper()
	contained, err := ContainedCommand(ContainedSpec{
		Path:        "/bin/sleep",
		Args:        []string{"30"},
		Scope:       scope,
		Containment: Containment{Slice: testSlice, CPUWeight: 50, MemoryMax: "200M", TasksMax: 64},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := contained.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = contained.Cmd.Process.Kill()
		_, _ = contained.Cmd.Process.Wait()
		contained.Release()
	})
	return contained
}

func TestContainedCommandLandsInScopeOnLinux(t *testing.T) {
	userManagerReachable(t)
	scope := fmt.Sprintf("vrooli-test-%d", os.Getpid())
	contained := startContainedSleep(t, scope)
	if contained.Method != MethodSystemdRun {
		t.Fatalf("method = %q, want %q", contained.Method, MethodSystemdRun)
	}
	got, err := ProcessScope(contained.Cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, testSlice+"/"+scope+".scope") {
		t.Fatalf("child cgroup %q is not under %s/%s.scope", got, testSlice, scope)
	}
	if contained.Scope.Kind != ScopeKindCgroup || contained.Scope.Path != got {
		t.Fatalf("scope ref %+v does not name the child's cgroup %q", contained.Scope, got)
	}
	root := filepath.Join(cgroupMount, got)
	pids, _ := os.ReadFile(filepath.Join(root, "pids.max"))
	mem, _ := os.ReadFile(filepath.Join(root, "memory.max"))
	if strings.TrimSpace(string(pids)) != "64" || strings.TrimSpace(string(mem)) != "209715200" {
		t.Fatalf("limits not applied: pids.max=%q memory.max=%q", pids, mem)
	}
}

func TestContainedCommandFallsBackToCgroupWriteWithoutSystemdRun(t *testing.T) {
	// The fallback creates a cgroup under this process's own cgroup, which
	// must be one the user owns. A test started from a root-owned session
	// scope (a tool shell, CI) re-runs itself inside a delegated user scope
	// so the path is exercised for real rather than skipped.
	if os.Getenv("PLATFORM_GO_DELEGATED_HELPER") != "1" {
		own, _ := ProcessScope(os.Getpid())
		if probe := filepath.Join(cgroupMount, filepath.Dir(own), "vrooli-test-probe"); os.Mkdir(probe, 0o755) != nil {
			userManagerReachable(t)
			cmd := exec.Command("systemd-run", "--user", "--scope", "--quiet", "-p", "Delegate=yes", os.Args[0], "-test.run", "^TestContainedCommandFallsBackToCgroupWriteWithoutSystemdRun$", "-test.v")
			cmd.Env = append(os.Environ(), "PLATFORM_GO_DELEGATED_HELPER=1")
			output, err := cmd.CombinedOutput()
			if err != nil || !strings.Contains(string(output), "--- PASS") {
				t.Fatalf("fallback inside a delegated scope: %v\n%s", err, output)
			}
			return
		} else {
			_ = os.Remove(probe)
		}
	}
	dir := t.TempDir()
	for _, tool := range []string{"sleep"} {
		path, err := exec.LookPath(tool)
		if err != nil {
			t.Skip(err)
		}
		if err := os.Symlink(path, filepath.Join(dir, tool)); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	contained, err := ContainedCommand(ContainedSpec{
		Path:        filepath.Join(dir, "sleep"),
		Args:        []string{"30"},
		Scope:       fmt.Sprintf("vrooli-test-fallback-%d", os.Getpid()),
		Containment: Containment{Slice: testSlice, MemoryMax: "200M", TasksMax: 64},
	})
	if err != nil {
		if strings.Contains(err.Error(), "undetermined") {
			t.Skipf("cgroup-write fallback cannot run from this process's cgroup: %v", err)
		}
		t.Fatal(err)
	}
	if contained.Method != MethodCgroupWrite {
		t.Fatalf("method = %q, want %q", contained.Method, MethodCgroupWrite)
	}
	if err := contained.Start(); err != nil {
		if strings.Contains(err.Error(), "undetermined") {
			t.Skipf("cgroup-write fallback cannot run from this process's cgroup: %v", err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = contained.Cmd.Process.Kill()
		_, _ = contained.Cmd.Process.Wait()
		contained.Release()
	})
	got, _ := ProcessScope(contained.Cmd.Process.Pid)
	if !strings.HasSuffix(got, contained.Scope.Path) && got != contained.Scope.Path {
		t.Fatalf("child cgroup %q is not the fallback scope %q", got, contained.Scope.Path)
	}
	pids, _ := os.ReadFile(filepath.Join(cgroupMount, got, "pids.max"))
	if strings.TrimSpace(string(pids)) != "64" {
		t.Fatalf("pids.max = %q", pids)
	}
}

func TestFreezeAndThawScope(t *testing.T) {
	userManagerReachable(t)
	scope := fmt.Sprintf("vrooli-test-freeze-%d", os.Getpid())
	contained := startContainedSleep(t, scope)
	if err := FreezeScope(contained.Scope); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		frozen, err := ScopeFrozen(contained.Scope)
		if err != nil {
			t.Fatal(err)
		}
		if frozen {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("scope never reported frozen")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := ThawScope(contained.Scope); err != nil {
		t.Fatal(err)
	}
	if frozen, _ := ScopeFrozen(contained.Scope); frozen {
		t.Fatal("scope still frozen after thaw")
	}
	self, _ := ProcessScope(os.Getpid())
	if err := FreezeScope(ScopeRef{Kind: ScopeKindCgroup, Path: "/user.slice/user-1000.slice/user@1000.service"}); err == nil {
		t.Fatal("froze a cgroup outside the agent slices")
	}
	if strings.Contains(self, testSlice) {
		t.Logf("note: the test process itself runs under %s", self)
	}
}

func TestContainSelfAdoptsCallingProcess(t *testing.T) {
	userManagerReachable(t)
	if os.Getenv("PLATFORM_GO_CONTAIN_SELF_HELPER") == "1" {
		scope := fmt.Sprintf("vrooli-test-self-%d", os.Getpid())
		ref, method, err := ContainSelf(scope, Containment{Slice: testSlice, TasksMax: 64, MemoryMax: "200M"})
		if err != nil {
			fmt.Fprintln(os.Stderr, "contain-self:", err)
			os.Exit(3)
		}
		got, _ := ProcessScope(os.Getpid())
		fmt.Printf("%s\n%s\n%s\n", method, ref.String(), got)
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run", "^TestContainSelfAdoptsCallingProcess$")
	cmd.Env = append(os.Environ(), "PLATFORM_GO_CONTAIN_SELF_HELPER=1")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("helper failed: %v\n%s", err, output)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 3 {
		t.Fatalf("helper output %q", output)
	}
	method, ref, cgroup := lines[0], lines[1], lines[2]
	if method != MethodTransientUnit && method != MethodCgroupWrite {
		t.Fatalf("method %q", method)
	}
	if !strings.Contains(cgroup, testSlice+"/vrooli-test-self-") || !strings.HasSuffix(ref, cgroup) {
		t.Fatalf("helper did not move itself: method=%s ref=%s cgroup=%s", method, ref, cgroup)
	}
	// The scope is empty once the helper exited; systemd garbage-collects it.
}
