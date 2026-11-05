package main

import (
	"github.com/vrooli/vrooli/resources/internal/agentapp"
	"github.com/vrooli/vrooli/resources/opencode/cli/internal/configcli"
	"github.com/vrooli/vrooli/resources/opencode/cli/internal/ensurecli"
	"github.com/vrooli/vrooli/resources/opencode/cli/internal/permissionscli"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	agentinstall "github.com/vrooli/vrooli/packages/resource-agent-install"
)

const (
	appName = "opencode"
	// upstreamPinnedVersion mirrors resource.json upstream_cli.version_pinned.
	// `permissions doctor` and `upstream-check` use it as the pinned baseline.
	upstreamPinnedVersion = "1.18.30"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() { agentapp.Main(newApp) }

func newApp() (*cliapp.ResourceApp, error) {
	return agentapp.New(agentapp.Spec{
		Name:        appName,
		Description: "OpenCode resource CLI",
		Build:       agentapp.Build{Fingerprint: buildFingerprint, Timestamp: buildTimestamp, SourceRoot: buildSourceRoot},
		DataDir:     []string{".local", "share", "opencode"},
		WrapInstall: func(install cliapp.Command) cliapp.Command {
			run := install.Run
			install.Run = func(args []string) error {
				if err := run(args); err != nil {
					return err
				}
				return configcli.Default().Ensure(nil)
			}
			return install
		},
		InstallCommands: []cliapp.Command{ensurecli.Commands(ensurecli.Default())},
		Posture:         agentharness.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"OpenCode native permission rules are projected alongside tool.execute.before; plugin firing and refusal require a live canary on the installed version."}},
		Groups: func(install agentinstall.Spec) ([]cliapp.SubcommandGroup, error) {
			return []cliapp.SubcommandGroup{
				configcli.Commands(configcli.Default()),
				permissionscli.HookCommands(permissionscli.Default(agentapp.Version, upstreamPinnedVersion)),
				permissionscli.Commands(permissionscli.Default(agentapp.Version, upstreamPinnedVersion)),
				// npm is the reliable latest-version source: the GitHub
				// releases/latest API returns a null tag for sst/opencode,
				// while the npm dist-tag `latest` tracks the shipped binary.
				agentapp.NPMUpstream(install, upstreamPinnedVersion, "opencode-ai"),
			}, nil
		},
	})
}
