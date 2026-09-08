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

func TestAttachedRegistryPersistsJoinedTransports(t *testing.T) {
	db, err := sql.Open("sqlite", "file:attached-multi-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	defer db.Close()
	svc, err := NewServiceWithDB(db)
	require.NoError(t, err)
	first, err := svc.Pair(context.Background(), PairInput{HostNodeID: "node-1", Kind: "android", Serial: "serial-db", Transport: "adb", HostNodeOnline: true})
	require.NoError(t, err)
	second, err := svc.Pair(context.Background(), PairInput{HostNodeID: "node-1", Kind: "android", Serial: "serial-db", Transport: "remote-control", HostNodeOnline: true})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, []string{"adb", "remote-control"}, second.Transports)
	reloaded, err := NewServiceWithDB(db)
	require.NoError(t, err)
	items := reloaded.List(context.Background())
	require.Len(t, items, 1)
	require.Equal(t, []string{"adb", "remote-control"}, items[0].Transports)
}

func TestAttachedRegistryUpgradesLegacySchemaForTransportJoin(t *testing.T) {
	db, err := sql.Open("sqlite", "file:attached-legacy-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE bridge_attached_devices (id TEXT PRIMARY KEY, name TEXT NOT NULL DEFAULT '', host_node_id TEXT NOT NULL, kind TEXT NOT NULL, transport TEXT NOT NULL DEFAULT '', serial TEXT NOT NULL DEFAULT '', os_version TEXT NOT NULL DEFAULT '', trust_state TEXT NOT NULL, reachability TEXT NOT NULL, health_reason TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, revoked_at TEXT NOT NULL DEFAULT '')`)
	require.NoError(t, err)
	svc, err := NewServiceWithDB(db)
	require.NoError(t, err)
	device, err := svc.Pair(context.Background(), PairInput{HostNodeID: "node-legacy", Kind: "android", Serial: "legacy-serial", Transport: "adb", HostNodeOnline: true})
	require.NoError(t, err)
	require.Equal(t, []string{"adb"}, device.Transports)
	reloaded := svc.List(context.Background())
	require.Len(t, reloaded, 1)
	require.Equal(t, []string{"adb"}, reloaded[0].Transports)
}

func TestAndroidPairUsesStableSerialIdentityAndConverges(t *testing.T) {
	svc := NewService()
	first, err := svc.Pair(context.Background(), PairInput{HostNodeID: "node-1", Kind: "android", Serial: "R9TT608Q6MH", HostNodeOnline: true})
	require.NoError(t, err)
	second, err := svc.Pair(context.Background(), PairInput{HostNodeID: "node-1", Kind: "android", Serial: "R9TT608Q6MH", HostNodeOnline: false})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, "android-024665203bca17fa", first.ID)
	require.Equal(t, "unreachable", second.Reachability)
	require.Len(t, svc.List(context.Background()), 1)
}

func TestAndroidPairJoinsDistinctTransportsUnderOneDevice(t *testing.T) {
	svc := NewService()
	first, err := svc.Pair(context.Background(), PairInput{HostNodeID: "node-1", Kind: "android", Serial: "serial-multi", Transport: "adb", HostNodeOnline: true})
	require.NoError(t, err)
	second, err := svc.Pair(context.Background(), PairInput{HostNodeID: "node-1", Kind: "android", Serial: "serial-multi", Transport: "remote-control", HostNodeOnline: true})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, []string{"adb", "remote-control"}, second.Transports)
	items := svc.List(context.Background())
	require.Len(t, items, 1)
	require.Equal(t, second.Transports, items[0].Transports)
}

type presenceStub struct{ online map[string]bool }

func (p presenceStub) IsOnline(nodeID string) bool { return p.online[nodeID] }

func TestListProjectsReachabilityFromLiveNodePresence(t *testing.T) {
	svc := NewServiceWithRepositoryAndPresence(newMemoryRepository(), presenceStub{online: map[string]bool{"node-1": true}})
	device, err := svc.Pair(context.Background(), PairInput{HostNodeID: "node-1", Kind: "android", Serial: "serial-1", HostNodeOnline: false})
	require.NoError(t, err)
	require.Equal(t, "unreachable", device.Reachability)

	items := svc.List(context.Background())
	require.Len(t, items, 1)
	require.Equal(t, "reachable", items[0].Reachability)
	require.Empty(t, items[0].HealthReason)
}

func TestListProjectsOfflineNodeWithExplicitReason(t *testing.T) {
	svc := NewServiceWithRepositoryAndPresence(newMemoryRepository(), presenceStub{online: map[string]bool{}})
	_, err := svc.Pair(context.Background(), PairInput{HostNodeID: "node-1", Kind: "android", Serial: "serial-1", HostNodeOnline: true})
	require.NoError(t, err)

	items := svc.List(context.Background())
	require.Len(t, items, 1)
	require.Equal(t, "unreachable", items[0].Reachability)
	require.Equal(t, "host node node-1 is offline", items[0].HealthReason)
}
