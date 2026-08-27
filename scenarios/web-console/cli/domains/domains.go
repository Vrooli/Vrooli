package domains

import (
	"web-console/cli/domains/ai"
	"web-console/cli/domains/capabilities"
	"web-console/cli/domains/conversation"
	"web-console/cli/domains/events"
	filepreview "web-console/cli/domains/file_preview"
	"web-console/cli/domains/hooks"
	"web-console/cli/domains/machines"
	"web-console/cli/domains/metrics"
	"web-console/cli/domains/session"
	"web-console/cli/domains/settings"
	"web-console/cli/domains/shortcuts"
	targets "web-console/cli/domains/targets"
	"web-console/cli/domains/terminal"
	"web-console/cli/domains/workspace"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Every proto-bound domain now
// flows through SubcommandGroups (built from cli/manifest.json via
// cliapp.LoadFromManifest), so there are no code-built flat groups today.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups. Proto-bound
// domains (ai, capabilities, conversation, events, machine, metrics,
// session, settings, shortcuts, target, workspace) are built from the embedded
// cli/manifest.json — the single source of truth for the CLI surface — by
// each domain's Register(core, manifest) calling cliapp.LoadFromManifest.
// The capabilities/events/metrics groups expose a single command whose name
// matches the group, so they set DefaultSubcommand to preserve the flat
// `web-console <group>` invocation.
//
// terminal stays code-registered: its commands (screen, send-text,
// send-keys, wait-idle) can't be expressed as one-command-per-method
// manifest bindings (send-text and send-keys both map to SendInput), so its
// TerminalService methods are declared in cli/manifest.json's omitted[].
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	aiGroup, err := ai.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	capabilitiesGroup, err := capabilities.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	conversationGroup, err := conversation.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	filePreviewGroup, err := filepreview.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	eventsGroup, err := events.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	metricsGroup, err := metrics.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	sessionGroup, err := session.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	targetGroup, err := targets.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	machineGroup, err := machines.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	settingsGroup, err := settings.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	shortcutsGroup, err := shortcuts.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	workspaceGroup, err := workspace.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	return []cliapp.SubcommandGroup{
		sessionGroup,
		targetGroup,
		machineGroup,
		terminal.Register(core),
		hooks.Register(),
		workspaceGroup,
		settingsGroup,
		shortcutsGroup,
		aiGroup,
		conversationGroup,
		filePreviewGroup,
		eventsGroup,
		metricsGroup,
		capabilitiesGroup,
	}, nil
}
