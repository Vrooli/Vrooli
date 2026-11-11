package control_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	"google.golang.org/protobuf/proto"
	control "lighting-control/internal/control"
)

func TestOfflineInventoryPreservesObservationAgeAndNeverSuppressesStorageErrors(t *testing.T) {
	ctx := t.Context()
	clock := scheduletest.New(time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC))
	b := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 19, Serial: "saved-observation", Zones: []control.SDKZone{{Name: "Header", LEDCount: 8, Static: true}}}}}
	svc, db := newService(t, b, clock)
	_, saved, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	require.Len(t, saved, 1)
	_, err = svc.Enroll(ctx, saved[0].ControllerId, true)
	require.NoError(t, err)
	saved[0].Enrolled = true
	clock.Advance(time.Hour)
	b.discoverErr = control.ErrTransport
	status, offline, err := svc.ListControllers(ctx, true)
	require.NoError(t, err)
	require.Len(t, offline, 1)
	require.Equal(t, controlv1.BackendState_BACKEND_STATE_UNAVAILABLE, status.State)
	require.True(t, proto.Equal(saved[0], offline[0]), "offline reads cannot refresh last-seen or capability observations")
	require.Equal(t, clock.Now().UTC().Format(time.RFC3339Nano), status.CheckedAt, "only the backend error observation is current")
	require.NoError(t, db.Close())
	status, current, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	require.Empty(t, current)
	require.NotEqual(t, controlv1.BackendState_BACKEND_STATE_READY, status.State)
	_, _, err = svc.ListControllers(ctx, true)
	require.ErrorContains(t, err, "list persisted controllers")
}
