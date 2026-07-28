package conversation

import (
	"context"
	"errors"
	"log"
	"strings"

	"connectrpc.com/connect"

	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
)

// Deps wires the seams the Connect conversation handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// ConversationServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// ErrSessionNotFound is the sentinel for an unknown session id. Mapped to
// CodeNotFound.
var ErrSessionNotFound = errors.New("session not found")

// ErrNotFound is the sentinel for an unknown event id or referenced file.
// Mapped to CodeNotFound.
var ErrNotFound = errors.New("not found")

// ErrInvalidArgument is the sentinel for malformed/missing fields and
// validation failures (e.g. summarizing a non-assistant event). Mapped to
// CodeInvalidArgument.
var ErrInvalidArgument = errors.New("invalid argument")

func (h *connectHandler) Get(_ context.Context, req *connect.Request[conversationv1.GetRequest]) (*connect.Response[conversationv1.GetResponse], error) {
	sessionID := strings.TrimSpace(req.Msg.GetSessionId())
	if sessionID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}
	state, err := h.deps.Service.Get(sessionID, req.Msg.GetSinceSequence(), int(req.Msg.GetLimit()), req.Msg.GetBeforeSequence())
	if err != nil {
		return nil, h.classify(err, "conversation.Get")
	}
	return connect.NewResponse(&conversationv1.GetResponse{
		SessionId:      state.SessionID,
		Events:         eventsToProto(state.Events),
		Cursor:         cursorToProto(state.Cursor),
		HasMore:        state.HasMore,
		OldestSequence: state.OldestSequence,
		NewestSequence: state.NewestSequence,
		TotalCount:     state.TotalCount,
	}), nil
}

func (h *connectHandler) Search(_ context.Context, req *connect.Request[conversationv1.SearchRequest]) (*connect.Response[conversationv1.SearchResponse], error) {
	matches, truncated, total, err := h.deps.Service.Search(strings.TrimSpace(req.Msg.GetSessionId()), req.Msg.GetQuery(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, h.classify(err, "conversation.Search")
	}
	out := make([]*conversationv1.SearchMatch, 0, len(matches))
	for _, match := range matches {
		out = append(out, &conversationv1.SearchMatch{EventId: match.EventID, Sequence: match.Sequence, Excerpt: match.Excerpt})
	}
	return connect.NewResponse(&conversationv1.SearchResponse{Matches: out, Truncated: truncated, TotalMatches: total}), nil
}

func (h *connectHandler) GetRange(_ context.Context, req *connect.Request[conversationv1.GetRangeRequest]) (*connect.Response[conversationv1.GetResponse], error) {
	state, err := h.deps.Service.GetRange(strings.TrimSpace(req.Msg.GetSessionId()), req.Msg.GetFromSequence(), req.Msg.GetToSequence())
	if err != nil {
		return nil, h.classify(err, "conversation.GetRange")
	}
	return connect.NewResponse(&conversationv1.GetResponse{SessionId: state.SessionID, Events: eventsToProto(state.Events), Cursor: cursorToProto(state.Cursor)}), nil
}

func (h *connectHandler) UpdateCursor(_ context.Context, req *connect.Request[conversationv1.UpdateCursorRequest]) (*connect.Response[conversationv1.UpdateCursorResponse], error) {
	sessionID := strings.TrimSpace(req.Msg.GetSessionId())
	if sessionID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}
	cursor, err := h.deps.Service.UpdateCursor(sessionID, CursorPatch{
		LastSeenSequence:        req.Msg.GetLastSeenSequence(),
		HasLastSeenSequence:     req.Msg.GetHasLastSeenSequence(),
		LastListenedSequence:    req.Msg.GetLastListenedSequence(),
		HasLastListenedSequence: req.Msg.GetHasLastListenedSequence(),
	})
	if err != nil {
		return nil, h.classify(err, "conversation.UpdateCursor")
	}
	return connect.NewResponse(&conversationv1.UpdateCursorResponse{
		Cursor: cursorToProto(cursor),
	}), nil
}

func (h *connectHandler) SummarizeEvent(ctx context.Context, req *connect.Request[conversationv1.SummarizeEventRequest]) (*connect.Response[conversationv1.SummarizeEventResponse], error) {
	sessionID := strings.TrimSpace(req.Msg.GetSessionId())
	eventID := strings.TrimSpace(req.Msg.GetEventId())
	if sessionID == "" || eventID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id and event_id are required"))
	}
	result, err := h.deps.Service.SummarizeEvent(ctx, sessionID, eventID)
	if err != nil {
		return nil, h.classify(err, "conversation.SummarizeEvent")
	}
	return connect.NewResponse(&conversationv1.SummarizeEventResponse{
		Summarized:       result.Summarized,
		SpeechParagraphs: append([]string(nil), result.SpeechParagraphs...),
		Error:            result.Error,
	}), nil
}

// classify maps the package's sentinel errors to Connect codes. Anything
// unrecognized is logged and returned as CodeInternal so the caller never sees
// an opaque framework error.
func (h *connectHandler) classify(err error, op string) error {
	switch {
	case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		h.deps.Logger.Printf("%s: %v", op, err)
		return connect.NewError(connect.CodeInternal, err)
	}
}

func eventToProto(e Event) *conversationv1.ConversationEvent {
	return &conversationv1.ConversationEvent{
		Id:                       e.ID,
		SessionId:                e.SessionID,
		Source:                   e.Source,
		Role:                     e.Role,
		Text:                     e.Text,
		SpeechParagraphs:         append([]string(nil), e.SpeechParagraphs...),
		OriginalSpeechParagraphs: append([]string(nil), e.OriginalSpeechParagraphs...),
		Summarized:               e.Summarized,
		CreatedAt:                e.CreatedAt,
		Sequence:                 e.Sequence,
		DeliveryState:            e.DeliveryState,
		TtsState:                 e.TTSState,
		ConsumptionState:         e.ConsumptionState,
	}
}

func eventsToProto(in []Event) []*conversationv1.ConversationEvent {
	out := make([]*conversationv1.ConversationEvent, 0, len(in))
	for _, e := range in {
		out = append(out, eventToProto(e))
	}
	return out
}

func cursorToProto(c Cursor) *conversationv1.ConversationCursor {
	return &conversationv1.ConversationCursor{
		LastSeenSequence:     c.LastSeenSequence,
		LastListenedSequence: c.LastListenedSequence,
	}
}
