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
	out, err := intaudio.Fade(ctx, req.Msg.Audio, req.Msg.Format, req.Msg.FadeInSeconds, req.Msg.FadeOutSeconds, req.Msg.OutputFormat)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	return connect.NewResponse(&audiov1.FadeResponse{Audio: out, ContentType: contentTypeFor(req.Msg.OutputFormat)}), nil
}
