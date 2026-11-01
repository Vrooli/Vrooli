package main

import (
	"github.com/vrooli/vrooli/resources/grok/cli/internal/permissionscli"
	"github.com/vrooli/vrooli/resources/grok/cli/internal/upstream"
	"github.com/vrooli/vrooli/resources/internal/agentapp"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck/upstreamverb"
	agentinstall "github.com/vrooli/vrooli/packages/resource-agent-install"
)

const (
	appName = "grok"
	// upstreamPinnedVersion mirrors resource.json upstream_cli.version_pinned.
	// `upstream-check` uses it as the pinned baseline; drift warns.
	upstreamPinnedVersion = "0.2.72"
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
		Description: "Grok Build resource CLI",
		Build:       agentapp.Build{Fingerprint: buildFingerprint, Timestamp: buildTimestamp, SourceRoot: buildSourceRoot},
		DataDir:     []string{".grok"},
		Posture:     agentharness.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"Grok native permission rules remain active; verify the installed Grok version with a PreToolUse runner canary before treating the portable hook as enforced."}},
		Groups: func(agentinstall.Spec) ([]cliapp.SubcommandGroup, error) {
			return []cliapp.SubcommandGroup{
				// Grok is not on npm/GitHub releases — its latest version is a bare
				// text pointer at https://x.ai/cli/<channel>, so we override the
				// upstream-check fetcher (see internal/upstream).
				upstreamverb.Commands(upstream.Handlers(appName, upstreamPinnedVersion)),
				// Manage Grok's native [permission] rules + PreToolUse deny hook.
				permissionscli.HookCommands(permissionscli.Default(agentapp.Version, upstreamPinnedVersion)),
				permissionscli.Commands(permissionscli.Default(agentapp.Version, upstreamPinnedVersion)),
			}, nil
		},
	})
}
