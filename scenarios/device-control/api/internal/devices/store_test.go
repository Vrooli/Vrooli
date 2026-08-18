package devices

import (
	"testing"
	"time"

	"device-control/internal/identity"
	"github.com/stretchr/testify/require"
)

func TestStoreRetainsStableIdentityWhenDeviceDisappears(t *testing.T) {
	store := NewStore()
	seen := time.Date(2026, 8, 12, 15, 0, 0, 0, time.UTC)
	store.Upsert(Record{ID: "android-a", Kind: "physical", Serial: "serial-a", Status: "available", Health: "available", ObservedAt: seen})
	store.MarkAbsent(seen.Add(time.Minute), func(record Record) string { return "device not present on host node host-a" })
	record, ok := store.Get("android-a")
	require.True(t, ok)
	require.Equal(t, "serial-a", record.Serial)
	require.Equal(t, "unreachable", record.Health)
	require.Contains(t, record.HealthReason, "host node host-a")
	require.Equal(t, seen, record.FirstSeenAt)
}

func TestStoreForgetRemovesOnlyExplicitlyRequestedIdentity(t *testing.T) {
	store := NewStore()
	store.Upsert(Record{ID: "android-a", Kind: "physical", Serial: "serial-a"})
	store.Upsert(Record{ID: "android-b", Kind: "physical", Serial: "serial-b"})

	require.True(t, store.Forget("android-a"))
	_, found := store.Get("android-a")
	require.False(t, found)
	_, found = store.Get("android-b")
	require.True(t, found)
	require.False(t, store.Forget("android-a"))
}

func TestStoreUpsertIdentityRefusesAddressOnlyCorrelation(t *testing.T) {
	store := NewStore()
	store.Upsert(Record{ID: "android-old", Kind: "physical", Serial: "old-serial", Endpoint: "192.168.1.179:34483"})
	got := store.UpsertIdentity(Record{ID: "android-stable", Kind: "physical", Serial: "R9TT608Q6MH", Endpoint: "192.168.1.179:34483"})
	require.Equal(t, "android-stable", got.ID)
	old, found := store.Get("android-old")
	require.True(t, found)
	require.Equal(t, "address-only-correlation-refused", got.IdentityReason)
	require.Equal(t, "address-only-correlation-refused", old.IdentityReason)
	_, found = store.Get("android-stable")
	require.True(t, found)
}

func TestStoreUpsertIdentityMergesTransportProfiles(t *testing.T) {
	store := NewStore()
	first := store.UpsertIdentity(Record{ID: "tv", Kind: "physical", Serial: "tv-serial", StrategyID: "android-adb", Transport: "usb", Endpoint: "usb", Capabilities: nil})
	require.Equal(t, "tv", first.ID)
	second := store.UpsertIdentity(Record{ID: "tv", Kind: "physical", Serial: "tv-serial", IdentityKey: "tv-serial", StrategyID: "android-tv-remote", Transport: "mdns", Endpoint: "tv.local:6466"})
	require.Equal(t, "tv", second.ID)
	require.Len(t, second.Transports, 2)
	transportIDs := []string{second.Transports[0].StrategyID, second.Transports[1].StrategyID}
	require.ElementsMatch(t, []string{"android-adb", "android-tv-remote"}, transportIDs)
}

func TestStoreMergesGoogleTVRemoteAndCastOnlyWhenClaimsAgree(t *testing.T) {
	store := NewStore()
	remote := store.UpsertIdentity(Record{ID: "android-tv:bt-1", IdentityKey: "AA:BB:CC:DD:EE:FF", IdentityKind: string(identity.BluetoothMAC), Kind: "physical", Serial: "AA:BB:CC:DD:EE:FF", StrategyID: "android-tv-remote", Transport: "mdns"})
	cast := store.UpsertIdentity(Record{ID: "google-cast:bt-1", IdentityKey: "AA:BB:CC:DD:EE:FF", IdentityKind: string(identity.CastID), Kind: "physical", Serial: "AA:BB:CC:DD:EE:FF", StrategyID: "google-cast", Transport: "cast"})
	require.NotEqual(t, remote.ID, cast.ID, "different claim kinds must not merge")

	castAgain := store.UpsertIdentity(Record{ID: "google-cast:cast-1", IdentityKey: "cast-1", IdentityKind: string(identity.CastID), Kind: "physical", Serial: "cast-1", StrategyID: "google-cast", Transport: "cast-secondary"})
	castSameClaim := store.UpsertIdentity(Record{ID: "google-cast:cast-2", IdentityKey: "cast-1", IdentityKind: string(identity.CastID), Kind: "physical", Serial: "cast-1", StrategyID: "google-cast", Transport: "cast-tertiary"})
	require.Equal(t, castAgain.ID, castSameClaim.ID)
	require.Len(t, castSameClaim.Transports, 2)
}

func TestStoreMergeAndSplitRestoresIdentitySnapshots(t *testing.T) {
	store := NewStore()
	canonical := store.UpsertIdentity(Record{ID: "tv-a", IdentityKey: "cast-a", IdentityKind: string(identity.CastID), Claims: []identity.IdentityClaim{{Kind: identity.CastID, Value: "cast-a", Evidence: "observed"}}, Kind: "physical", StrategyID: "google-cast", Transport: "cast", Endpoint: "192.168.1.10:8009"})
	member := store.UpsertIdentity(Record{ID: "tv-b", IdentityKey: "cast-b", IdentityKind: string(identity.CastID), Claims: []identity.IdentityClaim{{Kind: identity.CastID, Value: "cast-b", Evidence: "observed"}}, Kind: "physical", StrategyID: "android-tv-remote", Transport: "mdns", Endpoint: "192.168.1.10:6466"})
	claim := identity.IdentityClaim{Kind: identity.CastID, Value: "cast-a", Evidence: "owner-asserted"}
	merged, err := store.Merge(canonical.ID, member.ID, claim)
	require.NoError(t, err)
	require.Contains(t, merged.IdentityReason, "merged-on")
	_, found := store.Get(member.ID)
	require.False(t, found)
	restored, err := store.Split(canonical.ID)
	require.NoError(t, err)
	require.Len(t, restored, 2)
	_, found = store.Get(member.ID)
	require.True(t, found)
}
