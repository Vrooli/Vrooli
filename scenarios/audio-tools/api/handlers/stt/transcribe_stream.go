package stt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/logx"
	"audio-tools/internal/protomap"
	"audio-tools/internal/stt/segmenter"
	"audio-tools/internal/stt/session"
	"audio-tools/internal/sttengine"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// reportStreamActive resolves the engine that will serve this stream to its
// backing local resource and marks that resource's capacity claim active for
// the session, returning the claim id to release on teardown. Returns "" (and
// reports nothing) when capacity reporting is not wired, the engine is not a
// local resource, or the registry is unavailable. Best-effort and advisory.
func (h *connectHandler) reportStreamActive(ctx context.Context, engineID string) string {
	if h.deps.Capacity == nil || h.deps.Registry == nil {
		return ""
	}
	if engineID == "" {
		if resolution, err := h.resolveEngine(ctx); err == nil {
			engineID = resolution.Selected
		}
	}
	engine, ok := h.deps.Registry.Engine(engineID)
	if !ok || engine.Kind != sttengine.KindLocalResource || engine.Resource == "" {
		return ""
	}
	return h.deps.Capacity.Active(ctx, engine.Resource)
}

// TranscribeStream is the Connect bidi-stream implementation of browser voice
// capture. It feeds the STT pipeline and session ledger directly.
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
	return h.transcribeStream(ctx, connectBidiStreamTransport{BidiStream: stream})
}

func (h *connectHandler) transcribeStream(ctx context.Context, stream transcribeStreamTransport) error {
	startCfg, registry, ledger, resumed, err := h.openStreamSession(ctx, stream)
	if err != nil {
		return err
	}
	// Confirm admission before the client starts sending PCM. Browser consumers
	// use this response as the server-side readiness boundary; without it, a
	// client that correctly waits for admission before sending would deadlock.
	if err := stream.Send(protoSessionStatus(&sttchain.SessionStatusEvent{
		SessionID:  startCfg.SessionId,
		Generation: startCfg.Generation,
		State:      "accepted",
	})); err != nil {
		return err
	}
	// A processed acknowledgement may have compacted the PCM that produced a
	// committed segment before the browser received that segment. On resume,
	// replay the durable commits from the ledger before processing any retained
	// audio. Consumers deduplicate these stable segment IDs across reconnects.
	if resumed {
		if err := replayCommittedSegments(stream, ledger, startCfg); err != nil {
			return err
		}
	}

	chunkCh := make(chan sttchain.AudioChunk, 16)
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	idleTimeout := h.deps.StreamIdleTimeout
	if idleTimeout <= 0 {
		idleTimeout = defaultStreamIdleTimeout
	}
	timerFactory := h.deps.StreamIdleTimerFactory
	if timerFactory == nil {
		timerFactory = newWallStreamIdleTimer
	}
	activity := make(chan struct{}, 1)
	inputDone := make(chan struct{}, 1)
	idleExpired := watchStreamIdle(streamCtx, timerFactory(idleTimeout), idleTimeout, activity, inputDone, cancel)
	env := envelope.FromConnectStream(stream.RequestHeader())
	env.Provider, env.Key = h.applyDefaultCredential(ctx, "stt", env.Provider, env.Key)
	start := sttchain.StreamStart{
		ProtocolVersion:         startCfg.ProtocolVersion,
		SessionID:               startCfg.SessionId,
		Generation:              startCfg.Generation,
		ResumeToken:             startCfg.ResumeToken,
		Language:                startCfg.Language,
		InitialPrompt:           startCfg.InitialPrompt,
		SkipSpeakerVerification: startCfg.SkipSpeakerVerification,
		// input_format declares the inbound codec; empty proto enum maps to
		// "" so the Segmenter sniffs the first chunk.
		InputFormat:     protomap.AudioFormatFromProto(startCfg.GetInputFormat()),
		InputSampleRate: startCfg.GetInputSampleRateHz(),
		BYOKProvider:    env.Provider,
		BYOKKey:         env.Key,
		LPBSToken:       env.LPBSToken,
		UserIdentity:    env.UserIdentity,
	}

	events := make(chan sttchain.StreamEvent, 16)
	seg := segmenter.New(segmenter.Deps{Chain: h.deps.Chain, Selector: h.deps.Selector, Engine: h.deps.Engine, Registry: h.deps.Registry, EngineResolver: h.deps.EngineResolver, SpeakerIsolation: currentSpeakerIsolation(h.deps), SpeakerExtraction: currentSpeakerExtraction(h.deps)})
	cfg := h.resolveStreamPipelineConfig(ctx)
	// StreamStart.config carries the per-session engine override. Apply it before
	// entering the Segmenter so the selected engine matches the request.
	if engineID := streamStartEngineID(startCfg); engineID != "" {
		cfg.EngineID = engineID
	}

	// Report transcription activity to the capacity broker for the whole session
	// (advisory, best-effort): mark the backing local resource active now and
	// idle when the session ends. Bracketing the session — not each segment —
	// avoids active/idle thrash from the per-segment Transcribe calls. Idle uses
	// a detached context because streamCtx is cancelled on session teardown.
	if claimID := h.reportStreamActive(ctx, cfg.EngineID); claimID != "" {
		defer h.deps.Capacity.Idle(context.WithoutCancel(ctx), claimID)
	}

	runErrCh := make(chan error, 1)
	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		runErrCh <- seg.Run(streamCtx, start, cfg, chunkCh, events)
	}()

	forwarded := make(chan struct {
		terminal *sttchain.StreamEvent
		err      error
	}, 1)
	go func() {
		terminal, err := h.forwardStreamEvents(ctx, stream, registry, ledger, &start, events, resumed)
		forwarded <- struct {
			terminal *sttchain.StreamEvent
			err      error
		}{terminal: terminal, err: err}
	}()

	// The server ledger is authoritative for any audio that remains unprocessed
	// after a disconnect. Feed that retained prefix into the new pipeline before
	// accepting new client data; on resume the browser may already have compacted
	// these chunks after receiving an adapter-acceptance acknowledgement. Client
	// retries of the same sequence are then deduplicated by the ledger.
	if resumed {
		for _, chunk := range ledger.Snapshot().Replay {
			select {
			case chunkCh <- sttchain.AudioChunk{Audio: chunk.Audio, Sequence: chunk.Sequence, StartSample: chunk.StartSample, EndSample: chunk.EndSample}:
			case <-runDone:
				cancel()
				runErr := <-runErrCh
				if runErr == nil {
					runErr = fmt.Errorf("stream ended before retained audio replay completed")
				}
				return connect.NewError(connect.CodeInternal, runErr)
			case <-streamCtx.Done():
				cancel()
				return streamCtx.Err()
			case result := <-forwarded:
				cancel()
				if result.err != nil {
					return result.err
				}
				return connect.NewError(connect.CodeInternal, fmt.Errorf("stream ended before retained audio replay completed"))
			}
		}
	}
	pumpErr := h.pumpStreamChunks(streamCtx, stream, registry, ledger, chunkCh, activity, inputDone)

	result := <-forwarded
	terminal, err := result.terminal, result.err
	if err != nil {
		logStreamHandlerFailure(h.deps.Logger, "forward_events", err)
		return err
	}
	select {
	case <-idleExpired:
		// Do not join the input pump: its synchronous Receive may still be
		// blocked. Returning tears down the Connect transport and unblocks it.
		cancel()
		ledger.Fail(session.TerminalReason("input_idle_timeout"))
		if err := registry.PersistContext(ctx, ledger); err != nil {
			logStreamHandlerFailure(h.deps.Logger, "idle_persist", err)
			return connect.NewError(connect.CodeInternal, err)
		}
		return nil
	default:
	}

	var pumpFailure error
	select {
	case pumpFailure = <-pumpErr:
	case <-runDone:
		// A provider can fail before the client sends StreamEnd. In that case
		// Receive may remain blocked for the lifetime of the request. Once the
		// pipeline has stopped, only join the pump if it has already observed
		// the end of input; otherwise close the stream promptly and preserve the
		// unprocessed ledger as a failed, resumable turn.
		select {
		case <-inputDone:
			pumpFailure = <-pumpErr
		default:
			// The pump publishes its result before closing chunkCh. A runner
			// that finished because input failed may win the outer select, so
			// prefer an already-published receive/validation error here.
			pumpFinished := false
			select {
			case pumpFailure = <-pumpErr:
				pumpFinished = true
			default:
			}
			if !pumpFinished {
				runErr := <-runErrCh
				cancel()
				if ctx.Err() != nil || errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
					// Client disconnects can cancel the pipeline before the receive
					// pump has delivered its transport error. Keep that turn resumable.
					if err := registry.PersistContext(context.WithoutCancel(ctx), ledger); err != nil {
						logStreamHandlerFailure(h.deps.Logger, "early_interruption_persist", err)
						return connect.NewError(connect.CodeInternal, err)
					}
					return nil
				}
				if runErr != nil {
					logPostDoneStreamError(h.deps.Logger, startCfg.Language, runErr)
				}
				markStreamPipelineFailure(ledger)
				if err := h.finishStream(ctx, stream, registry, ledger, terminal, nil); err != nil {
					logStreamHandlerFailure(h.deps.Logger, "early_pipeline_finish", err)
					return err
				}
				return nil
			}
		}
	}
	runErr := <-runErrCh
	interrupted := isStreamTransportInterruption(pumpFailure)
	if runErr != nil && !interrupted {
		logPostDoneStreamError(h.deps.Logger, startCfg.Language, runErr)
		markStreamPipelineFailure(ledger)
	}
	if interrupted {
		if err := registry.PersistContext(context.WithoutCancel(ctx), ledger); err != nil {
			logStreamHandlerFailure(h.deps.Logger, "interruption_persist", err)
			return connect.NewError(connect.CodeInternal, err)
		}
		return nil
	}
	if err := h.finishStream(ctx, stream, registry, ledger, terminal, pumpFailure); err != nil {
		logStreamHandlerFailure(h.deps.Logger, "finish", err)
		return err
	}
	return nil
}

func replayCommittedSegments(stream transcribeStreamTransport, ledger *session.Ledger, start *sttv1.StreamStart) error {
	if stream == nil || ledger == nil || start == nil {
		return nil
	}
	for _, committed := range ledger.CommittedSegments() {
		segment := &sttchain.SegmentEvent{
			SegmentID:     committed.ID,
			Generation:    start.GetGeneration(),
			Text:          committed.Text,
			StartSample:   committed.StartSample,
			EndSample:     committed.EndSample,
			StartMs:       committed.StartSample * 1000 / int64(audioformat.CanonicalSampleRate),
			EndMs:         committed.EndSample * 1000 / int64(audioformat.CanonicalSampleRate),
			SpeakerPolicy: committed.SpeakerPolicy,
		}
		if err := stream.Send(protoSegment(segment)); err != nil {
			return err
		}
	}
	return nil
}

func logStreamHandlerFailure(lg logx.Logger, stage string, err error) {
	if lg == nil || err == nil {
		return
	}
	cause := err
	for depth := 0; depth < 8; depth++ {
		unwrapped := errors.Unwrap(cause)
		if unwrapped == nil {
			break
		}
		cause = unwrapped
	}
	class := "other"
	switch {
	case errors.Is(err, session.ErrSegmentConflict):
		class = "segment_conflict"
	case errors.Is(err, session.ErrTerminal):
		class = "terminal_ledger"
	case errors.Is(err, session.ErrResourceExhausted):
		class = "ledger_capacity"
	}
	lg.Printf("event=stt_stream_handler_failure stage=%s code=%s error_class=%s error_type=%T cause_type=%T", stage, connect.CodeOf(err), class, err, cause)
}

func isStreamTransportInterruption(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errStreamTransportInterrupted) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var pumpErr *streamPumpError
	return errors.As(err, &pumpErr) && pumpErr.stage == "receive" && errors.Is(pumpErr.cause, io.ErrUnexpectedEOF)
}

func markStreamPipelineFailure(ledger *session.Ledger) {
	ledger.Fail(session.TerminalReason("pipeline_failed"))
}

// forwardStreamEvents translates chain events and commits segments before sending them.
func (h *connectHandler) forwardStreamEvents(
	ctx context.Context,
	stream transcribeStreamTransport,
	registry *session.Registry,
	ledger *session.Ledger,
	start *sttchain.StreamStart,
	events <-chan sttchain.StreamEvent,
	resumed bool,
) (*sttchain.StreamEvent, error) {
	var terminal *sttchain.StreamEvent
	segmentOrdinal := 0
	for ev := range events {
		if ev.Kind == sttchain.StreamEventError && errors.Is(ev.Error, audioformat.ErrStreamDecode) {
			ledger.Fail(session.TerminalReason("input_decode_failed"))
		}
		if ev.Kind == sttchain.StreamEventDone {
			copy := ev
			terminal = &copy
			continue
		}
		if ev.Kind == sttchain.StreamEventSegment && ev.Segment != nil {
			if err := commitStreamSegment(ctx, registry, ledger, start, ev.Segment, segmentOrdinal, resumed); err != nil {
				return nil, err
			}
			segmentOrdinal++
		}
		if out := protoForEvent(ev); out != nil {
			if err := stream.Send(out); err != nil {
				return nil, err
			}
		}
	}
	return terminal, nil
}

func commitStreamSegment(ctx context.Context, registry *session.Registry, ledger *session.Ledger, start *sttchain.StreamStart, segment *sttchain.SegmentEvent, ordinal int, resumed bool) error {
	if segment.SegmentID == "" {
		segment.SegmentID = fmt.Sprintf("%s:%d:%d:%d", start.SessionID, ordinal, segment.StartSample, segment.EndSample)
	}
	segment.Generation = start.Generation
	committed := session.Segment{ID: segment.SegmentID, Text: segment.Text, StartSample: segment.StartSample, EndSample: segment.EndSample, SpeakerPolicy: segment.SpeakerPolicy}
	if _, err := ledger.Commit(committed); err != nil {
		if resumed && errors.Is(err, session.ErrSegmentConflict) {
			for _, existing := range ledger.CommittedSegments() {
				if existing.ID == committed.ID {
					// Re-decoding the retained prefix can revise VAD boundaries and
					// wording. A stable segment ID still names the first durable
					// commit, so keep its text and sample range across replay.
					segment.Text = existing.Text
					segment.StartSample = existing.StartSample
					segment.EndSample = existing.EndSample
					segment.StartMs = existing.StartSample * 1000 / int64(audioformat.CanonicalSampleRate)
					segment.EndMs = existing.EndSample * 1000 / int64(audioformat.CanonicalSampleRate)
					segment.SpeakerPolicy = existing.SpeakerPolicy
					return nil
				}
			}
		}
		ledger.Fail(session.TerminalReason("commit_conflict"))
		return connect.NewError(connect.CodeInternal, err)
	}
	if err := registry.PersistContext(ctx, ledger); err != nil {
		ledger.Fail(session.TerminalReason("persistence_failed"))
		return connect.NewError(connect.CodeInternal, err)
	}
	return nil
}

// finishStream reconciles both workers, persists the ledger, and emits the
// final acknowledgement and Done event in protocol order.
func (h *connectHandler) finishStream(
	ctx context.Context,
	stream transcribeStreamTransport,
	registry *session.Registry,
	ledger *session.Ledger,
	terminal *sttchain.StreamEvent,
	pumpFailure error,
) error {
	if pumpFailure != nil && !errors.Is(pumpFailure, io.EOF) {
		ledger.Fail(session.TerminalReason("input_failed"))
	}
	state := ledger.Snapshot()
	if state.TerminalReason == session.TerminalNone && state.ReceivedSequence >= 0 {
		if err := ledger.AcknowledgeProcessed(uint64(state.ReceivedSequence)); err != nil {
			ledger.Fail(session.TerminalReason("processed_ack_failed"))
		} else if err := ledger.Complete(); err != nil && !errors.Is(err, session.ErrTerminal) {
			ledger.Fail(session.TerminalIncompleteCoverage)
		}
	}
	state = ledger.Snapshot()
	if err := registry.PersistContext(ctx, ledger); err != nil {
		logStreamHandlerFailure(h.deps.Logger, "finish_persist", err)
		return connect.NewError(connect.CodeInternal, err)
	}
	ack := &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Acknowledgement{Acknowledgement: &sttv1.StreamAcknowledgement{
		ReceivedSequence: state.ReceivedSequence, ProcessedSequence: state.ProcessedSequence,
		DeliveryClass: sttv1.StreamDeliveryClass_STREAM_DELIVERY_CLASS_DURABLE,
	}}}
	if err := stream.Send(ack); err != nil {
		logStreamHandlerFailure(h.deps.Logger, "send_ack", err)
		return err
	}
	if terminal == nil {
		terminal = &sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
	}
	if terminal.Done == nil {
		terminal.Done = &sttchain.DoneEvent{}
	}
	terminal.Done.ProcessedSequence = state.ProcessedSequence
	terminal.Done.TerminalReason = string(state.TerminalReason)
	if out := protoForEvent(*terminal); out != nil {
		if err := stream.Send(out); err != nil {
			logStreamHandlerFailure(h.deps.Logger, "send_done", err)
			return err
		}
	}
	return nil
}

// logStreamPumpFailure records only safe, payload-free stream metadata. The
// underlying error string is intentionally omitted because transport errors
// can include caller-controlled values.
func logStreamPumpFailure(lg logx.Logger, err error, state session.Snapshot) {
	if lg == nil || err == nil {
		return
	}
	stage, causeKind, connectCode, outcome := streamPumpFailureMetadata(err)
	lastEndSample := int64(-1)
	if n := len(state.Replay); n > 0 {
		lastEndSample = state.Replay[n-1].EndSample
	}
	lg.Printf("event=stt_stream_input_failure stage=%s cause_kind=%s connect_code=%s outcome=%s received_sequence=%d processed_sequence=%d retained_bytes=%d received_end_sample=%d committed_segments=%d",
		stage, causeKind, connectCode, outcome, state.ReceivedSequence, state.ProcessedSequence,
		state.RetainedBytes, lastEndSample, len(state.Committed))
}

// openStreamSession validates the handshake and opens its durable session ledger.
func (h *connectHandler) openStreamSession(
	ctx context.Context,
	stream transcribeStreamTransport,
) (*sttv1.StreamStart, *session.Registry, *session.Ledger, bool, error) {
	if h.deps.Chain == nil || h.deps.Selector == nil {
		return nil, nil, nil, false, connect.NewError(connect.CodeUnavailable, fmt.Errorf("stt streaming pipeline not configured"))
	}
	first, err := stream.Receive()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil, nil, false, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("client closed before StreamStart"))
		}
		return nil, nil, nil, false, connect.NewError(connect.CodeInternal, err)
	}
	startPayload, ok := first.GetPayload().(*sttv1.TranscribeStreamRequest_Start)
	if !ok {
		return nil, nil, nil, false, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("first message must be StreamStart"))
	}
	startCfg := startPayload.Start
	if startCfg.ProtocolVersion != 2 || startCfg.SessionId == "" || startCfg.ResumeToken == "" {
		return nil, nil, nil, false, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("TranscribeStream requires protocol_version=2, session_id, and resume_token"))
	}
	if err := validateStreamStartEngine(startCfg, h.deps.Registry); err != nil {
		return nil, nil, nil, false, connect.NewError(connect.CodeInvalidArgument, err)
	}
	registry := h.deps.Sessions
	if registry == nil {
		registry = session.NewRegistry(0)
	}
	ledger, resumed, err := registry.OpenContext(ctx, startCfg.SessionId, startCfg.ResumeToken)
	if err != nil {
		return nil, nil, nil, false, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return startCfg, registry, ledger, resumed, nil
}

// streamStartEngineID returns only the explicit per-session engine selection.
// The rest of StreamConfig remains operator-owned until the proto grows field
// presence for request-level overrides; engine identity is special because a
// transport must never silently replace the engine the caller selected.
func streamStartEngineID(start *sttv1.StreamStart) string {
	if start == nil || start.GetConfig() == nil {
		return ""
	}
	return strings.TrimSpace(start.GetConfig().GetEngineId())
}

func validateStreamStartEngine(start *sttv1.StreamStart, registry *sttengine.Registry) error {
	engineID := streamStartEngineID(start)
	if engineID == "" || registry == nil {
		return nil
	}
	if _, ok := registry.Engine(engineID); !ok {
		return fmt.Errorf("unknown engine_id %q (see ListEngines)", engineID)
	}
	return nil
}

// logPostDoneStreamError records a Segmenter/Selector error that surfaced
// after the events channel closed — i.e. after the stream already emitted a
// Done event. Per the streaming parity contract the Connect handler must NOT
// return a hard Connect error once Done was sent, so the error is logged-and-
// swallowed here rather than returned. context.Canceled is normal teardown
// and is ignored. lg is a required seam (NewConnectHandler guarantees non-nil).
func logPostDoneStreamError(lg logx.Logger, language string, err error) {
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}
	mapped := mapChainError(err)
	lg.Printf("event=stt_stream_post_done_error language=%q code=%s err=%v",
		language, connect.CodeOf(mapped), err)
}
