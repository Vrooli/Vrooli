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
	inFmt := audioFormatString(req.Msg.GetFormat())
	outFmt := audioFormatString(req.Msg.GetOutputFormat())
	out, err := intaudio.Volume(ctx, req.Msg.Audio, inFmt, req.Msg.GainDb, outFmt)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	ct := outFmt
	if ct == "" {
		ct = inFmt
	}
	return connect.NewResponse(&audiov1.VolumeResponse{Audio: out, ContentType: contentTypeFor(ct)}), nil
}
