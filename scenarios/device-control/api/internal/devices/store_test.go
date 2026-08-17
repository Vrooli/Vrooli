package devices

import (
	"testing"
	"time"

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

func TestStoreUpsertIdentityRemovesEndpointAlias(t *testing.T) {
	store := NewStore()
	store.Upsert(Record{ID: "android-old", Kind: "physical", Serial: "192.168.1.179:34483"})
	got := store.UpsertIdentity(Record{ID: "android-stable", Kind: "physical", Serial: "R9TT608Q6MH", Endpoint: "192.168.1.179:34483"})
	require.Equal(t, "android-stable", got.ID)
	_, found := store.Get("android-old")
	require.False(t, found)
	_, found = store.Get("android-stable")
	require.True(t, found)
}

func TestStoreUpsertIdentityMergesTransportProfiles(t *testing.T) {
	store := NewStore()
	first := store.UpsertIdentity(Record{ID: "tv", Kind: "physical", Serial: "tv-serial", StrategyID: "android-adb", Transport: "usb", Endpoint: "usb", Capabilities: nil})
	require.Equal(t, "tv", first.ID)
	second := store.UpsertIdentity(Record{ID: "tv", Kind: "physical", Serial: "tv-serial", StrategyID: "android-tv-remote", Transport: "mdns", Endpoint: "tv.local:6466"})
	require.Equal(t, "tv", second.ID)
	require.Len(t, second.Transports, 2)
	transportIDs := []string{second.Transports[0].StrategyID, second.Transports[1].StrategyID}
	require.ElementsMatch(t, []string{"android-adb", "android-tv-remote"}, transportIDs)
}
