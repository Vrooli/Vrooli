package cleanup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypedCleanupRemoteCommandIsFixedHelperInvocation(t *testing.T) {
	command := typedCleanupRemoteCommand()
	require.Contains(t, command, "--cleanup-stdin")
	require.Contains(t, command, "sudo -n")
	require.NotContains(t, command, "uninstall")
	require.NotContains(t, command, "--apply")
	require.NotContains(t, command, "sh -c")
}
