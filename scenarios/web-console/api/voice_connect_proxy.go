package main

import (
	"context"
	"errors"
	"io"
	"net/http"

	"connectrpc.com/connect"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"

	audiotoolsint "web-console/integrations/audiotools"
)

type voiceConnectProxy struct {
	resolver audiotoolsint.URLResolver
	http     *http.Client
}

func newVoiceConnectProxy(resolver audiotoolsint.URLResolver) *connect.Handler {
	proxy := &voiceConnectProxy{resolver: resolver, http: &http.Client{}}
	return connect.NewBidiStreamHandler(sttconnect.STTServiceTranscribeStreamProcedure, proxy.transcribeStream)
}

func (p *voiceConnectProxy) transcribeStream(ctx context.Context, downstream *connect.BidiStream[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent]) error {
	if p == nil || p.resolver == nil {
		return connect.NewError(connect.CodeUnavailable, io.ErrClosedPipe)
	}
	baseURL, err := p.resolver.Resolve()
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	upstream := sttconnect.NewSTTServiceClient(p.http, baseURL).TranscribeStream(ctx)
	copyAudioEnvelope(downstream.RequestHeader(), upstream.RequestHeader())

	go func() {
		for {
			request, receiveErr := downstream.Receive()
			if receiveErr != nil {
				if !errors.Is(receiveErr, io.EOF) {
					cancel()
				}
				_ = upstream.CloseRequest()
				return
			}
			if sendErr := upstream.Send(request); sendErr != nil {
				cancel()
				return
			}
		}
	}()

	for {
		response, receiveErr := upstream.Receive()
		if receiveErr != nil {
			if !errors.Is(receiveErr, io.EOF) {
				cancel()
				_ = upstream.CloseResponse()
				return receiveErr
			}
			return nil
		}
		if sendErr := downstream.Send(response); sendErr != nil {
			cancel()
			_ = upstream.CloseResponse()
			return sendErr
		}
	}
}

func copyAudioEnvelope(from, to http.Header) {
	for _, name := range []string{"X-Audio-BYOK-Provider", "X-Audio-BYOK-Key", "X-Audio-LPBS-Token", "X-Audio-User-Identity"} {
		for _, value := range from.Values(name) {
			to.Add(name, value)
		}
	}
}
