package attached

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestAttachedRegistryPersistsPairAndRevoke(t *testing.T) {
	db, err := sql.Open("sqlite", "file:attached-test-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	defer db.Close()
	svc, err := NewServiceWithDB(db)
	require.NoError(t, err)
	device, err := svc.Pair(context.Background(), PairInput{HostNodeID: "node-1", Kind: "android", HostNodeOnline: false})
	require.NoError(t, err)
	require.Equal(t, "unreachable", device.Reachability)
	require.Equal(t, "host node node-1 is offline", device.HealthReason)
	reloaded, err := NewServiceWithDB(db)
	require.NoError(t, err)
	require.Len(t, reloaded.List(context.Background()), 1)
	_, err = reloaded.Revoke(context.Background(), device.ID)
	require.NoError(t, err)
	require.Empty(t, reloaded.List(context.Background()))
}
