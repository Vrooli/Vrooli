package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	runidentity "agent-manager/internal/identity"
	"github.com/google/uuid"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type treasurySessionKeyStoreFixture struct {
	calls   int
	key     string
	revokes int
}

func (s *treasurySessionKeyStoreFixture) ResolveOrMint(_ context.Context, _ uuid.UUID) (string, error) {
	s.calls++
	return s.key, nil
}

func (s *treasurySessionKeyStoreFixture) Acquire(ctx context.Context, id uuid.UUID) (string, func(), error) {
	key, err := s.ResolveOrMint(ctx, id)
	return key, func() {}, err
}

func (s *treasurySessionKeyStoreFixture) AcquireContinuation(ctx context.Context, id uuid.UUID) (string, func(), error) {
	return s.Acquire(ctx, id)
}

func (s *treasurySessionKeyStoreFixture) RevokeRunCredentialUse(context.Context, uuid.UUID, string) error {
	s.revokes++
	return nil
}

func TestResolveTreasurySessionKeyRequiresVerifiedTreasurySpendScope(t *testing.T) {
	secret := []byte("treasury-session-test-secret")
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	store := &treasurySessionKeyStoreFixture{key: key}
	runID := uuid.New()
	makeToken := func(scopes []string) string {
		t.Helper()
		token, err := runidentity.GenerateToken(&runidentity.Claims{
			RunID: runID, Scopes: scopes, IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}, secret)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}

	got, err := resolveTreasurySessionKey(context.Background(), store, runID, makeToken([]string{"agent-manager:read"}), secret)
	if err != nil || got != "" || store.calls != 0 {
		t.Fatalf("non-Treasury identity provisioned a session key: got=%q calls=%d err=%v", got, store.calls, err)
	}
	got, err = resolveTreasurySessionKey(context.Background(), store, runID, makeToken([]string{"treasury:write"}), secret)
	if err != nil || got != key || store.calls != 1 {
		t.Fatalf("authorized run did not receive its run-owned key: got=%q calls=%d err=%v", got, store.calls, err)
	}

	if _, err := resolveTreasurySessionKey(context.Background(), store, runID, makeToken([]string{"treasury:write"}), []byte("wrong-secret")); err == nil {
		t.Fatal("forged identity reached the key store")
	}
	if store.calls != 1 {
		t.Fatalf("invalid identity reached the key store: calls=%d", store.calls)
	}
	if _, err := resolveTreasurySessionKey(context.Background(), nil, runID, makeToken([]string{"treasury:write"}), secret); err == nil {
		t.Fatal("Treasury-scoped run proceeded without a key authority")
	}
}

func TestMergedEnvVarsProtectsTreasurySessionKeyFromCustomEnv(t *testing.T) {
	run := &domain.Run{RunMode: domain.RunModeInPlace}
	executor := &RunExecutor{
		run:       run,
		customEnv: map[string]string{treasurySessionKeyEnv: "attacker-chosen-key"},
	}
	env := executor.MergedEnvVars()
	if env[treasurySessionKeyEnv] != "" {
		t.Fatalf("Treasury key was exposed before launch lease acquisition: %q", env[treasurySessionKeyEnv])
	}

	withoutKey := &RunExecutor{run: run, customEnv: map[string]string{treasurySessionKeyEnv: "attacker-chosen-key"}}
	if got := withoutKey.MergedEnvVars()[treasurySessionKeyEnv]; got != "" {
		t.Fatalf("caller-supplied Treasury key survived without a server-issued grant: %q", got)
	}
}

func TestMergedEnvVarsNeverExposesEvaluatorReceiptCapability(t *testing.T) {
	run := &domain.Run{RunMode: domain.RunModeInPlace}
	custom := map[string]string{
		"VROOLI_AM_HOOK_RECEIPT_URL":           "http://127.0.0.1:1234/api/v1/runtime/hook-receipts",
		"VROOLI_AM_HOOK_RECEIPT_TOKEN":         "caller-chosen-capability",
		"VROOLI_AM_HOOK_RECEIPT_INVOCATION_ID": "caller-chosen-invocation",
	}
	executor := &RunExecutor{run: run, customEnv: custom}
	env := executor.MergedEnvVars()
	for key := range custom {
		if env[key] != "" {
			t.Fatalf("workload environment exposes evaluator receipt value for %s", key)
		}
	}
}

func TestRunExecutorInjectsOnlyVerifiedTreasuryRunKey(t *testing.T) {
	secret := []byte("treasury-session-test-secret")
	runID := uuid.New()
	token, err := runidentity.GenerateToken(&runidentity.Claims{
		RunID: runID, Scopes: []string{"treasury:write"}, IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}, secret)
	if err != nil {
		t.Fatal(err)
	}
	store := &treasurySessionKeyStoreFixture{key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
	executor := &RunExecutor{
		run: &domain.Run{ID: runID, RunMode: domain.RunModeInPlace}, identityToken: token,
		identitySecret: secret, treasurySessionKeys: store,
		customEnv: map[string]string{treasurySessionKeyEnv: "caller-key"},
	}
	if err := executor.prepareTreasurySessionKey(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := executor.MergedEnvVars()[treasurySessionKeyEnv]; got != "" || store.calls != 1 {
		t.Fatalf("run key escaped into prelaunch environment: got=%q calls=%d", got, store.calls)
	}
	request := &runner.LaunchRequest{Env: []string{treasurySessionKeyEnv + "=caller-key", "SAFE=value"}}
	release, err := treasurySessionLaunchPreparer(store, runID, token, secret)(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if len(request.Env) != 2 || request.Env[0] != "SAFE=value" || request.Env[1] != treasurySessionKeyEnv+"="+store.key {
		t.Fatalf("launch did not receive only protected run key: %#v", request.Env)
	}

	var owner Orchestrator
	WithTreasurySessionKeyStore(store)(&owner)
	if owner.credentialUseReleaser == nil {
		t.Fatal("run-owned key owner was not registered for terminal cleanup")
	}
	if err := owner.credentialUseReleaser.RevokeRunCredentialUse(context.Background(), runID, "completed"); err != nil || store.revokes != 1 {
		t.Fatalf("terminal cleanup did not revoke run key: revokes=%d err=%v", store.revokes, err)
	}
}

func TestCredentialAuthorityTreasurySessionKeySurvivesReopenAndRevokes(t *testing.T) {
	path := t.TempDir() + "/agent-manager-run-keys.enc"
	const passphrase = "test-only-run-key-authority-passphrase"
	runID := uuid.New()
	open := func() (*credentialauthority.Authority, error) {
		return credentialauthority.OpenEncryptedFile(path, passphrase)
	}

	firstOwner := NewCredentialAuthorityTreasurySessionKeysWithOpener(open)
	first, err := firstOwner.ResolveOrMint(context.Background(), runID)
	if err != nil {
		t.Fatalf("mint run-owned Treasury key: %v", err)
	}

	// A fresh store wrapper models Agent Manager or the executor reopening the
	// encrypted authority after a restart. It must recover the same key so the
	// Treasury mandate session remains bound to this run.
	reopenedOwner := NewCredentialAuthorityTreasurySessionKeysWithOpener(open)
	reopened, err := reopenedOwner.ResolveOrMint(context.Background(), runID)
	if err != nil || reopened != first {
		t.Fatalf("reopened authority did not recover the same run-owned key: equal=%t err=%v", reopened == first, err)
	}

	rotated, err := reopenedOwner.Rotate(context.Background(), runID, "rotation-1")
	if err != nil || rotated == first {
		t.Fatalf("rotation did not replace the active run key: changed=%t err=%v", rotated != first, err)
	}
	rotationReplay, err := reopenedOwner.Rotate(context.Background(), runID, "rotation-1")
	if err != nil || rotationReplay != rotated {
		t.Fatalf("same rotation request did not replay its result: same=%t err=%v", rotationReplay == rotated, err)
	}
	active, err := reopenedOwner.ResolveOrMint(context.Background(), runID)
	if err != nil || active != rotated {
		t.Fatalf("active key resolution diverged after rotation: same=%t err=%v", active == rotated, err)
	}
	identity, err := credentialauthority.ParseIdentity(treasurySessionIdentityPrefix + runID.String())
	if err != nil {
		t.Fatalf("parse run-scoped key identity: %v", err)
	}
	authority, err := open()
	if err != nil {
		t.Fatalf("reopen authority after rotation: %v", err)
	}
	state, err := authority.Resolve(identity, treasurySessionKeyField)
	if err != nil || strings.Contains(state, first) {
		t.Fatalf("retired key remains recoverable from current authority state: err=%v", err)
	}

	if err := reopenedOwner.RevokeRunCredentialUse(context.Background(), runID, "completed"); err != nil {
		t.Fatalf("revoke terminal run key: %v", err)
	}
	authority, err = open()
	if err != nil {
		t.Fatalf("reopen authority after revocation: %v", err)
	}
	if _, err := authority.Resolve(identity, treasurySessionKeyField); err != nil {
		t.Fatalf("terminal run's revocation tombstone was not retained: %v", err)
	}
	if _, err := reopenedOwner.ResolveOrMint(context.Background(), runID); err == nil {
		t.Fatal("terminal key cleanup allowed a later request to mint a replacement key")
	}
	if _, err := reopenedOwner.Rotate(context.Background(), runID, "rotation-after-terminal"); err == nil {
		t.Fatal("terminal key cleanup allowed a later request to rotate a replacement key")
	}
	stateRaw, err := authority.Resolve(identity, treasurySessionKeyField)
	if err != nil {
		t.Fatalf("terminal revocation tombstone was not retained: %v", err)
	}
	var tombstone treasurySessionKeyState
	if err := json.Unmarshal([]byte(stateRaw), &tombstone); err != nil || !tombstone.Revoked || tombstone.Key != "" {
		t.Fatalf("terminal tombstone retained key or lost revoked state: state=%+v err=%v", tombstone, err)
	}
}

func TestTreasurySessionKeyLeaseCoversExecutorLaunchInjection(t *testing.T) {
	path := t.TempDir() + "/launch-lease.enc"
	open := func() (*credentialauthority.Authority, error) {
		return credentialauthority.OpenEncryptedFile(path, "launch-lease-test-passphrase")
	}
	store := NewCredentialAuthorityTreasurySessionKeysWithOpener(open)
	runID, secret := uuid.New(), []byte("launch-lease-identity-secret")
	token, err := runidentity.GenerateToken(&runidentity.Claims{RunID: runID, Scopes: []string{"treasury:write"}, IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}, secret)
	if err != nil {
		t.Fatal(err)
	}
	launchReq := &runner.LaunchRequest{Env: []string{"SAFE=1"}}
	release, err := treasurySessionLaunchPreparer(store, runID, token, secret)(context.Background(), launchReq)
	if err != nil {
		t.Fatal(err)
	}
	if len(launchReq.Env) != 2 || !strings.HasPrefix(launchReq.Env[1], treasurySessionKeyEnv+"=") {
		t.Fatalf("launcher did not receive authority key: %#v", launchReq.Env)
	}
	terminalDone := make(chan error, 1)
	go func() {
		releaseGate := treasurySessionKeyGate.writeLock(runID)
		defer releaseGate()
		terminalDone <- store.RevokeRunCredentialUse(context.Background(), runID, "completed")
	}()
	select {
	case err := <-terminalDone:
		t.Fatalf("terminal cleanup overtook launch injection: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	// This is the launcher acceptance boundary: only after it returns may the
	// key lease release and terminal cleanup persist the keyless tombstone.
	release()
	select {
	case err := <-terminalDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("terminal cleanup did not proceed after launcher accepted the process")
	}
	if _, err := store.ResolveOrMint(context.Background(), runID); err == nil {
		t.Fatal("terminal cleanup left the injected key available")
	}
}

func TestTreasuryContinuationRebindsFreshKeyAfterTerminalTombstone(t *testing.T) {
	path := t.TempDir() + "/continuation-rebind.enc"
	open := func() (*credentialauthority.Authority, error) {
		return credentialauthority.OpenEncryptedFile(path, "continuation-rebind-test-passphrase")
	}
	continuedKey := strings.Repeat("2", 64)
	keys := []string{strings.Repeat("1", 64), continuedKey}
	store := NewCredentialAuthorityTreasurySessionKeysWithGenerator(open, func() (string, error) {
		if len(keys) == 0 {
			return "", errors.New("unexpected extra key generation")
		}
		key := keys[0]
		keys = keys[1:]
		return key, nil
	})
	runID, secret := uuid.New(), []byte("continuation-rebind-identity-secret")
	identityToken, err := runidentity.GenerateToken(&runidentity.Claims{RunID: runID, Scopes: []string{"treasury:write"}, IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}, secret)
	if err != nil {
		t.Fatal(err)
	}
	oldKey, err := store.ResolveOrMint(context.Background(), runID)
	if err != nil {
		t.Fatal(err)
	}
	releaseGate := treasurySessionKeyGate.writeLock(runID)
	err = store.RevokeRunCredentialUse(context.Background(), runID, "completed")
	releaseGate()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResolveOrMint(context.Background(), runID); err == nil {
		t.Fatal("ordinary broker path recovered a terminally revoked key")
	}

	launchReq := &runner.LaunchRequest{}
	release, err := treasurySessionContinuationLaunchPreparer(store, runID, identityToken, secret)(context.Background(), launchReq)
	if err != nil {
		t.Fatalf("explicit continuation could not rebind a fresh key: %v", err)
	}
	if len(launchReq.Env) != 1 || launchReq.Env[0] != treasurySessionKeyEnv+"="+continuedKey {
		t.Fatalf("continuation did not receive its fresh key: %#v", launchReq.Env)
	}
	if strings.Contains(strings.Join(launchReq.Env, "\n"), oldKey) {
		t.Fatal("retired private key leaked into continuation")
	}
	current, err := store.ResolveOrMint(context.Background(), runID)
	if err != nil || current == oldKey || current != strings.TrimPrefix(launchReq.Env[0], treasurySessionKeyEnv+"=") {
		t.Fatalf("continuation key was not the only active key: changed=%t err=%v", current != oldKey, err)
	}
	terminalDone := make(chan error, 1)
	go func() {
		releaseGate := treasurySessionKeyGate.writeLock(runID)
		defer releaseGate()
		terminalDone <- store.RevokeRunCredentialUse(context.Background(), runID, "completed-again")
	}()
	select {
	case err := <-terminalDone:
		t.Fatalf("terminal cleanup overtook continuation launch lease: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	release()
	select {
	case err := <-terminalDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("terminal cleanup did not proceed after continuation launch")
	}
	if _, err := store.ResolveOrMint(context.Background(), runID); err == nil {
		t.Fatal("terminal cleanup did not revoke the continuation key")
	}
}

func TestTreasuryRunKeyResolveRacingTerminalCleanupCannotRecreateKey(t *testing.T) {
	path := t.TempDir() + "/agent-manager-run-keys.enc"
	const passphrase = "test-only-run-key-authority-passphrase"
	runID := uuid.New()
	open := func() (*credentialauthority.Authority, error) {
		return credentialauthority.OpenEncryptedFile(path, passphrase)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	key := "1123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	store := NewCredentialAuthorityTreasurySessionKeysWithGenerator(open, func() (string, error) {
		close(started)
		<-release
		return key, nil
	})
	resolveDone := make(chan error, 1)
	go func() {
		got, err := store.ResolveOrMint(context.Background(), runID)
		if err == nil && got != key {
			err = errors.New("resolver returned unexpected fixture key")
		}
		resolveDone <- err
	}()
	<-started // Resolver is inside its serialized read/mint/write critical section.
	revokeDone := make(chan error, 1)
	go func() { revokeDone <- store.RevokeRunCredentialUse(context.Background(), runID, "completed") }()

	select {
	case err := <-revokeDone:
		t.Fatalf("terminal cleanup passed the in-flight mint instead of serializing: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	if err := <-resolveDone; err != nil {
		t.Fatalf("in-flight resolve failed: %v", err)
	}
	if err := <-revokeDone; err != nil {
		t.Fatalf("terminal cleanup failed after resolve: %v", err)
	}

	if _, err := store.ResolveOrMint(context.Background(), runID); err == nil {
		t.Fatal("request after terminal cleanup recreated the run key")
	}
	if _, err := store.Rotate(context.Background(), runID, "after-terminal"); err == nil {
		t.Fatal("request after terminal cleanup rotated the run key")
	}
}

func TestTreasuryRunKeyRequestWaitsForTerminalCommitAndCleanup(t *testing.T) {
	path := t.TempDir() + "/agent-manager-run-keys.enc"
	const passphrase = "test-only-run-key-authority-passphrase"
	runID := uuid.New()
	open := func() (*credentialauthority.Authority, error) {
		return credentialauthority.OpenEncryptedFile(path, passphrase)
	}
	store := NewCredentialAuthorityTreasurySessionKeysWithOpener(open)
	if _, err := store.ResolveOrMint(context.Background(), runID); err != nil {
		t.Fatalf("mint initial run key: %v", err)
	}

	// The lifecycle owner holds the write gate from before committing terminal
	// status through revocation. A broker request arriving during that interval
	// must not return the prior key after the durable terminal commit.
	releaseGate := treasurySessionKeyGate.writeLock(runID)
	gateLocked := true
	defer func() {
		if gateLocked {
			releaseGate()
		}
	}()
	requestDone := make(chan error, 1)
	go func() {
		_, err := store.ResolveOrMint(context.Background(), runID)
		requestDone <- err
	}()
	select {
	case err := <-requestDone:
		t.Fatalf("broker request escaped the terminal transition gate: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	if err := store.RevokeRunCredentialUse(context.Background(), runID, "completed"); err != nil {
		t.Fatalf("terminal key cleanup failed: %v", err)
	}
	releaseGate()
	gateLocked = false
	if err := <-requestDone; err == nil {
		t.Fatal("broker request returned a key after the terminal commit and cleanup")
	}
}

func TestTreasurySessionKeyGatesDoNotSerializeIndependentRuns(t *testing.T) {
	firstRun, secondRun := uuid.New(), uuid.New()
	releaseFirst := treasurySessionKeyGate.writeLock(firstRun)
	defer releaseFirst()
	secondDone := make(chan struct{})
	go func() {
		releaseSecond := treasurySessionKeyGate.readLock(secondRun)
		releaseSecond()
		close(secondDone)
	}()
	select {
	case <-secondDone:
	case <-time.After(time.Second):
		t.Fatal("terminal work for one run blocked an independent run's key delivery")
	}
}
