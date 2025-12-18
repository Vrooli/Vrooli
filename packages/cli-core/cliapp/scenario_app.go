package cliapp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// ScenarioOptions bundles common wiring for scenario CLIs so individual
// scenarios don't have to repeat config loading, API client setup, stale
// checking, and configure command plumbing.
type ScenarioOptions struct {
	Name               string
	Version            string
	Description        string
	DefaultAPIBase     string
	APIEnvVars         []string
	APIPortEnvVars     []string
	APIPortDetector    func() string
	ConfigDirEnvVars   []string
	SourceRootEnvVars  []string
	ColorEnabled       *bool
	OnColor            func(enabled bool)
	Commands           []CommandGroup
	TokenKeys          []string
	APIBaseKeys        []string
	TokenEnvVars       []string
	Preflight          func(cmd Command, global GlobalOptions, app *ScenarioApp) error
	BuildFingerprint   string
	BuildTimestamp     string
	BuildSourceRoot    string
	HTTPClientOptions  cliutil.HTTPClientOptions
	HTTPTimeoutEnvVars []string
	DefaultHTTPTimeout time.Duration
	AllowAnonymous     bool
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

// SetCommands rebuilds the CLI with the provided command groups while keeping
// the shared wiring intact.
func (a *ScenarioApp) SetCommands(commands []CommandGroup) {
	a.options.Commands = commands

	colorEnabled := DefaultColorEnabled()
	if a.options.ColorEnabled != nil {
		colorEnabled = *a.options.ColorEnabled
	}

	preflight := func(cmd Command, global GlobalOptions) error {
		if cmd.NeedsAPI {
			if _, err := cliutil.ValidateAPIBase(a.APIBaseOptions()); err != nil {
				// If auto-start is enabled, try to start the scenario
				if global.AutoStart {
					if startErr := a.tryAutoStart(); startErr != nil {
						return fmt.Errorf("failed to auto-start %s: %w", a.options.Name, startErr)
					}
					// Retry API validation after starting
					if _, err := cliutil.ValidateAPIBase(a.APIBaseOptions()); err != nil {
						return fmt.Errorf("%s API still not reachable after auto-start", a.options.Name)
					}
				} else {
					// Provide actionable error with auto-start suggestion
					return fmt.Errorf("%s API is not reachable.\n\nTo auto-start the scenario:\n  %s --auto-start %s\n\nOr start manually:\n  vrooli scenario start %s", a.options.Name, a.options.Name, cmd.Name, a.options.Name)
				}
			}
			if !a.options.AllowAnonymous && strings.TrimSpace(a.tokenSource()) == "" {
				return fmt.Errorf("API token is required for %s; set one via configure or %s", cmd.Name, strings.Join(a.options.TokenEnvVars, ", "))
			}
		}
		if a.options.Preflight != nil {
			return a.options.Preflight(cmd, global, a)
		}
		return nil
	}

	a.CLI = NewApp(AppOptions{
		Name:         a.options.Name,
		Version:      a.options.Version,
		Description:  a.options.Description,
		Commands:     commands,
		APIOverride:  &a.APIOverride,
		ColorEnabled: colorEnabled,
		OnColor:      a.options.OnColor,
		StaleChecker: a.StaleChecker,
		Preflight:    preflight,
	})
}

// APIBaseOptions returns the current API base resolution options for use in
// tests or custom flows.
func (a *ScenarioApp) APIBaseOptions() cliutil.APIBaseOptions {
	return a.baseOptions()
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

func applyTimeoutOpts(opts ScenarioOptions) cliutil.HTTPClientOptions {
	clientOpts := opts.HTTPClientOptions
	if clientOpts.Timeout == 0 {
		clientOpts.Timeout = cliutil.ResolveTimeout(opts.HTTPTimeoutEnvVars, opts.DefaultHTTPTimeout)
	}
	return clientOpts
}

// tryAutoStart attempts to start the scenario via vrooli and waits for the API to become available.
func (a *ScenarioApp) tryAutoStart() error {
	fmt.Printf("Starting %s...\n", a.options.Name)

	// Start the scenario in background
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "start", a.options.Name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vrooli scenario start failed: %w", err)
	}

	// Wait for API to become available (poll port detector)
	fmt.Printf("Waiting for %s API...\n", a.options.Name)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := cliutil.ValidateAPIBase(a.APIBaseOptions()); err == nil {
			fmt.Printf("%s API is ready\n", a.options.Name)
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for API to become available")
}
