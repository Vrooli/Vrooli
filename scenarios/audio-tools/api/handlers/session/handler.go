// Package session hosts the SessionService Connect-RPC handler.
package session

import (
	"context"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/modulekit"
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

// SendText fans an assistant message out to every subscriber by
// emitting AssistantDelta + AssistantFinal events. The session pipeline
// owns any actual TTS-out audio; this handler only crosses the
// session-boundary text in -> event-stream surface.
func (h *connectHandler) SendText(ctx context.Context, req *connect.Request[sessv1.SendTextRequest]) (*connect.Response[sessv1.SendTextResponse], error) {
	s, err := h.deps.Registry.Get(req.Msg.SessionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	text := req.Msg.Text
	if text == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("text required"))
	}
	s.EmitEvent(intsession.SessionEvent{
		EventID:        uuid.NewString(),
		SessionID:      s.ID(),
		Type:           intsession.EventAssistantDelta,
		EmittedAt:      time.Now(),
		AssistantDelta: &intsession.AssistantDelta{Text: text},
	})
	s.EmitEvent(intsession.SessionEvent{
		EventID:        uuid.NewString(),
		SessionID:      s.ID(),
		Type:           intsession.EventAssistantFinal,
		EmittedAt:      time.Now(),
		AssistantFinal: &intsession.AssistantFinal{Text: text, HadAudio: false},
	})
	return connect.NewResponse(&sessv1.SendTextResponse{}), nil
}

// Subscribe streams session events to the client until the session
// closes or the client disconnects.
func (h *connectHandler) Subscribe(ctx context.Context, req *connect.Request[sessv1.SubscribeRequest], stream *connect.ServerStream[sessv1.SubscribeResponse]) error {
	s, err := h.deps.Registry.Get(req.Msg.SessionId)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, err)
	}
	_, ch, err := s.Subscribe(ctx, 64)
	if err != nil {
		return connect.NewError(connect.CodeResourceExhausted, err)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			pe := toProto(ev)
			if pe == nil {
				continue
			}
			if err := stream.Send(&sessv1.SubscribeResponse{Event: pe}); err != nil {
				return err
			}
			if ev.Type == intsession.EventClosed {
				return nil
			}
		}
	}
}

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "session.open", Path: "/vrooli.audio_tools.v1.session.SessionService/OpenSession", Method: "POST", Summary: "Open a voice session", Category: "session"},
	{ID: "session.close", Path: "/vrooli.audio_tools.v1.session.SessionService/CloseSession", Method: "POST", Category: "session"},
	{ID: "session.send_text", Path: "/vrooli.audio_tools.v1.session.SessionService/SendText", Method: "POST", Category: "session"},
	{ID: "session.send_cancel", Path: "/vrooli.audio_tools.v1.session.SessionService/SendCancel", Method: "POST", Summary: "Explicit barge-in", Category: "session"},
	{ID: "session.subscribe", Path: "/vrooli.audio_tools.v1.session.SessionService/Subscribe", Method: "POST", Summary: "Server-streaming session events", Category: "session"},
}

func Module(registry *intsession.Registry, logger *log.Logger) modulekit.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, h := sessconnect.NewSessionServiceHandler(NewConnectHandler(Deps{Registry: registry, Logger: logger}))
	return modulekit.Module{
		Name: "session",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
