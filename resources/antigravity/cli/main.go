package main

import (
	"github.com/vrooli/vrooli/resources/antigravity/cli/internal/permissionscli"
	"github.com/vrooli/vrooli/resources/antigravity/cli/internal/upstream"
	"github.com/vrooli/vrooli/resources/internal/agentapp"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck/upstreamverb"
	agentinstall "github.com/vrooli/vrooli/packages/resource-agent-install"
)

const (
	appName = "antigravity"
	// upstreamPinnedVersion mirrors resource.json upstream_cli.version_pinned.
	// `upstream-check` uses it as the known-good baseline; drift warns.
	upstreamPinnedVersion = "1.0.13"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() { agentapp.Main(newApp) }

func newApp() (*cliapp.ResourceApp, error) {
	return agentapp.New(agentapp.Spec{
		Name:             appName,
		Description:      "Antigravity CLI resource CLI",
		Build:            agentapp.Build{Fingerprint: buildFingerprint, Timestamp: buildTimestamp, SourceRoot: buildSourceRoot},
		DataDir:          []string{".gemini"},
		NoModelDiscovery: true,
		Posture:          agentharness.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"Antigravity permissions and the projected PreToolUse hook require a live canary before being treated as enforced."}},
		Groups: func(agentinstall.Spec) ([]cliapp.SubcommandGroup, error) {
			return []cliapp.SubcommandGroup{
				// Antigravity is not on npm/GitHub releases — its latest version is
				// served as a per-platform JSON manifest from the auto-updater
				// service, so we override the upstream-check fetcher (see
				// internal/upstream).
				upstreamverb.Commands(upstream.Handlers(appName, upstreamPinnedVersion)),
				// Manage Antigravity's native `permissions` allow/deny/ask grants in
				// ~/.gemini/antigravity-cli/settings.json (see internal/permissions).
				permissionscli.HookCommands(permissionscli.Default(agentapp.Version, upstreamPinnedVersion)),
				permissionscli.Commands(permissionscli.Default(agentapp.Version, upstreamPinnedVersion)),
			}, nil
		},
	})
}
