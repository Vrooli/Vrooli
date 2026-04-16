package manifest

import (
	"scenario-to-cloud/cli/internal/appctx"
	manifestcmd "scenario-to-cloud/cli/manifest"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Manifest",
		Commands: []cliapp.Command{
			{
				Name:        "manifest",
				NeedsAPI:    true,
				Description: "Manifest operations (validate, schema, init, template, doctor, fix)",
				Run: func(args []string) error {
					return manifestcmd.Run(deps.ManifestClient, args)
				},
			},
		},
	}
}
