package setup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/capabilitycatalog"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/dockerhost"
	"github.com/vrooli/vrooli/internal/hostpresentation"
	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/onboardinghandoff"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/ports"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/projectstate"
	"github.com/vrooli/vrooli/internal/resources"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	setupNone = "none"
)

const (
	defaultEnvironment = "development"
	defaultAPIPort     = 8092
	onboardingSlug     = "vrooli-onboarding"
	onboardingSkipEnv  = "VROOLI_SKIP_ONBOARDING"
)

var (
	inspectDockerHealthFn    = dockerhost.InspectHealth
	ensureContainerRuntimeFn = vrooliruntime.EnsureTool
)

var (
	launchOnboardingAsOperatorFn = launchOnboardingAsOperator
	invokingUIDFn                = os.Geteuid
	startOnboardingScenarioFn    = startOnboardingScenario
)

type Options struct {
	DryRun bool
	// BootstrapOnly applies host requirements needed to make the native Vrooli
	// CLI build possible, then returns before credential, resource, and
	// completion side effects. It is used only by cross-platform bootstrap
	// flows whose temporary CLI cannot yet link the host credential backend.
	BootstrapOnly bool
	// CredentialPassphraseStdin reads one credential-store passphrase from
	// standard input instead of opening /dev/tty. It is intended for the
	// Bridge's secret-safe bootstrap channel; the passphrase is never logged or
	// written to the setup result.
	CredentialPassphraseStdin bool
	SudoMode                  string
	// MaintenanceWindow acknowledges that a setup safeguard may interrupt
	// an active graphical or remote-desktop session.
	MaintenanceWindow bool
	Environment       string
	Resources         string
	Scenarios         string
	Yes               string
	Verbose           bool
	IncludeOptional   bool
	// ResultPath writes the versioned terminal result to a separate JSON file.
	// It never changes human setup output streams.
	ResultPath string
	// Onboarding selects the post-bootstrap configuration surface. Empty uses
	// auto classification; the CLI parser supplies the explicit five modes.
	Onboarding onboardinghandoff.Mode
	// Subcommand selects an alternate setup mode. Empty string runs the
	// default apply flow. Recognized values: "status", "explain".
	Subcommand string
	// ExplainName is the requirement name to look up when Subcommand is
	// "explain". Ignored otherwise.
	ExplainName string
}

type apiLaunchSpec struct {
	Command string
	Args    []string
	LogFile string
	Env     []string
	Port    int
}

type cliInstallManager interface {
	InstallScenarioCLI(name string) error
	EnsureScenarioCLI(name string) error
	InstallResourceCLI(name string) error
	InstallAllScenarioCLIs() error
	InstallEnabledResourceCLIs() error
}

type setupDeps struct {
	currentHost                 func() vrooliruntime.Host
	loadProject                 func(string) (scenario.Scenario, error)
	markComplete                func(string, string) error
	syncResourceSchema          func(string) error
	newCLIInstallManager        func(root, home string) (cliInstallManager, error)
	recordProjectInstall        func(root, home string) error
	resolveHostRequirements     func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error)
	inspectRequirements         func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error)
	ensureRequirements          func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error)
	ensureBootstrapTools        func(home string, opts vrooliruntime.EnsureOptions) error
	newPortsManager             func(root, home string) (*ports.Manager, error)
	startProjectAPI             func(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error
	startOrchestrator           func(root, home string, stdout, stderr io.Writer) error
	healthCheck                 func(port int, timeout time.Duration) error
	loadDotEnv                  func(path string) (map[string]string, error)
	now                         func() time.Time
	osExecutable                func() (string, error)
	onboardingPortCommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)
	openOnboardingURL           func(url string) error
	onboardingCLIExecutable     func() (string, error)
	runOnboardingCLI            func(executable, root string, stdin io.Reader, stdout, stderr io.Writer) error
	detectPresentation          func(context.Context) hostpresentation.Capability
	resourceController          func(root, home string) resourceRunner
	installPrivilegeBroker      func(context.Context, string) (privilegebroker.SetupStatus, error)
	inspectPrivilegeBroker      func() privilegebroker.SetupStatus
	configureCredentialBackend  func(io.Writer, io.Writer) error
	discoverCapabilities        func(context.Context, string, string) ([]operatorcapability.Status, error)
}

type setupService struct {
	deps setupDeps
}

func defaultSetupDeps(repoRoots ...string) setupDeps {
	repoRoot := ""
	if len(repoRoots) > 0 {
		repoRoot = repoRoots[0]
	}
	return setupDeps{
		currentHost:        vrooliruntime.Current,
		loadProject:        project.LoadProject,
		markComplete:       markComplete,
		syncResourceSchema: syncResourceSchemaArtifacts,
		newCLIInstallManager: func(root, home string) (cliInstallManager, error) {
			return cliinstall.NewManager(root, home)
		},
		recordProjectInstall:    cliinstall.RecordProjectSetup,
		resolveHostRequirements: hostreq.Resolve,
		inspectRequirements:     vrooliruntime.InspectRequirements,
		ensureRequirements:      vrooliruntime.EnsureRequirements,
		ensureBootstrapTools:    ensureBootstrapHostTools,
		newPortsManager: func(root, home string) (*ports.Manager, error) {
			return ports.NewManager(root, home)
		},
		startProjectAPI:   startProjectAPI,
		startOrchestrator: startOrchestrator,
		healthCheck:       waitForHTTPHealth,
		loadDotEnv:        loadDotEnv,
		now:               time.Now,
		osExecutable:      os.Executable,
		onboardingPortCommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return shell.NewCommandContext(ctx, name, args...).CombinedOutput()
		},
		openOnboardingURL: func(url string) error {
			return scenarioexec.OpenURL(shell.LookPath, scenarioexec.RunSubprocess, url)
		},
		onboardingCLIExecutable: func() (string, error) { return exec.LookPath("vrooli-onboarding") },
		runOnboardingCLI: func(executable, root string, stdin io.Reader, stdout, stderr io.Writer) error {
			return scenarioexec.RunSubprocess(scenarioexec.SubprocessSpec{Name: executable, Args: []string{"wizard", "run", "--interactive"}, Dir: root, Stdin: stdin, Stdout: stdout, Stderr: stderr})
		},
		detectPresentation: hostpresentation.Detect,
		resourceController: func(root, home string) resourceRunner {
			return resources.NewController(root, home)
		},
		installPrivilegeBroker: func(ctx context.Context, executable string) (privilegebroker.SetupStatus, error) {
			return privilegebroker.DefaultInstallerForRepo(executable, repoRoot).Install(ctx)
		},
		inspectPrivilegeBroker: privilegebroker.Inspect,
		configureCredentialBackend: func(stdout, stderr io.Writer) error {
			return configureCredentialBackend(stdout, stderr)
		},
		discoverCapabilities: capabilitycatalog.Discover,
	}
}

func newSetupService(deps setupDeps) *setupService {
	if deps.configureCredentialBackend == nil {
		deps.configureCredentialBackend = func(io.Writer, io.Writer) error { return nil }
	}
	if deps.discoverCapabilities == nil {
		deps.discoverCapabilities = func(context.Context, string, string) ([]operatorcapability.Status, error) { return nil, nil }
	}
	if deps.onboardingCLIExecutable == nil {
		deps.onboardingCLIExecutable = func() (string, error) { return exec.LookPath("vrooli-onboarding") }
	}
	if deps.runOnboardingCLI == nil {
		deps.runOnboardingCLI = func(executable, root string, stdin io.Reader, stdout, stderr io.Writer) error {
			return scenarioexec.RunSubprocess(scenarioexec.SubprocessSpec{Name: executable, Args: []string{"wizard", "run", "--interactive"}, Dir: root, Stdin: stdin, Stdout: stdout, Stderr: stderr})
		}
	}
	if deps.detectPresentation == nil {
		deps.detectPresentation = hostpresentation.Detect
	}
	return &setupService{deps: deps}
}

//nolint:gocyclo // setup orchestrates independent resource, capability, and phase outcomes.
func RunSetupWithOptions(root, home string, opts Options, stdout, stderr io.Writer) error {
	return newSetupService(defaultSetupDeps(root)).RunSetupWithOptions(root, home, opts, stdout, stderr)
}

func (s *setupService) RunSetupWithOptions(root, home string, opts Options, stdout, stderr io.Writer) (err error) {
	var terminalReport vrooliruntime.Report
	var terminalReportErr error
	var degradedResources []string
	var onboardingResult *OnboardingResult
	var readinessVerdict *SetupReadiness
	statePath := ""
	if locator, locatorErr := projectstate.NewLocator(home, root); locatorErr == nil {
		statePath = locator.ActiveSetupPath()
	}
	progress := newProgressCoordinator(progressWriter(stderr, stdout), progressOptions{DryRun: opts.DryRun, StatePath: statePath})
	defer func() {
		// Every terminal path carries the verdict, including a dry run and an
		// early return. Computing it only on the happy path would leave the
		// most common inspection without the one fact it exists to report.
		if readinessVerdict == nil && err == nil {
			verdict := verifySetupReadiness(root, terminalReport, terminalReportErr)
			readinessVerdict = &verdict
		}
		result := finalizeSetupResultConfiguration(setupTerminalResult(progress.CurrentPhase(), terminalReport, err, degradedResources), home, root, err, readinessVerdict)
		result.Onboarding = onboardingResult
		if writeErr := writeSetupResult(opts.ResultPath, result); writeErr != nil && err == nil {
			err = writeErr
		}
	}()
	switch opts.Subcommand {
	case "status":
		return s.runSetupStatus(root, home, opts, stdout)
	case "explain":
		return s.runSetupExplain(root, home, opts, stdout)
	}
	progress.Start()
	// Evaluate the named return value when setup exits. Passing err directly to
	// defer would capture its initial nil value, causing a failed setup (for
	// example, an incomplete ownership migration) to render as "Complete".
	defer func() { progress.Finish(err) }()
	flow := setupFlow{service: s, root: root, home: home, opts: opts, stdout: stdout, stderr: stderr, progress: progress}
	if err := flow.prepareProject(); err != nil {
		return err
	}
	resolved, err := flow.resolveRequirements()
	if err != nil {
		return err
	}
	report, stop, err := flow.applyRequirements(resolved)
	terminalReport = report
	resolved.report = report
	if err != nil || stop {
		return err
	}
	completion, err := flow.completeSetup(resolved)
	degradedResources = completion.degradedResources
	onboardingResult = completion.onboardingResult
	readinessVerdict = completion.readinessVerdict
	return err
}

type setupFlow struct {
	service  *setupService
	root     string
	home     string
	opts     Options
	stdout   io.Writer
	stderr   io.Writer
	progress *progressCoordinator
}

type resolvedSetup struct {
	requirements hostreq.Resolution
	executable   string
	ensure       vrooliruntime.EnsureOptions
	report       vrooliruntime.Report
}

type setupCompletion struct {
	degradedResources []string
	onboardingResult  *OnboardingResult
	readinessVerdict  *SetupReadiness
}

func (f setupFlow) prepareProject() error {
	f.progress.StartPhase(PhaseValidation)
	if err := f.service.deps.currentHost().ValidateSetup(); err != nil {
		return err
	}
	f.progress.CompletePhase()
	f.progress.StartPhase(PhaseProject)
	if _, err := f.service.deps.loadProject(f.root); err != nil {
		return err
	}
	f.progress.CompletePhase()
	if f.opts.DryRun {
		return nil
	}
	f.progress.StartPhase(PhaseFilesystem)
	if err := ensureProjectFilesystemWithRecovery(f.root, f.home); err != nil {
		return err
	}
	locator, err := projectstate.NewLocator(f.home, f.root)
	if err != nil {
		return err
	}
	if err := runOwnershipMigration(locator, f.stdout, f.stderr); err != nil {
		return err
	}
	f.progress.CompletePhase()
	return nil
}

func (f setupFlow) resolveRequirements() (resolvedSetup, error) {
	f.progress.StartPhase(PhaseResolution)
	requirements, err := f.service.deps.resolveHostRequirements(f.root, f.home, hostreq.ResolveOptions{Environment: f.opts.Environment, When: "setup", Resources: f.opts.Resources, Scenarios: f.opts.Scenarios, Platform: hostreq.CurrentPlatform()})
	if err != nil {
		return resolvedSetup{}, err
	}
	f.progress.CompletePhase()
	requirements = bootstrapAwareRequirements(requirements)
	executable, err := f.service.deps.osExecutable()
	if err != nil {
		return resolvedSetup{}, fmt.Errorf("resolve executable for onboarding apply grant: %w", err)
	}
	requirements = addOnboardingApplyPrivilegeRequirement(requirements, executable)
	return resolvedSetup{requirements: requirements, executable: executable, ensure: vrooliruntime.EnsureOptions{Environment: f.opts.Environment, SudoMode: f.opts.SudoMode, DryRun: f.opts.DryRun, AutoInstall: true, IncludeOptional: f.opts.IncludeOptional, MaintenanceWindow: f.opts.MaintenanceWindow, Stdout: f.stdout, Stderr: f.stderr, OnOperation: f.progress.Operation}}, nil
}

func (f setupFlow) applyRequirements(resolved resolvedSetup) (vrooliruntime.Report, bool, error) {
	if !f.opts.DryRun {
		f.progress.StartPhase(PhaseBootstrap)
		f.progress.Operation("Checking bootstrap tools (git, go)")
		_, _ = fmt.Fprintln(f.stdout, "[INFO]    Checking bootstrap tools (git, go)...")
		if err := f.service.deps.ensureBootstrapTools(f.home, resolved.ensure); err != nil {
			return vrooliruntime.Report{}, false, err
		}
		f.progress.CompletePhase()
	}
	f.progress.StartPhase(PhaseRequirements)
	f.progress.Operation("Applying selected host requirements")
	_, _ = fmt.Fprintln(f.stdout, "[INFO]    Applying selected host requirements...")
	report, err := f.service.deps.ensureRequirements(resolved.ensure, resolved.requirements)
	renderSetupRequirementResult(f.stdout, f.opts, report)
	if err != nil && !f.opts.DryRun {
		return report, false, err
	}
	f.progress.CompletePhase()
	if f.opts.DryRun {
		_, _ = fmt.Fprintln(f.stdout, "[INFO]    Dry-run mode skips git configuration, resource installation, and setup completion markers")
		return report, true, nil
	}
	return report, false, nil
}

func (f setupFlow) completeSetup(resolved resolvedSetup) (setupCompletion, error) {
	var completion setupCompletion
	f.progress.StartPhase(PhaseGeneratedPackages)
	f.progress.Operation("Generating repository packages")
	_, _ = fmt.Fprintln(f.stdout, "[INFO]    Generating repository packages needed by the control plane...")
	if err := lifecycle.ProvisionGeneratedPackages(f.root, f.home, f.stdout, f.stdout); err != nil {
		return completion, fmt.Errorf("provision generated packages: %w", err)
	}
	f.progress.CompletePhase()
	if f.opts.BootstrapOnly {
		_, _ = fmt.Fprintln(f.stdout, "[INFO]    Bootstrap-only setup applied host requirements; native CLI finalization is still required")
		return completion, nil
	}
	if err := f.configureCredentialsAndBroker(resolved.executable); err != nil {
		return completion, err
	}
	degraded, err := f.reconcileResourcesAndCLI()
	if err != nil {
		return completion, err
	}
	completion.degradedResources = degraded
	onboarding, err := f.finalizeInstall()
	if err != nil {
		return completion, err
	}
	completion.onboardingResult = onboarding
	verdict := verifySetupReadiness(f.root, resolved.report, nil)
	completion.readinessVerdict = &verdict
	f.progress.CompletePhase()
	f.progress.StartPhase(PhaseCompletion)
	if len(degraded) > 0 {
		_, _ = fmt.Fprintf(f.stdout, "[WARN]    Setup completed with degraded optional resources: %s\n", strings.Join(degraded, ", "))
	}
	renderSetupReadinessVerdict(f.stdout, verdict, configurationAlreadyComplete(f.home, f.root))
	f.progress.CompletePhase()
	return completion, nil
}

func (f setupFlow) configureCredentialsAndBroker(executable string) error {
	f.progress.StartPhase(PhaseCredentials)
	f.progress.Operation("Configuring the credential backend")
	_, _ = fmt.Fprintln(f.stdout, "[INFO]    Configuring the credential backend...")
	if f.opts.CredentialPassphraseStdin {
		passphrase, err := readCredentialPassphraseStdin()
		if err != nil {
			return err
		}
		if err := configureCredentialBackendWithPassphrase(f.stdout, f.stderr, passphrase); err != nil {
			return err
		}
	} else if err := f.service.deps.configureCredentialBackend(f.stdout, f.stderr); err != nil {
		return err
	}
	f.progress.CompletePhase()
	f.progress.StartPhase(PhaseCredentialCapabilities)
	f.progress.Operation("Discovering operator capabilities")
	if err := discoverAndQueueCapabilities(context.Background(), f.service.deps.discoverCapabilities, f.root, f.home, f.stdout); err != nil {
		return err
	}
	f.progress.CompletePhase()
	f.progress.StartPhase(PhasePrivilegeBroker)
	f.progress.Operation("Installing the privilege broker")
	_, _ = fmt.Fprintln(f.stdout, "[INFO]    Installing the privilege broker...")
	status, err := f.service.deps.installPrivilegeBroker(context.Background(), executable)
	if err != nil {
		return err
	}
	renderPrivilegeBrokerStatus(f.stdout, status)
	f.progress.CompletePhase()
	return nil
}

func (f setupFlow) reconcileResourcesAndCLI() ([]string, error) {
	f.progress.StartPhase(PhaseGit)
	f.progress.Operation("Configuring Git defaults")
	_, _ = fmt.Fprintln(f.stdout, "[INFO]    Configuring Git defaults...")
	if err := configureGit(f.root); err != nil {
		return nil, err
	}
	f.progress.CompletePhase()
	f.progress.StartPhase(PhaseResources)
	f.progress.Operation("Reconciling selected resources")
	_, _ = fmt.Fprintln(f.stdout, "[INFO]    Reconciling selected resources...")
	degraded, err := f.service.maybeInstallResources(f.root, f.home, f.opts, f.stdout, f.stderr, f.progress.Operation)
	if err != nil {
		return nil, err
	}
	f.progress.CompletePhase()
	f.progress.StartPhase(PhaseCLI)
	if strings.TrimSpace(f.opts.Resources) == setupNone {
		f.progress.Operation("Skipping resource CLI synchronization")
		_, _ = fmt.Fprintln(f.stdout, "[INFO]    Skipping resource CLI schema synchronization (resources=none)")
	} else {
		_, _ = fmt.Fprintln(f.stdout, "[INFO]    Synchronizing resource CLI schemas...")
		if err := f.service.deps.syncResourceSchema(f.root); err != nil {
			return nil, err
		}
	}
	f.progress.CompletePhase()
	return degraded, nil
}

func (f setupFlow) finalizeInstall() (*OnboardingResult, error) {
	f.progress.StartPhase(PhaseFinalize)
	f.progress.Operation("Refreshing selected scenario and resource CLIs")
	_, _ = fmt.Fprintln(f.stdout, "[INFO]    Refreshing the bootstrap and selected scenario CLIs...")
	manager, err := f.service.deps.newCLIInstallManager(f.root, f.home)
	if err != nil {
		return nil, err
	}
	if err := manager.EnsureScenarioCLI("secrets-manager"); err != nil {
		return nil, fmt.Errorf("refresh secrets-manager bootstrap CLI: %w", err)
	}
	if err := installSelectedCLIs(manager, f.opts.Resources, f.opts.Scenarios, f.progress.Operation); err != nil {
		return nil, err
	}
	if err := f.service.deps.recordProjectInstall(f.root, f.home); err != nil {
		return nil, fmt.Errorf("record project install inventory: %w", err)
	}
	if err := f.service.deps.markComplete(f.home, f.root); err != nil {
		return nil, err
	}
	result, handoffErr := f.service.runOnboardingHandoff(f.root, f.home, f.opts, f.stdout, f.stderr)
	if handoffErr != nil {
		_, _ = fmt.Fprintf(f.stderr, "[WARN]    Onboarding handoff unavailable: %v\n", handoffErr)
	}
	return result, nil
}

func renderSetupReadinessVerdict(stdout io.Writer, verdict SetupReadiness, markerPresent bool) {
	switch {
	case verdict.Source == ReadinessSourceUnavailable:
		_, _ = fmt.Fprintf(stdout, "[WARN]    Setup finished; configuration could not be verified: %s\n", verdict.Reason)
	case !markerPresent:
		// Configuration is pending by definition here, so the readiness
		// remediation must not be printed: its ready/degraded wording asserts
		// that configuration is complete, which directly contradicts the line
		// above and is how a run ended with "remains pending" immediately
		// followed by "configuration are complete".
		_, _ = fmt.Fprintln(stdout, "[INFO]    Bootstrap setup completed; configuration remains pending until onboarding reports completion.")
		if len(verdict.Blockers) > 0 {
			_, _ = fmt.Fprintf(stdout, "[ACTION]  Unresolved: %s. Finish configuration in onboarding: `vrooli-onboarding wizard run --interactive`\n", strings.Join(verdict.Blockers, ", "))
			return
		}
		_, _ = fmt.Fprintln(stdout, "[ACTION]  Finish configuration in onboarding: `vrooli-onboarding wizard run --interactive`")
		return
	case verdict.Status == ReadinessStatusReady:
		_, _ = fmt.Fprintln(stdout, "[INFO]    Setup completed; configuration verified ready.")
	case verdict.Status == ReadinessStatusDegraded:
		_, _ = fmt.Fprintln(stdout, "[INFO]    Setup completed; configuration verified with optional items unresolved.")
	default:
		_, _ = fmt.Fprintf(stdout, "[WARN]    Setup completed, but configuration is not verified: %s\n", strings.Join(verdict.Blockers, ", "))
	}
	if verdict.Status != ReadinessStatusReady {
		_, _ = fmt.Fprintf(stdout, "[ACTION]  %s\n", readinessRemediation(verdict))
	}
}

// finalizeSetupResultConfiguration decides setup's last word.
//
// A present marker is no longer sufficient on its own. The marker is written by
// onboarding, and trusting it alone is how a run reported success over a host
// with a required credential absent and a required safeguard unapplied. When
// the verdict contradicts the marker, the honest answer is that configuration
// is pending, and the remediation names the one command that resolves it.
func finalizeSetupResultConfiguration(result SetupResult, home, root string, runErr error, verdict *SetupReadiness) SetupResult {
	if runErr != nil {
		return result
	}
	result.Readiness = verdict
	if configurationAlreadyComplete(home, root) {
		result.ConfigurationPending = false
		if result.Status != SetupStatusSuccess {
			return result
		}
		if verdict == nil {
			result.Category = SetupCategorySuccess
			result.Remediation = "Setup and onboarding configuration are complete."
			return result
		}
		if verdict.Status == ReadinessStatusMissing || verdict.Status == ReadinessStatusUnsupported {
			result.Category = SetupCategoryConfigurationPending
			result.ConfigurationPending = true
			result.Remediation = readinessRemediation(*verdict)
			return result
		}
		result.Category = SetupCategorySuccess
		result.Remediation = readinessRemediation(*verdict)
		return result
	}
	if result.Status == SetupStatusSuccess || result.Status == SetupStatusDegraded {
		result.ConfigurationPending = true
		if result.Status == SetupStatusSuccess {
			result.Category = SetupCategoryConfigurationPending
			result.Remediation = "Bootstrap completed. Continue in vrooli-onboarding to finish configuration."
			if verdict != nil && len(verdict.Blockers) > 0 {
				result.Remediation = readinessRemediation(*verdict)
			}
		}
	}
	return result
}

const (
	ownershipMigrationVersion             = 2
	ownershipMigrationBatchEntries uint64 = 1_000_000
)

var ownershipMigrationClasses = []string{
	repocontract.HomeKeyBin,
	repocontract.HomeKeyCache,
	repocontract.HomeKeyLogs,
	repocontract.HomeKeyMetrics,
	repocontract.HomeKeyProcesses,
	repocontract.HomeKeyBuild,
	repocontract.HomeKeyTestRuns,
	repocontract.HomeKeySecretsEnc,
	// The state class holds the operator-input queue and the scenario state a
	// scenario process reads and writes as the operator. An elevated setup run
	// that writes there leaves a root-owned file the onboarding API cannot
	// open, and the operator meets it as an opaque server error inside the
	// flow. Migrating it is what keeps one elevated run from making the
	// in-flow surface unusable.
	repocontract.HomeKeyState,
	"backups",
	"artifacts",
}

func installSelectedCLIs(manager cliInstallManager, resourceSelector, scenarioSelector string, onOperation ...func(string)) error {
	operation := func(label string) {}
	if len(onOperation) > 0 && onOperation[0] != nil {
		operation = onOperation[0]
	}
	resources := strings.TrimSpace(resourceSelector)
	switch resources {
	case setupNone:
	case "", "enabled":
		operation("Refreshing enabled resource CLIs")
		if err := manager.InstallEnabledResourceCLIs(); err != nil {
			return err
		}
	default:
		for _, name := range splitSelection(resources) {
			operation("Refreshing resource CLI " + name)
			if err := manager.InstallResourceCLI(name); err != nil {
				return err
			}
		}
	}

	scenarios := strings.TrimSpace(scenarioSelector)
	switch scenarios {
	case "", setupNone:
	case "all":
		operation("Refreshing all scenario CLIs")
		if err := manager.InstallAllScenarioCLIs(); err != nil {
			return err
		}
	default:
		for _, name := range splitSelection(scenarios) {
			operation("Refreshing scenario CLI " + name)
			if err := manager.InstallScenarioCLI(name); err != nil {
				return err
			}
		}
	}
	return nil
}

func splitSelection(value string) []string {
	items := make([]string, 0)
	for _, raw := range strings.Split(value, ",") {
		if name := strings.TrimSpace(raw); name != "" {
			items = append(items, name)
		}
	}
	return items
}

// bootstrapAwareRequirements makes the bootstrap contract explicit without
// coupling it to the root manifest's environment filters. git and Go are always
// required, ordered first, and installed by ensureBootstrapHostTools before any
// other requirement can invoke them. When present, rasdaemon is ordered before
// mcelog because mcelog's modern-distribution fallback can only recognize that
// it is superseded after rasdaemon is active. Docker is deliberately removed
// from the global set; selected container resources demand it later via
// preflightDockerResources.
func setupNeeded(home, root, slug string) bool {
	if forceSetupApplies(slug) {
		return true
	}
	locator, err := projectstate.NewLocator(home, root)
	if err != nil {
		return true
	}
	return !locator.HasBootstrapComplete()
}
