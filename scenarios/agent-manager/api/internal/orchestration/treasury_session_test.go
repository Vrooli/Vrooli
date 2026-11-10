package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	runidentity "agent-manager/internal/identity"
	"github.com/google/uuid"
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
		run:                run,
		customEnv:          map[string]string{treasurySessionKeyEnv: "attacker-chosen-key"},
		treasurySessionKey: "authority-issued-key",
	}
	env := executor.MergedEnvVars()
	if env[treasurySessionKeyEnv] != "authority-issued-key" {
		t.Fatalf("custom environment replaced run-owned Treasury key: %q", env[treasurySessionKeyEnv])
	}

	withoutKey := &RunExecutor{run: run, customEnv: map[string]string{treasurySessionKeyEnv: "attacker-chosen-key"}}
	if got := withoutKey.MergedEnvVars()[treasurySessionKeyEnv]; got != "" {
		t.Fatalf("caller-supplied Treasury key survived without a server-issued grant: %q", got)
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
	if got := executor.MergedEnvVars()[treasurySessionKeyEnv]; got != store.key || store.calls != 1 {
		t.Fatalf("verified run did not get its protected session key: got=%q calls=%d", got, store.calls)
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
