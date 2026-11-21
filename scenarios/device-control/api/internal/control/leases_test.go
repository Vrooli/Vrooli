package control

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
)

func TestObservationOnlyFlowDoesNotRestoreUnchangedDeviceState(t *testing.T) {
	adapter := &flowAuthStrategy{Strategy: fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)}
	svc := New(strategyregistry.New(adapter))
	result, err := svc.Run(context.Background(), Flow{RequireUnlocked: true, Steps: []Step{{ID: "frame", Kind: "observe"}}}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Empty(t, adapter.restoredStates, "read-only observation must not invoke a device writer")
	require.Empty(t, result.Restoration)
}

func TestLeaseAndAuditSurviveServiceReconstruction(t *testing.T) {
	svc, db := testService(t)
	session, err := svc.Acquire("fake", "operator", 0)
	require.NoError(t, err)
	require.NotEmpty(t, session.LeaseToken)
	require.NoError(t, func() error { _, err := svc.Kill(session.ID, "test"); return err }())

	reloaded, err := NewWithDB(strategyregistry.New(), db)
	require.NoError(t, err)
	sessions := reloaded.ListSessions()
	require.Len(t, sessions, 1)
	require.Equal(t, "killed", sessions[0].State)
}

func TestHeldLeaseCanBeReleasedAfterServiceReconstruction(t *testing.T) {
	svc, db := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)

	reloaded, err := NewWithDB(strategyregistry.New(), db)
	require.NoError(t, err)
	released, err := reloaded.Release(session.ID)
	require.NoError(t, err)
	require.Equal(t, "released", released.State)
}

func TestConcurrentControllersOnlyCurrentLeaseHolderActs(t *testing.T) { // AUTH-03
	svc, _ := testService(t)
	first, err := svc.Acquire("fake", "actor-a", time.Minute)
	require.NoError(t, err)
	second, err := svc.Acquire("fake", "actor-b", time.Minute)
	require.Error(t, err, "a second controller cannot acquire an already-held device")
	require.Empty(t, second.LeaseToken)
	_, err = svc.ActuateDevice(context.Background(), "fake", "actor-b", "forged-token", DirectActuation{Key: "DPAD_DOWN"})
	require.Error(t, err)
	require.Empty(t, svc.Audit())
	_, err = svc.ActuateDevice(context.Background(), "fake", "actor-a", first.LeaseToken, DirectActuation{Key: "DPAD_DOWN"})
	require.NoError(t, err)
	require.Len(t, svc.Audit(), 1)
}

// REM-08: Bridge-owned peripheral revocation is checked at the input boundary
// and kills only the attached device's lease; unrelated local sessions remain
// independent because the check is scoped by the device's host relation.
func TestRevokedAttachedDeviceCannotActuateHeldLease(t *testing.T) {
	svc, _ := testService(t)
	svc.attached = staticAttachedReader{devices: []AttachedDevice{{ID: "fake", HostNodeID: "bridge-host", TrustState: "revoked", Reachability: "reachable"}}}
	observedAt := time.Now().UTC()
	svc.devices.Upsert(devicedomain.Record{ID: "fake", Kind: "physical", HostNodeID: "bridge-host", StrategyID: "fake", Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, ObservedAt: observedAt, Capabilities: []strategy.Capability{{Name: string(strategy.CapInput), Status: strategy.StatusAvailable}}, Transports: []strategy.DeviceTransport{{Name: "usb", StrategyID: "fake", Health: strategy.StatusAvailable, ObservedAt: observedAt, Capabilities: map[string]strategy.Capability{strategy.CapInput: {Name: strategy.CapInput, Status: strategy.StatusAvailable}}}}})
	lease, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	_, err = svc.ActuateDevice(context.Background(), "fake", "operator", lease.LeaseToken, DirectActuation{Key: "DPAD_DOWN"})
	require.ErrorContains(t, err, "no longer authorized")
	require.Empty(t, svc.Audit(), "revoked input must not reach the transport or create an actuation receipt")
}

func TestLocalDesktopLeaseDoesNotRequireBridgeAttachment(t *testing.T) {
	svc, _ := testService(t)
	svc.attached = staticAttachedReader{}
	svc.devices.Upsert(devicedomain.Record{ID: "desktop", Kind: "desktop", HostNodeID: "local", StrategyID: "fake", Transport: "local", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapScreenshot, Status: strategy.StatusAvailable}}})
	lease, err := svc.Acquire("desktop", "operator", time.Minute)
	require.NoError(t, err)
	result, err := svc.RunWithLease(context.Background(), Flow{ID: "desktop-capture", Steps: []Step{{ID: "observe", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}}}}, "desktop", "operator", lease.LeaseToken)
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Len(t, result.Evidence, 1)
}

func TestLeaseRejectsOverlappingRunsAndCancelsOriginal(t *testing.T) {
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	finished := make(chan RunResult, 1)
	go func() {
		result, _ := svc.RunWithLease(context.Background(), Flow{Steps: []Step{{ID: "wait", Kind: "wait", TimeoutMS: 5000, Arguments: map[string]any{"settle_ms": float64(5000)}}}}, "fake", "operator", session.LeaseToken)
		finished <- result
	}()
	require.Eventually(t, func() bool {
		svc.sessionMu.Lock()
		defer svc.sessionMu.Unlock()
		return svc.activeCancels[session.ID] != nil
	}, time.Second, 5*time.Millisecond)
	_, err = svc.RunWithLease(context.Background(), Flow{Steps: []Step{{ID: "wait", Kind: "wait"}}}, "fake", "operator", session.LeaseToken)
	require.ErrorContains(t, err, "already has an active run")
	_, err = svc.Release(session.ID)
	require.NoError(t, err)
	select {
	case result := <-finished:
		require.Equal(t, "cancelled", result.Disposition)
	case <-time.After(time.Second):
		t.Fatal("original run was not cancelled")
	}
}

func TestReleasedLeaseCannotRegisterRun(t *testing.T) {
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	_, err = svc.Release(session.ID)
	require.NoError(t, err)
	_, err = svc.execute(context.Background(), Flow{Steps: []Step{{ID: "wait", Kind: "wait"}}}, "fake", "operator", session, false, false)
	require.ErrorContains(t, err, "lease is no longer held")
}

func TestDeviceDisconnectReleasesLeaseAndRetainsPriorEvidence(t *testing.T) { // [REQ:DVC-P0-004]
	svc, _ := testService(t)
	fake, _ := svc.registry.Get("fake")
	fake.(*fakes.Strategy).ActuateErr = &strategy.AvailabilityError{Reason: "adb transport disappeared", NextAction: "Reconnect the device over USB."}
	result, err := svc.Run(context.Background(), Flow{Steps: []Step{
		{ID: "before", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}},
		{ID: "actuate", Kind: "key", Target: "ENTER", RequiredCapabilities: []string{strategy.CapInput}},
	}}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "device_disconnected", result.Disposition)
	require.True(t, result.Incomplete)
	require.Equal(t, "actuate", result.DisconnectStep)
	require.Len(t, result.Evidence, 1)
	require.Equal(t, "transport", result.Chapters[1].FailureClass)
	require.Equal(t, 1, result.Chapters[1].Attempts)
	require.Empty(t, svc.ListLiveSessions())
}

func TestListLiveSessionsExcludesFinishedLeases(t *testing.T) {
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	require.Len(t, svc.ListLiveSessions(), 1)
	_, err = svc.Release(session.ID)
	require.NoError(t, err)
	require.Empty(t, svc.ListLiveSessions())
	require.Len(t, svc.ListSessions(), 1)
}

func TestKillStopsAnInFlightFlow(t *testing.T) { // [REQ:DVC-P0-009]
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	finished := make(chan RunResult, 1)
	go func() {
		result, _ := svc.RunWithLease(context.Background(), Flow{Steps: []Step{{ID: "wait", Kind: "wait", TimeoutMS: 5000, Arguments: map[string]any{"settle_ms": float64(5000)}}}}, "fake", "operator", session.LeaseToken)
		finished <- result
	}()
	time.Sleep(25 * time.Millisecond)
	_, err = svc.Kill(session.ID, "operator requested kill")
	require.NoError(t, err)
	result := <-finished
	require.Equal(t, "cancelled", result.Disposition)
	require.NotEmpty(t, result.Chapters)
}

func TestObservationSessionsCoexistWithoutInputAuthority(t *testing.T) {
	svc, db := testService(t)
	ctx := context.Background()
	observe, err := svc.AcquireObservationContext(ctx, "fake", "reader", time.Minute)
	require.NoError(t, err)
	control, err := svc.AcquireContext(ctx, "fake", "controller", time.Minute)
	require.NoError(t, err)
	second, err := svc.AcquireObservationContext(ctx, "fake", "reader-2", time.Minute)
	require.NoError(t, err)
	_, err = svc.AcquireContext(ctx, "fake", "competitor", time.Minute)
	require.Error(t, err)
	require.NoError(t, svc.ValidateObservationLease(ctx, "fake", observe.LeaseToken))
	require.NoError(t, svc.ValidateObservationLease(ctx, "fake", control.LeaseToken))
	require.Error(t, svc.ValidateLease(ctx, "fake", observe.LeaseToken))
	_, err = svc.ActuateDevice(ctx, "fake", "reader", observe.LeaseToken, DirectActuation{Key: "DPAD_DOWN"})
	require.ErrorContains(t, err, "observation session")
	require.Empty(t, svc.Audit())
	require.NoError(t, svc.ValidateLease(ctx, "fake", control.LeaseToken))
	require.Error(t, svc.ValidateObservationLease(ctx, "wrong", observe.LeaseToken))
	rebuilt, err := NewWithDB(svc.registry, db)
	require.NoError(t, err)
	require.Error(t, rebuilt.ValidateLease(ctx, "fake", observe.LeaseToken))
	require.NoError(t, rebuilt.ValidateObservationLease(ctx, "fake", observe.LeaseToken))
	listed := rebuilt.ListLiveSessionsContext(ctx)
	require.Len(t, listed, 3)
	observationCount := 0
	for _, v := range listed {
		if v.ObservationOnly {
			observationCount++
		}
	}
	require.Equal(t, 2, observationCount)
	_, err = svc.KillContext(ctx, observe.ID, "revoke observation")
	require.NoError(t, err)
	require.Error(t, svc.ValidateObservationLease(ctx, "fake", observe.LeaseToken))
	require.NoError(t, svc.ValidateLease(ctx, "fake", control.LeaseToken))
	require.NoError(t, svc.ValidateObservationLease(ctx, "fake", second.LeaseToken))
	_, err = svc.ReleaseContext(ctx, control.ID)
	require.NoError(t, err)
	require.NoError(t, svc.ValidateObservationLease(ctx, "fake", second.LeaseToken))
	_, err = svc.AcquireContext(ctx, "fake", "successor", time.Minute)
	require.NoError(t, err)
}

func TestObservationSessionSchemaPreservesLegacyControl(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "legacy.sqlite"))
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE device_control_sessions (id TEXT PRIMARY KEY,device_id TEXT NOT NULL,actor TEXT NOT NULL,state TEXT NOT NULL,lease_token TEXT NOT NULL DEFAULT '',kill_reason TEXT NOT NULL DEFAULT '',expires_at TEXT NOT NULL,created_at TEXT NOT NULL)`)
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = db.Exec(`INSERT INTO device_control_sessions VALUES ('legacy','fake','actor','held','token','',?,?)`, now.Add(time.Minute).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	require.NoError(t, err)
	svc, err := NewWithDB(strategyregistry.New(fakes.New("fake", strategy.StatusAvailable, strategy.CapInput)), db)
	require.NoError(t, err)
	require.NoError(t, svc.ValidateLease(context.Background(), "fake", "token"))
	observation, err := svc.AcquireObservationContext(context.Background(), "fake", "reader", time.Minute)
	require.NoError(t, err)
	require.True(t, observation.ObservationOnly)
	require.Error(t, svc.ValidateLease(context.Background(), "fake", observation.LeaseToken))
}

func TestObservationSessionsAreBounded(t *testing.T) {
	svc, _ := testService(t)
	ctx := context.Background()
	for _, ttl := range []time.Duration{0, -time.Second, 11 * time.Minute} {
		_, err := svc.AcquireObservationContext(ctx, "fake", "reader", ttl)
		require.Error(t, err)
	}
	var first Session
	for i := 0; i < 64; i++ {
		v, err := svc.AcquireObservationContext(ctx, "fake", "reader", time.Minute)
		require.NoError(t, err)
		if i == 0 {
			first = v
		}
	}
	_, err := svc.AcquireObservationContext(ctx, "fake", "reader", time.Minute)
	require.Error(t, err)
	_, err = svc.AcquireContext(ctx, "fake", "controller", time.Minute)
	require.NoError(t, err)
	_, err = svc.ReleaseContext(ctx, first.ID)
	require.NoError(t, err)
	_, err = svc.AcquireObservationContext(ctx, "fake", "reader", time.Minute)
	require.NoError(t, err)
}
