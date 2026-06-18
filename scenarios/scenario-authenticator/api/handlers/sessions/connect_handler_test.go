package sessions

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"connectrpc.com/connect"

	apidb "github.com/vrooli/api-core/database"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/sessions"

	"scenario-authenticator/internal/accounts"
	"scenario-authenticator/internal/audit"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/clock"
	"scenario-authenticator/internal/realm"
	"scenario-authenticator/internal/redisstate"
	intsessions "scenario-authenticator/internal/sessions"
	dbtest "scenario-authenticator/internal/testutil/db"
)

func newSvc(t *testing.T) *accounts.Service {
	t.Helper()
	d := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(accounts.Schema), apidb.SchemaProviderFunc(audit.Schema)); err != nil {
		t.Fatalf("schemas: %v", err)
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer := authcrypto.NewSigner(authcrypto.NewKeysFromPair(priv, &priv.PublicKey), authcrypto.SignerConfig{Issuer: realm.Issuer})
	clk := clock.System{}
	svc := accounts.NewService(accounts.ServiceConfig{
		Repo: accounts.NewSQLiteRepository(d, clk), Signer: signer,
		Sessions: intsessions.NewManager(redisstate.NewMemory(), nil),
		Audit:    audit.NewSQLiteLogger(d, clk), Clock: clk,
	})
	return svc
}

func TestSessionsListAndRevokeAll(t *testing.T) {
	svc := newSvc(t)
	h := NewConnectHandler(Deps{Service: svc})
	ctx := context.Background()

	// Two logins → two sessions for the same account.
	if _, err := svc.Register(ctx, accounts.RegisterParams{Email: "s@b.co", Password: "Passw0rd"}, accounts.RequestMeta{}); err != nil {
		t.Fatalf("register: %v", err)
	}
	res, err := svc.Login(ctx, accounts.LoginParams{Email: "s@b.co", Password: "Passw0rd"}, accounts.RequestMeta{})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	access := res.AccessToken

	list, err := h.ListSessions(ctx, connect.NewRequest(&sessionsv1.ListSessionsRequest{AccessToken: access}))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Msg.Sessions) < 2 {
		t.Fatalf("want >=2 sessions, got %d", len(list.Msg.Sessions))
	}

	all, err := h.RevokeAllSessions(ctx, connect.NewRequest(&sessionsv1.RevokeAllSessionsRequest{AccessToken: access}))
	if err != nil {
		t.Fatalf("revoke all: %v", err)
	}
	if all.Msg.RevokedCount < 2 {
		t.Fatalf("revoked %d", all.Msg.RevokedCount)
	}
	after, _ := h.ListSessions(ctx, connect.NewRequest(&sessionsv1.ListSessionsRequest{AccessToken: access}))
	if len(after.Msg.Sessions) != 0 {
		t.Fatalf("sessions remain: %d", len(after.Msg.Sessions))
	}
}

func TestRevokeSessionIdempotent(t *testing.T) {
	svc := newSvc(t)
	h := NewConnectHandler(Deps{Service: svc})
	ctx := context.Background()
	// Revoking an unknown/blank session id is a no-op success — the
	// device-sync-hub un-pair contract.
	if _, err := h.RevokeSession(ctx, connect.NewRequest(&sessionsv1.RevokeSessionRequest{SessionId: "does-not-exist"})); err != nil {
		t.Fatalf("revoke unknown: %v", err)
	}
	if _, err := h.RevokeSession(ctx, connect.NewRequest(&sessionsv1.RevokeSessionRequest{SessionId: ""})); err != nil {
		t.Fatalf("revoke blank: %v", err)
	}
}

func TestListSessionsUnauthenticated(t *testing.T) {
	svc := newSvc(t)
	h := NewConnectHandler(Deps{Service: svc})
	_, err := h.ListSessions(context.Background(), connect.NewRequest(&sessionsv1.ListSessionsRequest{AccessToken: "garbage"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}
