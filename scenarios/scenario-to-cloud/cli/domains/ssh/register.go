package ssh

import (
	"scenario-to-cloud/cli/internal/appctx"
	sshcmd "scenario-to-cloud/cli/ssh"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "SSH",
		Commands: []cliapp.Command{
			{
				Name:        "ssh",
				NeedsAPI:    true,
				Description: "SSH access workflows (keys, generate, test, bootstrap, copy-key)",
				Run: func(args []string) error {
					return sshcmd.Run(deps.SSHClient, args)
				},
			},
		},
	}
}
