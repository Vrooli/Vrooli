package edge

import (
	edgecmd "scenario-to-cloud/cli/edge"
	"scenario-to-cloud/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Edge",
		Commands: []cliapp.Command{
			{
				Name:        "edge",
				NeedsAPI:    true,
				Description: "Edge and TLS management (dns-check, dns-records, caddy, tls, tls-renew)",
				Run: func(args []string) error {
					return edgecmd.Run(deps.EdgeClient, args)
				},
			},
		},
	}
}
