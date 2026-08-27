package lifecycle

import (
	"bufio"
	"context"
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
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqrun"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/ports"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/resources"
	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	resourcemanifest "github.com/vrooli/vrooli/internal/resources/manifest"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioenv"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const (
	lifecycleParameterA = 2
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
	// Engagements resolves Baseline Modes engagement state so the lifecycle can
	// route a serving instance's run/build CWD to the frozen restore-point copy
	// while an engagement is open (see effectiveSourceDir). Nil means "no
	// engagement awareness" — every instance runs from its working tree, the
	// pre-Baseline-Modes behavior. Injected at construction by the CLI edge so
	// the lifecycle never imports internal/baselinefloor directly.
	Engagements EngagementResolver
	deps        lifecycleDeps
	// sinks are the progress-event consumers. Nil means "just the built-in
	// text renderer" (the Runner itself implements ProgressSink); populated
	// via WithProgressSink.
	sinks         []ProgressSink
	sinksMu       sync.RWMutex
	containmentMu sync.Mutex
	containment   map[int]func()
}

const lifecycleEnvironmentEnv = "VROOLI_ENVIRONMENT"

type lifecycleLogContext struct {
	Scenario  string
	Operation string
	Phase     string
	// RunID, when set, overrides the generated run identifier so a caller can
	// pin the id shared with a persisted run record.
	RunID string
}

// RunMeta is the lifecycle-run bookkeeping a phase run produces: the run-id used
// in the log markers and the wall-clock boundaries. The CLI uses it to populate
// the typed test-run result and persist the run record.
type RunMeta struct {
	RunID     string
	StartedAt time.Time
	EndedAt   time.Time
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

type lifecycleDeps struct {
	sleep                   func(time.Duration)
	now                     func() time.Time
	signalProcessGroup      func(int, bool) error
	signalPID               func(int, bool) error
	listeningPIDs           func(int) ([]int, error)
	readScenarioRecords     func(string, string) ([]process.Record, error)
	isPIDRunning            func(int) bool
	cleanStaleLocks         func() error
	resourceStatus          func(string, bool) (resourcecontrol.Status, error)
	resourceManifest        func(string) (resourcemanifest.ResourceManifest, error)
	runResource             func(string, []string, io.Writer, io.Writer) error
	runResourceCLI          func(string, []string, io.Writer, io.Writer) error
	inspectPort             func(int) network.PortInspection
	readProcessEnv          func(int) (map[string]string, error)
	enforceHostRequirements func(hostreqrun.Options) (vrooliruntime.Report, error)
	runtimeRegistry         func(context.Context, string) (scenarioRuntimeStore, error)
	hostSession             func(context.Context, string) (hostsession.Snapshot, error)
	ensureRuntimeSupervisor func(context.Context, string, io.Writer, io.Writer) error
	captureInitiator        func() process.InitiatorInfo
}

type hostProbeDeps struct {
	stat        func(string) (os.FileInfo, error)
	readFile    func(string) ([]byte, error)
	lookPath    func(string) (string, error)
	getenv      func(string) string
	userHomeDir func() (string, error)
	walkDir     func(string, fs.WalkDirFunc) error
	// goListJSON runs `go list -deps -json .` in dir and returns the raw JSON
	// stream on stdout. It is the input-precision adapter seam: the freshness
	// engine uses it to derive the exact import closure of a Go binary (only the
	// packages it actually compiles), falling back to the static replace-dir walk
	// when it is nil or returns an error. Tests inject canned JSON.
	goListJSON func(dir string) ([]byte, error)
	// goListJSONContext is the cancellation-aware form used by a bounded
	// lifecycle start. The legacy seam above remains for callers without a
	// lifecycle context.
	goListJSONContext func(context.Context, string) ([]byte, error)
	// goEnv resolves the named `go env` determinants to their effective values
	// (toolchain defaults + overrides actually in effect), returning only keys
	// with a non-empty value. It is the build-environment seam: the freshness
	// engine keys output-determining vars (GOOS/GOARCH/CGO_ENABLED/…) so a
	// byte-identical source tree cross-compiled or built with CGO flipped reads as
	// stale. Unlike os.Getenv it reflects the value the toolchain will actually
	// use (e.g. CGO_ENABLED=1 when unset). Returns an empty map when go is absent
	// (omit-on-unknown). Tests inject canned values.
	goEnv func(keys ...string) map[string]string
	// nodeVersion returns the host Node.js major version (e.g. "20"), empty when
	// node is absent. Keyed into the ui-bundle digest because Vite/esbuild output
	// can differ across Node majors from identical source. Tests inject.
	nodeVersion func() string
	// recognizeArtifact is the OS artifact-recognition seam: given a stat'd path
	// and its FileInfo, it reports capability-flagged evidence of whether the path
	// is a runnable compiled build artifact (exec bit on Unix, executable
	// extension on Windows). The freshness decision path consumes its evidence and
	// degrades to "assume runnable" when the probe is unavailable, keeping
	// runtime.GOOS out of decision logic. Tests inject canned evidence.
	recognizeArtifact func(path string, info os.FileInfo) artifactEvidence
	// volumeCaseEvidence reports whether the volume holding a path is
	// case-insensitive, so manifest rel-path comparison can case-fold there. When
	// unavailable it degrades to case-sensitive (correctness-safe). Tests inject.
	volumeCaseEvidence func(path string) caseEvidence
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
		stat:               os.Stat,
		readFile:           os.ReadFile,
		lookPath:           exec.LookPath,
		getenv:             os.Getenv,
		userHomeDir:        os.UserHomeDir,
		walkDir:            filepath.WalkDir,
		goListJSON:         defaultGoListJSON,
		goListJSONContext:  defaultGoListJSONContext,
		goEnv:              defaultGoEnv,
		nodeVersion:        defaultNodeVersion,
		recognizeArtifact:  recognizeArtifact,
		volumeCaseEvidence: hostVolumeCaseEvidence,
	}
}

// defaultGoListJSON runs the Go toolchain's dependency lister for the package in
// dir, matching how scenarios build (GOWORK=off). A bounded timeout keeps a wedged
// toolchain from stalling a scenario start; on any failure the caller falls back
// to the static input walk, so an error here is never fatal.
func defaultGoListJSON(dir string) ([]byte, error) {
	return defaultGoListJSONContext(context.Background(), dir)
}

func defaultGoListJSONContext(parent context.Context, dir string) ([]byte, error) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(parent, tuning.LifecycleOperationTimeout())
	defer cancel()
	cmd := shell.NewCommandContext(ctx, goBin, "list", "-deps", "-json", ".")
	cmd.Dir = dir
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"GOWORK=off"})
	return cmd.Output()
}

type StartOptions struct {
	// Context carries cancellation from the owning caller through the complete
	// recursive start graph. Nil preserves the historical background context
	// for library callers that do not need cancellation.
	Context            context.Context
	CustomPath         string
	CleanStale         bool
	BestEffort         bool
	ForceSetup         bool
	ForceSetupScenario string
	Operation          string
	// Variant selects which instance to start ("" / "live" for the canonical
	// primary, "shadow" etc. for an alternate). It is sugar for a "name@variant"
	// argument; both are resolved through scenarioruntime.ParseInstanceKey at the
	// entry point, so passing them disagreeing is a hard error. See §1a.
	Variant string
	// stopFirst makes the start begin with an unconditional stop + settle of
	// the target instance — restart semantics. Set only by Runner.Restart;
	// unexported so external callers express restart through Restart.
	stopFirst bool
	// hostRequirementsPreflighted contains every scenario path covered by the
	// one tree-level host-requirement pass.
	hostRequirementsPreflighted map[string]struct{}
}

type StopOptions struct {
	// Context carries cancellation through teardown. Nil preserves the
	// historical background behavior for library callers.
	Context    context.Context
	CustomPath string
	// Variant selects which instance to stop. Empty / "live" stops only the
	// canonical instance and never reaps a sibling shadow (and vice versa).
	Variant string
}

type PhaseOptions struct {
	Context                 context.Context
	CustomPath              string
	AllowSkipMissingRuntime bool
	ManageRuntime           bool
	ProjectMode             bool
	// RunID, when set, is used verbatim as the lifecycle run identifier instead
	// of generating one. It lets the CLI share a single id across the log
	// markers, the persisted test-run record, and a detached child's run (so the
	// parent that launched `--detach` and the child agree on the run-id).
	RunID string
}

type Result struct {
	Scenario           scenario.Scenario
	AllocatedPorts     map[string]int
	Health             string
	FailedDependencies []string
	FailedResources    []string
	AlreadyRunning     bool
	// CredentialGaps names the declared credentials that did not resolve for
	// this start. The scenario still ran; these are the resources that cannot
	// do their job until an operator acts.
	CredentialGaps []resourceenv.MissingCredential
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
	environment := strings.TrimSpace(os.Getenv(lifecycleEnvironmentEnv))
	if environment == "" {
		environment = "development"
	}
	return &Runner{
		Root:        filepath.Clean(root),
		Home:        filepath.Clean(home),
		Environment: hostreq.NormalizeEnvironment(environment),
		Out:         stdout,
		Err:         stderr,
		Ports:       manager,
		Logger:      logx.WithSubsystem(baseLogger, "lifecycle"),
		Engagements: defaultEngagementResolver,
		deps:        deps,
	}, nil
}

func (r *Runner) environmentProfile() string {
	return hostreq.NormalizeEnvironment(r.Environment)
}

//nolint:gocyclo // runtime dependency assembly retains optional-resource and compatibility fallbacks.
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
			manager, err := cliinstall.NewManager(r.Root, r.Home)
			if err != nil {
				return fmt.Errorf("ensure resource CLI %s: %w", name, err)
			}
			if err := manager.EnsureResourceCLI(name); err != nil {
				return fmt.Errorf("ensure resource CLI %s: %w", name, err)
			}
			return resources.NewController(r.Root, r.Home).RunResourceCLI(name, args, stdout, stderr)
		}
	}
	if deps.cleanStaleLocks == nil {
		deps.cleanStaleLocks = func() error {
			_, err := maintenance.NewController(r.Root, r.Home).CleanStaleLocks()
			return err
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
	if deps.runtimeRegistry == nil {
		deps.runtimeRegistry = func(ctx context.Context, home string) (scenarioRuntimeStore, error) {
			return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
		}
	}
	if deps.hostSession == nil {
		deps.hostSession = hostsession.DefaultProvider{}.Current
	}
	if deps.captureInitiator == nil {
		deps.captureInitiator = process.Initiator
	}
	if deps.ensureRuntimeSupervisor == nil {
		deps.ensureRuntimeSupervisor = func(ctx context.Context, home string, stdout io.Writer, stderr io.Writer) error {
			cfg := runtimesupervisor.EnvConfig()
			cfg.HomeDir = home
			cfg.Stdout = stdout
			cfg.Stderr = stderr
			return runtimesupervisor.EnsureRunning(ctx, cfg)
		}
	}
	return deps
}

// recordSlug returns the on-disk record/log/lock directory name for an
// instance: the bare scenario slug for the live instance, "scenario@variant"
// for any non-live variant. Record, log, and advisory-lock paths use it so two
// variants of the same scenario never clobber each other's files. "@" is
// filename-safe on ext4 and Windows, so it is used directly. See §1a.
func recordSlug(item scenario.Scenario) string {
	return scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}.Slug()
}

func (r *Runner) Start(name string, opts StartOptions) (Result, error) {
	key, err := scenarioruntime.ParseInstanceKey(name, opts.Variant)
	if err != nil {
		return Result{}, err
	}
	opts.Variant = key.Variant
	// Attach-then-take-over loop: a busy lock held by a live in-flight start
	// is awaited (same verdict, no ErrScenarioBusy); a busy lock whose owner
	// dies mid-flight is retried so this caller resumes the start. Bounded so
	// a pathological churn of dying owners cannot spin forever.
	const maxTakeoverAttempts = 3
	for attempt := 0; ; attempt++ {
		release, err := r.acquireScenarioLock(key.Slug())
		if err == nil {
			result, startErr := r.startLocked(key.Scenario, opts, newStartSession(opts.Context))
			release()
			return result, startErr
		}
		if !errors.Is(err, ErrScenarioBusy) {
			r.logError("Scenario start blocked by concurrent invocation", err, logx.AttrScenario, key.Slug())
			return Result{}, err
		}
		item, loadErr := r.loadScenario(key.Scenario, opts.CustomPath)
		if loadErr != nil {
			return Result{}, loadErr
		}
		item.Variant = key.Variant
		attach := r.attachToInFlightStart(item, err)
		if attach.takeOver {
			if attempt+1 >= maxTakeoverAttempts {
				// Name the holder: "abandoned" alone sent operators hunting for a
				// dead process when a live one was holding the lock the whole time.
				var busy *ScenarioBusyError
				if errors.As(err, &busy) && busy.HolderPID > 0 {
					return Result{}, fmt.Errorf("start %q: %d successive takeover attempts failed; lock still held by pid %d", key.Slug(), maxTakeoverAttempts, busy.HolderPID)
				}
				return Result{}, fmt.Errorf("start %q: %d successive in-flight starts were abandoned; giving up takeover", key.Slug(), maxTakeoverAttempts)
			}
			continue
		}
		if attach.err != nil {
			r.logError("Scenario start blocked by concurrent invocation", attach.err, logx.AttrScenario, key.Slug())
			return Result{}, attach.err
		}
		return attach.result, nil
	}
}

// StartContext is the explicit cancellation-aware entry point. Start keeps
// the historical argument order for library compatibility; new callers should
// prefer this form so the operation context cannot be omitted accidentally.
func (r *Runner) StartContext(ctx context.Context, name string, opts StartOptions) (Result, error) {
	opts.Context = ctx
	return r.Start(name, opts)
}

// startLocked is the lock-free body of Start. Callers must already hold the
// per-scenario advisory lock for `name` (acquireScenarioLock). Used by Start
// and Restart to avoid double-acquiring the lock from the same goroutine.
func (r *Runner) startLocked(name string, opts StartOptions, session *startSession) (Result, error) {
	// Validate and converge the target's host requirements before restart tears
	// down a working instance or dependency bootstrap performs unrelated work.
	// The execute path retains its enforcement as a safety net for recursive and
	// future start paths, but this preflight is the user-visible transaction
	// boundary for top-level starts and restarts.
	item, err := r.loadScenario(name, opts.CustomPath)
	if err != nil {
		return Result{}, err
	}
	treePaths, err := r.startTreeScenarioPaths(item)
	if err != nil {
		return Result{}, err
	}
	if err := r.enforceScenarioHostRequirementsTree(item, treePaths); err != nil {
		return Result{}, fmt.Errorf("preflight host requirements for scenario %q: %w", name, err)
	}
	opts.hostRequirementsPreflighted = make(map[string]struct{}, len(treePaths))
	for _, path := range treePaths {
		opts.hostRequirementsPreflighted[path] = struct{}{}
	}

	// Durable start-operation record: every top-level start/restart is
	// introspectable by other processes for the duration of the run and
	// after. Nil recorder (registry unavailable) degrades to an unrecorded
	// start — the record is progress, never authority.
	if recorder := r.beginStartOperationRecord(name, opts); recorder != nil {
		detach := r.attachSink(recorder)
		defer detach()
		defer recorder.close()
	}
	if opts.stopFirst {
		// Restart semantics: unconditional teardown before the start body,
		// announced (and rendered) before "starting …" like the historical
		// stop+start sequence.
		if err := r.stopLocked(name, StopOptions{Context: opts.Context, Variant: opts.Variant}); err != nil {
			return Result{}, err
		}
		if err := r.waitForInstanceReleased(opts.Context, name, opts.Variant); err != nil {
			return Result{}, err
		}
	}
	r.publish(ProgressEvent{Kind: EventOperationStarted, Scenario: name, Operation: defaultIfEmpty(opts.Operation, "start")})
	r.logInfo("Scenario start requested",
		logx.AttrScenario, name,
		"best_effort", opts.BestEffort,
		"clean_stale", opts.CleanStale,
		"force_setup", opts.ForceSetup,
	)
	result, err := r.startWithState(name, opts, session)
	if err != nil {
		r.publish(ProgressEvent{Kind: EventOperationFailed, Scenario: name, Operation: defaultIfEmpty(opts.Operation, "start"), Err: err})
		r.logError("Scenario start failed", err, logx.AttrScenario, name)
		return Result{}, err
	}
	if !result.AlreadyRunning {
		// The reuse fast-path publishes its own AlreadyRunning completion
		// (with the "is already running" line) inside startScenario.
		r.publish(ProgressEvent{Kind: EventOperationCompleted, Scenario: name, Operation: defaultIfEmpty(opts.Operation, "start"), Verdict: result.Health})
	}
	r.logInfo("Scenario start completed",
		logx.AttrScenario, name,
		logx.AttrStatus, result.Health,
		"already_running", result.AlreadyRunning,
		"failed_dependencies", len(result.FailedDependencies),
	)
	return result, nil
}

func (r *Runner) startWithState(name string, opts StartOptions, session *startSession) (Result, error) {
	item, err := r.loadScenario(name, opts.CustomPath)
	if err != nil {
		return Result{}, err
	}
	// Stamp the requested variant onto the descriptor so every downstream
	// registry/lock/port/storage derivation addresses this instance. Empty ⇒
	// live, so the pre-variant path is unchanged.
	item.Variant = opts.Variant
	// Port policy is enforced here rather than inside scenario.ReadService so
	// that Stop, Status, List, and other observation-only paths can still
	// operate on manifests whose ports pre-date the canonical bands.
	if err := scenario.ValidateManifestPorts(item.ServicePath, item.Manifest.Ports); err != nil {
		return Result{}, err
	}
	return r.startScenario(item, opts, session)
}

func (r *Runner) startScenario(item scenario.Scenario, opts StartOptions, session *startSession) (Result, error) {
	branch := session.childStack(item.Slug)
	// --clean-stale: reconcile stale runtime registry state (expired claims,
	// dead-owner instances) before the top-level start so a previous crash
	// doesn't block port allocation. Dependencies skip this (len(stack) > 0):
	// one reconcile per user-initiated start is enough.
	if opts.CleanStale && len(branch.stack) == 1 {
		r.logDebug("Reconciling stale runtime registry state before scenario start", logx.AttrScenario, item.Slug)
		if cleanErr := r.runtimeDeps().cleanStaleLocks(); cleanErr != nil {
			return Result{}, cleanErr
		}
	}
	failedDeps, failedResources, err := r.bootstrapScenarioDependencies(item, opts, branch)
	if err != nil {
		return Result{}, err
	}

	forceSetup := forceSetupFor(opts, item.Slug)
	observed, err := r.observeRuntime(item, forceSetup, branch)
	if err != nil {
		return Result{}, err
	}
	plan := planStart(observed.planInput())
	switch plan.Decision {
	case decisionReuseRunning:
		health := scenario.EvaluateHealth(item.Manifest.HealthConfig(), observed.View.Ports)
		r.publish(ProgressEvent{Kind: EventOperationCompleted, Scenario: item.Slug, Verdict: health, AlreadyRunning: true})
		r.logInfo("Scenario already running and healthy",
			logx.AttrScenario, item.Slug,
			logx.AttrStatus, health,
			"registry_instance", observed.View.Instance.InstanceID,
		)
		return Result{
			Scenario:           item,
			AllocatedPorts:     observed.View.Ports,
			Health:             health,
			FailedDependencies: failedDeps,
			FailedResources:    failedResources,
			AlreadyRunning:     true,
		}, nil
	case decisionStopThenStart:
		// Running-but-unfit, or a stale registry row whose leftover claims
		// would collide with a fresh start.
		r.logDebug("Stopping existing instance before start",
			logx.AttrScenario, item.Slug, "reason", plan.RestartReason)
		// Build the replacement while the current instance still owns the
		// serving process. A failed setup therefore returns without taking a
		// healthy instance offline; the later artifact-swap phase will make the
		// output write itself atomic for concurrently served UI bundles.
		if observed.View.Authoritative && string(observed.View.Instance.Status) != scenarioruntime.StatusFailed {
			if err := r.prepareReplacementArtifacts(ctxOrBackground(opts.Context), item, observed.View); err != nil {
				return Result{}, err
			}
		}
		if err := r.stopLocked(item.Slug, StopOptions{Context: opts.Context, Variant: item.Variant}); err != nil {
			return Result{}, err
		}
		if err := r.waitForInstanceReleased(opts.Context, item.Slug, item.Variant); err != nil {
			return Result{}, err
		}
	}
	return r.executeStart(item, opts, forceSetup, branch, failedDeps, failedResources)
}

func ctxOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (r *Runner) prepareReplacementArtifacts(ctx context.Context, item scenario.Scenario, view registryRuntimeView) error {
	env := envFromRuntimeView(item.Manifest, view)
	if _, err := r.runWithLifecycleLog(startLifecycleLogContext(item.Slug, "stage", "setup"), func(logWriter, childWriter io.Writer) error {
		_, execErr := r.executePhaseDetailed(ctx, item, "setup", env, logWriter, childWriter, false)
		return execErr
	}); err != nil {
		return fmt.Errorf("prepare replacement artifacts for %s: %w", item.Slug, err)
	}
	r.stampScenarioFreshness(item)
	return nil
}

// executeStart runs the start steps for one instance whose teardown/reuse
// arbitration is already settled: registry lease, host requirements, port
// environment, setup (when needed), develop, health gate, registry
// finalization, and supervisor handoff. It owns the failure rollback for
// side effects it created.
//
//nolint:gocyclo // start execution preserves dependency, setup, process, timeout, and ownership transitions.
func (r *Runner) executeStart(item scenario.Scenario, opts StartOptions, forceSetup bool, session *startSession, failedDeps, failedResources []string) (result Result, err error) {
	ctx := session.context()
	cleanupOnError := false
	runtimeSession := disabledRuntimeRegistrySession()
	defer func() {
		if err == nil {
			if closeErr := runtimeSession.close(); closeErr != nil {
				err = closeErr
			}
			return
		}
		if runtimeErr := runtimeSession.fail(ctx, err); runtimeErr != nil {
			err = errors.Join(err, runtimeErr)
		}
		_ = runtimeSession.close()
		if !cleanupOnError {
			return
		}
		// Rollback is intentionally scoped to the scenario currently being started.
		// Dependencies and resources that were started earlier in the recursive chain
		// are shared runtime infrastructure and may already be needed by other live
		// scenarios, so this rollback must not unwind them opportunistically.
		if cleanupErr := r.cleanupScenarioRuntimeWithRegistryContext(ctx, item.Slug, item.Variant, opts.CustomPath, false, false); cleanupErr != nil {
			r.logError("Failed to roll back failed scenario start", cleanupErr, logx.AttrScenario, recordSlug(item))
			err = errors.Join(err, fmt.Errorf("rollback failed: %w", cleanupErr))
		}
	}()

	runtimeSession, err = r.beginRuntimeRegistryStart(ctx, item)
	if err != nil {
		return Result{}, err
	}

	if _, covered := opts.hostRequirementsPreflighted[item.Path]; !covered {
		if err := r.enforceScenarioHostRequirements(item); err != nil {
			return Result{}, err
		}
	}

	env, err := r.prepareScenarioEnvironment(ctx, item, runtimeSession)
	cleanupOnError = true
	if err != nil {
		return Result{}, err
	}
	if err := runtimeSession.adoptOrReservePorts(ctx, item, env); err != nil {
		return Result{}, err
	}
	runtimeSession.injectEnv(env.EnvVars)

	setupNeeded, _, err := r.setupNeededCached(item, forceSetup, session)
	if err != nil {
		return Result{}, err
	}

	if setupNeeded {
		r.publish(ProgressEvent{Kind: EventPhaseStarted, Scenario: item.Slug, Phase: "setup"})
		r.logInfo("Executing setup phase for scenario", logx.AttrScenario, item.Slug, logx.AttrPhase, "setup")
		if err := runtimeSession.setPhase(ctx, "setup"); err != nil {
			return Result{}, err
		}
		if err := runtimeSession.keepLeaseAlive(ctx, r.leaseRenewalWarning(item, "setup"), func() error {
			_, err := r.runWithLifecycleLog(startLifecycleLogContext(item.Slug, opts.Operation, "setup"), func(logWriter, childWriter io.Writer) error {
				_, err := r.executePhaseDetailed(ctx, item, "setup", env.EnvVars, logWriter, childWriter, forceSetup)
				return err
			})
			return err
		}); err != nil {
			return Result{}, err
		}
		// Record the content-fingerprint manifest for every freshness-checked
		// artifact now that setup has (re)built them. Subsequent checks become
		// manifest-authoritative; the next freshness eval reads a stat-cache
		// stamp instead of walking the source tree.
		r.stampScenarioFreshness(item)
		if err := runtimeSession.heartbeat(ctx); err != nil {
			return Result{}, err
		}
		r.publish(ProgressEvent{Kind: EventPhaseCompleted, Scenario: item.Slug, Phase: "setup"})
	}

	r.publish(ProgressEvent{Kind: EventPhaseStarted, Scenario: item.Slug, Phase: "develop"})
	r.logInfo("Executing develop phase for scenario", logx.AttrScenario, item.Slug, logx.AttrPhase, "develop")
	if err := runtimeSession.setPhase(ctx, "develop"); err != nil {
		return Result{}, err
	}
	if err := runtimeSession.keepLeaseAlive(ctx, r.leaseRenewalWarning(item, "develop"), func() error {
		_, err := r.runWithLifecycleLog(startLifecycleLogContext(item.Slug, opts.Operation, "develop"), func(logWriter, childWriter io.Writer) error {
			_, err := r.executePhaseDetailed(ctx, item, "develop", env.EnvVars, logWriter, childWriter, false)
			return err
		})
		return err
	}); err != nil {
		return Result{}, err
	}
	if err := runtimeSession.heartbeat(ctx); err != nil {
		return Result{}, err
	}
	r.publish(ProgressEvent{Kind: EventPhaseCompleted, Scenario: item.Slug, Phase: "develop"})

	r.publish(ProgressEvent{Kind: EventHealthWaiting, Scenario: item.Slug})
	healthStatus, err := r.waitForHealth(ctx, item, env.EnvVars)
	if err != nil {
		return Result{}, err
	}
	if err := runtimeSession.recordHealth(ctx, item, env, healthStatus); err != nil {
		return Result{}, err
	}

	if err := runtimeSession.bindPorts(ctx); err != nil {
		return Result{}, err
	}
	if err := runtimeSession.markRunning(ctx); err != nil {
		return Result{}, err
	}
	if err := r.ensureRuntimeSupervisor(ctx, runtimeSession); err != nil {
		return Result{}, err
	}
	// Ownership must change hands before this process does, or the instance is
	// left leased to a PID that is about to disappear.
	r.attachSupervision(ctx, &runtimeSession, item)
	if err := runtimeSession.publishPeerRecord(ctx, r.Home); err != nil {
		return Result{}, err
	}

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

	r.printCredentialGapSummary(env)

	result = Result{
		Scenario:           item,
		AllocatedPorts:     env.AllocatedPorts,
		Health:             healthStatus,
		FailedDependencies: failedDeps,
		FailedResources:    failedResources,
		CredentialGaps:     env.CredentialGaps,
	}
	cleanupOnError = false
	return result, nil
}

func (r *Runner) bootstrapScenarioDependencies(item scenario.Scenario, opts StartOptions, session *startSession) ([]string, []string, error) {
	failedDeps, err := r.ensureDependencies(item, opts, session)
	if err != nil {
		return nil, nil, err
	}
	failedResources, err := r.ensureResourceDependencies(item, opts)
	if err != nil {
		return nil, nil, err
	}
	return failedDeps, failedResources, nil
}

// printCredentialGapSummary tells the operator, at the end of a successful
// start, exactly which variable is missing from which resource and the command
// that fixes it. The start succeeded; this is the follow-up work, so it reads
// as a to-do list rather than as a failure.
func (r *Runner) printCredentialGapSummary(env ports.Environment) {
	if len(env.CredentialGaps) == 0 {
		return
	}
	// Routed through r.Err directly rather than consoleErr: this is actionable
	// operator work, so it must survive quiet mode exactly as error replay does.
	out := r.Err
	if out == nil {
		return
	}
	fmt.Fprintf(out, "\nCredentials not resolved (%d); the scenario is running and these resources are degraded:\n",
		len(env.CredentialGaps))
	for _, gap := range env.CredentialGaps {
		label := gap.Env
		if gap.Label != "" {
			label += " (" + gap.Label + ")"
		}
		requirement := "optional"
		if gap.Required {
			requirement = "required"
		}
		fmt.Fprintf(out, "  %s → %s [%s]\n      %s\n", gap.Resource, label, requirement, gap.Remediation)
	}
	fmt.Fprintf(out, "  Provisioning a credential takes effect on the next resource use; no control-plane restart is needed.\n\n")
}

// logCredentialGaps emits one line per start, not one per descriptor: a
// resource declares several credentials, and identical
// warnings would bury the one fact the operator needs.
func (r *Runner) logCredentialGaps(slug string, env ports.Environment) {
	if len(env.CredentialGaps) == 0 {
		return
	}
	required := 0
	for _, gap := range env.CredentialGaps {
		if gap.Required {
			required++
		}
	}
	first := env.CredentialGaps[0]
	r.logWarn("Scenario started with unresolved credentials",
		logx.AttrScenario, slug,
		"credential_gaps", len(env.CredentialGaps),
		"required_gaps", required,
		"provider_state", string(env.CredentialProvider),
		"first_variable", first.Env,
		"remediation", first.Remediation,
	)
}

func (r *Runner) prepareScenarioEnvironment(ctx context.Context, item scenario.Scenario, runtimeSession runtimeRegistrySession) (ports.Environment, error) {
	if err := r.cleanupFixedPortOrphans(item); err != nil {
		return ports.Environment{}, err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	env, err := r.Ports.BuildEnvironmentWithRuntimeClaims(item, nil, runtimeSession.portClaimOptions(ctx))
	if err != nil {
		return ports.Environment{}, err
	}
	r.logCredentialGaps(item.Slug, env)

	return env, nil
}

func (r *Runner) Stop(name string, opts StopOptions) error {
	key, err := scenarioruntime.ParseInstanceKey(name, opts.Variant)
	if err != nil {
		return err
	}
	opts.Variant = key.Variant
	release, err := r.acquireScenarioLock(key.Slug())
	if err != nil {
		r.logError("Scenario stop blocked by concurrent invocation", err, logx.AttrScenario, key.Slug())
		return err
	}
	defer release()
	return r.stopLocked(key.Scenario, opts)
}

// StopContext is the explicit cancellation-aware stop entry point.
func (r *Runner) StopContext(ctx context.Context, name string, opts StopOptions) error {
	opts.Context = ctx
	return r.Stop(name, opts)
}

// stopLocked is the lock-free body of Stop. Callers must already hold the
// per-scenario advisory lock. Used internally by startScenario (which is
// itself called under the Start/Restart lock) and by Restart.
func (r *Runner) stopLocked(name string, opts StopOptions) error {
	slug := scenarioruntime.InstanceKey{Scenario: name, Variant: opts.Variant}.Slug()
	r.publish(ProgressEvent{Kind: EventStopStarted, Scenario: slug, Operation: "stop"})
	r.logInfo("Scenario stop requested", logx.AttrScenario, slug)
	if err := r.cleanupScenarioRuntimeWithRegistryContext(opts.Context, name, opts.Variant, opts.CustomPath, true, true); err != nil {
		r.logError("Failed to remove scenario locks", err, logx.AttrScenario, slug)
		return err
	}
	r.logInfo("Scenario stop completed", logx.AttrScenario, slug)
	return nil
}

// cleanupScenarioRuntimeWithRegistry tears down one instance of a scenario.
// `name` is the bare scenario slug and `variant` selects the instance; the two
// are combined into a record slug for all on-disk record/log/lock operations
// and into an InstanceFilter for the registry, so stopping one variant never
// reaps a sibling (the reap-sibling bug fixed here). Empty variant ⇒ live.
func (r *Runner) cleanupScenarioRuntimeWithRegistryContext(ctx context.Context, name, variant, customPath string, includeManifestFixedPorts bool, writeRegistry bool) error {
	deps := r.runtimeDeps()
	if ctx == nil {
		ctx = context.Background()
	}
	key := scenarioruntime.InstanceKey{Scenario: name, Variant: variant}.Normalize()
	slug := key.Slug()
	runtimeStop := runtimeRegistryStopSession{}
	if writeRegistry {
		var err error
		runtimeStop, err = r.beginRuntimeRegistryStop(ctx, key.Scenario, key.Variant)
		if err != nil {
			return err
		}
	}
	defer runtimeStop.close()

	records, err := deps.readScenarioRecords(r.Home, slug)
	if err != nil {
		return err
	}

	processDir, err := process.ScenarioProcessDir(r.Home, slug)
	if err != nil {
		return err
	}
	stepFiles, globErr := filepath.Glob(filepath.Join(processDir, "*.json"))
	if globErr != nil {
		return globErr
	}

	liveRecords := process.LiveRecords(records)
	if len(liveRecords) > 0 {
		if err := r.terminateAndAwait(ctx, liveRecords); err != nil {
			return err
		}
	}

	for _, stepFile := range stepFiles {
		step := strings.TrimSuffix(filepath.Base(stepFile), filepath.Ext(stepFile))
		_ = process.RemoveScenarioRecord(r.Home, slug, step)
	}

	portsToCheck := make(map[int]struct{})
	if includeManifestFixedPorts {
		if item, loadErr := r.loadScenario(name, customPath); loadErr == nil {
			for _, portSummary := range item.Manifest.SortedPorts() {
				if portSummary.FixedPort != nil {
					portsToCheck[*portSummary.FixedPort] = struct{}{}
				}
			}
		}
	}

	if err := r.killOrphansOnPortsContext(ctx, portsToCheck); err != nil {
		return err
	}

	if err := r.verifyPortsReleasedContext(ctx, key, portsToCheck); err != nil {
		return err
	}

	if err := runtimeStop.finish(ctx); err != nil {
		return err
	}
	if err := scenarioenv.Remove(r.Home, key.Scenario); err != nil {
		return fmt.Errorf("remove scenario peer record: %w", err)
	}
	return nil
}

// Restart shares the start pipeline: it is a start whose first step is an
// unconditional stop (stopFirst). Fresh artifacts are reused; callers opt
// into an unconditional rebuild with ForceSetup.
func (r *Runner) Restart(name string, opts StartOptions) (Result, error) {
	key, err := scenarioruntime.ParseInstanceKey(name, opts.Variant)
	if err != nil {
		return Result{}, err
	}
	opts.Variant = key.Variant
	slug := key.Slug()
	release, err := r.acquireScenarioLock(slug)
	if err != nil {
		r.logError("Scenario restart blocked by concurrent invocation", err, logx.AttrScenario, slug)
		return Result{}, err
	}
	defer release()
	r.logInfo("Scenario restart requested", logx.AttrScenario, slug)
	if opts.ForceSetup {
		opts.ForceSetupScenario = key.Scenario
	}
	opts.Operation = "restart"
	opts.stopFirst = true
	result, err := r.startLocked(key.Scenario, opts, newStartSession(opts.Context))
	if err != nil {
		r.logError("Scenario restart failed", err, logx.AttrScenario, slug)
		return Result{}, err
	}
	r.logInfo("Scenario restart completed", logx.AttrScenario, slug, logx.AttrStatus, result.Health)
	return result, nil
}

// RestartContext is the explicit cancellation-aware restart entry point.
func (r *Runner) RestartContext(ctx context.Context, name string, opts StartOptions) (Result, error) {
	opts.Context = ctx
	return r.Restart(name, opts)
}

// enforceScenarioHostRequirements resolves and installs host requirements
// declared directly on the scenario. Resource-level declarations are handled by
// enforceResourceHostRequirements before each resource dep starts, so scope is
// kept tight: only the root manifest plus the scenario's own declarations.
// A scenario with no declared hostTools/hostSafeguards yields a no-op.
func (r *Runner) enforceScenarioHostRequirements(item scenario.Scenario) error {
	return r.enforceScenarioHostRequirementsTree(item, []string{item.Path})
}

func (r *Runner) startTreeScenarioPaths(root scenario.Scenario) ([]string, error) {
	paths := []string{}
	seen := map[string]struct{}{}
	var visit func(scenario.Scenario) error
	visit = func(item scenario.Scenario) error {
		if _, ok := seen[item.Path]; ok {
			return nil
		}
		seen[item.Path] = struct{}{}
		paths = append(paths, item.Path)
		for dependencyName := range item.Manifest.Dependencies.Scenarios {
			dependency, err := r.loadScenario(dependencyName, "")
			if err != nil {
				return err
			}
			if err := visit(dependency); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root); err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func (r *Runner) enforceScenarioHostRequirementsTree(item scenario.Scenario, paths []string) error {
	deps := r.runtimeDeps()
	if deps.enforceHostRequirements == nil {
		return nil
	}
	if _, err := deps.enforceHostRequirements(hostreqrun.Options{
		Root:          r.Root,
		Home:          r.Home,
		Environment:   r.environmentProfile(),
		When:          "develop",
		Resources:     "none",
		ScenarioPaths: paths,
		AutoInstall:   true,
		Stdout:        r.Out,
		Stderr:        r.Err,
		Label:         "scenario-tree:" + item.Slug,
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
	r.admitResourceCapacity(resourceName)
	return nil
}

// admitResourceCapacity runs the advisory capacity broker admission for a
// resource before it starts (plan §7 Phase 3). It is ALWAYS advisory in V1:
// dormant unless the resource declares a `capacity` block in resource.json and
// enforcement is enabled, and it NEVER blocks the start — any error is logged,
// never propagated. Flag OFF (VROOLI_CAPACITY_ENFORCE=off) or no declared block
// makes this a byte-identical no-op.
func (r *Runner) admitResourceCapacity(resourceName string) {
	result, err := capacity.AdmitResource(context.Background(), capacity.AdmitOptions{
		Root:         r.Root,
		ResourceName: resourceName,
	})
	if err != nil {
		r.logWarn("Capacity admission skipped (advisory)", logx.AttrDependency, resourceName, "error", err.Error())
		return
	}
	if result.Skipped {
		return
	}
	r.logInfo("Capacity admission recorded",
		logx.AttrDependency, resourceName,
		"verdict", result.Verdict.Kind,
		"granted_bytes", result.Verdict.GrantedBytes,
		"enforce", result.Enforce,
		"claim_id", result.ClaimID)
	for _, warn := range result.Verdict.Warnings {
		r.logWarn("Capacity admission warning", logx.AttrDependency, resourceName, "warning", warn)
	}
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

// verifyPortsReleased polls each port for up to ~2 s after the kill loop and
// returns a loud error if any are still held. Surfacing this at stop time
// lets Restart fail fast with a diagnostic rather than silently racing into
// a Start that will itself fail with the generic "port already in use".
func (r *Runner) verifyPortsReleased(key scenarioruntime.InstanceKey, portsToCheck map[int]struct{}) error {
	// Compatibility wrapper for tests and library callers without a lifecycle
	// context. The start/stop path uses the context-aware variant below.
	return r.verifyPortsReleasedContext(context.Background(), key, portsToCheck)
}

func (r *Runner) verifyPortsReleasedContext(ctx context.Context, key scenarioruntime.InstanceKey, portsToCheck map[int]struct{}) error {
	if len(portsToCheck) == 0 {
		return nil
	}
	scenarioName := key.Slug()
	deps := r.runtimeDeps()
	stillBound := make(map[int][]int)
	err := AwaitContext(ctx, r.awaitClock(), AwaitPolicy{Timeout: tuning.LifecycleTransitionTimeout(), Interval: tuning.LifecyclePollInterval()}, func() (bool, error) {
		stillBound = make(map[int][]int)
		for port := range portsToCheck {
			got, err := deps.listeningPIDs(port)
			if err != nil {
				// listeningPIDs swallows exec errors to nil; any real error
				// means we cannot verify, so surface it.
				return false, fmt.Errorf("verify port %d released: %w", port, err)
			}
			if len(got) > 0 {
				stillBound[port] = got
			}
		}
		return len(stillBound) == 0, nil
	})
	if err != nil && ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil && !errors.Is(err, ErrAwaitExpired) {
		return err
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
	return r.killOrphansOnPortsContext(context.Background(), portsToCheck)
}

func (r *Runner) killOrphansOnPortsContext(ctx context.Context, portsToCheck map[int]struct{}) error {
	deps := r.runtimeDeps()
	pidsToKill := []int{}
	for port := range portsToCheck {
		pids, err := deps.listeningPIDs(port)
		if err != nil {
			return err
		}
		pidsToKill = append(pidsToKill, pids...)
	}
	return r.terminatePIDs(ctx, pidsToKill)
}

func (r *Runner) cleanupFixedPortOrphans(item scenario.Scenario) error {
	key := scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}.Normalize()
	// Fixed ports are a live-only privilege (§1a / P1): a non-live variant never
	// claims them and so must never clean up "orphans" on them — that would reach
	// across the variant boundary and kill the live instance's fixed-port process.
	if !key.IsLive() {
		return nil
	}
	portsToCheck := make(map[int]struct{})
	for _, portSummary := range item.Manifest.SortedPorts() {
		if portSummary.FixedPort == nil {
			continue
		}
		portsToCheck[*portSummary.FixedPort] = struct{}{}
	}
	if len(portsToCheck) == 0 {
		return nil
	}
	return r.killManagedScenarioListenersContext(context.Background(), portsToCheck, key)
}

// envPortOrphanStrict disables the aggressive start-time fallback. When set
// to "true" the start path only kills listeners whose env vars positively
// identify them as this scenario's children; any other listener causes the
// usual "port already in use" error to surface so the operator can diagnose
// it. Leave unset in production — orphan children that lost env inheritance
// (node grandchildren under vite, for example) are the common real cause.
const envPortOrphanStrict = "VROOLI_PORT_ORPHAN_STRICT"

func (r *Runner) killManagedScenarioListenersContext(ctx context.Context, portsToCheck map[int]struct{}, key scenarioruntime.InstanceKey) error {
	key = key.Normalize()
	scenarioName := key.Slug()
	deps := r.runtimeDeps()
	targets := make(map[int]struct{})
	fallbackPorts := make(map[int][]int) // port -> pids seen without env match
	for port := range portsToCheck {
		inspection := deps.inspectPort(port)
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
			// Match on BOTH scenario and variant so orphan cleanup never reaches
			// across a scenario OR variant boundary. A listener with no
			// VROOLI_VARIANT (legacy / pre-variant process) is treated as live.
			listenerKey := scenarioruntime.InstanceKey{
				Scenario: strings.TrimSpace(env["VROOLI_SCENARIO"]),
				Variant:  strings.TrimSpace(env[scenarioruntime.EnvVariant]),
			}.Normalize()
			if listenerKey != key {
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
	pids := make([]int, 0, len(targets))
	for pid := range targets {
		pids = append(pids, pid)
	}
	return r.terminatePIDs(ctx, pids)
}

func listeningPIDs(port int) ([]int, error) {
	path, err := exec.LookPath("lsof")
	if err != nil {
		return nil, nil
	}
	cmd := shell.NewCommand(path, "-tiTCP:"+strconv.Itoa(port), "-sTCP:LISTEN")
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
	servicePath := filepath.Join(resolved, repocontractmeta.ProjectConfigDir, "service.json")
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

//nolint:gocyclo // lifecycle conditions combine independent dependency and process predicates.
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
		ok, err := jsonPathExistsWithDeps(checkPath(strings.SplitN(jsonSpec, ":", lifecycleParameterA)[0]), jsonSpec, deps)
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
	if key := condition.EnvSet; key != "" {
		if strings.TrimSpace(env[key]) == "" && strings.TrimSpace(deps.getenv(key)) == "" {
			return false, fmt.Sprintf("environment variable %q is not set", key), nil
		}
	}
	if key := condition.EnvNotSet; key != "" {
		if strings.TrimSpace(env[key]) != "" || strings.TrimSpace(deps.getenv(key)) != "" {
			return false, fmt.Sprintf("environment variable %q is set", key), nil
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
	parts := strings.SplitN(spec, ":", lifecycleParameterA)
	if len(parts) != lifecycleParameterA {
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

// firstFileNewerWithDeps walks root and returns the first file (in walk order)
// whose mtime is strictly after target's and that passes the include filter.
// Skip directories (.git, node_modules, dist, data, …) are pruned with
// fs.SkipDir so the walk never descends into them — this both avoids the
// 4400+-module node_modules sweep and keeps the result deterministic. The
// returned path is the offending file, which callers thread into honest
// "what changed" reason strings.
func firstFileNewerWithDepsContext(ctx context.Context, root, target string, deps hostProbeDeps, include func(path string, d fs.DirEntry) bool) (string, bool) {
	if err := ctx.Err(); err != nil {
		return "", false
	}
	if _, err := deps.stat(root); err != nil {
		return "", false
	}
	targetInfo, err := deps.stat(target)
	if err != nil {
		return "", false
	}
	targetTime := targetInfo.ModTime()

	var offender string
	walkErr := deps.walkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && isSkippableDirName(d.Name()) {
				return filepath.SkipDir
			}
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
			offender = path
			return errStopWalk
		}
		return nil
	})

	if errors.Is(walkErr, errStopWalk) {
		return offender, true
	}
	return "", false
}

// isSkippableDirName reports whether a directory basename is a build/VCS/output
// directory that the freshness walk should never descend into, applied at walk
// time so the subtree is pruned (fs.SkipDir) rather than filtered file-by-file.
func isSkippableDirName(name string) bool {
	switch name {
	case ".git", ".idea", ".vscode", "node_modules", "dist", "build", "coverage", "tmp", "data":
		return true
	}
	return false
}

// relForReason renders path relative to base (slash-separated) for human-facing
// reason strings, falling back to the absolute path when no clean relative form
// exists.
func relForReason(base, path string) string {
	if base == "" {
		return path
	}
	if rel, err := filepath.Rel(base, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return path
}

var errStopWalk = errors.New("stop walk")

type fileDependencySpec struct {
	Name string
	Spec string
}

func fileDependencySpecsWithDeps(packageJSON string, deps hostProbeDeps) ([]string, error) {
	dependencies, err := fileDependenciesWithDeps(packageJSON, deps)
	if err != nil {
		return nil, err
	}
	specs := make([]string, 0, len(dependencies))
	for _, dependency := range dependencies {
		specs = append(specs, dependency.Spec)
	}
	return specs, nil
}

func fileDependenciesWithDeps(packageJSON string, deps hostProbeDeps) ([]fileDependencySpec, error) {
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
	specs := []fileDependencySpec{}
	for _, deps := range []map[string]string{
		doc.Dependencies,
		doc.DevDependencies,
		doc.PeerDependencies,
		doc.OptionalDependencies,
	} {
		for name, value := range deps {
			if strings.HasPrefix(value, "file:") {
				specs = append(specs, fileDependencySpec{Name: name, Spec: value})
			}
		}
	}
	sort.Slice(specs, func(i, j int) bool {
		if specs[i].Spec != specs[j].Spec {
			return specs[i].Spec < specs[j].Spec
		}
		return specs[i].Name < specs[j].Name
	})
	return specs, nil
}

func resolveCheckPath(base, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(base, filepath.FromSlash(path))
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

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
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
