package main

import (
	"context"
	"errors"
	"io"
	"net/http"

	"connectrpc.com/connect"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
	audioruntimev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_runtime"
	"google.golang.org/protobuf/proto"

	"web-console/handlers/audio_runtime"
	audiotoolsint "web-console/integrations/audiotools"
)

// voiceConnectProxy is the API transport adapter between the web-console-owned
// streaming contract and audio-tools' matching wire contract.
type voiceConnectProxy struct {
	resolver audiotoolsint.URLResolver
	http     *http.Client
}

func newVoiceConnectProxy(resolver audiotoolsint.URLResolver) *voiceConnectProxy {
	return &voiceConnectProxy{resolver: resolver, http: &http.Client{}}
}

func (p *voiceConnectProxy) TranscribeStream(ctx context.Context, downstream *connect.BidiStream[audioruntimev1.TranscribeStreamRequest, audioruntimev1.TranscribeStreamEvent]) error {
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
			wireRequest, convertErr := toAudioToolsStreamRequest(request)
			if convertErr != nil {
				cancel()
				return
			}
			if sendErr := upstream.Send(wireRequest); sendErr != nil {
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
		wireResponse, convertErr := fromAudioToolsStreamEvent(response)
		if convertErr != nil {
			cancel()
			_ = upstream.CloseResponse()
			return convertErr
		}
		if sendErr := downstream.Send(wireResponse); sendErr != nil {
			cancel()
			_ = upstream.CloseResponse()
			return sendErr
		}
	}
}

func toAudioToolsStreamRequest(request *audioruntimev1.TranscribeStreamRequest) (*sttv1.TranscribeStreamRequest, error) {
	data, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	converted := new(sttv1.TranscribeStreamRequest)
	if err := proto.Unmarshal(data, converted); err != nil {
		return nil, err
	}
	return converted, nil
}

func fromAudioToolsStreamEvent(event *sttv1.TranscribeStreamEvent) (*audioruntimev1.TranscribeStreamEvent, error) {
	data, err := proto.Marshal(event)
	if err != nil {
		return nil, err
	}
	converted := new(audioruntimev1.TranscribeStreamEvent)
	if err := proto.Unmarshal(data, converted); err != nil {
		return nil, err
	}
	return converted, nil
}

func copyAudioEnvelope(from, to http.Header) {
	for _, name := range []string{"X-Audio-BYOK-Provider", "X-Audio-BYOK-Key", "X-Audio-LPBS-Token", "X-Audio-User-Identity"} {
		for _, value := range from.Values(name) {
			to.Add(name, value)
		}
	}
}

var _ audio_runtime.StreamingTranscriber = (*voiceConnectProxy)(nil)
