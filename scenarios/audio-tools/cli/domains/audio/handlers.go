package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/cli-core/cliapp"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	audioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio/audio_v1connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client audioconnect.AudioProcessingServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: audioconnect.NewAudioProcessingServiceClient(httpClient, baseURL),
	}
}

// transcode re-encodes the input. The manifest supplies defaults for
// --output-format (wav); sample rate / channels / bitrate default to 0, which
// the server interprets as passthrough or format default.
func (h *handlers) transcode(ctx cliapp.RunContext) error {
	data, err := readAudioInput(ctx, "input")
	if err != nil {
		return err
	}
	sampleRate, err := int32Flag(ctx, "sample-rate", 0, 192000)
	if err != nil {
		return err
	}
	channels, err := int32Flag(ctx, "channels", 0, 8)
	if err != nil {
		return err
	}
	bitrate, err := int32Flag(ctx, "bitrate", 0, 512000)
	if err != nil {
		return err
	}
	resp, err := h.client.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{
		Audio:        data,
		InputFormat:  audioFormatFromString(ctx.Flag("input-format")),
		OutputFormat: audioFormatFromString(ctx.Flag("output-format")),
		SampleRate:   sampleRate,
		Channels:     channels,
		Bitrate:      bitrate,
	}))
	if err != nil {
		return cliapp.WrapAPIError("transcode", err, nil)
	}
	return writeAudioReport(ctx, "transcode", resp.Msg.Audio, ctx.Flag("output"))
}

func (h *handlers) trim(ctx cliapp.RunContext) error {
	data, err := readAudioInput(ctx, "input")
	if err != nil {
		return err
	}
	start, err := floatFlag(ctx, "start-seconds", 0, 86400)
	if err != nil {
		return err
	}
	end, err := floatFlag(ctx, "end-seconds", 0, 86400)
	if err != nil {
		return err
	}
	resp, err := h.client.Trim(context.Background(), connect.NewRequest(&audiov1.TrimRequest{
		Audio:        data,
		Format:       audioFormatFromString(ctx.Flag("format")),
		StartSeconds: start,
		EndSeconds:   end,
	}))
	if err != nil {
		return cliapp.WrapAPIError("trim", err, nil)
	}
	return writeAudioReport(ctx, "trim", resp.Msg.Audio, ctx.Flag("output"))
}

func (h *handlers) merge(ctx cliapp.RunContext) error {
	paths := nonEmpty(ctx.FlagValues("input"))
	if len(paths) == 0 {
		return fmt.Errorf("merge: --input is required (repeat it for every source)")
	}
	format := audioFormatFromString(ctx.Flag("format"))
	sources := make([]*audiov1.MergeSource, 0, len(paths))
	for _, path := range paths {
		data, err := readFile(path)
		if err != nil {
			return err
		}
		sources = append(sources, &audiov1.MergeSource{Audio: data, Format: format})
	}
	crossfade, err := floatFlag(ctx, "crossfade-seconds", 0, 60)
	if err != nil {
		return err
	}
	resp, err := h.client.Merge(context.Background(), connect.NewRequest(&audiov1.MergeRequest{
		Sources:          sources,
		OutputFormat:     audioFormatFromString(ctx.Flag("output-format")),
		CrossfadeSeconds: crossfade,
	}))
	if err != nil {
		return cliapp.WrapAPIError("merge", err, nil)
	}
	return writeAudioReport(ctx, "merge", resp.Msg.Audio, ctx.Flag("output"))
}

// split writes one file per chunk into --output (a directory). Chunk names are
// chunk-000.<ext>, chunk-001.<ext>, … using the requested output format.
func (h *handlers) split(ctx cliapp.RunContext) error {
	data, err := readAudioInput(ctx, "input")
	if err != nil {
		return err
	}
	chunkSeconds, err := floatFlag(ctx, "chunk-seconds", 0, 3600)
	if err != nil {
		return err
	}
	boundaries, err := floatListFlag(ctx, "boundaries-seconds")
	if err != nil {
		return err
	}
	if chunkSeconds <= 0 && len(boundaries) == 0 {
		return fmt.Errorf("split: set --chunk-seconds or --boundaries-seconds")
	}
	outDir := ctx.Flag("output")
	if outDir == "" {
		return fmt.Errorf("split: --output directory is required")
	}
	resp, err := h.client.Split(context.Background(), connect.NewRequest(&audiov1.SplitRequest{
		Audio:             data,
		Format:            audioFormatFromString(ctx.Flag("format")),
		ChunkSeconds:      chunkSeconds,
		BoundariesSeconds: boundaries,
		OutputFormat:      audioFormatFromString(ctx.Flag("output-format")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("split", err, nil)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %s: %w", outDir, err)
	}
	ext := ctx.Flag("output-format")
	if ext == "" {
		ext = "wav"
	}
	written := make([]string, 0, len(resp.Msg.Chunks))
	for i, chunk := range resp.Msg.Chunks {
		name := fmt.Sprintf("chunk-%03d.%s", i, ext)
		path := filepath.Join(outDir, name)
		if err := writeFile(path, chunk.Audio); err != nil {
			return err
		}
		written = append(written, fmt.Sprintf("%s (start %.3fs, duration %.3fs)", path, chunk.StartSeconds, chunk.DurationSeconds))
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Wrote %d chunk(s) to %s.", len(written), outDir)},
		Changes: written,
	})
}

func (h *handlers) fade(ctx cliapp.RunContext) error {
	data, err := readAudioInput(ctx, "input")
	if err != nil {
		return err
	}
	fadeIn, err := floatFlag(ctx, "fade-in-seconds", 0, 3600)
	if err != nil {
		return err
	}
	fadeOut, err := floatFlag(ctx, "fade-out-seconds", 0, 3600)
	if err != nil {
		return err
	}
	resp, err := h.client.Fade(context.Background(), connect.NewRequest(&audiov1.FadeRequest{
		Audio:          data,
		Format:         audioFormatFromString(ctx.Flag("format")),
		FadeInSeconds:  fadeIn,
		FadeOutSeconds: fadeOut,
		OutputFormat:   audioFormatFromString(ctx.Flag("output-format")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("fade", err, nil)
	}
	return writeAudioReport(ctx, "fade", resp.Msg.Audio, ctx.Flag("output"))
}

func (h *handlers) volume(ctx cliapp.RunContext) error {
	data, err := readAudioInput(ctx, "input")
	if err != nil {
		return err
	}
	gain, err := floatFlag(ctx, "gain-db", -60, 24)
	if err != nil {
		return err
	}
	resp, err := h.client.Volume(context.Background(), connect.NewRequest(&audiov1.VolumeRequest{
		Audio:        data,
		Format:       audioFormatFromString(ctx.Flag("format")),
		GainDb:       gain,
		OutputFormat: audioFormatFromString(ctx.Flag("output-format")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("volume", err, nil)
	}
	return writeAudioReport(ctx, "volume", resp.Msg.Audio, ctx.Flag("output"))
}

func (h *handlers) normalize(ctx cliapp.RunContext) error {
	data, err := readAudioInput(ctx, "input")
	if err != nil {
		return err
	}
	target, err := floatFlag(ctx, "target-lufs", -40, -5)
	if err != nil {
		return err
	}
	resp, err := h.client.Normalize(context.Background(), connect.NewRequest(&audiov1.NormalizeRequest{
		Audio:        data,
		Format:       audioFormatFromString(ctx.Flag("format")),
		Method:       normalizationMethodFromString(ctx.Flag("method")),
		TargetLufs:   target,
		OutputFormat: audioFormatFromString(ctx.Flag("output-format")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("normalize", err, nil)
	}
	return writeAudioReport(ctx, "normalize", resp.Msg.Audio, ctx.Flag("output"))
}

// extractMetadata writes readable JSON to --output. Use "-" to stream it to
// stdout instead of a file.
func (h *handlers) extractMetadata(ctx cliapp.RunContext) error {
	data, err := readAudioInput(ctx, "input")
	if err != nil {
		return err
	}
	resp, err := h.client.ExtractMetadata(context.Background(), connect.NewRequest(&audiov1.ExtractMetadataRequest{
		Audio:  data,
		Format: audioFormatFromString(ctx.Flag("format")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("extract-metadata", err, nil)
	}
	body, err := protojson.MarshalOptions{Indent: "  ", UseProtoNames: true, EmitUnpopulated: true}.Marshal(resp.Msg)
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}
	body = append(body, '\n')
	out := ctx.Flag("output")
	if out == "-" {
		_, err := ctx.Stdout().Write(body)
		return err
	}
	if out == "" {
		return fmt.Errorf("extract-metadata: --output is required")
	}
	if err := writeFile(out, body); err != nil {
		return err
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Wrote metadata to %s.", out)},
	})
}

// -----------------------------------------------------------------------------
// Shared input / output / conversion helpers.
// -----------------------------------------------------------------------------

func readAudioInput(ctx cliapp.RunContext, flagName string) ([]byte, error) {
	path := strings.TrimSpace(ctx.Flag(flagName))
	if path == "" {
		return nil, fmt.Errorf("--%s is required", flagName)
	}
	return readFile(path)
}

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("read %s: file is empty", path)
	}
	return data, nil
}

func writeFile(path string, data []byte) error {
	if path == "" {
		return fmt.Errorf("output path is required")
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeAudioReport(ctx cliapp.RunContext, operation string, data []byte, path string) error {
	if path == "" {
		return fmt.Errorf("%s: --output is required", operation)
	}
	if err := writeFile(path, data); err != nil {
		return err
	}
	if ctx.JSON() {
		return json.NewEncoder(ctx.Stdout()).Encode(map[string]any{
			"operation": operation,
			"path":      path,
			"bytes":     len(data),
		})
	}
	return ctx.RenderMutation(cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Wrote %d bytes to %s.", len(data), path)},
	})
}

func audioFormatFromString(s string) commonv1.AudioFormat {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto", "detect":
		return commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED
	case "wav":
		return commonv1.AudioFormat_AUDIO_FORMAT_WAV
	case "mp3":
		return commonv1.AudioFormat_AUDIO_FORMAT_MP3
	case "flac":
		return commonv1.AudioFormat_AUDIO_FORMAT_FLAC
	case "ogg":
		return commonv1.AudioFormat_AUDIO_FORMAT_OGG
	case "webm":
		return commonv1.AudioFormat_AUDIO_FORMAT_WEBM
	case "opus":
		return commonv1.AudioFormat_AUDIO_FORMAT_OPUS
	case "aac":
		return commonv1.AudioFormat_AUDIO_FORMAT_AAC
	case "pcm_s16le", "pcm-s16le":
		return commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE
	default:
		return commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED
	}
}

func normalizationMethodFromString(s string) audiov1.NormalizationMethod {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ebu_r128", "ebu-r128", "ebu", "r128":
		return audiov1.NormalizationMethod_NORMALIZATION_METHOD_EBU_R128
	case "rms":
		return audiov1.NormalizationMethod_NORMALIZATION_METHOD_RMS
	case "peak":
		return audiov1.NormalizationMethod_NORMALIZATION_METHOD_PEAK
	default:
		return audiov1.NormalizationMethod_NORMALIZATION_METHOD_UNSPECIFIED
	}
}

func floatFlag(ctx cliapp.RunContext, name string, min, max float64) (float64, error) {
	raw := strings.TrimSpace(ctx.Flag(name))
	if raw == "" {
		raw = "0"
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("--%s: %q is not a number: %w", name, raw, err)
	}
	if v < min || v > max {
		return 0, fmt.Errorf("--%s: %g is outside [%g, %g]", name, v, min, max)
	}
	return v, nil
}

func int32Flag(ctx cliapp.RunContext, name string, min, max int32) (int32, error) {
	raw := strings.TrimSpace(ctx.Flag(name))
	if raw == "" {
		raw = "0"
	}
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("--%s: %q is not an integer: %w", name, raw, err)
	}
	n := int32(v)
	if n < min || n > max {
		return 0, fmt.Errorf("--%s: %d is outside [%d, %d]", name, n, min, max)
	}
	return n, nil
}

func floatListFlag(ctx cliapp.RunContext, name string) ([]float64, error) {
	values := ctx.FlagValues(name)
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strings.Split(value, ",")...)
	}
	out := make([]float64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		v, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return nil, fmt.Errorf("--%s: %q is not a number: %w", name, part, err)
		}
		if v < 0 {
			return nil, fmt.Errorf("--%s: boundaries must be >= 0, got %g", name, v)
		}
		out = append(out, v)
	}
	return out, nil
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}
