package session

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/protomap"
	intsession "audio-tools/internal/session"

	sessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/session"
)

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the Connect handler. All Deps seams
// (Logger, Clock) are required; nil values panic.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		panic("session.NewConnectHandler requires Deps.Logger")
	}
	if d.Clock == nil {
		panic("session.NewConnectHandler requires Deps.Clock")
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) OpenSession(ctx context.Context, req *connect.Request[sessv1.OpenSessionRequest]) (*connect.Response[sessv1.OpenSessionResponse], error) {
	transport := req.Msg.GetTransport()
	if transport == sessv1.SessionTransport_SESSION_TRANSPORT_UNSPECIFIED {
		transport = sessv1.SessionTransport_SESSION_TRANSPORT_FAKE
	}
	s := intsession.New(intsession.Options{
		Transport: protomap.SessionTransportFromProto(transport),
		Voice:     req.Msg.GetVoice(),
		Language:  req.Msg.GetLanguage(),
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
		EmittedAt:      h.deps.Clock.Now(),
		AssistantDelta: &intsession.AssistantDelta{Text: text},
	})
	s.EmitEvent(intsession.SessionEvent{
		EventID:        uuid.NewString(),
		SessionID:      s.ID(),
		Type:           intsession.EventAssistantFinal,
		EmittedAt:      h.deps.Clock.Now(),
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
