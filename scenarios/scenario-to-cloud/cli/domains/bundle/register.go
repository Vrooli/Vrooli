package bundle

import (
	bundlecmd "scenario-to-cloud/cli/bundle"
	"scenario-to-cloud/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Bundles",
		Commands: []cliapp.Command{
			{
				Name:        "bundle",
				NeedsAPI:    true,
				Description: "Bundle operations (build, list, stats, cleanup, VPS bundle ops)",
				Run: func(args []string) error {
					return bundlecmd.Run(deps.BundleClient, args)
				},
			},
		},
	}
}
