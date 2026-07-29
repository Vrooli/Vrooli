package channelmanager

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
)

// [REQ:CHANMGR-P1-001] The operator CLI must carry both opaque BAS references.
// Keeping this at registration level prevents a future flag edit from silently
// restoring the impossible profile-only browser dispatch contract.
func TestAssignAutomationRequiresProfileAndWorkflowReferences(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{})
	var command *cliapp.Command
	for i := range group.Subcommands {
		if group.Subcommands[i].Name == "assign-automation" {
			command = &group.Subcommands[i]
			break
		}
	}
	require.NotNil(t, command)

	flags := map[string]cliapp.Flag{}
	for _, flag := range command.Args.Flags {
		flags[flag.Name] = flag
	}
	require.True(t, flags["session-profile-ref"].Required)
	require.True(t, flags["workflow-ref"].Required)
}
