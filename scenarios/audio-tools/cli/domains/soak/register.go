package soak

import (
	"os"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "soak"

type handlers struct {
	core   *cliapp.ScenarioApp
	getenv func(string) string
	getwd  func() (string, error)
}

func Register(core *cliapp.ScenarioApp, getenv func(string) string, getwd func() (string, error)) cliapp.SubcommandGroup {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if getwd == nil {
		getwd = os.Getwd
	}
	h := &handlers{core: core, getenv: getenv, getwd: getwd}
	return cliapp.SubcommandGroup{
		Name:        GroupName,
		Description: "Real browser-to-composer long-form dictation qualification",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{{
			Name:        "run",
			Description: "Run the owner-authorized Audio Tools task through the BAS session broker and emit one conformance.Run document",
			NeedsAPI:    true,
			Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
				{Name: "lane", Description: "accelerated or realtime"},
				{Name: "profile", Description: "realistic or continuous"},
				{Name: "turns", Description: "Number of product turns"},
				{Name: "feed-ms", Description: "Wall-clock fixture feed time per turn"},
				{Name: "fault", Description: "Optional leased fault profile"},
				{Name: "reference-text", Description: "Reference transcript for realtime quality assertions"},
				{Name: "reference-cycle-ms", Description: "Optional repeated reference-cycle duration for looped long-form fixtures"},
				{Name: "engine-id", Description: "Exact provider engine id"},
				{Name: "model-id", Description: "Exact provider model id"},
				{Name: "strategy", Description: "Streaming strategy"},
				{Name: "policy", Description: "Dictation policy profile"},
				{Name: "shape", Description: "Accelerated virtual corpus shape: burst or chunked"},
				{Name: "simulated-minutes", Description: "Accelerated virtual-corpus duration target (default 60)"},
				{Name: "mobile-recovery", Description: "true|false — run the emulated 390x844 retained-coverage and metadata-only diagnostic export proof"},
				{Name: "evidence-path", Description: "Evidence JSON path (defaults to coverage/<run-id>.json)"},
			}},
			RunCtx: h.run,
		}},
	}
}
