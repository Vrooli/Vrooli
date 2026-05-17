package audio

import (
	"context"

	intaudio "audio-tools/internal/audio"

	"connectrpc.com/connect"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
)

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
