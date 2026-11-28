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
	inFmt := audioFormatString(req.Msg.GetInputFormat())
	outFmt := audioFormatString(req.Msg.GetOutputFormat())
	out, err := intaudio.TranscodeWithInput(ctx, req.Msg.Audio, inFmt, outFmt,
		int(req.Msg.SampleRate), int(req.Msg.Channels), int(req.Msg.Bitrate))
	if err != nil {
		return nil, mapAudioErr(err)
	}
	return connect.NewResponse(&audiov1.TranscodeResponse{
		Audio:           out,
		ContentType:     intaudio.ContentTypeFor(outFmt),
		DurationSeconds: intaudio.DurationSeconds(ctx, out),
	}), nil
}
