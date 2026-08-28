package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/resources/grok/cli/internal/permissionscli"
	"github.com/vrooli/vrooli/resources/grok/cli/internal/upstream"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck/upstreamverb"
	agentinstall "github.com/vrooli/vrooli/packages/resource-agent-install"
)

const (
	appName    = "grok"
	appVersion = "0.1.0"
	// upstreamPinnedVersion mirrors resource.json upstream_cli.version_pinned.
	// `upstream-check` uses it as the pinned baseline; drift warns.
	upstreamPinnedVersion = "0.2.72"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() {
	app, err := newApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.CLI.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newApp() (*cliapp.ResourceApp, error) {
	env := cliapp.StandardResourceEnv(appName, cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name:                appName,
		Version:             appVersion,
		Description:         "Grok Build resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	app.SetCommandsWithSubgroups(
		append(app.StandardLifecycleCommands(), cliapp.CommandGroup{Title: "Installation", Commands: []cliapp.Command{agentinstall.DirectInstallCommand(agentinstall.Spec{Binary: "grok", BinDir: filepath.Join(os.Getenv("HOME"), ".local", "bin"), DataDir: filepath.Join(os.Getenv("HOME"), ".grok"), Version: upstreamPinnedVersion, URLTemplate: "https://x.ai/cli/grok-${version}-${os}-${arch}"})}}),
		[]cliapp.SubcommandGroup{
			agentharness.ModelDiscoveryCommands(agentharness.ModelDiscoveryConfig{Runner: appName, CatalogPath: agentharness.ResourceCatalogPath(appName)}),
			agentharness.CodingPolicyCommands(agentharness.CodingPolicyConfig{Runner: appName, CatalogPath: agentharness.ResourceCatalogPath(appName), Posture: agentharness.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"Grok native permission rules remain active; verify the installed Grok version with a PreToolUse runner canary before treating the portable hook as enforced."}}}),
			// Grok is not on npm/GitHub releases — its latest version is a bare
			// text pointer at https://x.ai/cli/<channel>, so we override the
			// upstream-check fetcher (see internal/upstream).
			upstreamverb.Commands(upstream.Handlers(appName, upstreamPinnedVersion)),
			// Manage Grok's native [permission] rules + PreToolUse deny hook.
			permissionscli.HookCommands(permissionscli.Default(appVersion, upstreamPinnedVersion)),
			permissionscli.Commands(permissionscli.Default(appVersion, upstreamPinnedVersion)),
		},
	)
	return app, nil
}
