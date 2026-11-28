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
	"google.golang.org/protobuf/proto"

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
	return &handlers{core: core, client: audioconnect.NewAudioProcessingServiceClient(httpClient, baseURL)}
}

// The six single-output transforms share one renderer-separated action: the
// service response is returned once, while file writing and both output modes
// remain byte-for-byte compatible with the previous handler.
func audioOutputPrimitive[Req any, Raw any, Resp proto.Message](operation string,
	build func(cliapp.OperationContext) (*Req, error),
	invoke func(context.Context, *connect.Request[Req]) (*connect.Response[Raw], error),
	audio func(Resp) []byte,
) cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (Resp, error) {
		var zero Resp
		req, err := build(ctx)
		if err != nil {
			return zero, err
		}
		resp, err := invoke(context.Background(), connect.NewRequest(req))
		if err != nil {
			return zero, cliapp.WrapAPIError(operation, err, nil)
		}
		message, ok := any(resp.Msg).(Resp)
		if !ok {
			return zero, fmt.Errorf("%s: unexpected response type %T", operation, resp.Msg)
		}
		data, path := audio(message), ctx.Flag("output")
		if path == "" {
			return zero, fmt.Errorf("%s: --output is required", operation)
		}
		if err := writeFile(path, data); err != nil {
			return zero, err
		}
		return message, nil
	}, func(ctx cliapp.RunContext, resp Resp) error {
		return ctx.RenderMutation(cliapp.MutationReport{Result: []string{fmt.Sprintf("Wrote %d bytes to %s.", len(audio(resp)), ctx.Flag("output"))}})
	}, func(ctx cliapp.RunContext, resp Resp) error {
		return json.NewEncoder(ctx.Stdout()).Encode(map[string]any{"operation": operation, "path": ctx.Flag("output"), "bytes": len(audio(resp))})
	})
}

func (h *handlers) transcodePrimitive() cliapp.PrimitiveHandler {
	return audioOutputPrimitive("transcode", func(ctx cliapp.OperationContext) (*audiov1.TranscodeRequest, error) {
		data, err := readAudioInput(ctx, "input")
		if err != nil {
			return nil, err
		}
		sample, err := int32Flag(ctx, "sample-rate", 0, 192000)
		if err != nil {
			return nil, err
		}
		channels, err := int32Flag(ctx, "channels", 0, 8)
		if err != nil {
			return nil, err
		}
		bitrate, err := int32Flag(ctx, "bitrate", 0, 512000)
		if err != nil {
			return nil, err
		}
		return &audiov1.TranscodeRequest{Audio: data, InputFormat: audioFormatFromString(ctx.Flag("input-format")), OutputFormat: audioFormatFromString(ctx.Flag("output-format")), SampleRate: sample, Channels: channels, Bitrate: bitrate}, nil
	}, h.client.Transcode, func(r *audiov1.TranscodeResponse) []byte { return r.GetAudio() })
}
func (h *handlers) transcode(ctx cliapp.RunContext) error { return h.transcodePrimitive().Run(ctx) }

func (h *handlers) trimPrimitive() cliapp.PrimitiveHandler {
	return audioOutputPrimitive("trim", func(ctx cliapp.OperationContext) (*audiov1.TrimRequest, error) {
		data, err := readAudioInput(ctx, "input")
		if err != nil {
			return nil, err
		}
		start, err := floatFlag(ctx, "start-seconds", 0, 86400)
		if err != nil {
			return nil, err
		}
		end, err := floatFlag(ctx, "end-seconds", 0, 86400)
		if err != nil {
			return nil, err
		}
		return &audiov1.TrimRequest{Audio: data, Format: audioFormatFromString(ctx.Flag("format")), StartSeconds: start, EndSeconds: end}, nil
	}, h.client.Trim, func(r *audiov1.TrimResponse) []byte { return r.GetAudio() })
}
func (h *handlers) trim(ctx cliapp.RunContext) error { return h.trimPrimitive().Run(ctx) }

func (h *handlers) mergePrimitive() cliapp.PrimitiveHandler {
	return audioOutputPrimitive("merge", func(ctx cliapp.OperationContext) (*audiov1.MergeRequest, error) {
		paths := nonEmpty(ctx.FlagValues("input"))
		if len(paths) == 0 {
			return nil, fmt.Errorf("merge: --input is required (repeat it for every source)")
		}
		format := audioFormatFromString(ctx.Flag("format"))
		sources := make([]*audiov1.MergeSource, 0, len(paths))
		for _, path := range paths {
			data, err := readFile(path)
			if err != nil {
				return nil, err
			}
			sources = append(sources, &audiov1.MergeSource{Audio: data, Format: format})
		}
		crossfade, err := floatFlag(ctx, "crossfade-seconds", 0, 60)
		if err != nil {
			return nil, err
		}
		return &audiov1.MergeRequest{Sources: sources, OutputFormat: audioFormatFromString(ctx.Flag("output-format")), CrossfadeSeconds: crossfade}, nil
	}, h.client.Merge, func(r *audiov1.MergeResponse) []byte { return r.GetAudio() })
}
func (h *handlers) merge(ctx cliapp.RunContext) error { return h.mergePrimitive().Run(ctx) }

func (h *handlers) splitPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutationWithJSONRenderer(func(ctx cliapp.OperationContext) (*audiov1.SplitResponse, error) {
		data, err := readAudioInput(ctx, "input")
		if err != nil {
			return nil, err
		}
		chunkSeconds, err := floatFlag(ctx, "chunk-seconds", 0, 3600)
		if err != nil {
			return nil, err
		}
		boundaries, err := floatListFlag(ctx, "boundaries-seconds")
		if err != nil {
			return nil, err
		}
		if chunkSeconds <= 0 && len(boundaries) == 0 {
			return nil, fmt.Errorf("split: set --chunk-seconds or --boundaries-seconds")
		}
		outDir := ctx.Flag("output")
		if outDir == "" {
			return nil, fmt.Errorf("split: --output directory is required")
		}
		resp, err := h.client.Split(context.Background(), connect.NewRequest(&audiov1.SplitRequest{Audio: data, Format: audioFormatFromString(ctx.Flag("format")), ChunkSeconds: chunkSeconds, BoundariesSeconds: boundaries, OutputFormat: audioFormatFromString(ctx.Flag("output-format"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("split", err, nil)
		}
		if err := os.MkdirAll(outDir, 0o700); err != nil {
			return nil, fmt.Errorf("create output directory %s: %w", outDir, err)
		}
		ext := fileExtension(ctx.Flag("output-format"))
		for i, chunk := range resp.Msg.Chunks {
			name := fmt.Sprintf("chunk-%03d.%s", i, ext)
			path := filepath.Join(outDir, name)
			if err := writeFile(path, chunk.Audio); err != nil {
				return nil, err
			}
		}
		return resp.Msg, nil
	}, func(ctx cliapp.OperationContext, r *audiov1.SplitResponse) cliapp.MutationReport {
		outDir := ctx.Flag("output")
		ext := fileExtension(ctx.Flag("output-format"))
		changes := make([]string, 0, len(r.GetChunks()))
		for i, chunk := range r.GetChunks() {
			changes = append(changes, fmt.Sprintf("%s (start %.3fs, duration %.3fs)", filepath.Join(outDir, fmt.Sprintf("chunk-%03d.%s", i, ext)), chunk.GetStartSeconds(), chunk.GetDurationSeconds()))
		}
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Wrote %d chunk(s) to %s.", len(changes), outDir)}, Changes: changes}
	}, func(ctx cliapp.RunContext, r *audiov1.SplitResponse) error {
		type chunkFile struct {
			Path            string  `json:"path"`
			StartSeconds    float64 `json:"start_seconds"`
			DurationSeconds float64 `json:"duration_seconds"`
			ContentType     string  `json:"content_type,omitempty"`
			BytesWritten    int     `json:"bytes_written"`
		}
		chunks := make([]chunkFile, 0, len(r.GetChunks()))
		for i, chunk := range r.GetChunks() {
			chunks = append(chunks, chunkFile{
				Path:            filepath.Join(ctx.Flag("output"), fmt.Sprintf("chunk-%03d.%s", i, fileExtension(ctx.Flag("output-format")))),
				StartSeconds:    chunk.GetStartSeconds(),
				DurationSeconds: chunk.GetDurationSeconds(),
				ContentType:     chunk.GetContentType(),
				BytesWritten:    len(chunk.GetAudio()),
			})
		}
		return cliapp.PrintJSON(ctx.Stdout(), struct {
			Operation       string      `json:"operation"`
			OutputDirectory string      `json:"output_directory"`
			Chunks          []chunkFile `json:"chunks"`
		}{Operation: "split", OutputDirectory: ctx.Flag("output"), Chunks: chunks})
	})
}

func (h *handlers) split(ctx cliapp.RunContext) error { return h.splitPrimitive().Run(ctx) }

func (h *handlers) fadePrimitive() cliapp.PrimitiveHandler {
	return audioOutputPrimitive("fade", func(ctx cliapp.OperationContext) (*audiov1.FadeRequest, error) {
		data, err := readAudioInput(ctx, "input")
		if err != nil {
			return nil, err
		}
		fi, err := floatFlag(ctx, "fade-in-seconds", 0, 3600)
		if err != nil {
			return nil, err
		}
		fo, err := floatFlag(ctx, "fade-out-seconds", 0, 3600)
		if err != nil {
			return nil, err
		}
		return &audiov1.FadeRequest{Audio: data, Format: audioFormatFromString(ctx.Flag("format")), FadeInSeconds: fi, FadeOutSeconds: fo, OutputFormat: audioFormatFromString(ctx.Flag("output-format"))}, nil
	}, h.client.Fade, func(r *audiov1.FadeResponse) []byte { return r.GetAudio() })
}
func (h *handlers) fade(ctx cliapp.RunContext) error { return h.fadePrimitive().Run(ctx) }

func (h *handlers) volumePrimitive() cliapp.PrimitiveHandler {
	return audioOutputPrimitive("volume", func(ctx cliapp.OperationContext) (*audiov1.VolumeRequest, error) {
		data, err := readAudioInput(ctx, "input")
		if err != nil {
			return nil, err
		}
		gain, err := floatFlag(ctx, "gain-db", -60, 24)
		if err != nil {
			return nil, err
		}
		return &audiov1.VolumeRequest{Audio: data, Format: audioFormatFromString(ctx.Flag("format")), GainDb: gain, OutputFormat: audioFormatFromString(ctx.Flag("output-format"))}, nil
	}, h.client.Volume, func(r *audiov1.VolumeResponse) []byte { return r.GetAudio() })
}
func (h *handlers) volume(ctx cliapp.RunContext) error { return h.volumePrimitive().Run(ctx) }

func (h *handlers) normalizePrimitive() cliapp.PrimitiveHandler {
	return audioOutputPrimitive("normalize", func(ctx cliapp.OperationContext) (*audiov1.NormalizeRequest, error) {
		data, err := readAudioInput(ctx, "input")
		if err != nil {
			return nil, err
		}
		target, err := floatFlag(ctx, "target-lufs", -40, -5)
		if err != nil {
			return nil, err
		}
		return &audiov1.NormalizeRequest{Audio: data, Format: audioFormatFromString(ctx.Flag("format")), Method: normalizationMethodFromString(ctx.Flag("method")), TargetLufs: target, OutputFormat: audioFormatFromString(ctx.Flag("output-format"))}, nil
	}, h.client.Normalize, func(r *audiov1.NormalizeResponse) []byte { return r.GetAudio() })
}
func (h *handlers) normalize(ctx cliapp.RunContext) error { return h.normalizePrimitive().Run(ctx) }

// extract-metadata preserves its explicit stdout JSON stream mode and file
// mutation summary while keeping request execution independent of --json.
func (h *handlers) extractMetadataPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*audiov1.ExtractMetadataResponse, error) {
		data, err := readAudioInput(ctx, "input")
		if err != nil {
			return nil, err
		}
		resp, err := h.client.ExtractMetadata(context.Background(), connect.NewRequest(&audiov1.ExtractMetadataRequest{Audio: data, Format: audioFormatFromString(ctx.Flag("format"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("extract-metadata", err, nil)
		}
		out := ctx.Flag("output")
		if out == "" {
			return nil, fmt.Errorf("extract-metadata: --output is required")
		}
		if out != "-" {
			body, err := protojson.MarshalOptions{Indent: "  ", UseProtoNames: true, EmitUnpopulated: true}.Marshal(resp.Msg)
			if err != nil {
				return nil, fmt.Errorf("encode metadata: %w", err)
			}
			body = append(body, '\n')
			if err := writeFile(out, body); err != nil {
				return nil, err
			}
		}
		return resp.Msg, nil
	}, func(ctx cliapp.RunContext, r *audiov1.ExtractMetadataResponse) error {
		if ctx.Flag("output") == "-" {
			return writeMetadataJSON(ctx.Stdout(), r)
		}
		return cliapp.RenderProtoMutation(ctx, r, cliapp.MutationReport{Result: []string{fmt.Sprintf("Wrote metadata to %s.", ctx.Flag("output"))}})
	}, func(ctx cliapp.RunContext, r *audiov1.ExtractMetadataResponse) error {
		if ctx.Flag("output") == "-" {
			return writeMetadataJSON(ctx.Stdout(), r)
		}
		return cliapp.PrintProtoJSON(ctx.Stdout(), r)
	})
}

func writeMetadataJSON(w interface{ Write([]byte) (int, error) }, r *audiov1.ExtractMetadataResponse) error {
	body, err := protojson.MarshalOptions{Indent: "  ", UseProtoNames: true, EmitUnpopulated: true}.Marshal(r)
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}
	body = append(body, '\n')
	_, err = w.Write(body)
	return err
}

func (h *handlers) extractMetadata(ctx cliapp.RunContext) error {
	return h.extractMetadataPrimitive().Run(ctx)
}

// Shared input / output / conversion helpers.
func readAudioInput(ctx cliapp.OperationContext, flagName string) ([]byte, error) {
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
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	// os.WriteFile does not change permissions when the file already exists.
	// Keep transformed speech private on both first write and overwrite.
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("restrict permissions for %s: %w", path, err)
	}
	return nil
}

func fileExtension(format string) string {
	if format == "" {
		return "wav"
	}
	return format
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

func floatFlag(ctx cliapp.OperationContext, name string, min, max float64) (float64, error) {
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

func int32Flag(ctx cliapp.OperationContext, name string, min, max int32) (int32, error) {
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

func floatListFlag(ctx cliapp.OperationContext, name string) ([]float64, error) {
	values := ctx.FlagValues(name)
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strings.Split(v, ",")...)
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
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}
