// Package executor runs planned test commands under hard bounds: a per-command
// timeout, a no-output watchdog, process-group cleanup, captured output
// excerpts, and a failure classification. It is deliberately ignorant of Unit
// Health's domain types so the validation package can fake it in tests.
package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/envkit-go"
	"github.com/vrooli/platform-go"
)

// Failure classes Unit Health distinguishes. The validation package maps these
// onto maturity finding codes.
const (
	ClassNone              = ""
	ClassTestFailure       = "test_failure"
	ClassMissingDependency = "missing_dependency"
	ClassMisconfiguration  = "misconfiguration"
	ClassTimeoutHang       = "timeout_hang"
	ClassNoOutputStall     = "no_output_stall"
	ClassSystem            = "system"
	ClassUnsupported       = "unsupported"
)

// Status values for a command result.
const (
	StatusPassed  = "passed"
	StatusFailed  = "failed"
	StatusTimeout = "timeout"
	StatusError   = "error"
)

const (
	defaultNoOutputTimeout = 2 * time.Minute
	maxExcerptBytes        = 8 << 10 // 8 KiB tail per stream
)

// Command is a single planned command to run.
type Command struct {
	WorkspaceID string
	Name        string
	// Executable is an absolute or PATH-resolved executable name. Args are
	// passed directly to os/exec; callers must never provide a shell command
	// string here.
	Executable      string
	Args            []string
	Dir             string
	TimeoutSeconds  int
	NoOutputTimeout time.Duration
	// Env is appended to the inherited environment with deterministic key
	// replacement semantics.
	Env       map[string]string
	Artifacts []Artifact
	Resources ResourceLimits
	Hermetic  HermeticPolicy
}

type Artifact struct {
	Label string
	Kind  string
	Path  string
}

type ResourceLimits struct {
	CPUWeight   int
	MemoryBytes int64
	MaxWorkers  int
}

type HermeticPolicy struct {
	Network            string
	Filesystem         string
	TemporaryRoot      bool
	RestoreEnvironment bool
	DetectChildLeaks   bool
	DetectOpenHandles  bool
	OrderIndependent   bool
}

// HermeticCapabilities is host evidence, not policy. A policy may request a
// capability that this executor cannot enforce; callers must fail closed and
// surface the typed reason rather than silently weakening the run.
type HermeticCapabilities struct {
	NetworkDeny         bool
	AllowDeclaredNet    bool
	WorkspaceReadonly   bool
	TemporaryRoot       bool
	RestoreEnvironment  bool
	ChildLeakDetection  bool
	OpenHandleDetection bool
	OrderIndependent    bool
}

// HostHermeticCapabilities describes the guarantees implemented by the
// portable executor. Network/filesystem sandboxing and observation adapters
// remain explicit extension points instead of being implied by process-group
// containment.
func HostHermeticCapabilities() HermeticCapabilities {
	return HermeticCapabilities{
		NetworkDeny:        networkDenyAvailable(),
		WorkspaceReadonly:  workspaceReadonlyAvailable(),
		TemporaryRoot:      true,
		RestoreEnvironment: true,
		ChildLeakDetection: childLeakDetectionAvailable(),
	}
}

// Result is the outcome of running one Command.
type Result struct {
	WorkspaceID   string
	Name          string
	Command       string
	Status        string
	ExitCode      int
	Stdout        string
	Stderr        string
	FailureClass  string
	FailureReason string
	DurationMS    int64
	CPUTimeMS     int64
	PeakRSSBytes  int64
}

// Runner executes a single Command.
type Runner interface {
	Run(ctx context.Context, cmd Command) Result
}

// Bounded is the default Runner. Zero value is usable.
type Bounded struct {
	// NoOutputTimeout cancels a command that produces no output for this long.
	// Defaults to defaultNoOutputTimeout when zero.
	NoOutputTimeout time.Duration
}

// RunAll executes commands with bounded concurrency, preserving input order in
// the returned results.
func RunAll(ctx context.Context, runner Runner, commands []Command, maxConcurrency int) []Result {
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	results := make([]Result, len(commands))
	if len(commands) == 0 {
		return results
	}
	if maxConcurrency > len(commands) {
		maxConcurrency = len(commands)
	}
	type job struct {
		index   int
		command Command
	}
	jobs := make(chan job)
	var wg sync.WaitGroup
	for worker := 0; worker < maxConcurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					results[job.index] = Result{WorkspaceID: job.command.WorkspaceID, Name: job.command.Name, Command: formatCommand(job.command.Executable, job.command.Args), Status: StatusError, FailureClass: ClassSystem, FailureReason: ctx.Err().Error()}
					continue
				}
				results[job.index] = runner.Run(ctx, job.command)
			}
		}()
	}
	for i, command := range commands {
		select {
		case jobs <- job{index: i, command: command}:
		case <-ctx.Done():
			results[i] = Result{WorkspaceID: command.WorkspaceID, Name: command.Name, Command: formatCommand(command.Executable, command.Args), Status: StatusError, FailureClass: ClassSystem, FailureReason: ctx.Err().Error()}
		}
	}
	close(jobs)
	wg.Wait()
	return results
}

// Run executes one command under the configured bounds.
func (b Bounded) Run(ctx context.Context, cmd Command) Result {
	res := Result{WorkspaceID: cmd.WorkspaceID, Name: cmd.Name, Command: formatCommand(cmd.Executable, cmd.Args)}
	if reason := unsupportedHermeticPolicy(cmd.Hermetic); reason != "" {
		res.Status = StatusError
		res.FailureClass = ClassUnsupported
		res.FailureReason = reason
		return res
	}
	if strings.TrimSpace(cmd.Executable) == "" {
		res.Status = StatusError
		res.FailureClass = ClassMisconfiguration
		res.FailureReason = "typed command has no executable"
		return res
	}
	path, err := exec.LookPath(cmd.Executable)
	if err != nil || path == "" {
		res.Status = StatusError
		res.FailureClass = ClassMissingDependency
		res.FailureReason = "required command not found: " + cmd.Executable
		return res
	}

	noOutput := b.NoOutputTimeout
	if cmd.NoOutputTimeout > 0 {
		noOutput = cmd.NoOutputTimeout
	}
	if noOutput <= 0 {
		noOutput = defaultNoOutputTimeout
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if cmd.TimeoutSeconds > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(cmd.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	// Watchdog context layered on top so the no-output stall can cancel
	// independently of the hard timeout.
	watchCtx, watchCancel := context.WithCancel(runCtx)
	defer watchCancel()

	var c *exec.Cmd
	var temporaryRoot string
	goWorkDir, err := createGoWorkDir("unit-health-")
	if err != nil {
		res.Status = StatusError
		res.FailureClass = ClassSystem
		res.FailureReason = "create Go work directory: " + err.Error()
		return res
	}
	defer os.RemoveAll(goWorkDir)
	if cmd.Hermetic.TemporaryRoot || cmd.Hermetic.Filesystem == "temporary_root" {
		temporaryRoot, err = os.MkdirTemp("", "unit-health-test-")
		if err != nil {
			res.Status = StatusError
			res.FailureClass = ClassSystem
			res.FailureReason = "create hermetic temporary root: " + err.Error()
			return res
		}
		defer os.RemoveAll(temporaryRoot)
	}
	if cmd.Hermetic.Filesystem == "workspace_readonly" && temporaryRoot == "" {
		temporaryRoot, err = os.MkdirTemp("", "unit-health-test-")
		if err != nil {
			res.Status = StatusError
			res.FailureClass = ClassSystem
			res.FailureReason = "create hermetic temporary root: " + err.Error()
			return res
		}
		defer os.RemoveAll(temporaryRoot)
	}
	executable, args, err := hermeticCommand(path, cmd.Args, cmd.Hermetic, cmd.Dir, temporaryRoot)
	if err != nil {
		res.Status = StatusError
		res.FailureClass = ClassUnsupported
		res.FailureReason = err.Error()
		return res
	}
	c = exec.CommandContext(watchCtx, executable, args...)
	if cmd.Dir != "" {
		c.Dir = cmd.Dir
	}
	c.Env = envkit.Toolchain(envkit.WithOverlay(envkit.Env(scrubbedEnviron(path)), envkit.SameScenario, envkit.Env{"GOWORK=off", "CI=1", "GOTMPDIR=" + goWorkDir}), envkit.ToolchainOptions{})
	keys := make([]string, 0, len(cmd.Env))
	canonicalEnv := make(map[string]string, len(cmd.Env))
	for key, value := range cmd.Env {
		canonicalKey := key
		if runtime.GOOS == "windows" {
			canonicalKey = strings.ToUpper(key)
		}
		canonicalEnv[canonicalKey] = value
	}
	for key := range canonicalEnv {
		keys = append(keys, key)
	}
	if temporaryRoot != "" {
		// These are the standard temporary/cache locations for the supported
		// runtimes. They keep mutable runner state out of the target workspace;
		// the parent process is never mutated.
		for _, key := range []string{"TMPDIR", "TMP", "TEMP", "GOTMPDIR", "XDG_CACHE_HOME"} {
			canonicalEnv[key] = temporaryRoot
		}
		keys = keys[:0]
		for key := range canonicalEnv {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := canonicalEnv[key]
		c.Env = append(c.Env, key+"="+value)
	}
	if err := platform.ConfigureCommand(c, platform.ProcessOptions{Detached: true}); err != nil {
		res.Status = StatusError
		res.FailureClass = ClassMisconfiguration
		res.FailureReason = "configure process containment: " + err.Error()
		return res
	}
	var containmentClose func()
	var containmentMu sync.RWMutex
	var containmentOnce sync.Once
	// On cancel (hard timeout or no-output stall) kill the whole process group
	// so children (e.g. pnpm -> node) die too. WaitDelay bounds how long Wait
	// blocks on output pipes still held open by orphaned grandchildren.
	c.Cancel = func() error {
		killGroup(c)
		containmentMu.RLock()
		closeContainment := containmentClose
		containmentMu.RUnlock()
		if closeContainment != nil {
			containmentOnce.Do(closeContainment)
		}
		return os.ErrProcessDone
	}
	c.WaitDelay = 3 * time.Second

	stdout := newTailWriter(maxExcerptBytes)
	stderr := newTailWriter(maxExcerptBytes)
	c.Stdout = stdout
	c.Stderr = stderr

	start := time.Now()
	resourceCollector := metrics.Start()
	if err := c.Start(); err != nil {
		resourceCollector.Stop()
		res.Status = StatusError
		res.FailureClass = ClassSystem
		res.FailureReason = "failed to start command: " + err.Error()
		res.DurationMS = time.Since(start).Milliseconds()
		return res
	}
	assignedContainment, assignErr := platform.AssignProcessContainment(c.Process)
	processGroupID, _ := platform.ProcessGroupID(c.Process.Pid)
	containmentMu.Lock()
	containmentClose = assignedContainment
	containmentMu.Unlock()
	err = assignErr
	if err != nil {
		_ = c.Process.Kill()
		_ = c.Wait()
		resourceCollector.Stop()
		res.Status = StatusError
		res.FailureClass = ClassSystem
		res.FailureReason = "attach process containment: " + err.Error()
		return res
	}
	defer func() {
		containmentMu.RLock()
		closeContainment := containmentClose
		containmentMu.RUnlock()
		if closeContainment != nil {
			containmentOnce.Do(closeContainment)
		}
	}()

	stalled := watchNoOutput(watchCtx, watchCancel, []*tailWriter{stdout, stderr}, noOutput)
	waitErr := c.Wait()
	resourceCollector.ObserveProcess(c.ProcessState)
	resourceMetrics := resourceCollector.Stop()
	// Ensure the whole process group is gone even on normal exit.
	killGroup(c)
	childLeak := cmd.Hermetic.DetectChildLeaks && !waitForProcessGroupExit(processGroupID)

	res.DurationMS = time.Since(start).Milliseconds()
	if c.ProcessState != nil {
		res.CPUTimeMS = (c.ProcessState.UserTime() + c.ProcessState.SystemTime()).Milliseconds()
	}
	if resourceMetrics != nil && resourceMetrics.GetResources() != nil {
		res.PeakRSSBytes = resourceMetrics.GetResources().GetPeakRssBytes()
	}
	res.Stdout = stdout.String()
	res.Stderr = stderr.String()
	res.ExitCode = c.ProcessState.ExitCode()

	switch {
	case stalled.Load():
		res.Status = StatusTimeout
		res.FailureClass = ClassNoOutputStall
		res.FailureReason = "no output for " + noOutput.String() + "; likely hung"
	case errors.Is(runCtx.Err(), context.DeadlineExceeded):
		res.Status = StatusTimeout
		res.FailureClass = ClassTimeoutHang
		res.FailureReason = "exceeded timeout"
	case childLeak:
		res.Status = StatusError
		res.FailureClass = ClassSystem
		res.FailureReason = "descendant process remained in the contained process group"
	case waitErr == nil:
		res.Status = StatusPassed
		res.FailureClass = ClassNone
	default:
		res.Status = StatusFailed
		res.FailureClass = classifyFailure(stdout.String(), stderr.String())
		res.FailureReason = waitErr.Error()
	}
	return res
}

func unsupportedHermeticPolicy(policy HermeticPolicy) string {
	capabilities := HostHermeticCapabilities()
	if policy.Network == "deny" && !capabilities.NetworkDeny {
		return "network-deny hermetic execution is unavailable on this host; use a host sandbox adapter"
	}
	if policy.Network == "allow_declared" && !capabilities.AllowDeclaredNet {
		return "allow-declared network hermetic execution is unavailable without declared egress rules; use a host sandbox adapter"
	}
	if policy.Filesystem == "workspace_readonly" && !capabilities.WorkspaceReadonly {
		return "workspace-readonly hermetic execution is unavailable on this host; use a host sandbox adapter"
	}
	if policy.DetectChildLeaks && !capabilities.ChildLeakDetection {
		return "child-leak detection is unavailable on this host; use a process-observation adapter"
	}
	if policy.DetectOpenHandles && !capabilities.OpenHandleDetection {
		return "open-handle detection is unavailable on this host; use a process-observation adapter"
	}
	if policy.OrderIndependent && !capabilities.OrderIndependent {
		return "order-independence execution is unavailable without a shuffle adapter"
	}
	return ""
}

var (
	sandboxProbe        sync.Once
	networkSandboxMode  string
	networkSandboxPath  string
	readonlySandboxPath string
)

func probeSandboxCapabilities() {
	sandboxProbe.Do(func() {
		truePath, trueErr := exec.LookPath("true")
		if trueErr != nil {
			return
		}
		switch runtime.GOOS {
		case "linux":
			if bwrap, err := exec.LookPath("bwrap"); err == nil {
				if exec.Command(bwrap, "--die-with-parent", "--unshare-net", "--", truePath).Run() == nil {
					networkSandboxMode, networkSandboxPath = "bwrap", bwrap
				}
				if exec.Command(bwrap, "--die-with-parent", "--ro-bind", "/", "/", "--dev", "/dev", "--proc", "/proc", "--", truePath).Run() == nil {
					readonlySandboxPath = bwrap
				}
			}
			if networkSandboxMode == "" {
				if unshare, err := exec.LookPath("unshare"); err == nil && exec.Command(unshare, "--net", "--", truePath).Run() == nil {
					networkSandboxMode, networkSandboxPath = "unshare", unshare
				}
			}
		case "darwin":
			if sandbox, err := exec.LookPath("sandbox-exec"); err == nil {
				probe := exec.Command(sandbox, "-p", "(version 1) (deny network*)", truePath)
				if probe.Run() == nil {
					networkSandboxMode, networkSandboxPath = "sandbox-exec", sandbox
				}
			}
		}
	})
}

func networkDenyAvailable() bool {
	probeSandboxCapabilities()
	return networkSandboxPath != ""
}

func workspaceReadonlyAvailable() bool {
	probeSandboxCapabilities()
	return readonlySandboxPath != ""
}

func hermeticCommand(path string, args []string, policy HermeticPolicy, dir, temporaryRoot string) (string, []string, error) {
	if policy.Network == "deny" && !networkDenyAvailable() {
		return "", nil, fmt.Errorf("network-deny hermetic execution is unavailable on this host; use a host sandbox adapter")
	}
	if policy.Filesystem == "workspace_readonly" {
		if !workspaceReadonlyAvailable() {
			return "", nil, fmt.Errorf("workspace-readonly hermetic execution is unavailable on this host; use a host sandbox adapter")
		}
		return readonlySandboxPath, bwrapReadonlyCommand(path, args, dir, temporaryRoot, policy.Network == "deny"), nil
	}
	if policy.Network != "deny" {
		return path, args, nil
	}
	if !networkDenyAvailable() {
		return "", nil, fmt.Errorf("network-deny hermetic execution is unavailable on this host; use a host sandbox adapter")
	}
	if networkSandboxMode == "bwrap" {
		wrapped := []string{"--die-with-parent", "--unshare-net", "--", path}
		return networkSandboxPath, append(wrapped, args...), nil
	}
	if networkSandboxMode == "unshare" {
		wrapped := []string{"--net", "--", path}
		return networkSandboxPath, append(wrapped, args...), nil
	}
	// sandbox-exec accepts a policy expression as an argv value; no shell is
	// involved. The profile is intentionally deny-only for undeclared network.
	return networkSandboxPath, append([]string{"-p", "(version 1) (deny network*)", path}, args...), nil
}

func bwrapReadonlyCommand(path string, args []string, dir, temporaryRoot string, denyNetwork bool) []string {
	wrapped := []string{"--die-with-parent", "--ro-bind", "/", "/", "--dev", "/dev", "--proc", "/proc"}
	if denyNetwork {
		wrapped = append(wrapped, "--unshare-net")
	}
	if temporaryRoot != "" {
		wrapped = append(wrapped, "--bind", temporaryRoot, temporaryRoot)
	}
	if dir != "" {
		wrapped = append(wrapped, "--chdir", dir)
	}
	wrapped = append(wrapped, "--", path)
	return append(wrapped, args...)
}

func formatCommand(executable string, args []string) string {
	parts := append([]string{executable}, args...)
	return strings.Join(parts, " ")
}

func killGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	groupID, err := platform.ProcessGroupID(cmd.Process.Pid)
	if err == nil && groupID > 0 {
		_ = platform.SignalProcessGroup(groupID, true)
		return
	}
	_ = platform.KillProcess(cmd.Process.Pid, true)
}

func createGoWorkDir(prefix string) (string, error) {
	base := strings.TrimSpace(os.Getenv("VROOLI_HOME"))
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			base = filepath.Join(home, ".vrooli")
		} else {
			base = os.TempDir()
		}
	}
	root := filepath.Join(base, "tmp", "go-work")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(root, prefix)
}

// scenarioIdentityEnvVars are the launch-time variables that bind a process to
// THIS scenario instance (Unit Health itself). Leaking them into a validated
// scenario's test commands makes those tests impersonate Unit Health — e.g. a
// UI e2e script reading UI_PORT would drive Unit Health's UI instead of its
// own scenario's. Tests must see the same clean environment a developer shell
// provides.
var scenarioIdentityEnvVars = map[string]struct{}{
	"UI_PORT":                  {},
	"API_PORT":                 {},
	"SCENARIO_NAME":            {},
	"SCENARIO_PATH":            {},
	"SCENARIO_DATA_DIR":        {},
	"SCENARIO_MODE":            {},
	"VROOLI_SCENARIO":          {},
	"VROOLI_SCENARIO_DIR":      {},
	"VROOLI_STORAGE_NAMESPACE": {},
	"VROOLI_PROCESS_ID":        {},
	"VROOLI_STEP":              {},
	"VROOLI_LIFECYCLE_MANAGED": {},
}

// scrubbedEnviron returns the inherited environment minus this scenario's
// identity variables. When Unit Health has already resolved the Go executable,
// that executable is the toolchain authority: an inherited GOROOT/GOTOOLDIR
// may point at a different installation and make the selected binary load an
// incompatible standard library. Omitting those overrides lets Go use the root
// embedded in (or discovered from) the resolved executable on every platform.
func scrubbedEnviron(executable string) []string {
	env := os.Environ()
	out := make([]string, 0, len(env))
	scrubGoRoot := isGoExecutable(executable)
	for _, kv := range env {
		key, _, ok := strings.Cut(kv, "=")
		if ok && (isScenarioIdentityEnv(key) || (scrubGoRoot && isGoRootEnv(key))) {
			continue
		}
		out = append(out, kv)
	}
	return out
}

func isGoExecutable(executable string) bool {
	name := strings.TrimSuffix(strings.ToLower(filepath.Base(executable)), ".exe")
	return name == "go"
}

func isGoRootEnv(key string) bool {
	return strings.EqualFold(key, "GOROOT") || strings.EqualFold(key, "GOTOOLDIR")
}

func isScenarioIdentityEnv(key string) bool {
	if runtime.GOOS == "windows" {
		key = strings.ToUpper(key)
	}
	_, drop := scenarioIdentityEnvVars[key]
	return drop
}

// classifyFailure inspects output to refine a nonzero exit into a more specific
// class than a bare test failure.
func classifyFailure(stdout, stderr string) string {
	combined := strings.ToLower(stdout + "\n" + stderr)
	switch {
	case strings.Contains(combined, "command not found"),
		strings.Contains(combined, "executable file not found"),
		strings.Contains(combined, "is not recognized"),
		// Node "Cannot find module 'x'" / pnpm lockfile errors mean the
		// dependency install is missing or stale, not a test misconfiguration.
		// The quote after "module" distinguishes node from Go's unquoted
		// "go: cannot find module providing package …".
		strings.Contains(combined, "err_module_not_found"),
		strings.Contains(combined, "cannot find module '"),
		strings.Contains(combined, "err_pnpm_no_lockfile"),
		strings.Contains(combined, "frozen-lockfile"),
		strings.Contains(combined, "missing dependencies in the lockfile"):
		return ClassMissingDependency
	case strings.Contains(combined, "no such tool"),
		strings.Contains(combined, "missing go.sum"),
		strings.Contains(combined, "cannot find module"),
		strings.Contains(combined, "no test files"),
		strings.Contains(combined, "cannot find package"):
		return ClassMisconfiguration
	default:
		return ClassTestFailure
	}
}
