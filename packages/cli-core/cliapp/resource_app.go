package cliapp

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// ResourceOptions bundles common wiring for resource CLIs so individual
// resources don't have to repeat stale checking and control-plane delegation.
type ResourceOptions struct {
	Name                string
	Version             string
	Description         string
	Commands            []CommandGroup
	SourceRootEnvVars   []string
	ControlPlaneEnvVars []string
	ColorEnabled        *bool
	OnColor             func(enabled bool)
	Preflight           func(cmd Command, global GlobalOptions, app *ResourceApp) error
	BuildFingerprint    string
	BuildTimestamp      string
	BuildSourceRoot     string
	SourceContextPath   string
	ManifestSourcePath  string
	FreshnessInputs     []string
	LookPathFunc        func(string) (string, error)
	CommandRunner       func(*exec.Cmd) error
}

// ResourceApp encapsulates the shared CLI scaffolding for a resource CLI.
type ResourceApp struct {
	CLI          *App
	StaleChecker *cliutil.StaleChecker

	options ResourceOptions
}

// NewResourceApp builds a ResourceApp with default stale checking and command
// delegation helpers. Commands can be updated later via SetCommands.
func NewResourceApp(opts ResourceOptions) (*ResourceApp, error) {
	app := &ResourceApp{
		StaleChecker: cliutil.NewStaleChecker(opts.Name, opts.BuildFingerprint, opts.BuildTimestamp, opts.BuildSourceRoot, opts.SourceRootEnvVars...),
		options:      opts,
	}
	if strings.TrimSpace(opts.SourceContextPath) == "" {
		app.StaleChecker.SourceContextPath = ".."
	} else {
		app.StaleChecker.SourceContextPath = opts.SourceContextPath
	}
	if strings.TrimSpace(opts.ManifestSourcePath) == "" {
		app.StaleChecker.ManifestSourcePath = "resource.json"
	} else {
		app.StaleChecker.ManifestSourcePath = opts.ManifestSourcePath
	}
	if len(opts.FreshnessInputs) > 0 {
		app.StaleChecker.FreshnessInputs = append([]string(nil), opts.FreshnessInputs...)
	} else {
		app.StaleChecker.FreshnessInputs = []string{"cli/**", "resource.json"}
	}
	app.SetCommands(opts.Commands)
	return app, nil
}

// SetCommands rebuilds the CLI with the provided command groups while keeping
// the shared wiring intact.
func (a *ResourceApp) SetCommands(commands []CommandGroup) {
	a.options.Commands = commands

	colorEnabled := DefaultColorEnabled()
	if a.options.ColorEnabled != nil {
		colorEnabled = *a.options.ColorEnabled
	}

	preflight := func(cmd Command, global GlobalOptions) error {
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
		ColorEnabled: colorEnabled,
		OnColor:      a.options.OnColor,
		Preflight:    preflight,
	})
}

// StandardLifecycleCommands returns the standard resource lifecycle command set
// wired to delegate through the Go-native vrooli control plane.
func (a *ResourceApp) StandardLifecycleCommands() []CommandGroup {
	standard := []struct {
		name         string
		description  string
		resourceArgs []string
	}{
		{name: "info", description: "Show resource metadata", resourceArgs: []string{"info"}},
		{name: "status", description: "Show resource status", resourceArgs: []string{"status"}},
		{name: "install", description: "Install the resource", resourceArgs: []string{"install"}},
		{name: "uninstall", description: "Uninstall the resource", resourceArgs: []string{"uninstall"}},
		{name: "start", description: "Start the resource", resourceArgs: []string{"start"}},
		{name: "stop", description: "Stop the resource", resourceArgs: []string{"stop"}},
		{name: "restart", description: "Restart the resource", resourceArgs: []string{"restart"}},
		{name: "logs", description: "Show resource logs", resourceArgs: []string{"logs"}},
	}

	commands := make([]Command, 0, len(standard))
	for _, item := range standard {
		commands = append(commands, a.DelegatingCommand(item.name, item.description, item.resourceArgs...))
	}

	return []CommandGroup{{
		Title:    "Lifecycle",
		Commands: commands,
	}}
}

// DelegatingCommand returns a resource CLI command that delegates to
// `vrooli resource <subcommand> <resource>`.
func (a *ResourceApp) DelegatingCommand(name, description string, resourceArgs ...string) Command {
	return Command{
		Name:        name,
		Description: description,
		Run: func(args []string) error {
			if a.StaleChecker != nil {
				a.StaleChecker.ReexecArgs = append([]string(nil), os.Args[1:]...)
				if restarted := a.StaleChecker.CheckAndMaybeRebuild(); restarted {
					return nil
				}
			}
			delegated := append([]string{"resource"}, resourceArgs...)
			delegated = append(delegated, a.options.Name)
			delegated = append(delegated, args...)
			return a.DelegateToControlPlane(delegated...)
		},
	}
}

// DelegateToControlPlane shells out to the configured vrooli binary.
func (a *ResourceApp) DelegateToControlPlane(args ...string) error {
	cliPath, err := a.resolveControlPlaneBinary()
	if err != nil {
		return err
	}
	cmd := exec.Command(cliPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if a.options.CommandRunner != nil {
		return a.options.CommandRunner(cmd)
	}
	return cmd.Run()
}

func (a *ResourceApp) resolveControlPlaneBinary() (string, error) {
	for _, key := range a.options.ControlPlaneEnvVars {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value, nil
		}
	}
	lookPath := exec.LookPath
	if a.options.LookPathFunc != nil {
		lookPath = a.options.LookPathFunc
	}
	path, err := lookPath("vrooli")
	if err != nil {
		return "", fmt.Errorf("locate vrooli control plane binary: %w", err)
	}
	return path, nil
}
