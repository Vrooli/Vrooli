package audio

import (
	"context"

	intaudio "audio-tools/internal/audio"

	"connectrpc.com/connect"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
)

func (h *connectHandler) Split(ctx context.Context, req *connect.Request[audiov1.SplitRequest]) (*connect.Response[audiov1.SplitResponse], error) {
	if err := requireBytes(req.Msg.Audio); err != nil {
		return nil, err
	}
	chunks, err := intaudio.Split(ctx, req.Msg.Audio, req.Msg.Format, req.Msg.ChunkSeconds, req.Msg.BoundariesSeconds, req.Msg.OutputFormat)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	out := &audiov1.SplitResponse{Chunks: make([]*audiov1.SplitChunk, 0, len(chunks))}
	for _, c := range chunks {
		out.Chunks = append(out.Chunks, &audiov1.SplitChunk{
			Audio: c.Audio, ContentType: c.ContentType,
			StartSeconds: c.Start, DurationSeconds: c.Duration,
		})
	}
	return connect.NewResponse(out), nil
}
