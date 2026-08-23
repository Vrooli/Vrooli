package setup

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/capabilitycatalog"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/dockerhost"
	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/ports"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/projectstate"
	"github.com/vrooli/vrooli/internal/resources"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	onboardingapplyprivileges "github.com/vrooli/vrooli/internal/safeguards/onboarding-apply-privileges"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	"github.com/vrooli/vrooli/internal/shell"
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

var launchOnboardingAsOperatorFn = launchOnboardingAsOperator

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
	resourceController          func(root, home string) resourceRunner
	installPrivilegeBroker      func(context.Context, string) (privilegebroker.SetupStatus, error)
	inspectPrivilegeBroker      func() privilegebroker.SetupStatus
	configureCredentialBackend  func(io.Writer, io.Writer) error
	discoverCapabilities        func(context.Context, string, string) ([]operatorcapability.Status, error)
}

type setupService struct {
	deps setupDeps
}

func defaultSetupDeps() setupDeps {
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
		resourceController: func(root, home string) resourceRunner {
			return resources.NewController(root, home)
		},
		installPrivilegeBroker: func(ctx context.Context, executable string) (privilegebroker.SetupStatus, error) {
			return privilegebroker.DefaultInstaller(executable).Install(ctx)
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
	return &setupService{deps: deps}
}

func RunSetupWithOptions(root, home string, opts Options, stdout, stderr io.Writer) error {
	return newSetupService(defaultSetupDeps()).RunSetupWithOptions(root, home, opts, stdout, stderr)
}

func (s *setupService) RunSetupWithOptions(root, home string, opts Options, stdout, stderr io.Writer) (err error) {
	stage := "validation"
	var terminalReport vrooliruntime.Report
	defer func() {
		if writeErr := writeSetupResult(opts.ResultPath, setupTerminalResult(stage, terminalReport, err)); writeErr != nil && err == nil {
			err = writeErr
		}
	}()
	switch opts.Subcommand {
	case "status":
		return s.runSetupStatus(root, home, opts, stdout)
	case "explain":
		return s.runSetupExplain(root, home, opts, stdout)
	}

	if err := s.deps.currentHost().ValidateSetup(); err != nil {
		return err
	}

	stage = "project"
	if _, err := s.deps.loadProject(root); err != nil {
		return err
	}

	if !opts.DryRun {
		stage = "filesystem"
		if err := ensureProjectFilesystem(root, home); err != nil {
			return err
		}
		// Heal any root-owned strays a prior sudo'd run left in the operator's
		// home. No-op unless this invocation is itself root-via-sudo.
		if reowned, rerr := config.ReconcileVrooliOwnership(); rerr != nil {
			fmt.Fprintf(stderr, "warning: could not reconcile ~/.vrooli ownership: %v\n", rerr)
		} else if reowned > 0 {
			fmt.Fprintf(stdout, "Reclaimed ownership of %d root-owned entries under ~/.vrooli.\n", reowned)
		}
	}
	stage = "resolution"
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
	}
	if !opts.DryRun {
		stage = "bootstrap"
		_, _ = fmt.Fprintln(stdout, "[INFO]    Checking bootstrap tools (git, go)...")
		if err := s.deps.ensureBootstrapTools(home, ensureOptions); err != nil {
			return err
		}
	}

	stage = "requirements"
	_, _ = fmt.Fprintln(stdout, "[INFO]    Applying selected host requirements...")
	report, ensureErr := s.deps.ensureRequirements(ensureOptions, requirements)
	terminalReport = report
	renderSetupRequirementResult(stdout, opts, report)
	if ensureErr != nil && !opts.DryRun {
		return ensureErr
	}
	if opts.DryRun {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Dry-run mode skips git configuration, resource installation, and setup completion markers")
		return nil
	}
	stage = "generated-packages"
	_, _ = fmt.Fprintln(stdout, "[INFO]    Generating repository packages needed by the control plane...")
	if err := lifecycle.ProvisionGeneratedPackages(root, home, stdout, stdout); err != nil {
		return fmt.Errorf("provision generated packages: %w", err)
	}
	if opts.BootstrapOnly {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Bootstrap-only setup applied host requirements; native CLI finalization is still required")
		return nil
	}
	stage = "credentials"
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
	stage = "credential-capabilities"
	if err := discoverAndQueueCapabilities(context.Background(), s.deps.discoverCapabilities, root, home, stdout); err != nil {
		return err
	}
	stage = "privilege-broker"
	_, _ = fmt.Fprintln(stdout, "[INFO]    Installing the privilege broker...")
	if executableErr != nil {
		return fmt.Errorf("resolve executable for privilege broker: %w", executableErr)
	}
	brokerStatus, brokerErr := s.deps.installPrivilegeBroker(context.Background(), executable)
	if brokerErr != nil {
		return brokerErr
	}
	renderPrivilegeBrokerStatus(stdout, brokerStatus)
	stage = "git"
	_, _ = fmt.Fprintln(stdout, "[INFO]    Configuring Git defaults...")
	if err := configureGit(root); err != nil {
		return err
	}
	stage = "resources"
	_, _ = fmt.Fprintln(stdout, "[INFO]    Reconciling selected resources...")
	if err := s.maybeInstallResources(root, home, opts, stdout, stderr); err != nil {
		return err
	}
	stage = "cli"
	if strings.TrimSpace(opts.Resources) == "none" {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Skipping resource CLI schema synchronization (resources=none)")
	} else {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Synchronizing resource CLI schemas...")
		if err := s.deps.syncResourceSchema(root); err != nil {
			return err
		}
	}
	stage = "finalize"
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
	if err := installSelectedCLIs(cliManager, opts.Resources, opts.Scenarios); err != nil {
		return err
	}
	if err := s.deps.recordProjectInstall(root, home); err != nil {
		return fmt.Errorf("record project install inventory: %w", err)
	}
	if err := s.deps.markComplete(home, root); err != nil {
		return err
	}
	if err := s.maybeOpenOnboarding(root, home, stdout, stderr); err != nil {
		_, _ = fmt.Fprintf(stderr, "[WARN]    Unable to auto-open onboarding: %v\n", err)
		_, _ = fmt.Fprintln(stdout, "[ACTION]  Continue configuration with: vrooli scenario open vrooli-onboarding")
		_, _ = fmt.Fprintln(stdout, "[ACTION]  Or use the onboarding CLI/API to resolve pending operator inputs.")
	}
	_, _ = fmt.Fprintln(stdout, "[INFO]    Setup completed successfully.")
	_, _ = fmt.Fprintln(stdout, "[INFO]    Bootstrap setup completed; configuration remains pending until onboarding reports completion.")
	return nil
}

// installSelectedCLIs keeps CLI installation aligned with the same selectors
// used for host requirements and resource installation. In particular, an
// explicit "none" must not build every CLI in the repository: a minimal fresh
// host should only pay for the capabilities the operator selected.
func installSelectedCLIs(manager cliInstallManager, resourceSelector, scenarioSelector string) error {
	resources := strings.TrimSpace(resourceSelector)
	switch resources {
	case "none":
	case "", "enabled":
		if err := manager.InstallEnabledResourceCLIs(); err != nil {
			return err
		}
	default:
		for _, name := range splitSelection(resources) {
			if err := manager.InstallResourceCLI(name); err != nil {
				return err
			}
		}
	}

	scenarios := strings.TrimSpace(scenarioSelector)
	switch scenarios {
	case "", "none":
	case "all":
		if err := manager.InstallAllScenarioCLIs(); err != nil {
			return err
		}
	default:
		for _, name := range splitSelection(scenarios) {
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
func bootstrapAwareRequirements(resolution hostreq.Resolution) hostreq.Resolution {
	byName := make(map[string]hostreq.ResolvedRequirement, len(resolution.Tools)+2)
	for _, requirement := range resolution.Tools {
		name := strings.ToLower(strings.TrimSpace(requirement.Name))
		if name == "" || name == "docker" {
			continue
		}
		byName[name] = requirement
	}
	for _, name := range []string{"git", "go"} {
		if _, ok := byName[name]; ok {
			requirement := byName[name]
			requirement.Required = true
			requirement.Environments = []string{"development", "production", "minimal"}
			byName[name] = requirement
			continue
		}
		byName[name] = hostreq.ResolvedRequirement{
			Name:         name,
			Kind:         hostreq.KindTool,
			Required:     true,
			Reasons:      []string{"Bootstrap source operations and subsequent source rebuilds"},
			When:         []string{"setup"},
			Environments: []string{"development", "production", "minimal"},
			Provenance: []hostreq.Provenance{{
				Kind: "root", Name: "vrooli-bootstrap", Path: "internal/setup/setup.go", Source: "internal/setup/setup.go",
			}},
		}
	}
	ordered := make([]hostreq.ResolvedRequirement, 0, len(byName))
	for _, name := range []string{"git", "go"} {
		ordered = append(ordered, byName[name])
		delete(byName, name)
	}
	if requirement, ok := byName["rasdaemon"]; ok {
		ordered = append(ordered, requirement)
		delete(byName, "rasdaemon")
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		ordered = append(ordered, byName[name])
	}
	resolution.Tools = ordered
	return resolution
}

func addOnboardingApplyPrivilegeRequirement(resolution hostreq.Resolution, executable string) hostreq.Resolution {
	tools := make([]string, 0)
	for _, requirement := range resolution.Tools {
		if requirement.Privilege == hostreqspec.PrivilegeElevated {
			tools = append(tools, requirement.Name)
		}
	}
	safeguards := make([]string, 0)
	for _, requirement := range resolution.Safeguards {
		if requirement.Privilege == hostreqspec.PrivilegeElevated {
			safeguards = append(safeguards, requirement.Name)
		}
	}
	if len(tools) == 0 && len(safeguards) == 0 {
		return resolution
	}
	grant := hostreq.ResolvedRequirement{
		Name:       "onboarding_apply_privileges",
		Kind:       hostreq.KindSafeguard,
		Required:   true,
		Privilege:  hostreqspec.PrivilegeElevated,
		Platforms:  []string{"linux", "macos"},
		Config:     onboardingapplyprivileges.ConfigForRequirements(executable, tools, safeguards),
		Reasons:    []string{"Allow onboarding apply to execute selected elevated host requirements without a second prompt"},
		Provenance: []hostreq.Provenance{{Kind: "root", Name: "vrooli-setup", Path: "internal/setup/setup.go", Source: "internal/setup/setup.go"}},
	}
	resolution.Safeguards = append([]hostreq.ResolvedRequirement{grant}, resolution.Safeguards...)
	return resolution
}

func ensureBootstrapHostTools(home string, opts vrooliruntime.EnsureOptions) error {
	host := vrooliruntime.Current()
	if err := ensureBootstrapPackageManager(host, home, opts, exec.LookPath, shell.Run); err != nil {
		return err
	}
	for _, name := range []string{"git", "go"} {
		status, err := vrooliruntime.EnsureTool(name, opts)
		if err != nil {
			return fmt.Errorf("install bootstrap host tool %s: %w", name, err)
		}
		if !bootstrapToolSatisfied(status) {
			detail := strings.TrimSpace(strings.Join(status.Notes, "; "))
			if detail == "" {
				detail = string(status.ExecutionState)
			}
			return fmt.Errorf("bootstrap host tool %s is unavailable: %s", name, detail)
		}
		recoverHostToolPATH(home)
	}
	return nil
}

const homebrewInstallURL = "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh"

// ensureBootstrapPackageManager is the Homebrew-specific bootstrap for macOS's
// only remaining chicken-and-egg:
// a fresh host has curl but no package manager capable of installing Go. Linux
// distributions already ship their package manager, and Windows needs no
// bootstrap because winget ships with supported Windows installations.
func ensureBootstrapPackageManager(
	host vrooliruntime.Host,
	home string,
	opts vrooliruntime.EnsureOptions,
	lookPath func(string) (string, error),
	run func(shell.Spec) error,
) error {
	if host.OS != "darwin" || strings.TrimSpace(host.PackageManager) != "" {
		return nil
	}
	if _, err := lookPath("curl"); err != nil {
		return fmt.Errorf("bootstrap Homebrew: curl is required")
	}
	script, err := os.CreateTemp("", "vrooli-homebrew-install-*.sh")
	if err != nil {
		return fmt.Errorf("bootstrap Homebrew: create installer staging file: %w", err)
	}
	scriptPath := script.Name()
	if err := script.Close(); err != nil {
		_ = os.Remove(scriptPath)
		return fmt.Errorf("bootstrap Homebrew: close installer staging file: %w", err)
	}
	defer os.Remove(scriptPath)
	if err := run(shell.Spec{
		Name: "curl", Args: []string{"-fsSL", "--proto", "=https", "--tlsv1.2", "-o", scriptPath, homebrewInstallURL},
		Stdout: opts.Stdout, Stderr: opts.Stderr,
	}); err != nil {
		return fmt.Errorf("bootstrap Homebrew: download official installer: %w", err)
	}
	env := append(os.Environ(), "NONINTERACTIVE=1", "HOME="+home)
	if err := run(shell.Spec{
		Name: "/bin/bash", Args: []string{scriptPath}, Env: env,
		Stdout: opts.Stdout, Stderr: opts.Stderr, Stdin: os.Stdin,
	}); err != nil {
		return fmt.Errorf("bootstrap Homebrew: run official installer: %w", err)
	}
	recoverHostToolPATH(home)
	if _, err := lookPath("brew"); err != nil {
		return fmt.Errorf("bootstrap Homebrew: installer completed but brew is not available on PATH")
	}
	return nil
}

func bootstrapToolSatisfied(status vrooliruntime.ItemStatus) bool {
	switch status.ExecutionState {
	case vrooliruntime.ExecutionAlreadyPresent, vrooliruntime.ExecutionInstalled:
		return true
	default:
		return false
	}
}

func recoverHostToolPATH(home string) {
	_ = os.Setenv("PATH", hostreqkit.AugmentUserToolPath(home, os.Getenv("PATH"), os.Getenv("LOCALAPPDATA")))
}

func RunBuild(root, home string, stdout, stderr io.Writer) error {
	if err := lifecycle.ProvisionGeneratedPackages(root, home, stdout, stdout); err != nil {
		return fmt.Errorf("provision generated packages: %w", err)
	}
	buildDir := filepath.Join(config.RepoConfigDir(root), "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return err
	}

	gitCommit := "unknown"
	if output, err := shell.Output(shell.Spec{Dir: root, Name: "git", Args: []string{"rev-parse", "HEAD"}}); err == nil {
		if value := strings.TrimSpace(string(output)); value != "" {
			gitCommit = value
		}
	}
	buildTime := time.Now().UTC().Format(time.RFC3339)

	if err := buildProjectBinary(root, filepath.Join(buildDir, "vrooli-api"), "./cmd/vrooli-api", []string{"cmd/vrooli-api", "internal"}, gitCommit, buildTime, stdout, stderr); err != nil {
		return err
	}
	if err := buildProjectBinary(root, filepath.Join(buildDir, "vrooli"), "./cmd/vrooli", []string{"cmd/vrooli", "internal"}, gitCommit, buildTime, stdout, stderr); err != nil {
		return err
	}
	if err := buildProjectBinary(root, filepath.Join(buildDir, "vrooli-agent-launcher"), "./cmd/vrooli-agent-launcher", []string{"cmd/vrooli-agent-launcher", "packages/cli-core"}, gitCommit, buildTime, stdout, stderr); err != nil {
		return err
	}
	if err := buildNestedModuleBinary(filepath.Join(root, "cmd", "vrooli-policy-runner"), filepath.Join(buildDir, "vrooli-policy-runner"), stdout, stderr); err != nil {
		return err
	}
	return nil
}

func RunDevelopWithOptions(root, home string, opts Options, stdout, stderr io.Writer) error {
	return newSetupService(defaultSetupDeps()).RunDevelopWithOptions(root, home, opts, stdout, stderr)
}

func (s *setupService) RunDevelopWithOptions(root, home string, opts Options, stdout, stderr io.Writer) error {
	if err := s.deps.currentHost().ValidateDevelop(); err != nil {
		return err
	}

	projectScenario, err := s.deps.loadProject(root)
	if err != nil {
		return err
	}

	if setupNeeded(home, root, projectScenario.Slug) {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Running setup before develop")
		if err := s.RunSetupWithOptions(root, home, opts, stdout, stderr); err != nil {
			return err
		}
	}

	manager, err := s.deps.newPortsManager(root, home)
	if err != nil {
		return err
	}
	projectEnv, err := manager.BuildProjectEnvironment(projectScenario)
	if err != nil {
		return err
	}
	if err := s.applyDotEnv(root); err != nil {
		return err
	}
	overlay := make(envkit.Env, 0, len(projectEnv.EnvVars))
	for key, value := range projectEnv.EnvVars {
		overlay = append(overlay, key+"="+value)
	}
	env := envkit.WithOverlay(envkit.Env(os.Environ()), envkit.ForeignScenario, overlay)
	apiPort := resolveAPIPort(projectEnv.EnvVars)
	if apiPort <= 0 {
		apiPort = defaultAPIPort
	}

	healthy, err := apiAlreadyHealthy(apiPort)
	if err != nil {
		return err
	}
	healthTimeout := 30 * time.Second
	if !healthy {
		spec, err := buildAPILaunchSpec(root, home, env, apiPort)
		if err != nil {
			return err
		}
		healthTimeout = developHealthTimeout(spec)
		if err := s.deps.startProjectAPI(root, spec, stdout, stderr); err != nil {
			return err
		}
	}

	if err := s.deps.healthCheck(apiPort, healthTimeout); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "🚀 Vrooli API healthy on port %d with native scenario management\n", apiPort)
	if err := s.maybeOpenOnboarding(root, home, stdout, stderr); err != nil {
		_, _ = fmt.Fprintf(stderr, "[WARN]    Unable to auto-open onboarding: %v\n", err)
	}
	if strings.EqualFold(strings.TrimSpace(opts.Scenarios), "none") {
		_, _ = fmt.Fprintln(stdout, "Vrooli orchestrator skipped (--scenarios none)")
		return nil
	}
	return s.deps.startOrchestrator(root, home, stdout, stderr)
}

func developHealthTimeout(spec apiLaunchSpec) time.Duration {
	// A release install intentionally carries the bootstrap CLI, not a second
	// project API binary. The first develop therefore falls back to `go run`,
	// which may need to download the pinned toolchain and module graph on a
	// genuinely fresh host. Keep the normal fast-start budget for prebuilt
	// launchers while giving that one cold path enough deterministic runway.
	if filepath.Base(spec.Command) == "go" && len(spec.Args) > 0 && spec.Args[0] == "run" {
		return 2 * time.Minute
	}
	return 30 * time.Second
}

func buildProjectBinary(root, outputPath, target string, fingerprintPaths []string, gitCommit, buildTime string, stdout, stderr io.Writer) error {
	fingerprint, err := buildinfo.ComputeSourceFingerprintForPaths(root, fingerprintPaths...)
	if err != nil {
		return err
	}

	ldflags := fmt.Sprintf(
		"-s -w -X %s.GitCommit=%s -X %s.BuildTime=%s -X %s.Fingerprint=%s",
		"github.com/vrooli/vrooli/internal/buildinfo",
		gitCommit,
		"github.com/vrooli/vrooli/internal/buildinfo",
		buildTime,
		"github.com/vrooli/vrooli/internal/buildinfo",
		fingerprint,
	)

	env := append([]string(nil), os.Environ()...)
	env = append(env, "CGO_ENABLED=0")
	return shell.Run(shell.Spec{
		Name:   "go",
		Args:   []string{"build", "-trimpath", "-ldflags", ldflags, "-o", outputPath, target},
		Dir:    root,
		Env:    env,
		Stdout: stdout,
		Stderr: stderr,
		Stdin:  os.Stdin,
	})
}

// buildNestedModuleBinary builds a binary whose source lives in its own Go
// module beneath the repo root.
//
// vrooli-policy-runner is deliberately a separate module: it is the process
// boundary for native coding-agent hooks, runs on every Bash tool call, and
// must not drag in the main module's dependency graph. That isolation means it
// cannot be built with `go build ./cmd/...` from the repo root, so it builds
// from its own directory and does not receive the main module's buildinfo
// ldflags (those -X symbols do not exist in this module).
func buildNestedModuleBinary(moduleDir, outputPath string, stdout, stderr io.Writer) error {
	env := append([]string(nil), os.Environ()...)
	env = append(env, "CGO_ENABLED=0")
	return shell.Run(shell.Spec{
		Name:   "go",
		Args:   []string{"build", "-trimpath", "-o", outputPath, "."},
		Dir:    moduleDir,
		Env:    env,
		Stdout: stdout,
		Stderr: stderr,
		Stdin:  os.Stdin,
	})
}

func ensureProjectFilesystem(root, home string) error {
	locator, err := projectstate.NewLocator(home, root)
	if err != nil {
		return err
	}
	// Repo-project paths (under the repo root, covered by layout.project_config_dir).
	for _, path := range []string{
		filepath.Join(root, "data"),
		filepath.Join(config.RepoConfigDir(root), "build"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	// Operator-home paths: resolve names from the runtime_home authority and
	// route every create through the owned-write seam so a sudo'd setup never
	// leaves root-owned dirs in the operator's home.
	homeDirs := make([]string, 0, 4)
	for _, key := range []string{repocontract.HomeKeyBin, repocontract.HomeKeyLogs, repocontract.HomeKeyProcesses} {
		dir, err := repocontract.RuntimeHomeEntryPath(home, key)
		if err != nil {
			return err
		}
		homeDirs = append(homeDirs, dir)
	}
	homeDirs = append(homeDirs, locator.SetupStateDir())
	for _, dir := range homeDirs {
		if _, err := config.EnsureOwnedDir(dir); err != nil {
			return err
		}
	}
	return nil
}

func configureGit(root string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		return nil
	}
	cmd := exec.Command("git", "config", "core.filemode", "false")
	cmd.Dir = root
	return cmd.Run()
}

func (s *setupService) maybeInstallResources(root, home string, opts Options, stdout, stderr io.Writer) error {
	selection := strings.TrimSpace(opts.Resources)
	if selection == "" {
		selection = "enabled"
	}
	if selection == "none" {
		return nil
	}

	controller := s.deps.resourceController(root, home)
	if selection == "enabled" {
		names, err := enabledResourceNames(root)
		if err != nil {
			return err
		}
		if err := preflightDockerResources(root, home, names, opts); err != nil {
			return err
		}
		for _, name := range names {
			if err := controller.Run(name, []string{"install"}, stdout, stderr); err != nil {
				return err
			}
		}
		return nil
	}

	names := []string{}
	for _, raw := range strings.Split(selection, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	if err := preflightDockerResources(root, home, names, opts); err != nil {
		return err
	}
	for _, name := range names {
		if err := controller.Run(name, []string{"install"}, stdout, stderr); err != nil {
			return err
		}
	}
	return nil
}

func preflightDockerResources(root, home string, names []string, setupOpts ...Options) error {
	if len(names) == 0 {
		return nil
	}
	controller := resources.NewController(root, home)
	needsDocker := []string{}
	for _, name := range names {
		manifest, err := controller.ResourceManifest(name)
		if err != nil {
			continue
		}
		switch strings.TrimSpace(manifest.Driver) {
		case "docker-service", "compose-service":
			needsDocker = append(needsDocker, name)
		}
	}
	if len(needsDocker) == 0 {
		return nil
	}
	if len(setupOpts) == 0 {
		// Keep the narrow helper seam used by callers that only want to inspect
		// readiness. The setup apply path below supplies Options and performs
		// the provider ladder's repair/provisioning step.
		health := inspectDockerHealthFn()
		if health.InfoOK {
			return nil
		}
		detail := strings.TrimSpace(health.Detail)
		if detail == "" {
			detail = "Docker daemon is not reachable"
		}
		return fmt.Errorf("selected resources require Docker (%s), but Docker is not healthy: %s", strings.Join(needsDocker, ", "), dockerhost.DiagnosticLine(detail))
	}
	applyOpts := setupOpts[0]
	status, err := ensureContainerRuntimeFn("docker", vrooliruntime.EnsureOptions{
		Environment: applyOpts.Environment, SudoMode: applyOpts.SudoMode, DryRun: applyOpts.DryRun,
		AutoInstall: true, IncludeOptional: applyOpts.IncludeOptional, MaintenanceWindow: applyOpts.MaintenanceWindow,
	})
	if err != nil {
		return fmt.Errorf("selected resources require Docker (%s), but container-runtime setup failed: %w", strings.Join(needsDocker, ", "), err)
	}
	if status.Installed || applyOpts.DryRun {
		return nil
	}
	detail := "container runtime is not ready"
	if len(status.Notes) > 0 {
		detail = status.Notes[len(status.Notes)-1]
	}
	return fmt.Errorf("selected resources require Docker (%s), but container-runtime setup did not complete (provider=%s, state=%s): %s", strings.Join(needsDocker, ", "), status.SelectedProvider, status.ExecutionState, detail)
}

type resourceRunner interface {
	Run(name string, args []string, stdout, stderr io.Writer) error
}

func enabledResourceNames(root string) ([]string, error) {
	home, err := config.HomeDir()
	if err != nil {
		return nil, err
	}
	return resources.NewController(root, home).EnabledResourceNames()
}

func syncResourceSchemaArtifacts(root string) error {
	report, err := resources.SyncSchemaArtifacts(root)
	if err != nil {
		return err
	}
	if report.Passed {
		return nil
	}
	if len(report.MissingReferences) > 0 {
		first := report.MissingReferences[0]
		return fmt.Errorf("resource schema sync failed: scenario %s references missing resource %s", first.Scenario, first.Resource)
	}
	return fmt.Errorf("resource schema sync failed")
}

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

func (s *setupService) applyDotEnv(root string) error {
	values, err := s.deps.loadDotEnv(filepath.Join(root, ".env"))
	if err != nil {
		return err
	}
	for key, value := range values {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}

func loadDotEnv(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" {
			values[key] = value
		}
	}
	return values, scanner.Err()
}

func resolveAPIPort(values map[string]string) int {
	if raw := strings.TrimSpace(values["VROOLI_API_PORT"]); raw != "" {
		if port, err := strconv.Atoi(raw); err == nil && port > 0 {
			return port
		}
	}
	if raw := strings.TrimSpace(os.Getenv("VROOLI_API_PORT")); raw != "" {
		if port, err := strconv.Atoi(raw); err == nil && port > 0 {
			return port
		}
	}
	return defaultAPIPort
}

func apiAlreadyHealthy(port int) (bool, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		return false, nil
	}
	defer response.Body.Close()
	return response.StatusCode >= 200 && response.StatusCode < 300, nil
}

func buildAPILaunchSpec(root, home string, env []string, port int) (apiLaunchSpec, error) {
	logsDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyLogs)
	if err != nil {
		return apiLaunchSpec{}, err
	}
	binDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return apiLaunchSpec{}, err
	}
	logFile := filepath.Join(logsDir, "vrooli-api.log")
	for _, candidate := range []struct {
		command string
		args    []string
	}{
		{command: filepath.Join(config.RepoConfigDir(root), "build", "vrooli-api")},
		{command: filepath.Join(binDir, "vrooli-api")},
		{command: "go", args: []string{"run", "./cmd/vrooli-api"}},
	} {
		if candidate.command == "go" {
			if _, err := exec.LookPath("go"); err != nil {
				continue
			}
		} else if _, err := os.Stat(candidate.command); err != nil {
			continue
		}
		return apiLaunchSpec{
			Command: candidate.command,
			Args:    candidate.args,
			LogFile: logFile,
			Env:     env,
			Port:    port,
		}, nil
	}
	return apiLaunchSpec{}, fmt.Errorf("no project-level vrooli-api launcher found")
}

func startProjectAPI(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error {
	if err := os.MkdirAll(filepath.Dir(spec.LogFile), 0o755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(spec.LogFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := exec.Command(spec.Command, spec.Args...)
	cmd.Dir = root
	cmd.Env = spec.Env
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)
	if !platform.IsPIDRunning(cmd.Process.Pid) {
		return fmt.Errorf("vrooli-api exited immediately: %w", err)
	}
	return nil
}

func waitForHTTPHealth(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	for {
		response, err := client.Get(url)
		if err == nil {
			response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("vrooli-api failed health check on port %d", port)
		}
		time.Sleep(1 * time.Second)
	}
}

func startOrchestrator(root, home string, stdout, stderr io.Writer) error {
	service := orchestrator.New(root, home, stdout, stderr)
	status, exists, err := service.Status("vrooli-orchestrator")
	if err == nil && exists && status.Processes > 0 {
		return nil
	}
	_, err = service.Start("vrooli-orchestrator", lifecycle.StartOptions{})
	return err
}

func forceSetupApplies(slug string) bool {
	if strings.ToLower(strings.TrimSpace(os.Getenv("FORCE_SETUP"))) != "true" {
		return false
	}
	target := strings.TrimSpace(os.Getenv("FORCE_SETUP_SCENARIO"))
	return target == "" || target == slug
}

type onboardingPreferences struct {
	AutoOpen   *bool  `json:"auto_open,omitempty"`
	PromptedAt string `json:"prompted_at,omitempty"`
	Completed  bool   `json:"completed,omitempty"`
	Skipped    bool   `json:"skipped,omitempty"`
}

func (s *setupService) maybeOpenOnboarding(root, home string, stdout, stderr io.Writer) error {
	if onboardingDisabledByEnv() {
		return nil
	}
	if !onboardingScenarioExists(root) {
		return nil
	}

	configPath, err := onboardingConfigPath(home)
	if err != nil {
		return err
	}
	doc, prefs, err := loadOnboardingPreferences(configPath)
	if err != nil {
		return err
	}
	if onboardingAlreadyHandled(prefs) {
		return nil
	}

	executable, err := s.deps.osExecutable()
	if err != nil {
		return err
	}
	if err := launchDetachedOnboarding(root, executable); err != nil {
		return err
	}
	url, err := s.resolveOnboardingURL(executable)
	if err != nil {
		return err
	}
	if err := s.deps.openOnboardingURL(url); err != nil {
		return err
	}

	prefs.PromptedAt = s.deps.now().UTC().Format(time.RFC3339)
	if err := saveOnboardingPreferences(configPath, doc, prefs); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "[INFO]    Opening Vrooli onboarding at %s\n", url)
	return nil
}

func onboardingDisabledByEnv() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(onboardingSkipEnv)))
	return value == "1" || value == "true" || value == "yes"
}

func onboardingScenarioExists(root string) bool {
	_, err := os.Stat(scenario.ServicePath(root, onboardingSlug))
	return err == nil
}

func onboardingAlreadyHandled(prefs onboardingPreferences) bool {
	// PromptedAt is telemetry, not completion. A browser may have failed to
	// open, or the operator may have closed onboarding before applying. The
	// next setup run must offer the continuation again.
	if prefs.Completed || prefs.Skipped {
		return true
	}
	return prefs.AutoOpen != nil && !*prefs.AutoOpen
}

func onboardingConfigPath(home string) (string, error) {
	return filepath.Join(home, ".config", "vrooli", "config.json"), nil
}

func loadOnboardingPreferences(path string) (map[string]json.RawMessage, onboardingPreferences, error) {
	doc := map[string]json.RawMessage{}
	file, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return doc, onboardingPreferences{}, nil
		}
		return nil, onboardingPreferences{}, err
	}
	if len(file) == 0 {
		return doc, onboardingPreferences{}, nil
	}
	if err := json.Unmarshal(file, &doc); err != nil {
		return nil, onboardingPreferences{}, err
	}
	var prefs onboardingPreferences
	if raw, ok := doc["onboarding"]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &prefs); err != nil {
			return nil, onboardingPreferences{}, err
		}
	}
	return doc, prefs, nil
}

func saveOnboardingPreferences(path string, doc map[string]json.RawMessage, prefs onboardingPreferences) error {
	if doc == nil {
		doc = map[string]json.RawMessage{}
	}
	raw, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	doc["onboarding"] = raw
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func launchDetachedOnboarding(root, executable string) error {
	return launchOnboardingAsOperatorFn(root, executable)
}

func launchOnboardingAsOperator(root, executable string) error {
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return platform.RunAsInvokingUserInSession(ctx, executable,
		[]string{"scenario", "start", onboardingSlug},
		platform.IdentityCommandOptions{Dir: root, Stdin: devNull, Stdout: devNull, Stderr: devNull})
}

func (s *setupService) resolveOnboardingURL(executable string) (string, error) {
	deadline := s.deps.now().Add(30 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		output, err := s.deps.onboardingPortCommandRunner(ctx, executable, "scenario", "port", onboardingSlug, "UI_PORT")
		cancel()
		text := strings.TrimSpace(string(output))
		if err == nil {
			port, parseErr := strconv.Atoi(text)
			if parseErr == nil && port > 0 {
				return fmt.Sprintf("http://127.0.0.1:%d", port), nil
			}
		}
		if s.deps.now().After(deadline) {
			if text == "" {
				text = "port could not be resolved before timeout"
			}
			return "", fmt.Errorf("onboarding UI not ready: %s", text)
		}
		time.Sleep(1 * time.Second)
	}
}

func markComplete(home, root string) error {
	locator, err := projectstate.NewLocator(home, root)
	if err != nil {
		return err
	}
	if _, err := config.EnsureOwnedDir(locator.SetupStateDir()); err != nil {
		return err
	}

	payload := map[string]any{
		"setup_version": "2.0.0",
		"completed_at":  time.Now().Format(time.RFC3339),
		"phase":         "bootstrap_complete",
		"project_key":   locator.ProjectKey(),
		"root":          locator.Root(),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return config.WriteOwnedFile(locator.BootstrapCompletePath(), data, 0o644)
}

// runSetupStatus runs an inspection-only pass and prints the grouped overview.
// No mutating operations, safe to run without sudo.
func (s *setupService) runSetupStatus(root, home string, opts Options, stdout io.Writer) error {
	if err := s.deps.currentHost().ValidateSetup(); err != nil {
		return err
	}
	if _, err := s.deps.loadProject(root); err != nil {
		return err
	}
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
	requirements = bootstrapAwareRequirements(requirements)
	if executable, executableErr := s.deps.osExecutable(); executableErr == nil {
		requirements = addOnboardingApplyPrivilegeRequirement(requirements, executable)
	}
	report, err := s.deps.inspectRequirements(opts.Environment, requirements)
	if err != nil {
		return err
	}
	report = vrooliruntime.AnnotateInspectOnly(report, opts.IncludeOptional)
	_, _ = fmt.Fprintf(
		stdout,
		"[INFO]    Host requirements status (environment=%s)\n",
		displaySelection(report.Environment, displaySelection(opts.Environment, defaultEnvironment)),
	)
	mode := renderModeGrouped
	if opts.Verbose {
		mode = renderModeVerbose
	}
	renderSetupRequirementOverview(stdout, report, false, mode)
	renderPrivilegeBrokerStatus(stdout, s.deps.inspectPrivilegeBroker())
	return nil
}

func renderPrivilegeBrokerStatus(stdout io.Writer, status privilegebroker.SetupStatus) {
	if status.Available {
		_, _ = fmt.Fprintf(stdout, "[INFO]    Privilege broker: available (%s)\n", status.SocketPath)
		return
	}
	if status.Supported {
		_, _ = fmt.Fprintf(stdout, "[INFO]    Privilege broker: unavailable — %s\n", status.Reason)
	} else {
		_, _ = fmt.Fprintf(stdout, "[INFO]    Privilege broker: unsupported — %s\n", status.Reason)
	}
	if status.Recovery != "" {
		_, _ = fmt.Fprintf(stdout, "[INFO]    Privilege broker recovery: %s\n", status.Recovery)
	}
}

// runSetupExplain prints the full per-item block for one requirement.
func (s *setupService) runSetupExplain(root, home string, opts Options, stdout io.Writer) error {
	name := strings.TrimSpace(opts.ExplainName)
	if name == "" {
		return fmt.Errorf("setup explain requires a requirement name")
	}
	if err := s.deps.currentHost().ValidateSetup(); err != nil {
		return err
	}
	if _, err := s.deps.loadProject(root); err != nil {
		return err
	}
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
	requirements = bootstrapAwareRequirements(requirements)
	if executable, executableErr := s.deps.osExecutable(); executableErr == nil {
		requirements = addOnboardingApplyPrivilegeRequirement(requirements, executable)
	}
	report, err := s.deps.inspectRequirements(opts.Environment, requirements)
	if err != nil {
		return err
	}
	item, ok := findItemByName(report, name)
	if !ok {
		return fmt.Errorf("no host requirement named %q (run 'vrooli setup status' to list)", name)
	}
	_, _ = fmt.Fprintf(stdout, "[INFO]    %s\n", item.Name)
	renderRequirementVerboseItem(stdout, item, false)
	return nil
}
