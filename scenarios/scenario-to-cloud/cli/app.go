package main

import (
	"fmt"
	"io"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"scenario-to-cloud/cli/bundle"
	"scenario-to-cloud/cli/deployment"
	"scenario-to-cloud/cli/edge"
	"scenario-to-cloud/cli/inspect"
	"scenario-to-cloud/cli/manifest"
	"scenario-to-cloud/cli/preflight"
	"scenario-to-cloud/cli/process"
	"scenario-to-cloud/cli/redeploy"
	"scenario-to-cloud/cli/scenario"
	"scenario-to-cloud/cli/secrets"
	"scenario-to-cloud/cli/ssh"
	"scenario-to-cloud/cli/status"
	"scenario-to-cloud/cli/task"
	"scenario-to-cloud/cli/vps"
)

const (
	appName        = "scenario-to-cloud"
	appVersion     = "0.2.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

// App is the main CLI application container.
type App struct {
	core *cliapp.ScenarioApp

	// Domain clients
	statusClient     *status.Client
	manifestClient   *manifest.Client
	bundleClient     *bundle.Client
	deploymentClient *deployment.Client
	preflightClient  *preflight.Client
	vpsClient        *vps.Client
	inspectClient    *inspect.Client
	processClient    *process.Client
	edgeClient       *edge.Client
	sshClient        *ssh.Client
	secretsClient    *secrets.Client
	scenarioClient   *scenario.Client
	taskClient       *task.Client
}

// NewApp creates a new CLI application instance.
func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "scenario-to-cloud CLI",
		DefaultAPIBase:     defaultAPIBase,
		APIEnvVars:         env.APIEnvVars,
		APIPortEnvVars:     env.APIPortEnvVars,
		APIPortDetector:    cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:   env.ConfigDirEnvVars,
		SourceRootEnvVars:  env.SourceRootEnvVars,
		TokenEnvVars:       env.TokenEnvVars,
		HTTPTimeoutEnvVars: env.HTTPTimeoutEnvVars,
		BuildFingerprint:   buildFingerprint,
		BuildTimestamp:     buildTimestamp,
		BuildSourceRoot:    buildSourceRoot,
		DefaultHTTPTimeout: 2 * time.Minute,
		AllowAnonymous:     true,
	})
	if err != nil {
		return nil, err
	}

	app := &App{
		core:             core,
		statusClient:     status.NewClient(core.APIClient),
		manifestClient:   manifest.NewClient(core.APIClient),
		bundleClient:     bundle.NewClient(core.APIClient),
		deploymentClient: deployment.NewClient(core.APIClient),
		preflightClient:  preflight.NewClient(core.APIClient),
		vpsClient:        vps.NewClient(core.APIClient),
		inspectClient:    inspect.NewClient(core.APIClient),
		processClient:    process.NewClient(core.APIClient),
		edgeClient:       edge.NewClient(core.APIClient),
		sshClient:        ssh.NewClient(core.APIClient),
		secretsClient:    secrets.NewClient(core.APIClient),
		scenarioClient:   scenario.NewClient(core.APIClient),
		taskClient:       task.NewClient(core.APIClient),
	}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	// Health commands
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{
				Name:        "status",
				NeedsAPI:    true,
				Description: "Check API and dependency health",
				Run:         func(args []string) error { return status.Run(a.statusClient, args) },
			},
		},
	}

	// Core domain commands (new screaming architecture)
	domains := cliapp.CommandGroup{
		Title: "Domains",
		Commands: []cliapp.Command{
			{
				Name:        "manifest",
				NeedsAPI:    true,
				Description: "Manifest operations (validate)",
				Run:         func(args []string) error { return manifest.Run(a.manifestClient, args) },
			},
			{
				Name:        "bundle",
				NeedsAPI:    true,
				Description: "Bundle operations (build)",
				Run:         func(args []string) error { return bundle.Run(a.bundleClient, args) },
			},
			{
				Name:        "deployment",
				NeedsAPI:    true,
				Description: "Deployment lifecycle (create, list, get, delete, execute, start, stop, history)",
				Run:         func(args []string) error { return deployment.Run(a.deploymentClient, args) },
			},
			{
				Name:        "redeploy",
				NeedsAPI:    true,
				Description: "Create/update and execute deployment in one command",
				Run:         func(args []string) error { return redeploy.Run(a.deploymentClient, args) },
			},
			{
				Name:        "preflight",
				NeedsAPI:    true,
				Description: "Preflight checks (run)",
				Run:         func(args []string) error { return preflight.Run(a.preflightClient, args) },
			},
			{
				Name:        "vps",
				NeedsAPI:    true,
				Description: "VPS operations (setup, deploy)",
				Run:         func(args []string) error { return vps.Run(a.vpsClient, args) },
			},
			{
				Name:        "inspect",
				NeedsAPI:    true,
				Description: "Inspection operations (status, live, drift, logs, files)",
				Run:         func(args []string) error { return inspect.Run(a.inspectClient, args) },
			},
			{
				Name:        "process",
				NeedsAPI:    true,
				Description: "Process control (kill, restart, control, vps-action)",
				Run:         func(args []string) error { return process.Run(a.processClient, args) },
			},
			{
				Name:        "edge",
				NeedsAPI:    true,
				Description: "Edge & TLS management (dns-check, dns-records, caddy, tls, tls-renew)",
				Run:         func(args []string) error { return edge.Run(a.edgeClient, args) },
			},
			{
				Name:        "ssh",
				NeedsAPI:    true,
				Description: "SSH key management (keys, generate, delete, test, copy-key)",
				Run:         func(args []string) error { return ssh.Run(a.sshClient, args) },
			},
			{
				Name:        "secrets",
				NeedsAPI:    true,
				Description: "Secrets management (get)",
				Run:         func(args []string) error { return secrets.Run(a.secretsClient, args) },
			},
			{
				Name:        "scenario",
				NeedsAPI:    true,
				Description: "Scenario discovery (list, ports, deps)",
				Run:         func(args []string) error { return scenario.Run(a.scenarioClient, args) },
			},
			{
				Name:        "task",
				NeedsAPI:    true,
				Description: "AI task management (create, list, get, stop, agent-status)",
				Run:         func(args []string) error { return task.Run(a.taskClient, args) },
			},
		},
	}

	// Backward-compatible aliases (deprecated, will be removed in v1.0)
	aliases := cliapp.CommandGroup{
		Title: "Aliases (deprecated)",
		Commands: []cliapp.Command{
			{
				Name:        "manifest-validate",
				NeedsAPI:    true,
				Description: "(deprecated: use 'manifest validate')",
				Run: func(args []string) error {
					fmt.Fprintln(deprecationWriter, "Warning: 'manifest-validate' is deprecated. Use 'manifest validate' instead.")
					return manifest.Run(a.manifestClient, append([]string{"validate"}, args...))
				},
			},
			{
				Name:        "plan",
				NeedsAPI:    true,
				Description: "(deprecated: use 'deployment plan')",
				Run: func(args []string) error {
					fmt.Fprintln(deprecationWriter, "Warning: 'plan' is deprecated. Use 'deployment plan' instead.")
					return deployment.Run(a.deploymentClient, append([]string{"plan"}, args...))
				},
			},
			{
				Name:        "bundle-build",
				NeedsAPI:    true,
				Description: "(deprecated: use 'bundle build')",
				Run: func(args []string) error {
					fmt.Fprintln(deprecationWriter, "Warning: 'bundle-build' is deprecated. Use 'bundle build' instead.")
					return bundle.Run(a.bundleClient, append([]string{"build"}, args...))
				},
			},
			{
				Name:        "vps-setup-plan",
				NeedsAPI:    true,
				Description: "(deprecated: use 'vps setup plan')",
				Run: func(args []string) error {
					fmt.Fprintln(deprecationWriter, "Warning: 'vps-setup-plan' is deprecated. Use 'vps setup plan' instead.")
					return vps.Run(a.vpsClient, append([]string{"setup", "plan"}, args...))
				},
			},
			{
				Name:        "vps-setup-apply",
				NeedsAPI:    true,
				Description: "(deprecated: use 'vps setup apply')",
				Run: func(args []string) error {
					fmt.Fprintln(deprecationWriter, "Warning: 'vps-setup-apply' is deprecated. Use 'vps setup apply' instead.")
					return vps.Run(a.vpsClient, append([]string{"setup", "apply"}, args...))
				},
			},
			{
				Name:        "vps-deploy-plan",
				NeedsAPI:    true,
				Description: "(deprecated: use 'vps deploy plan')",
				Run: func(args []string) error {
					fmt.Fprintln(deprecationWriter, "Warning: 'vps-deploy-plan' is deprecated. Use 'vps deploy plan' instead.")
					return vps.Run(a.vpsClient, append([]string{"deploy", "plan"}, args...))
				},
			},
			{
				Name:        "vps-deploy-apply",
				NeedsAPI:    true,
				Description: "(deprecated: use 'vps deploy apply')",
				Run: func(args []string) error {
					fmt.Fprintln(deprecationWriter, "Warning: 'vps-deploy-apply' is deprecated. Use 'vps deploy apply' instead.")
					return vps.Run(a.vpsClient, append([]string{"deploy", "apply"}, args...))
				},
			},
			{
				Name:        "vps-inspect-plan",
				NeedsAPI:    true,
				Description: "(deprecated: use 'inspect plan')",
				Run: func(args []string) error {
					fmt.Fprintln(deprecationWriter, "Warning: 'vps-inspect-plan' is deprecated. Use 'inspect plan' instead.")
					return inspect.Run(a.inspectClient, append([]string{"plan"}, args...))
				},
			},
			{
				Name:        "vps-inspect-apply",
				NeedsAPI:    true,
				Description: "(deprecated: use 'inspect status')",
				Run: func(args []string) error {
					fmt.Fprintln(deprecationWriter, "Warning: 'vps-inspect-apply' is deprecated. Use 'inspect status' instead.")
					return inspect.Run(a.inspectClient, append([]string{"status"}, args...))
				},
			},
		},
	}

	// Configuration commands
	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, domains, aliases, config}
}

// Run executes the CLI with the given arguments.
func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

// deprecationWriter is where deprecation warnings are written.
// Set to io.Discard during tests to suppress output.
var deprecationWriter = io.Discard
