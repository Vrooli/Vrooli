package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/resources/antigravity/cli/internal/permissionscli"
	"github.com/vrooli/vrooli/resources/antigravity/cli/internal/upstream"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck/upstreamverb"
	agentinstall "github.com/vrooli/vrooli/packages/resource-agent-install"
)

const (
	appName    = "antigravity"
	appVersion = "0.1.0"
	// upstreamPinnedVersion mirrors resource.json upstream_cli.version_pinned.
	// `upstream-check` uses it as the known-good baseline; drift warns.
	upstreamPinnedVersion = "1.0.13"
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
		Description:         "Antigravity CLI resource CLI",
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
		append(app.StandardLifecycleCommands(), cliapp.CommandGroup{Title: "Installation", Commands: []cliapp.Command{agentinstall.DirectInstallCommand(agentinstall.Spec{Binary: "agy", BinDir: filepath.Join(os.Getenv("HOME"), ".local", "bin"), DataDir: filepath.Join(os.Getenv("HOME"), ".gemini"), Version: upstreamPinnedVersion, URLTemplate: "https://antigravity-cli-auto-updater-974169037036.us-central1.run.app/artifacts/${os}_${arch}.tar.gz", ArchiveEntry: "antigravity"})}}),
		[]cliapp.SubcommandGroup{
			agentharness.CodingPolicyCommands(agentharness.CodingPolicyConfig{Runner: appName, CatalogPath: agentharness.ResourceCatalogPath(appName), Posture: agentharness.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"Antigravity permissions and the projected PreToolUse hook require a live canary before being treated as enforced."}}}),
			// Antigravity is not on npm/GitHub releases — its latest version is
			// served as a per-platform JSON manifest from the auto-updater
			// service, so we override the upstream-check fetcher (see
			// internal/upstream).
			upstreamverb.Commands(upstream.Handlers(appName, upstreamPinnedVersion)),
			// Manage Antigravity's native `permissions` allow/deny/ask grants in
			// ~/.gemini/antigravity-cli/settings.json (see internal/permissions).
			permissionscli.HookCommands(permissionscli.Default(appVersion, upstreamPinnedVersion)),
			permissionscli.Commands(permissionscli.Default(appVersion, upstreamPinnedVersion)),
		},
	)
	return app, nil
}
