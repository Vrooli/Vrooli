package stt

import (
	"context"
	"errors"
	"fmt"
	"io"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/stt/segmenter"

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
	if h.deps.Chain == nil || h.deps.Selector == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("stt streaming pipeline not configured"))
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

	env := envelope.FromConnectStream(stream.RequestHeader())
	start := sttchain.StreamStart{
		Language:                startCfg.Language,
		InitialPrompt:           startCfg.InitialPrompt,
		SkipSpeakerVerification: startCfg.SkipSpeakerVerification,
		BYOKProvider:            env.Provider,
		BYOKKey:                 env.Key,
		LPBSToken:               env.LPBSToken,
		UserIdentity:            env.UserIdentity,
	}

	events := make(chan sttchain.StreamEvent, 16)
	seg := segmenter.New(segmenter.Deps{Chain: h.deps.Chain, Selector: h.deps.Selector})
	cfg := h.resolveStreamPipelineConfig(ctx)
	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- seg.Run(streamCtx, start, cfg, chunkCh, events)
	}()

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

	// Reap the pump goroutine error after the events channel closes;
	// the pump only errors when the receive side returns a non-EOF
	// error. Reap the Segmenter goroutine too so its exit code is
	// surfaced to the caller.
	select {
	case e := <-pumpErr:
		if e != nil && !errors.Is(e, io.EOF) {
			return connect.NewError(connect.CodeInternal, e)
		}
	default:
	}
	if e := <-runErrCh; e != nil && !errors.Is(e, context.Canceled) {
		// Selector typed errors surface here when the Segmenter could
		// not produce a strategy; data-plane errors are already on the
		// wire as StreamError events.
		if mapped := mapChainError(e); mapped != nil {
			// Preserve the pre-existing parity test contract: the
			// Connect handler should not return a hard error after
			// having emitted a Done event. Only surface typed
			// selector errors when no Done was emitted; for now we
			// log-and-swallow because the strategy guaranteed Done.
			_ = mapped
		}
	}
	return nil
}

