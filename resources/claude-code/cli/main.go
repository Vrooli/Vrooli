package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/vrooli/vrooli/resources/claude-code/cli/internal/configcli"
	"github.com/vrooli/vrooli/resources/claude-code/cli/internal/permissions"
	"github.com/vrooli/vrooli/resources/claude-code/cli/internal/permissionscli"
	"github.com/vrooli/vrooli/resources/claude-code/cli/internal/statusline"
	"github.com/vrooli/vrooli/resources/internal/agentapp"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	agentinstall "github.com/vrooli/vrooli/packages/resource-agent-install"
)

const (
	appName = "claude-code"
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

func main() { agentapp.Main(newApp) }

func newApp() (*cliapp.ResourceApp, error) {
	settings := &permissions.Adapter{}
	if adapter, adapterErr := permissions.DefaultAdapter(); adapterErr == nil {
		settings = adapter
	}
	statusLine := statusline.DefaultHandlers(settings.SettingsPath, settings.HookStateDir)
	return agentapp.New(agentapp.Spec{
		Name:        appName,
		Description: "Claude Code resource CLI",
		Build:       agentapp.Build{Fingerprint: buildFingerprint, Timestamp: buildTimestamp, SourceRoot: buildSourceRoot},
		DataDir:     []string{".claude"},
		Posture:     agentharness.EnforcementPosture{Permissions: "hook_verified", Caveats: []string{"Claude native permission denials remain active; the source-controlled PreToolUse matcher is verified by data-only replay and a non-mutating live probe."}},
		WrapInstall: func(install cliapp.Command) cliapp.Command { return installWithStatusLine(install, statusLine) },
		CommandGroups: []cliapp.CommandGroup{{Title: "Execution", Commands: []cliapp.Command{{
			Name:        "run",
			Description: "Run the resource-owned Claude Code executable",
			Usage:       "resource-claude-code run -- <claude arguments>",
			Run:         runClaude,
		}}}},
		Groups: func(install agentinstall.Spec) ([]cliapp.SubcommandGroup, error) {
			return []cliapp.SubcommandGroup{
				configcli.Commands(configcli.Default()),
				statusline.Commands(statusLine),
				permissionscli.HookCommands(permissionscli.Default(agentapp.Version, upstreamPinnedVersion)),
				permissionscli.Commands(permissionscli.Default(agentapp.Version, upstreamPinnedVersion)),
				agentapp.NPMUpstream(install, upstreamPinnedVersion, "@anthropic-ai/claude-code"),
			}, nil
		},
	})
}

// installWithStatusLine makes the resource install (`vrooli resource install
// claude-code` runs install-direct) also reconcile the statusline quota feed,
// so every host that installs the resource gets the feed reproducibly.
func installWithStatusLine(install cliapp.Command, feed *statusline.Handlers) cliapp.Command {
	direct := install.Run
	install.Description += "; reconcile the statusline quota feed"
	install.Run = func(args []string) error {
		if err := direct(args); err != nil {
			return err
		}
		return feed.Install()
	}
	return install
}
