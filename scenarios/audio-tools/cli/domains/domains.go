package domains

import (
	"audio-tools/cli/domains/audio"
	"audio-tools/cli/domains/diagnostics"
	"audio-tools/cli/domains/health"
	"audio-tools/cli/domains/provider"
	"audio-tools/cli/domains/settings"
	"audio-tools/cli/domains/stt"
	"audio-tools/cli/domains/summarize"
	"audio-tools/cli/domains/tts"
	"audio-tools/cli/domains/usage"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups from domain packages.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates the audio-tools domain command groups.
// Each domain mirrors one of the Connect-RPC services authored
// under packages/proto/schemas/audio-tools/v1/.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		stt.Register(core),
		tts.Register(core),
		summarize.Register(core),
		audio.Register(core),
		settings.Register(core),
		usage.Register(core),
		diagnostics.Register(core),
		health.Register(core),
		provider.Register(core),
	}
}
