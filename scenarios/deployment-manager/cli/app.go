package main

import (
	"errors"
	"fmt"
	"time"

	"deployment-manager/cli/bundles"
	"deployment-manager/cli/cmdutil"
	"deployment-manager/cli/deployments"
	"deployment-manager/cli/overview"
	"deployment-manager/cli/profiles"
	"deployment-manager/cli/signing"
	"deployment-manager/cli/swaps"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "deployment-manager"
	appVersion     = "1.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

// App wires the cli-core scaffolding to the scenario-specific commands.
type App struct {
	core        *cliapp.ScenarioApp
	api         *cliutil.APIClient
	http        *cliutil.HTTPClient
	overview    *overview.Commands
	profiles    *profiles.Commands
	swaps       *swaps.Commands
	deployments *deployments.Commands
	bundles     *bundles.Commands
	signing     *signing.Commands
}

// NewApp constructs the CLI application.
func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "Deployment Manager CLI",
		DefaultAPIBase:     defaultAPIBase,
		APIEnvVars:         env.APIEnvVars,
		APIPortEnvVars:     env.APIPortEnvVars,
		APIPortDetector:    cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:   env.ConfigDirEnvVars,
		SourceRootEnvVars:  env.SourceRootEnvVars,
		TokenEnvVars:       env.TokenEnvVars,
		HTTPTimeoutEnvVars: env.HTTPTimeoutEnvVars,
		TokenKeys:          []string{"token", "api_token"},
		BuildFingerprint:   buildFingerprint,
		BuildTimestamp:     buildTimestamp,
		BuildSourceRoot:    buildSourceRoot,
		DefaultHTTPTimeout: 10 * time.Minute,
		AllowAnonymous:     true,
	})
	if err != nil {
		return nil, err
	}

	app := &App{
		core:        core,
		api:         core.APIClient,
		http:        core.HTTPClient,
		overview:    overview.New(core.APIClient),
		profiles:    profiles.New(core.APIClient),
		swaps:       swaps.New(core.APIClient),
		deployments: deployments.New(core.APIClient),
		bundles:     bundles.New(core.APIClient),
		signing:     signing.New(core.APIClient),
	}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

// Run executes the CLI with provided arguments.
func (a *App) Run(args []string) error {
	remaining := applyGlobalFormat(args)
	return a.core.CLI.Run(remaining)
}

func (a *App) runProfile(args []string) error {
	if len(args) == 0 {
		return errors.New("profile subcommand is required")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "create":
		return a.profiles.Create(rest)
	case "list":
		return a.profiles.List(rest)
	case "show":
		return a.profiles.Show(rest)
	case "delete":
		return a.profiles.Delete(rest)
	case "export":
		return a.profiles.Export(rest)
	case "import":
		return a.profiles.Import(rest)
	case "update":
		return a.profiles.Update(rest)
	case "set":
		return a.profiles.Set(rest)
	case "swap":
		return a.profiles.Swap(rest)
	case "versions":
		return a.profiles.Versions(rest)
	case "analyze":
		return a.profiles.Analyze(rest)
	case "save":
		return a.profiles.Save(rest)
	case "diff":
		return a.profiles.Diff(rest)
	case "rollback":
		return a.profiles.Rollback(rest)
	default:
		return fmt.Errorf("unknown profile subcommand: %s", sub)
	}
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	overview := cliapp.CommandGroup{
		Title: "Overview",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API and dependency health", Run: a.overview.Status},
			{Name: "analyze", NeedsAPI: true, Description: "Analyze scenario dependencies", Run: a.overview.Analyze},
			{Name: "fitness", NeedsAPI: true, Description: "Calculate platform fitness scores", Run: a.overview.Fitness},
		},
	}

	profiles := cliapp.CommandGroup{
		Title: "Profiles",
		Commands: []cliapp.Command{
			{Name: "profiles", NeedsAPI: true, Description: "List deployment profiles", Run: a.profiles.List},
			{Name: "profile", NeedsAPI: true, Description: "Profile management commands", Run: a.runProfile},
		},
	}

	swaps := cliapp.CommandGroup{
		Title: "Swaps",
		Commands: []cliapp.Command{
			{Name: "swaps", NeedsAPI: true, Description: "Dependency swap tools", Run: a.swaps.Run},
		},
	}

	deployments := cliapp.CommandGroup{
		Title: "Deployments",
		Commands: []cliapp.Command{
			{Name: "deploy", NeedsAPI: true, Description: "Deploy a profile", Run: a.deployments.Deploy},
			{Name: "deploy-desktop", NeedsAPI: true, Description: "Orchestrate complete bundled desktop deployment", Run: a.deployments.DeployDesktop},
			{Name: "deployment", NeedsAPI: true, Description: "Manage deployment records", Run: a.deployments.Deployment},
			{Name: "build", NeedsAPI: true, Description: "Cross-compile service binaries", Run: a.deployments.Build},
			{Name: "logs", NeedsAPI: true, Description: "Fetch deployment logs", Run: a.deployments.Logs},
			{Name: "packagers", NeedsAPI: true, Description: "Packager discovery", Run: a.deployments.Packagers},
			{Name: "package", NeedsAPI: true, Description: "Package a profile", Run: a.deployments.PackageProfile},
			{Name: "validate", NeedsAPI: true, Description: "Validate deployment profile", Run: a.deployments.Validate},
			{Name: "estimate-cost", NeedsAPI: true, Description: "Estimate deployment costs", Run: a.deployments.EstimateCost},
			{Name: "bundle", NeedsAPI: true, Description: "Bundle manifest operations (assemble, export, validate)", Run: a.bundles.Run},
		},
	}

	secrets := cliapp.CommandGroup{
		Title: "Secrets",
		Commands: []cliapp.Command{
			{Name: "secrets", NeedsAPI: true, Description: "Secret discovery and templates", Run: a.profiles.Secrets},
		},
	}

	signing := cliapp.CommandGroup{
		Title: "Code Signing",
		Commands: []cliapp.Command{
			{Name: "signing", NeedsAPI: true, Description: "Configure code signing for deployments", Run: a.signing.Run},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{overview, profiles, swaps, deployments, secrets, signing, config}
}

// applyGlobalFormat consumes leading global format flags (--json, --format <fmt>)
// so they do not conflict with per-command flag parsing.
func applyGlobalFormat(args []string) []string {
	if len(args) == 0 {
		return args
	}
	remaining := []string{}
	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if arg == "--json" {
			cmdutil.SetGlobalFormat("json")
			continue
		}
		if arg == "--format" {
			if i+1 < len(args) {
				cmdutil.SetGlobalFormat(args[i+1])
				skipNext = true
			}
			continue
		}
		remaining = append(remaining, arg)
		// stop scanning once we hit the command name (first non-global flag token)
		break
	}
	// append the rest (command + subargs)
	if len(remaining) > 0 {
		idx := indexOf(args, remaining[0])
		if idx >= 0 && idx+1 < len(args) {
			remaining = append(remaining, args[idx+1:]...)
		}
	}
	return remaining
}

func indexOf(slice []string, target string) int {
	for i, v := range slice {
		if v == target {
			return i
		}
	}
	return -1
}
