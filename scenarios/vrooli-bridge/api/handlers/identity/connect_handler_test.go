package identity_test

import (
	"context"
	"testing"

	identityH "vrooli-bridge/handlers/identity"
	internalidentity "vrooli-bridge/internal/identity"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity"
)

type fakeForwarder struct {
	owner internalidentity.Owner
	err   error
}

func (f fakeForwarder) Login(context.Context, internalidentity.Credentials) (internalidentity.Owner, error) {
	return f.owner, f.err
}

func (f fakeForwarder) Register(context.Context, internalidentity.Registration) (internalidentity.Owner, error) {
	return f.owner, f.err
}

func (f fakeForwarder) Refresh(context.Context, string) (internalidentity.Owner, error) {
	return f.owner, f.err
}

// handler is the subset of the (unexported) connect handler the tests drive.
type handler interface {
	Login(context.Context, *connect.Request[identityv1.LoginRequest]) (*connect.Response[identityv1.LoginResponse], error)
	Register(context.Context, *connect.Request[identityv1.RegisterRequest]) (*connect.Response[identityv1.RegisterResponse], error)
	Refresh(context.Context, *connect.Request[identityv1.RefreshRequest]) (*connect.Response[identityv1.RefreshResponse], error)
}

func newHandler(f internalidentity.Owner, err error) handler {
	return identityH.NewConnectHandler(identityH.Deps{Forwarder: fakeForwarder{owner: f, err: err}})
}

func TestLogin(t *testing.T) {
	t.Run("ok returns token + identity", func(t *testing.T) {
		h := newHandler(internalidentity.Owner{Token: "jwt", Email: "o@x.io", UserID: "u-1"}, nil)
		resp, err := h.Login(context.Background(), connect.NewRequest(&identityv1.LoginRequest{Email: "o@x.io", Password: "pw"}))
		require.NoError(t, err)
		assert.Equal(t, "jwt", resp.Msg.GetToken())
		assert.Equal(t, "o@x.io", resp.Msg.GetEmail())
		assert.Equal(t, "u-1", resp.Msg.GetUserId())
	})

	t.Run("invalid credentials -> unauthenticated", func(t *testing.T) {
		h := newHandler(internalidentity.Owner{}, internalidentity.ErrInvalidCredentials)
		_, err := h.Login(context.Background(), connect.NewRequest(&identityv1.LoginRequest{Email: "o@x.io", Password: "bad"}))
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	})

	t.Run("authenticator down -> unavailable", func(t *testing.T) {
		h := newHandler(internalidentity.Owner{}, internalidentity.ErrAuthUnavailable)
		_, err := h.Login(context.Background(), connect.NewRequest(&identityv1.LoginRequest{Email: "o@x.io", Password: "pw"}))
		assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	})
}

func TestRegister(t *testing.T) {
	t.Run("ok returns token", func(t *testing.T) {
		h := newHandler(internalidentity.Owner{Token: "jwt-new", Email: "new@x.io", UserID: "u-2"}, nil)
		resp, err := h.Register(context.Background(), connect.NewRequest(&identityv1.RegisterRequest{Email: "new@x.io", Password: "Str0ng!pw"}))
		require.NoError(t, err)
		assert.Equal(t, "jwt-new", resp.Msg.GetToken())
		assert.Equal(t, "u-2", resp.Msg.GetUserId())
	})

	t.Run("duplicate email -> already exists", func(t *testing.T) {
		h := newHandler(internalidentity.Owner{}, internalidentity.ErrEmailTaken)
		_, err := h.Register(context.Background(), connect.NewRequest(&identityv1.RegisterRequest{Email: "dup@x.io", Password: "pw"}))
		assert.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
	})

	t.Run("weak input -> invalid argument", func(t *testing.T) {
		h := newHandler(internalidentity.Owner{}, internalidentity.ErrInvalidInput)
		_, err := h.Register(context.Background(), connect.NewRequest(&identityv1.RegisterRequest{Email: "x@x.io", Password: "weak"}))
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
}
