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

// TranscribeStream is the Connect bidi-stream implementation of the
// streaming STT surface. Both the proto-RPC client path and the
// /api/v1/voice/stream WebSocket transport feed through the same
// sttchain.Chain.Stream entry point — only the wire shape differs.
//
// Wire shape on the Connect side:
//
//	Client → server:  StreamStart { ... }, then AudioChunk*, then StreamEnd
//	Server → client:  StreamPartial*, StreamSegment*, StreamWakeWord*,
//	                  StreamSpeakerRejection*, StreamError?, StreamDone
//
// The chain returns a typed `<-chan sttchain.StreamEvent`; this handler
// translates each event to its proto counterpart and forwards it.
//
// When no streaming-capable provider accepts the session, the chain
// emits a synthetic Segment + Done pair from the buffered fallback path
// so consumers see a consistent shape regardless of routing.
func (h *connectHandler) TranscribeStream(
	ctx context.Context,
	stream *connect.BidiStream[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent],
) error {
	if h.deps.Chain == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("stt chain not configured"))
	}

	// Build the StreamStart from the first inbound message; pump
	// subsequent audio_chunk payloads onto the chain's chunk channel in
	// a goroutine so the chain can begin emitting events immediately.
	first, err := stream.Receive()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("client closed before StreamStart"))
		}
		return connect.NewError(connect.CodeInternal, err)
	}
	startPayload, ok := first.GetPayload().(*sttv1.TranscribeStreamRequest_Start)
	if !ok {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("first message must be StreamStart"))
	}
	startCfg := startPayload.Start

	chunkCh := make(chan sttchain.AudioChunk, 16)
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Pump audio chunks → chunkCh.
	pumpErr := make(chan error, 1)
	go func() {
		defer close(chunkCh)
		for {
			msg, err := stream.Receive()
			if errors.Is(err, io.EOF) {
				pumpErr <- nil
				return
			}
			if err != nil {
				pumpErr <- err
				return
			}
			switch p := msg.GetPayload().(type) {
			case *sttv1.TranscribeStreamRequest_AudioChunk:
				select {
				case chunkCh <- sttchain.AudioChunk{Audio: p.AudioChunk}:
				case <-streamCtx.Done():
					pumpErr <- streamCtx.Err()
					return
				}
			case *sttv1.TranscribeStreamRequest_End:
				pumpErr <- nil
				return
			case *sttv1.TranscribeStreamRequest_Start:
				pumpErr <- fmt.Errorf("duplicate StreamStart")
				return
			}
		}
	}()

	start := sttchain.StreamStart{
		Language:                startCfg.Language,
		InitialPrompt:           startCfg.InitialPrompt,
		SkipSpeakerVerification: startCfg.SkipSpeakerVerification,
		BYOKProvider:            stream.RequestHeader().Get("X-Audio-BYOK-Provider"),
		BYOKKey:                 stream.RequestHeader().Get("X-Audio-BYOK-Key"),
		LPBSToken:               stream.RequestHeader().Get("X-Audio-LPBS-Token"),
		UserIdentity:            stream.RequestHeader().Get("X-Audio-User-Identity"),
	}

	events, err := h.deps.Chain.Stream(streamCtx, start, chunkCh)
	if err != nil {
		return mapChainError(err)
	}

	// Forward each typed StreamEvent to the wire as the matching proto oneof.
	for ev := range events {
		out := protoForEvent(ev)
		if out == nil {
			continue
		}
		if err := stream.Send(out); err != nil {
			return err
		}
	}

	// Reap the pump goroutine error after the chain channel closes; the
	// pump only errors when the receive side returns a non-EOF error.
	select {
	case e := <-pumpErr:
		if e != nil && !errors.Is(e, io.EOF) {
			return connect.NewError(connect.CodeInternal, e)
		}
	default:
	}
	return nil
}

// protoForEvent translates a chain StreamEvent to its proto wire shape.
func protoForEvent(ev sttchain.StreamEvent) *sttv1.TranscribeStreamEvent {
	switch ev.Kind {
	case sttchain.StreamEventPartial:
		if ev.Partial == nil {
			return nil
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_Partial{
				Partial: &sttv1.StreamPartial{Text: ev.Partial.Text},
			},
		}
	case sttchain.StreamEventSegment:
		if ev.Segment == nil {
			return nil
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_Segment{
				Segment: &sttv1.StreamSegment{
					Text:             ev.Segment.Text,
					StartMs:          ev.Segment.StartMs,
					EndMs:            ev.Segment.EndMs,
					DetectedLanguage: ev.Segment.DetectedLanguage,
					ProviderTier:     string(ev.Segment.ProviderTier),
					ModelId:          ev.Segment.ModelID,
					LatencyMs:        ev.Segment.LatencyMs,
				},
			},
		}
	case sttchain.StreamEventWakeWord:
		if ev.WakeWord == nil {
			return nil
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_WakeWord{
				WakeWord: &sttv1.StreamWakeWord{Score: ev.WakeWord.Score, SampleId: ev.WakeWord.SampleID},
			},
		}
	case sttchain.StreamEventSpeakerRejection:
		if ev.SpeakerRejection == nil {
			return nil
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_SpeakerRejection{
				SpeakerRejection: &sttv1.StreamSpeakerRejection{
					Reason:        ev.SpeakerRejection.Reason,
					FallbackUsed:  ev.SpeakerRejection.FallbackUsed,
				},
			},
		}
	case sttchain.StreamEventError:
		msg := ""
		if ev.Error != nil {
			msg = ev.Error.Error()
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_Error{
				Error: &sttv1.StreamError{Code: "provider_failure", Message: msg},
			},
		}
	case sttchain.StreamEventDone:
		done := ev.Done
		if done == nil {
			done = &sttchain.DoneEvent{}
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_Done{
				Done: &sttv1.StreamDone{
					FinalText:       done.FinalText,
					ProviderTier:    string(done.LockedTier),
					ProviderId:      done.ProviderID,
					ModelId:         done.ModelID,
					LatencyMs:       done.LatencyMs,
					FellBackToUnary: done.FellBackToUnary,
				},
			},
		}
	}
	return nil
}
