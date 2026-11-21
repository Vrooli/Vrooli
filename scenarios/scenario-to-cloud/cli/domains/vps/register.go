package vps

import (
	"scenario-to-cloud/cli/internal/appctx"
	vpscmd "scenario-to-cloud/cli/vps"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "VPS",
		Commands: []cliapp.Command{
			{
				Name:        "vps",
				NeedsAPI:    true,
				Description: "VPS operations (setup, deploy)",
				Run: func(args []string) error {
					return vpscmd.Run(deps.VPSClient, args)
				},
			},
		},
	}
}
