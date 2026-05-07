package cliapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	repocontract "github.com/vrooli/repo-contract-go"
)

var (
	currentExecutablePath  = os.Executable
	findScenarioRepoRoot   = repocontract.FindRepoRootFromEnvOrCWD
	resolveScenarioRoot    = repocontract.ResolveScenarioPath
	resolveScenarioCLIPath = func(root, scenario string) (string, error) {
		return repocontract.ResolveScenarioFile(root, scenario, "cli")
	}
	execScenarioStartCommand = func(name string) *exec.Cmd {
		return exec.Command("vrooli", "scenario", "start", name)
	}
)

func defaultIfEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func defaultStringSlice(value, fallback []string) []string {
	if len(value) == 0 {
		return append([]string(nil), fallback...)
	}
	return append([]string(nil), value...)
}

// ScenarioOptions bundles common wiring for scenario CLIs so individual
// scenarios don't have to repeat config loading, API client setup, stale
// checking, and configure command plumbing.
type ScenarioOptions struct {
	Name               string
	Version            string
	Description        string
	DefaultAPIBase     string
	APIPrefix          string
	HealthPath         string
	LegacyHealthPaths  []string
	APIEnvVars         []string
	APIPortEnvVars     []string
	APIPortDetector    func() string
	ConfigDirEnvVars   []string
	SourceRootEnvVars  []string
	ColorEnabled       *bool
	OnColor            func(enabled bool)
	Commands           []CommandGroup
	SubcommandGroups   []SubcommandGroup
	TokenKeys          []string
	APIBaseKeys        []string
	TokenEnvVars       []string
	Preflight          func(cmd Command, global GlobalOptions, app *ScenarioApp) error
	BuildFingerprint   string
	BuildTimestamp     string
	BuildSourceRoot    string
	SourceContextPath  string
	ManifestSourcePath string
	FreshnessInputs    []string
	HTTPClientOptions  cliutil.HTTPClientOptions
	HTTPTimeoutEnvVars []string
	DefaultHTTPTimeout time.Duration
	AllowAnonymous     bool
}

// StandardScenarioOptions provides a higher-level constructor for the common
// scenario CLI shape: standard env wiring, standard operational commands, and
// optional custom command registration hooks.
type StandardScenarioOptions struct {
	Name                    string
	Version                 string
	Description             string
	DefaultAPIBase          string
	APIPrefix               string
	HealthPath              string
	LegacyHealthPaths       []string
	ExtraAPIEnvVars         []string
	ExtraAPIPortEnvVars     []string
	ExtraConfigDirEnvVars   []string
	ExtraSourceRootEnvVars  []string
	ExtraTokenEnvVars       []string
	ExtraHTTPTimeoutEnvVars []string
	ColorEnabled            *bool
	OnColor                 func(enabled bool)
	Preflight               func(cmd Command, global GlobalOptions, app *ScenarioApp) error
	BuildFingerprint        string
	BuildTimestamp          string
	BuildSourceRoot         string
	ManifestSourcePath      string
	FreshnessInputs         []string
	HTTPClientOptions       cliutil.HTTPClientOptions
	DefaultHTTPTimeout      time.Duration
	AllowAnonymous          bool
	IncludeStatusCommand    *bool
	IncludeConfigureCommand *bool
	ConfigureAPIBaseKeys    []string
	ConfigureTokenKeys      []string
	CommandGroups           func(app *ScenarioApp) []CommandGroup
	SubcommandGroups        func(app *ScenarioApp) []SubcommandGroup
}

// ScenarioApp encapsulates the shared CLI scaffolding for a scenario CLI.
type ScenarioApp struct {
	ConfigFile   *cliutil.ConfigFile
	Config       cliutil.APIConfig
	APIOverride  string
	HTTPClient   *cliutil.HTTPClient
	APIClient    *cliutil.APIClient
	CLI          *App
	StaleChecker *cliutil.StaleChecker
	tokenSource  func() string
	warnedLocal  bool

	options     ScenarioOptions
	baseOptions func() cliutil.APIBaseOptions
}

// NewScenarioApp builds a ScenarioApp with default preflight (API validation),
// stale checking, and config persistence. Commands can be updated later via
// SetCommands to avoid circular setup in callers.
func NewScenarioApp(opts ScenarioOptions) (*ScenarioApp, error) {
	if len(opts.APIBaseKeys) == 0 {
		opts.APIBaseKeys = []string{"api_base"}
	}
	if strings.TrimSpace(opts.APIPrefix) == "" {
		opts.APIPrefix = "/api/v1"
	}
	if strings.TrimSpace(opts.HealthPath) == "" {
		opts.HealthPath = "/health"
	}
	if len(opts.TokenKeys) == 0 {
		opts.TokenKeys = []string{"token", "api_token"}
	}
	if len(opts.TokenEnvVars) == 0 {
		slug := strings.ToUpper(strings.ReplaceAll(opts.Name, "-", "_"))
		opts.TokenEnvVars = []string{slug + "_API_TOKEN", "VROOLI_API_TOKEN"}
	}
	if len(opts.HTTPTimeoutEnvVars) == 0 {
		slug := strings.ToUpper(strings.ReplaceAll(opts.Name, "-", "_"))
		opts.HTTPTimeoutEnvVars = []string{slug + "_HTTP_TIMEOUT", "VROOLI_HTTP_TIMEOUT"}
	}
	if opts.DefaultHTTPTimeout == 0 {
		opts.DefaultHTTPTimeout = 30 * time.Second
	}

	configFile, cfg, err := cliutil.LoadAPIConfig(opts.Name, opts.ConfigDirEnvVars...)
	if err != nil {
		return nil, err
	}

	app := &ScenarioApp{
		ConfigFile:   configFile,
		Config:       cfg,
		APIOverride:  "",
		HTTPClient:   cliutil.NewHTTPClient(applyTimeoutOpts(opts)),
		StaleChecker: cliutil.NewStaleChecker(opts.Name, opts.BuildFingerprint, opts.BuildTimestamp, opts.BuildSourceRoot, opts.SourceRootEnvVars...),
		options:      opts,
	}
	app.StaleChecker.SourceContextPath = opts.SourceContextPath
	app.StaleChecker.ManifestSourcePath = opts.ManifestSourcePath
	app.StaleChecker.FreshnessInputs = append([]string(nil), opts.FreshnessInputs...)
	app.tokenSource = func() string {
		for _, env := range opts.TokenEnvVars {
			if val := strings.TrimSpace(os.Getenv(env)); val != "" {
				return val
			}
		}
		return app.Config.Token
	}
	app.baseOptions = func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{
			Override:     app.APIOverride,
			EnvVars:      opts.APIEnvVars,
			ConfigBase:   app.Config.APIBase,
			PortEnvVars:  opts.APIPortEnvVars,
			PortDetector: opts.APIPortDetector,
			DefaultBase:  opts.DefaultAPIBase,
		}
	}
	app.APIClient = cliutil.NewAPIClient(app.HTTPClient, app.APIBaseOptions, app.tokenSource)
	app.SetCommands(opts.Commands)
	return app, nil
}

// NewStandardScenarioApp builds the common scenario CLI shape with standard env
// derivation, standard base commands, and optional custom command hooks.
func NewStandardScenarioApp(opts StandardScenarioOptions) (*ScenarioApp, error) {
	env := StandardScenarioEnv(opts.Name, ScenarioEnvOptions{
		ExtraAPIEnvVars:         opts.ExtraAPIEnvVars,
		ExtraAPIPortEnvVars:     opts.ExtraAPIPortEnvVars,
		ExtraConfigDirEnvVars:   opts.ExtraConfigDirEnvVars,
		ExtraSourceRootEnvVars:  opts.ExtraSourceRootEnvVars,
		ExtraTokenEnvVars:       opts.ExtraTokenEnvVars,
		ExtraHTTPTimeoutEnvVars: opts.ExtraHTTPTimeoutEnvVars,
	})

	app, err := NewScenarioApp(ScenarioOptions{
		Name:               opts.Name,
		Version:            opts.Version,
		Description:        opts.Description,
		DefaultAPIBase:     opts.DefaultAPIBase,
		APIPrefix:          opts.APIPrefix,
		HealthPath:         opts.HealthPath,
		LegacyHealthPaths:  opts.LegacyHealthPaths,
		APIEnvVars:         env.APIEnvVars,
		APIPortEnvVars:     env.APIPortEnvVars,
		APIPortDetector:    cliutil.DetectPortFromVrooli(opts.Name, "API_PORT"),
		ConfigDirEnvVars:   env.ConfigDirEnvVars,
		SourceRootEnvVars:  env.SourceRootEnvVars,
		ColorEnabled:       opts.ColorEnabled,
		OnColor:            opts.OnColor,
		TokenEnvVars:       env.TokenEnvVars,
		Preflight:          opts.Preflight,
		BuildFingerprint:   opts.BuildFingerprint,
		BuildTimestamp:     opts.BuildTimestamp,
		BuildSourceRoot:    opts.BuildSourceRoot,
		SourceContextPath:  "..",
		ManifestSourcePath: defaultIfEmpty(opts.ManifestSourcePath, ".vrooli/service.json"),
		FreshnessInputs:    defaultStringSlice(opts.FreshnessInputs, []string{"cli/**", ".vrooli/service.json"}),
		HTTPClientOptions:  opts.HTTPClientOptions,
		HTTPTimeoutEnvVars: env.HTTPTimeoutEnvVars,
		DefaultHTTPTimeout: opts.DefaultHTTPTimeout,
		AllowAnonymous:     opts.AllowAnonymous,
	})
	if err != nil {
		return nil, err
	}

	commands := app.StandardBaseCommandGroups(StandardBaseCommandOptions{
		IncludeStatusCommand:    opts.IncludeStatusCommand,
		IncludeConfigureCommand: opts.IncludeConfigureCommand,
		ConfigureAPIBaseKeys:    opts.ConfigureAPIBaseKeys,
		ConfigureTokenKeys:      opts.ConfigureTokenKeys,
	})
	if opts.CommandGroups != nil {
		commands = append(commands, opts.CommandGroups(app)...)
	}

	var subcommandGroups []SubcommandGroup
	if opts.SubcommandGroups != nil {
		subcommandGroups = opts.SubcommandGroups(app)
	}

	app.SetCommandsWithSubgroups(commands, subcommandGroups)
	return app, nil
}

// SetCommands rebuilds the CLI with the provided command groups while keeping
// the shared wiring intact.
func (a *ScenarioApp) SetCommands(commands []CommandGroup) {
	a.SetCommandsWithSubgroups(commands, nil)
}

// SetCommandsWithSubgroups rebuilds the CLI with both flat command groups and
// hierarchical subcommand groups.
func (a *ScenarioApp) SetCommandsWithSubgroups(commands []CommandGroup, subcommandGroups []SubcommandGroup) {
	a.options.Commands = commands
	a.options.SubcommandGroups = subcommandGroups

	colorEnabled := DefaultColorEnabled()
	if a.options.ColorEnabled != nil {
		colorEnabled = *a.options.ColorEnabled
	}

	preflight := func(cmd Command, global GlobalOptions) error {
		a.warnIfRunningScenarioLocalBinary()

		if cmd.NeedsAPI {
			if err := a.ensureAPIReachable(cmd, global.AutoStart); err != nil {
				return err
			}
		}
		if global.DryRun {
			a.HTTPClient.SetDryRun(true)
		}
		if a.options.Preflight != nil {
			return a.options.Preflight(cmd, global, a)
		}
		return nil
	}

	a.CLI = NewApp(AppOptions{
		Name:             a.options.Name,
		Version:          a.options.Version,
		Description:      a.options.Description,
		Commands:         commands,
		SubcommandGroups: subcommandGroups,
		APIOverride:      &a.APIOverride,
		ColorEnabled:     colorEnabled,
		OnColor:          a.options.OnColor,
		StaleChecker:     a.StaleChecker,
		Preflight:        preflight,
	})
	a.CLI.AttachScenario(a)
}

func (a *ScenarioApp) warnIfRunningScenarioLocalBinary() {
	if a.warnedLocal || strings.TrimSpace(os.Getenv("VROOLI_SUPPRESS_CLI_PATH_WARNING")) != "" {
		return
	}
	executablePath, err := currentExecutablePath()
	if err != nil {
		return
	}
	relativeScenario, cliDir, ok := resolveScenarioLocalCLIContext(a.options.Name)
	if !ok || !sameScenarioPath(filepath.Dir(executablePath), cliDir) {
		return
	}

	a.warnedLocal = true
	fmt.Fprintf(os.Stderr, "Warning: running %s from a scenario-local CLI binary (%s).\n", a.options.Name, executablePath)
	fmt.Fprintf(os.Stderr, "Install and run the canonical binary instead:\n")
	fmt.Fprintf(os.Stderr, "  cd %s/cli && ./install.sh\n", relativeScenario)
	fmt.Fprintf(os.Stderr, "  %s <command>\n", a.options.Name)
}

func isScenarioLocalCLIExecutablePath(appName, executablePath string) bool {
	_, cliDir, ok := resolveScenarioLocalCLIContext(appName)
	if !ok {
		return false
	}
	return sameScenarioPath(filepath.Dir(executablePath), cliDir)
}

func resolveScenarioLocalCLIContext(appName string) (string, string, bool) {
	normalizedName := strings.TrimSpace(appName)
	if normalizedName == "" {
		return "", "", false
	}

	root, err := findScenarioRepoRoot()
	if err != nil {
		return "", "", false
	}
	scenarioRoot, err := resolveScenarioRoot(root, normalizedName)
	if err != nil {
		return "", "", false
	}
	cliDir, err := resolveScenarioCLIPath(root, normalizedName)
	if err != nil {
		return "", "", false
	}
	relativeScenario, err := filepath.Rel(root, scenarioRoot)
	if err != nil {
		return "", "", false
	}
	relativeScenario = filepath.ToSlash(relativeScenario)
	if relativeScenario == "." || strings.HasPrefix(relativeScenario, "../") {
		return "", "", false
	}
	return relativeScenario, cliDir, true
}

func sameScenarioPath(a, b string) bool {
	a = filepath.ToSlash(filepath.Clean(strings.TrimSpace(a)))
	b = filepath.ToSlash(filepath.Clean(strings.TrimSpace(b)))
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(a, b)
}

// APIBaseOptions returns the current API base resolution options for use in
// tests or custom flows.
func (a *ScenarioApp) APIBaseOptions() cliutil.APIBaseOptions {
	return a.baseOptions()
}

// APIPrefix returns the normalized versioned API prefix for scenario routes.
func (a *ScenarioApp) APIPrefix() string {
	return normalizeAPIPath(a.options.APIPrefix)
}

// HealthPath returns the normalized root health endpoint path.
func (a *ScenarioApp) HealthPath() string {
	return normalizeAPIPath(a.options.HealthPath)
}

// APIBase returns the resolved API base. If the configured base is root-level,
// the scenario API prefix is appended.
func (a *ScenarioApp) APIBase() string {
	base := strings.TrimRight(strings.TrimSpace(cliutil.DetermineAPIBase(a.APIBaseOptions())), "/")
	if base == "" {
		return ""
	}
	prefix := a.APIPrefix()
	if prefix == "" || prefix == "/" {
		return base
	}
	if strings.HasSuffix(base, prefix) {
		return base
	}
	return base + prefix
}

// APIRootBase returns the resolved root API base with any configured API
// prefix removed.
func (a *ScenarioApp) APIRootBase() string {
	base := strings.TrimRight(strings.TrimSpace(cliutil.DetermineAPIBase(a.APIBaseOptions())), "/")
	if base == "" {
		return ""
	}
	prefix := a.APIPrefix()
	if prefix != "" && prefix != "/" && strings.HasSuffix(base, prefix) {
		return strings.TrimRight(strings.TrimSuffix(base, prefix), "/")
	}
	return base
}

// APIPath normalizes a versioned API path against the configured API prefix.
func (a *ScenarioApp) APIPath(path string) string {
	path = normalizeAPIPath(path)
	if path == "" {
		return ""
	}
	prefix := a.APIPrefix()
	if prefix == "" || prefix == "/" {
		return path
	}
	base := strings.TrimRight(strings.TrimSpace(cliutil.DetermineAPIBase(a.APIBaseOptions())), "/")
	if strings.HasSuffix(base, prefix) {
		return path
	}
	return prefix + path
}

// APIRootPath normalizes an operational root path such as /health.
func (a *ScenarioApp) APIRootPath(path string) string {
	return normalizeAPIPath(path)
}

// Get performs a GET request against the scenario's versioned API base.
func (a *ScenarioApp) Get(path string, query url.Values) ([]byte, error) {
	if a.APIClient == nil {
		a.APIClient = cliutil.NewAPIClient(a.HTTPClient, a.APIBaseOptions, a.tokenSource)
	}
	return a.APIClient.Get(a.APIPath(path), query)
}

// Request performs an HTTP request against the scenario's versioned API base.
func (a *ScenarioApp) Request(method, path string, query url.Values, body interface{}) ([]byte, error) {
	if a.APIClient == nil {
		a.APIClient = cliutil.NewAPIClient(a.HTTPClient, a.APIBaseOptions, a.tokenSource)
	}
	return a.APIClient.Request(method, a.APIPath(path), query, body)
}

// GetRoot performs a GET request against the scenario's root API base.
func (a *ScenarioApp) GetRoot(path string, query url.Values) ([]byte, error) {
	return a.rootRequest("GET", a.APIRootPath(path), query, nil)
}

// RequestRoot performs an HTTP request against the scenario's root API base.
func (a *ScenarioApp) RequestRoot(method, path string, query url.Values, body interface{}) ([]byte, error) {
	return a.rootRequest(method, a.APIRootPath(path), query, body)
}

// SaveConfig persists the current API config to disk.
func (a *ScenarioApp) SaveConfig() error {
	return a.ConfigFile.Save(a.Config)
}

// ConfigureCommand returns a standard "configure" command that supports
// updating api_base and token (with optional aliases).
func (a *ScenarioApp) ConfigureCommand(apiBaseKeys, tokenKeys []string) Command {
	if len(apiBaseKeys) == 0 {
		apiBaseKeys = a.options.APIBaseKeys
	}
	if len(tokenKeys) == 0 {
		tokenKeys = a.options.TokenKeys
	}
	return Command{
		Name:        "configure",
		NeedsAPI:    false,
		Description: "View or update CLI settings (api_base, token)",
		Usage:       fmt.Sprintf("%s configure [<key> <value>]", a.options.Name),
		HelpText:    "Run without arguments to print the current config. Supported keys include api_base and token.",
		Run: func(args []string) error {
			if len(args) == 0 {
				payload, _ := json.MarshalIndent(a.Config, "", "  ")
				fmt.Println(string(payload))
				return nil
			}
			if len(args) != 2 {
				return fmt.Errorf("usage: configure <key> <value>")
			}
			key := args[0]
			value := args[1]

			switch {
			case keyMatch(key, apiBaseKeys):
				a.Config.APIBase = value
			case keyMatch(key, tokenKeys):
				a.Config.Token = value
			default:
				return fmt.Errorf("unknown configuration key: %s", key)
			}

			if err := a.SaveConfig(); err != nil {
				return err
			}
			fmt.Printf("Updated %s\n", key)
			return nil
		},
	}
}

func keyMatch(key string, allowed []string) bool {
	for _, k := range allowed {
		if k == key {
			return true
		}
	}
	return false
}

func normalizeAPIPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func applyTimeoutOpts(opts ScenarioOptions) cliutil.HTTPClientOptions {
	clientOpts := opts.HTTPClientOptions
	if clientOpts.Timeout == 0 {
		clientOpts.Timeout = cliutil.ResolveTimeout(opts.HTTPTimeoutEnvVars, opts.DefaultHTTPTimeout)
	}
	return clientOpts
}

type apiRecoveryContext struct {
	ResolvedAPIBase   string
	ConfiguredAPIBase string
	DetectedAPIBase   string
	Cause             string
	MissingAPIBase    bool
}

func (a *ScenarioApp) ensureAPIReachable(cmd Command, autoStart bool) error {
	base, err := cliutil.ValidateAPIBase(a.APIBaseOptions())
	if err != nil {
		if autoStart {
			if startErr := a.tryAutoStart(); startErr != nil {
				return fmt.Errorf("failed to auto-start %s: %w", a.options.Name, startErr)
			}
			base, err = cliutil.ValidateAPIBase(a.APIBaseOptions())
		}
		if err != nil {
			return a.apiRecoveryError(cmd.Name, apiRecoveryContext{
				ResolvedAPIBase:   base,
				ConfiguredAPIBase: strings.TrimSpace(a.Config.APIBase),
				DetectedAPIBase:   a.detectedAPIBase(),
				Cause:             err.Error(),
				MissingAPIBase:    true,
			})
		}
	}

	if !a.options.AllowAnonymous && strings.TrimSpace(a.tokenSource()) == "" {
		return fmt.Errorf("API token is required for %s; set one via configure or %s", cmd.Name, strings.Join(a.options.TokenEnvVars, ", "))
	}

	probeErr := a.probeAPIHealth()
	if probeErr == nil {
		return nil
	} else if autoStart {
		if startErr := a.tryAutoStart(); startErr != nil {
			return fmt.Errorf("failed to auto-start %s: %w", a.options.Name, startErr)
		}
		probeErr = a.probeAPIHealth()
		if probeErr == nil {
			return nil
		}
		return a.apiRecoveryError(cmd.Name, apiRecoveryContext{
			ResolvedAPIBase:   strings.TrimSpace(cliutil.DetermineAPIBase(a.APIBaseOptions())),
			ConfiguredAPIBase: strings.TrimSpace(a.Config.APIBase),
			DetectedAPIBase:   a.detectedAPIBase(),
			Cause:             probeErr.Error(),
		})
	}

	return a.apiRecoveryError(cmd.Name, apiRecoveryContext{
		ResolvedAPIBase:   strings.TrimSpace(cliutil.DetermineAPIBase(a.APIBaseOptions())),
		ConfiguredAPIBase: strings.TrimSpace(a.Config.APIBase),
		DetectedAPIBase:   a.detectedAPIBase(),
		Cause:             probeErr.Error(),
	})
}

func (a *ScenarioApp) probeAPIHealth() error {
	_, err := a.fetchHealth()
	return err
}

func (a *ScenarioApp) detectedAPIBase() string {
	if a.options.APIPortDetector == nil {
		return ""
	}
	port := strings.TrimSpace(a.options.APIPortDetector())
	if port == "" {
		return ""
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

func (a *ScenarioApp) apiRecoveryError(commandName string, ctx apiRecoveryContext) error {
	report := NewAPIRecoveryReport(APIRecoveryReportOptions{
		AppName:           a.options.Name,
		CommandName:       commandName,
		ResolvedAPIBase:   ctx.ResolvedAPIBase,
		ConfiguredAPIBase: ctx.ConfiguredAPIBase,
		DetectedAPIBase:   ctx.DetectedAPIBase,
		Cause:             ctx.Cause,
		MissingAPIBase:    ctx.MissingAPIBase,
	})
	rendered, err := RenderOperationalReportString(report)
	if err != nil {
		return fmt.Errorf("render API recovery report: %w", err)
	}
	return errors.New(strings.TrimRight(rendered, "\n"))
}

// tryAutoStart attempts to start the scenario via vrooli and waits for the API to become available.
func (a *ScenarioApp) tryAutoStart() error {
	if cliutil.IsAgentControlledContext() {
		envVar := firstNonEmpty(a.options.APIEnvVars)
		if envVar == "" {
			envVar = strings.ToUpper(strings.ReplaceAll(a.options.Name, "-", "_")) + "_API_BASE"
		}
		return fmt.Errorf("%s API base was not detected inside an agent sandbox; set %s or run `vrooli scenario status %s` / `vrooli scenario start %s` outside the session", a.options.Name, envVar, a.options.Name, a.options.Name)
	}
	fmt.Printf("Starting %s...\n", a.options.Name)

	// Scenario start is itself a lifecycle orchestration command. Let it run to
	// completion instead of imposing an arbitrary local subprocess deadline.
	cmd := execScenarioStartCommand(a.options.Name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vrooli scenario start failed: %w", err)
	}

	// Wait for API to become available (poll port detector)
	fmt.Printf("Waiting for %s API...\n", a.options.Name)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := cliutil.ValidateAPIBase(a.APIBaseOptions()); err == nil && a.probeAPIHealth() == nil {
			fmt.Printf("%s API is ready\n", a.options.Name)
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for API to become available")
}

func firstNonEmpty(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
