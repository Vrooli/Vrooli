//go:build linux

package platform

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// childrenOf lists the live processes whose parent is pid.
func childrenOf(t *testing.T, pid int) []int {
	t.Helper()
	entries, err := os.ReadDir("/proc")
	if err != nil {
		t.Fatal(err)
	}
	var children []int
	for _, entry := range entries {
		child, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		stat, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			continue
		}
		fields := strings.Fields(string(stat[strings.LastIndexByte(string(stat), ')')+1:]))
		if len(fields) > 1 && fields[1] == strconv.Itoa(pid) {
			children = append(children, child)
		}
	}
	return children
}

// readHoldFixture waits for a fixture file through the package helper and
// returns its content.
func readHoldFixture(t *testing.T, path string) []byte {
	t.Helper()
	if err := waitForFile(path, 10*time.Second); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	// The file appears before its writer finishes; wait for content.
	deadline := time.Now().Add(10 * time.Second)
	for {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			return data
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s stayed empty", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// The regression: a launcher that forks at once (rootlesskit re-executes
// itself) must have no descendant when its placement runs, however long the
// placement takes, so nothing it creates can stay in the caller's scope.
func TestStartHeldTargetCannotForkBeforeRelease(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "forked")
	cmd := exec.Command("/bin/sh", "-c", `/bin/sleep 30 & echo $! > "$1"; wait`, "fixture", marker)
	held, err := StartHeld(cmd, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if data, err := os.ReadFile(marker); err == nil {
			if child, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				_ = KillProcess(child, true)
			}
		}
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	time.Sleep(200 * time.Millisecond) // ample time for an unheld target to fork
	if children := childrenOf(t, held.Pid()); len(children) != 0 {
		t.Fatalf("held process already has children %v before placement", children)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("target ran before release: %v", err)
	}
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	child := strings.TrimSpace(string(readHoldFixture(t, marker)))
	if children := childrenOf(t, held.Pid()); !slices.Contains(children, mustAtoi(t, child)) {
		t.Fatalf("released target's child %s not found under %d (%v)", child, held.Pid(), children)
	}
}

func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	n, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestExecHoldHelperProcess is the held target for identity tests: it
// records exactly what it was executed with.
func TestExecHoldHelperProcess(t *testing.T) {
	out := os.Getenv("PLATFORM_GO_EXEC_HOLD_HELPER")
	if out == "" {
		return
	}
	wd, _ := os.Getwd()
	record := append([]string{wd, strconv.Itoa(len(os.Args))}, os.Args...)
	record = append(record, os.Environ()...)
	if err := os.WriteFile(out, []byte(strings.Join(record, "\x00")), 0o600); err != nil {
		t.Fatal(err)
	}
}

// The target keeps its pid, argv tail, working directory and exact
// environment: the gate's shell bookkeeping never leaks into what the caller
// configured, whether or not the caller set the variables a shell manages.
func TestStartHeldPreservesTargetIdentity(t *testing.T) {
	for _, callerSet := range []bool{false, true} {
		t.Run("caller-set-shell-vars="+strconv.FormatBool(callerSet), func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, "out")
			env := []string{"A=1", "QUOTED=it's \"x\" $HOME", "PLATFORM_GO_EXEC_HOLD_HELPER=" + out}
			if callerSet {
				env = append(env, "PWD=/not/the/dir", "OLDPWD=/old", "SHLVL=7", "vrooli_exec_hold=caller")
			}
			args := []string{"-test.run=^TestExecHoldHelperProcess$", "two words", "", "three"}
			cmd := exec.Command(os.Args[0], args...)
			cmd.Env = env
			cmd.Dir = dir
			held, err := StartHeld(cmd, nil)
			if err != nil {
				t.Fatal(err)
			}
			pid := held.Pid()
			if err := held.Release(); err != nil {
				t.Fatal(err)
			}
			if err := cmd.Wait(); err != nil {
				t.Fatal(err)
			}
			if cmd.ProcessState.Pid() != pid {
				t.Fatalf("pid changed across release: %d -> %d", pid, cmd.ProcessState.Pid())
			}
			record := strings.Split(string(readHoldFixture(t, out)), "\x00")
			argc := mustAtoi(t, record[1])
			gotArgs, gotEnv := record[2:2+argc], record[2+argc:]
			if record[0] != dir {
				t.Fatalf("target directory = %q, want %q", record[0], dir)
			}
			if gotArgs[0] != cmd.Args[3] || !slices.Equal(gotArgs[1:], args) {
				t.Fatalf("target argv = %q, want [%s %q]", gotArgs, cmd.Args[3], args)
			}
			gotEnv = slices.DeleteFunc(gotEnv, func(entry string) bool { return strings.HasPrefix(entry, "_=") }) // bash-as-sh exports the exec path
			slices.Sort(gotEnv)
			want := slices.Clone(env)
			slices.Sort(want)
			if !slices.Equal(gotEnv, want) {
				t.Fatalf("target environment = %q, want exactly %q", gotEnv, want)
			}
		})
	}
}

// The gate's environment restoration is verified on the exec'd image itself,
// which a shell target would otherwise rewrite.
func TestHoldScriptRestoresShellManagedVariables(t *testing.T) {
	script := holdScript(3, []string{"PWD=/x/y", "SHLVL=4", "PWD=/a'b"})
	for _, want := range []string{"PWD='/a'\\''b'; export PWD", "unset OLDPWD", "SHLVL='4'; export SHLVL", "unset vrooli_exec_hold", `exec "$0" "$@"`} {
		if !strings.Contains(script, want) {
			t.Fatalf("hold script %q lacks %q", script, want)
		}
	}
	if !strings.HasPrefix(script, "IFS= read -r vrooli_exec_hold <&3 || exit 125") {
		t.Fatalf("hold script does not start at the gate: %q", script)
	}
}

func TestStartHeldAbortNeverRunsTarget(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	cmd := exec.Command("/bin/sh", "-c", `: > "$1"`, "fixture", marker)
	held, err := StartHeld(cmd, nil)
	if err != nil {
		t.Fatal(err)
	}
	held.Abort()
	err = cmd.Wait()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != holdAbortExit {
		t.Fatalf("aborted hold exit = %v, want %d", err, holdAbortExit)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("aborted target ran: %v", err)
	}
	if err := held.Release(); err != nil {
		t.Fatalf("release after abort must be inert: %v", err)
	}
}

func TestStartHeldReportsMissingTargetSynchronously(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	started := false
	_, err := StartHeld(exec.Command(missing), func(c *exec.Cmd) error { started = true; return c.Start() })
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) || !errors.Is(err, fs.ErrNotExist) || started {
		t.Fatalf("missing target err = %v started=%t; want a synchronous fork/exec error", err, started)
	}
	dir := t.TempDir()
	if _, err := StartHeld(exec.Command(dir), nil); err == nil {
		t.Fatal("a directory target must be refused before spawning")
	}
}

func TestStartHeldParentExitNeverRunsTarget(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	cmd := exec.Command("/bin/sh", "-c", `: > "$1"`, "fixture", marker)
	held, err := StartHeld(cmd, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Losing the gate's write end without a token (a crashed caller) is EOF.
	_ = held.hold.(*linuxHold).gate.Close()
	err = cmd.Wait()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != holdAbortExit {
		t.Fatalf("orphaned hold exit = %v, want %d", err, holdAbortExit)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("orphaned target ran: %v", err)
	}
}

// Opt-in host proof: a target that forks at once, placed through the real
// user manager while held, has every descendant in the new scope.
func TestStartHeldPlacesWholeTreeInNativeScope(t *testing.T) {
	if os.Getenv("PLATFORM_GO_EXEC_HOLD_SCOPE_PROBE") != "1" {
		t.Skip("set PLATFORM_GO_EXEC_HOLD_SCOPE_PROBE=1 to create a transient services-slice scope")
	}
	marker := filepath.Join(t.TempDir(), "forked")
	cmd := exec.Command("/bin/sh", "-c", `/bin/sleep 30 & echo $! > "$1"; wait`, "fixture", marker)
	held, err := StartHeld(cmd, nil)
	if err != nil {
		t.Fatal(err)
	}
	name := "vrooli-test-exec-hold-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ref, _, err := AdoptIntoScope(AdoptSpec{PID: held.Pid(), Scope: name, Slice: ServicesSlice, Description: "platform-go exec hold probe"})
	if err != nil {
		held.Abort()
		_ = cmd.Wait()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = StopScope(ref)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	child := mustAtoi(t, strings.TrimSpace(string(readHoldFixture(t, marker))))
	for _, pid := range []int{held.Pid(), child} {
		scope, err := ProcessScope(pid)
		if err != nil || scope != ref.Path {
			t.Fatalf("pid %d scope = %q (%v), want %q", pid, scope, err, ref.Path)
		}
	}
}
