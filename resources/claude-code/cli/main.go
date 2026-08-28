package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/vrooli/vrooli/resources/claude-code/cli/internal/permissionscli"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/cli-core/upstreamcheck/upstreamverb"
	agentinstall "github.com/vrooli/vrooli/packages/resource-agent-install"
)

const (
	appName    = "claude-code"
	appVersion = "0.1.0"
	// upstreamPinnedVersion mirrors resource.json upstream_cli.version_pinned.
	// `permissions doctor` warns when the installed claude CLI version
	// diverges from this string.
	upstreamPinnedVersion = "2.1.185"
)

func runClaude(args []string) error {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return fmt.Errorf("claude arguments are required")
	}
	path := os.Getenv("CLAUDE_CODE_PATH")
	if path == "" {
		path = "claude"
	}
	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

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
	commands := append(app.StandardLifecycleCommands(),
		cliapp.CommandGroup{Title: "Installation", Commands: []cliapp.Command{agentinstall.DirectInstallCommand(agentinstall.Spec{Binary: "claude", BinDir: filepath.Join(os.Getenv("HOME"), ".local", "bin"), DataDir: filepath.Join(os.Getenv("HOME"), ".claude"), Version: upstreamPinnedVersion, NPM: "@anthropic-ai/claude-code"})}},
		cliapp.CommandGroup{Title: "Execution", Commands: []cliapp.Command{{
			Name:        "run",
			Description: "Run the resource-owned Claude Code executable",
			Usage:       "resource-claude-code run -- <claude arguments>",
			Run:         runClaude,
		}}},
	)
	app.SetCommandsWithSubgroups(
		commands,
		[]cliapp.SubcommandGroup{
			agentharness.ModelDiscoveryCommands(agentharness.ModelDiscoveryConfig{Runner: appName, CatalogPath: agentharness.ResourceCatalogPath(appName)}),
			agentharness.CodingPolicyCommands(agentharness.CodingPolicyConfig{Runner: appName, CatalogPath: agentharness.ResourceCatalogPath(appName), Posture: agentharness.EnforcementPosture{Permissions: "hook_verified", Caveats: []string{"Claude native permission denials remain active; the source-controlled PreToolUse matcher is verified by data-only replay and a non-mutating live probe."}}}),
			permissionscli.HookCommands(permissionscli.Default(appVersion, upstreamPinnedVersion)),
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
