package control_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	control "lighting-control/internal/control"
)

type configuredBackend struct {
	fakeBackend
	lengths    []uint32
	states     []*controlv1.ZoneState
	fail       bool
	nackStatus uint32
}

func (b *configuredBackend) ApplyStableZone(context.Context, string, string, *controlv1.ZoneState) (control.SDKCommandResult, error) {
	panic("configured writer required")
}
func (b *configuredBackend) ApplyConfiguredStableZone(_ context.Context, _, _ string, count uint32, _ map[string]uint32, d *controlv1.ZoneState) (control.SDKCommandResult, error) {
	b.lengths = append(b.lengths, count)
	b.states = append(b.states, d)
	if b.nackStatus != 0 {
		return control.SDKCommandResult{LocalWriteCompleted: true, AckStatus: b.nackStatus}, &control.SDKAckError{PacketID: 1000, Status: b.nackStatus}
	}
	if b.fail {
		return control.SDKCommandResult{}, errors.New("disconnected")
	}
	b.controllers[0].Zones[0].LEDCount = count
	return control.SDKCommandResult{LocalWriteCompleted: true, BackendAcknowledged: true}, nil
}
func TestExplicitZoneLengthPersistsReplaysReconnectAndPreservesOff(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Now())
	b := &configuredBackend{fakeBackend: fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 7, Name: "Terminal", Serial: "terminal", Zones: []control.SDKZone{{Name: "Header 1", LEDMin: 0, LEDMax: 500, Flags: 2, Static: true}}}}}}
	s, db := newService(t, b, clock)
	_, cs, err := s.ListControllers(ctx)
	require.NoError(t, err)
	id, z := cs[0].ControllerId, cs[0].Zones[0].ZoneId
	require.True(t, cs[0].Zones[0].SetupNeeded)
	_, _, err = s.ConfigureZone(ctx, id, z, 20)
	require.ErrorIs(t, err, control.ErrNotEnrolled)
	_, err = s.Enroll(ctx, id, true)
	require.NoError(t, err)
	for _, count := range []uint32{0, 501, 65536} {
		_, _, err = s.ConfigureZone(ctx, id, z, count)
		require.ErrorIs(t, err, control.ErrInvalidState)
	}
	result, err := s.SetZoneState(ctx, staticState(id, z, 1, 2, 3))
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_DEFERRED, result.Operation)
	require.Empty(t, b.lengths)
	zone, results, err := s.ConfigureZone(ctx, id, z, 30)
	require.NoError(t, err)
	require.False(t, zone.SetupNeeded)
	require.Equal(t, uint32(30), zone.ConfiguredLedCount)
	require.Len(t, results, 1)
	require.Equal(t, []uint32{30}, b.lengths)
	require.NotNil(t, b.states[0].GetStaticColor())
	_, _, err = s.SetManualOff(ctx, true, nil, nil)
	require.NoError(t, err)
	// Restart the service, change transient SDK handles, and lose backend config.
	b.controllers[0].ID = 88
	b.controllers[0].Zones[0].LEDCount = 0
	resumed := control.NewService(control.NewSQLiteRepository(db, clock), b, clock)
	_, cs, err = resumed.ListControllers(ctx)
	require.NoError(t, err)
	require.Equal(t, z, cs[0].Zones[0].ZoneId)
	require.True(t, cs[0].Zones[0].SetupNeeded)
	results, err = resumed.Reconcile(ctx)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NotNil(t, b.states[len(b.states)-1].GetOff())
	require.Equal(t, uint32(30), b.lengths[len(b.lengths)-1])
	zone, _, err = resumed.ConfigureZone(ctx, id, z, 40)
	require.NoError(t, err)
	require.Equal(t, uint32(40), zone.ConfiguredLedCount)
	require.NotNil(t, b.states[len(b.states)-1].GetOff())
	// A failed replay keeps intent and cannot acknowledge output.
	b.fail = true
	_, results, err = resumed.ConfigureZone(ctx, id, z, 50)
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_FAILED, results[0].Result.Operation)
	_, cs, err = resumed.ListControllers(ctx)
	require.NoError(t, err)
	require.Equal(t, uint32(50), cs[0].Zones[0].ConfiguredLedCount)
	require.True(t, cs[0].Zones[0].SetupNeeded)
}
func TestLegacyResizableZoneRetainsIDAndDesiredRows(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Now())
	b := &configuredBackend{fakeBackend: fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 7, Name: "Terminal", Serial: "terminal", Zones: []control.SDKZone{{Name: "Header 1", LEDCount: 12, Static: true}}}}}}
	s, db := newService(t, b, clock)
	_, cs, err := s.ListControllers(ctx)
	require.NoError(t, err)
	id, z := cs[0].ControllerId, cs[0].Zones[0].ZoneId
	_, err = s.Enroll(ctx, id, true)
	require.NoError(t, err)
	_, err = s.SetZoneState(ctx, staticState(id, z, 1, 2, 3))
	require.NoError(t, err)
	// Database from before SDK limits/flags were retained.
	b.controllers[0].Zones[0].LEDMax = 500
	b.controllers[0].Zones[0].Flags = 2
	_, cs, err = s.ListControllers(ctx)
	require.NoError(t, err)
	require.Equal(t, z, cs[0].Zones[0].ZoneId)
	zone, _, err := s.ConfigureZone(ctx, id, z, 24)
	require.NoError(t, err)
	require.Equal(t, z, zone.ZoneId)
	var desired, owned int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM lighting_desired_state WHERE zone_id=?`, z).Scan(&desired))
	require.Equal(t, 1, desired)
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM lighting_zone_ownership WHERE zone_id=? AND source='manual'`, z).Scan(&owned))
	require.Equal(t, 1, owned)
	b.controllers[0].Zones[0].LEDCount = 0
	_, cs, err = s.ListControllers(ctx)
	require.NoError(t, err)
	require.Equal(t, z, cs[0].Zones[0].ZoneId)
}

func TestAdversarialMigrationDoesNotStealExistingFixedZone(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Now())
	original := control.SDKZone{Name: "Header", LEDCount: 12, Index: 0, Static: true}
	b := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 7, Serial: "a-review", Zones: []control.SDKZone{original}}}}
	svc, _ := newService(t, b, clock)
	_, before, err := svc.ListControllers(ctx)
	require.NoError(t, err)
	oldID := before[0].Zones[0].ZoneId
	b.controllers[0].Zones = append(b.controllers[0].Zones, control.SDKZone{Name: "Header", LEDMin: 0, LEDMax: 500, Flags: 2, Index: 1, Static: true})
	_, after, err := svc.ListControllers(ctx)
	if err != nil {
		require.ErrorIs(t, err, control.ErrAmbiguousZone)
		return
	}
	require.Len(t, after[0].Zones, 2, "both physical zones must remain represented")
	for _, z := range after[0].Zones {
		if z.ZoneId == oldID {
			require.EqualValues(t, 12, z.LedCount, "legacy fixed zone ID must not silently rebind to new configurable zone")
		}
	}
}

func TestPublicInventoryRetainsAdvertisedZoneEffectsAcrossStorage(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Now())
	b := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 7, Serial: "effect-fixture", Zones: []control.SDKZone{{Name: "Header", LEDCount: 4, Static: true, Modes: []control.SDKMode{{Name: "Static"}, {Name: "Breathing"}, {Name: "Rainbow"}}}}}}}
	s, db := newService(t, b, clock)
	_, cs, err := s.ListControllers(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"Breathing", "Rainbow"}, cs[0].Zones[0].SupportedEffects)
	// Persisted inventory remains meaningful while the optional backend is absent.
	resumed := control.NewService(control.NewSQLiteRepository(db, clock), nil, clock)
	_, cs, err = resumed.ListControllers(ctx, true)
	require.NoError(t, err)
	require.Equal(t, []string{"Breathing", "Rainbow"}, cs[0].Zones[0].SupportedEffects)
}

func TestConfigurationRejectsImpossibleAggregateBeforePersistence(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Now())
	b := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 7, Serial: "large-fixture", Zones: []control.SDKZone{{Name: "Fixed", LEDCount: 40000, Static: true}, {Name: "Header", LEDMin: 1, LEDMax: 65535, Flags: 2, Static: true}}}}}
	s, db := newService(t, b, clock)
	_, cs, err := s.ListControllers(ctx)
	require.NoError(t, err)
	id := cs[0].ControllerId
	_, err = s.Enroll(ctx, id, true)
	require.NoError(t, err)
	for _, z := range cs[0].Zones {
		if z.Name == "Header" {
			_, _, err = s.ConfigureZone(ctx, id, z.ZoneId, 30000)
			require.ErrorIs(t, err, control.ErrInvalidState)
		}
	}
	var rows int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM lighting_zone_configuration`).Scan(&rows))
	require.Zero(t, rows)
	require.Empty(t, b.writes)
}

func TestResizeRejectionIsFailedRatherThanUnacknowledgedPending(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Now())
	b := &configuredBackend{fakeBackend: fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 7, Serial: "nack-fixture", Zones: []control.SDKZone{{Name: "Header", LEDMax: 500, Flags: 2, Static: true}}}}}, nackStatus: 3}
	s, _ := newService(t, b, clock)
	_, cs, err := s.ListControllers(ctx)
	require.NoError(t, err)
	id, z := cs[0].ControllerId, cs[0].Zones[0].ZoneId
	_, err = s.Enroll(ctx, id, true)
	require.NoError(t, err)
	_, results, err := s.ConfigureZone(ctx, id, z, 24)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_FAILED, results[0].Result.Operation)
	require.Equal(t, uint32(3), results[0].Result.BackendAckStatus)
	require.True(t, results[0].Result.LocalWriteCompleted)
	require.False(t, results[0].Result.BackendAcknowledged)
}

func TestInitialOffPersistenceFailureDoesNotPublishUsableLength(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Now())
	b := &configuredBackend{fakeBackend: fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 7, Serial: "off-failure-fixture", Zones: []control.SDKZone{{Name: "Header", LEDMax: 500, Flags: 2, Static: true}}}}}}
	s, db := newService(t, b, clock)
	_, cs, err := s.ListControllers(ctx)
	require.NoError(t, err)
	id, z := cs[0].ControllerId, cs[0].Zones[0].ZoneId
	_, err = s.Enroll(ctx, id, true)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER fail_initial_off BEFORE INSERT ON lighting_desired_state BEGIN SELECT RAISE(ABORT,'initial Off storage unavailable'); END`)
	require.NoError(t, err)
	_, _, err = s.ConfigureZone(ctx, id, z, 24)
	require.ErrorContains(t, err, "initial Off storage unavailable")
	var rows int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM lighting_zone_configuration`).Scan(&rows))
	require.Zero(t, rows)
	require.Empty(t, b.lengths)
}
