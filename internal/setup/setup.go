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
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
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

//nolint:gocyclo // setup orchestrates independent resource, capability, and phase outcomes.
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
	progress.StartPhase(PhaseValidation)

	if err := s.deps.currentHost().ValidateSetup(); err != nil {
		return err
	}
	progress.CompletePhase()

	progress.StartPhase(PhaseProject)
	if _, err := s.deps.loadProject(root); err != nil {
		return err
	}
	progress.CompletePhase()

	if !opts.DryRun {
		progress.StartPhase(PhaseFilesystem)
		if err := ensureProjectFilesystemWithRecovery(root, home); err != nil {
			return err
		}
		if locator, locatorErr := projectstate.NewLocator(home, root); locatorErr != nil {
			return locatorErr
		} else if err := runOwnershipMigration(locator, stdout, stderr); err != nil {
			return err
		}
		progress.CompletePhase()
	}
	progress.StartPhase(PhaseResolution)
	requirements, err := s.deps.resolveHostRequirements(root, home, hostreq.ResolveOptions{
		Environment: opts.Environment,
		When:        "setup",
		Resources:   opts.Resources,
		Scenarios:   opts.Scenarios,
		Platform:    hostreq.CurrentPlatform(),
	})
	if err != nil {
		return err
	}
	progress.CompletePhase()
	requirements = bootstrapAwareRequirements(requirements)
	// The onboarding apply API runs later and cannot safely open an interactive
	// sudo prompt. During this setup pass, provision one literal grant for the
	// elevated host items selected by the same resolution.
	executable, executableErr := s.deps.osExecutable()
	if executableErr != nil {
		return fmt.Errorf("resolve executable for onboarding apply grant: %w", executableErr)
	}
	requirements = addOnboardingApplyPrivilegeRequirement(requirements, executable)
	ensureOptions := vrooliruntime.EnsureOptions{
		Environment:       opts.Environment,
		SudoMode:          opts.SudoMode,
		DryRun:            opts.DryRun,
		AutoInstall:       true,
		IncludeOptional:   opts.IncludeOptional,
		MaintenanceWindow: opts.MaintenanceWindow,
		Stdout:            stdout,
		Stderr:            stderr,
		OnOperation:       progress.Operation,
	}
	if !opts.DryRun {
		progress.StartPhase(PhaseBootstrap)
		progress.Operation("Checking bootstrap tools (git, go)")
		_, _ = fmt.Fprintln(stdout, "[INFO]    Checking bootstrap tools (git, go)...")
		if err := s.deps.ensureBootstrapTools(home, ensureOptions); err != nil {
			return err
		}
		progress.CompletePhase()
	}

	progress.StartPhase(PhaseRequirements)
	progress.Operation("Applying selected host requirements")
	_, _ = fmt.Fprintln(stdout, "[INFO]    Applying selected host requirements...")
	report, ensureErr := s.deps.ensureRequirements(ensureOptions, requirements)
	terminalReport = report
	renderSetupRequirementResult(stdout, opts, report)
	if ensureErr != nil && !opts.DryRun {
		return ensureErr
	}
	progress.CompletePhase()
	if opts.DryRun {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Dry-run mode skips git configuration, resource installation, and setup completion markers")
		return nil
	}
	progress.StartPhase(PhaseGeneratedPackages)
	progress.Operation("Generating repository packages")
	_, _ = fmt.Fprintln(stdout, "[INFO]    Generating repository packages needed by the control plane...")
	if err := lifecycle.ProvisionGeneratedPackages(root, home, stdout, stdout); err != nil {
		return fmt.Errorf("provision generated packages: %w", err)
	}
	progress.CompletePhase()
	if opts.BootstrapOnly {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Bootstrap-only setup applied host requirements; native CLI finalization is still required")
		return nil
	}
	progress.StartPhase(PhaseCredentials)
	progress.Operation("Configuring the credential backend")
	_, _ = fmt.Fprintln(stdout, "[INFO]    Configuring the credential backend...")
	if opts.CredentialPassphraseStdin {
		passphrase, readErr := readCredentialPassphraseStdin()
		if readErr != nil {
			return readErr
		}
		if err := configureCredentialBackendWithPassphrase(stdout, stderr, passphrase); err != nil {
			return err
		}
	} else if err := s.deps.configureCredentialBackend(stdout, stderr); err != nil {
		return err
	}
	progress.CompletePhase()
	progress.StartPhase(PhaseCredentialCapabilities)
	progress.Operation("Discovering operator capabilities")
	if err := discoverAndQueueCapabilities(context.Background(), s.deps.discoverCapabilities, root, home, stdout); err != nil {
		return err
	}
	progress.CompletePhase()
	progress.StartPhase(PhasePrivilegeBroker)
	progress.Operation("Installing the privilege broker")
	_, _ = fmt.Fprintln(stdout, "[INFO]    Installing the privilege broker...")
	if executableErr != nil {
		return fmt.Errorf("resolve executable for privilege broker: %w", executableErr)
	}
	brokerStatus, brokerErr := s.deps.installPrivilegeBroker(context.Background(), executable)
	if brokerErr != nil {
		return brokerErr
	}
	renderPrivilegeBrokerStatus(stdout, brokerStatus)
	progress.CompletePhase()
	progress.StartPhase(PhaseGit)
	progress.Operation("Configuring Git defaults")
	_, _ = fmt.Fprintln(stdout, "[INFO]    Configuring Git defaults...")
	if err := configureGit(root); err != nil {
		return err
	}
	progress.CompletePhase()
	progress.StartPhase(PhaseResources)
	progress.Operation("Reconciling selected resources")
	_, _ = fmt.Fprintln(stdout, "[INFO]    Reconciling selected resources...")
	var resourceErr error
	degradedResources, resourceErr = s.maybeInstallResources(root, home, opts, stdout, stderr, progress.Operation)
	if resourceErr != nil {
		return resourceErr
	}
	progress.CompletePhase()
	progress.StartPhase(PhaseCLI)
	if strings.TrimSpace(opts.Resources) == setupNone {
		progress.Operation("Skipping resource CLI synchronization")
		_, _ = fmt.Fprintln(stdout, "[INFO]    Skipping resource CLI schema synchronization (resources=none)")
	} else {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Synchronizing resource CLI schemas...")
		if err := s.deps.syncResourceSchema(root); err != nil {
			return err
		}
	}
	progress.CompletePhase()
	progress.StartPhase(PhaseFinalize)
	progress.Operation("Refreshing selected scenario and resource CLIs")
	_, _ = fmt.Fprintln(stdout, "[INFO]    Refreshing the bootstrap and selected scenario CLIs...")
	cliManager, err := s.deps.newCLIInstallManager(root, home)
	if err != nil {
		return err
	}
	// secrets-manager is a bootstrap control-plane CLI, not an optional
	// scenario capability: recovery migration and credential diagnostics use
	// it even when the operator selected no scenario CLIs. Always freshness-
	// check it so an installed binary cannot keep an older authority policy.
	if err := cliManager.EnsureScenarioCLI("secrets-manager"); err != nil {
		return fmt.Errorf("refresh secrets-manager bootstrap CLI: %w", err)
	}
	if err := installSelectedCLIs(cliManager, opts.Resources, opts.Scenarios, progress.Operation); err != nil {
		return err
	}
	if err := s.deps.recordProjectInstall(root, home); err != nil {
		return fmt.Errorf("record project install inventory: %w", err)
	}
	if err := s.deps.markComplete(home, root); err != nil {
		return err
	}
	var handoffErr error
	onboardingResult, handoffErr = s.runOnboardingHandoff(root, home, opts, stdout, stderr)
	if handoffErr != nil {
		_, _ = fmt.Fprintf(stderr, "[WARN]    Onboarding handoff unavailable: %v\n", handoffErr)
	}
	// The verdict is computed after the handoff, so a wizard the operator just
	// finished is reflected in setup's last word rather than in the next run's.
	verdict := verifySetupReadiness(root, terminalReport, terminalReportErr)
	readinessVerdict = &verdict
	progress.CompletePhase()
	progress.StartPhase(PhaseCompletion)
	if len(degradedResources) > 0 {
		_, _ = fmt.Fprintf(stdout, "[WARN]    Setup completed with degraded optional resources: %s\n", strings.Join(degradedResources, ", "))
	}
	// The last line of a setup run states a verified verdict rather than an
	// unconditional success. The previous unconditional line was true about the
	// bootstrap steps and silent about whether the host was actually configured.
	renderSetupReadinessVerdict(stdout, verdict, configurationAlreadyComplete(home, root))
	progress.CompletePhase()
	return nil
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
