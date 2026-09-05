package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/platform-go"
)

func TestBoundedRunPasses(t *testing.T) {
	cmd := helperCommand("echo")
	cmd.WorkspaceID, cmd.Name, cmd.TimeoutSeconds = "w", "echo", 10
	res := Bounded{}.Run(context.Background(), cmd)
	if res.Status != StatusPassed {
		t.Fatalf("status = %q (%s), want passed", res.Status, res.FailureReason)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit = %d, want 0", res.ExitCode)
	}
	if got := res.Stdout; got == "" {
		t.Errorf("expected captured stdout, got empty")
	}
	if res.PeakRSSBytes <= 0 {
		t.Errorf("peak RSS = %d, want positive resource evidence", res.PeakRSSBytes)
	}
}

func TestBoundedRunNonzeroIsFailed(t *testing.T) {
	cmd := helperCommand("fail")
	cmd.WorkspaceID, cmd.TimeoutSeconds = "w", 10
	res := Bounded{}.Run(context.Background(), cmd)
	if res.Status != StatusFailed {
		t.Fatalf("status = %q, want failed", res.Status)
	}
	if res.ExitCode != 3 {
		t.Errorf("exit = %d, want 3", res.ExitCode)
	}
	if res.FailureClass != ClassTestFailure {
		t.Errorf("class = %q, want test_failure", res.FailureClass)
	}
}

func TestBoundedRunTimeoutHang(t *testing.T) {
	cmd := helperCommand("sleep")
	cmd.WorkspaceID, cmd.TimeoutSeconds = "w", 1
	res := Bounded{}.Run(context.Background(), cmd)
	if res.Status != StatusTimeout {
		t.Fatalf("status = %q (%s), want timeout", res.Status, res.FailureReason)
	}
	if res.FailureClass != ClassTimeoutHang {
		t.Errorf("class = %q, want timeout_hang", res.FailureClass)
	}
}

func TestBoundedRunTimeoutReapsGrandchild(t *testing.T) {
	cmd := helperCommand("spawn-grandchild")
	cmd.WorkspaceID, cmd.Name, cmd.TimeoutSeconds = "w", "grandchild", 1
	res := Bounded{}.Run(context.Background(), cmd)
	if res.Status != StatusTimeout {
		t.Fatalf("status = %q (%s), want timeout", res.Status, res.FailureReason)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(res.Stdout))
	if err != nil || pid <= 0 {
		t.Fatalf("grandchild pid = %q, err=%v; want a positive pid", res.Stdout, err)
	}
	// The executor must contain descendants, not merely terminate the direct
	// test runner. Give the native process-group/Job Object cleanup a short
	// bounded window, then make any failure recoverable for this test process.
	t.Cleanup(func() {
		if platform.IsPIDRunning(pid) {
			_ = platform.KillProcess(pid, true)
		}
	})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !platform.IsPIDRunning(pid) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("grandchild pid %d survived timeout cleanup", pid)
}

func TestBoundedRunCancellationReapsGrandchild(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := helperCommand("spawn-grandchild")
	cmd.WorkspaceID, cmd.Name, cmd.TimeoutSeconds = "w", "cancel-grandchild", 60
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	cmd.Env["UNIT_HEALTH_GRANDCHILD_PID_FILE"] = pidFile
	resultCh := make(chan Result, 1)
	go func() { resultCh <- (Bounded{}).Run(ctx, cmd) }()

	var pid int
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if raw, readErr := os.ReadFile(pidFile); readErr == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(raw)))
			if pid > 0 {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	if pid <= 0 {
		t.Fatal("grandchild did not report readiness before cancellation")
	}
	cancel()
	res := <-resultCh
	if res.Status == StatusPassed {
		t.Fatalf("cancellation unexpectedly passed: %+v", res)
	}
	t.Cleanup(func() {
		if platform.IsPIDRunning(pid) {
			_ = platform.KillProcess(pid, true)
		}
	})
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !platform.IsPIDRunning(pid) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("grandchild pid %d survived cancellation cleanup", pid)
}

func TestBoundedRunNoOutputStall(t *testing.T) {
	cmd := helperCommand("sleep")
	cmd.WorkspaceID, cmd.TimeoutSeconds = "w", 60
	res := Bounded{NoOutputTimeout: 500 * time.Millisecond}.Run(context.Background(), cmd)
	if res.Status != StatusTimeout {
		t.Fatalf("status = %q, want timeout", res.Status)
	}
	if res.FailureClass != ClassNoOutputStall {
		t.Errorf("class = %q, want no_output_stall", res.FailureClass)
	}
}

func TestBoundedRunMissingCommand(t *testing.T) {
	res := Bounded{}.Run(context.Background(), Command{
		WorkspaceID: "w", Executable: "definitely-not-a-real-binary-xyz", TimeoutSeconds: 5,
	})
	if res.Status != StatusError || res.FailureClass != ClassMissingDependency {
		t.Fatalf("got status=%q class=%q, want error/missing_dependency", res.Status, res.FailureClass)
	}
}

func TestScrubbedEnvironBindsResolvedGoToolchain(t *testing.T) {
	t.Setenv("GOROOT", filepath.Join(t.TempDir(), "foreign-go-root"))
	t.Setenv("GOTOOLDIR", filepath.Join(t.TempDir(), "foreign-go-tools"))
	t.Setenv("UNIT_HEALTH_PRESERVED_ENV", "present")

	env := scrubbedEnviron(filepath.Join("managed", "go", "bin", "go"))
	for _, kv := range env {
		key, _, _ := strings.Cut(kv, "=")
		if isGoRootEnv(key) {
			t.Fatalf("resolved Go command inherited incompatible toolchain override %q", kv)
		}
	}
	if !environmentContains(env, "UNIT_HEALTH_PRESERVED_ENV", "present") {
		t.Fatal("unrelated inherited environment was not preserved")
	}
}

func TestScrubbedEnvironPreservesGoRootForNonGoCommand(t *testing.T) {
	want := filepath.Join(t.TempDir(), "language-runtime-root")
	t.Setenv("GOROOT", want)

	if !environmentContains(scrubbedEnviron("node"), "GOROOT", want) {
		t.Fatal("non-Go command lost an unrelated inherited environment variable")
	}
}

func TestIsGoExecutableCrossPlatformNames(t *testing.T) {
	for _, executable := range []string{"go", filepath.Join("managed", "bin", "go"), filepath.Join("managed", "bin", "go.exe"), filepath.Join("managed", "bin", "GO.EXE")} {
		if !isGoExecutable(executable) {
			t.Errorf("isGoExecutable(%q) = false, want true", executable)
		}
	}
	for _, executable := range []string{"cargo", "gofumpt", "go-test"} {
		if isGoExecutable(executable) {
			t.Errorf("isGoExecutable(%q) = true, want false", executable)
		}
	}
}

func environmentContains(env []string, wantKey, wantValue string) bool {
	for _, kv := range env {
		key, value, ok := strings.Cut(kv, "=")
		if ok && strings.EqualFold(key, wantKey) && value == wantValue {
			return true
		}
	}
	return false
}

func TestBoundedRunFailsClosedForUnsupportedHermeticPolicy(t *testing.T) {
	policies := []HermeticPolicy{{Network: "allow_declared"}, {DetectOpenHandles: true}, {OrderIndependent: true}}
	if !HostHermeticCapabilities().NetworkDeny {
		policies = append(policies, HermeticPolicy{Network: "deny"})
	}
	if !HostHermeticCapabilities().WorkspaceReadonly {
		policies = append(policies, HermeticPolicy{Filesystem: "workspace_readonly"})
	}
	if !HostHermeticCapabilities().ChildLeakDetection {
		policies = append(policies, HermeticPolicy{DetectChildLeaks: true})
	}
	for _, policy := range policies {
		res := Bounded{}.Run(context.Background(), Command{Executable: "echo", Args: []string{"ok"}, Hermetic: policy})
		if res.Status != StatusError || res.FailureClass != ClassUnsupported {
			t.Fatalf("policy=%+v status=%q class=%q reason=%q", policy, res.Status, res.FailureClass, res.FailureReason)
		}
	}
}

func TestBoundedRunWorkspaceReadonlyUsesNativeSandboxWhenAvailable(t *testing.T) {
	cmd := helperCommand("echo")
	cmd.Hermetic = HermeticPolicy{Filesystem: "workspace_readonly"}
	res := Bounded{}.Run(context.Background(), cmd)
	if HostHermeticCapabilities().WorkspaceReadonly {
		if res.Status != StatusPassed {
			t.Fatalf("workspace-readonly command status=%q class=%q reason=%q", res.Status, res.FailureClass, res.FailureReason)
		}
		return
	}
	if res.Status != StatusError || res.FailureClass != ClassUnsupported {
		t.Fatalf("unsupported workspace-readonly command status=%q class=%q reason=%q", res.Status, res.FailureClass, res.FailureReason)
	}
}

func TestBoundedRunChildLeakObservationCapability(t *testing.T) {
	cmd := helperCommand("echo")
	cmd.Hermetic = HermeticPolicy{DetectChildLeaks: true}
	res := Bounded{}.Run(context.Background(), cmd)
	if HostHermeticCapabilities().ChildLeakDetection {
		if res.Status != StatusPassed {
			t.Fatalf("child-leak observation status=%q class=%q reason=%q", res.Status, res.FailureClass, res.FailureReason)
		}
		return
	}
	if res.Status != StatusError || res.FailureClass != ClassUnsupported {
		t.Fatalf("unsupported child-leak observation status=%q class=%q reason=%q", res.Status, res.FailureClass, res.FailureReason)
	}
}

func TestBoundedRunNetworkDenyUsesNativeSandboxWhenAvailable(t *testing.T) {
	res := Bounded{}.Run(context.Background(), Command{Executable: "echo", Args: []string{"ok"}, Hermetic: HermeticPolicy{Network: "deny"}, TimeoutSeconds: 10})
	if HostHermeticCapabilities().NetworkDeny {
		if res.Status != StatusPassed {
			t.Fatalf("network-denied command status=%q class=%q reason=%q", res.Status, res.FailureClass, res.FailureReason)
		}
		return
	}
	if res.Status != StatusError || res.FailureClass != ClassUnsupported {
		t.Fatalf("unsupported network-denied command status=%q class=%q reason=%q", res.Status, res.FailureClass, res.FailureReason)
	}
}

func TestHostHermeticCapabilitiesDoNotOverclaimSandboxing(t *testing.T) {
	capabilities := HostHermeticCapabilities()
	if !capabilities.TemporaryRoot || !capabilities.RestoreEnvironment {
		t.Fatalf("portable guarantees = %+v", capabilities)
	}
	if capabilities.OpenHandleDetection {
		t.Fatalf("unsupported sandbox capabilities were overclaimed: %+v", capabilities)
	}
}

func TestBoundedRunUsesAndCleansTemporaryRoot(t *testing.T) {
	cmd := helperCommand("print-tmp")
	cmd.Hermetic, cmd.TimeoutSeconds = HermeticPolicy{Filesystem: "temporary_root"}, 10
	res := Bounded{}.Run(context.Background(), cmd)
	if res.Status != StatusPassed {
		t.Fatalf("status=%q class=%q reason=%q", res.Status, res.FailureClass, res.FailureReason)
	}
	root := strings.TrimSpace(res.Stdout)
	if root == "" {
		t.Fatal("temporary root was not projected into TMPDIR")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("temporary root still exists after command: %q (err=%v)", root, err)
	}
}

func TestBoundedRunSupportsNativeWorkingDirectoryWithSpaces(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "path with spaces", "naïve")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := helperCommand("echo")
	cmd.Dir, cmd.TimeoutSeconds = dir, 10
	res := Bounded{}.Run(context.Background(), cmd)
	if res.Status != StatusPassed || res.Stdout != "hello" {
		t.Fatalf("status=%q stdout=%q reason=%q", res.Status, res.Stdout, res.FailureReason)
	}
}

func TestRunAllPreservesOrderAndConcurrency(t *testing.T) {
	cmds := []Command{helperCommand("echo"), helperCommand("fail"), helperCommand("echo")}
	cmds[0].WorkspaceID, cmds[1].WorkspaceID, cmds[2].WorkspaceID = "a", "b", "c"
	cmds[0].TimeoutSeconds, cmds[1].TimeoutSeconds, cmds[2].TimeoutSeconds = 10, 10, 10
	results := RunAll(context.Background(), Bounded{}, cmds, 2)
	if len(results) != 3 {
		t.Fatalf("want 3 results, got %d", len(results))
	}
	if results[0].WorkspaceID != "a" || results[1].WorkspaceID != "b" || results[2].WorkspaceID != "c" {
		t.Errorf("order not preserved: %+v", results)
	}
	if results[1].Status != StatusFailed {
		t.Errorf("b should fail, got %q", results[1].Status)
	}
}

func TestBoundedRunScrubsScenarioIdentityEnv(t *testing.T) {
	t.Setenv("UI_PORT", "24851")
	t.Setenv("SCENARIO_NAME", "unit-health")
	t.Setenv("VROOLI_SCENARIO", "unit-health")
	t.Setenv("UNIT_HEALTH_TEST_KEEP", "kept")
	cmd := helperCommand("print-env")
	cmd.WorkspaceID, cmd.Name, cmd.TimeoutSeconds = "w", "env", 10
	res := Bounded{}.Run(context.Background(), cmd)
	if res.Status != StatusPassed {
		t.Fatalf("status = %q (%s), want passed", res.Status, res.FailureReason)
	}
	want := "UI_PORT=[] SCENARIO_NAME=[] VROOLI_SCENARIO=[] KEEP=[kept] CI=[1]"
	if !strings.Contains(res.Stdout, want) {
		t.Fatalf("stdout = %q, want it to contain %q", res.Stdout, want)
	}
}

func TestBoundedRunExplicitEnvOverridesScrub(t *testing.T) {
	t.Setenv("UI_PORT", "24851")
	cmd := helperCommand("print-ui-port")
	cmd.WorkspaceID, cmd.Name, cmd.TimeoutSeconds = "w", "env", 10
	cmd.Env["UI_PORT"] = "12345"
	res := Bounded{}.Run(context.Background(), cmd)
	if res.Status != StatusPassed {
		t.Fatalf("status = %q (%s), want passed", res.Status, res.FailureReason)
	}
	if !strings.Contains(res.Stdout, "UI_PORT=[12345]") {
		t.Fatalf("stdout = %q, want explicit UI_PORT=12345 preserved", res.Stdout)
	}
}

func TestBoundedRunUsesAndReapsGovernedGoWorkDir(t *testing.T) {
	runtimeHome := t.TempDir()
	t.Setenv("VROOLI_HOME", runtimeHome)
	cmd := helperCommand("print-gotmp")
	cmd.WorkspaceID, cmd.Name, cmd.TimeoutSeconds = "w", "gotmp", 10
	res := Bounded{}.Run(context.Background(), cmd)
	if res.Status != StatusPassed {
		t.Fatalf("status = %q (%s), want passed", res.Status, res.FailureReason)
	}
	if !strings.HasPrefix(strings.TrimSpace(res.Stdout), filepath.Join(runtimeHome, "tmp", "go-work")) {
		t.Fatalf("GOTMPDIR = %q, want governed runtime-home path", res.Stdout)
	}
	entries, err := os.ReadDir(filepath.Join(runtimeHome, "tmp", "go-work"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("reaped Go work directory still has entries: %v", entries)
	}
}

func TestScenarioIdentityEnvironmentMatchingIsPortable(t *testing.T) {
	if !isScenarioIdentityEnv("UI_PORT") || !isScenarioIdentityEnv("SCENARIO_NAME") {
		t.Fatal("canonical scenario identity variables were not recognized")
	}
	if runtime.GOOS == "windows" && !isScenarioIdentityEnv("ui_port") {
		t.Fatal("Windows environment matching must be case-insensitive")
	}
}

func helperCommand(action string) Command {
	return Command{
		Executable: os.Args[0],
		Args:       []string{"-test.run=^TestExecutorHelperProcess$", "--", action},
		Env:        map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
	}
}

// TestExecutorHelperProcess is a platform-neutral child process used by
// executor tests. Keeping the helper in Go avoids making Windows tests depend
// on sh/cmd/powershell and exercises the same typed executable+argv contract
// as production adapters.
func TestExecutorHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	action := os.Args[len(os.Args)-1]
	switch action {
	case "echo":
		fmt.Fprint(os.Stdout, "hello")
	case "fail":
		fmt.Fprint(os.Stderr, "boom")
		os.Exit(3)
	case "sleep":
		time.Sleep(30 * time.Second)
	case "spawn-grandchild":
		child := exec.Command(os.Args[0], "-test.run=^TestExecutorHelperProcess$", "--", "grandchild")
		child.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		if err := child.Start(); err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(4)
		}
		fmt.Fprint(os.Stdout, child.Process.Pid)
		if pidFile := os.Getenv("UNIT_HEALTH_GRANDCHILD_PID_FILE"); pidFile != "" {
			_ = os.WriteFile(pidFile, []byte(strconv.Itoa(child.Process.Pid)), 0o600)
		}
		_ = child.Wait()
	case "grandchild":
		time.Sleep(30 * time.Second)
	case "print-tmp":
		fmt.Fprint(os.Stdout, os.Getenv("TMPDIR"))
	case "print-gotmp":
		fmt.Fprint(os.Stdout, os.Getenv("GOTMPDIR"))
	case "print-env":
		fmt.Fprintf(os.Stdout, "UI_PORT=[%s] SCENARIO_NAME=[%s] VROOLI_SCENARIO=[%s] KEEP=[%s] CI=[%s]", os.Getenv("UI_PORT"), os.Getenv("SCENARIO_NAME"), os.Getenv("VROOLI_SCENARIO"), os.Getenv("UNIT_HEALTH_TEST_KEEP"), os.Getenv("CI"))
	case "print-ui-port":
		fmt.Fprintf(os.Stdout, "UI_PORT=[%s]", os.Getenv("UI_PORT"))
	}
	os.Exit(0)
}
