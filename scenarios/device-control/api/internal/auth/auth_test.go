package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	_ "modernc.org/sqlite"
)

type fakeResolver struct {
	value        string
	status       ProviderStatus
	err          error
	resolveCalls int
}

func (f *fakeResolver) Provision(_ context.Context, _, _, value string) error {
	f.value, f.status.Configured = value, true
	return nil
}
func (f *fakeResolver) Resolve(context.Context, string, string) (string, error) {
	f.resolveCalls++
	return f.value, f.err
}
func (f *fakeResolver) Delete(context.Context, string, string) error          { f.value = ""; return nil }
func (f *fakeResolver) Status(context.Context, string, string) ProviderStatus { return f.status }

type fakeUnlocker struct {
	request strategy.UnlockRequest
	result  strategy.UnlockResult
}

func (f *fakeUnlocker) Unlock(_ context.Context, request strategy.UnlockRequest) (strategy.UnlockResult, error) {
	f.request = request
	return f.result, nil
}

func testProfile() Profile {
	return Profile{ID: "profile-1", DeviceID: "android-test", Method: MethodPIN, CredentialIdentity: CredentialNamespace + "android-test/profile-1", CredentialField: "unlock", Verification: "fresh_lock_state_unlocked"}
}

func TestProfilePersistenceNeverStoresCredentialMaterial(t *testing.T) {
	db, err := sql.Open("sqlite", "file:auth-profile-test?mode=memory&cache=shared")
	require.NoError(t, err)
	defer db.Close()
	resolver := &fakeResolver{status: ProviderStatus{Provider: "fake", ProviderState: "available"}}
	store, err := NewStore(db, resolver)
	require.NoError(t, err)
	profile, err := store.Create(context.Background(), testProfile())
	require.NoError(t, err)
	_, err = store.Provision(context.Background(), profile.ID, strings.NewReader("runtime-only-fixture"))
	require.NoError(t, err)
	rows, err := db.Query(`SELECT id, device_id, method, credential_identity, credential_field, verification, status FROM device_control_auth_profiles`)
	require.NoError(t, err)
	defer rows.Close()
	var values []string
	for rows.Next() {
		var v [7]string
		require.NoError(t, rows.Scan(&v[0], &v[1], &v[2], &v[3], &v[4], &v[5], &v[6]))
		values = append(values, v[:]...)
	}
	require.NotContains(t, values, "runtime-only-fixture")
}

func TestProfileUpdateChangesMetadataWithoutResolvingCredential(t *testing.T) {
	resolver := &fakeResolver{status: ProviderStatus{ProviderState: "available"}}
	store, err := NewStore(nil, resolver)
	require.NoError(t, err)
	profile, err := store.Create(context.Background(), testProfile())
	require.NoError(t, err)
	updated, err := store.Update(context.Background(), profile.ID, Profile{Method: MethodNumeric, Verification: "fresh_lock_state_unlocked", Policy: Policy{MaxAttempts: 1}})
	require.NoError(t, err)
	require.Equal(t, MethodNumeric, updated.Method)
	require.Equal(t, profile.CredentialIdentity, updated.CredentialIdentity)
	require.Equal(t, profile.CreatedAt, updated.CreatedAt)
	require.NotEqual(t, profile.UpdatedAt, updated.UpdatedAt)
	require.Empty(t, resolver.value)
}

func TestUnlockShortCircuitsAlreadyUnlockedWithoutResolving(t *testing.T) {
	resolver := &fakeResolver{err: errors.New("resolver must not be called"), status: ProviderStatus{ProviderState: "available"}}
	store, err := NewStore(nil, resolver)
	require.NoError(t, err)
	profile, err := store.Create(context.Background(), testProfile())
	require.NoError(t, err)
	unlocker := &fakeUnlocker{result: strategy.UnlockResult{Outcome: OutcomeUnlocked}}
	result, err := store.Unlock(context.Background(), profile.ID, profile.DeviceID, strategy.DeviceState{LockState: "unlocked"}, unlocker, func(context.Context) (strategy.DeviceState, error) {
		return strategy.DeviceState{LockState: "unlocked"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeAlreadyUnlocked, result.Outcome)
	require.Nil(t, unlocker.request.Secret)
}

func TestUnlockMapsProviderStatesWithoutExposingValue(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"absent", credentialauthority.ErrProviderAbsent, OutcomeProviderAbsent},
		{"unavailable", credentialauthority.ErrProviderUnavailable, OutcomeProviderDown},
		{"unconfigured", credentialauthority.ErrUnconfigured, OutcomeUnconfigured},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolver := &fakeResolver{err: tc.err, status: ProviderStatus{ProviderState: tc.name}}
			store, err := NewStore(nil, resolver)
			require.NoError(t, err)
			profile, err := store.Create(context.Background(), testProfile())
			require.NoError(t, err)
			result, err := store.Unlock(context.Background(), profile.ID, profile.DeviceID, strategy.DeviceState{LockState: "locked"}, &fakeUnlocker{}, func(context.Context) (strategy.DeviceState, error) { return strategy.DeviceState{}, nil })
			require.NoError(t, err)
			require.Equal(t, tc.want, result.Outcome)
		})
	}
}

func TestUnlockRejectsNonNumericMethodsBeforeResolvingCredential(t *testing.T) {
	for _, tc := range []struct {
		method  string
		outcome string
	}{
		{MethodPassword, OutcomeUnsupported},
		{MethodPattern, OutcomeUnsupported},
		{MethodBiometric, OutcomeHumanRequired},
		{MethodHumanGated, OutcomeHumanRequired},
	} {
		t.Run(tc.method, func(t *testing.T) {
			resolver := &fakeResolver{value: "runtime-only-fixture", status: ProviderStatus{ProviderState: "available", Configured: true}}
			store, err := NewStore(nil, resolver)
			require.NoError(t, err)
			profile := testProfile()
			profile.Method = tc.method
			profile, err = store.Create(context.Background(), profile)
			require.NoError(t, err)
			result, err := store.Unlock(context.Background(), profile.ID, profile.DeviceID, strategy.DeviceState{LockState: "locked"}, &fakeUnlocker{}, func(context.Context) (strategy.DeviceState, error) { return strategy.DeviceState{}, nil })
			require.NoError(t, err)
			require.Equal(t, tc.outcome, result.Outcome)
			require.Zero(t, resolver.resolveCalls)
		})
	}
	resolver := &fakeResolver{status: ProviderStatus{ProviderState: "available"}}
	store, err := NewStore(nil, resolver)
	require.NoError(t, err)
	invalid := testProfile()
	invalid.Method = "mystery"
	_, err = store.Create(context.Background(), invalid)
	require.ErrorContains(t, err, "unsupported authentication method")
	require.Zero(t, resolver.resolveCalls)
}

func TestUnlockRequiresFreshUnlockedPostcondition(t *testing.T) {
	resolver := &fakeResolver{value: "runtime-only-fixture", status: ProviderStatus{ProviderState: "available", Configured: true}}
	store, err := NewStore(nil, resolver)
	require.NoError(t, err)
	profile, err := store.Create(context.Background(), testProfile())
	require.NoError(t, err)
	result, err := store.Unlock(context.Background(), profile.ID, profile.DeviceID, strategy.DeviceState{LockState: "locked"}, &fakeUnlocker{result: strategy.UnlockResult{Outcome: OutcomeUnlocked, Attempts: 1}}, func(context.Context) (strategy.DeviceState, error) {
		return strategy.DeviceState{LockState: "locked"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeVerifiedFailed, result.Outcome)
}

func TestUnlockFailsClosedForMissingRevokedAndUnknownState(t *testing.T) {
	resolver := &fakeResolver{value: "runtime-only-fixture", status: ProviderStatus{ProviderState: "available", Configured: true}}
	store, err := NewStore(nil, resolver)
	require.NoError(t, err)
	profile, err := store.Create(context.Background(), testProfile())
	require.NoError(t, err)

	unknown, err := store.Unlock(context.Background(), profile.ID, profile.DeviceID, strategy.DeviceState{LockState: "unknown"}, &fakeUnlocker{}, func(context.Context) (strategy.DeviceState, error) { return strategy.DeviceState{}, nil })
	require.NoError(t, err)
	require.Equal(t, OutcomeUnknownState, unknown.Outcome)
	require.Zero(t, resolver.resolveCalls)

	revoked, err := store.Revoke(context.Background(), profile.ID)
	require.NoError(t, err)
	require.Equal(t, ProfileRevoked, revoked.Status)
	revokedResult, err := store.Unlock(context.Background(), profile.ID, profile.DeviceID, strategy.DeviceState{LockState: "locked"}, &fakeUnlocker{}, func(context.Context) (strategy.DeviceState, error) { return strategy.DeviceState{}, nil })
	require.NoError(t, err)
	require.Equal(t, OutcomeProfileRevoked, revokedResult.Outcome)
	require.Zero(t, resolver.resolveCalls)

	missing, err := store.Unlock(context.Background(), "missing", profile.DeviceID, strategy.DeviceState{LockState: "locked"}, &fakeUnlocker{}, func(context.Context) (strategy.DeviceState, error) { return strategy.DeviceState{}, nil })
	require.NoError(t, err)
	require.Equal(t, OutcomeProfileMissing, missing.Outcome)
	require.Zero(t, resolver.resolveCalls)
}
