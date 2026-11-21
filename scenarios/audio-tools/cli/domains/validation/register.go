// Package validation owns the scenario-local, deliberately out-of-band audio
// qualification commands. They are not Connect-RPC methods because their
// responsibility is to coordinate multiple product paths and persist durable
// evidence for the requirements business phase.
package validation

import (
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "validation"

func Register(core *cliapp.ScenarioApp, now func() time.Time, newTicker func(time.Duration) *time.Ticker, getenv func(string) string, getwd func() (string, error)) cliapp.SubcommandGroup {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if getwd == nil {
		getwd = func() (string, error) { return "", nil }
	}
	return cliapp.SubcommandGroup{
		Name:        GroupName,
		Description: "Expensive out-of-band audio validation and evidence freshness",
		NeedsAPI:    false,
		Subcommands: []cliapp.Command{
			{
				Name:        "run-expensive",
				Description: "Run the hour-scale TTS and continuous synthetic-dictation qualification and record evidence",
				NeedsAPI:    true,
				Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
					{Name: "duration-minutes", Description: "Qualification duration; must be at least 60 minutes"},
					{Name: "evidence-path", Description: "Optional evidence JSON path"},
				}},
				RunCtx: newHandlersWithDependencies(core, now, newTicker, getenv, getwd).runExpensive,
			},
			{
				Name:        "check-freshness",
				Description: "Fail when any out-of-band audio validation is missing or stale",
				Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
					{Name: "requirements-path", Description: "Optional requirements directory"},
				}},
				RunCtx: newHandlersWithDependencies(core, now, newTicker, getenv, getwd).checkFreshness,
			},
			{
				Name:        "run-simulated",
				Description: "Run local policy and adapter qualification and persist a labelled simulated receipt",
				NeedsAPI:    true,
				Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
					{Name: "device-profile", Description: "Observed host profile for the simulated receipt (for example D1 or D2)"},
				}},
				RunCtx: newHandlersWithDependencies(core, now, newTicker, getenv, getwd).runSimulated,
			},
			{
				Name:        "run-speaker-policy",
				Description: "Exercise live batch/native speaker policy and fail-closed controls, then record owner evidence",
				NeedsAPI:    true,
				Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
					{Name: "file", Description: "WAV fixture used for the owner-backed policy probe"},
					{Name: "device-profile", Description: "Observed host profile for the receipt (defaults to D1)"},
				}},
				RunCtx: newHandlersWithDependencies(core, now, newTicker, getenv, getwd).runSpeakerPolicy,
			},
			{
				Name:        "run-host-portability",
				Description: "Measure D1 CPU/accelerator routes and record explicit D2 unavailable evidence",
				NeedsAPI:    true,
				RunCtx:      newHandlersWithDependencies(core, now, newTicker, getenv, getwd).runHostPortability,
			},
			{
				Name:        "run-acceptance-join",
				Description: "Validate fresh owner receipts and persist one provenance-preserving acceptance join",
				NeedsAPI:    true,
				RunCtx:      newHandlersWithDependencies(core, now, newTicker, getenv, getwd).runAcceptanceJoin,
			},
		},
	}
}
