package main

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	sessionsH "web-console/handlers/sessions"
)

// sessionsConnectIface mirrors the methods on the unexported *connectHandler
// inside handlers/sessions, so tests in package main can drive each RPC
// directly without booting an HTTP server.
type sessionsConnectIface interface {
	Create(context.Context, *connect.Request[sessionsv1.CreateRequest]) (*connect.Response[sessionsv1.CreateResponse], error)
	List(context.Context, *connect.Request[sessionsv1.ListRequest]) (*connect.Response[sessionsv1.ListResponse], error)
	Get(context.Context, *connect.Request[sessionsv1.GetRequest]) (*connect.Response[sessionsv1.GetResponse], error)
	Delete(context.Context, *connect.Request[sessionsv1.DeleteRequest]) (*connect.Response[sessionsv1.DeleteResponse], error)
	ListRecoverable(context.Context, *connect.Request[sessionsv1.ListRecoverableRequest]) (*connect.Response[sessionsv1.ListRecoverableResponse], error)
	DismissRecoverable(context.Context, *connect.Request[sessionsv1.DismissRecoverableRequest]) (*connect.Response[sessionsv1.DismissRecoverableResponse], error)
	Recover(context.Context, *connect.Request[sessionsv1.RecoverRequest]) (*connect.Response[sessionsv1.RecoverResponse], error)
	GetPolicy(context.Context, *connect.Request[sessionsv1.GetPolicyRequest]) (*connect.Response[sessionsv1.GetPolicyResponse], error)
	UpdatePolicy(context.Context, *connect.Request[sessionsv1.UpdatePolicyRequest]) (*connect.Response[sessionsv1.UpdatePolicyResponse], error)
}

func newSessionsConnectHandlerForServer(srv *Server) sessionsConnectIface {
	return sessionsH.NewConnectHandler(sessionsH.Deps{Service: &sessionsH.Adapter{
		Manager:          srv.sessions,
		Store:            srv.sessionStore,
		Idempotency:      srv.idempotency,
		Events:           srv.events,
		Metrics:          srv.metrics,
		Conversations:    srv.conversations,
		CodexCheckpoints: srv.codexCheckpointStore,
		Workspace:        srv.workspace,
		CopyCodexHome:    copyCodexHome,
	}})
}

// callCreate is a small wrapper that builds a CreateRequest from common
// inputs and returns the resulting Session (or the Connect error).
func callCreate(t *testing.T, srv *Server, cols, rows int, idempotencyKey string) (*sessionsv1.Session, error) {
	t.Helper()
	req := connect.NewRequest(&sessionsv1.CreateRequest{
		Cols: int32(cols),
		Rows: int32(rows),
	})
	if idempotencyKey != "" {
		req.Header().Set("X-Idempotency-Key", idempotencyKey)
	}
	resp, err := newSessionsConnectHandlerForServer(srv).Create(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetSession(), nil
}

func callDelete(t *testing.T, srv *Server, id string) error {
	t.Helper()
	_, err := newSessionsConnectHandlerForServer(srv).Delete(context.Background(),
		connect.NewRequest(&sessionsv1.DeleteRequest{Id: id}))
	return err
}

func callGet(t *testing.T, srv *Server, id string) (*sessionsv1.Session, error) {
	t.Helper()
	resp, err := newSessionsConnectHandlerForServer(srv).Get(context.Background(),
		connect.NewRequest(&sessionsv1.GetRequest{Id: id}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetSession(), nil
}

func callList(t *testing.T, srv *Server) ([]*sessionsv1.Session, error) {
	t.Helper()
	resp, err := newSessionsConnectHandlerForServer(srv).List(context.Background(),
		connect.NewRequest(&sessionsv1.ListRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetSessions(), nil
}

func callUpdatePolicy(t *testing.T, srv *Server, id, mode, duration string) (*sessionsv1.PolicyView, error) {
	t.Helper()
	resp, err := newSessionsConnectHandlerForServer(srv).UpdatePolicy(context.Background(),
		connect.NewRequest(&sessionsv1.UpdatePolicyRequest{
			Id:     id,
			Policy: &sessionsv1.ExpirationPolicy{Mode: mode, Duration: duration},
		}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetPolicy(), nil
}

func callGetPolicy(t *testing.T, srv *Server, id string) (*sessionsv1.PolicyView, error) {
	t.Helper()
	resp, err := newSessionsConnectHandlerForServer(srv).GetPolicy(context.Background(),
		connect.NewRequest(&sessionsv1.GetPolicyRequest{Id: id}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetPolicy(), nil
}

// connectCode returns the Connect code of an error, or CodeUnknown.
func connectCode(err error) connect.Code {
	var ce *connect.Error
	if errors.As(err, &ce) {
		return ce.Code()
	}
	return connect.CodeUnknown
}
