package control_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"lighting-control/internal/control"
)

func enrolledPolicyFixture(t *testing.T) (*control.Service, *sql.DB, *fakeBackend, *scheduletest.FakeClock, string, string) {
	t.Helper()
	clock := scheduletest.New(time.Date(2026, 11, 11, 1, 0, 0, 0, time.UTC))
	backend := &fakeBackend{protocol: 6, controllers: []control.SDKController{{ID: 41, Serial: "policy-ownership", Zones: []control.SDKZone{{Name: "Strip", LEDCount: 1, Static: true}}}}}
	service, db := newService(t, backend, clock)
	_, controllers, err := service.ListControllers(t.Context())
	require.NoError(t, err)
	id, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
	_, err = service.Enroll(t.Context(), id, true)
	require.NoError(t, err)
	return service, db, backend, clock, id, zone
}

func TestAutomaticStateWaitsForFreshPolicyAcrossRestartAndReconnect(t *testing.T) {
	_, db, backend, clock, id, zone := enrolledPolicyFixture(t)
	ctx := t.Context()
	backend.controllers[0].LEDCount = 1
	backend.controllers[0].Zones[0].Modes = []control.SDKMode{{Name: "Static"}, {Name: "Breathing", SDKModeParameters: control.SDKModeParameters{Flags: 1 | 1<<4, BrightnessMax: 100}}}
	modes := &modeAwareBackend{fakeBackend: backend}
	service := control.NewService(control.NewSQLiteRepository(db, clock), modes, clock)
	old := &controlv1.ZoneState{ControllerId: id, ZoneId: zone, DesiredMode: &controlv1.ZoneState_Effect{Effect: &controlv1.EffectZoneState{EffectId: "Breathing", Color: &controlv1.Color{Red: 255}, BrightnessPercent: 20}}}
	_, err := service.ApplyPolicyState(ctx, old)
	require.NoError(t, err)
	require.Len(t, modes.modes, 1)
	require.Equal(t, "Breathing", modes.modes[0].mode.Name)

	clock.Advance(time.Hour)
	restarted := control.NewService(control.NewSQLiteRepository(db, clock), modes, clock)
	results, err := restarted.Reconcile(ctx)
	require.NoError(t, err)
	require.Empty(t, results)
	require.Len(t, modes.modes, 1, "reconnect must never replay the stale pulse before current source evaluation")
	require.Empty(t, backend.writes)

	staleFallback := staticState(id, zone, 0, 0, 0)
	_, err = restarted.ApplyPolicyState(ctx, staleFallback)
	require.NoError(t, err)
	require.Len(t, backend.writes, 1)
	require.Equal(t, []control.RGB{{}}, backend.writes[0].colors)
	states, _, err := restarted.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, proto.Equal(staleFallback, states[0].Desired))
}

func TestAutomaticPendingStatePersistsBeforeWriteAndRetriesOnlyFreshEvaluation(t *testing.T) {
	_, db, backend, clock, id, zone := enrolledPolicyFixture(t)
	ctx := t.Context()
	stopped := control.NewService(control.NewSQLiteRepository(db, clock), nil, clock)
	requested := staticState(id, zone, 30, 60, 90)
	result, err := stopped.ApplyPolicyState(ctx, requested)
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_DEFERRED, result.Operation)
	states, _, err := stopped.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, proto.Equal(requested, states[0].Desired))
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_DEFERRED, states[0].LastApply.Operation)

	reconnected := control.NewService(control.NewSQLiteRepository(db, clock), backend, clock)
	_, err = reconnected.Reconcile(ctx)
	require.NoError(t, err)
	require.Empty(t, backend.writes)
	current := staticState(id, zone, 5, 10, 15)
	result, err = reconnected.ApplyPolicyState(ctx, current)
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED, result.Operation)
	require.Equal(t, []control.RGB{{R: 2, G: 4, B: 5}}, backend.writes[0].colors)
}

func TestExplicitPolicySelectionArchivesManualBaseAndReleasesItsPriority(t *testing.T) {
	service, db, backend, _, id, zone := enrolledPolicyFixture(t)
	ctx := t.Context()
	manual := staticState(id, zone, 0, 255, 0)
	_, err := service.SetZoneState(ctx, manual)
	require.NoError(t, err)
	policy := staticState(id, zone, 255, 0, 0)
	_, err = service.ApplyPolicyState(ctx, policy)
	require.NoError(t, err)
	require.Equal(t, uint8(89), backend.writes[1].colors[0].G)

	selected, selectionErr := service.SelectPolicy(ctx, control.PolicySelection{ID: rand.Text()})
	require.NoError(t, selectionErr)
	require.True(t, selected)
	var archived []byte
	var source string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT state,source FROM lighting_previous_zone_state WHERE controller_id=? AND zone_id=?`, id, zone).Scan(&archived, &source))
	wantBytes, err := proto.Marshal(manual)
	require.NoError(t, err)
	require.Equal(t, wantBytes, archived)
	require.Equal(t, "manual", source)
	_, err = service.ApplyPolicyState(ctx, policy)
	require.NoError(t, err)
	require.Equal(t, uint8(89), backend.writes[2].colors[0].R)
	require.Zero(t, backend.writes[2].colors[0].G)
	// Selecting another global preset must retain the archived manual bytes.
	selected, selectionErr = service.SelectPolicy(ctx, control.PolicySelection{ID: rand.Text()})
	require.NoError(t, selectionErr)
	require.True(t, selected)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT state FROM lighting_previous_zone_state WHERE controller_id=? AND zone_id=?`, id, zone).Scan(&archived))
	require.Equal(t, wantBytes, archived)
}

func TestLegacyDesiredBytesRequireExplicitAdoptionAndPreserveOlderManualOff(t *testing.T) {
	service, db, backend, clock, id, zone := enrolledPolicyFixture(t)
	ctx := t.Context()
	legacy := staticState(id, zone, 100, 0, 0)
	original, err := proto.Marshal(legacy)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO lighting_desired_state(controller_id,zone_id,state,updated_at) VALUES(?,?,?,?)`, id, zone, original, clock.Now().Format(time.RFC3339Nano))
	require.NoError(t, err)
	_, err = service.Reconcile(ctx)
	require.NoError(t, err)
	require.Empty(t, backend.writes)
	policy := staticState(id, zone, 0, 0, 255)
	result, err := service.ApplyPolicyState(ctx, policy)
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_DEFERRED, result.Operation)
	require.Contains(t, result.Reason, "Review")
	states, _, err := service.DesiredState(ctx)
	require.NoError(t, err)
	require.Contains(t, states[0].LastApply.Reason, "Save & apply")

	// A preexisting Off record retains its authority independently of missing
	// zone ownership. Neither Off nor its expiry adopts the underlying bytes.
	repo := control.NewSQLiteRepository(db, clock)
	require.NoError(t, repo.SetManualOverride(ctx, &controlv1.ManualOverride{Active: true, AllEnrolled: true, CreatedAt: timestamppb.New(clock.Now()), ExpiresAt: timestamppb.New(clock.Now().Add(time.Minute))}))
	_, err = service.Reconcile(ctx)
	require.NoError(t, err)
	require.Len(t, backend.writes, 1)
	require.Equal(t, []control.RGB{{}}, backend.writes[0].colors)
	clock.Advance(time.Minute)
	_, err = service.Reconcile(ctx)
	require.NoError(t, err)
	require.Len(t, backend.writes, 1, "expiry cannot manufacture manual provenance")
	var retained []byte
	require.NoError(t, db.QueryRowContext(ctx, `SELECT state FROM lighting_desired_state WHERE controller_id=? AND zone_id=?`, id, zone).Scan(&retained))
	require.Equal(t, original, retained)

	selected, selectionErr := service.SelectPolicy(ctx, control.PolicySelection{ID: rand.Text(), RequestedAt: clock.Now()})
	require.NoError(t, selectionErr)
	require.True(t, selected)
	_, err = service.ApplyPolicyState(ctx, policy)
	require.NoError(t, err)
	require.Len(t, backend.writes, 2)
	require.Equal(t, uint8(89), backend.writes[1].colors[0].B)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT state FROM lighting_previous_zone_state WHERE controller_id=? AND zone_id=?`, id, zone).Scan(&retained))
	require.Equal(t, original, retained)
}

type policyPersistenceFailure struct{ control.Repository }

func (policyPersistenceFailure) SavePolicyDesired(context.Context, *controlv1.ZoneState, *controlv1.ZoneApplyResult) error {
	return errors.New("fixture pending state persistence failure")
}

func TestPolicyWriteRequiresDurablePendingIntent(t *testing.T) {
	_, db, backend, clock, id, zone := enrolledPolicyFixture(t)
	repository := policyPersistenceFailure{Repository: control.NewSQLiteRepository(db, clock)}
	service := control.NewService(repository, backend, clock)
	_, err := service.ApplyPolicyState(t.Context(), staticState(id, zone, 1, 2, 3))
	require.ErrorContains(t, err, "pending state persistence failure")
	require.Empty(t, backend.writes)
}

func TestPolicySelectionChecksOffPrecedenceBeforeAdoptingManualOwnership(t *testing.T) {
	for _, newer := range []bool{false, true} {
		t.Run(map[bool]string{false: "older Off is released", true: "newer Off preserves manual owner"}[newer], func(t *testing.T) {
			service, db, _, clock, id, zone := enrolledPolicyFixture(t)
			ctx := t.Context()
			_, err := service.SetZoneState(ctx, staticState(id, zone, 1, 2, 3))
			require.NoError(t, err)
			requestedAt := clock.Now()
			revision, err := service.PolicyRevision(ctx)
			require.NoError(t, err)
			if newer {
				clock.Advance(time.Second)
			} else {
				requestedAt = requestedAt.Add(time.Second)
			}
			_, _, err = service.SetManualOff(ctx, true, nil, nil)
			require.NoError(t, err)
			if !newer {
				revision, err = service.PolicyRevision(ctx)
				require.NoError(t, err)
			}
			// Backend availability has no bearing on the durable transition.
			service = control.NewService(control.NewSQLiteRepository(db, clock), nil, clock)
			selected, err := service.SelectPolicy(ctx, control.PolicySelection{ID: rand.Text(), BaseRevision: &revision, RequestedAt: requestedAt})
			require.NoError(t, err)
			require.Equal(t, !newer, selected)
			_, override, err := service.DesiredState(ctx)
			require.NoError(t, err)
			require.Equal(t, newer, override.Active)
			source, err := control.NewSQLiteRepository(db, clock).DesiredSource(ctx, id, zone)
			require.NoError(t, err)
			require.Equal(t, map[bool]string{false: "policy", true: "manual"}[newer], source)
			var count int
			require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lighting_previous_zone_state`).Scan(&count))
			require.Equal(t, map[bool]int{false: 1, true: 0}[newer], count)
		})
	}
}

type interleavedSelectionRepository struct {
	control.Repository
	selected               chan struct{}
	offStarted             chan struct{}
	offCompleted           chan struct{}
	clearedWhileOffWaiting bool
}

func (r *interleavedSelectionRepository) SelectPolicy(ctx context.Context, selection control.PolicySelection, override *controlv1.ManualOverride) (control.SelectionResult, error) {
	committed, err := r.Repository.SelectPolicy(ctx, selection, override)
	if err != nil {
		return control.SelectionSuperseded, err
	}
	close(r.selected)
	<-r.offStarted
	select {
	case <-r.offCompleted:
		return control.SelectionSuperseded, errors.New("new manual Off completed inside policy selection")
	default:
		r.clearedWhileOffWaiting = true
	}
	return committed, nil
}

func TestDirectOffCannotInterleaveBetweenPolicyAdoptionAndRelease(t *testing.T) {
	_, db, _, clock, _, _ := enrolledPolicyFixture(t)
	ctx := t.Context()
	repo := &interleavedSelectionRepository{Repository: control.NewSQLiteRepository(db, clock), selected: make(chan struct{}), offStarted: make(chan struct{}), offCompleted: make(chan struct{})}
	service := control.NewService(repo, nil, clock)
	offErr := make(chan error, 1)
	go func() {
		<-repo.selected
		close(repo.offStarted)
		_, _, err := service.SetManualOff(ctx, true, nil, nil)
		close(repo.offCompleted)
		offErr <- err
	}()
	selected, err := service.SelectPolicy(ctx, control.PolicySelection{ID: rand.Text(), RequestedAt: clock.Now()})
	require.NoError(t, err)
	require.True(t, selected)
	require.NoError(t, <-offErr)
	require.True(t, repo.clearedWhileOffWaiting)
	_, override, err := service.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, override.Active, "the direct Off ordered after selection must remain durable")
}

func TestPendingTimedOffCannotReplaceNewerScopedOff(t *testing.T) {
	service, db, _, clock, id, zone := enrolledPolicyFixture(t)
	ctx := t.Context()
	_, err := service.SetZoneState(ctx, staticState(id, zone, 1, 2, 3))
	require.NoError(t, err)
	requestedAt := clock.Now()
	revision, err := service.PolicyRevision(ctx)
	require.NoError(t, err)
	clock.Advance(-time.Second)
	newer, _, err := service.SetManualOff(ctx, false, []string{id}, nil)
	require.NoError(t, err)
	selected, err := service.SelectPolicy(ctx, control.PolicySelection{ID: rand.Text(), BaseRevision: &revision, RequestedAt: requestedAt, Off: true, ExpiresAt: timestamppb.New(requestedAt.Add(time.Minute))})
	require.NoError(t, err)
	require.False(t, selected)
	_, override, err := service.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, proto.Equal(newer, override), "superseded selection must preserve scope, creation time and expiry")
	source, err := control.NewSQLiteRepository(db, clock).DesiredSource(ctx, id, zone)
	require.NoError(t, err)
	require.Equal(t, "manual", source)
}

func TestOffSelectionRetryRetainsOriginalFenceTimestampAndExpiry(t *testing.T) {
	_, db, _, clock, _, _ := enrolledPolicyFixture(t)
	ctx := t.Context()
	service := control.NewService(control.NewSQLiteRepository(db, clock), nil, clock)
	requestedAt := clock.Now()
	expires := timestamppb.New(requestedAt.Add(time.Minute))
	selection := control.PolicySelection{ID: rand.Text(), RequestedAt: requestedAt, Off: true, ExpiresAt: expires}
	selected, err := service.SelectPolicy(ctx, selection)
	require.NoError(t, err)
	require.True(t, selected)
	clock.Advance(time.Second)
	selected, err = service.SelectPolicy(ctx, selection)
	require.NoError(t, err)
	require.True(t, selected)
	_, override, err := service.DesiredState(ctx)
	require.NoError(t, err)
	require.Equal(t, requestedAt, override.CreatedAt.AsTime())
	require.True(t, proto.Equal(expires, override.ExpiresAt))
}

func TestSelectionReceiptFailureRollsBackOwnershipArchiveAndOff(t *testing.T) {
	service, db, _, clock, id, zone := enrolledPolicyFixture(t)
	ctx := t.Context()
	_, err := service.SetZoneState(ctx, staticState(id, zone, 1, 2, 3))
	require.NoError(t, err)
	originalOff, _, err := service.SetManualOff(ctx, true, nil, nil)
	require.NoError(t, err)
	clock.Advance(time.Second)
	revision, err := service.PolicyRevision(ctx)
	require.NoError(t, err)
	selection := control.PolicySelection{ID: rand.Text(), BaseRevision: &revision, RequestedAt: clock.Now()}
	_, err = db.ExecContext(ctx, `CREATE TRIGGER reject_selection_receipt BEFORE INSERT ON lighting_policy_selection BEGIN SELECT RAISE(ABORT,'fixture receipt failure'); END`)
	require.NoError(t, err)
	selected, err := service.SelectPolicy(ctx, selection)
	require.ErrorContains(t, err, "fixture receipt failure")
	require.False(t, selected)
	_, override, err := service.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, proto.Equal(originalOff, override))
	repo := control.NewSQLiteRepository(db, clock)
	source, err := repo.DesiredSource(ctx, id, zone)
	require.NoError(t, err)
	require.Equal(t, "manual", source)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lighting_previous_zone_state`).Scan(&count))
	require.Zero(t, count)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lighting_policy_selection`).Scan(&count))
	require.Zero(t, count)
	afterFailure, err := service.PolicyRevision(ctx)
	require.NoError(t, err)
	require.Equal(t, revision, afterFailure, "receipt failure must roll back causal order too")
	_, err = db.ExecContext(ctx, `DROP TRIGGER reject_selection_receipt`)
	require.NoError(t, err)
	selected, err = service.SelectPolicy(ctx, selection)
	require.NoError(t, err)
	require.True(t, selected)
	source, err = repo.DesiredSource(ctx, id, zone)
	require.NoError(t, err)
	require.Equal(t, "policy", source)
	_, override, err = service.DesiredState(ctx)
	require.NoError(t, err)
	require.False(t, override.Active)
}

func TestExpiredSelectionDoesNotAdoptOwnershipOrReleaseOff(t *testing.T) {
	service, db, _, clock, id, zone := enrolledPolicyFixture(t)
	ctx := t.Context()
	_, err := service.SetZoneState(ctx, staticState(id, zone, 1, 2, 3))
	require.NoError(t, err)
	originalOff, _, err := service.SetManualOff(ctx, true, nil, nil)
	require.NoError(t, err)
	selection := control.PolicySelection{ID: rand.Text(), RequestedAt: clock.Now(), ExpiresAt: timestamppb.New(clock.Now().Add(time.Second))}
	clock.Advance(2 * time.Second)
	selected, err := service.SelectPolicy(ctx, selection)
	require.NoError(t, err)
	require.False(t, selected)
	_, override, err := service.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, proto.Equal(originalOff, override))
	source, err := control.NewSQLiteRepository(db, clock).DesiredSource(ctx, id, zone)
	require.NoError(t, err)
	require.Equal(t, "manual", source)
}

func TestIdenticalSameClockOffCommandsAdvanceCausalRevision(t *testing.T) {
	service, _, _, clock, _, _ := enrolledPolicyFixture(t)
	ctx := t.Context()
	first, _, err := service.SetManualOff(ctx, true, nil, nil)
	require.NoError(t, err)
	revision, err := service.PolicyRevision(ctx)
	require.NoError(t, err)
	selection := control.PolicySelection{ID: rand.Text(), BaseRevision: &revision, RequestedAt: clock.Now()}
	second, _, err := service.SetManualOff(ctx, true, nil, nil)
	require.NoError(t, err)
	require.True(t, proto.Equal(first, second), "identical timestamp and payload cannot stand in for causal ordering")
	after, err := service.PolicyRevision(ctx)
	require.NoError(t, err)
	require.Greater(t, after, revision)
	selected, err := service.SelectPolicy(ctx, selection)
	require.NoError(t, err)
	require.False(t, selected)
	_, override, err := service.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, proto.Equal(second, override))
}

func TestUnknownLegacyPendingPreservesOffUntilNewExplicitSelection(t *testing.T) {
	service, db, _, clock, _, _ := enrolledPolicyFixture(t)
	ctx := t.Context()
	_, _, err := service.SetManualOff(ctx, true, nil, nil)
	require.NoError(t, err)
	// Simulate a persisted Off that predates causal metadata.
	_, err = db.ExecContext(ctx, `DELETE FROM lighting_override_order`)
	require.NoError(t, err)
	selected, err := service.SelectPolicy(ctx, control.PolicySelection{ID: rand.Text(), RequestedAt: clock.Now()})
	require.NoError(t, err)
	require.False(t, selected)
	_, override, err := service.DesiredState(ctx)
	require.NoError(t, err)
	require.True(t, override.Active)
	revision, err := service.PolicyRevision(ctx)
	require.NoError(t, err)
	require.Zero(t, revision)
	selected, err = service.SelectPolicy(ctx, control.PolicySelection{ID: rand.Text(), BaseRevision: &revision, RequestedAt: clock.Now()})
	require.NoError(t, err)
	require.True(t, selected)
	_, override, err = service.DesiredState(ctx)
	require.NoError(t, err)
	require.False(t, override.Active)
}
