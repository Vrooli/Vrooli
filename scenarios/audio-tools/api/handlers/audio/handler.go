// Package audio hosts the AudioProcessingService Connect-RPC handler.
package audio

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	intaudio "audio-tools/internal/audio"
	"audio-tools/internal/modulekit"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	audioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio/audio_v1connect"
)

type Deps struct {
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// requireBytes is a small guard: every audio op requires a non-empty payload.
func requireBytes(b []byte) error {
	if len(b) == 0 {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("audio bytes are required"))
	}
	return nil
}

// mapAudioErr maps internal errors to connect codes. ErrFFmpegMissing
// is FailedPrecondition (operator action required); everything else is
// Internal so the client sees a fatal upstream failure.
func mapAudioErr(err error) error {
	if errors.Is(err, intaudio.ErrFFmpegMissing) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func (h *connectHandler) Transcode(ctx context.Context, req *connect.Request[audiov1.TranscodeRequest]) (*connect.Response[audiov1.TranscodeResponse], error) {
	if err := requireBytes(req.Msg.Audio); err != nil {
		return nil, err
	}
	out, err := intaudio.TranscodeOpts(ctx, req.Msg.Audio, req.Msg.OutputFormat,
		int(req.Msg.SampleRate), int(req.Msg.Channels), int(req.Msg.Bitrate))
	if err != nil {
		return nil, mapAudioErr(err)
	}
	ct := "audio/wav"
	if req.Msg.OutputFormat != "" {
		ct = contentTypeFor(req.Msg.OutputFormat)
	}
	return connect.NewResponse(&audiov1.TranscodeResponse{Audio: out, ContentType: ct}), nil
}

func (h *connectHandler) Trim(ctx context.Context, req *connect.Request[audiov1.TrimRequest]) (*connect.Response[audiov1.TrimResponse], error) {
	if err := requireBytes(req.Msg.Audio); err != nil {
		return nil, err
	}
	out, err := intaudio.Trim(ctx, req.Msg.Audio, req.Msg.Format, req.Msg.StartSeconds, req.Msg.EndSeconds, "")
	if err != nil {
		return nil, mapAudioErr(err)
	}
	return connect.NewResponse(&audiov1.TrimResponse{Audio: out, ContentType: contentTypeFor(req.Msg.Format)}), nil
}

func (h *connectHandler) Merge(ctx context.Context, req *connect.Request[audiov1.MergeRequest]) (*connect.Response[audiov1.MergeResponse], error) {
	if len(req.Msg.Sources) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("at least one source required"))
	}
	bs := make([][]byte, 0, len(req.Msg.Sources))
	fmts := make([]string, 0, len(req.Msg.Sources))
	for _, s := range req.Msg.Sources {
		bs = append(bs, s.Audio)
		fmts = append(fmts, s.Format)
	}
	out, err := intaudio.Merge(ctx, bs, fmts, req.Msg.OutputFormat, req.Msg.CrossfadeSeconds)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	return connect.NewResponse(&audiov1.MergeResponse{Audio: out, ContentType: contentTypeFor(req.Msg.OutputFormat)}), nil
}

func (h *connectHandler) Split(ctx context.Context, req *connect.Request[audiov1.SplitRequest]) (*connect.Response[audiov1.SplitResponse], error) {
	if err := requireBytes(req.Msg.Audio); err != nil {
		return nil, err
	}
	chunks, err := intaudio.Split(ctx, req.Msg.Audio, req.Msg.Format, req.Msg.ChunkSeconds, req.Msg.BoundariesSeconds, req.Msg.OutputFormat)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	out := &audiov1.SplitResponse{Chunks: make([]*audiov1.SplitChunk, 0, len(chunks))}
	for _, c := range chunks {
		out.Chunks = append(out.Chunks, &audiov1.SplitChunk{
			Audio: c.Audio, ContentType: c.ContentType,
			StartSeconds: c.Start, DurationSeconds: c.Duration,
		})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) Fade(ctx context.Context, req *connect.Request[audiov1.FadeRequest]) (*connect.Response[audiov1.FadeResponse], error) {
	if err := requireBytes(req.Msg.Audio); err != nil {
		return nil, err
	}
	out, err := intaudio.Fade(ctx, req.Msg.Audio, req.Msg.Format, req.Msg.FadeInSeconds, req.Msg.FadeOutSeconds, req.Msg.OutputFormat)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	return connect.NewResponse(&audiov1.FadeResponse{Audio: out, ContentType: contentTypeFor(req.Msg.OutputFormat)}), nil
}

func (h *connectHandler) Volume(ctx context.Context, req *connect.Request[audiov1.VolumeRequest]) (*connect.Response[audiov1.VolumeResponse], error) {
	if err := requireBytes(req.Msg.Audio); err != nil {
		return nil, err
	}
	out, err := intaudio.Volume(ctx, req.Msg.Audio, req.Msg.Format, req.Msg.GainDb, req.Msg.OutputFormat)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	return connect.NewResponse(&audiov1.VolumeResponse{Audio: out, ContentType: contentTypeFor(req.Msg.OutputFormat)}), nil
}

func (h *connectHandler) Normalize(ctx context.Context, req *connect.Request[audiov1.NormalizeRequest]) (*connect.Response[audiov1.NormalizeResponse], error) {
	if err := requireBytes(req.Msg.Audio); err != nil {
		return nil, err
	}
	out, err := intaudio.Normalize(ctx, req.Msg.Audio, req.Msg.Format, req.Msg.Method, req.Msg.TargetLufs, req.Msg.OutputFormat)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	return connect.NewResponse(&audiov1.NormalizeResponse{Audio: out, ContentType: contentTypeFor(req.Msg.OutputFormat), MeasuredLufs: req.Msg.TargetLufs}), nil
}

func (h *connectHandler) ExtractMetadata(ctx context.Context, req *connect.Request[audiov1.ExtractMetadataRequest]) (*connect.Response[audiov1.ExtractMetadataResponse], error) {
	if err := requireBytes(req.Msg.Audio); err != nil {
		return nil, err
	}
	m, err := intaudio.Probe(ctx, req.Msg.Audio)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	return connect.NewResponse(&audiov1.ExtractMetadataResponse{
		Metadata: &audiov1.AudioMetadata{
			DurationSeconds: m.DurationSeconds,
			SampleRate:      m.SampleRate,
			Channels:        m.Channels,
			Bitrate:         m.Bitrate,
			Codec:           m.Codec,
			Format:          m.Format,
			Tags:            m.Tags,
		},
	}), nil
}

func contentTypeFor(format string) string {
	switch format {
	case "wav", "":
		return "audio/wav"
	case "mp3":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "aac":
		return "audio/aac"
	case "ogg":
		return "audio/ogg"
	}
	return "application/octet-stream"
}

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "audio.transcode", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Transcode", Method: "POST", Summary: "Transcode audio (Connect-RPC)", Category: "audio"},
	{ID: "audio.transcode_multipart", Path: "/api/v1/audio/transcode", Method: "POST", Summary: "Multipart audio transcode", Category: "audio",
		RESTException: &modulekit.RESTException{Reason: modulekit.RESTReasonMultipartUpload, Note: "Audio bytes via multipart form-data."}},
	{ID: "audio.trim", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Trim", Method: "POST", Category: "audio"},
	{ID: "audio.merge", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Merge", Method: "POST", Category: "audio"},
	{ID: "audio.split", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Split", Method: "POST", Category: "audio"},
	{ID: "audio.fade", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Fade", Method: "POST", Category: "audio"},
	{ID: "audio.volume", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Volume", Method: "POST", Category: "audio"},
	{ID: "audio.normalize", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Normalize", Method: "POST", Category: "audio"},
	{ID: "audio.extract_metadata", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/ExtractMetadata", Method: "POST", Category: "audio"},
}

func Module(logger *log.Logger) modulekit.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, h := audioconnect.NewAudioProcessingServiceHandler(NewConnectHandler(Deps{Logger: logger}))
	return modulekit.Module{
		Name: "audio",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
			r.Handle("/api/v1/audio/transcode", multipartTranscodeHandler()).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

// multipartTranscodeHandler accepts a multipart form with an "audio"
// file part and optional "output_format" / "sample_rate" / "channels"
// / "bitrate" fields. Returns the transcoded bytes verbatim.
func multipartTranscodeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f, _, err := r.FormFile("audio")
		if err != nil {
			http.Error(w, "audio file required", http.StatusBadRequest)
			return
		}
		defer f.Close()
		buf := make([]byte, 0, 1<<20)
		tmp := make([]byte, 32<<10)
		for {
			n, err := f.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
			}
			if err != nil {
				break
			}
		}
		format := r.FormValue("output_format")
		if format == "" {
			format = "wav"
		}
		out, err := intaudio.TranscodeOpts(r.Context(), buf, format,
			atoiOr(r.FormValue("sample_rate"), 0),
			atoiOr(r.FormValue("channels"), 0),
			atoiOr(r.FormValue("bitrate"), 0),
		)
		if err != nil {
			status := http.StatusBadGateway
			if errors.Is(err, intaudio.ErrFFmpegMissing) {
				status = http.StatusFailedDependency
			}
			http.Error(w, fmt.Sprintf("transcode failed: %v", err), status)
			return
		}
		w.Header().Set("Content-Type", contentTypeFor(format))
		_, _ = w.Write(out)
	})
}

func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	if err != nil {
		return def
	}
	return v
}
