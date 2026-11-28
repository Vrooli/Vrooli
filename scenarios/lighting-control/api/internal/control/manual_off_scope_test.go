package control_test

import (
	"connectrpc.com/connect"
	"context"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scheduletest"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	controlhandler "lighting-control/handlers/control"
	"lighting-control/internal/control"
	"testing"
	"time"
)

// A scoped Off must survive SQLite reopening and handle changes without
// becoming an all-enrolled request or overwriting independent manual intent.
func TestPublicScopedOffSurvivesRosterExpansionAndWidensWithoutClearing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	clock := scheduletest.New(time.Now())
	b := &fakeBackend{protocol: 6, controllers: []control.SDKController{
		{ID: 10, Serial: "g-one", Zones: []control.SDKZone{{Name: "Header", LEDCount: 2, Static: true}}},
		{ID: 20, Serial: "g-two", Zones: []control.SDKZone{{Name: "Header", LEDCount: 2, Static: true}}},
	}}
	svc, db := newService(t, b, clock)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Scopes: []string{"lighting-control:write"}})
	h := controlhandler.NewConnectHandler(svc, nil)
	inv, err := h.ListControllers(ctx, connect.NewRequest(&controlv1.ListControllersRequest{}))
	require.NoError(t, err)
	require.Len(t, inv.Msg.Controllers, 2)
	ids := map[uint32]string{}
	zs := map[uint32]string{}
	for _, c := range inv.Msg.Controllers {
		_, err = h.EnrollController(ctx, connect.NewRequest(&controlv1.EnrollControllerRequest{ControllerId: c.ControllerId, Enrolled: true}))
		require.NoError(t, err)
		_, err = h.SetZoneState(ctx, connect.NewRequest(&controlv1.SetZoneStateRequest{Desired: staticState(c.ControllerId, c.Zones[0].ZoneId, 90, 40, 20)}))
		require.NoError(t, err)
	}
	// Identify durable IDs by fresh service output ordering's associated handle
	// indirectly through serial-derived response order, which repository sorts.
	for _, c := range inv.Msg.Controllers {
		if c.ControllerId == scopedOffControllerID(t, b, 10) {
			ids[10] = c.ControllerId
			zs[10] = c.Zones[0].ZoneId
		} else {
			ids[20] = c.ControllerId
			zs[20] = c.Zones[0].ZoneId
		}
	}
	_, err = h.SetManualOff(ctx, connect.NewRequest(&controlv1.SetManualOffRequest{ControllerIds: []string{ids[10]}}))
	require.NoError(t, err)
	b.controllers[0].ID = 101
	b.controllers[1].ID = 202
	b.controllers[0].Zones = append(b.controllers[0].Zones, control.SDKZone{Name: "New Header", LEDCount: 1, Index: 1, Static: true})
	svc = control.NewService(control.NewSQLiteRepository(db, clock), b, clock)
	h = controlhandler.NewConnectHandler(svc, nil)
	before := len(b.writes)
	_, err = h.Reconcile(ctx, connect.NewRequest(&controlv1.ReconcileRequest{}))
	require.NoError(t, err)
	one, two, newZone := false, false, false
	for _, w := range b.writes[before:] {
		switch w.controller {
		case 101:
			one = true
			for _, rgb := range w.colors {
				require.Equal(t, control.RGB{}, rgb, "scoped Off covers new zone of selected physical controller")
			}
			if w.zone == 1 {
				newZone = true
			}
		case 202:
			two = true
			require.NotEqual(t, control.RGB{}, w.colors[0], "independent controller retains manual color")
		default:
			t.Fatalf("stale SDK handle %d", w.controller)
		}
	}
	require.True(t, one)
	require.True(t, two)
	require.True(t, newZone)
	desired, err := h.GetDesiredState(ctx, connect.NewRequest(&controlv1.GetDesiredStateRequest{}))
	require.NoError(t, err)
	require.True(t, desired.Msg.ManualOverride.Active)
	require.False(t, desired.Msg.ManualOverride.AllEnrolled)
	require.Equal(t, []string{ids[10]}, desired.Msg.ManualOverride.ControllerIds)
	for _, z := range desired.Msg.Zones {
		if z.Desired.ZoneId == zs[10] || z.Desired.ZoneId == zs[20] {
			require.NotNil(t, z.Desired.GetStaticColor(), "base remains intact")
		}
	}
	// The normal global Off action widens protection directly; no intermediate
	// clear may expose A while B is added to the scope.
	before = len(b.writes)
	global, err := h.SetManualOff(ctx, connect.NewRequest(&controlv1.SetManualOffRequest{AllEnrolled: true}))
	require.NoError(t, err)
	require.True(t, global.Msg.Override.AllEnrolled)
	require.Greater(t, len(b.writes), before)
	for _, w := range b.writes[before:] {
		for _, rgb := range w.colors {
			require.Equal(t, control.RGB{}, rgb, "global action preserves A Off and also covers B")
		}
	}

}
func scopedOffControllerID(t *testing.T, b *fakeBackend, handle uint32) string {
	t.Helper()
	clock := scheduletest.New(time.Now())
	one := &fakeBackend{protocol: 6}
	for _, c := range b.controllers {
		if c.ID == handle {
			one.controllers = []control.SDKController{c}
		}
	}
	svc, _ := newService(t, one, clock)
	_, cs, err := svc.ListControllers(context.Background())
	require.NoError(t, err)
	require.Len(t, cs, 1)
	return cs[0].ControllerId
}
