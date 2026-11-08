package main

import (
	"fmt"

	"github.com/vrooli/vrooli/resources/codex/cli/internal/configcli"
	"github.com/vrooli/vrooli/resources/codex/cli/internal/permissionscli"
	"github.com/vrooli/vrooli/resources/internal/agentapp"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	agentinstall "github.com/vrooli/vrooli/packages/resource-agent-install"
)

const (
	appName    = "codex"
	npmPackage = "@openai/codex"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() { agentapp.Main(newApp) }

func newApp() (*cliapp.ResourceApp, error) {
	return agentapp.New(agentapp.Spec{
		Name:               appName,
		Description:        "Codex resource CLI",
		Build:              agentapp.Build{Fingerprint: buildFingerprint, Timestamp: buildTimestamp, SourceRoot: buildSourceRoot},
		DataDir:            []string{".codex"},
		UpstreamSourceKind: "npm",
		UpstreamSourceID:   npmPackage,
		NativeSettings:     configcli.NativeSettings(),
		EnsureSteps:        []agentapp.EnsureStep{permissionscli.IntentMigrationStep()},
		Posture:            agentharness.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"Codex has no native seam for Vrooli's command-pattern rules; Vrooli records them beside the config and projects a PreToolUse hook, but hook firing requires a live canary on the installed Codex version."}},
		Groups: func(install agentinstall.Spec) ([]cliapp.SubcommandGroup, error) {
			pinnedVersion, err := install.PinnedVersion()
			if err != nil {
				return nil, fmt.Errorf("read Codex acquisition pin: %w", err)
			}
			return []cliapp.SubcommandGroup{
				configcli.Commands(configcli.Default()),
				permissionscli.HookCommands(permissionscli.Default(agentapp.Version, pinnedVersion)),
				permissionscli.Commands(permissionscli.Default(agentapp.Version, pinnedVersion)),
				agentapp.NPMUpstream(install, pinnedVersion, npmPackage),
			}, nil
		},
	})
}
