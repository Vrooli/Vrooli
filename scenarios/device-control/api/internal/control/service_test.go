package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	authdomain "device-control/internal/auth"
	devicedomain "device-control/internal/devices"
	internalflows "device-control/internal/flows"
	identitydomain "device-control/internal/identity"
	"device-control/strategy"
	"device-control/strategy/androidtvremote"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"

	"github.com/stretchr/testify/require"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	_ "modernc.org/sqlite"
)

func testService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:control-test-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	fake := fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)
	svc, err := NewWithDB(strategyregistry.New(fake), db)
	require.NoError(t, err)
	return svc, db
}

type flowAuthResolver struct{ value string }

type pairingDiscoveryFixture struct{}

func (pairingDiscoveryFixture) DiscoverMDNS(context.Context) ([]androidtvremote.Device, error) {
	return []androidtvremote.Device{{
		Serial:      "bt-tv-1",
		IdentityKey: "bt-tv-1",
		Endpoint:    "tv.local:6466",
	}}, nil
}

type pairingClientFixture struct{ pin string }

func (f *pairingClientFixture) Pair(_ context.Context, _ androidtvremote.Device, pin string) ([]byte, error) {
	f.pin = pin
	return []byte("fixture-certificate"), nil
}

type interactivePairingSessionFixture struct {
	pin    string
	closed bool
}

func (f *interactivePairingSessionFixture) Complete(_ context.Context, pin string) ([]byte, error) {
	f.pin = pin
	return []byte("fixture-certificate"), nil
}

func (f *interactivePairingSessionFixture) Close() error {
	f.closed = true
	return nil
}

type interactivePairingClientFixture struct {
	session *interactivePairingSessionFixture
}

func (f *interactivePairingClientFixture) Pair(context.Context, androidtvremote.Device, string) ([]byte, error) {
	return nil, nil
}

func (f *interactivePairingClientFixture) Begin(context.Context, androidtvremote.Device) (androidtvremote.PairingSession, error) {
	f.session = &interactivePairingSessionFixture{}
	return f.session, nil
}

type leakyPairingClientFixture struct{}

func (leakyPairingClientFixture) Pair(_ context.Context, _ androidtvremote.Device, pin string) ([]byte, error) {
	return nil, errors.New("remote rejected PIN " + pin)
}

type pairingStoreFixture struct {
	serial      string
	certificate []byte
}

func (f *pairingStoreFixture) SavePairingCertificate(_ context.Context, serial string, certificate []byte) error {
	f.serial = serial
	f.certificate = append([]byte(nil), certificate...)
	return nil
}

func (f *pairingStoreFixture) LoadPairingCertificate(context.Context, string) ([]byte, error) {
	return append([]byte(nil), f.certificate...), nil
}

func (r *flowAuthResolver) Provision(_ context.Context, _, _, value string) error {
	r.value = value
	return nil
}

func (r *flowAuthResolver) Resolve(context.Context, string, string) (string, error) {
	return r.value, nil
}

func (r *flowAuthResolver) Delete(context.Context, string, string) error {
	r.value = ""
	return nil
}

func (r *flowAuthResolver) Status(context.Context, string, string) authdomain.ProviderStatus {
	return authdomain.ProviderStatus{Provider: "fake", ProviderState: "available", Configured: r.value != ""}
}

type flowAuthStrategy struct {
	*fakes.Strategy
	locked              bool
	failReadAfterUnlock bool
	cancelOnUnlock      context.CancelFunc
}

func (s *flowAuthStrategy) ReadState(ctx context.Context) (strategy.DeviceState, error) {
	if err := ctx.Err(); err != nil {
		return strategy.DeviceState{}, err
	}
	if !s.locked && s.failReadAfterUnlock {
		return strategy.DeviceState{}, errors.New("verification unavailable")
	}
	lockState := "unlocked"
	if s.locked {
		lockState = "locked"
	}
	return strategy.DeviceState{LockState: lockState, ScreenState: "on", Orientation: "portrait"}, nil
}

func (s *flowAuthStrategy) Unlock(_ context.Context, request strategy.UnlockRequest) (strategy.UnlockResult, error) {
	for i := range request.Secret {
		request.Secret[i] = 0
	}
	s.locked = false
	if s.cancelOnUnlock != nil {
		s.cancelOnUnlock()
	}
	return strategy.UnlockResult{Outcome: authdomain.OutcomeUnlocked, Attempts: 1}, nil
}

func (s *flowAuthStrategy) RestoreState(ctx context.Context, state strategy.DeviceState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if state.LockState == "locked" {
		s.locked = true
	}
	return nil
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

func TestUnlockAuditRetainsSafeTransactionMetadata(t *testing.T) {
	svc, _ := testService(t)
	svc.recordAuthAudit(context.Background(), "operator", "fake", "device_unlock", authdomain.OutcomeUnlocked, "lease-1", &authdomain.UnlockResponse{
		ProfileID: "profile-1", Method: authdomain.MethodPIN, Attempts: 1, ProviderState: "available", BeforeLockState: "locked", AfterLockState: "unlocked",
	})

	records := svc.Audit()
	require.Len(t, records, 1)
	require.Equal(t, "profile-1", records[0].ProfileID)
	require.Equal(t, authdomain.MethodPIN, records[0].Method)
	require.Equal(t, 1, records[0].Attempts)
	require.Equal(t, "available", records[0].ProviderState)
	require.Equal(t, "locked", records[0].BeforeLockState)
	require.Equal(t, "unlocked", records[0].AfterLockState)
	require.NotContains(t, string(mustJSON(t, records[0])), "runtime-only-fixture")
}

func TestDirectActuationRequiresLeaseAndWritesOneNonEvidenceAudit(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.ActuateDevice(context.Background(), "fake", "operator", "", DirectActuation{Key: "DPAD_DOWN"})
	require.ErrorContains(t, err, "requires a lease")
	require.Empty(t, svc.Audit())

	lease, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	record, err := svc.ActuateDevice(context.Background(), "fake", "operator", lease.LeaseToken, DirectActuation{Key: "DPAD_DOWN"})
	require.NoError(t, err)
	require.True(t, record.Interactive)
	require.False(t, record.EvidenceBacked)
	require.NotEmpty(t, record.CausationID)
	audits := svc.Audit()
	require.Len(t, audits, 1)
	require.Equal(t, record.ID, audits[0].ID)
	require.Equal(t, "direct-actuation", audits[0].Verb)
}

func TestFailedDirectActuationStillWritesInteractiveAudit(t *testing.T) {
	svc, _ := testService(t)
	fake, ok := svc.registry.Get("fake")
	require.True(t, ok)
	fake.(*fakes.Strategy).ActuateErr = errors.New("fixture transport unavailable")
	lease, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)

	_, err = svc.ActuateDevice(context.Background(), "fake", "operator", lease.LeaseToken, DirectActuation{Key: "DPAD_DOWN"})
	require.ErrorContains(t, err, "fixture transport unavailable")
	audits := svc.Audit()
	require.Len(t, audits, 1)
	require.Equal(t, "failed", audits[0].Outcome)
	require.True(t, audits[0].Interactive)
	require.False(t, audits[0].EvidenceBacked)
}

func TestUnlockRequiresHeldLease(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.UnlockDevice(context.Background(), "profile-1", "fake", "operator", "")
	require.ErrorContains(t, err, "active device lease")
}

func TestPairingPINNeverPersistsOutsideInjectedExchange(t *testing.T) {
	const pin = "835B64"
	pairing := &pairingClientFixture{}
	store := &pairingStoreFixture{}
	remote := androidtvremote.New(
		androidtvremote.WithDiscovery(pairingDiscoveryFixture{}),
		androidtvremote.WithPairingClient(pairing),
		androidtvremote.WithPairingStore(store),
	)
	svc := New(strategyregistry.New(remote))

	result, err := svc.PairDeviceSecret(context.Background(), "android-tv:bt-tv-1", []byte(pin))
	require.NoError(t, err)
	require.Equal(t, "paired", result.Outcome)
	require.Equal(t, pin, pairing.pin, "the injected protocol exchange is the only PIN consumer")
	require.Equal(t, "bt-tv-1", store.serial)
	require.Equal(t, []byte("fixture-certificate"), store.certificate)
	materialized, ok := svc.devices.Get("android-tv:bt-tv-1")
	require.True(t, ok)
	require.Equal(t, "bt-tv-1", materialized.IdentityKey)

	require.NotContains(t, string(mustJSON(t, result)), pin)
	require.NotContains(t, string(mustJSON(t, svc.Audit())), pin)
	declaration, err := remote.Describe(context.Background())
	require.NoError(t, err)
	require.NotContains(t, string(mustJSON(t, declaration)), pin)
	require.Empty(t, svc.artifacts, "pairing must not create evidence artifacts")
}

func TestPairingFailureDoesNotExposePIN(t *testing.T) {
	const pin = "654321"
	remote := androidtvremote.New(
		androidtvremote.WithDiscovery(pairingDiscoveryFixture{}),
		androidtvremote.WithPairingClient(leakyPairingClientFixture{}),
	)
	svc := New(strategyregistry.New(remote))

	_, err := svc.PairDeviceSecret(context.Background(), "android-tv:bt-tv-1", []byte(pin))
	require.Error(t, err)
	require.NotContains(t, err.Error(), pin)
	require.NotContains(t, err.Error(), "remote rejected")
	require.NotContains(t, string(mustJSON(t, svc.Audit())), pin)
}

func TestInteractivePairingStartsHandshakeBeforePINSubmission(t *testing.T) {
	pairing := &interactivePairingClientFixture{}
	store := &pairingStoreFixture{}
	remote := androidtvremote.New(
		androidtvremote.WithDiscovery(pairingDiscoveryFixture{}),
		androidtvremote.WithPairingClient(pairing),
		androidtvremote.WithPairingStore(store),
	)
	svc := New(strategyregistry.New(remote))

	pairingID, err := svc.BeginPairDevice(context.Background(), "android-tv:bt-tv-1")
	require.NoError(t, err)
	require.NotEmpty(t, pairingID)
	require.NotNil(t, pairing.session)
	require.Empty(t, pairing.session.pin, "the PIN must be submitted only after the TV handshake starts")

	result, err := svc.CompletePairDevice(context.Background(), "android-tv:bt-tv-1", pairingID, []byte("835B64"))
	require.NoError(t, err)
	require.Equal(t, "paired", result.Outcome)
	require.Equal(t, "835B64", pairing.session.pin)
	require.True(t, pairing.session.closed)
	require.Equal(t, "bt-tv-1", store.serial)
}

func TestUnlockDeviceUsesPromotedWirelessStrategy(t *testing.T) {
	wireless := &flowAuthStrategy{
		Strategy: fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot),
		locked:   true,
	}
	svc := New(strategyregistry.New(wireless))
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: wireless.ID(), Transport: "wireless"})
	svc.transportStrategies["wireless-device"] = wireless
	resolver := &flowAuthResolver{value: "runtime-only-fixture"}
	store, err := authdomain.NewStore(nil, resolver)
	require.NoError(t, err)
	svc.auth = store
	profile, err := store.Create(context.Background(), authdomain.Profile{
		ID: "profile-wireless", DeviceID: "wireless-device", Method: authdomain.MethodPIN,
		CredentialIdentity: authdomain.CredentialNamespace + "wireless-device/profile-wireless", CredentialField: "unlock",
		Verification: "fresh_lock_state_unlocked",
	})
	require.NoError(t, err)
	lease, err := svc.Acquire("wireless-device", "operator", time.Minute)
	require.NoError(t, err)

	result, err := svc.UnlockDevice(context.Background(), profile.ID, "wireless-device", "operator", lease.LeaseToken)
	require.NoError(t, err)
	require.Equal(t, authdomain.OutcomeUnlocked, result.Outcome)
	require.Equal(t, "locked", result.BeforeLockState)
	require.Equal(t, "unlocked", result.AfterLockState)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return encoded
}

func TestLockedAuthFlowRestoresOriginalKeyguardState(t *testing.T) {
	strategyUnderTest := &flowAuthStrategy{Strategy: fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot), locked: true}
	service := New(strategyregistry.New(strategyUnderTest))
	resolver := &flowAuthResolver{value: "runtime-only-fixture"}
	store, err := authdomain.NewStore(nil, resolver)
	require.NoError(t, err)
	service.auth = store
	profile, err := store.Create(context.Background(), authdomain.Profile{
		ID: "profile-flow", DeviceID: "fake", Method: authdomain.MethodPIN,
		CredentialIdentity: authdomain.CredentialNamespace + "fake/profile-flow", CredentialField: "unlock",
		Verification: "fresh_lock_state_unlocked",
	})
	require.NoError(t, err)

	result, err := service.Run(context.Background(), Flow{RequireUnlocked: true, AuthProfileID: profile.ID}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.True(t, strategyUnderTest.locked, "flow must restore a device that started locked")
	require.Len(t, result.Restoration, 1)
	require.Equal(t, "restored", result.Restoration[0].Status)
	audits := service.Audit()
	require.Equal(t, authdomain.OutcomeUnlocked, audits[0].Outcome)
	require.Equal(t, profile.ID, audits[0].ProfileID)
	require.NotContains(t, string(mustJSON(t, audits[0])), resolver.value)
}

func TestAuthVerificationFailureRestoresOriginalKeyguardState(t *testing.T) {
	strategyUnderTest := &flowAuthStrategy{
		Strategy:            fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot),
		locked:              true,
		failReadAfterUnlock: true,
	}
	service := New(strategyregistry.New(strategyUnderTest))
	resolver := &flowAuthResolver{value: "runtime-only-fixture"}
	store, err := authdomain.NewStore(nil, resolver)
	require.NoError(t, err)
	service.auth = store
	profile, err := store.Create(context.Background(), authdomain.Profile{
		ID: "profile-verification-failure", DeviceID: "fake", Method: authdomain.MethodPIN,
		CredentialIdentity: authdomain.CredentialNamespace + "fake/profile-verification-failure", CredentialField: "unlock",
		Verification: "fresh_lock_state_unlocked",
	})
	require.NoError(t, err)

	result, err := service.Run(context.Background(), Flow{RequireUnlocked: true, AuthProfileID: profile.ID}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "auth_failed", result.Disposition)
	require.True(t, strategyUnderTest.locked, "verification failure must restore a device that started locked")
	require.Len(t, result.Restoration, 1)
	require.Equal(t, "restored", result.Restoration[0].Status)
}

func TestCancelledAuthFlowRestoresStateWithIndependentCleanupContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	strategyUnderTest := &flowAuthStrategy{
		Strategy: fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot),
		locked:   true,
	}
	strategyUnderTest.cancelOnUnlock = cancel
	service := New(strategyregistry.New(strategyUnderTest))
	resolver := &flowAuthResolver{value: "runtime-only-fixture"}
	store, err := authdomain.NewStore(nil, resolver)
	require.NoError(t, err)
	service.auth = store
	profile, err := store.Create(context.Background(), authdomain.Profile{
		ID: "profile-cancelled", DeviceID: "fake", Method: authdomain.MethodPIN,
		CredentialIdentity: authdomain.CredentialNamespace + "fake/profile-cancelled", CredentialField: "unlock",
		Verification: "fresh_lock_state_unlocked",
	})
	require.NoError(t, err)

	result, err := service.Run(ctx, Flow{RequireUnlocked: true, AuthProfileID: profile.ID}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "auth_failed", result.Disposition)
	require.True(t, strategyUnderTest.locked, "cancelled auth must restore a device that started locked")
	require.Len(t, result.Restoration, 1)
	require.Equal(t, "restored", result.Restoration[0].Status)
}

func TestAgentRefusesWithoutSkillAndPromotesPassingRun(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.StartAgent(context.Background(), "observe the screen", "fake", "operator", false)
	require.ErrorContains(t, err, "prompt-manager device-control skill is unavailable")

	run, err := svc.StartAgent(context.Background(), "observe the screen", "fake", "operator", true)
	require.NoError(t, err)
	require.Equal(t, "completed", run.State)
	promoted, err := svc.PromoteAgent(run.ID)
	require.NoError(t, err)
	require.Equal(t, "promoted", promoted.State)
	require.NotEmpty(t, promoted.PromotedFlowID)
	export, err := svc.ExportFlow(promoted.PromotedFlowID)
	require.NoError(t, err)
	require.NotEmpty(t, export.Flow.Steps)
}

type framelessAgentStrategy struct {
	*fakes.Strategy
	commands []strategy.MediaCommand
}

func (s *framelessAgentStrategy) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{Properties: map[string]strategy.PropertyValue{
		"application": {Value: "YouTube", Status: strategy.StatusAvailable, Transport: s.ID()},
	}}, nil
}

func (s *framelessAgentStrategy) ControlMedia(_ context.Context, command strategy.MediaCommand) error {
	s.commands = append(s.commands, command)
	return nil
}

func TestFramelessAgentUsesTypedStateAndDryRunDoesNotActuate(t *testing.T) {
	strategyUnderTest := &framelessAgentStrategy{Strategy: fakes.New("frameless", strategy.StatusAvailable, strategy.CapMedia)}
	svc := New(strategyregistry.New(strategyUnderTest))
	svc.devices.Upsert(devicedomain.Record{ID: "frameless", StrategyID: "frameless", Transport: "frameless", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapMedia, Status: strategy.StatusAvailable}}})

	run, err := svc.StartAgentWithOptions(context.Background(), "pause whatever is playing", "frameless", "operator", true, true)
	require.NoError(t, err)
	require.Equal(t, "completed", run.State)
	require.Equal(t, "passed", run.Result.Disposition)
	require.True(t, run.DryRun)
	require.Empty(t, strategyUnderTest.commands)
	require.Contains(t, run.Result.Chapters[len(run.Result.Chapters)-1].Message, "without sending an actuation")
}

type sequenceAgentPlanner struct {
	plans []internalflows.AgentPlan
	calls int
}

func (p *sequenceAgentPlanner) Plan(context.Context, internalflows.AgentWorld) (internalflows.AgentPlan, error) {
	index := p.calls
	p.calls++
	if index >= len(p.plans) {
		return internalflows.AgentPlan{GoalMet: true}, nil
	}
	return p.plans[index], nil
}

func TestAgentUsesBoundedGoalLoopAndRecordsEachWorldModel(t *testing.T) {
	strategyUnderTest := &framelessAgentStrategy{Strategy: fakes.New("frameless-loop", strategy.StatusAvailable, strategy.CapMedia)}
	svc := New(strategyregistry.New(strategyUnderTest))
	svc.devices.Upsert(devicedomain.Record{ID: "frameless-loop", StrategyID: "frameless-loop", Transport: "frameless-loop", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapMedia, Status: strategy.StatusAvailable}}})
	planner := &sequenceAgentPlanner{plans: []internalflows.AgentPlan{{StepKind: "media-pause", Action: "pause"}, {GoalMet: true}}}
	svc.SetAgentPlanner(planner)

	run, err := svc.StartAgentWithOptions(context.Background(), "pause and verify", "frameless-loop", "operator", true, true)
	require.NoError(t, err)
	require.Equal(t, "completed", run.State)
	require.Equal(t, "passed", run.Result.Disposition)
	require.Equal(t, 2, planner.calls)
	require.Contains(t, run.Result.Chapters[0].ID, "world-model-1")
	require.Contains(t, run.Result.Chapters[1].ID, "iteration-1")
	require.Contains(t, run.Result.Chapters[2].ID, "world-model-2")
	require.Contains(t, run.Result.Chapters[3].ID, "goal-2")
	require.Empty(t, strategyUnderTest.commands)
}

type pollingStateStrategy struct{}

func (pollingStateStrategy) ID() string { return "polling-state" }
func (pollingStateStrategy) Describe(context.Context) (strategy.Declaration, error) {
	return strategy.Declaration{
		StrategyID: "polling-state", Status: strategy.StatusAvailable,
		Capabilities: map[string]strategy.Capability{
			strategy.CapProperty: {Name: strategy.CapProperty, Status: strategy.StatusAvailable, StateClass: strategy.StateBearing},
		},
		Properties:       []strategy.PropertyDescriptor{{Name: "enabled", ValueType: "boolean", StateClass: strategy.StateBearing}},
		StateObservation: strategy.StateObservation{Mode: "poll", Interval: time.Millisecond},
	}, nil
}

func (pollingStateStrategy) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{Properties: map[string]strategy.PropertyValue{
		"enabled": {Value: true, Status: strategy.StatusAvailable, Transport: "polling-state"},
	}}, nil
}

func TestDeclaredPollingTransportStartsObservationWithoutCastSpecialCase(t *testing.T) {
	svc := New(strategyregistry.New(pollingStateStrategy{}))
	subscription := svc.SubscribeStateChanges(2)
	defer subscription.Cancel()
	svc.startObserverLocked(devicedomain.Record{ID: "poll-device", StrategyID: "polling-state", Transport: "polling-state", Status: strategy.StatusAvailable})
	defer func() {
		svc.mu.Lock()
		if cancel := svc.observerCancels["poll-device"]; cancel != nil {
			cancel()
		}
		svc.mu.Unlock()
	}()

	select {
	case event := <-subscription.Events:
		require.Equal(t, "poll-device", event.DeviceID)
		require.Equal(t, "polling-state", event.Transport)
		require.Equal(t, "enabled", event.Attribute)
		require.Equal(t, true, event.NewValue)
	case <-time.After(time.Second):
		t.Fatal("declared polling transport did not emit state")
	}
}

func TestBridgeInventoryFailureIsExplicitlyDegraded(t *testing.T) {
	svc := NewWithAttached(strategyregistry.New(), failingAttachedReader{})
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "bridge host node is unavailable", devices[0].HealthReason)
	require.Equal(t, strategy.StatusUnavailable, devices[0].Status)
	require.NotNil(t, devices[0].Capabilities)
}

func TestDeviceCapabilitiesUseEmptyArrayWhenUnavailable(t *testing.T) {
	device := deviceFromRecord(devicedomain.Record{ID: "offline", Status: strategy.StatusUnavailable})
	require.NotNil(t, device.Capabilities)
	require.Empty(t, device.Capabilities)
}

func TestAndroidCapabilitySelfTestUnavailableResultEmitsPhysicalTargetVerdict(t *testing.T) {
	svc, _ := testService(t)
	svc.devices.Upsert(devicedomain.Record{ID: "fake", Serial: "serial-1", HostNodeID: "host-1", Kind: "physical"})
	result, err := svc.RunAndroidCapabilitySelfTest(context.Background(), "fake", "operator", "")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.NotNil(t, result.Verdict)
	require.Equal(t, commonv1.DeviceKind_DEVICE_KIND_PHYSICAL, result.Verdict.Target.DeviceKind)
	require.Contains(t, result.Verdict.Detail, "device_id=fake")
	require.Contains(t, result.Verdict.Detail, "serial=serial-1")
	require.Contains(t, result.Verdict.Detail, "host_node_id=host-1")
}

func TestAndroidCapabilitySelfTestRefreshesBridgeInventoryBeforeLease(t *testing.T) {
	reader := staticAttachedReader{devices: []AttachedDevice{{
		ID: "bridge-android-1", Name: "Galaxy", HostNodeID: "host-1", Kind: "android",
		Transport: "usb", Serial: "serial-1", OSVersion: "13", TrustState: "trusted", Reachability: "reachable",
	}}}
	svc := NewWithAttached(strategyregistry.New(), reader)
	result, err := svc.RunAndroidCapabilitySelfTest(context.Background(), "bridge-android-1", "operator", "")
	require.NoError(t, err)
	require.Equal(t, "unavailable", result.Disposition)
	require.Equal(t, "serial-1", result.Serial)
	require.Equal(t, "host-1", result.HostNodeID)
}

func TestAndroidCapabilitySelfTestAcceptsEmulatorKind(t *testing.T) {
	svc, _ := testService(t)
	svc.devices.Upsert(devicedomain.Record{ID: "emulator-1", Kind: "emulator", Serial: "emulator-1", HostNodeID: "host-1"})
	result, err := svc.RunAndroidCapabilitySelfTest(context.Background(), "emulator-1", "operator", "")
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestAndroidCapabilitySelfTestReportsCapabilityGap(t *testing.T) {
	svc, _ := testService(t)
	result, err := svc.RunAndroidCapabilitySelfTest(context.Background(), "fake", "operator", "")
	require.NoError(t, err)
	require.Equal(t, "unavailable", result.Disposition)
	require.Contains(t, result.Reason, "not present in device-control inventory")
	require.Empty(t, result.Chapters)
}

func TestUSBProbeDistinguishesAndroidBusPresence(t *testing.T) {
	oldLookPath, oldUSBCommand := execLookPath, usbBusCommand
	t.Cleanup(func() { execLookPath, usbBusCommand = oldLookPath, oldUSBCommand })
	execLookPath = func(string) (string, error) { return "/usr/bin/tool", nil }
	usbBusCommand = func() ([]byte, error) {
		return []byte("Bus 001 Device 002: ID 04e8:6860 Samsung Electronics Co., Ltd\n"), nil
	}
	status, reason := usbBusProbe()
	require.Equal(t, "available", status)
	require.Contains(t, reason, "USB bus")

	usbBusCommand = func() ([]byte, error) { return []byte("Bus 001 Device 001: ID 1d6b:0002 Linux Foundation\n"), nil }
	status, reason = usbBusProbe()
	require.Equal(t, "unavailable", status)
	require.Contains(t, reason, "data-capable cable")
}

func TestCrossPlatformUSBInspectorsUseHostCommands(t *testing.T) { // [REQ:DVC-P0-003]
	oldHost, oldLookPath, oldProfiler, oldWindows := strategy.HostOS, execLookPath, systemProfilerCommand, windowsUSBCommand
	t.Cleanup(func() {
		strategy.HostOS, execLookPath, systemProfilerCommand, windowsUSBCommand = oldHost, oldLookPath, oldProfiler, oldWindows
	})
	execLookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	strategy.HostOS = "darwin"
	systemProfilerCommand = func() ([]byte, error) { return []byte("Vendor ID: 0x04e8\nProduct ID: 0x6860\n"), nil }
	status, reason := usbBusProbe()
	require.Equal(t, "available", status)
	require.NotContains(t, reason, "usbutils")
	strategy.HostOS = "windows"
	windowsUSBCommand = func() ([]byte, error) { return []byte("InstanceId : USB\\VID_04E8&PID_6860\nStatus : OK\n"), nil }
	status, reason = usbBusProbe()
	require.Equal(t, "available", status)
	require.NotContains(t, reason, "usbutils")
	strategy.HostOS = "plan9"
	status, reason = usbBusProbe()
	require.Equal(t, "cannot-inspect", status)
	require.Contains(t, reason, "plan9")
}

func TestUSBLinkStabilityReportsFlaps(t *testing.T) { // [REQ:DVC-P0-003]
	oldHost, oldLookPath, oldCommand := strategy.HostOS, execLookPath, usbBusCommand
	t.Cleanup(func() { strategy.HostOS, execLookPath, usbBusCommand = oldHost, oldLookPath, oldCommand })
	strategy.HostOS = "linux"
	execLookPath = func(string) (string, error) { return "/usr/bin/lsusb", nil }
	outputs := [][]byte{
		[]byte("Bus 001 Device 002: ID 04e8:6860 Samsung\n"),
		[]byte("Bus 001 Device 001: ID 1d6b:0002 Linux Foundation\n"),
		[]byte("Bus 001 Device 002: ID 04e8:6860 Samsung\n"),
	}
	usbBusCommand = func() ([]byte, error) { out := outputs[0]; outputs = outputs[1:]; return out, nil }
	result := sampleUSBLink()
	require.Equal(t, "available", result.Status)
	require.Equal(t, 2, result.FlapCount)
	require.Contains(t, result.Reason, "intermittent")
}

func TestAndroidOnboardingNamesExactSDKRepairCommand(t *testing.T) {
	old := execLookPath
	t.Cleanup(func() { execLookPath = old })
	execLookPath = func(string) (string, error) { return "", os.ErrNotExist }
	rungs := (&Service{}).Onboarding("android")
	for _, rung := range rungs {
		if rung["id"] == "android-sdk" {
			require.Contains(t, rung["next_action"], "vrooli resource install android-sdk")
			return
		}
	}
	t.Fatal("android-sdk rung not returned")
}

func TestGoogleTVOnboardingReflectsStoredRemotePairing(t *testing.T) {
	rungs := (&Service{}).Onboarding("google-tv")
	rungs = applyGoogleTVOnboardingProbe(rungs, []DiscoveredService{
		{StrategyID: "android-tv-remote", Name: "Living Room", Paired: true},
		{StrategyID: "google-cast", Name: "SmartTV 4K"},
	}, nil)
	byID := map[string]map[string]string{}
	for _, rung := range rungs {
		byID[rung["id"]] = rung
	}
	require.Equal(t, "available", byID["google-tv-discovery"]["status"])
	require.Equal(t, "available", byID["remote-pairing"]["status"])
	require.Equal(t, "available", byID["paired-transport"]["status"])
	require.Contains(t, byID["paired-transport"]["next_action"], "without another code")
}

func TestGoogleTVOnboardingPreservesPartialDiscovery(t *testing.T) {
	rungs := (&Service{}).Onboarding("google-tv")
	rungs = applyGoogleTVOnboardingProbe(rungs, []DiscoveredService{
		{StrategyID: "google-cast", Name: "Living Room"},
	}, errors.New("android-tv-remote: browse deadline exceeded"))
	byID := map[string]map[string]string{}
	for _, rung := range rungs {
		byID[rung["id"]] = rung
	}
	require.Equal(t, "degraded", byID["lan-multicast"]["status"])
	require.Equal(t, "available", byID["google-tv-discovery"]["status"])
	require.Contains(t, byID["google-tv-discovery"]["next_action"], "Living Room")
	require.Contains(t, byID["google-tv-discovery"]["next_action"], "degraded")
}

func TestBridgeFailureDoesNotAddPseudoDeviceBesidePhysicalInventory(t *testing.T) {
	adapter := &enumeratingFake{
		Strategy: fakes.New("fake-android", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot),
		devices:  []strategy.Device{{ID: "android-stable", Serial: "serial-1", Model: "Pixel", OSVersion: "13", StrategyID: "fake-android", Transport: "usb", Health: strategy.StatusAvailable}},
	}
	svc := NewWithAttached(strategyregistry.New(adapter), failingAttachedReader{})
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "android-stable", devices[0].ID)
}

func TestADBInventoryClassifiesEmulatorSerialAsEmulator(t *testing.T) {
	adapter := &enumeratingFake{
		Strategy: fakes.New("fake-android", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot),
		devices:  []strategy.Device{{ID: "android-emulator", Serial: "emulator-5554", Model: "sdk_gphone", OSVersion: "16", StrategyID: "fake-android", Transport: "usb", Health: strategy.StatusAvailable}},
	}
	devices := NewWithAttached(strategyregistry.New(adapter), nil).Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "emulator", devices[0].Kind)
	require.Equal(t, "emulator-5554", devices[0].Serial)
}

func TestBridgeAttachmentMergesWithLocalPhysicalIdentity(t *testing.T) { // [REQ:DVC-P0-003]
	adapter := &enumeratingFake{
		Strategy: fakes.New("fake-android", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot),
		devices:  []strategy.Device{{ID: "android-stable", Serial: "serial-1", Model: "Pixel", OSVersion: "13", StrategyID: "fake-android", Transport: "usb", Health: strategy.StatusAvailable}},
	}
	reader := staticAttachedReader{devices: []AttachedDevice{{ID: "android-stable", Name: "Pixel", HostNodeID: "swarminator", Kind: "android", Transport: "usb", Serial: "serial-1", OSVersion: "13", TrustState: "trusted", Reachability: "reachable"}}}
	svc := NewWithAttached(strategyregistry.New(adapter), reader)
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "android-stable", devices[0].ID)
	require.Equal(t, "swarminator", devices[0].HostNodeID)
	require.Equal(t, "fake-android", devices[0].StrategyID)
	require.Equal(t, strategy.StatusAvailable, devices[0].Status)
}

func TestBridgeOnlyAndroidAttachmentIsAPhysicalDevice(t *testing.T) {
	reader := staticAttachedReader{devices: []AttachedDevice{{
		ID: "bridge-android-1", Name: "Galaxy", HostNodeID: "host-1", Kind: "android",
		Transport: "usb", Serial: "serial-1", OSVersion: "13", TrustState: "trusted", Reachability: "reachable",
	}}}
	svc := NewWithAttached(strategyregistry.New(), reader)
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "physical", devices[0].Kind)
	require.Equal(t, "android-adb", devices[0].StrategyID)
	require.Equal(t, "Galaxy", devices[0].Model)
}

func TestBridgeOnlyAttachmentRemainsListedWhenBridgeGoesOffline(t *testing.T) {
	reader := &sequenceAttachedReader{responses: [][]AttachedDevice{{{
		ID: "bridge-android-1", Name: "Galaxy", HostNodeID: "host-1", Kind: "android",
		Transport: "usb", Serial: "serial-1", OSVersion: "13", TrustState: "trusted", Reachability: "reachable",
	}}, nil}}
	svc := NewWithAttached(strategyregistry.New(), reader)
	first := svc.Devices(context.Background())
	require.Len(t, first, 1)
	require.Equal(t, strategy.StatusAvailable, first[0].Status)

	second := svc.Devices(context.Background())
	require.Len(t, second, 1)
	require.Equal(t, "bridge-android-1", second[0].ID)
	require.Equal(t, "physical", second[0].Kind)
	require.Equal(t, strategy.HealthUnreachable, second[0].Status)
	require.Contains(t, second[0].HealthReason, "host-1")
}

func TestEnumerationFailureRemainsVisibleAsUnreachableDevice(t *testing.T) {
	failing := &failingEnumeratingFake{Strategy: fakes.New("lan-failure", strategy.StatusAvailable, strategy.CapInput), err: errors.New("multicast interface unavailable")}
	svc := New(strategyregistry.New(failing))
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "unreachable:lan-failure", devices[0].ID)
	require.Equal(t, strategy.HealthUnreachable, devices[0].Health)
	require.Contains(t, devices[0].HealthReason, "multicast interface unavailable")
}

func TestFlowTransportDefaultsToUSBAndWirelessMustBeExplicit(t *testing.T) { // [REQ:DVC-P0-011]
	svc, _ := testService(t)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: "fake", Transport: "wireless"})
	promoted := fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)
	svc.transportStrategies["wireless-device"] = promoted

	usb, ok := svc.strategyForFlow("wireless-device", "")
	require.True(t, ok)
	require.NotSame(t, promoted, usb)

	wireless, ok := svc.strategyForFlow("wireless-device", "wireless")
	require.True(t, ok)
	require.Same(t, promoted, wireless)

	_, ok = svc.strategyForFlow("wireless-device", "bluetooth")
	require.False(t, ok)
}

func TestValidateUsesPromotedWirelessDeviceStrategy(t *testing.T) {
	db, err := sql.Open("sqlite", "file:control-test-wireless-validate-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	generic := fakes.New("fake", strategy.StatusUnavailable, strategy.CapScreenshot)
	svc, err := NewWithDB(strategyregistry.New(generic), db)
	require.NoError(t, err)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: "fake", Transport: "wireless"})
	svc.transportStrategies["wireless-device"] = fakes.New("fake-wireless", strategy.StatusAvailable, strategy.CapScreenshot)

	report := svc.Validate(context.Background(), Flow{
		Transport: "wireless",
		Steps:     []Step{{ID: "observe", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}}},
	}, "wireless-device")

	require.True(t, report.Runnable)
	require.Empty(t, report.Gaps)
}

func TestWirelessTransportStateSurvivesServiceReconstruction(t *testing.T) { // [REQ:DVC-P0-011]
	svc, db := testService(t)
	first := newPersistentWirelessStrategy()
	svc.registry = strategyregistry.New(first)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: first.ID(), Transport: "usb"})

	device, err := svc.PromoteWireless(context.Background(), "wireless-device")
	require.NoError(t, err)
	require.Equal(t, "wireless", device.Transport)

	reloadedStrategy := newPersistentWirelessStrategy()
	reloaded, err := NewWithDB(strategyregistry.New(reloadedStrategy), db)
	require.NoError(t, err)
	record, ok := reloaded.devices.Get("wireless-device")
	require.True(t, ok)
	require.Equal(t, "wireless", record.Transport)
	restored, ok := reloaded.transportStrategies["wireless-device"]
	require.True(t, ok)
	provider, ok := restored.(interface{ WirelessEndpoint() string })
	require.True(t, ok)
	require.Equal(t, "192.168.1.42:5555", provider.WirelessEndpoint())
}

func TestObservedTransportProfilesSurviveServiceReconstruction(t *testing.T) {
	svc, db := testService(t)
	first := &enumeratingFake{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput), devices: []strategy.Device{{
		ID: "tv-usb", Serial: "tv-serial", IdentityKey: "tv-serial", IdentityKind: string(identitydomain.ADBSerial), Model: "Living room TV", StrategyID: "android-adb", Transport: "usb", Endpoint: "usb", Health: strategy.StatusAvailable,
	}}}
	second := &enumeratingFake{Strategy: fakes.New("android-tv-remote", strategy.StatusAvailable, strategy.CapMedia), devices: []strategy.Device{{
		ID: "tv-remote", Serial: "tv-serial", IdentityKey: "tv-serial", IdentityKind: string(identitydomain.ADBSerial), Model: "Living room TV", StrategyID: "android-tv-remote", Transport: "mdns", Endpoint: "tv.local:6466", Health: strategy.StatusAvailable,
	}}}
	svc.registry = strategyregistry.New(first, second)

	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Len(t, devices[0].Transports, 2)

	reloaded, err := NewWithDB(strategyregistry.New(), db)
	require.NoError(t, err)
	restored, ok := reloaded.devices.Get("tv-usb")
	require.True(t, ok)
	require.Len(t, restored.Transports, 2)
	require.ElementsMatch(t, []string{"android-adb", "android-tv-remote"}, []string{restored.Transports[0].StrategyID, restored.Transports[1].StrategyID})
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM device_control_transport_profiles WHERE device_id = ?`, "tv-usb").Scan(&count))
	require.Equal(t, 2, count)
}

func TestIdentityMergeSplitPreservesClaimsAndAuditReachability(t *testing.T) {
	svc, db := testService(t)
	claimA := identitydomain.IdentityClaim{Kind: identitydomain.CastID, Value: "cast-a", StrategyID: "google-cast", Evidence: "observed"}
	claimB := identitydomain.IdentityClaim{Kind: identitydomain.CastID, Value: "cast-b", StrategyID: "google-cast", Evidence: "observed"}
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "tv-canonical", Kind: "physical", IdentityKey: "cast-a", IdentityKind: string(identitydomain.CastID), Claims: []identitydomain.IdentityClaim{claimA}, StrategyID: "google-cast", Transport: "cast", Endpoint: "192.168.1.10:8009"})
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "tv-member", Kind: "physical", IdentityKey: "cast-b", IdentityKind: string(identitydomain.CastID), Claims: []identitydomain.IdentityClaim{claimB}, StrategyID: "android-tv-remote", Transport: "mdns", Endpoint: "192.168.1.10:6466"})
	_, err := db.Exec(`INSERT INTO device_control_audits (id, actor, device_id, lease_id, verb, outcome, created_at, redaction_verified) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, "audit-before-merge", "operator", "tv-member", "lease", "key", "passed", time.Now().UTC().Format(time.RFC3339Nano), 1)
	require.NoError(t, err)

	merged, err := svc.MergeDevices(context.Background(), "tv-canonical", "tv-member", "cast-id=cast-a")
	require.NoError(t, err)
	require.Equal(t, "tv-canonical", merged.ID)
	require.Len(t, merged.Claims, 3, "the owner assertion and both observed claims remain readable")
	require.Len(t, svc.AuditForDevice(context.Background(), "tv-canonical"), 1)

	split, err := svc.SplitDevice(context.Background(), "tv-canonical")
	require.NoError(t, err)
	require.Len(t, split, 2)
	require.Len(t, svc.AuditForDevice(context.Background(), "tv-canonical"), 1)
	require.Len(t, svc.AuditForDevice(context.Background(), "tv-member"), 1)

	reloaded, err := NewWithDB(strategyregistry.New(), db)
	require.NoError(t, err)
	require.Len(t, reloaded.AuditForDevice(context.Background(), "tv-canonical"), 1)
	require.Len(t, reloaded.AuditForDevice(context.Background(), "tv-member"), 1)
}

func TestFlowsAndAuditsSelectBothGenericTransportProfiles(t *testing.T) {
	minimum, maximum := 0.0, 100.0
	first := fakes.NewPropertyOnly("hub-rest", strategy.PropertyDescriptor{Name: "brightness", ValueType: "number", Writable: true, Minimum: &minimum, Maximum: &maximum}, 10.0)
	second := fakes.NewPropertyOnly("hub-mdns", strategy.PropertyDescriptor{Name: "brightness", ValueType: "number", Writable: true, Minimum: &minimum, Maximum: &maximum}, 20.0)
	svc, _ := testService(t)
	svc.registry = strategyregistry.New(first, second)
	claim := identitydomain.IdentityClaim{Kind: identitydomain.ADBSerial, Value: "hub-serial", Evidence: "observed"}
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "hub-device", Kind: "physical", Serial: "hub-serial", Claims: []identitydomain.IdentityClaim{claim}, StrategyID: first.ID(), Transport: "rest", Endpoint: "ha.example", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "hub-device-mdns", Kind: "physical", Serial: "hub-serial", Claims: []identitydomain.IdentityClaim{claim}, StrategyID: second.ID(), Transport: "mdns", Endpoint: "tv.local:6466", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})

	for _, test := range []struct {
		transport string
		value     float64
	}{
		{transport: "rest", value: 30},
		{transport: "mdns", value: 40},
	} {
		result, err := svc.Run(context.Background(), Flow{Transport: test.transport, Steps: []Step{{ID: "set-brightness", Kind: "property-set", RequiredCapabilities: []string{strategy.CapProperty}, Arguments: map[string]any{"name": "brightness", "value": test.value}}}}, "hub-device", "operator")
		require.NoError(t, err)
		require.Equal(t, "passed", result.Disposition)
	}
	audits := svc.Audit()
	require.Len(t, audits, 2)
	require.ElementsMatch(t, []string{"rest", "mdns"}, []string{audits[0].Transport, audits[1].Transport})
}

func TestReadDeviceStateMergesStateBearingTransportProperties(t *testing.T) {
	minimum, maximum := 0.0, 100.0
	first := fakes.NewPropertyOnly("state-rest", strategy.PropertyDescriptor{Name: "brightness", ValueType: "number", StateClass: strategy.StateBearing, Minimum: &minimum, Maximum: &maximum}, 30.0)
	second := fakes.NewPropertyOnly("state-mdns", strategy.PropertyDescriptor{Name: "volume", ValueType: "number", StateClass: strategy.StateBearing, Minimum: &minimum, Maximum: &maximum}, 0.7)
	svc := New(strategyregistry.New(first, second))
	claim := identitydomain.IdentityClaim{Kind: identitydomain.CastID, Value: "state-device", Evidence: "observed"}
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "state-device", Kind: "physical", Claims: []identitydomain.IdentityClaim{claim}, IdentityKey: "state-device", StrategyID: first.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "state-device-mdns", Kind: "physical", Claims: []identitydomain.IdentityClaim{claim}, IdentityKey: "state-device", StrategyID: second.ID(), Transport: "mdns", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})

	state, err := svc.ReadDeviceState(context.Background(), "state-device")
	require.NoError(t, err)
	require.Equal(t, 30.0, state.Properties["brightness"].Value)
	require.Equal(t, 0.7, state.Properties["volume"].Value)
	require.Equal(t, "rest", state.Properties["brightness"].Transport)
	require.Equal(t, "mdns", state.Properties["volume"].Transport)
}

func TestWirelessInventoryUsesRestoredAdapterCapabilities(t *testing.T) {
	svc, db := testService(t)
	first := newPersistentWirelessStrategy()
	svc.registry = strategyregistry.New(first)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: first.ID(), Transport: "usb"})
	_, err := svc.PromoteWireless(context.Background(), "wireless-device")
	require.NoError(t, err)

	reloadedStrategy := newPersistentWirelessStrategy()
	reloaded, err := NewWithDB(strategyregistry.New(reloadedStrategy), db)
	require.NoError(t, err)

	devices := reloaded.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, strategy.StatusAvailable, devices[0].Status)
	for _, name := range []string{strategy.CapInput, strategy.CapScreenshot} {
		var found strategy.Capability
		for _, capability := range devices[0].Capabilities {
			if capability.Name == name {
				found = capability
				break
			}
		}
		require.Equal(t, strategy.StatusAvailable, found.Status, "wireless capability %s", name)
	}
}

func TestWirelessReconnectPersistsRotatedEndpoint(t *testing.T) {
	svc, db := testService(t)
	first := newPersistentWirelessStrategy()
	svc.registry = strategyregistry.New(first)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: first.ID(), Transport: "usb"})
	_, err := svc.PromoteWireless(context.Background(), "wireless-device")
	require.NoError(t, err)

	reloaded, err := NewWithDB(strategyregistry.New(newPersistentWirelessStrategy()), db)
	require.NoError(t, err)
	device, err := reloaded.ReconnectWireless(context.Background(), "wireless-device")
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, device.Status)
	restored, ok := reloaded.transportStrategies["wireless-device"].(interface{ WirelessEndpoint() string })
	require.True(t, ok)
	require.Equal(t, "192.168.1.43:37123", restored.WirelessEndpoint())

	state := reloaded.transportStates["wireless-device"]
	require.Equal(t, "192.168.1.43:37123", state.Endpoint)
}

func TestRequestStorageUsesRoutedDatabaseAndFileRoots(t *testing.T) { // [REQ:DVC-P0-011]
	primaryPath := filepath.Join(t.TempDir(), "primary.db")
	routed, err := coredb.Open(context.Background(), coredb.Config{
		Driver:       coredb.DriverSQLite,
		DSN:          primaryPath,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })

	primaryRoots := storage.Paths{DataDir: t.TempDir()}
	roots := filerouting.New(primaryRoots)
	fake := fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)
	svc, err := NewWithDB(strategyregistry.New(fake), routed, roots)
	require.NoError(t, err)

	leaseID := "device-control-routing-test"
	testCtx := coredb.WithTestMode(context.Background())
	testDBPath := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, routed.InstallTestPool(context.Background(), testDBPath, leaseID, time.Minute))
	testRoots := storage.Paths{DataDir: t.TempDir()}
	require.NoError(t, roots.InstallTestRoots(testRoots, leaseID, time.Minute))
	t.Cleanup(func() { _ = routed.ClearTestPool(leaseID) })
	t.Cleanup(func() { _ = roots.ClearTestRoots(leaseID) })

	_, err = routed.ExecContext(testCtx, `CREATE TABLE device_control_sessions (
 id TEXT PRIMARY KEY, device_id TEXT NOT NULL, actor TEXT NOT NULL,
 state TEXT NOT NULL, lease_token TEXT NOT NULL DEFAULT '', kill_reason TEXT NOT NULL DEFAULT '',
 expires_at TEXT NOT NULL, created_at TEXT NOT NULL
 )`)
	require.NoError(t, err)

	session, err := svc.AcquireContext(testCtx, "fake", "routing-test", time.Minute)
	require.NoError(t, err)
	require.Len(t, svc.ListSessionsContext(testCtx), 1)
	require.Empty(t, svc.ListSessions(), "test-mode session must not leak into the primary database")

	require.NoError(t, svc.persistArtifact(testCtx, "routed-artifact", []byte("redacted"), "log"))
	artifact, kind, err := svc.ArtifactContext(testCtx, "routed-artifact")
	require.NoError(t, err)
	require.Equal(t, "log", kind)
	require.Equal(t, []byte("redacted"), artifact)
	require.Equal(t, int64(1), roots.LeaseStats().TestRootWrites)
	_, err = os.Stat(filepath.Join(testRoots.DataDir, "evidence", "routed-artifact.bin"))
	require.NoError(t, err)

	_, _ = svc.ReleaseContext(testCtx, session.ID)
}

func TestRunRetainsRedactedCaptureAndTapCoordinates(t *testing.T) { // [REQ:DVC-P0-008]
	svc, _ := testService(t)
	result, err := svc.Run(context.Background(), Flow{ID: "flow-1", Steps: []Step{
		{ID: "capture", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}},
		{ID: "tap", Kind: "tap", Target: "12,34", RequiredCapabilities: []string{strategy.CapInput}},
	}}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Len(t, result.Evidence, 1)
	require.NotEmpty(t, result.Evidence[0].SHA256)
	require.Greater(t, result.Evidence[0].SizeBytes, int64(0))
	artifactPath := svc.artifacts[result.Evidence[0].ID]
	contents, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	require.Len(t, contents, int(result.Evidence[0].SizeBytes))
	fake, ok := svc.registry.Get("fake")
	require.True(t, ok)
	actuator := fake.(*fakes.Strategy)
	calls := actuator.Calls()
	require.Len(t, calls, 1)
	require.InDelta(t, 12, calls[0].Pointer.X, 0)
	require.InDelta(t, 34, calls[0].Pointer.Y, 0)
}

func TestPropertyActuationPublishesLocalStateChangeWithSharedCausation(t *testing.T) {
	minimum, maximum := 0.0, 100.0
	property := fakes.NewPropertyOnly("property-device", strategy.PropertyDescriptor{
		Name: "brightness", ValueType: "number", Writable: true, Minimum: &minimum, Maximum: &maximum,
	}, 20.0)
	db, err := sql.Open("sqlite", "file:control-test-property-events-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	svc, err := NewWithDB(strategyregistry.New(property), db)
	require.NoError(t, err)
	subscription := svc.SubscribeStateChanges(1)
	t.Cleanup(subscription.Cancel)

	result, err := svc.Run(context.Background(), Flow{Steps: []Step{{
		ID: "set-brightness", Kind: "property-set", RequiredCapabilities: []string{strategy.CapProperty},
		Arguments: map[string]any{"name": "brightness", "value": 40.0},
	}}}, "property-device", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)

	select {
	case event := <-subscription.Events:
		require.Equal(t, "property-device", event.DeviceID)
		require.Equal(t, "brightness", event.Attribute)
		require.Equal(t, 20.0, event.OldValue)
		require.Equal(t, 40.0, event.NewValue)
		require.Equal(t, strategy.StateBearing, event.StateClass)
		require.NotEmpty(t, event.CausationID)
		require.NotEmpty(t, event.ObservedAt)
		require.NotEmpty(t, svc.Audit())
		require.Equal(t, event.CausationID, svc.Audit()[0].CausationID)
	case <-time.After(time.Second):
		t.Fatal("property actuation did not publish a local state-change event")
	}
}

func TestSensorReadPublishesOnlyObservedTransitions(t *testing.T) {
	sensor := fakes.NewSensorOnly("sensor-rest", strategy.SensorReading{Name: "temperature", Value: 21.0, StateClass: strategy.StateBearing})
	svc := New(strategyregistry.New(sensor))
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "sensor-device", Kind: "physical", Serial: "sensor-serial", StrategyID: sensor.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	subscription := svc.SubscribeStateChanges(2)
	t.Cleanup(subscription.Cancel)
	flow := Flow{Transport: "rest", Steps: []Step{{ID: "read", Kind: "sensor-read", RequiredCapabilities: []string{strategy.CapSensor}}}}

	result, err := svc.Run(context.Background(), flow, "sensor-device", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	select {
	case event := <-subscription.Events:
		t.Fatalf("first sensor observation unexpectedly emitted event: %+v", event)
	default:
	}

	sensor.Readings[0].Value = 22.5
	result, err = svc.Run(context.Background(), flow, "sensor-device", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	select {
	case event := <-subscription.Events:
		require.Equal(t, "sensor-device", event.DeviceID)
		require.Equal(t, "rest", event.Transport)
		require.Equal(t, "temperature", event.Attribute)
		require.Equal(t, 21.0, event.OldValue)
		require.Equal(t, 22.5, event.NewValue)
		require.Equal(t, strategy.StateBearing, event.StateClass)
		require.NotEmpty(t, event.CausationID)
	case <-time.After(time.Second):
		t.Fatal("sensor transition did not publish a local state-change event")
	}
}

func TestRunRejectsUnredactedCaptureWithoutActor(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Run(context.Background(), Flow{AllowUnredactedCapture: true, Steps: []Step{{ID: "capture", Kind: "observe"}}}, "fake", "")
	require.ErrorContains(t, err, "requires an actor")
}

func TestUnredactedCaptureIsAuditedWithActor(t *testing.T) {
	svc, _ := testService(t)
	result, err := svc.Run(context.Background(), Flow{AllowUnredactedCapture: true, Steps: []Step{{ID: "capture", Kind: "observe"}}}, "fake", "owner-1")
	require.NoError(t, err)
	require.Len(t, result.Evidence, 1)
	require.True(t, result.Evidence[0].OptedOut)
	records := svc.Audit()
	require.NotEmpty(t, records)
	require.Equal(t, "owner-1", records[0].Actor)
	require.True(t, records[0].RedactionOptedOut)
}

func TestRunReusesHeldLeaseTokenWithoutConflict(t *testing.T) { // [REQ:DVC-P0-004]
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	result, err := svc.RunWithLease(context.Background(), Flow{Steps: []Step{{ID: "wait", Kind: "wait"}}}, "fake", "operator", session.LeaseToken)
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Len(t, svc.ListSessions(), 1)
	require.Equal(t, "held", svc.ListSessions()[0].State)
	_, err = svc.Release(session.ID)
	require.NoError(t, err)
}

func TestFrameDifferenceFailsWhenActuationIsSuppressed(t *testing.T) { // [REQ:DVC-P0-008]
	svc, _ := testService(t)
	result, err := svc.Run(context.Background(), Flow{SuppressActuation: true, Steps: []Step{
		{ID: "before", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}},
		{ID: "actuate", Kind: "key", Target: "KEYCODE_APP_SWITCH", RequiredCapabilities: []string{strategy.CapInput}},
		{ID: "after", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}},
		{ID: "changed", Kind: "assert-frame-different"},
	}}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.Contains(t, result.Chapters[len(result.Chapters)-1].Message, "identical")
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

type failingAttachedReader struct{}

func (failingAttachedReader) List(context.Context) ([]AttachedDevice, error) {
	return nil, context.DeadlineExceeded
}

type staticAttachedReader struct{ devices []AttachedDevice }

type persistentWirelessStrategy struct {
	*fakes.Strategy
	endpoint string
}

func newPersistentWirelessStrategy() *persistentWirelessStrategy {
	return &persistentWirelessStrategy{Strategy: fakes.New("persistent-wireless", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)}
}

func (s *persistentWirelessStrategy) Describe(ctx context.Context) (strategy.Declaration, error) {
	if s.endpoint == "" {
		return strategy.UnavailableDeclaration(s.ID(), "wireless endpoint has not been restored", []strategy.Capability{
			{Name: strategy.CapInput},
			{Name: strategy.CapScreenshot},
		}, "restore a verified wireless endpoint"), nil
	}
	return s.Strategy.Describe(ctx)
}

func (s *persistentWirelessStrategy) Enumerate(context.Context) ([]strategy.Device, error) {
	if s.endpoint == "" {
		return nil, nil
	}
	return []strategy.Device{{
		ID: "wireless-device", Serial: "serial-1", Model: "Test phone", StrategyID: s.ID(),
		Transport: "wireless", Health: strategy.StatusAvailable,
	}}, nil
}

func (s *persistentWirelessStrategy) ForDevice(string) strategy.Strategy { return s }

func (s *persistentWirelessStrategy) PromoteWireless(context.Context) error {
	s.endpoint = "192.168.1.42:5555"
	return nil
}

func (s *persistentWirelessStrategy) WirelessEndpoint() string { return s.endpoint }

func (s *persistentWirelessStrategy) ReconnectWireless(context.Context) error {
	s.endpoint = "192.168.1.43:37123"
	return nil
}

func (s *persistentWirelessStrategy) RestoreWireless(endpoint string) strategy.Strategy {
	return &persistentWirelessStrategy{Strategy: s.Strategy, endpoint: endpoint}
}

func (r staticAttachedReader) List(context.Context) ([]AttachedDevice, error) { return r.devices, nil }

type sequenceAttachedReader struct {
	responses [][]AttachedDevice
	index     int
}

func (r *sequenceAttachedReader) List(context.Context) ([]AttachedDevice, error) {
	if r.index >= len(r.responses) {
		return nil, nil
	}
	response := r.responses[r.index]
	r.index++
	return response, nil
}

type enumeratingFake struct {
	*fakes.Strategy
	devices []strategy.Device
}

func (f *enumeratingFake) Enumerate(context.Context) ([]strategy.Device, error) {
	return f.devices, nil
}

type failingEnumeratingFake struct {
	*fakes.Strategy
	err error
}

func (f *failingEnumeratingFake) Enumerate(context.Context) ([]strategy.Device, error) {
	return nil, f.err
}
