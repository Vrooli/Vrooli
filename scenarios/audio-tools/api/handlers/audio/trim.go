package audio

import (
	"context"

	intaudio "audio-tools/internal/audio"

	"connectrpc.com/connect"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
)

func (h *connectHandler) Trim(ctx context.Context, req *connect.Request[audiov1.TrimRequest]) (*connect.Response[audiov1.TrimResponse], error) {
	if err := requireBytes(req.Msg.Audio); err != nil {
		return nil, err
	}
	inFmt := audioFormatString(req.Msg.GetFormat())
	out, err := intaudio.Trim(ctx, req.Msg.Audio, inFmt, req.Msg.StartSeconds, req.Msg.EndSeconds, "")
	if err != nil {
		return nil, mapAudioErr(err)
	}
	return connect.NewResponse(&audiov1.TrimResponse{Audio: out, ContentType: contentTypeFor(inFmt)}), nil
}
