package audio

import (
	"context"

	intaudio "audio-tools/internal/audio"

	"connectrpc.com/connect"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
)

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
