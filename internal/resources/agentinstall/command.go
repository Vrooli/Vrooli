package agentinstall

import (
	"context"

	"github.com/vrooli/cli-core/cliapp"
)

// DirectInstallCommand exposes the shell-free installer to a resource CLI.
func DirectInstallCommand(spec Spec) cliapp.Command {
	return cliapp.Command{
		Name: "install-direct", Description: "Install the upstream CLI into the user-owned prefix",
		Run: func(_ []string) error { return Install(context.Background(), spec) },
	}
}
