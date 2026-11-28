package control_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	controlhandler "lighting-control/handlers/control"
	"lighting-control/internal/control"
)

// The Light Desk requests unavailable inventory so enrolled wiring remains
// inspectable when the optional native owner has quiesced or disconnected.
// This crosses the public Connect handler and the actual persisted roster.
func TestNativeEConsumerRetainsEnrolledInventoryWhenBindingUnavailable(t *testing.T) {
	for _, unavailable := range []string{"native-quiesced", "transport-disconnected"} {
		t.Run(unavailable, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			ctx := t.Context()
			clock := scheduletest.New(time.Now())
			backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 19, Serial: "native-e-consumer", Zones: []control.SDKZone{{Name: "Header", LEDCount: 8, Static: true}}}}}
			seed, db := newService(t, backend, clock)
			_, roster, err := seed.ListControllers(ctx)
			require.NoError(t, err)
			require.Len(t, roster, 1)
			id, zone := roster[0].ControllerId, roster[0].Zones[0].ZoneId
			_, err = seed.Enroll(ctx, id, true)
			require.NoError(t, err)
			_, _, err = seed.SetManualOff(ctx, true, nil, nil)
			require.NoError(t, err)
			writesBefore := len(backend.writes)
			var offline control.Backend = control.NewBackendBinding()
			if unavailable == "transport-disconnected" {
				backend.discoverErr = control.ErrTransport
				offline = backend
			}
			service := control.NewService(control.NewSQLiteRepository(db, clock), offline, clock)
			handler := controlhandler.NewConnectHandler(service, nil)
			current, err := handler.ListControllers(ctx, connect.NewRequest(&controlv1.ListControllersRequest{}))
			require.NoError(t, err)
			require.Empty(t, current.Msg.Controllers, "unavailable inventory must not be presented as connected")
			response, err := handler.ListControllers(ctx, connect.NewRequest(&controlv1.ListControllersRequest{IncludeUnavailable: true}))
			require.NoError(t, err)
			require.NotEqual(t, controlv1.BackendState_BACKEND_STATE_READY, response.Msg.Backend.State)
			require.Len(t, response.Msg.Controllers, 1, "explicit offline inventory retains enrolled wiring")
			require.Equal(t, id, response.Msg.Controllers[0].ControllerId)
			require.True(t, response.Msg.Controllers[0].Enrolled)
			require.Equal(t, zone, response.Msg.Controllers[0].Zones[0].ZoneId)
			_, override, err := service.DesiredState(ctx)
			require.NoError(t, err)
			require.True(t, override.Active)
			require.Len(t, backend.writes, writesBefore, "offline inspection cannot emit commands")
		})
	}
}
