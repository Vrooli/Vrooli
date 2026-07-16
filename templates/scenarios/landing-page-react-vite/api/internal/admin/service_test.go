package admin_test

import (
	"context"
	"landing-page-react-vite-api/internal/admin"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

func TestLoginAndTokenRoundTrip(t *testing.T) {
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, admin.Schema)
	_, err := db.Exec(`DELETE FROM admin_users`)
	require.NoError(t, err)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO admin_users (email, password_hash) VALUES ($1, $2)`, "admin@localhost", string(hash))
	require.NoError(t, err)

	svc := admin.NewService(db, []byte("test-session-secret"))
	ctx := context.Background()
	require.NoError(t, svc.Login(ctx, "admin@localhost", "secret"))
	require.ErrorIs(t, svc.Login(ctx, "admin@localhost", "wrong"), admin.ErrInvalidCredentials)
	require.ErrorIs(t, svc.Login(ctx, "nobody@localhost", "secret"), admin.ErrInvalidCredentials)

	token := svc.EncodeToken("admin@localhost")
	email, ok := svc.DecodeToken(token)
	require.True(t, ok)
	require.Equal(t, "admin@localhost", email)

	_, ok = svc.DecodeToken(token + "tamper")
	require.False(t, ok)
	_, ok = svc.DecodeToken("garbage")
	require.False(t, ok)
}

func TestInterceptorGatesOnCookie(t *testing.T) {
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, admin.Schema)
	svc := admin.NewService(db, []byte("test-session-secret"))

	called := false
	next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		called = true
		return connect.NewResponse(&landingv1.AdminSessionResponse{}), nil
	})
	guarded := svc.Interceptor()(next)
	ctx := context.Background()

	// No cookie -> rejected.
	req := connect.NewRequest(&landingv1.SessionRequest{})
	_, err := guarded(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.False(t, called)

	// Valid session cookie -> passes through.
	authed := connect.NewRequest(&landingv1.SessionRequest{})
	authed.Header().Set("Cookie", svc.SessionCookie("admin@localhost").String())
	_, err = guarded(ctx, authed)
	require.NoError(t, err)
	require.True(t, called)
}
