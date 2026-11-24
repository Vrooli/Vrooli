package stt

import (
	"errors"
	"io"
	"net/http"
	"time"

	"audio-tools/internal/httporigin"

	"connectrpc.com/connect"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

const BrowserTranscribeStreamWebSocketPath = "/ws/stt/transcribe-stream-v2"

var transcribeStreamWebSocketUpgrader = websocket.Upgrader{
	HandshakeTimeout: 10 * time.Second,
	ReadBufferSize:   4096,
	WriteBufferSize:  4096,
	CheckOrigin: func(req *http.Request) bool {
		origin := req.Header.Get("Origin")
		return origin == "" || httporigin.Matches(req, origin)
	},
}

func (h *connectHandler) serveTranscribeStreamWebSocket(w http.ResponseWriter, req *http.Request) {
	conn, err := transcribeStreamWebSocketUpgrader.Upgrade(w, req, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(DefaultMaxRequestBytes)

	stream := &webSocketTranscribeStream{conn: conn, header: req.Header.Clone()}
	if err := h.transcribeStream(req.Context(), stream); err != nil && req.Context().Err() == nil {
		h.deps.Logger.Printf("event=stt_stream_websocket_handler_failure code=%s error_type=%T", connect.CodeOf(err), err)
		_ = stream.Send(webSocketAdmissionFailure(err))
	}
}

type webSocketTranscribeStream struct {
	conn   *websocket.Conn
	header http.Header
}

func (s *webSocketTranscribeStream) Receive() (*sttv1.TranscribeStreamRequest, error) {
	messageType, payload, err := s.conn.ReadMessage()
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			// StreamEnd is the protocol's clean input boundary. A clean socket
			// close without it must not be interpreted as a complete turn.
			return nil, errors.Join(errStreamTransportInterrupted, io.ErrUnexpectedEOF)
		}
		return nil, errors.Join(errStreamTransportInterrupted, err)
	}
	if messageType != websocket.BinaryMessage {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("stream messages must be protobuf binary frames"))
	}
	request := new(sttv1.TranscribeStreamRequest)
	if err := proto.Unmarshal(payload, request); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid stream protobuf message"))
	}
	return request, nil
}

func (s *webSocketTranscribeStream) Send(event *sttv1.TranscribeStreamEvent) error {
	encoded, err := proto.Marshal(event)
	if err != nil {
		return err
	}
	if err := s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return s.conn.WriteMessage(websocket.BinaryMessage, encoded)
}

func (s *webSocketTranscribeStream) RequestHeader() http.Header { return s.header }

func webSocketAdmissionFailure(err error) *sttv1.TranscribeStreamEvent {
	code := connect.CodeOf(err)
	message := "stream request failed"
	switch code {
	case connect.CodeInvalidArgument:
		message = "invalid stream start"
	case connect.CodeFailedPrecondition:
		message = "stream session cannot be resumed"
	case connect.CodeUnavailable:
		message = "streaming service unavailable"
	case connect.CodeUnauthenticated, connect.CodePermissionDenied:
		message = "stream access denied"
	}
	return &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Error{Error: &sttv1.StreamError{
		Code: "stream_" + code.String(), Message: message,
	}}}
}
