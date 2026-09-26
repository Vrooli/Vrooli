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
	"scenario-authenticator/internal/realm"
	"scenario-authenticator/internal/redisstate"
	intsessions "scenario-authenticator/internal/sessions"

	dbtest "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"
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
	clk := schedule.System()
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
	if _, err := h.ListSessions(ctx, connect.NewRequest(&sessionsv1.ListSessionsRequest{AccessToken: access})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("revoked access token still listed sessions: %v", err)
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

func TestAuthorizedSingleSessionRevokeRequiresOwnerOrAdmin(t *testing.T) {
	svc := newSvc(t)
	h := NewConnectHandler(Deps{Service: svc})
	ctx := context.Background()
	admin, err := svc.Register(ctx, accounts.RegisterParams{Email: "session-admin@b.co", Password: "Passw0rd"}, accounts.RequestMeta{})
	if err != nil {
		t.Fatalf("register admin: %v", err)
	}
	member, err := svc.Register(ctx, accounts.RegisterParams{Email: "session-member@b.co", Password: "Passw0rd"}, accounts.RequestMeta{})
	if err != nil {
		t.Fatalf("register member: %v", err)
	}
	list, err := h.ListSessions(ctx, connect.NewRequest(&sessionsv1.ListSessionsRequest{AccessToken: member.AccessToken}))
	if err != nil || len(list.Msg.Sessions) != 1 {
		t.Fatalf("member sessions = %v, err=%v", list.Msg.Sessions, err)
	}
	sessionID := list.Msg.Sessions[0].Id

	if _, err := h.RevokeAuthorizedSession(ctx, connect.NewRequest(&sessionsv1.RevokeAuthorizedSessionRequest{
		AccessToken: admin.AccessToken, SessionId: sessionID,
	})); err != nil {
		t.Fatalf("admin revoke: %v", err)
	}
	if _, err := h.ListSessions(ctx, connect.NewRequest(&sessionsv1.ListSessionsRequest{AccessToken: member.AccessToken})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("revoked member access token still listed sessions: %v", err)
	}

	_, err = h.RevokeAuthorizedSession(ctx, connect.NewRequest(&sessionsv1.RevokeAuthorizedSessionRequest{
		AccessToken: "invalid", SessionId: sessionID,
	}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("invalid authorized revoke error = %v, want unauthenticated", err)
	}
}
