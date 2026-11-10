package control_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	"google.golang.org/protobuf/types/known/timestamppb"
	control "lighting-control/internal/control"
)

type fakeBackend struct {
	controllers []control.SDKController
	protocol    uint32
	protocolErr error
	discoverErr error
	writes      []sentColor
}
type sentColor struct {
	controller, zone uint32
	colors           []control.RGB
}

func (f *fakeBackend) ProtocolVersion(context.Context) (uint32, error) {
	return f.protocol, f.protocolErr
}

func (f *fakeBackend) Discover(context.Context) ([]control.SDKController, error) {
	return f.controllers, f.discoverErr
}

func (f *fakeBackend) SendZoneColors(_ context.Context, c, z uint32, colors []control.RGB) (control.SDKCommandResult, error) {
	f.writes = append(f.writes, sentColor{c, z, append([]control.RGB(nil), colors...)})
	return control.SDKCommandResult{LocalWriteCompleted: true, BackendAcknowledged: true, AckStatus: 0}, nil
}

func newService(t *testing.T, backend control.Backend, clock *scheduletest.FakeClock) (*control.Service, *sql.DB) {
	t.Helper()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(control.Schema)))
	repo := control.NewSQLiteRepository(db, clock)
	return control.NewService(repo, backend, clock), db
}

func TestDiscoveryUsesStableUniqueIDsAndZoneFingerprintsNotOrder(t *testing.T) {
	clock := scheduletest.New(time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC))
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 77, Name: "Board", Zones: []control.SDKZone{{Name: "Header", LEDCount: 4, Index: 0, Static: true}, {Name: "Strip", LEDCount: 8, Index: 1, Static: true}}}}}
	svc, _ := newService(t, backend, clock)
	_, first, err := svc.ListControllers(context.Background())
	require.NoError(t, err)
	backend.controllers[0].Zones[0], backend.controllers[0].Zones[1] = backend.controllers[0].Zones[1], backend.controllers[0].Zones[0]
	_, second, err := svc.ListControllers(context.Background())
	require.NoError(t, err)
	require.Equal(t, "77", first[0].ControllerId, "SDK v6 unique ID must be the persisted controller identity")
	require.Equal(t, first[0].ControllerId, second[0].ControllerId)
	require.Equal(t, zoneIDs(first[0]), zoneIDs(second[0]), "zone IDs must not depend on current SDK order")
}

func TestDuplicateZoneFingerprintsFailClosed(t *testing.T) {
	clock := scheduletest.New(time.Now())
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 4, Zones: []control.SDKZone{{Name: "Header", LEDCount: 4, Index: 0, Static: true}}}}}
	svc, _ := newService(t, backend, clock)
	_, controllers, err := svc.ListControllers(context.Background())
	require.NoError(t, err)
	_, err = svc.Enroll(context.Background(), "4", true)
	require.NoError(t, err)
	backend.controllers[0].Zones = []control.SDKZone{{Name: "Header", LEDCount: 4, Index: 0, Static: true}, {Name: "Header", LEDCount: 4, Index: 1, Static: true}}
	clock.Advance(6 * time.Second) // expire the bounded discovery cache before simulating a changed device map
	result, err := svc.SetZoneState(context.Background(), staticState("4", controllers[0].Zones[0].ZoneId, 1, 2, 3))
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_FAILED, result.Operation)
	require.Contains(t, result.Reason, control.ErrAmbiguousZone.Error())
}

func TestManualOffPriorityExpiryPersistenceAndCommandTruth(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC))
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 77, Name: "Board", Zones: []control.SDKZone{{Name: "Strip", LEDCount: 2, Index: 0, Static: true}}}}}
	svc, db := newService(t, backend, clock)
	status, controllers, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	require.Equal(t, controlv1.BackendReason_BACKEND_REASON_READY, status.Reason)
	require.Len(t, controllers, 1)
	id := controllers[0].ControllerId
	zone := controllers[0].Zones[0].ZoneId
	_, err = svc.Enroll(ctx, id, true)
	require.NoError(t, err)
	desired := staticState(id, zone, 255, 0, 0)
	result, err := svc.SetZoneState(ctx, desired)
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED, result.Operation)
	require.True(t, result.BackendAcknowledged)
	require.Zero(t, result.BackendAckStatus)
	require.Len(t, backend.writes, 1)
	require.Equal(t, uint32(77), backend.writes[0].controller)
	require.Equal(t, []control.RGB{{R: 255, G: 0, B: 0}, {R: 255, G: 0, B: 0}}, backend.writes[0].colors)
	expires := timestamppb.New(clock.Now().Add(time.Minute))
	override, offResults, err := svc.SetManualOff(ctx, true, nil, expires)
	require.NoError(t, err)
	require.True(t, override.Active)
	require.Len(t, offResults, 1)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED, offResults[0].Operation)
	require.True(t, offResults[0].BackendAcknowledged)
	require.True(t, offResults[0].LocalWriteCompleted)
	green := staticState(id, zone, 0, 255, 0)
	result, err = svc.SetZoneState(ctx, green)
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_DEFERRED, result.Operation)
	require.Contains(t, result.Reason, "manual Off")
	require.Len(t, backend.writes, 2, "manual Off must prevent lower-priority color commands")
	readback, override, err := svc.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, override.Active)
	require.True(t, readback[0].Desired.GetOff() != nil, "effective desired state reports manual Off")
	// A new repository over the same durable database must recover both the
	// requested state underneath the override and the override's expiry.
	repo := control.NewSQLiteRepository(db, clock)
	reloaded := control.NewService(repo, backend, clock)
	readback, override, err = reloaded.DesiredState(ctx)
	require.NoError(t, err)
	require.Equal(t, expires.AsTime(), override.ExpiresAt.AsTime())
	require.True(t, readback[0].Desired.GetOff() != nil)
	clock.Advance(time.Minute)
	readback, override, err = reloaded.DesiredState(ctx)
	require.NoError(t, err)
	require.False(t, override.Active)
	require.Equal(t, uint32(255), readback[0].Desired.GetStaticColor().Color.Green)
	_, err = reloaded.Reconcile(ctx)
	require.NoError(t, err)
	require.Len(t, backend.writes, 3, "expired override allows persisted desired state to resume")
	require.Equal(t, uint8(255), backend.writes[2].colors[0].G, "reconcile should send the persisted green state after expiry")
}

func TestDesiredStateRetriesAfterBackendReconnect(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC))
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 61, Name: "Board", Zones: []control.SDKZone{{Name: "Header", LEDCount: 1, Index: 0, Static: true}}}}}
	svc, db := newService(t, backend, clock)
	_, controllers, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	id, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
	_, err = svc.Enroll(ctx, id, true)
	require.NoError(t, err)
	// Simulate the scenario starting while the optional resource is disabled.
	noBackend := control.NewService(control.NewSQLiteRepository(db, clock), nil, clock)
	desired := staticState(id, zone, 20, 40, 60)
	result, err := noBackend.SetZoneState(ctx, desired)
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_DEFERRED, result.Operation)
	require.Empty(t, backend.writes, "no SDK operation occurs while the backend is absent")
	// The backend is enabled/reconnected later. One explicit reconcile refreshes
	// protocol v6 identities and retries the durable request.
	reconnected := control.NewService(control.NewSQLiteRepository(db, clock), backend, clock)
	results, err := reconnected.Reconcile(ctx)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED, results[0].Result.Operation)
	require.True(t, results[0].Result.BackendAcknowledged)
	require.Equal(t, []control.RGB{{R: 20, G: 40, B: 60}}, backend.writes[0].colors)
}

func TestManualOffRejectsAmbiguousScope(t *testing.T) {
	svc, _ := newService(t, nil, scheduletest.New(time.Now()))
	_, _, err := svc.SetManualOff(context.Background(), false, nil, nil)
	require.ErrorIs(t, err, control.ErrInvalidState)
	_, _, err = svc.SetManualOff(context.Background(), true, []string{"77"}, nil)
	require.ErrorIs(t, err, control.ErrInvalidState)
}

func TestListControllersSeparatesAbsentDevicesFromPersistedUnavailable(t *testing.T) {
	ctx := context.Background()
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 88, Name: "Controller", Zones: []control.SDKZone{{Name: "Zone", LEDCount: 1, Index: 0, Static: true}}}}}
	svc, _ := newService(t, backend, scheduletest.New(time.Now()))
	_, controllers, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	require.Len(t, controllers, 1)
	backend.controllers = nil
	backend.discoverErr = control.ErrNoDevice
	status, current, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	require.Empty(t, current, "default discovery excludes absent persisted devices")
	require.Equal(t, controlv1.BackendReason_BACKEND_REASON_PROBE_FAILED, status.Reason)
	require.Contains(t, status.Detail, "no OpenRGB controllers")
	_, includingUnavailable, err := svc.ListControllers(ctx, true)
	require.NoError(t, err)
	require.Len(t, includingUnavailable, 1)
}

func TestMalformedDesiredStateRejectedBeforePersistence(t *testing.T) {
	svc, _ := newService(t, nil, scheduletest.New(time.Now()))
	bad := staticState("77", "zone", 256, 0, 0)
	_, err := svc.SetZoneState(context.Background(), bad)
	require.ErrorIs(t, err, control.ErrInvalidState)
	states, _, err := svc.DesiredState(context.Background())
	require.NoError(t, err)
	require.Empty(t, states)
}

func TestUnsupportedEffectReturnsExplicitOutcome(t *testing.T) {
	ctx := context.Background()
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 9, Zones: []control.SDKZone{{Name: "Strip", LEDCount: 2, Index: 0, Static: true}}}}}
	svc, _ := newService(t, backend, scheduletest.New(time.Now()))
	_, controllers, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	_, err = svc.Enroll(ctx, "9", true)
	require.NoError(t, err)
	state := &controlv1.ZoneState{ControllerId: "9", ZoneId: controllers[0].Zones[0].ZoneId, DesiredMode: &controlv1.ZoneState_Effect{Effect: &controlv1.EffectZoneState{EffectId: "breathing", Color: &controlv1.Color{Red: 1, Green: 2, Blue: 3}, BrightnessPercent: 100}}}
	result, err := svc.SetZoneState(ctx, state)
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_UNSUPPORTED, result.Operation)
	require.Contains(t, result.Reason, control.ErrUnsupported.Error())
	require.Empty(t, backend.writes, "unsupported effects must not be sent as though static colors were equivalent")
}

func TestBackendErrorReasonsRemainDistinguishable(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		reason controlv1.BackendReason
		state  controlv1.BackendState
	}{
		{name: "stopped", err: control.ErrBackendUnavailable, reason: controlv1.BackendReason_BACKEND_REASON_STOPPED, state: controlv1.BackendState_BACKEND_STATE_UNAVAILABLE},
		{name: "protocol", err: control.ErrProtocolMismatch, reason: controlv1.BackendReason_BACKEND_REASON_PROTOCOL_INCOMPATIBLE, state: controlv1.BackendState_BACKEND_STATE_PROTOCOL_MISMATCH},
		{name: "permission", err: control.ErrPermission, reason: controlv1.BackendReason_BACKEND_REASON_PERMISSION_DENIED, state: controlv1.BackendState_BACKEND_STATE_UNAVAILABLE},
		{name: "probe", err: errors.New("backend probe failed"), reason: controlv1.BackendReason_BACKEND_REASON_PROBE_FAILED, state: controlv1.BackendState_BACKEND_STATE_UNAVAILABLE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := &fakeBackend{protocolErr: tc.err}
			svc, _ := newService(t, backend, scheduletest.New(time.Now()))
			got := svc.BackendStatus(context.Background())
			require.Equal(t, tc.reason, got.Reason)
			require.Equal(t, tc.state, got.State)
		})
	}
}

func staticState(controller, zone string, r, g, b uint32) *controlv1.ZoneState {
	return &controlv1.ZoneState{ControllerId: controller, ZoneId: zone, DesiredMode: &controlv1.ZoneState_StaticColor{StaticColor: &controlv1.StaticZoneState{Color: &controlv1.Color{Red: r, Green: g, Blue: b}, BrightnessPercent: 100}}}
}

func zoneIDs(c *controlv1.Controller) map[string]string {
	out := map[string]string{}
	for _, z := range c.Zones {
		out[z.Name] = z.ZoneId
	}
	return out
}
