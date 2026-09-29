package exec

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/types"
)

var liveBwrapAliases = flag.Bool("live-bwrap-aliases", false, "Verify workspace aliases with the installed Linux bubblewrap")

func TestLivePolicyFiles(t *testing.T) {
	if !*liveBwrapAliases {
		t.Skip("pass -live-bwrap-aliases for the no-model host check")
	}
	root := t.TempDir()
	merged := filepath.Join(root, "merged")
	if err := os.Mkdir(merged, 0700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "owner-policy")
	if err := os.WriteFile(source, []byte("owner-policy"), 0600); err != nil {
		t.Fatal(err)
	}
	sb := &types.Sandbox{ID: uuid.New(), MergedDir: merged, LowerDir: merged}
	cfg := BwrapConfig{ReadOnlyBinds: map[string]string{}, PolicyFiles: []types.PolicyFile{{Source: source, Target: "/etc/fixture/requirements.toml", SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("owner-policy")))}}}
	for _, dir := range []string{"/bin", "/usr", "/lib", "/lib64"} {
		if _, err := os.Stat(dir); err == nil {
			cfg.ReadOnlyBinds[dir] = dir
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := Exec(ctx, process.NewOSExecStarter(), sb, driver.ContainmentRequired, cfg, "/bin/sh", "-c", `set -eu
for file in "$@"; do
test "$(cat "$file")" = owner-policy
if (printf bypass > "$file") 2>/dev/null; then exit 11; fi
if rm "$file" 2>/dev/null; then exit 12; fi
done`, "probe", source, cfg.PolicyFiles[0].Target)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("policy aliases must be readable and immutable: %+v, %v", result, err)
	}
}

func TestLiveWorkspaceWritePolicy(t *testing.T) {
	if !*liveBwrapAliases {
		t.Skip("pass -live-bwrap-aliases for the no-model host check")
	}
	root := t.TempDir()
	merged, home := filepath.Join(root, "merged"), filepath.Join(root, "home")
	for _, dir := range []string{merged, home, filepath.Join(merged, "src"), filepath.Join(merged, "controls"), filepath.Join(merged, ".git")} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	control := filepath.Join(merged, "controls", "boundary.json")
	if err := os.WriteFile(control, []byte("owner"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../controls/boundary.json", filepath.Join(merged, "src", "escape")); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "project")
	sb := &types.Sandbox{ID: uuid.New(), MergedDir: merged, LowerDir: merged, HomeMergedDir: home, ProjectRoot: project,
		Behavior: types.SandboxBehavior{WritePolicy: &types.WorkspaceWritePolicy{Paths: []string{"src"}}}}
	cfg := BwrapConfig{HostHome: "/home/fixture", MirrorProjectRoot: true, ReadOnlyBinds: map[string]string{}}
	for _, dir := range []string{"/bin", "/usr", "/lib", "/lib64"} {
		if _, err := os.Stat(dir); err == nil {
			cfg.ReadOnlyBinds[dir] = dir
		}
	}
	for _, alias := range []string{"/workspace", project, merged} {
		t.Run(alias, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// Exercise shell redirection, new files, deletion, rename, symlink
			// escape and git metadata, not just the argument builder.
			script := `set -eu
cd "$1"
printf allowed > src/allowed
test "$(cat controls/boundary.json)" = owner
if (printf forbidden > controls/boundary.json) 2>/dev/null; then exit 11; fi
if (printf forbidden > controls/new) 2>/dev/null; then exit 12; fi
if rm controls/boundary.json 2>/dev/null; then exit 13; fi
if mv controls controls-old 2>/dev/null; then exit 14; fi
if (printf forbidden > src/escape) 2>/dev/null; then exit 15; fi
if (printf forbidden > .git/index) 2>/dev/null; then exit 16; fi
test "$(cat controls/boundary.json)" = owner
`
			result, err := Exec(ctx, process.NewOSExecStarter(), sb, driver.ContainmentRequired, cfg, "/bin/sh", "-c", script, "check", alias)
			if err != nil {
				t.Fatal(err)
			}
			if result.ExitCode != 0 {
				t.Fatalf("write policy failed: %+v", result)
			}
		})
	}
	// An explicit empty grant stays read-only even when ordinary project
	// mirroring is disabled. It must not degrade into an omitted policy.
	sb.Behavior.WritePolicy.Paths = nil
	cfg.MirrorProjectRoot = false
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := Exec(ctx, process.NewOSExecStarter(), sb, driver.ContainmentRequired, cfg,
		"/bin/sh", "-c", `if (printf forbidden > "$1/src/allowed") 2>/dev/null; then exit 1; fi; test "$(cat "$1/controls/boundary.json")" = owner`, "check", project)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("empty write policy: result=%+v error=%v", result, err)
	}
}

func TestLiveWorkspaceAliasesUnderTemporaryDirectory(t *testing.T) {
	if !*liveBwrapAliases {
		t.Skip("pass -live-bwrap-aliases for the no-model host check")
	}
	root := t.TempDir()
	merged, home := filepath.Join(root, "merged"), filepath.Join(root, "home")
	for _, dir := range []string{merged, home} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(merged, "seed"), []byte("visible overlay"), 0600); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "project")
	sandbox := &types.Sandbox{ID: uuid.New(), MergedDir: merged, LowerDir: merged, HomeMergedDir: home, ProjectRoot: project}
	cfg := BwrapConfig{HostHome: "/home/fixture", MirrorProjectRoot: true, WorkingDir: project, ReadOnlyBinds: map[string]string{}}
	for _, dir := range []string{"/bin", "/usr", "/lib", "/lib64"} {
		if _, err := os.Stat(dir); err == nil {
			cfg.ReadOnlyBinds[dir] = dir
		}
	}
	for _, alias := range []string{project, merged} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		command, args := BuildExecCommand(sandbox, cfg, "/usr/bin/cmp", "/workspace/seed", filepath.Join(alias, "seed"))
		output, err := osexec.CommandContext(ctx, command, args...).CombinedOutput()
		cancel()
		if err != nil {
			t.Fatalf("alias %s is shadowed: %v: %s", alias, err, output)
		}
	}
}

func TestIsBwrapAvailable(t *testing.T) {
	ctx := context.Background()
	starter := process.NewOSExecStarter()
	_, lookErr := osexec.LookPath("bwrap")
	if lookErr != nil {
		// Not installed in CI; assert the probe agrees.
		available, _, _ := driver.IsBwrapAvailable(ctx, starter)
		if available {
			t.Error("driver.IsBwrapAvailable returned true but bwrap is not in PATH")
		}
		return
	}
	available, version, err := driver.IsBwrapAvailable(ctx, starter)
	if !available {
		t.Errorf("driver.IsBwrapAvailable returned false but bwrap is installed: %v", err)
	}
	if version == "" {
		t.Log("warning: bwrap version string is empty")
	}
}

func TestIsProcessRunning(t *testing.T) {
	if !IsProcessRunning(os.Getpid()) {
		t.Error("current process should be running")
	}
	if IsProcessRunning(999999) {
		t.Error("non-existent process should not be running")
	}
}

func TestExitInfoFromState_NormalExit(t *testing.T) {
	// Running a successful command and reading its state.
	cmd := osexec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("`true` failed unexpectedly: %v", err)
	}
	exitCode, signal, oom := ExitInfoFromState(cmd.ProcessState, nil)
	if exitCode != 0 || signal != 0 || oom {
		t.Errorf("ExitInfoFromState(true) = (%d, %d, %v), want (0, 0, false)", exitCode, signal, oom)
	}
}

func TestExitInfoFromState_NonZeroExit(t *testing.T) {
	cmd := osexec.Command("sh", "-c", "exit 7")
	err := cmd.Run()
	exitCode, signal, oom := ExitInfoFromState(cmd.ProcessState, err)
	if exitCode != 7 || signal != 0 || oom {
		t.Errorf("ExitInfoFromState(exit 7) = (%d, %d, %v), want (7, 0, false)", exitCode, signal, oom)
	}
}

// TestExec_TimeoutReturns124 verifies that wall-clock timeout enforcement
// returns the standard exit code 124 — required for handlers/process.go to
// surface TimedOut=true to the client.
func TestExec_TimeoutReturns124(t *testing.T) {
	tmp := t.TempDir()
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: tmp,
		LowerDir:  tmp,
	}
	cfg := DefaultBwrapConfig()
	cfg.ResourceLimits.TimeoutSec = 1
	// driver.ContainmentNone runs in s.MergedDir directly, no bwrap dependency.
	result, err := Exec(context.Background(), process.NewOSExecStarter(), sandbox, driver.ContainmentNone, cfg, "sh", "-c", "sleep 5")
	if err != nil {
		t.Fatalf("Exec returned unexpected error: %v", err)
	}
	if result.ExitCode != 124 {
		t.Errorf("expected ExitCode=124 on timeout, got %d", result.ExitCode)
	}
	if result.Error == nil {
		t.Error("expected non-nil result.Error on timeout")
	}
}

// TestStartProcess_OnExitFiresExactlyOnce locks in the OnExit reaper
// invariant: the callback is invoked exactly once per StartProcess, even
// when the process exits very quickly.
func TestStartProcess_OnExitFiresExactlyOnce(t *testing.T) {
	tmp := t.TempDir()
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: tmp,
		LowerDir:  tmp,
	}
	cfg := DefaultBwrapConfig()
	var fires atomic.Int32
	done := make(chan struct{})
	cfg.OnExit = func(exitCode, signal int, oomKilled bool) {
		fires.Add(1)
		close(done)
	}

	pid, _, err := StartProcess(context.Background(), process.NewOSExecStarter(), sandbox, driver.ContainmentNone, cfg, "true")
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	if pid <= 0 {
		t.Errorf("expected positive PID, got %d", pid)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("OnExit did not fire within 5s")
	}

	// Sleep a moment to catch any spurious second invocation.
	time.Sleep(50 * time.Millisecond)
	if got := fires.Load(); got != 1 {
		t.Errorf("OnExit fired %d times, want exactly 1", got)
	}
}

// TestStartProcess_WaitHappensWhenOnExitNil ensures we still reap zombies
// when OnExit is not configured (background processes always need a
// goroutine that calls Wait).
func TestStartProcess_WaitHappensWhenOnExitNil(t *testing.T) {
	tmp := t.TempDir()
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: tmp,
		LowerDir:  tmp,
	}
	cfg := DefaultBwrapConfig()
	cfg.OnExit = nil

	// Use a marker file to know when the child has exited; that's an
	// indirect signal that the reaper goroutine ran (cmd.Wait returned).
	marker := filepath.Join(tmp, "done")
	pid, _, err := StartProcess(context.Background(), process.NewOSExecStarter(), sandbox, driver.ContainmentNone, cfg, "sh", "-c", "touch "+marker)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	if pid <= 0 {
		t.Fatalf("expected positive PID, got %d", pid)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("child never wrote marker file: %v", err)
	}

	// Give the reaper a chance to call Wait. If it didn't, the process
	// would still be in 'Z' state — best-effort check via /proc.
	time.Sleep(100 * time.Millisecond)
	if data, err := os.ReadFile(filepath.Join("/proc", itoa(pid), "status")); err == nil {
		if containsString(string(data), "State:\tZ") {
			t.Error("child is a zombie — Wait was not called by the reaper")
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func containsString(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
