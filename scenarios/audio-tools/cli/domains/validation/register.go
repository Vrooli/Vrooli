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
		},
	}
}
