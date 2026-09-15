package recovery_test

import (
	"context"
	"testing"

	"tunnel-manager/internal/recovery"
	"tunnel-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

func TestControlPlaneLifecyclePresenceUsesControlPlaneJSONFlag(t *testing.T) {
	runner := &mocks.FakeCmdRunner{Out: []byte(`{"installed":true,"running":false}`)}
	lifecycle := recovery.NewControlPlaneLifecycle(runner.Run)

	require.True(t, lifecycle.CloudflaredUnitPresent(context.Background()))
	require.Len(t, runner.Calls, 1)
	require.Equal(t, "vrooli", runner.Calls[0].Name)
	require.Equal(t, []string{"resource", "status", "cloudflared", "--json"}, runner.Calls[0].Args)
}

func TestControlPlaneLifecyclePresenceFailsClosedOnStatusError(t *testing.T) {
	runner := &mocks.FakeCmdRunner{Err: context.DeadlineExceeded}
	lifecycle := recovery.NewControlPlaneLifecycle(runner.Run)

	require.False(t, lifecycle.CloudflaredUnitPresent(context.Background()))
}
