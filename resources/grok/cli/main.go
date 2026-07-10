package main

import (
	"fmt"
	"os"

	"resource-grok/cli/internal/permissionscli"
	"resource-grok/cli/internal/upstream"

	"github.com/vrooli/cli-core/agentpolicy"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck/upstreamverb"
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
		app.StandardLifecycleCommands(),
		[]cliapp.SubcommandGroup{
			agentpolicy.CodingPolicyCommands(agentpolicy.CodingPolicyConfig{Runner: appName, CatalogPath: agentpolicy.ResourceCatalogPath(appName), Posture: agentpolicy.EnforcementPosture{Permissions: "hook_backed", Caveats: []string{"Grok native permission rules are supplemented by a PreToolUse Bash hook."}}}),
			// Grok is not on npm/GitHub releases — its latest version is a bare
			// text pointer at https://x.ai/cli/<channel>, so we override the
			// upstream-check fetcher (see internal/upstream).
			upstreamverb.Commands(upstream.Handlers(appName, upstreamPinnedVersion)),
			// Manage Grok's native [permission] rules + PreToolUse deny hook.
			permissionscli.Commands(permissionscli.Default(appVersion, upstreamPinnedVersion)),
		},
	)
	return app, nil
}
