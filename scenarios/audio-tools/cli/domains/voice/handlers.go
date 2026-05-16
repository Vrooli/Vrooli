package voice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client sttconnect.STTServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: sttconnect.NewSTTServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) transcribe(ctx cliapp.RunContext) error {
	path := ctx.Flag("file")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	format := ctx.Flag("format")
	if format == "" {
		format = "wav"
	}
	req := connect.NewRequest(&sttv1.TranscribeRequest{
		Audio:    data,
		Format:   format,
		Language: ctx.Flag("language"),
	})
	resp, err := h.client.Transcribe(context.Background(), req)
	if err != nil {
		return cliapp.WrapAPIError("transcribe", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no transcription")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Transcribed via %s/%s (%.0fms): %s",
				resp.Msg.ProviderTier, resp.Msg.ProviderId, resp.Msg.LatencyMs, resp.Msg.Text),
		},
	})
}

// transcribeStream feeds the audio file chunk-by-chunk through the
// bidi-streaming TranscribeStream RPC and prints one human-readable
// line per emitted event (partial, segment, wake_word, speaker_rejection,
// error, done). Exits non-zero on a chain error.
func (h *handlers) transcribeStream(ctx cliapp.RunContext) error {
	path := ctx.Flag("file")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	chunkBytes := 32 * 1024
	if cb := ctx.Flag("chunk-bytes"); cb != "" {
		v, err := strconv.Atoi(cb)
		if err != nil || v <= 0 {
			return fmt.Errorf("--chunk-bytes must be a positive integer: %q", cb)
		}
		chunkBytes = v
	}

	stream := h.client.TranscribeStream(context.Background())
	if err := stream.Send(&sttv1.TranscribeStreamRequest{
		Payload: &sttv1.TranscribeStreamRequest_Start{
			Start: &sttv1.StreamStart{Language: ctx.Flag("language")},
		},
	}); err != nil {
		return cliapp.WrapAPIError("transcribe-stream start", err, nil)
	}

	// Push chunks then end. Errors break out and let the receive loop
	// drain whatever events the server has already emitted before
	// returning.
	go func() {
		for off := 0; off < len(data); off += chunkBytes {
			end := off + chunkBytes
			if end > len(data) {
				end = len(data)
			}
			if err := stream.Send(&sttv1.TranscribeStreamRequest{
				Payload: &sttv1.TranscribeStreamRequest_AudioChunk{AudioChunk: data[off:end]},
			}); err != nil {
				return
			}
		}
		_ = stream.Send(&sttv1.TranscribeStreamRequest{
			Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}},
		})
		_ = stream.CloseRequest()
	}()

	var lastErr error
	for {
		ev, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return cliapp.WrapAPIError("transcribe-stream receive", err, nil)
		}
		switch e := ev.GetEvent().(type) {
		case *sttv1.TranscribeStreamEvent_Partial:
			fmt.Fprintf(ctx.Stdout(), "partial: %s\n", e.Partial.Text)
		case *sttv1.TranscribeStreamEvent_Segment:
			fmt.Fprintf(ctx.Stdout(), "segment [%s/%s %.0fms]: %s\n",
				e.Segment.ProviderTier, e.Segment.ModelId, e.Segment.LatencyMs, e.Segment.Text)
		case *sttv1.TranscribeStreamEvent_WakeWord:
			fmt.Fprintf(ctx.Stdout(), "wake-word: score=%.3f sample=%s\n", e.WakeWord.Score, e.WakeWord.SampleId)
		case *sttv1.TranscribeStreamEvent_SpeakerRejection:
			fmt.Fprintf(ctx.Stdout(), "speaker-rejection: %s (fallback=%v)\n", e.SpeakerRejection.Reason, e.SpeakerRejection.FallbackUsed)
		case *sttv1.TranscribeStreamEvent_Error:
			fmt.Fprintf(ctx.Stderr(), "error: %s — %s\n", e.Error.Code, e.Error.Message)
			lastErr = fmt.Errorf("%s: %s", e.Error.Code, e.Error.Message)
		case *sttv1.TranscribeStreamEvent_Done:
			tier := e.Done.ProviderTier
			if tier == "" {
				tier = "unknown"
			}
			fallback := ""
			if e.Done.FellBackToUnary {
				fallback = " (fellback=unary)"
			}
			fmt.Fprintf(ctx.Stdout(), "done [%s/%s %.0fms%s]: %s\n",
				tier, e.Done.ModelId, e.Done.LatencyMs, fallback, e.Done.FinalText)
		}
	}
	return lastErr
}
