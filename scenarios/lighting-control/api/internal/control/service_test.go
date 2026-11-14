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
	require.NoError(t, control.EnsureIdentitySource(context.Background(), db))
	repo := control.NewSQLiteRepository(db, clock)
	return control.NewService(repo, backend, clock), db
}

func TestDiscoveryUsesSerialIdentityAndZoneFingerprintsNotOrder(t *testing.T) {
	clock := scheduletest.New(time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC))
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 77, Name: "Board", Vendor: "Maker", Serial: "board-serial", Zones: []control.SDKZone{{Name: "Header", LEDCount: 4, Index: 0, Static: true}, {Name: "Strip", LEDCount: 8, Index: 1, Static: true}}}}}
	svc, _ := newService(t, backend, clock)
	_, first, err := svc.ListControllers(context.Background())
	require.NoError(t, err)
	firstID := first[0].ControllerId
	backend.controllers[0].Zones[0], backend.controllers[0].Zones[1] = backend.controllers[0].Zones[1], backend.controllers[0].Zones[0]
	backend.controllers[0].ID = 99
	_, second, err := svc.ListControllers(context.Background())
	require.NoError(t, err)
	require.Equal(t, firstID, second[0].ControllerId, "serial identity survives SDK handle changes")
	require.Equal(t, "openrgb_sdk_serial", second[0].IdentitySource)
	require.Equal(t, zoneIDs(first[0]), zoneIDs(second[0]), "zone IDs must not depend on current SDK order")
	_, err = svc.Enroll(context.Background(), second[0].ControllerId, true)
	require.NoError(t, err)
	result, err := svc.SetZoneState(context.Background(), staticState(second[0].ControllerId, second[0].Zones[0].ZoneId, 12, 34, 56))
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED, result.Operation)
	require.Equal(t, uint32(99), backend.writes[0].controller, "writes resolve the current transient SDK handle")
}

func TestPCIIdentityIgnoresTransientI2CBusNumberAndMutationsUseFreshHandle(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Now())
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{
		ID: 7, Type: 3, Name: "GPU", Vendor: "Vendor", Description: "Board", Location: "I2C: SMBus adapter at 0000:01:00.0 (/dev/i2c-4), address 0x50",
		Zones: []control.SDKZone{{Name: "GPU", LEDCount: 1, Index: 0, Static: true}},
	}}}
	svc, _ := newService(t, backend, clock)
	_, controllers, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	require.Len(t, controllers, 1)
	id, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
	require.Equal(t, "openrgb_linux_pci_i2c_location", controllers[0].IdentitySource)
	_, err = svc.Enroll(ctx, id, true)
	require.NoError(t, err)

	backend.controllers[0].ID = 42
	backend.controllers[0].Location = "I2C: SMBus adapter at 0000:01:00.0 (/dev/i2c-18), address 0x50"
	result, err := svc.SetZoneState(ctx, staticState(id, zone, 10, 20, 30))
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED, result.Operation)
	require.Equal(t, uint32(42), backend.writes[0].controller, "mutation must resolve the live transient SDK handle after bus renumbering")
	_, refreshed, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	require.Equal(t, id, refreshed[0].ControllerId, "allocated /dev/i2c number is not durable identity")
}

type modeAwareBackend struct {
	*fakeBackend
	modes []sentMode
}

type sentMode struct {
	controller uint32
	target     control.SDKModeTarget
	mode       control.SDKMode
}

type stableFakeBackend struct {
	*fakeBackend
	result      control.SDKCommandResult
	applyErr    error
	controller  string
	fingerprint string
}

type blockingStableBackend struct {
	*fakeBackend
	entered chan struct{}
	release chan struct{}
}

func (b *blockingStableBackend) ApplyStableZone(context.Context, string, string, *controlv1.ZoneState) (control.SDKCommandResult, error) {
	b.entered <- struct{}{}
	<-b.release
	return control.SDKCommandResult{LocalWriteCompleted: true, BackendAcknowledged: true}, nil
}

func (b *stableFakeBackend) ApplyStableZone(_ context.Context, controller, fingerprint string, _ *controlv1.ZoneState) (control.SDKCommandResult, error) {
	b.controller = controller
	b.fingerprint = fingerprint
	return b.result, b.applyErr
}

func TestStableBackendSeparatesAckFromCachedModeReadback(t *testing.T) {
	ctx := context.Background()
	base := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 13, Serial: "board-13", Zones: []control.SDKZone{{Name: "Strip", LEDCount: 2, Index: 0, Static: true}}}}}
	backend := &stableFakeBackend{fakeBackend: base, result: control.SDKCommandResult{LocalWriteCompleted: true, BackendAcknowledged: true, CachedModeReadback: controlv1.CachedModeReadback_CACHED_MODE_READBACK_MATCHES}}
	svc, _ := newService(t, backend, scheduletest.New(time.Now()))
	_, controllers, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	controller, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
	_, err = svc.Enroll(ctx, controller, true)
	require.NoError(t, err)
	result, err := svc.SetZoneState(ctx, staticState(controller, zone, 12, 34, 56))
	require.NoError(t, err)
	require.Equal(t, controller, backend.controller)
	require.Equal(t, "strip|2", backend.fingerprint)
	require.True(t, result.BackendAcknowledged)
	require.Equal(t, controlv1.CachedModeReadback_CACHED_MODE_READBACK_MATCHES, result.CachedModeReadback)

	backend.result.CachedModeReadback = controlv1.CachedModeReadback_CACHED_MODE_READBACK_MISMATCH
	backend.applyErr = control.ErrModeReadbackMismatch
	result, err = svc.SetZoneState(ctx, staticState(controller, zone, 56, 34, 12))
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_FAILED, result.Operation)
	require.True(t, result.BackendAcknowledged, "a successful ACK must remain visible despite a mismatching later cache readback")
	require.Equal(t, controlv1.CachedModeReadback_CACHED_MODE_READBACK_MISMATCH, result.CachedModeReadback)
}

func TestDesiredStateMutationsUseOneSerializedWriter(t *testing.T) {
	ctx := context.Background()
	base := &fakeBackend{protocol: 6, controllers: []control.SDKController{{Serial: "writer-serial", Zones: []control.SDKZone{{Name: "Strip", LEDCount: 2, Index: 0, Static: true}}}}}
	backend := &blockingStableBackend{fakeBackend: base, entered: make(chan struct{}, 2), release: make(chan struct{}, 2)}
	svc, _ := newService(t, backend, scheduletest.New(time.Now()))
	_, controllers, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	id, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
	_, err = svc.Enroll(ctx, id, true)
	require.NoError(t, err)

	firstDone := make(chan error, 1)
	go func() {
		_, err := svc.SetZoneState(ctx, staticState(id, zone, 1, 2, 3))
		firstDone <- err
	}()
	select {
	case <-backend.entered:
	case <-time.After(time.Second):
		t.Fatal("first desired-state writer did not reach the backend")
	}
	secondDone := make(chan error, 1)
	go func() {
		_, err := svc.SetZoneState(ctx, staticState(id, zone, 4, 5, 6))
		secondDone <- err
	}()
	select {
	case <-backend.entered:
		t.Fatal("second writer entered the backend before the first persisted its result")
	case <-time.After(25 * time.Millisecond):
	}
	backend.release <- struct{}{}
	select {
	case <-backend.entered:
	case <-time.After(time.Second):
		t.Fatal("second desired-state writer did not proceed after the first completed")
	}
	backend.release <- struct{}{}
	require.NoError(t, <-firstDone)
	require.NoError(t, <-secondDone)
}

func (b *modeAwareBackend) SendMode(_ context.Context, controller uint32, target control.SDKModeTarget, mode control.SDKMode) (control.SDKCommandResult, error) {
	b.modes = append(b.modes, sentMode{controller: controller, target: target, mode: mode})
	return control.SDKCommandResult{LocalWriteCompleted: true, BackendAcknowledged: true}, nil
}

func TestNativeModeUsesExactZoneAndManualOffExitsEffect(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Now())
	backend := &modeAwareBackend{fakeBackend: &fakeBackend{protocol: 6, controllers: []control.SDKController{{
		ID: 12, Serial: "gpu-12", LEDCount: 2,
		Zones: []control.SDKZone{{Name: "GPU", LEDCount: 2, Index: 0, Static: true, ActiveMode: 1, Modes: []control.SDKMode{
			{Name: "Static"},
			{Name: "Breathing", SDKModeParameters: control.SDKModeParameters{Flags: 1 | 1<<4, BrightnessMin: 0, BrightnessMax: 100}},
		}}},
	}}}}
	svc, _ := newService(t, backend, clock)
	_, controllers, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	id, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
	_, err = svc.Enroll(ctx, id, true)
	require.NoError(t, err)
	effect := &controlv1.ZoneState{ControllerId: id, ZoneId: zone, DesiredMode: &controlv1.ZoneState_Effect{Effect: &controlv1.EffectZoneState{EffectId: "Breathing", Color: &controlv1.Color{Red: 1, Green: 2, Blue: 3}, BrightnessPercent: 20}}}
	result, err := svc.SetZoneState(ctx, effect)
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED, result.Operation)
	require.Len(t, backend.modes, 1)
	require.NotNil(t, backend.modes[0].target.ZoneIndex, "zone mode must retain exact zone scope")
	require.Equal(t, uint32(0), *backend.modes[0].target.ZoneIndex)
	require.Equal(t, "Breathing", backend.modes[0].mode.Name)
	require.Empty(t, backend.writes, "native effect must not be overwritten by an LED color packet")

	_, offResults, err := svc.SetManualOff(ctx, false, []string{id}, nil)
	require.NoError(t, err)
	require.Len(t, offResults, 1)
	require.Equal(t, "Static", backend.modes[1].mode.Name, "Off must exit the active effect before writing black")
	require.Len(t, backend.writes, 1)
}

func TestDuplicateZoneFingerprintsFailClosed(t *testing.T) {
	clock := scheduletest.New(time.Now())
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 4, Serial: "board-4", Zones: []control.SDKZone{{Name: "Header", LEDCount: 4, Index: 0, Static: true}}}}}
	svc, _ := newService(t, backend, clock)
	_, controllers, err := svc.ListControllers(context.Background())
	require.NoError(t, err)
	_, err = svc.Enroll(context.Background(), controllers[0].ControllerId, true)
	require.NoError(t, err)
	backend.controllers[0].Zones = []control.SDKZone{{Name: "Header", LEDCount: 4, Index: 0, Static: true}, {Name: "Header", LEDCount: 4, Index: 1, Static: true}}
	result, err := svc.SetZoneState(context.Background(), staticState(controllers[0].ControllerId, controllers[0].Zones[0].ZoneId, 1, 2, 3))
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_FAILED, result.Operation)
	require.Contains(t, result.Reason, control.ErrAmbiguousZone.Error())
}

func TestControllersWithoutUniqueStableIdentityCannotBeEnrolled(t *testing.T) {
	clock := scheduletest.New(time.Now())
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{
		{ID: 12, Name: "Controller", Vendor: "Maker", Zones: []control.SDKZone{{Name: "Zone", LEDCount: 1, Static: true}}},
		{ID: 13, Name: "Controller", Vendor: "Maker", Zones: []control.SDKZone{{Name: "Zone", LEDCount: 1, Static: true}}},
	}}
	svc, _ := newService(t, backend, clock)
	_, controllers, err := svc.ListControllers(context.Background())
	require.NoError(t, err)
	require.Len(t, controllers, 2)
	for _, controller := range controllers {
		require.Contains(t, []string{"openrgb_sdk_transient_handle", "openrgb_identity_ambiguous"}, controller.IdentitySource)
		_, err := svc.Enroll(context.Background(), controller.ControllerId, true)
		require.ErrorIs(t, err, control.ErrUnstableIdentity)
	}
	require.Empty(t, backend.writes)
}

func TestManualOffPriorityExpiryPersistenceAndCommandTruth(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC))
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 77, Name: "Board", Serial: "board-77", Zones: []control.SDKZone{{Name: "Strip", LEDCount: 2, Index: 0, Static: true}}}}}
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
	require.Equal(t, []control.RGB{{R: 89, G: 0, B: 0}, {R: 89, G: 0, B: 0}}, backend.writes[0].colors)
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
	require.Equal(t, green.String(), readback[0].Desired.String(), "manual Off is reported separately and must preserve base desired state")
	// A new repository over the same durable database must recover both the
	// requested state underneath the override and the override's expiry.
	repo := control.NewSQLiteRepository(db, clock)
	reloaded := control.NewService(repo, backend, clock)
	readback, override, err = reloaded.DesiredState(ctx)
	require.NoError(t, err)
	require.Equal(t, expires.AsTime(), override.ExpiresAt.AsTime())
	require.Equal(t, green.String(), readback[0].Desired.String(), "base desired state survives process restart while Off is active")
	clock.Advance(time.Minute)
	readback, override, err = reloaded.DesiredState(ctx)
	require.NoError(t, err)
	require.False(t, override.Active)
	require.Equal(t, uint32(255), readback[0].Desired.GetStaticColor().Color.Green)
	_, err = reloaded.Reconcile(ctx)
	require.NoError(t, err)
	require.Len(t, backend.writes, 3, "expired override allows persisted desired state to resume")
	require.Equal(t, uint8(89), backend.writes[2].colors[0].G, "reconcile should send the persisted green state after expiry")
}

func TestPolicyApplicationRetainsManualZoneDesiredState(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Now())
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 31, Serial: "policy-base", Zones: []control.SDKZone{{Name: "Strip", LEDCount: 2, Index: 0, Static: true}}}}}
	svc, _ := newService(t, backend, clock)
	_, controllers, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	id, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
	_, err = svc.Enroll(ctx, id, true)
	require.NoError(t, err)
	manual := staticState(id, zone, 0, 255, 0)
	_, err = svc.SetZoneState(ctx, manual)
	require.NoError(t, err)
	policy := staticState(id, zone, 255, 0, 0)
	for range 3 {
		result, err := svc.ApplyPolicyState(ctx, policy)
		require.NoError(t, err)
		require.Equal(t, manual.GetStaticColor().Color, result.Desired.GetStaticColor().Color, "manual color must govern every policy tick")
		require.Equal(t, uint32(35), result.Desired.GetStaticColor().BrightnessPercent)
		require.Equal(t, uint8(89), backend.writes[len(backend.writes)-1].colors[0].G)
	}
	zones, _, err := svc.DesiredState(ctx)
	require.NoError(t, err)
	require.Equal(t, manual.String(), zones[0].Desired.String(), "automatic policy output must not replace the saved manual zone intent")
	require.Equal(t, uint32(35), zones[0].LastApply.Desired.GetStaticColor().BrightnessPercent, "command evidence describes capped emission")

	_, _, err = svc.SetManualOff(ctx, true, nil, nil)
	require.NoError(t, err)
	policy = staticState(id, zone, 0, 0, 255)
	writes := len(backend.writes)
	result, err := svc.ApplyPolicyState(ctx, policy)
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_DEFERRED, result.Operation)
	require.Len(t, backend.writes, writes, "manual Off must retain precedence over policy output")
	zones, override, err := svc.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, override.Active)
	require.Equal(t, manual.String(), zones[0].Desired.String(), "policy and Off outcomes must leave base manual intent intact")
}

func TestDesiredStateRetriesAfterBackendReconnect(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC))
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 61, Name: "Board", Serial: "board-61", Zones: []control.SDKZone{{Name: "Header", LEDCount: 1, Index: 0, Static: true}}}}}
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
	require.Equal(t, []control.RGB{{R: 7, G: 14, B: 21}}, backend.writes[0].colors)
}

func TestManualOffRetriesWhenBackendReturnsAfterBoot(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC))
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 62, Name: "Board", Serial: "board-62", Zones: []control.SDKZone{{Name: "Header", LEDCount: 1, Index: 0, Static: true}}}}}
	initial, db := newService(t, backend, clock)
	_, controllers, err := initial.ListControllers(ctx)
	require.NoError(t, err)
	id := controllers[0].ControllerId
	_, err = initial.Enroll(ctx, id, true)
	require.NoError(t, err)
	backend.writes = nil

	// Manual Off is durable even when the optional backend is stopped.
	stopped := control.NewService(control.NewSQLiteRepository(db, clock), nil, clock)
	override, results, err := stopped.SetManualOff(ctx, false, []string{id}, nil)
	require.NoError(t, err)
	require.True(t, override.Active)
	require.Len(t, results, 1)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_DEFERRED, results[0].Operation)
	require.Empty(t, backend.writes)

	// The runtime startup reconcile must turn the durable override into an Off
	// command once the backend is ready again.
	reconnected := control.NewService(control.NewSQLiteRepository(db, clock), backend, clock)
	reconciled, err := reconnected.Reconcile(ctx)
	require.NoError(t, err)
	require.Len(t, reconciled, 1)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED, reconciled[0].Result.Operation)
	require.Equal(t, []control.RGB{{}}, backend.writes[0].colors)
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
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 88, Name: "Controller", Serial: "board-88", Zones: []control.SDKZone{{Name: "Zone", LEDCount: 1, Index: 0, Static: true}}}}}
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
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 9, Serial: "board-9", Zones: []control.SDKZone{{Name: "Strip", LEDCount: 2, Index: 0, Static: true}}}}}
	svc, _ := newService(t, backend, scheduletest.New(time.Now()))
	_, controllers, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	_, err = svc.Enroll(ctx, controllers[0].ControllerId, true)
	require.NoError(t, err)
	state := &controlv1.ZoneState{ControllerId: controllers[0].ControllerId, ZoneId: controllers[0].Zones[0].ZoneId, DesiredMode: &controlv1.ZoneState_Effect{Effect: &controlv1.EffectZoneState{EffectId: "breathing", Color: &controlv1.Color{Red: 1, Green: 2, Blue: 3}, BrightnessPercent: 100}}}
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

func TestReconcileRejectsUnavailableDiscoveryWithNoDesiredZones(t *testing.T) {
	backend := &fakeBackend{protocol: 6, discoverErr: errors.New("fixture discovery transport failed")}
	svc, _ := newService(t, backend, scheduletest.New(time.Now()))
	results, err := svc.Reconcile(context.Background())
	require.Error(t, err, "nil desired rows must not turn discovery failure into successful policy reconciliation")
	require.Empty(t, results)
	require.Empty(t, backend.writes)
}
