package audio

import (
	"context"
	"errors"

	intaudio "audio-tools/internal/audio"

	"connectrpc.com/connect"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
)

func (h *connectHandler) Merge(ctx context.Context, req *connect.Request[audiov1.MergeRequest]) (*connect.Response[audiov1.MergeResponse], error) {
	if len(req.Msg.Sources) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("at least one source required"))
	}
	bs := make([][]byte, 0, len(req.Msg.Sources))
	fmts := make([]string, 0, len(req.Msg.Sources))
	for _, s := range req.Msg.Sources {
		bs = append(bs, s.Audio)
		fmts = append(fmts, audioFormatString(s.GetFormat()))
	}
	outFmt := audioFormatString(req.Msg.GetOutputFormat())
	out, err := intaudio.Merge(ctx, bs, fmts, outFmt, req.Msg.CrossfadeSeconds)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	return connect.NewResponse(&audiov1.MergeResponse{Audio: out, ContentType: contentTypeFor(outFmt)}), nil
}
