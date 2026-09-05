package domains

import (
	"audio-tools/cli/domains/audio"
	"audio-tools/cli/domains/corpus"
	"audio-tools/cli/domains/diagnostics"
	"audio-tools/cli/domains/experiment"
	"audio-tools/cli/domains/health"
	"audio-tools/cli/domains/provider"
	"audio-tools/cli/domains/settings"
	"audio-tools/cli/domains/soak"
	"audio-tools/cli/domains/stt"
	"audio-tools/cli/domains/summarize"
	"audio-tools/cli/domains/tts"
	"audio-tools/cli/domains/usage"
	"audio-tools/cli/domains/validation"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups from domain packages.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates the audio-tools domain command groups.
// Each domain mirrors one of the Connect-RPC services authored under
// packages/proto/schemas/audio-tools/v1/.
//
// The proto-bound command surface (name/flags/positionals/governance and
// the Service.Method binding for each command) is declared once in
// cli/manifest.json — the single source of truth. The aggregator passes
// the embedded manifest bytes through unchanged; each domain's Register
// calls cliapp.LoadFromManifest with its group name and a bindings map
// wiring "<Service>.<Method>" → the handler in that domain's handlers.go.
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	return subcommandGroups(core, manifest, nil, nil, nil, nil)
}

// SubcommandGroupsWithClock is the runtime entrypoint. Long-running
// qualification commands receive the process clock from the composition root;
// tests can use SubcommandGroups without acquiring a wall-clock dependency.
func SubcommandGroupsWithClock(core *cliapp.ScenarioApp, manifest []byte, now func() time.Time, getenv func(string) string, getwd func() (string, error)) ([]cliapp.SubcommandGroup, error) {
	return subcommandGroups(core, manifest, now, time.NewTicker, getenv, getwd)
}

func subcommandGroups(core *cliapp.ScenarioApp, manifest []byte, now func() time.Time, newTicker func(time.Duration) *time.Ticker, getenv func(string) string, getwd func() (string, error)) ([]cliapp.SubcommandGroup, error) {
	validationGroup := validation.Register(core, now, newTicker, getenv, getwd)
	soakGroup := soak.Register(core, now, getenv)
	sttGroup, err := stt.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	ttsGroup, err := tts.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	summarizeGroup, err := summarize.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	audioGroup, err := audio.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	settingsGroup, err := settings.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	usageGroup, err := usage.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	diagnosticsGroup, err := diagnostics.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	healthGroup, err := health.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	providerGroup, err := provider.RegisterWithClock(core, manifest, now)
	if err != nil {
		return nil, err
	}
	corpusGroup, err := corpus.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	experimentGroup, err := experiment.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	return []cliapp.SubcommandGroup{
		validationGroup,
		soakGroup,
		sttGroup,
		ttsGroup,
		summarizeGroup,
		audioGroup,
		settingsGroup,
		usageGroup,
		diagnosticsGroup,
		healthGroup,
		providerGroup,
		corpusGroup,
		experimentGroup,
	}, nil
}
