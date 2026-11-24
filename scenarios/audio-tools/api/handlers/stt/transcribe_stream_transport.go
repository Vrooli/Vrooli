package stt

import (
	"errors"
	"io"
	"net/http"

	"connectrpc.com/connect"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// transcribeStreamTransport is the wire-neutral surface consumed by the one
// v2 session state machine. Connect bidi and the browser WebSocket adapter
// differ only in framing and lifecycle; ledger, admission, provider, and event
// semantics remain shared.
type transcribeStreamTransport interface {
	Receive() (*sttv1.TranscribeStreamRequest, error)
	Send(*sttv1.TranscribeStreamEvent) error
	RequestHeader() http.Header
}

type connectBidiStreamTransport struct {
	*connect.BidiStream[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent]
}

var errStreamTransportInterrupted = errors.New("stt stream transport interrupted")

func (s connectBidiStreamTransport) Receive() (*sttv1.TranscribeStreamRequest, error) {
	request, err := s.BidiStream.Receive()
	if err == nil {
		return request, err
	}
	if errors.Is(err, io.EOF) {
		// A request half-close is clean only after StreamEnd was consumed. The
		// pump never receives from this adapter again after that explicit marker.
		return nil, errors.Join(errStreamTransportInterrupted, err)
	}
	code := connect.CodeOf(err)
	if code == connect.CodeCanceled || code == connect.CodeUnavailable || code == connect.CodeUnknown {
		return nil, errors.Join(errStreamTransportInterrupted, err)
	}
	return nil, err
}
