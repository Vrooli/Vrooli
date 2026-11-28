package stt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"audio-tools/cli/domains/internal/providerlabels"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	"google.golang.org/protobuf/encoding/protojson"
)

func (h *handlers) transcribePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustom(
		func(ctx cliapp.OperationContext) (*sttv1.TranscribeResponse, error) {
			path := ctx.Flag("file")
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", path, err)
			}
			request := connect.NewRequest(&sttv1.TranscribeRequest{
				Audio: data, Format: audioFormatFromFlag(ctx.Flag("format")), Language: ctx.Flag("language"),
			})
			if modelID := strings.TrimSpace(ctx.Flag("model")); modelID != "" {
				request.Header().Set("X-Audio-Model-ID", modelID)
			}
			resp, err := h.client.Transcribe(context.Background(), request)
			if err != nil {
				return nil, cliapp.WrapAPIError("transcribe", err, nil)
			}
			if resp == nil || resp.Msg == nil {
				return nil, fmt.Errorf("server returned no transcription")
			}
			return resp.Msg, nil
		},
		func(ctx cliapp.RunContext, resp *sttv1.TranscribeResponse) error {
			return cliapp.RenderMutationReport(ctx.Stdout(), cliapp.MutationReport{Result: []string{
				fmt.Sprintf("Transcribed via %s/%s (%.0fms): %s",
					providerlabels.TierLabel(resp.GetProviderTier()), resp.GetProviderId(), resp.GetLatencyMs(), resp.GetText()),
			}})
		},
	)
}

// Compatibility wrappers keep direct handler-level callers on the same
// primitive path as manifest dispatch.
func (h *handlers) transcribe(ctx cliapp.RunContext) error { return h.transcribePrimitive().Run(ctx) }

// transcribeStream feeds the audio file chunk-by-chunk through the
// bidi-streaming TranscribeStream RPC and prints one line per emitted event.
// With --json, each line is the canonical proto JSON event (JSON Lines).
// Exits non-zero on a chain error.
func (h *handlers) transcribeStream(ctx cliapp.RunContext) error {
	data, err := readStreamAudio(ctx)
	if err != nil {
		return err
	}
	chunkBytes, err := streamChunkBytes(ctx.Flag("chunk-bytes"))
	if err != nil {
		return err
	}

	streamClient := h.stream
	if streamClient == nil {
		streamClient = h.client
	}
	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := streamClient.TranscribeStream(streamCtx)
	if err := startTranscribeStream(stream, ctx.Flag("language"), ctx.Flag("engine")); err != nil {
		return err
	}

	// Push chunks then end. Errors break out and let the receive loop
	// drain whatever events the server has already emitted before
	// returning.
	sendErrCh := make(chan error, 1)
	go func() {
		sendErrCh <- sendTranscribeAudio(stream, data, chunkBytes)
	}()

	var lastErr error
	for {
		ev, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			cancel()
			sendErr := <-sendErrCh
			return errors.Join(cliapp.WrapAPIError("transcribe-stream receive", err, nil), sendErr)
		}
		if err := writeTranscribeStreamEvent(ctx, ev); err != nil {
			return err
		}
		if eventErr := renderTranscribeStreamEvent(ctx, ev); eventErr != nil {
			lastErr = eventErr
		}
	}
	if err := <-sendErrCh; err != nil {
		return err
	}
	return lastErr
}

func readStreamAudio(ctx cliapp.RunContext) ([]byte, error) {
	path := ctx.Flag("file")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return data, nil
}

func streamChunkBytes(raw string) (int, error) {
	if raw == "" {
		return 32 * 1024, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("--chunk-bytes must be a positive integer: %q", raw)
	}
	return value, nil
}

func startTranscribeStream(stream *connect.BidiStreamForClient[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent], language, engineID string) error {
	sessionID, err := streamSessionIdentity()
	if err != nil {
		return err
	}
	resumeToken, err := streamSessionIdentity()
	if err != nil {
		return err
	}
	start := &sttv1.StreamStart{Language: language, ProtocolVersion: 2, SessionId: sessionID, ResumeToken: resumeToken}
	if engineID = strings.TrimSpace(engineID); engineID != "" {
		start.Config = &sttv1.StreamConfig{EngineId: engineID}
	}
	request := &sttv1.TranscribeStreamRequest{
		Payload: &sttv1.TranscribeStreamRequest_Start{
			Start: start,
		},
	}
	if err := stream.Send(request); err != nil {
		return cliapp.WrapAPIError("transcribe-stream start", err, nil)
	}
	return nil
}

func sendTranscribeAudio(stream *connect.BidiStreamForClient[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent], data []byte, chunkBytes int) error {
	var sequence uint64
	for offset := 0; offset < len(data); offset += chunkBytes {
		end := min(offset+chunkBytes, len(data))
		chunk := data[offset:end]
		digest := sha256.Sum256(chunk)
		request := &sttv1.TranscribeStreamRequest{
			Payload: &sttv1.TranscribeStreamRequest_AudioChunk{AudioChunk: &sttv1.StreamAudioChunk{
				Audio: chunk, Sequence: sequence, StartSample: int64(offset / 2), EndSample: int64(end / 2), Sha256: digest[:],
			}},
		}
		if err := stream.Send(request); err != nil {
			return closeTranscribeRequest(stream, fmt.Errorf("transcribe-stream send audio chunk %d: %w", sequence, err))
		}
		sequence++
	}
	if err := stream.Send(&sttv1.TranscribeStreamRequest{
		Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}},
	}); err != nil {
		return closeTranscribeRequest(stream, fmt.Errorf("transcribe-stream send end: %w", err))
	}
	return closeTranscribeRequest(stream, nil)
}

func closeTranscribeRequest(stream *connect.BidiStreamForClient[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent], sendErr error) error {
	if err := stream.CloseRequest(); err != nil {
		sendErr = errors.Join(sendErr, fmt.Errorf("transcribe-stream close request: %w", err))
	}
	return sendErr
}

func writeTranscribeStreamEvent(ctx cliapp.RunContext, event *sttv1.TranscribeStreamEvent) error {
	if !ctx.JSON() {
		return nil
	}
	body, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal transcribe-stream event: %w", err)
	}
	_, err = ctx.Stdout().Write(append(body, '\n'))
	return err
}

func renderTranscribeStreamEvent(ctx cliapp.RunContext, event *sttv1.TranscribeStreamEvent) error {
	switch e := event.GetEvent().(type) {
	case *sttv1.TranscribeStreamEvent_Partial:
		if !ctx.JSON() {
			fmt.Fprintf(ctx.Stdout(), "partial: %s\n", e.Partial.Text)
		}
	case *sttv1.TranscribeStreamEvent_Segment:
		if !ctx.JSON() {
			fmt.Fprintf(ctx.Stdout(), "segment [%s/%s %.0fms]: %s\n",
				providerlabels.TierLabel(e.Segment.ProviderTier), e.Segment.ModelId, e.Segment.LatencyMs, e.Segment.Text)
		}
	case *sttv1.TranscribeStreamEvent_WakeWord:
		if !ctx.JSON() {
			fmt.Fprintf(ctx.Stdout(), "wake-word: score=%.3f sample=%s\n", e.WakeWord.Score, e.WakeWord.SampleId)
		}
	case *sttv1.TranscribeStreamEvent_SpeakerRejection:
		if !ctx.JSON() {
			fmt.Fprintf(ctx.Stdout(), "speaker-rejection: %s (fallback=%v)\n", e.SpeakerRejection.Reason, e.SpeakerRejection.FallbackUsed)
		}
	case *sttv1.TranscribeStreamEvent_Error:
		if !ctx.JSON() {
			fmt.Fprintf(ctx.Stderr(), "error: %s — %s\n", e.Error.Code, e.Error.Message)
		}
		return fmt.Errorf("%s: %s", e.Error.Code, e.Error.Message)
	case *sttv1.TranscribeStreamEvent_Done:
		if !ctx.JSON() {
			fallback := ""
			if e.Done.FellBackToUnary {
				fallback = " (streaming degraded: buffered mode)"
			}
			fmt.Fprintf(ctx.Stdout(), "done [%s/%s %.0fms%s]: %s\n",
				providerlabels.TierLabel(e.Done.ProviderTier), e.Done.ModelId, e.Done.LatencyMs, fallback, e.Done.FinalText)
		}
		if reason := strings.TrimSpace(e.Done.GetTerminalReason()); reason != "" && reason != "none" && reason != "completed" {
			return fmt.Errorf("transcribe-stream terminated: %s", reason)
		}
	}
	return nil
}

func streamSessionIdentity() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate stream identity: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}

// formats prints the STT ingress capability matrix: the input codecs the
// API accepts, whether the local ffmpeg decode backend is present, and the
// fixed canonical PCM target. Human-friendly output per
// feedback_cli_default_human_output.
