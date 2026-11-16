package audio

import (
	"context"

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
