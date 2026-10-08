package audio

import (
	"context"

	intaudio "audio-tools/internal/audio"

	"connectrpc.com/connect"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
)

func (h *connectHandler) ExtractMetadata(ctx context.Context, req *connect.Request[audiov1.ExtractMetadataRequest]) (*connect.Response[audiov1.ExtractMetadataResponse], error) {
	if err := requireBytes(req.Msg.Audio); err != nil {
		return nil, err
	}
	m, err := intaudio.Probe(ctx, req.Msg.Audio)
	if err != nil {
		return nil, mapAudioErr(err)
	}
	return connect.NewResponse(&audiov1.ExtractMetadataResponse{
		Metadata: &audiov1.AudioMetadata{
			DurationSeconds: m.DurationSeconds,
			SampleRate:      m.SampleRate,
			Channels:        m.Channels,
			Bitrate:         m.Bitrate,
			Codec:           m.Codec,
			Format:          audioFormatFromString(m.Format),
			Tags:            m.Tags,
		},
	}), nil
}
