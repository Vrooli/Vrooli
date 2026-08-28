package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/resources/opencode/cli/internal/configcli"
	"github.com/vrooli/vrooli/resources/opencode/cli/internal/permissionscli"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/cli-core/upstreamcheck/upstreamverb"
	agentinstall "github.com/vrooli/vrooli/packages/resource-agent-install"
)

const (
	appName    = "opencode"
	appVersion = "0.1.0"
	// upstreamPinnedVersion mirrors resource.json upstream_cli.version_pinned.
	// `permissions doctor` and `upstream-check` use it as the pinned baseline.
	upstreamPinnedVersion = "1.17.9"
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
		Description:         "OpenCode resource CLI",
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
		append(app.StandardLifecycleCommands(), cliapp.CommandGroup{Title: "Installation", Commands: []cliapp.Command{agentinstall.DirectInstallCommand(agentinstall.Spec{Binary: "opencode", BinDir: filepath.Join(os.Getenv("HOME"), ".local", "bin"), DataDir: filepath.Join(os.Getenv("HOME"), ".local", "share", "opencode"), Version: upstreamPinnedVersion, URLTemplate: "https://github.com/sst/opencode/releases/download/v${version}/opencode-${os}-${arch}.tar.gz", ArchiveEntry: "opencode"})}}),
		[]cliapp.SubcommandGroup{
			agentharness.ModelDiscoveryCommands(agentharness.ModelDiscoveryConfig{Runner: appName, CatalogPath: agentharness.ResourceCatalogPath(appName)}),
			agentharness.CodingPolicyCommands(agentharness.CodingPolicyConfig{Runner: appName, CatalogPath: agentharness.ResourceCatalogPath(appName), Posture: agentharness.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"OpenCode native permission rules are projected alongside tool.execute.before; plugin firing and refusal require a live canary on the installed version."}}}),
			configcli.Commands(configcli.Default()),
			permissionscli.HookCommands(permissionscli.Default(appVersion, upstreamPinnedVersion)),
			permissionscli.Commands(permissionscli.Default(appVersion, upstreamPinnedVersion)),
			upstreamverb.Commands(upstreamcheck.Default(upstreamcheck.Config{
				DisplayName:   appName,
				InstalledCmd:  []string{"opencode", "--version"},
				PinnedVersion: upstreamPinnedVersion,
				// npm is the reliable latest-version source: the GitHub
				// releases/latest API returns a null tag for sst/opencode,
				// while the npm dist-tag `latest` tracks the shipped binary.
				SourceKind: upstreamcheck.SourceNPM,
				SourceID:   "opencode-ai",
			})),
		},
	)
	return app, nil
}
