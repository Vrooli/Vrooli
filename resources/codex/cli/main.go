package main

import (
	"fmt"
	"os"

	"resource-codex/cli/internal/permissionscli"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/cli-core/upstreamcheck/upstreamverb"
)

const (
	appName    = "codex"
	appVersion = "0.1.0"
	// upstreamPinnedVersion mirrors resource.json upstream_cli.version_pinned.
	// `permissions doctor` warns when the installed codex CLI version
	// diverges from this string.
	upstreamPinnedVersion = "0.141.0"
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
		Description:         "Codex resource CLI",
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
			permissionscli.Commands(permissionscli.Default(appVersion, upstreamPinnedVersion)),
			upstreamverb.Commands(upstreamcheck.Default(upstreamcheck.Config{
				DisplayName:   appName,
				InstalledCmd:  []string{"codex", "--version"},
				PinnedVersion: upstreamPinnedVersion,
				SourceKind:    upstreamcheck.SourceNPM,
				SourceID:      "@openai/codex",
			})),
		},
	)
	return app, nil
}
