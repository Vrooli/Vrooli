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
	inFmt := audioFormatString(req.Msg.GetFormat())
	outFmt := audioFormatString(req.Msg.GetOutputFormat())
	method := normalizationMethodString(req.Msg.GetMethod())
	out, err := intaudio.Normalize(ctx, req.Msg.Audio, inFmt, method, req.Msg.TargetLufs, outFmt)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	ct := outFmt
	if ct == "" {
		ct = inFmt
	}
	return connect.NewResponse(&audiov1.NormalizeResponse{Audio: out, ContentType: contentTypeFor(ct), MeasuredLufs: req.Msg.TargetLufs}), nil
}
