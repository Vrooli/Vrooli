package policy

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	"google.golang.org/protobuf/proto"
	control "lighting-control/internal/control"
	internalpolicy "lighting-control/internal/policy"
)

type brightnessBackend struct{ writes [][]control.RGB }

func (*brightnessBackend) ProtocolVersion(context.Context) (uint32, error) { return 6, nil }
func (*brightnessBackend) Discover(context.Context) ([]control.SDKController, error) {
	return []control.SDKController{{Serial: "brightness-fixture", Zones: []control.SDKZone{{Name: "Strip", LEDCount: 1, Static: true}}}}, nil
}
func (b *brightnessBackend) SendZoneColors(_ context.Context, _, _ uint32, colors []control.RGB) (control.SDKCommandResult, error) {
	b.writes = append(b.writes, append([]control.RGB(nil), colors...))
	return control.SDKCommandResult{LocalWriteCompleted: true, BackendAcknowledged: true}, nil
}

func TestMaximumBrightnessCapsManualIntentAndReappliesConfigWithoutOwnershipChange(t *testing.T) {
	ctx := t.Context()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(control.Schema), database.SchemaProviderFunc(internalpolicy.Schema)))
	require.NoError(t, control.EnsureIdentitySource(ctx, db))
	clock := scheduletest.New(time.Now())
	backend := &brightnessBackend{}
	repo := control.NewSQLiteRepository(db, clock)
	writer := control.NewService(repo, backend, clock)
	store := internalpolicy.NewSQLiteStore(db)
	service := NewService(store, nil, writer)
	_, controllers, err := writer.ListControllers(ctx)
	require.NoError(t, err)
	id, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
	_, err = writer.Enroll(ctx, id, true)
	require.NoError(t, err)
	desired := &controlv1.ZoneState{ControllerId: id, ZoneId: zone, DesiredMode: &controlv1.ZoneState_StaticColor{StaticColor: &controlv1.StaticZoneState{Color: &controlv1.Color{Red: 200}, BrightnessPercent: 80}}}
	result, err := writer.SetZoneState(ctx, desired)
	require.NoError(t, err)
	require.Equal(t, uint32(35), result.Desired.GetStaticColor().BrightnessPercent)
	require.Contains(t, result.Reason, "maximum brightness cap applied")
	require.Equal(t, uint8(70), backend.writes[len(backend.writes)-1][0].R)
	require.Equal(t, uint32(80), desired.GetStaticColor().BrightnessPercent, "caller intent is immutable")
	for _, cap := range []uint32{10, 70, 0, 100} {
		config := internalpolicy.DefaultConfig()
		config.BrightnessCapPercent = cap
		_, err = service.SetConfig(ctx, config)
		require.NoError(t, err)
		want := uint8(200 * min(cap, 80) / 100)
		require.Equal(t, want, backend.writes[len(backend.writes)-1][0].R)
		rows, _, err := writer.DesiredState(ctx)
		require.NoError(t, err)
		require.True(t, proto.Equal(desired, rows[0].Desired), "cap edits retain full saved intent")
		source, err := repo.DesiredSource(ctx, id, zone)
		require.NoError(t, err)
		require.Equal(t, "manual", source)
	}
	_, _, err = writer.SetManualOff(ctx, true, nil, nil)
	require.NoError(t, err)
	config := internalpolicy.DefaultConfig()
	config.BrightnessCapPercent = 5
	_, err = service.SetConfig(ctx, config)
	require.NoError(t, err)
	require.Equal(t, control.RGB{}, backend.writes[len(backend.writes)-1][0])
	_, override, err := writer.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, override.Active)
	require.NoError(t, writer.ClearManualOverride(ctx))
	require.Equal(t, uint8(10), backend.writes[len(backend.writes)-1][0].R)
	// A reopened service reads the persisted cap rather than a process-local copy.
	reopened := control.NewService(repo, backend, clock)
	reopened.BindBrightnessPolicy(internalpolicy.NewSQLiteStore(db))
	_, err = reopened.Reconcile(ctx)
	require.NoError(t, err)
	require.Equal(t, uint8(10), backend.writes[len(backend.writes)-1][0].R)
	// Cap edits also evaluate fresh automatic output instead of replaying stale
	// saved policy commands or leaving the former higher command in effect.
	service.sources = fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: clock.Now()}}
	service.now = clock.Now
	_, _, err = service.SetManual(ctx, policyv1.PresetKind_PRESET_KIND_STATIC, time.Time{})
	require.NoError(t, err)
	config = internalpolicy.DefaultConfig()
	config.BrightnessCapPercent = 10
	for _, preset := range config.Presets {
		if preset.Kind == policyv1.PresetKind_PRESET_KIND_STATIC {
			preset.BrightnessCapPercent = 90
			preset.Color = &controlv1.Color{Red: 200}
		}
	}
	_, err = service.SetConfig(ctx, config)
	require.NoError(t, err)
	require.Equal(t, uint8(20), backend.writes[len(backend.writes)-1][0].R)
	source, err := repo.DesiredSource(ctx, id, zone)
	require.NoError(t, err)
	require.Equal(t, "policy", source)
}

type brightnessModeBackend struct {
	brightnessBackend
	modes  []string
	active int32
}

func (b *brightnessModeBackend) Discover(context.Context) ([]control.SDKController, error) {
	return []control.SDKController{{Serial: "brightness-effect-fixture", LEDCount: 1, Zones: []control.SDKZone{{Name: "Strip", LEDCount: 1, Static: true, ActiveMode: b.active, Modes: []control.SDKMode{{Name: "Static"}, {Name: "Breathing"}}}}}}, nil
}
func (b *brightnessModeBackend) SendMode(_ context.Context, _ uint32, target control.SDKModeTarget, mode control.SDKMode) (control.SDKCommandResult, error) {
	b.modes = append(b.modes, mode.Name)
	b.active = int32(target.ModeIndex)
	return control.SDKCommandResult{LocalWriteCompleted: true, BackendAcknowledged: true}, nil
}

func TestLoweringCapStopsExistingUnboundedEffectBeforeReturning(t *testing.T) {
	ctx := t.Context()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(control.Schema), database.SchemaProviderFunc(internalpolicy.Schema)))
	require.NoError(t, control.EnsureIdentitySource(ctx, db))
	clock := scheduletest.New(time.Now())
	backend := &brightnessModeBackend{}
	repo := control.NewSQLiteRepository(db, clock)
	writer := control.NewService(repo, backend, clock)
	store := internalpolicy.NewSQLiteStore(db)
	service := NewService(store, nil, writer)
	config := internalpolicy.DefaultConfig()
	config.BrightnessCapPercent = 100
	_, err := service.SetConfig(ctx, config)
	require.NoError(t, err)
	_, controllers, err := writer.ListControllers(ctx)
	require.NoError(t, err)
	id, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
	_, err = writer.Enroll(ctx, id, true)
	require.NoError(t, err)
	desired := &controlv1.ZoneState{ControllerId: id, ZoneId: zone, DesiredMode: &controlv1.ZoneState_Effect{Effect: &controlv1.EffectZoneState{EffectId: "Breathing", Color: &controlv1.Color{Red: 255}, BrightnessPercent: 100}}}
	result, err := writer.SetZoneState(ctx, desired)
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED, result.Operation)
	require.Equal(t, []string{"Breathing"}, backend.modes)
	config.BrightnessCapPercent = 35
	_, err = service.SetConfig(ctx, config)
	require.NoError(t, err)
	require.Equal(t, []string{"Breathing", "Static"}, backend.modes, "lowered cap must exit the prior unlimited effect")
	require.Equal(t, [][]control.RGB{{{}}}, backend.writes, "Off must be sent before SetConfig returns")
	rows, override, err := writer.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, proto.Equal(desired, rows[0].Desired), "retain the effect for a future supported configuration")
	require.NotNil(t, rows[0].LastApply.Desired.GetOff())
	require.Contains(t, rows[0].LastApply.Reason, "static Off fallback")
	require.True(t, rows[0].LastApply.BackendAcknowledged)
	require.False(t, override.Active, "fallback must not publish a new ownership fence")
	source, err := repo.DesiredSource(ctx, id, zone)
	require.NoError(t, err)
	require.Equal(t, "manual", source)
	config.BrightnessCapPercent = 100
	_, err = service.SetConfig(ctx, config)
	require.NoError(t, err)
	require.Equal(t, []string{"Breathing", "Static", "Breathing"}, backend.modes)
}
