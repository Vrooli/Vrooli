package policy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
	"github.com/vrooli/api-core/storage"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	"google.golang.org/protobuf/proto"
	internalcontrol "lighting-control/internal/control"
	internalpolicy "lighting-control/internal/policy"
)

type memoryPolicyStore struct {
	config     *policyv1.PolicyConfig
	state      internalpolicy.State
	saveCalls  int
	failSaveAt int
}

func (s *memoryPolicyStore) Load(context.Context) (*policyv1.PolicyConfig, internalpolicy.State, error) {
	if s.config == nil {
		return nil, internalpolicy.State{}, internalpolicy.ErrNotFound
	}
	return s.config, s.state, nil
}

func (s *memoryPolicyStore) Save(_ context.Context, config *policyv1.PolicyConfig, state internalpolicy.State) error {
	s.saveCalls++
	if s.failSaveAt == s.saveCalls {
		return errors.New("fixture policy save failure")
	}
	if config == nil {
		return errors.New("config is required")
	}
	s.config, s.state = config, state
	return nil
}

type fixedPolicyInputs struct{ inputs internalpolicy.Inputs }

func (s fixedPolicyInputs) Read(context.Context) internalpolicy.Inputs { return s.inputs }

func TestSetConfigPersistsDraftWithoutEvaluatingOrChangingActivePreset(t *testing.T) {
	store := &memoryPolicyStore{config: internalpolicy.DefaultConfig(), state: internalpolicy.State{ManualPreset: policyv1.PresetKind_PRESET_KIND_OFF}}
	service := NewService(store, nil, nil)
	updated := internalpolicy.DefaultConfig()
	updated.Presets[policyv1.PresetKind_PRESET_KIND_COMBINED-1].Label = "Quiet workbench"

	if _, err := service.SetConfig(context.Background(), updated); err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
	if store.config.Presets[policyv1.PresetKind_PRESET_KIND_COMBINED-1].Label != "Quiet workbench" {
		t.Fatal("draft config was not persisted")
	}
	if store.state.ManualPreset != policyv1.PresetKind_PRESET_KIND_OFF {
		t.Fatalf("active manual preset changed to %v", store.state.ManualPreset)
	}
}

func TestSelectingNonOffPresetClearsManualOffButSelectingOffPreservesIt(t *testing.T) {
	ctx := context.Background()
	clock := scheduletest.New(time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC))
	db := dbtest.NewSQLite(t)
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(internalcontrol.Schema)); err != nil {
		t.Fatalf("ensure control schema: %v", err)
	}
	if err := internalcontrol.EnsureIdentitySource(ctx, db); err != nil {
		t.Fatalf("ensure identity source: %v", err)
	}
	controlService := internalcontrol.NewService(internalcontrol.NewSQLiteRepository(db, clock), nil, clock)
	if _, _, err := controlService.SetManualOff(ctx, true, nil, nil); err != nil {
		t.Fatalf("set manual Off: %v", err)
	}
	store := &memoryPolicyStore{config: internalpolicy.DefaultConfig()}
	service := NewService(store, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: clock.Now()}}, controlService)
	service.now = clock.Now
	if _, _, err := service.SetManual(ctx, policyv1.PresetKind_PRESET_KIND_HEALTH, time.Time{}); err != nil {
		t.Fatalf("select non-Off preset: %v", err)
	}
	_, override, err := controlService.DesiredState(ctx)
	if err != nil {
		t.Fatalf("read control override after preset selection: %v", err)
	}
	if override.Active {
		t.Fatal("selecting a non-Off preset must clear the explicit manual Off override")
	}
	if _, _, err := controlService.SetManualOff(ctx, true, nil, nil); err != nil {
		t.Fatalf("restore manual Off: %v", err)
	}
	if _, _, err := service.SetManual(ctx, policyv1.PresetKind_PRESET_KIND_OFF, time.Time{}); err != nil {
		t.Fatalf("select Off preset: %v", err)
	}
	_, override, err = controlService.DesiredState(ctx)
	if err != nil {
		t.Fatalf("read control override after Off selection: %v", err)
	}
	if !override.Active {
		t.Fatal("selecting Off must not clear the higher-priority explicit manual Off override")
	}
}

func TestDesiredForZoneUsesOffModeWithoutClaimingFallback(t *testing.T) {
	effective := &policyv1.EffectivePolicy{Preset: policyv1.PresetKind_PRESET_KIND_OFF}
	desired, fallback := desiredForZone("controller-1", &controlv1.ZoneCapability{ZoneId: "zone-1"}, internalpolicy.DefaultConfig(), effective)
	if _, ok := desired.DesiredMode.(*controlv1.ZoneState_Off); !ok {
		t.Fatalf("desired mode = %T, want explicit Off", desired.DesiredMode)
	}
	if fallback {
		t.Fatal("Off must not be reported as a static fallback")
	}
	if desired.ControllerId != "controller-1" || desired.ZoneId != "zone-1" {
		t.Fatalf("zone identity = %s/%s", desired.ControllerId, desired.ZoneId)
	}
}

func TestDesiredForZoneFallsBackWhenEffectIsNotAdvertised(t *testing.T) {
	config := internalpolicy.DefaultConfig()
	effective := &policyv1.EffectivePolicy{
		Preset:            policyv1.PresetKind_PRESET_KIND_COMBINED,
		Color:             &controlv1.Color{Red: 1, Green: 2, Blue: 3},
		BrightnessPercent: 25,
		EffectId:          "Breathing",
	}
	desired, fallback := desiredForZone("c", &controlv1.ZoneCapability{ZoneId: "z", SupportedEffects: []string{"Spectrum"}}, config, effective)
	static, ok := desired.DesiredMode.(*controlv1.ZoneState_StaticColor)
	if !ok {
		t.Fatalf("desired mode = %T, want static fallback", desired.DesiredMode)
	}
	if !fallback {
		t.Fatal("unsupported effect must be marked as static fallback")
	}
	if static.StaticColor.Color.Red != 1 || static.StaticColor.BrightnessPercent != 25 {
		t.Fatalf("fallback did not preserve configured effective color/brightness: %+v", static.StaticColor)
	}
}

func TestDesiredForZoneUsesConfiguredStaticFallbackForStaleEvidence(t *testing.T) {
	config := internalpolicy.DefaultConfig()
	effective := &policyv1.EffectivePolicy{Preset: policyv1.PresetKind_PRESET_KIND_HEALTH, StaticFallback: true}
	desired, fallback := desiredForZone("c", &controlv1.ZoneCapability{ZoneId: "z"}, config, effective)
	static, ok := desired.DesiredMode.(*controlv1.ZoneState_StaticColor)
	if !ok || !fallback {
		t.Fatalf("desired=%T fallback=%t; want explicit static fallback", desired.DesiredMode, fallback)
	}
	if static.StaticColor.Color.Red != config.StaleFallbackColor.Red || static.StaticColor.BrightnessPercent != config.StaleFallbackBrightnessPercent {
		t.Fatalf("fallback = %+v, want configured %+v at %d%%", static.StaticColor, config.StaleFallbackColor, config.StaleFallbackBrightnessPercent)
	}
}

func TestDesiredForZoneUsesAdvertisedNativeEffect(t *testing.T) {
	effective := &policyv1.EffectivePolicy{Preset: policyv1.PresetKind_PRESET_KIND_COMBINED, Color: &controlv1.Color{Green: 2}, BrightnessPercent: 12, EffectId: "Breathing"}
	desired, fallback := desiredForZone("c", &controlv1.ZoneCapability{ZoneId: "z", SupportedEffects: []string{"breathing"}}, internalpolicy.DefaultConfig(), effective)
	effect, ok := desired.DesiredMode.(*controlv1.ZoneState_Effect)
	if !ok || fallback {
		t.Fatalf("desired mode=%T fallback=%t; the advertised effect should remain native", desired.DesiredMode, fallback)
	}
	if effect.Effect.EffectId != "Breathing" || effect.Effect.BrightnessPercent != 12 {
		t.Fatalf("effect = %+v", effect.Effect)
	}
}

func TestZoneFallbackPreservesHealthAndEnforcesGlobalBrightnessCap(t *testing.T) {
	config := internalpolicy.DefaultConfig()
	config.BrightnessCapPercent = 4
	effective := &policyv1.EffectivePolicy{
		Preset:            policyv1.PresetKind_PRESET_KIND_COMBINED,
		Color:             &controlv1.Color{Red: 190},
		BrightnessPercent: 25,
		StaticFallback:    true,
		EffectId:          "Breathing",
	}
	desired, fallback := desiredForZone("c", &controlv1.ZoneCapability{ZoneId: "z", SupportedEffects: []string{"Breathing"}}, config, effective)
	state, ok := desired.DesiredMode.(*controlv1.ZoneState_StaticColor)
	if !ok || !fallback || state.StaticColor.Color.Red != 190 || state.StaticColor.BrightnessPercent != 4 {
		t.Fatalf("health static fallback = %+v, fallback %t", desired, fallback)
	}
	effective.Color = nil
	desired, _ = desiredForZone("c", &controlv1.ZoneCapability{ZoneId: "z"}, config, effective)
	if desired.GetStaticColor().BrightnessPercent != 4 {
		t.Fatalf("missing-color fallback bypassed cap: %+v", desired)
	}
}

func TestCachedPolicyStopsAtRequestExpiryAndSourceFreshnessBoundary(t *testing.T) {
	now := time.Now().UTC()
	for _, tt := range []struct {
		name string
		view *policyv1.EffectivePolicy
	}{
		{"manual expiry", &policyv1.EffectivePolicy{ManualPriority: true, RequestExpiresAt: timestamppb.New(now)}},
		{"source expiry", &policyv1.EffectivePolicy{Evidence: []*policyv1.EvidenceSource{
			{State: policyv1.EvidenceState_EVIDENCE_STATE_FRESH, SourceTimestamp: timestamppb.New(now.Add(-6 * time.Second))},
		}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if cachedPolicyFresh(tt.view, now.Add(-time.Second), now, 5) {
				t.Fatal("ten-second presentation cache outlived authoritative request/evidence deadline")
			}
		})
	}
}

type selectionControl struct {
	*internalcontrol.Service
	fail      string
	calls     []string
	expiries  []time.Time
	offActive bool
}

func (c *selectionControl) PolicyRevision(context.Context) (int64, error) { return 0, nil }

func (c *selectionControl) SelectPolicy(ctx context.Context, selection internalcontrol.PolicySelection) (bool, error) {
	if c.offActive && selection.BaseRevision == nil {
		return false, nil
	}
	c.calls = append(c.calls, "select")
	if c.fail == "select" {
		return false, errors.New("fixture selection failure")
	}
	if selection.Off {
		if _, _, err := c.SetManualOff(ctx, true, nil, selection.ExpiresAt); err != nil {
			return false, err
		}
	} else {
		if err := c.ClearManualOverride(ctx); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (c *selectionControl) SetManualOff(_ context.Context, all bool, ids []string, expires *timestamppb.Timestamp) (*controlv1.ManualOverride, []*controlv1.ZoneApplyResult, error) {
	c.calls = append(c.calls, "off")
	if expires != nil {
		c.expiries = append(c.expiries, expires.AsTime())
	}
	if c.fail == "off" {
		return nil, nil, errors.New("fixture Off failure")
	}
	c.offActive = true
	return &controlv1.ManualOverride{Active: true, AllEnrolled: all, ControllerIds: ids, ExpiresAt: expires}, nil, nil
}

func (c *selectionControl) ClearManualOverride(context.Context) error {
	c.calls = append(c.calls, "clear")
	if c.fail == "clear" {
		return errors.New("fixture clear failure")
	}
	c.offActive = false
	return nil
}

func (c *selectionControl) ListControllers(context.Context, ...bool) (*controlv1.BackendStatus, []*controlv1.Controller, error) {
	c.calls = append(c.calls, "list")
	return nil, nil, nil
}

func TestManualSelectionPartialFailureRetriesOriginalRequestBeforeApplying(t *testing.T) {
	now := time.Date(2026, 11, 11, 2, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name        string
		kind        policyv1.PresetKind
		fail        string
		saveFailure int
	}{
		{"initial save", policyv1.PresetKind_PRESET_KIND_OFF, "", 1},
		{"Off selection", policyv1.PresetKind_PRESET_KIND_OFF, "select", 0},
		{"Off fence", policyv1.PresetKind_PRESET_KIND_OFF, "off", 0},
		{"non-Off selection", policyv1.PresetKind_PRESET_KIND_STATIC, "select", 0},
		{"release Off", policyv1.PresetKind_PRESET_KIND_STATIC, "clear", 0},
		{"Off completion checkpoint", policyv1.PresetKind_PRESET_KIND_OFF, "", 2},
		{"non-Off completion checkpoint", policyv1.PresetKind_PRESET_KIND_STATIC, "", 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := &memoryPolicyStore{config: internalpolicy.DefaultConfig(), failSaveAt: tt.saveFailure}
			control := &selectionControl{fail: tt.fail, offActive: true}
			service := NewService(store, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: now}}, control)
			service.now = func() time.Time { return now }
			expiry := now.Add(time.Minute)
			effective, results, err := service.SetManual(context.Background(), tt.kind, expiry)
			require.Error(t, err)
			require.Nil(t, effective)
			require.Nil(t, results)
			require.NotContains(t, control.calls, "list", "partial commit cannot be reported or applied as complete")
			if tt.saveFailure == 1 {
				require.Empty(t, control.calls)
				require.False(t, store.state.ManualSelectionPending)
				return
			}
			require.True(t, store.state.ManualSelectionPending)
			require.Equal(t, expiry, store.state.ManualExpiry)
			require.Contains(t, err.Error(), "pending")
			control.fail, store.failSaveAt = "", 0
			effective, _, err = service.Evaluate(context.Background())
			require.NoError(t, err)
			require.False(t, store.state.ManualSelectionPending)
			require.True(t, effective.ManualPriority)
			require.Equal(t, expiry, effective.RequestExpiresAt.AsTime())
			require.Contains(t, control.calls, "list")
			for _, observed := range control.expiries {
				require.Equal(t, expiry, observed, "retries must never extend the operator request")
			}
		})
	}
}

func TestExpiredPendingSelectionAndUnrelatedClearKeepIndependentOff(t *testing.T) {
	now := time.Date(2026, 11, 11, 2, 0, 0, 0, time.UTC)
	store := &memoryPolicyStore{config: internalpolicy.DefaultConfig(), state: internalpolicy.State{
		ManualPreset:           policyv1.PresetKind_PRESET_KIND_STATIC,
		ManualExpiry:           now.Add(-time.Second),
		ManualSelectionPending: true,
	}}
	control := &selectionControl{offActive: true}
	service := NewService(store, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: now}}, control)
	service.now = func() time.Time { return now }
	_, _, err := service.Evaluate(context.Background())
	require.NoError(t, err)
	require.False(t, store.state.ManualSelectionPending)
	require.NotContains(t, control.calls, "clear")
	require.NotContains(t, control.calls, "select")
	_, _, _, err = service.ClearManual(context.Background())
	require.NoError(t, err)
	require.True(t, control.offActive)
	require.NotContains(t, control.calls, "clear")
}

func (c *selectionControl) Reconcile(context.Context) ([]*controlv1.ReconcileZoneResult, error) {
	c.calls = append(c.calls, "reconcile")
	return nil, nil
}

func TestReconcileTickCompletesPendingOffBeforeReplayingManualZones(t *testing.T) {
	now := time.Date(2026, 11, 11, 2, 0, 0, 0, time.UTC)
	store := &memoryPolicyStore{config: internalpolicy.DefaultConfig(), state: internalpolicy.State{
		ManualPreset:           policyv1.PresetKind_PRESET_KIND_OFF,
		ManualExpiry:           now.Add(time.Minute),
		ManualSelectionPending: true,
	}}
	control := &selectionControl{fail: "off"}
	service := NewService(store, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: now}}, control)
	service.now = func() time.Time { return now }
	service.reconcileTick(context.Background())
	require.NotContains(t, control.calls, "reconcile")
	require.NotContains(t, control.calls, "list")
	control.calls, control.fail = nil, ""
	service.reconcileTick(context.Background())
	require.Equal(t, []string{"select", "off", "reconcile", "list"}, control.calls)
	require.True(t, control.offActive)
}

func TestSelectedPolicyReleasesOffBeforeFreshEvaluation(t *testing.T) {
	now := time.Date(2026, 11, 11, 2, 0, 0, 0, time.UTC)
	store := &memoryPolicyStore{config: internalpolicy.DefaultConfig()}
	control := &selectionControl{offActive: true}
	service := NewService(store, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: now}}, control)
	service.now = func() time.Time { return now }
	effective, _, err := service.SetManual(context.Background(), policyv1.PresetKind_PRESET_KIND_STATIC, time.Time{})
	require.NoError(t, err)
	require.True(t, effective.ManualPriority)
	require.False(t, store.state.ManualSelectionPending)
	require.False(t, control.offActive)
}

func TestUnknownPendingSelectionPreservesActiveOff(t *testing.T) {
	now := time.Date(2026, 11, 11, 2, 0, 0, 0, time.UTC)
	store := &memoryPolicyStore{config: internalpolicy.DefaultConfig(), state: internalpolicy.State{
		ManualPreset:           policyv1.PresetKind_PRESET_KIND_STATIC,
		ManualRequestedAt:      now.Add(-time.Second),
		ManualSelectionPending: true,
	}}
	control := &selectionControl{offActive: true}
	service := NewService(store, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: now}}, control)
	service.now = func() time.Time { return now }
	_, _, err := service.Evaluate(context.Background())
	require.NoError(t, err)
	require.True(t, control.offActive)
	require.False(t, store.state.ManualSelectionPending)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_UNSPECIFIED, store.state.ManualPreset)
	require.NotContains(t, control.calls, "clear")
	require.NotContains(t, control.calls, "select")
}

// Inject a real direct Off immediately after the complete control transition,
// before the policy checkpoint/evaluation. The handler must not clear it later.
type offAfterSelectionControl struct {
	*internalcontrol.Service
	clock *scheduletest.FakeClock
}

func (c *offAfterSelectionControl) SelectPolicy(ctx context.Context, selection internalcontrol.PolicySelection) (bool, error) {
	selected, err := c.Service.SelectPolicy(ctx, selection)
	if err != nil {
		return selected, err
	}
	c.clock.Advance(time.Second)
	_, _, err = c.Service.SetManualOff(ctx, true, nil, nil)
	return selected, err
}

func TestNewerOffAfterControlSelectionSurvivesPolicyCheckpointAndRetry(t *testing.T) {
	for _, kind := range []policyv1.PresetKind{policyv1.PresetKind_PRESET_KIND_STATIC, policyv1.PresetKind_PRESET_KIND_OFF} {
		for _, checkpointFails := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/checkpointFails=%t", kind, checkpointFails), func(t *testing.T) {
				ctx := t.Context()
				clock := scheduletest.New(time.Date(2026, 11, 11, 4, 0, 0, 0, time.UTC))
				db := dbtest.NewSQLite(t)
				require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(internalcontrol.Schema)))
				require.NoError(t, internalcontrol.EnsureIdentitySource(ctx, db))
				realControl := internalcontrol.NewService(internalcontrol.NewSQLiteRepository(db, clock), nil, clock)
				control := &offAfterSelectionControl{Service: realControl, clock: clock}
				store := &memoryPolicyStore{config: internalpolicy.DefaultConfig()}
				if checkpointFails {
					store.failSaveAt = 2
				}
				service := NewService(store, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: clock.Now()}}, control)
				service.now = clock.Now
				requestedAt := clock.Now()
				_, _, err := service.SetManual(ctx, kind, requestedAt.Add(time.Minute))
				if checkpointFails {
					require.ErrorContains(t, err, "policy checkpoint pending")
					require.True(t, store.state.ManualSelectionPending)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, requestedAt, store.state.ManualRequestedAt)
				_, override, err := realControl.DesiredState(ctx)
				require.NoError(t, err)
				require.True(t, override.Active)
				require.Nil(t, override.ExpiresAt, "older timed selection cannot weaken newer indefinite Off")
				require.Equal(t, clock.Now(), override.CreatedAt.AsTime())
				// A fresh service retries the original request after checkpoint
				// failure; the newer Off supersedes either pending preset.
				service = NewService(store, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: clock.Now()}}, realControl)
				service.now = clock.Now
				_, _, err = service.Evaluate(ctx)
				require.NoError(t, err)
				require.False(t, store.state.ManualSelectionPending)
				if checkpointFails {
					require.Equal(t, policyv1.PresetKind_PRESET_KIND_UNSPECIFIED, store.state.ManualPreset)
				}
				_, override, err = realControl.DesiredState(ctx)
				require.NoError(t, err)
				require.True(t, override.Active)
				require.Nil(t, override.ExpiresAt)
			})
		}
	}
}

type failSelectionCheckpointStore struct{ internalpolicy.Store }

func (s failSelectionCheckpointStore) Save(ctx context.Context, config *policyv1.PolicyConfig, state internalpolicy.State) error {
	if state.ManualSelectionID != "" && !state.ManualSelectionPending {
		return errors.New("fixture policy completion checkpoint failure")
	}
	return s.Store.Save(ctx, config, state)
}

func TestCompletedSelectionRetryAfterDatabaseReopenPreservesNewManualZone(t *testing.T) {
	ctx := t.Context()
	clock := scheduletest.New(time.Date(2026, 11, 11, 4, 0, 0, 0, time.UTC))
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(internalcontrol.Schema), database.SchemaProviderFunc(internalpolicy.Schema)))
	require.NoError(t, internalcontrol.EnsureIdentitySource(ctx, db))
	repo := internalcontrol.NewSQLiteRepository(db, clock)
	require.NoError(t, repo.SyncControllers(ctx, []*controlv1.Controller{{ControllerId: "fixture", BackendId: "openrgb", IdentitySource: "openrgb_sdk_serial"}}, []internalcontrol.ZoneMapping{{ControllerID: "fixture", ZoneID: "zone", Fingerprint: "strip", Name: "strip", LEDCount: 1, Static: true}}))
	control := internalcontrol.NewService(repo, nil, clock)
	_, err := control.Enroll(ctx, "fixture", true)
	require.NoError(t, err)
	store := internalpolicy.NewSQLiteStore(db)
	_, _, err = internalpolicy.LoadOrInitialize(ctx, store)
	require.NoError(t, err)
	service := NewService(failSelectionCheckpointStore{Store: store}, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: clock.Now()}}, control)
	service.now = clock.Now
	_, _, err = service.SetManual(ctx, policyv1.PresetKind_PRESET_KIND_STATIC, time.Time{})
	require.ErrorContains(t, err, "completion checkpoint failure")
	_, pending, err := store.Load(ctx)
	require.NoError(t, err)
	require.True(t, pending.ManualSelectionPending)
	require.NotEmpty(t, pending.ManualSelectionID)
	require.NotNil(t, pending.ManualControlRevision)
	clock.Advance(time.Second)
	manual := &controlv1.ZoneState{ControllerId: "fixture", ZoneId: "zone", DesiredMode: &controlv1.ZoneState_StaticColor{StaticColor: &controlv1.StaticZoneState{Color: &controlv1.Color{Green: 255}, BrightnessPercent: 30}}}
	_, err = control.SetZoneState(ctx, manual)
	require.NoError(t, err)
	reopened := reopenPolicyDatabase(t, db)
	repo = internalcontrol.NewSQLiteRepository(reopened, clock)
	control = internalcontrol.NewService(repo, nil, clock)
	store = internalpolicy.NewSQLiteStore(reopened)
	service = NewService(store, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: clock.Now()}}, control)
	service.now = clock.Now
	_, _, err = service.Evaluate(ctx)
	require.NoError(t, err)
	_, completed, err := store.Load(ctx)
	require.NoError(t, err)
	require.False(t, completed.ManualSelectionPending)
	require.Equal(t, pending.ManualSelectionID, completed.ManualSelectionID)
	require.Equal(t, pending.ManualControlRevision, completed.ManualControlRevision)
	source, err := repo.DesiredSource(ctx, "fixture", "zone")
	require.NoError(t, err)
	require.Equal(t, "manual", source, "acknowledgement retry cannot adopt a newer operator's zone")
	rows, _, err := control.DesiredState(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.True(t, proto.Equal(manual, rows[0].Desired))
	var archived int
	require.NoError(t, reopened.QueryRowContext(ctx, `SELECT COUNT(*) FROM lighting_previous_zone_state`).Scan(&archived))
	require.Zero(t, archived, "retry must not archive the newer manual command")
}

type offBeforeSelectionControl struct {
	*internalcontrol.Service
	clock *scheduletest.FakeClock
	shift time.Duration
	fail  bool
}

func (c *offBeforeSelectionControl) SelectPolicy(ctx context.Context, selection internalcontrol.PolicySelection) (bool, error) {
	c.clock.Advance(c.shift)
	if _, _, err := c.Service.SetManualOff(ctx, true, nil, nil); err != nil {
		return false, err
	}
	if c.fail {
		return false, errors.New("fixture selection unavailable after later Off")
	}
	return c.Service.SelectPolicy(ctx, selection)
}

func TestLaterOffSurvivesSelectionWithEqualOrRolledBackClock(t *testing.T) {
	for _, kind := range []policyv1.PresetKind{policyv1.PresetKind_PRESET_KIND_STATIC, policyv1.PresetKind_PRESET_KIND_OFF} {
		for _, shift := range []time.Duration{0, -time.Second} {
			t.Run(fmt.Sprintf("%s/clockShift=%s", kind, shift), func(t *testing.T) {
				ctx := t.Context()
				clock := scheduletest.New(time.Date(2026, 11, 11, 4, 0, 0, 0, time.UTC))
				db := dbtest.NewSQLite(t)
				require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(internalcontrol.Schema)))
				control := internalcontrol.NewService(internalcontrol.NewSQLiteRepository(db, clock), nil, clock)
				store := &memoryPolicyStore{config: internalpolicy.DefaultConfig()}
				service := NewService(store, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: clock.Now()}}, &offBeforeSelectionControl{Service: control, clock: clock, shift: shift})
				service.now = clock.Now
				_, _, err := service.SetManual(ctx, kind, clock.Now().Add(time.Minute))
				require.NoError(t, err)
				_, override, err := control.DesiredState(ctx)
				require.NoError(t, err)
				require.True(t, override.Active)
				require.Nil(t, override.ExpiresAt, "an older timed request cannot weaken the later indefinite Off")
				require.Equal(t, clock.Now(), override.CreatedAt.AsTime())
				require.Equal(t, policyv1.PresetKind_PRESET_KIND_UNSPECIFIED, store.state.ManualPreset)
			})
		}
	}
}

func TestExplicitSelectionReleasesOlderOffWithEqualOrRolledBackClock(t *testing.T) {
	for _, shift := range []time.Duration{0, -time.Second} {
		t.Run(shift.String(), func(t *testing.T) {
			ctx := t.Context()
			clock := scheduletest.New(time.Date(2026, 11, 11, 4, 0, 0, 0, time.UTC))
			db := dbtest.NewSQLite(t)
			require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(internalcontrol.Schema)))
			control := internalcontrol.NewService(internalcontrol.NewSQLiteRepository(db, clock), nil, clock)
			_, _, err := control.SetManualOff(ctx, true, nil, nil)
			require.NoError(t, err)
			clock.Advance(shift)
			service := NewService(&memoryPolicyStore{config: internalpolicy.DefaultConfig()}, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: clock.Now()}}, control)
			service.now = clock.Now
			_, _, err = service.SetManual(ctx, policyv1.PresetKind_PRESET_KIND_STATIC, time.Time{})
			require.NoError(t, err)
			_, override, err := control.DesiredState(ctx)
			require.NoError(t, err)
			require.False(t, override.Active, "the later explicit selection must release the Off captured at preparation")
		})
	}
}

func reopenPolicyDatabase(t *testing.T, db *sql.DB) *sql.DB {
	t.Helper()
	ctx := t.Context()
	var seq int
	var name, path string
	require.NoError(t, db.QueryRowContext(ctx, `PRAGMA database_list`).Scan(&seq, &name, &path))
	require.NoError(t, db.Close())
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{TxLock: "immediate"})
	require.NoError(t, err)
	reopened, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	reopened.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = reopened.Close() })
	return reopened
}

func TestPendingSelectionRevisionSurvivesDatabaseReopenAndClockRollback(t *testing.T) {
	ctx := t.Context()
	clock := scheduletest.New(time.Date(2026, 11, 11, 4, 0, 0, 0, time.UTC))
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(internalcontrol.Schema), database.SchemaProviderFunc(internalpolicy.Schema)))
	control := internalcontrol.NewService(internalcontrol.NewSQLiteRepository(db, clock), nil, clock)
	store := internalpolicy.NewSQLiteStore(db)
	_, _, err := internalpolicy.LoadOrInitialize(ctx, store)
	require.NoError(t, err)
	service := NewService(store, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: clock.Now()}}, &offBeforeSelectionControl{Service: control, clock: clock, shift: -time.Second, fail: true})
	service.now = clock.Now
	_, _, err = service.SetManual(ctx, policyv1.PresetKind_PRESET_KIND_STATIC, time.Time{})
	require.ErrorContains(t, err, "selection unavailable")
	_, pending, err := store.Load(ctx)
	require.NoError(t, err)
	require.True(t, pending.ManualSelectionPending)
	require.NotNil(t, pending.ManualControlRevision)
	require.Zero(t, *pending.ManualControlRevision)
	reopened := reopenPolicyDatabase(t, db)
	control = internalcontrol.NewService(internalcontrol.NewSQLiteRepository(reopened, clock), nil, clock)
	store = internalpolicy.NewSQLiteStore(reopened)
	service = NewService(store, fixedPolicyInputs{inputs: internalpolicy.Inputs{Now: clock.Now()}}, control)
	service.now = clock.Now
	_, _, err = service.Evaluate(ctx)
	require.NoError(t, err)
	_, override, err := control.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, override.Active)
	_, state, err := store.Load(ctx)
	require.NoError(t, err)
	require.False(t, state.ManualSelectionPending)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_UNSPECIFIED, state.ManualPreset)
}
