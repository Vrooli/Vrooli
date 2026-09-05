package authcli

import (
	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type CommandID string

const (
	CommandStatus CommandID = "status"
)

func CommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{
			Name:    string(CommandStatus),
			Summary: "Report sign-in state for host tools (buf, future: claude, codex, gh, ...)",
			Group:   "Authentication",
			Help: commandtree.Help{
				Description: "Reads each tool's local credential store to report whether the operator is signed in. The default probe is offline; pass --check-expiry to additionally validate tokens against the upstream service.",
			},
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					{
						Name:        "--check-expiry",
						Description: "Validate tokens against the upstream service (generates network traffic; default off)",
					},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandStatus,
		},
	}
}
