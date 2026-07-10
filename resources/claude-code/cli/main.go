package main

import (
	"fmt"
	"os"

	"resource-claude-code/cli/internal/permissionscli"

	"github.com/vrooli/cli-core/agentpolicy"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/cli-core/upstreamcheck/upstreamverb"
)

const (
	appName    = "claude-code"
	appVersion = "0.1.0"
	// upstreamPinnedVersion mirrors resource.json upstream_cli.version_pinned.
	// `permissions doctor` warns when the installed claude CLI version
	// diverges from this string.
	upstreamPinnedVersion = "2.1.185"
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
		Description:         "Claude Code resource CLI",
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
			agentpolicy.CodingPolicyCommands(agentpolicy.CodingPolicyConfig{Runner: appName, CatalogPath: agentpolicy.ResourceCatalogPath(appName), Posture: agentpolicy.EnforcementPosture{Permissions: "hook_backed", Caveats: []string{"Claude native permission denials are backed by a PreToolUse Bash hook."}}}),
			permissionscli.Commands(permissionscli.Default(appVersion, upstreamPinnedVersion)),
			upstreamverb.Commands(upstreamcheck.Default(upstreamcheck.Config{
				DisplayName:   appName,
				InstalledCmd:  []string{"claude", "--version"},
				PinnedVersion: upstreamPinnedVersion,
				SourceKind:    upstreamcheck.SourceNPM,
				SourceID:      "@anthropic-ai/claude-code",
			})),
		},
	)
	return app, nil
}
