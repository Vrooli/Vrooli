package stt

import (
	"context"
	"errors"
	"fmt"
	"io"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/sttchain"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// TranscribeStream is the Connect bidi-stream implementation of the streaming
// STT surface. It mirrors the /api/v1/voice/stream WebSocket transport for
// non-browser consumers (other scenarios, CLIs). The browser path keeps the
// WS transport for tooling-ecosystem reasons.
//
// Minimal-implementation semantics: chunks are accumulated until the client
// sends StreamEnd, then the full audio is passed once through the provider
// chain. This satisfies the wire contract and gives clients a single
// StreamSegment + StreamDone event pair on success. A future iteration will
// add live VAD/wake-word/speaker events by sharing the voice.Service
// segmenter — that work is deferred per Phase 1b's minimum-viable scope.
func (h *connectHandler) TranscribeStream(
	ctx context.Context,
	stream *connect.BidiStream[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent],
) error {
	if h.deps.Chain == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("stt chain not configured"))
	}

	var (
		started bool
		startCfg *sttv1.StreamStart
		audio    []byte
	)

	for {
		msg, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		switch payload := msg.GetPayload().(type) {
		case *sttv1.TranscribeStreamRequest_Start:
			if started {
				return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("duplicate StreamStart"))
			}
			started = true
			startCfg = payload.Start
		case *sttv1.TranscribeStreamRequest_AudioChunk:
			if !started {
				return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("audio_chunk before StreamStart"))
			}
			audio = append(audio, payload.AudioChunk...)
		case *sttv1.TranscribeStreamRequest_End:
			break
		}
		if _, ok := msg.GetPayload().(*sttv1.TranscribeStreamRequest_End); ok {
			break
		}
	}

	if !started {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("no StreamStart received"))
	}

	chainReq := sttchain.Request{
		Audio:        audio,
		Format:       "webm",
		BYOKProvider: stream.RequestHeader().Get("X-Audio-BYOK-Provider"),
		BYOKKey:      stream.RequestHeader().Get("X-Audio-BYOK-Key"),
		LPBSToken:    stream.RequestHeader().Get("X-Audio-LPBS-Token"),
		UserIdentity: stream.RequestHeader().Get("X-Audio-User-Identity"),
	}
	if startCfg != nil {
		chainReq.Language = startCfg.Language
		chainReq.InitialPrompt = startCfg.InitialPrompt
		chainReq.SkipSpeakerVerification = startCfg.SkipSpeakerVerification
	}

	res, err := h.deps.Chain.Execute(ctx, chainReq)
	if err != nil {
		_ = stream.Send(&sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_Error{
				Error: &sttv1.StreamError{Code: "provider_failure", Message: err.Error()},
			},
		})
		return mapChainError(err)
	}

	if err := stream.Send(&sttv1.TranscribeStreamEvent{
		Event: &sttv1.TranscribeStreamEvent_Segment{
			Segment: &sttv1.StreamSegment{
				Text:             res.Text,
				DetectedLanguage: res.DetectedLanguage,
				ProviderTier:     string(res.Tier),
				ModelId:          res.ModelID,
				LatencyMs:        float64(res.Latency.Milliseconds()),
			},
		},
	}); err != nil {
		return err
	}
	if err := stream.Send(&sttv1.TranscribeStreamEvent{
		Event: &sttv1.TranscribeStreamEvent_Done{
			Done: &sttv1.StreamDone{FinalText: res.Text},
		},
	}); err != nil {
		return err
	}
	return nil
}
