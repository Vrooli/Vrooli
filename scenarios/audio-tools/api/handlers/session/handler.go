// Package session hosts the SessionService Connect-RPC handler.
//
// Wires SessionService methods to the internal/session pub/sub core.
// OpenSession / CloseSession / SendCancel are implemented end-to-end;
// SendText and Subscribe are stubbed pending the browser-voice transport
// integration follow-up.
package session

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	"audio-tools/internal/module"
	intsession "audio-tools/internal/session"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	sessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/session"
	sessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/session/session_v1connect"
)

type Deps struct {
	Registry *intsession.Registry
	Logger   *log.Logger
}

type connectHandler struct {
	sessconnect.UnimplementedSessionServiceHandler
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) OpenSession(ctx context.Context, req *connect.Request[sessv1.OpenSessionRequest]) (*connect.Response[sessv1.OpenSessionResponse], error) {
	transport := req.Msg.Transport
	if transport == "" {
		transport = "fake"
	}
	s := intsession.New(intsession.Options{
		Transport: transport,
		Voice:     req.Msg.Voice,
		Language:  req.Msg.Language,
	})
	h.deps.Registry.Add(s)
	return connect.NewResponse(&sessv1.OpenSessionResponse{SessionId: s.ID(), Transport: transport}), nil
}

func (h *connectHandler) CloseSession(ctx context.Context, req *connect.Request[sessv1.CloseSessionRequest]) (*connect.Response[sessv1.CloseSessionResponse], error) {
	s, err := h.deps.Registry.Get(req.Msg.SessionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	reason := req.Msg.Reason
	if reason == "" {
		reason = "client-request"
	}
	s.Close(reason)
	h.deps.Registry.Remove(s.ID())
	return connect.NewResponse(&sessv1.CloseSessionResponse{}), nil
}

func (h *connectHandler) SendCancel(ctx context.Context, req *connect.Request[sessv1.SendCancelRequest]) (*connect.Response[sessv1.SendCancelResponse], error) {
	s, err := h.deps.Registry.Get(req.Msg.SessionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	s.BargeIn(intsession.BargeInExplicit)
	return connect.NewResponse(&sessv1.SendCancelResponse{}), nil
}

// SendText: stub until TTS-out streaming through session lands. Future
// implementation drives the chosen transport's TTS pipeline.
func (h *connectHandler) SendText(ctx context.Context, req *connect.Request[sessv1.SendTextRequest]) (*connect.Response[sessv1.SendTextResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("session.SendText: TTS-out streaming through session is a follow-up after browser-voice transport-pipeline integration"))
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "session.open", Path: "/vrooli.audio_tools.v1.session.SessionService/OpenSession", Method: "POST", Summary: "Open a voice session", Category: "session"},
	{ID: "session.close", Path: "/vrooli.audio_tools.v1.session.SessionService/CloseSession", Method: "POST", Category: "session"},
	{ID: "session.send_text", Path: "/vrooli.audio_tools.v1.session.SessionService/SendText", Method: "POST", Category: "session"},
	{ID: "session.send_cancel", Path: "/vrooli.audio_tools.v1.session.SessionService/SendCancel", Method: "POST", Summary: "Explicit barge-in", Category: "session"},
	{ID: "session.subscribe", Path: "/vrooli.audio_tools.v1.session.SessionService/Subscribe", Method: "POST", Summary: "Server-streaming session events", Category: "session"},
}

func Module(registry *intsession.Registry, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := sessconnect.NewSessionServiceHandler(NewConnectHandler(Deps{Registry: registry, Logger: logger}))
	return module.Module{
		Name: "session",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
