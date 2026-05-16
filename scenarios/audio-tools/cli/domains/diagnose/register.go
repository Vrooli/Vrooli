// Package diagnose hosts the `audio-tools diagnose ...` subtree, mostly a
// stub today. Full provider-probe matrix lands in Phase G alongside
// Diagnostics Workbench UI.
package diagnose

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "diagnose",
		Description: "Diagnose provider availability and routing",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "providers",
				Description: "Probe enabled providers and print availability matrix (Phase G placeholder)",
				RunCtx: func(ctx cliapp.RunContext) error {
					fmt.Fprintln(ctx.Stdout(), "diagnose providers: implementation lands in Phase G (Diagnostics Workbench backend).")
					return nil
				},
			},
		},
	}
}
