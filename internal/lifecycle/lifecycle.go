package lifecycle

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqrun"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/ports"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/projectstate"
	"github.com/vrooli/vrooli/internal/resources"
	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	resourcemanifest "github.com/vrooli/vrooli/internal/resources/manifest"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
)

type Runner struct {
	Root        string
	Home        string
	Environment string
	Out         io.Writer
	Err         io.Writer
	Ports       *ports.Manager
	Logger      *slog.Logger
	Verbosity   Verbosity
	deps        lifecycleDeps
}

type lifecycleLogContext struct {
	Scenario  string
	Operation string
	Phase     string
}

// Verbosity controls how much of a lifecycle run is echoed to the console.
// The scenario-wide lifecycle log file always receives the full output
// regardless of this setting.
type Verbosity int

const (
	// VerbosityQuiet discards tool stdout and suppresses [INFO] step headers.
	// Errors and warnings still surface.
	VerbosityQuiet Verbosity = iota
	// VerbosityNormal is the default: [INFO] headers and final summary reach
	// the console; raw tool stdout (vite/pnpm) is kept in the log file only.
	VerbosityNormal
	// VerbosityVerbose tees all tool stdout through to the console.
	VerbosityVerbose
)

// WithVerbosity sets the console-verbosity mode and returns the runner for
// chaining. It is intended to be called once right after NewRunner.
func (r *Runner) WithVerbosity(v Verbosity) *Runner {
	r.Verbosity = v
	return r
}

// consoleOut returns the effective stdout writer for orchestrator-level
// output (infof/warnf step headers, slog text when routed here). It is
// suppressed only at VerbosityQuiet; at normal + verbose it reaches the
// console. Note: raw tool stdout (vite/pnpm) goes through childStdoutConsole
// instead so it can be suppressed at normal mode without hiding the
// step-header flow.
func (r *Runner) consoleOut() io.Writer {
	if r.Verbosity == VerbosityQuiet {
		return io.Discard
	}
	return r.Out
}

// consoleErr is the stderr counterpart to consoleOut. Error replay from
// failing steps is routed through r.Err directly (not consoleErr) so that
// failures are always visible regardless of mode.
func (r *Runner) consoleErr() io.Writer {
	if r.Verbosity == VerbosityQuiet {
		return io.Discard
	}
	return r.Err
}

// childStdoutConsole returns the effective console writer for raw
// child-process stdout produced by foreground lifecycle steps (e.g. vite
// build, pnpm install). Only VerbosityVerbose lets the flood reach the
// console; at normal and quiet it is discarded (the lifecycle log file
// still captures it via the MultiWriter wired in runWithLifecycleLog).
func (r *Runner) childStdoutConsole() io.Writer {
	if r.Verbosity == VerbosityVerbose {
		return r.Out
	}
	return io.Discard
}

// progressf emits a compact transition line to r.Out at VerbosityQuiet
// and VerbosityNormal. At VerbosityVerbose the raw slog debug stream and
// tool stdout already give a running picture, so duplicating pings here
// would add noise. The intent is to give users a visible heartbeat during
// long setups (vite rebuilds etc.) that would otherwise produce a silent
// 10+ second gap before the final summary; at Normal on a TTY the
// structured info-log stream is suppressed (see resolveQuiet in the top
// vrooli binary) so these pings become the primary in-flight signal.
// Written without color codes or carriage returns so the output stays
// CI- and log-capture-safe.
func (r *Runner) progressf(format string, args ...any) {
	if r.Verbosity == VerbosityVerbose || r.Out == nil {
		return
	}
	fmt.Fprintf(r.Out, format+"\n", args...)
}

type lifecycleDeps struct {
	sleep                   func(time.Duration)
	now                     func() time.Time
	signalProcessGroup      func(int, bool) error
	signalPID               func(int, bool) error
	listeningPIDs           func(int) ([]int, error)
	readScenarioRecords     func(string, string) ([]process.Record, error)
	isPIDRunning            func(int) bool
	resourceStatus          func(string, bool) (resourcecontrol.Status, error)
	resourceManifest        func(string) (resourcemanifest.ResourceManifest, error)
	runResource             func(string, []string, io.Writer, io.Writer) error
	runResourceCLI          func(string, []string, io.Writer, io.Writer) error
	inspectPort             func(int) (network.PortInspection, error)
	readProcessEnv          func(int) (map[string]string, error)
	enforceHostRequirements func(hostreqrun.Options) (vrooliruntime.Report, error)
}

type hostProbeDeps struct {
	stat        func(string) (os.FileInfo, error)
	readFile    func(string) ([]byte, error)
	lookPath    func(string) (string, error)
	getenv      func(string) string
	userHomeDir func() (string, error)
	walkDir     func(string, fs.WalkDirFunc) error
}

func defaultLifecycleDeps() lifecycleDeps {
	return lifecycleDeps{
		sleep:               time.Sleep,
		now:                 time.Now,
		signalProcessGroup:  signalProcessGroup,
		signalPID:           signalPID,
		listeningPIDs:       listeningPIDs,
		readScenarioRecords: process.ReadScenarioRecords,
		isPIDRunning:        process.IsPIDRunning,
	}
}

func defaultHostProbeDeps() hostProbeDeps {
	return hostProbeDeps{
		stat:        os.Stat,
		readFile:    os.ReadFile,
		lookPath:    exec.LookPath,
		getenv:      os.Getenv,
		userHomeDir: os.UserHomeDir,
		walkDir:     filepath.WalkDir,
	}
}

type StartOptions struct {
	CustomPath         string
	CleanStale         bool
	BestEffort         bool
	ForceSetup         bool
	ForceSetupScenario string
	Operation          string
}

type StopOptions struct {
	CustomPath string
}

type PhaseOptions struct {
	CustomPath              string
	Args                    []string
	AllowSkipMissingRuntime bool
	ManageRuntime           bool
	ProjectMode             bool
}

type Result struct {
	Scenario           scenario.Scenario
	AllocatedPorts     map[string]int
	Health             string
	FailedDependencies []string
	FailedResources    []string
	AlreadyRunning     bool
}

func NewRunner(root, home string, stdout, stderr io.Writer, logger ...*slog.Logger) (*Runner, error) {
	return newRunnerWithDeps(root, home, stdout, stderr, defaultLifecycleDeps(), logger...)
}

func newRunnerWithDeps(root, home string, stdout, stderr io.Writer, deps lifecycleDeps, logger ...*slog.Logger) (*Runner, error) {
	manager, err := ports.NewManager(root, home)
	if err != nil {
		return nil, err
	}
	baseLogger := slog.Default()
	if len(logger) > 0 && logger[0] != nil {
		baseLogger = logger[0]
	}
	return &Runner{
		Root:        filepath.Clean(root),
		Home:        filepath.Clean(home),
		Environment: hostreq.NormalizeEnvironment(""),
		Out:         stdout,
		Err:         stderr,
		Ports:       manager,
		Logger:      logx.WithSubsystem(baseLogger, "lifecycle"),
		deps:        deps,
	}, nil
}

func (r *Runner) environmentProfile() string {
	return hostreq.NormalizeEnvironment(r.Environment)
}

func (r *Runner) runtimeDeps() lifecycleDeps {
	deps := r.deps
	defaults := defaultLifecycleDeps()
	if deps.sleep == nil {
		deps.sleep = defaults.sleep
	}
	if deps.now == nil {
		deps.now = defaults.now
	}
	if deps.signalProcessGroup == nil {
		deps.signalProcessGroup = defaults.signalProcessGroup
	}
	if deps.signalPID == nil {
		deps.signalPID = defaults.signalPID
	}
	if deps.listeningPIDs == nil {
		deps.listeningPIDs = defaults.listeningPIDs
	}
	if deps.readScenarioRecords == nil {
		deps.readScenarioRecords = defaults.readScenarioRecords
	}
	if deps.isPIDRunning == nil {
		deps.isPIDRunning = defaults.isPIDRunning
	}
	if deps.resourceStatus == nil {
		deps.resourceStatus = func(name string, fast bool) (resourcecontrol.Status, error) {
			return resources.NewController(r.Root, r.Home).Status(name, fast)
		}
	}
	if deps.runResource == nil {
		deps.runResource = func(name string, args []string, stdout, stderr io.Writer) error {
			return resources.NewController(r.Root, r.Home).Run(name, args, stdout, stderr)
		}
	}
	if deps.resourceManifest == nil {
		deps.resourceManifest = func(name string) (resourcemanifest.ResourceManifest, error) {
			return resources.NewController(r.Root, r.Home).ResourceManifest(name)
		}
	}
	if deps.runResourceCLI == nil {
		deps.runResourceCLI = func(name string, args []string, stdout, stderr io.Writer) error {
			if err := cliinstall.NewManager(r.Root, r.Home).EnsureResourceCLI(name); err != nil {
				return fmt.Errorf("ensure resource CLI %s: %w", name, err)
			}
			return resources.NewController(r.Root, r.Home).RunResourceCLI(name, args, stdout, stderr)
		}
	}
	if deps.inspectPort == nil {
		deps.inspectPort = network.InspectPortListeners
	}
	if deps.readProcessEnv == nil {
		deps.readProcessEnv = process.ReadEnvironment
	}
	if deps.enforceHostRequirements == nil {
		deps.enforceHostRequirements = hostreqrun.Enforce
	}
	return deps
}

func (r *Runner) Start(name string, opts StartOptions) (Result, error) {
	r.progressf("starting %s...", name)
	r.logInfo("Scenario start requested",
		logx.AttrScenario, name,
		"best_effort", opts.BestEffort,
		"clean_stale", opts.CleanStale,
		"force_setup", opts.ForceSetup,
	)
	ready := make(map[string]struct{})
	result, err := r.startWithState(name, opts, ready, nil)
	if err != nil {
		r.logError("Scenario start failed", err, logx.AttrScenario, name)
		return Result{}, err
	}
	r.logInfo("Scenario start completed",
		logx.AttrScenario, name,
		logx.AttrStatus, result.Health,
		"already_running", result.AlreadyRunning,
		"failed_dependencies", len(result.FailedDependencies),
	)
	return result, nil
}

func (r *Runner) startWithState(name string, opts StartOptions, ready map[string]struct{}, stack []string) (Result, error) {
	item, err := r.loadScenario(name, opts.CustomPath)
	if err != nil {
		return Result{}, err
	}
	// Port policy is enforced here rather than inside scenario.ReadService so
	// that Stop, Status, List, and other observation-only paths can still
	// operate on manifests whose ports pre-date the canonical bands.
	if err := scenario.ValidateManifestPorts(item.ServicePath, item.Manifest.Ports); err != nil {
		return Result{}, err
	}
	return r.startScenario(item, opts, ready, stack)
}

func (r *Runner) startScenario(item scenario.Scenario, opts StartOptions, ready map[string]struct{}, stack []string) (result Result, err error) {
	deps := r.runtimeDeps()
	cleanupOnError := false
	defer func() {
		if err == nil || !cleanupOnError {
			return
		}
		// Rollback is intentionally scoped to the scenario currently being started.
		// Dependencies and resources that were started earlier in the recursive chain
		// are shared runtime infrastructure and may already be needed by other live
		// scenarios, so this rollback must not unwind them opportunistically.
		if cleanupErr := r.cleanupScenarioRuntime(item.Slug, opts.CustomPath, false); cleanupErr != nil {
			r.logError("Failed to roll back failed scenario start", cleanupErr, logx.AttrScenario, item.Slug)
			err = errors.Join(err, fmt.Errorf("rollback failed: %w", cleanupErr))
		}
	}()
	if opts.CleanStale && len(stack) == 0 {
		r.logDebug("Cleaning stale port locks before scenario start", logx.AttrScenario, item.Slug)
		if err := r.Ports.CleanStaleLocks(); err != nil {
			return Result{}, err
		}
	}

	failedDeps, failedResources, err := r.bootstrapScenarioDependencies(item, opts, ready, stack)
	if err != nil {
		return Result{}, err
	}

	records, err := deps.readScenarioRecords(r.Home, item.Slug)
	if err != nil {
		return Result{}, err
	}
	runtime := process.SummarizeScenario(item.Slug, records)
	forceSetup := opts.ForceSetup && (opts.ForceSetupScenario == "" || opts.ForceSetupScenario == item.Slug)
	if runtime.ProcessCount > 0 {
		strictHealthy := r.isScenarioHealthyStrict(item, runtime.Records)
		setupNeeded, _, setupErr := r.SetupNeeded(item, forceSetup)
		if setupErr != nil {
			return Result{}, setupErr
		}
		if strictHealthy && !setupNeeded {
			currentPorts := r.runtimePorts(item.Manifest, runtime.Records)
			health := scenario.EvaluateHealth(item.Manifest.HealthConfig(), currentPorts)
			r.progressf("%s is already running", item.Slug)
			r.logInfo("Scenario already running and healthy",
				logx.AttrScenario, item.Slug,
				logx.AttrStatus, health,
				logx.AttrProcesses, runtime.ProcessCount,
			)
			return Result{
				Scenario:           item,
				AllocatedPorts:     currentPorts,
				Health:             health,
				FailedDependencies: failedDeps,
				FailedResources:    failedResources,
				AlreadyRunning:     true,
			}, nil
		}
		if err := r.Stop(item.Slug, StopOptions{}); err != nil {
			return Result{}, err
		}
		deps.sleep(1 * time.Second)
	}

	if err := r.enforceScenarioHostRequirements(item); err != nil {
		return Result{}, err
	}

	env, err := r.prepareScenarioEnvironment(item, records)
	cleanupOnError = true
	if err != nil {
		return Result{}, err
	}

	setupNeeded, _, err := r.SetupNeeded(item, forceSetup)
	if err != nil {
		return Result{}, err
	}

	if setupNeeded {
		r.progressf("running setup phase for %s...", item.Slug)
		r.logInfo("Executing setup phase for scenario", logx.AttrScenario, item.Slug, logx.AttrPhase, "setup")
		if err := r.runWithLifecycleLog(startLifecycleLogContext(item.Slug, opts.Operation, "setup"), func(logWriter, childWriter io.Writer) error {
			_, err := r.ExecutePhaseDetailed(item, "setup", env.EnvVars, nil, logWriter, childWriter)
			return err
		}); err != nil {
			return Result{}, err
		}
	}

	r.progressf("running develop phase for %s...", item.Slug)
	r.logInfo("Executing develop phase for scenario", logx.AttrScenario, item.Slug, logx.AttrPhase, "develop")
	if err := r.runWithLifecycleLog(startLifecycleLogContext(item.Slug, opts.Operation, "develop"), func(logWriter, childWriter io.Writer) error {
		_, err := r.ExecutePhaseDetailed(item, "develop", env.EnvVars, nil, logWriter, childWriter)
		return err
	}); err != nil {
		return Result{}, err
	}

	r.progressf("waiting for %s to become healthy...", item.Slug)
	healthStatus, err := r.WaitForHealth(item, env.EnvVars)
	if err != nil {
		return Result{}, err
	}

	// Upgrade each fixed-port lock to record the real listener PID now that
	// the health check confirmed the binary bound. ensurePortClaimed wrote
	// the lock with the lifecycle runner's PID, so without this upgrade the
	// next ensurePortClaimed call would incorrectly trust the runner's PID
	// as proof the listener is alive.
	r.confirmFixedPortLocks(item)

	if len(failedDeps) > 0 {
		r.logWarn("Scenario started in degraded mode",
			logx.AttrScenario, item.Slug,
			logx.AttrStatus, healthStatus,
			"failed_dependencies", failedDeps,
		)
	}
	if len(failedResources) > 0 {
		r.logWarn("Scenario started with degraded resources",
			logx.AttrScenario, item.Slug,
			logx.AttrStatus, healthStatus,
			"failed_resources", failedResources,
		)
	}

	result = Result{
		Scenario:           item,
		AllocatedPorts:     env.AllocatedPorts,
		Health:             healthStatus,
		FailedDependencies: failedDeps,
		FailedResources:    failedResources,
	}
	cleanupOnError = false
	return result, nil
}

func (r *Runner) bootstrapScenarioDependencies(item scenario.Scenario, opts StartOptions, ready map[string]struct{}, stack []string) ([]string, []string, error) {
	failedDeps, err := r.ensureDependencies(item, opts, ready, append(stack, item.Slug))
	if err != nil {
		return nil, nil, err
	}
	failedResources, err := r.ensureResourceDependencies(item, opts)
	if err != nil {
		return nil, nil, err
	}
	return failedDeps, failedResources, nil
}

func (r *Runner) prepareScenarioEnvironment(item scenario.Scenario, records []process.Record) (ports.Environment, error) {
	if err := r.cleanupFixedPortOrphans(item, records); err != nil {
		return ports.Environment{}, err
	}

	env, err := r.Ports.BuildEnvironment(item, nil)
	if err != nil {
		return ports.Environment{}, err
	}

	if err := r.runWithLifecycleLog(lifecycleLogContext{Scenario: item.Slug, Operation: "environment", Phase: "database"}, func(logWriter, _ io.Writer) error {
		return r.ensureScenarioDatabase(item, env.EnvVars, logWriter)
	}); err != nil {
		return ports.Environment{}, err
	}

	return env, nil
}

func (r *Runner) Stop(name string, opts StopOptions) error {
	r.progressf("stopping %s...", name)
	r.logInfo("Scenario stop requested", logx.AttrScenario, name)
	if err := r.cleanupScenarioRuntime(name, opts.CustomPath, true); err != nil {
		r.logError("Failed to remove scenario locks", err, logx.AttrScenario, name)
		return err
	}
	r.logInfo("Scenario stop completed", logx.AttrScenario, name)
	return nil
}

func (r *Runner) cleanupScenarioRuntime(name, customPath string, includeManifestFixedPorts bool) error {
	deps := r.runtimeDeps()
	records, err := deps.readScenarioRecords(r.Home, name)
	if err != nil {
		return err
	}

	processDir := process.ScenarioProcessDir(r.Home, name)
	stepFiles, globErr := filepath.Glob(filepath.Join(processDir, "*.json"))
	if globErr != nil {
		return globErr
	}

	groups := make(map[int]struct{})
	for _, record := range process.LiveRecords(records) {
		pgid := record.PGID
		if pgid <= 0 {
			pgid = record.PID
		}
		if pgid > 0 {
			groups[pgid] = struct{}{}
		}
	}

	for pgid := range groups {
		_ = deps.signalProcessGroup(pgid, false)
	}
	if len(groups) > 0 {
		deps.sleep(2 * time.Second)
		for pgid := range groups {
			if deps.isPIDRunning(pgid) {
				_ = deps.signalProcessGroup(pgid, true)
			}
		}
		deps.sleep(500 * time.Millisecond)
	}

	for _, stepFile := range stepFiles {
		step := strings.TrimSuffix(filepath.Base(stepFile), filepath.Ext(stepFile))
		_ = process.RemoveScenarioRecord(r.Home, name, step)
	}

	portsToCheck := make(map[int]struct{})
	locks, err := r.Ports.LocksForScenario(name)
	if err != nil {
		return err
	}
	for _, lock := range locks {
		portsToCheck[lock.Port] = struct{}{}
	}

	if includeManifestFixedPorts {
		if item, loadErr := r.loadScenario(name, customPath); loadErr == nil {
			for _, portSummary := range item.Manifest.SortedPorts() {
				if portSummary.FixedPort != nil {
					portsToCheck[*portSummary.FixedPort] = struct{}{}
				}
			}
		}
	}

	if err := r.killOrphansOnPorts(portsToCheck); err != nil {
		return err
	}

	if err := r.verifyPortsReleased(name, portsToCheck); err != nil {
		return err
	}

	if err := r.Ports.RemoveScenarioLocks(name); err != nil {
		return err
	}
	return nil
}

func (r *Runner) Restart(name string, opts StartOptions) (Result, error) {
	deps := r.runtimeDeps()
	r.logInfo("Scenario restart requested", logx.AttrScenario, name)
	if err := r.Stop(name, StopOptions{}); err != nil {
		r.logError("Scenario restart failed during stop", err, logx.AttrScenario, name)
		return Result{}, err
	}
	deps.sleep(1 * time.Second)
	opts.ForceSetup = true
	opts.ForceSetupScenario = name
	opts.Operation = "restart"
	result, err := r.Start(name, opts)
	if err != nil {
		r.logError("Scenario restart failed during start", err, logx.AttrScenario, name)
		return Result{}, err
	}
	r.logInfo("Scenario restart completed", logx.AttrScenario, name, logx.AttrStatus, result.Health)
	return result, nil
}

// enforceScenarioHostRequirements resolves and installs host requirements
// declared directly on the scenario. Resource-level declarations are handled by
// enforceResourceHostRequirements before each resource dep starts, so scope is
// kept tight: only the root manifest plus the scenario's own declarations.
// A scenario with no declared hostTools/hostSafeguards yields a no-op.
func (r *Runner) enforceScenarioHostRequirements(item scenario.Scenario) error {
	deps := r.runtimeDeps()
	if deps.enforceHostRequirements == nil {
		return nil
	}
	if _, err := deps.enforceHostRequirements(hostreqrun.Options{
		Root:        r.Root,
		Home:        r.Home,
		Environment: r.environmentProfile(),
		When:        "develop",
		Resources:   "none",
		Scenarios:   item.Slug,
		AutoInstall: true,
		Stdout:      r.Out,
		Stderr:      r.Err,
		Label:       "scenario:" + item.Slug,
	}); err != nil {
		r.logError("Host requirements enforcement failed", err, logx.AttrScenario, item.Slug)
		return err
	}
	return nil
}

// enforceResourceHostRequirements installs the host requirements declared on a
// single resource manifest before the resource itself is started. Root-manifest
// tools are always included by the resolver, which is fine — handlers are
// idempotent.
func (r *Runner) enforceResourceHostRequirements(resourceName string) error {
	deps := r.runtimeDeps()
	if deps.enforceHostRequirements == nil {
		return nil
	}
	if _, err := deps.enforceHostRequirements(hostreqrun.Options{
		Root:        r.Root,
		Home:        r.Home,
		Environment: r.environmentProfile(),
		When:        "develop",
		Resources:   resourceName,
		Scenarios:   "none",
		AutoInstall: true,
		Stdout:      r.Out,
		Stderr:      r.Err,
		Label:       "resource:" + resourceName,
	}); err != nil {
		r.logError("Host requirements enforcement failed", err, logx.AttrDependency, resourceName)
		return err
	}
	return nil
}

func (r *Runner) logger() *slog.Logger {
	if r == nil || r.Logger == nil {
		return logx.WithSubsystem(slog.Default(), "lifecycle")
	}
	return r.Logger
}

func (r *Runner) logDebug(msg string, args ...any) {
	r.logger().Debug(msg, args...)
}

func (r *Runner) logInfo(msg string, args ...any) {
	r.logger().Info(msg, args...)
}

func (r *Runner) logWarn(msg string, args ...any) {
	r.logger().Warn(msg, args...)
}

func (r *Runner) logError(msg string, err error, args ...any) {
	logx.Error(r.logger(), msg, err, args...)
}

// confirmFixedPortLocks upgrades every fixed-port lock to point at the real
// listener PID observed via inspectPort. Errors are logged and swallowed;
// lock-confirmation is advisory and must never fail a healthy start.
func (r *Runner) confirmFixedPortLocks(item scenario.Scenario) {
	deps := r.runtimeDeps()
	for _, portSummary := range item.Manifest.SortedPorts() {
		if portSummary.FixedPort == nil {
			continue
		}
		port := *portSummary.FixedPort
		inspection, err := deps.inspectPort(port)
		if err != nil || !inspection.Inspection.Available {
			continue
		}
		if len(inspection.Listeners) == 0 {
			continue
		}
		realPID := inspection.Listeners[0].PID
		if realPID <= 0 {
			continue
		}
		if err := r.Ports.ConfirmLock(port, item.Slug, realPID); err != nil {
			r.logWarn("ConfirmLock failed",
				logx.AttrScenario, item.Slug,
				"port", port,
				"pid", realPID,
				"err", err.Error())
		}
	}
}

// verifyPortsReleased polls each port for up to ~2 s after the kill loop and
// returns a loud error if any are still held. Surfacing this at stop time
// lets Restart fail fast with a diagnostic rather than silently racing into
// a Start that will itself fail with the generic "port already in use".
func (r *Runner) verifyPortsReleased(scenarioName string, portsToCheck map[int]struct{}) error {
	if len(portsToCheck) == 0 {
		return nil
	}
	deps := r.runtimeDeps()
	const (
		maxAttempts = 20
		interval    = 100 * time.Millisecond
	)
	stillBound := make(map[int][]int)
	for port := range portsToCheck {
		var pids []int
		for attempt := 0; attempt < maxAttempts; attempt++ {
			got, err := deps.listeningPIDs(port)
			if err != nil {
				// listeningPIDs swallows exec errors to nil; any real error
				// means we cannot verify, so surface it.
				return fmt.Errorf("verify port %d released: %w", port, err)
			}
			pids = got
			if len(pids) == 0 {
				break
			}
			deps.sleep(interval)
		}
		if len(pids) > 0 {
			stillBound[port] = pids
		}
	}
	if len(stillBound) == 0 {
		return nil
	}
	// Emit one error-level line per stuck port and a single aggregated
	// error back up through Stop. We deliberately continue to the lock
	// cleanup on the caller's side so a partial stop still tears down
	// everything it can.
	for port, pids := range stillBound {
		r.logError("Port still bound after stop", nil,
			logx.AttrScenario, scenarioName,
			"port", port,
			"pids", pids)
	}
	return fmt.Errorf("stop %s: port(s) still bound after kill: %v", scenarioName, stillBound)
}

func (r *Runner) killOrphansOnPorts(portsToCheck map[int]struct{}) error {
	deps := r.runtimeDeps()
	for port := range portsToCheck {
		pids, err := deps.listeningPIDs(port)
		if err != nil {
			return err
		}
		for _, pid := range pids {
			_ = deps.signalPID(pid, false)
		}
	}

	deps.sleep(500 * time.Millisecond)

	for port := range portsToCheck {
		pids, err := deps.listeningPIDs(port)
		if err != nil {
			return err
		}
		for _, pid := range pids {
			_ = deps.signalPID(pid, true)
		}
	}
	return nil
}

func (r *Runner) cleanupFixedPortOrphans(item scenario.Scenario, records []process.Record) error {
	portsToCheck := make(map[int]struct{})
	for _, portSummary := range item.Manifest.SortedPorts() {
		if portSummary.FixedPort == nil {
			continue
		}
		if runtimeOwnerPID(records, *portSummary.FixedPort) > 0 {
			continue
		}
		portsToCheck[*portSummary.FixedPort] = struct{}{}
	}
	if len(portsToCheck) == 0 {
		return nil
	}
	return r.killManagedScenarioListeners(portsToCheck, item.Slug)
}

// envPortOrphanStrict disables the aggressive start-time fallback. When set
// to "true" the start path only kills listeners whose env vars positively
// identify them as this scenario's children; any other listener causes the
// usual "port already in use" error to surface so the operator can diagnose
// it. Leave unset in production — orphan children that lost env inheritance
// (node grandchildren under vite, for example) are the common real cause.
const envPortOrphanStrict = "VROOLI_PORT_ORPHAN_STRICT"

func (r *Runner) killManagedScenarioListeners(portsToCheck map[int]struct{}, scenarioName string) error {
	deps := r.runtimeDeps()
	targets := make(map[int]struct{})
	fallbackPorts := make(map[int][]int) // port -> pids seen without env match
	for port := range portsToCheck {
		inspection, err := deps.inspectPort(port)
		if err != nil {
			return err
		}
		if !inspection.Inspection.Available {
			continue
		}
		for _, listener := range inspection.Listeners {
			env, err := deps.readProcessEnv(listener.PID)
			if err != nil {
				fallbackPorts[port] = append(fallbackPorts[port], listener.PID)
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(env["VROOLI_LIFECYCLE_MANAGED"]), "true") {
				fallbackPorts[port] = append(fallbackPorts[port], listener.PID)
				continue
			}
			if strings.TrimSpace(env["VROOLI_SCENARIO"]) != scenarioName {
				// Different scenario owns the port — leave it. Orphan
				// fallback must not reach across scenario boundaries.
				continue
			}
			targets[listener.PID] = struct{}{}
		}
	}

	// Aggressive-fallback path: any listeners that could not be env-matched
	// are killed unless strict mode is enabled. Fixed ports are all in the
	// canonical band below 32768, so by policy no unrelated long-lived
	// process should hold them.
	if strings.EqualFold(strings.TrimSpace(os.Getenv(envPortOrphanStrict)), "true") {
		fallbackPorts = nil
	}
	for port, pids := range fallbackPorts {
		r.logWarn("Aggressive orphan kill fallback on fixed port",
			"port", port,
			"scenario", scenarioName,
			"pids", pids,
			"reason", "listener present but no VROOLI_SCENARIO env match")
		for _, pid := range pids {
			targets[pid] = struct{}{}
		}
	}

	if len(targets) == 0 {
		return nil
	}
	for pid := range targets {
		_ = deps.signalPID(pid, false)
	}
	deps.sleep(500 * time.Millisecond)
	for pid := range targets {
		if deps.isPIDRunning(pid) {
			_ = deps.signalPID(pid, true)
		}
	}
	return nil
}

func runtimeOwnerPID(records []process.Record, port int) int {
	for _, record := range process.LiveRecords(records) {
		if record.Port == port && record.PID > 0 {
			return record.PID
		}
	}
	return 0
}

func listeningPIDs(port int) ([]int, error) {
	path, err := exec.LookPath("lsof")
	if err != nil {
		return nil, nil
	}
	cmd := exec.Command(path, "-tiTCP:"+strconv.Itoa(port), "-sTCP:LISTEN")
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, nil
		}
		return nil, err
	}

	pids := []int{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		pid, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids, scanner.Err()
}

func (r *Runner) loadScenario(name, customPath string) (scenario.Scenario, error) {
	if strings.TrimSpace(customPath) == "" {
		item, err := scenario.Load(r.Root, name, scenario.SandboxEnvFromEnv())
		if err != nil {
			if errors.Is(err, scenario.ErrNotFound) {
				return scenario.Scenario{}, fmt.Errorf("scenario %q not found", name)
			}
			return scenario.Scenario{}, err
		}
		return item, nil
	}

	resolved := customPath
	if !filepath.IsAbs(resolved) {
		abs, err := filepath.Abs(resolved)
		if err != nil {
			return scenario.Scenario{}, err
		}
		resolved = abs
	}
	servicePath := filepath.Join(resolved, ".vrooli", "service.json")
	manifest, err := scenario.ReadService(servicePath)
	if err != nil {
		return scenario.Scenario{}, err
	}
	slug := name
	if slug == "" {
		slug = manifest.Service.Name
	}
	if slug == "" {
		slug = filepath.Base(resolved)
	}
	return scenario.Scenario{
		Slug:        slug,
		Path:        resolved,
		ServicePath: servicePath,
		Manifest:    manifest,
	}, nil
}

func stepConditionsMet(item scenario.Scenario, condition *scenario.Condition, env map[string]string) (bool, string, error) {
	return stepConditionsMetWithDeps(item, condition, env, defaultHostProbeDeps())
}

func stepConditionsMetWithDeps(item scenario.Scenario, condition *scenario.Condition, env map[string]string, deps hostProbeDeps) (bool, string, error) {
	if condition == nil {
		return true, "", nil
	}

	checkPath := func(target string) string {
		if strings.HasPrefix(target, "~") {
			if home, err := deps.userHomeDir(); err == nil {
				target = filepath.Join(home, strings.TrimPrefix(target, "~"))
			}
		}
		if filepath.IsAbs(target) {
			return filepath.Clean(target)
		}
		return filepath.Join(item.Path, filepath.FromSlash(target))
	}

	if condition.FileExists != "" {
		if _, err := deps.stat(checkPath(condition.FileExists)); err != nil {
			return false, fmt.Sprintf("required file %q is missing", condition.FileExists), nil
		}
	}
	if fileNotExists := condition.FileNotExists; fileNotExists != "" {
		if _, err := deps.stat(checkPath(fileNotExists)); err == nil {
			return false, fmt.Sprintf("file %q must not exist", fileNotExists), nil
		}
	}
	if condition.DirectoryExists != "" {
		info, err := deps.stat(checkPath(condition.DirectoryExists))
		if err != nil || !info.IsDir() {
			return false, fmt.Sprintf("required directory %q is missing", condition.DirectoryExists), nil
		}
	}
	if condition.ResourceEnabled != "" {
		dep, ok := item.Manifest.Dependencies.Resources[condition.ResourceEnabled]
		if !ok || !dep.Enabled {
			return false, fmt.Sprintf("resource %q is disabled", condition.ResourceEnabled), nil
		}
	}
	if jsonSpec := condition.JSONPathExists; jsonSpec != "" {
		ok, err := jsonPathExistsWithDeps(checkPath(strings.SplitN(jsonSpec, ":", 2)[0]), jsonSpec, deps)
		if err != nil {
			return false, "", err
		}
		if !ok {
			return false, fmt.Sprintf("JSON path %q was not found", jsonSpec), nil
		}
	}
	if command := condition.CommandExists; command != "" {
		if _, err := deps.lookPath(command); err != nil {
			return false, fmt.Sprintf("command %q is unavailable", command), nil
		}
	}
	if binary := condition.BinaryExists; binary != "" {
		if _, err := deps.lookPath(binary); err != nil {
			return false, fmt.Sprintf("command %q is unavailable", binary), nil
		}
	}
	if key := condition.EnvVarSet; key != "" {
		if strings.TrimSpace(env[key]) == "" && strings.TrimSpace(deps.getenv(key)) == "" {
			return false, fmt.Sprintf("environment variable %q is not set", key), nil
		}
	}
	if always := condition.Always; always != "" {
		lower := strings.ToLower(strings.TrimSpace(always))
		if lower == "false" || lower == "0" {
			return false, "step disabled by always=false", nil
		}
	}

	return true, "", nil
}

func jsonPathExistsWithDeps(filePath, spec string, deps hostProbeDeps) (bool, error) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return false, nil
	}

	data, err := deps.readFile(filePath)
	if err != nil {
		return false, nil
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return false, err
	}

	current := value
	for _, segment := range strings.Split(parts[1], ".") {
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[segment]
			if !ok {
				return false, nil
			}
			current = next
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(typed) {
				return false, nil
			}
			current = typed[index]
		default:
			return false, nil
		}
	}
	return current != nil, nil
}

func binariesNeedSetup(appRoot string, check scenario.ConditionCheck) (bool, string, error) {
	for _, target := range check.Targets {
		path := resolveCheckPath(appRoot, target)
		info, err := os.Stat(path)
		if err != nil || info.Mode()&0o111 == 0 {
			return true, "Missing binaries: " + strings.Join(check.Targets, ","), nil
		}

		binaryDir := filepath.Dir(path)
		if anyFileNewer(binaryDir, path, func(path string, d fs.DirEntry) bool {
			return strings.HasSuffix(path, ".go")
		}) {
			return true, "Missing binaries: " + strings.Join(check.Targets, ","), nil
		}
		for _, depFile := range []string{"go.mod", "go.sum"} {
			depPath := filepath.Join(binaryDir, depFile)
			if info, err := os.Stat(depPath); err == nil && info.ModTime().After(getModTime(path)) {
				return true, "Missing binaries: " + strings.Join(check.Targets, ","), nil
			}
		}

		replacePaths, err := localReplacePaths(filepath.Join(binaryDir, "go.mod"))
		if err != nil {
			return false, "", err
		}
		for _, replacePath := range replacePaths {
			resolved := filepath.Join(binaryDir, replacePath)
			if anyFileNewer(resolved, path, func(path string, d fs.DirEntry) bool {
				return strings.HasSuffix(path, ".go") || filepath.Base(path) == "go.mod"
			}) {
				return true, "Missing binaries: " + strings.Join(check.Targets, ","), nil
			}
		}
	}
	return false, "", nil
}

func uiBundleNeedsSetup(appRoot string, check scenario.ConditionCheck) (bool, string, error) {
	return uiBundleNeedsSetupWithDeps(appRoot, check, defaultHostProbeDeps())
}

func uiBundleNeedsSetupWithDeps(appRoot string, check scenario.ConditionCheck, deps hostProbeDeps) (bool, string, error) {
	bundlePath := resolveCheckPath(appRoot, defaultIfEmpty(check.BundlePath, "ui/dist/index.html"))
	sourceDir := resolveCheckPath(appRoot, defaultIfEmpty(check.SourceDir, "ui/src"))
	if _, err := deps.stat(bundlePath); err != nil {
		return true, "UI bundle outdated: " + defaultIfEmpty(check.BundlePath, "ui/dist/index.html"), nil
	}

	if anyFileNewerWithDeps(sourceDir, bundlePath, deps, func(path string, d fs.DirEntry) bool { return !d.IsDir() }) {
		return true, "UI bundle outdated: " + defaultIfEmpty(check.BundlePath, "ui/dist/index.html"), nil
	}

	uiDir := filepath.Dir(filepath.Dir(bundlePath))
	for _, file := range []string{"package.json", "vite.config.ts", "vite.config.js", "tsconfig.json", "index.html"} {
		configPath := filepath.Join(uiDir, file)
		if info, err := deps.stat(configPath); err == nil && info.ModTime().After(getModTimeWithDeps(bundlePath, deps)) {
			return true, "UI bundle outdated: " + defaultIfEmpty(check.BundlePath, "ui/dist/index.html"), nil
		}
	}

	watchDeps := true
	if check.WatchFileDependencies != nil {
		watchDeps = *check.WatchFileDependencies
	}
	if watchDeps {
		packageJSON := filepath.Join(uiDir, "package.json")
		specs, err := fileDependencySpecsWithDeps(packageJSON, deps)
		if err != nil {
			return false, "", err
		}
		excluded := make(map[string]struct{}, len(check.DependencyExcludes))
		for _, path := range check.DependencyExcludes {
			excluded[resolveCheckPath(uiDir, path)] = struct{}{}
		}
		for _, spec := range specs {
			resolved := resolveCheckPath(uiDir, strings.TrimPrefix(spec, "file:"))
			if _, skip := excluded[resolved]; skip {
				continue
			}
			if _, err := deps.stat(resolved); err != nil {
				return true, "UI bundle outdated: " + defaultIfEmpty(check.BundlePath, "ui/dist/index.html"), nil
			}
			if anyFileNewerWithDeps(resolved, bundlePath, deps, func(path string, d fs.DirEntry) bool {
				return !d.IsDir() && !strings.Contains(path, string(filepath.Separator)+"node_modules"+string(filepath.Separator)) && !strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator))
			}) {
				return true, "UI bundle outdated: " + defaultIfEmpty(check.BundlePath, "ui/dist/index.html"), nil
			}
		}
	}

	return false, "", nil
}

func anyScenarioInputNewerWithDeps(appRoot string, inputs []string, targetPath string, deps hostProbeDeps) (bool, error) {
	for _, input := range inputs {
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if hasGlobPattern(input) {
			matches, err := filepath.Glob(filepath.Join(appRoot, filepath.FromSlash(input)))
			if err != nil {
				return false, err
			}
			for _, match := range matches {
				newer, err := pathNewerThanTargetWithDeps(match, appRoot, targetPath, deps)
				if err != nil {
					return false, err
				}
				if newer {
					return true, nil
				}
			}
			continue
		}
		newer, err := pathNewerThanTargetWithDeps(resolveCheckPath(appRoot, input), appRoot, targetPath, deps)
		if err != nil {
			return false, err
		}
		if newer {
			return true, nil
		}
	}
	return false, nil
}

func pathNewerThanTargetWithDeps(path, appRoot, targetPath string, deps hostProbeDeps) (bool, error) {
	info, err := deps.stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return info.ModTime().After(getModTimeWithDeps(targetPath, deps)), nil
	}
	return anyFileNewerWithDeps(path, targetPath, deps, func(candidate string, d fs.DirEntry) bool {
		rel, err := filepath.Rel(appRoot, candidate)
		if err != nil {
			return false
		}
		rel = filepath.ToSlash(rel)
		return !shouldSkipLifecyclePath(rel)
	}), nil
}

func shouldSkipLifecyclePath(path string) bool {
	path = filepath.ToSlash(path)
	for _, skip := range []string{".git", ".idea", ".vscode", "node_modules", "dist", "build", "coverage", "tmp", "data"} {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	return false
}

func hasGlobPattern(value string) bool {
	return strings.ContainsAny(value, "*?[")
}

func resourcesNeedSetup(appRoot string, check scenario.ConditionCheck) bool {
	if check.Populated {
		return !projectstate.HasResourcesPopulated(appRoot)
	}
	if len(check.Resources) == 0 {
		return !projectstate.HasResourcesPopulated(appRoot)
	}
	for _, resourceName := range check.Resources {
		if !projectstate.HasResourcePopulated(appRoot, resourceName) {
			return true
		}
	}
	return false
}

func dependenciesNeedSetup(appRoot string, check scenario.ConditionCheck) bool {
	for _, path := range check.Paths {
		resolved := resolveCheckPath(appRoot, path)
		switch {
		case strings.HasSuffix(resolved, "package.json"):
			if _, err := os.Stat(filepath.Join(filepath.Dir(resolved), "node_modules")); err != nil {
				return true
			}
		case strings.HasSuffix(resolved, "go.mod"):
			if _, err := os.Stat(filepath.Join(filepath.Dir(resolved), "go.sum")); err != nil {
				if _, err := os.Stat(filepath.Join(filepath.Dir(resolved), "vendor")); err != nil {
					return true
				}
			}
		case strings.HasSuffix(resolved, "requirements.txt"):
			if _, err := os.Stat(filepath.Join(filepath.Dir(resolved), "venv")); err != nil {
				if _, err := os.Stat(filepath.Join(filepath.Dir(resolved), ".venv")); err != nil {
					return true
				}
			}
		case strings.HasSuffix(resolved, "Cargo.toml"):
			if _, err := os.Stat(filepath.Join(filepath.Dir(resolved), "target")); err != nil {
				return true
			}
		default:
			if _, err := os.Stat(resolved); err != nil {
				return true
			}
		}
	}
	return false
}

func dataNeedsSetup(appRoot string, check scenario.ConditionCheck) bool {
	target := resolveCheckPath(appRoot, defaultIfEmpty(check.Path, "data"))
	entries, err := os.ReadDir(target)
	return err != nil || len(entries) == 0
}

func filesNeedSetup(appRoot string, check scenario.ConditionCheck) bool {
	for _, path := range check.Paths {
		resolved := resolveCheckPath(appRoot, path)
		if _, err := os.Stat(resolved); err != nil {
			return true
		}
	}
	return false
}

func directoriesNeedSetup(appRoot string, check scenario.ConditionCheck) bool {
	for _, path := range check.Targets {
		resolved := resolveCheckPath(appRoot, path)
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			return true
		}
	}
	return false
}

func localReplacePaths(goModPath string) ([]string, error) {
	return localReplacePathsWithDeps(goModPath, defaultHostProbeDeps())
}

func localReplacePathsWithDeps(goModPath string, deps hostProbeDeps) ([]string, error) {
	data, err := deps.readFile(goModPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	paths := []string{}
	inBlock := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch line {
		case "", ")":
			if line == ")" {
				inBlock = false
			}
			continue
		case "replace (":
			inBlock = true
			continue
		}

		candidate := line
		if strings.HasPrefix(line, "replace ") {
			candidate = strings.TrimSpace(strings.TrimPrefix(line, "replace "))
		} else if !inBlock {
			continue
		}
		if !strings.Contains(candidate, "=>") {
			continue
		}

		fields := strings.Fields(candidate)
		if len(fields) == 0 {
			continue
		}
		path := fields[len(fields)-1]
		if strings.HasPrefix(path, "../") {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

func anyFileNewer(root, target string, include func(path string, d fs.DirEntry) bool) bool {
	return anyFileNewerWithDeps(root, target, defaultHostProbeDeps(), include)
}

func anyFileNewerWithDeps(root, target string, deps hostProbeDeps, include func(path string, d fs.DirEntry) bool) bool {
	if _, err := deps.stat(root); err != nil {
		return false
	}
	targetInfo, err := deps.stat(target)
	if err != nil {
		return false
	}
	targetTime := targetInfo.ModTime()

	walkErr := deps.walkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if include != nil && !include(path, d) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(targetTime) {
			return errStopWalk
		}
		return nil
	})

	return errors.Is(walkErr, errStopWalk)
}

var errStopWalk = errors.New("stop walk")

func getModTime(path string) time.Time {
	return getModTimeWithDeps(path, defaultHostProbeDeps())
}

func getModTimeWithDeps(path string, deps hostProbeDeps) time.Time {
	info, err := deps.stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func fileDependencySpecs(packageJSON string) ([]string, error) {
	return fileDependencySpecsWithDeps(packageJSON, defaultHostProbeDeps())
}

func fileDependencySpecsWithDeps(packageJSON string, deps hostProbeDeps) ([]string, error) {
	data, err := deps.readFile(packageJSON)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		PeerDependencies     map[string]string `json:"peerDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	specs := []string{}
	for _, deps := range []map[string]string{
		doc.Dependencies,
		doc.DevDependencies,
		doc.PeerDependencies,
		doc.OptionalDependencies,
	} {
		for _, value := range deps {
			if strings.HasPrefix(value, "file:") {
				specs = append(specs, value)
			}
		}
	}
	sort.Strings(specs)
	return specs, nil
}

func resolveCheckPath(base, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(base, filepath.FromSlash(path))
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func mergeEnv(base []string, overrides map[string]string) []string {
	merged := append([]string(nil), base...)
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		merged = setEnvValue(merged, key, overrides[key])
	}
	return merged
}

func setEnvValue(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			updated := append([]string(nil), env...)
			updated[i] = prefix + value
			return updated
		}
	}
	return append(env, prefix+value)
}

func inferStepPort(manifest scenario.ServiceManifest, step string, env map[string]string) int {
	key := scenario.InferPortEnvVar(manifest, step)
	if key == "" {
		return 0
	}
	port, _ := strconv.Atoi(env[key])
	return port
}

func healthPortsFromEnv(manifest scenario.ServiceManifest, env map[string]string) map[string]int {
	ports := make(map[string]int)
	for _, key := range manifest.PortEnvVars() {
		if port, err := strconv.Atoi(strings.TrimSpace(env[key])); err == nil && port > 0 {
			ports[key] = port
		}
	}
	for key, value := range env {
		if _, exists := ports[key]; exists || !strings.HasSuffix(key, "_PORT") {
			continue
		}
		if port, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && port > 0 {
			ports[key] = port
		}
	}
	return ports
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
