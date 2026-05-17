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

// streamConfigGet prints the resolved StreamConfig, including the five
// streaming-pipeline operator levers documented in
// docs/reference/configuration.md. The display format is
// human-friendly per feedback_cli_default_human_output.
func (h *handlers) streamConfigGet(ctx cliapp.RunContext) error {
	resp, err := h.client.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get-stream-config", err, nil)
	}
	cfg := resp.Msg.GetConfig()
	if cfg == nil {
		return fmt.Errorf("server returned no stream config")
	}
	fmt.Fprintf(ctx.Stdout(), "Streaming STT pipeline:\n")
	fmt.Fprintf(ctx.Stdout(), "  streaming_mode       = %s\n", orDefault(cfg.GetStreamingMode(), "auto"))
	fmt.Fprintf(ctx.Stdout(), "  strategy_preference  = %s\n", orDefault(cfg.GetStrategyPreference(), "auto"))
	fmt.Fprintf(ctx.Stdout(), "  vad_silence_ms       = %d\n", intOrDefault(cfg.GetVadSilenceMs(), 700))
	fmt.Fprintf(ctx.Stdout(), "  overlap_window_ms    = %d\n", intOrDefault(cfg.GetOverlapWindowMs(), 2000))
	fmt.Fprintf(ctx.Stdout(), "  overlap_commit_runs  = %d\n", intOrDefault(cfg.GetOverlapCommitRuns(), 2))
	fmt.Fprintf(ctx.Stdout(), "Legacy partial-window fields (still used by browser WS):\n")
	fmt.Fprintf(ctx.Stdout(), "  flush_interval_ms    = %d\n", cfg.GetFlushIntervalMs())
	fmt.Fprintf(ctx.Stdout(), "  min_delta_bytes      = %d\n", cfg.GetMinDeltaBytes())
	fmt.Fprintf(ctx.Stdout(), "  overlap_bytes        = %d\n", cfg.GetOverlapBytes())
	fmt.Fprintf(ctx.Stdout(), "  segment_silence_ms   = %d\n", cfg.GetSegmentSilenceMs())
	return nil
}

// streamConfigSet mutates the persisted StreamConfig. Each provided
// flag is sent with its corresponding has_ field set; omitted flags
// leave the persisted value untouched. The server enforces the
// documented ranges and returns InvalidArgument on a forbidden value.
func (h *handlers) streamConfigSet(ctx cliapp.RunContext) error {
	req := &sttv1.UpdateStreamConfigRequest{}
	if v := ctx.Flag("streaming-mode"); v != "" {
		req.StreamingMode = v
		req.HasStreamingMode = true
	}
	if v := ctx.Flag("strategy-preference"); v != "" {
		req.StrategyPreference = v
		req.HasStrategyPreference = true
	}
	if v := ctx.Flag("vad-silence-ms"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("--vad-silence-ms must be integer: %q", v)
		}
		req.VadSilenceMs = int32(n)
		req.HasVadSilenceMs = true
	}
	if v := ctx.Flag("overlap-window-ms"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("--overlap-window-ms must be integer: %q", v)
		}
		req.OverlapWindowMs = int32(n)
		req.HasOverlapWindowMs = true
	}
	if v := ctx.Flag("overlap-commit-runs"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("--overlap-commit-runs must be integer: %q", v)
		}
		req.OverlapCommitRuns = int32(n)
		req.HasOverlapCommitRuns = true
	}
	resp, err := h.client.UpdateStreamConfig(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("update-stream-config", err, nil)
	}
	cfg := resp.Msg.GetConfig()
	fmt.Fprintf(ctx.Stdout(), "Updated. Resolved streaming STT pipeline:\n")
	fmt.Fprintf(ctx.Stdout(), "  streaming_mode       = %s\n", orDefault(cfg.GetStreamingMode(), "auto"))
	fmt.Fprintf(ctx.Stdout(), "  strategy_preference  = %s\n", orDefault(cfg.GetStrategyPreference(), "auto"))
	fmt.Fprintf(ctx.Stdout(), "  vad_silence_ms       = %d\n", intOrDefault(cfg.GetVadSilenceMs(), 700))
	fmt.Fprintf(ctx.Stdout(), "  overlap_window_ms    = %d\n", intOrDefault(cfg.GetOverlapWindowMs(), 2000))
	fmt.Fprintf(ctx.Stdout(), "  overlap_commit_runs  = %d\n", intOrDefault(cfg.GetOverlapCommitRuns(), 2))
	return nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func intOrDefault(v, def int32) int32 {
	if v == 0 {
		return def
	}
	return v
}
