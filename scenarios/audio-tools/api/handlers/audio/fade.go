package audio

import (
	"context"

	intaudio "audio-tools/internal/audio"

	"connectrpc.com/connect"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
)

func (h *connectHandler) Fade(ctx context.Context, req *connect.Request[audiov1.FadeRequest]) (*connect.Response[audiov1.FadeResponse], error) {
	if err := requireBytes(req.Msg.Audio); err != nil {
		return nil, err
	}
	inFmt := audioFormatString(req.Msg.GetFormat())
	outFmt := audioFormatString(req.Msg.GetOutputFormat())
	out, err := intaudio.Fade(ctx, req.Msg.Audio, inFmt, req.Msg.FadeInSeconds, req.Msg.FadeOutSeconds, outFmt)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	ct := outFmt
	if ct == "" {
		ct = inFmt
	}
	return connect.NewResponse(&audiov1.FadeResponse{Audio: out, ContentType: contentTypeFor(ct)}), nil
}
