package soak

import (
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "soak"

type handlers struct {
	core   *cliapp.ScenarioApp
	now    func() time.Time
	getenv func(string) string
}

func Register(core *cliapp.ScenarioApp, now func() time.Time, getenv func(string) string) cliapp.SubcommandGroup {
	if now == nil {
		now = time.Now
	}
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	h := &handlers{core: core, now: now, getenv: getenv}
	return cliapp.SubcommandGroup{
		Name:        GroupName,
		Description: "Real browser-to-composer long-form dictation qualification",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{{
			Name:        "run",
			Description: "Drive Dictation Studio through BAS and emit one conformance.Run document",
			NeedsAPI:    true,
			Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
				{Name: "driver-url", Description: "Playwright driver URL (defaults to PLAYWRIGHT_DRIVER_URL)"},
				{Name: "ui-url", Description: "Audio-tools UI URL (defaults to AUDIO_TOOLS_UI_URL or UI_BASE_URL)"},
				{Name: "surface", Description: "Product surface: audio-tools or swarm-manager (defaults to audio-tools)"},
				{Name: "fixture", Description: "Absolute WAV fixture path used by BAS fake_media"},
				{Name: "lane", Description: "accelerated or realtime"},
				{Name: "profile", Description: "realistic or continuous"},
				{Name: "turns", Description: "Number of product turns"},
				{Name: "feed-ms", Description: "Wall-clock fixture feed time per turn"},
				{Name: "fault", Description: "Optional leased fault profile"},
				{Name: "reference-text", Description: "Reference transcript for realtime quality assertions"},
				{Name: "engine-id", Description: "Exact provider engine id"},
				{Name: "model-id", Description: "Exact provider model id"},
				{Name: "strategy", Description: "Streaming strategy"},
				{Name: "policy", Description: "Dictation policy profile"},
				{Name: "shape", Description: "Accelerated virtual corpus shape: burst or chunked"},
				{Name: "simulated-minutes", Description: "Accelerated virtual-corpus duration target (default 60)"},
				{Name: "evidence-path", Description: "Evidence JSON path (defaults to coverage/<run-id>.json)"},
			}},
			RunCtx: h.run,
		}},
	}
}
