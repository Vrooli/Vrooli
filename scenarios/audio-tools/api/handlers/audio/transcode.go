package audio

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	intaudio "audio-tools/internal/audio"

	"connectrpc.com/connect"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
)

func (h *connectHandler) Transcode(ctx context.Context, req *connect.Request[audiov1.TranscodeRequest]) (*connect.Response[audiov1.TranscodeResponse], error) {
	if err := requireBytes(req.Msg.Audio); err != nil {
		return nil, err
	}
	outFmt := audioFormatString(req.Msg.GetOutputFormat())
	out, err := intaudio.TranscodeOpts(ctx, req.Msg.Audio, outFmt,
		int(req.Msg.SampleRate), int(req.Msg.Channels), int(req.Msg.Bitrate))
	if err != nil {
		return nil, mapAudioErr(err)
	}
	ct := "audio/wav"
	if outFmt != "" {
		ct = contentTypeFor(outFmt)
	}
	return connect.NewResponse(&audiov1.TranscodeResponse{Audio: out, ContentType: ct}), nil
}

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
			// Mirror the Connect-side honest-error mapping: a missing ffmpeg
			// is an operator-fixable dependency gap (424), ffmpeg rejecting
			// the upload is a caller-fixable bad input (400), and anything
			// else is a genuine upstream failure (502).
			status := http.StatusBadGateway
			switch {
			case errors.Is(err, intaudio.ErrFFmpegMissing):
				status = http.StatusFailedDependency
			case errors.Is(err, intaudio.ErrFfmpegExec):
				status = http.StatusBadRequest
			}
			http.Error(w, fmt.Sprintf("transcode failed: %v", err), status)
			return
		}
		w.Header().Set("Content-Type", contentTypeFor(format))
		_, _ = w.Write(out)
	})
}
