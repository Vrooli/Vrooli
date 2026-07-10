package main

import (
	"fmt"
	"os"

	"resource-opencode/cli/internal/configcli"
	"resource-opencode/cli/internal/permissionscli"

	"github.com/vrooli/cli-core/agentpolicy"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/cli-core/upstreamcheck/upstreamverb"
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
		app.StandardLifecycleCommands(),
		[]cliapp.SubcommandGroup{
			agentpolicy.CodingPolicyCommands(agentpolicy.CodingPolicyConfig{Runner: appName, CatalogPath: agentpolicy.ResourceCatalogPath(appName), Posture: agentpolicy.EnforcementPosture{Permissions: "native"}}),
			configcli.Commands(configcli.Default()),
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
