package realtime_test

import (
	"testing"
	"time"

	internalrealtime "device-sync-hub/internal/realtime"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// drainPresence reads events until it finds the latest presence snapshot,
// returning the set of online device ids. SSE priming + broadcast can enqueue
// several presence events; the test cares about the converged set.
func latestPresence(t *testing.T, sub *internalrealtime.Subscription) map[string]bool {
	t.Helper()
	set := map[string]bool{}
	deadline := time.After(time.Second)
	for {
		select {
		case evt := <-sub.Events():
			if evt.Kind == internalrealtime.EventPresenceChanged {
				set = map[string]bool{}
				for _, p := range evt.Presence {
					set[p.DeviceID] = p.Online
				}
			}
		case <-time.After(50 * time.Millisecond):
			return set
		case <-deadline:
			return set
		}
	}
}

func TestPresenceReflectsConnections(t *testing.T) {
	hub := internalrealtime.NewHub(scheduletest.New(time.Time{}))

	a := hub.Subscribe("owner", "dev-a")
	defer a.Close()
	require.ElementsMatch(t, []string{"dev-a"}, hub.OnlineDevices("owner"))

	b := hub.Subscribe("owner", "dev-b")
	online := latestPresence(t, a)
	assert.True(t, online["dev-a"])
	assert.True(t, online["dev-b"])

	b.Close()
	assert.ElementsMatch(t, []string{"dev-a"}, hub.OnlineDevices("owner"))
}

func TestItemArrivedBroadcastVsDirected(t *testing.T) {
	hub := internalrealtime.NewHub(scheduletest.New(time.Time{}))
	a := hub.Subscribe("owner", "dev-a")
	defer a.Close()
	b := hub.Subscribe("owner", "dev-b")
	defer b.Close()
	// Drain priming presence events.
	latestPresence(t, a)
	latestPresence(t, b)

	// Broadcast reaches both.
	hub.EmitItemArrived("owner", "item-1", "", "dev-a")
	assert.True(t, sawItem(a, "item-1"))
	assert.True(t, sawItem(b, "item-1"))

	// Directed to dev-b reaches dev-b (target) and dev-a (origin) only.
	hub.EmitItemArrived("owner", "item-2", "dev-b", "dev-a")
	assert.True(t, sawItem(b, "item-2"))
	assert.True(t, sawItem(a, "item-2"), "origin sees its own directed item")

	// Directed to dev-b from dev-c must NOT reach dev-a.
	hub.EmitItemArrived("owner", "item-3", "dev-b", "dev-c")
	assert.True(t, sawItem(b, "item-3"))
	assert.False(t, sawItem(a, "item-3"))
}

func TestPairingRequestedFanOut(t *testing.T) {
	hub := internalrealtime.NewHub(scheduletest.New(time.Time{}))
	a := hub.Subscribe("owner", "dev-a")
	defer a.Close()
	latestPresence(t, a)

	hub.EmitPairingRequested("owner", internalrealtime.PairingInfo{DeviceID: "pending-1", Name: "New Tablet", Kind: "tablet"})
	evt := waitKind(a, internalrealtime.EventPairingRequested)
	require.NotNil(t, evt)
	require.NotNil(t, evt.Pairing)
	assert.Equal(t, "pending-1", evt.Pairing.DeviceID)
	assert.Equal(t, "New Tablet", evt.Pairing.Name)
}

// sawItem reports whether an item event for id arrives within a short window.
func sawItem(sub *internalrealtime.Subscription, id string) bool {
	for {
		select {
		case evt := <-sub.Events():
			if (evt.Kind == internalrealtime.EventItemArrived || evt.Kind == internalrealtime.EventItemDeleted) && evt.ItemID == id {
				return true
			}
		case <-time.After(100 * time.Millisecond):
			return false
		}
	}
}

func waitKind(sub *internalrealtime.Subscription, kind internalrealtime.EventKind) *internalrealtime.Event {
	for {
		select {
		case evt := <-sub.Events():
			if evt.Kind == kind {
				e := evt
				return &e
			}
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	}
}
