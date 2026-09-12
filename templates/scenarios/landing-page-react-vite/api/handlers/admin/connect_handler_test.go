package admin_test

import (
	"context"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	adminH "landing-page-react-vite-api/handlers/admin"
	internaladmin "landing-page-react-vite-api/internal/admin"
)

func TestLoginSessionLogoutFlow(t *testing.T) {
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, internaladmin.Schema)
	_, err := db.Exec(`DELETE FROM admin_users`)
	require.NoError(t, err)
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO admin_users (email, password_hash) VALUES ($1, $2)`, "admin@localhost", string(hash))
	require.NoError(t, err)

	h := adminH.NewConnectHandler(adminH.Deps{Service: internaladmin.NewService(db, []byte("test-session-secret"))})
	ctx := context.Background()

	// Bad credentials -> Unauthenticated.
	_, err = h.Login(ctx, connect.NewRequest(&landingv1.LoginRequest{Email: "admin@localhost", Password: "nope"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	// Good credentials -> authenticated + Set-Cookie.
	loginResp, err := h.Login(ctx, connect.NewRequest(&landingv1.LoginRequest{Email: "admin@localhost", Password: "secret"}))
	require.NoError(t, err)
	require.True(t, loginResp.Msg.Authenticated)
	cookie := loginResp.Header().Get("Set-Cookie")
	require.NotEmpty(t, cookie)

	// Session with the cookie -> authenticated.
	sessReq := connect.NewRequest(&landingv1.SessionRequest{})
	sessReq.Header().Set("Cookie", cookie)
	sess, err := h.Session(ctx, sessReq)
	require.NoError(t, err)
	require.True(t, sess.Msg.Authenticated)
	require.Equal(t, "admin@localhost", sess.Msg.Email)

	// Session without a cookie -> not authenticated (no error).
	anon, err := h.Session(ctx, connect.NewRequest(&landingv1.SessionRequest{}))
	require.NoError(t, err)
	require.False(t, anon.Msg.Authenticated)

	// Logout clears the cookie.
	out, err := h.Logout(ctx, connect.NewRequest(&landingv1.LogoutRequest{}))
	require.NoError(t, err)
	require.True(t, out.Msg.Success)
	require.Contains(t, out.Header().Get("Set-Cookie"), internaladmin.CookieName)
}
