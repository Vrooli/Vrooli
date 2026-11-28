package control

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "device-control/internal/auth"
	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	"device-control/strategy/androidtvremote"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
)

func TestSemanticReadyUsesVerifiedAuthenticationProfile(t *testing.T) {
	adapter := &semanticScreenFixture{Strategy: fakes.New("screen-fixture", strategy.StatusAvailable, strategy.CapScreenControl), state: strategy.DeviceState{ScreenState: "off", LockState: "locked"}}
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: adapter.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	store, err := authdomain.NewStore(nil, &flowAuthResolver{value: "1234"})
	require.NoError(t, err)
	svc.auth = store
	profile, err := store.Create(context.Background(), authdomain.Profile{ID: "profile", DeviceID: "target", Method: authdomain.MethodPIN, CredentialIdentity: authdomain.CredentialNamespace + "target/profile", CredentialField: "unlock", Verification: "fresh_lock_state_unlocked"})
	require.NoError(t, err)
	result, err := svc.Screen(context.Background(), ScreenRequest{Device: "target", Operation: "ready", AuthProfileID: profile.ID})
	require.NoError(t, err)
	require.Equal(t, "verified", result.Status)
	require.Equal(t, "on", result.After.ScreenState)
	require.Equal(t, "unlocked", result.After.LockState)
	require.NotContains(t, string(mustJSON(t, result)), "1234")
	require.Empty(t, svc.ListLiveSessions())
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
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: wireless.ID(), Transport: "wireless", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
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
