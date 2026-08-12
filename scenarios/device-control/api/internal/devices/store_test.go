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
