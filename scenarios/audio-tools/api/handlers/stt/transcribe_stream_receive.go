package stt

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/stt/session"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// pumpStreamChunks receives and validates client audio while the Segmenter runs.
func (h *connectHandler) pumpStreamChunks(
	ctx context.Context,
	stream transcribeStreamTransport,
	registry *session.Registry,
	ledger *session.Ledger,
	chunks chan<- sttchain.AudioChunk,
	activity chan<- struct{},
	inputDone chan<- struct{},
) <-chan error {
	pumpErr := make(chan error, 1)
	go func() {
		defer close(chunks)
		for {
			// Receive is tied to the Connect transport rather than ctx. Avoid
			// starting another receive after cancellation; returning from the
			// handler closes the transport and releases an in-flight Receive.
			select {
			case <-ctx.Done():
				select {
				case pumpErr <- ctx.Err():
				default:
				}
				return
			default:
			}
			err := h.receiveStreamChunk(ctx, stream, registry, ledger, chunks, activity)
			if err == errStreamContinue {
				continue
			}
			if err == nil {
				select {
				case inputDone <- struct{}{}:
				default:
				}
			}
			if err != nil && !errors.Is(err, io.EOF) {
				logStreamPumpFailure(h.deps.Logger, err, ledger.Snapshot())
			}
			select {
			case pumpErr <- err:
			case <-ctx.Done():
			}
			return
		}
	}()
	return pumpErr
}

var errStreamContinue = errors.New("continue stream receive")

// streamPumpError identifies the bounded operation that rejected inbound
// stream data. Its cause is for errors.Is/errors.As only; logs must use the
// safe metadata helper below and must never print Error() (which can contain
// caller-controlled transport details).
type streamPumpError struct {
	stage string
	cause error
}

func (e *streamPumpError) Error() string { return "stream input failed" }
func (e *streamPumpError) Unwrap() error { return e.cause }

func streamPumpFailureMetadata(err error) (stage, causeKind, connectCode, outcome string) {
	var pumpErr *streamPumpError
	if errors.As(err, &pumpErr) {
		stage = pumpErr.stage
		err = pumpErr.cause
	} else {
		stage = "unknown"
	}
	if err == nil {
		return stage, "none", "none", "none"
	}
	causeKind = "other"
	switch {
	case errors.Is(err, context.Canceled):
		causeKind, outcome = "context_canceled", "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		causeKind, outcome = "context_deadline", "deadline_exceeded"
	case errors.Is(err, io.EOF):
		causeKind, outcome = "io_eof", "eof"
	default:
		outcome = "error"
	}
	connectCode = connect.CodeOf(err).String()
	if _, ok := err.(*connect.Error); ok {
		causeKind = "connect_error"
	}
	return stage, causeKind, connectCode, outcome
}

func (h *connectHandler) receiveStreamChunk(
	ctx context.Context,
	stream transcribeStreamTransport,
	registry *session.Registry,
	ledger *session.Ledger,
	chunks chan<- sttchain.AudioChunk,
	activity chan<- struct{},
) error {
	msg, err := stream.Receive()
	if errors.Is(err, errStreamTransportInterrupted) {
		return &streamPumpError{stage: "receive", cause: err}
	}
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return &streamPumpError{stage: "receive", cause: err}
	}
	switch payload := msg.GetPayload().(type) {
	case *sttv1.TranscribeStreamRequest_AudioChunk:
		if payload.AudioChunk == nil {
			return &streamPumpError{stage: "validate_chunk", cause: errors.New("empty audio chunk")}
		}
		chunk := payload.AudioChunk
		select {
		case activity <- struct{}{}:
		default:
		}
		if len(chunk.Sha256) != 0 {
			digest := sha256.Sum256(chunk.Audio)
			if !bytes.Equal(digest[:], chunk.Sha256) {
				return &streamPumpError{stage: "validate_digest", cause: errors.New("digest mismatch")}
			}
		}
		result, receiveErr := ledger.Receive(session.Chunk{Sequence: chunk.Sequence, StartSample: chunk.StartSample, EndSample: chunk.EndSample, Audio: chunk.Audio})
		if receiveErr != nil {
			ledger.Fail(session.TerminalReason("receive_failed"))
			return &streamPumpError{stage: "ledger_receive", cause: receiveErr}
		}
		if result == session.ReceivedDuplicate {
			return errStreamContinue
		}
		if result != session.ReceivedDuplicate {
			if err := registry.PersistContext(ctx, ledger); err != nil {
				ledger.Fail(session.TerminalReason("persistence_failed"))
				return &streamPumpError{stage: "persist_chunk", cause: err}
			}
		}
		select {
		case chunks <- sttchain.AudioChunk{Audio: chunk.Audio, Sequence: chunk.Sequence, StartSample: chunk.StartSample, EndSample: chunk.EndSample, Digest: chunk.Sha256}:
			return errStreamContinue
		case <-ctx.Done():
			return &streamPumpError{stage: "forward_chunk", cause: ctx.Err()}
		}
	case *sttv1.TranscribeStreamRequest_End:
		return nil
	case *sttv1.TranscribeStreamRequest_Start:
		return &streamPumpError{stage: "validate_message", cause: errors.New("duplicate start")}
	default:
		return errStreamContinue
	}
}
