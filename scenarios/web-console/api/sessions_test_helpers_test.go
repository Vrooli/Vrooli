package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	sessionsH "web-console/handlers/sessions"
	"web-console/internal/sessionstore"
)

// sessionsConnectIface mirrors the methods on the unexported *connectHandler
// inside handlers/sessions, so tests in package main can drive each RPC
// directly without booting an HTTP server.
type sessionsConnectIface interface {
	Create(context.Context, *connect.Request[sessionsv1.CreateRequest]) (*connect.Response[sessionsv1.CreateResponse], error)
	List(context.Context, *connect.Request[sessionsv1.ListRequest]) (*connect.Response[sessionsv1.ListResponse], error)
	Get(context.Context, *connect.Request[sessionsv1.GetRequest]) (*connect.Response[sessionsv1.GetResponse], error)
	Archive(context.Context, *connect.Request[sessionsv1.ArchiveRequest]) (*connect.Response[sessionsv1.ArchiveResponse], error)
	Unarchive(context.Context, *connect.Request[sessionsv1.UnarchiveRequest]) (*connect.Response[sessionsv1.UnarchiveResponse], error)
	Delete(context.Context, *connect.Request[sessionsv1.DeleteRequest]) (*connect.Response[sessionsv1.DeleteResponse], error)
	ListRecoverable(context.Context, *connect.Request[sessionsv1.ListRecoverableRequest]) (*connect.Response[sessionsv1.ListRecoverableResponse], error)
	DismissRecoverable(context.Context, *connect.Request[sessionsv1.DismissRecoverableRequest]) (*connect.Response[sessionsv1.DismissRecoverableResponse], error)
	Recover(context.Context, *connect.Request[sessionsv1.RecoverRequest]) (*connect.Response[sessionsv1.RecoverResponse], error)
	Reopen(context.Context, *connect.Request[sessionsv1.ReopenRequest]) (*connect.Response[sessionsv1.ReopenResponse], error)
	GetArchiveRetention(context.Context, *connect.Request[sessionsv1.GetArchiveRetentionRequest]) (*connect.Response[sessionsv1.GetArchiveRetentionResponse], error)
	PruneArchive(context.Context, *connect.Request[sessionsv1.PruneArchiveRequest]) (*connect.Response[sessionsv1.PruneArchiveResponse], error)
	GetPolicy(context.Context, *connect.Request[sessionsv1.GetPolicyRequest]) (*connect.Response[sessionsv1.GetPolicyResponse], error)
	UpdatePolicy(context.Context, *connect.Request[sessionsv1.UpdatePolicyRequest]) (*connect.Response[sessionsv1.UpdatePolicyResponse], error)
}

func newSessionsConnectHandlerForServer(srv *Server) sessionsConnectIface {
	return sessionsH.NewConnectHandler(sessionsH.Deps{Service: &sessionsH.Adapter{
		Manager:             srv.sessions,
		Store:               srv.sessionStore,
		Idempotency:         srv.idempotency,
		Events:              srv.events,
		Metrics:             srv.metrics,
		Conversations:       srv.conversations,
		CodexCheckpoints:    srv.agentCheckpointStore,
		Workspace:           srv.workspace,
		CopyCodexHome:       copyCodexHome,
		AgentHistoryPresent: func(sessionstore.Metadata) bool { return true },
		RetentionPolicy: func() sessionsH.ArchiveRetentionPolicy {
			cfg := srv.sessions.GetConfig()
			return sessionsH.ArchiveRetentionPolicy{
				MessageLessAge: time.Duration(cfg.ArchiveMessageLessAgeDays) * 24 * time.Hour,
				AgentHomeAge:   time.Duration(cfg.ArchiveAgentHomeAgeDays) * 24 * time.Hour,
				MaxBytes:       cfg.ArchiveMaxBytes,
			}
		},
		AgentHistorySize:  archivedAgentHistorySize,
		PruneAgentHistory: pruneArchivedAgentHistory,
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
