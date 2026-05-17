package audio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/testutil/mocks"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	audioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio/audio_v1connect"
)

func newClient(t *testing.T) audioconnect.AudioProcessingServiceClient {
	t.Helper()
	mod := Module(&mocks.FakeLogger{})
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return audioconnect.NewAudioProcessingServiceClient(http.DefaultClient, srv.URL)
}

// TestEmptyAudioReturnsInvalidArgument exercises requireBytes on every
// RPC that accepts an audio payload. This is the cheapest happy-path
// guard the handler suite can provide without depending on ffmpeg.
func TestEmptyAudioReturnsInvalidArgument(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	cases := []struct {
		name string
		call func() error
	}{
		{"transcode", func() error {
			_, err := c.Transcode(ctx, connect.NewRequest(&audiov1.TranscodeRequest{}))
			return err
		}},
		{"trim", func() error {
			_, err := c.Trim(ctx, connect.NewRequest(&audiov1.TrimRequest{}))
			return err
		}},
		{"merge", func() error {
			_, err := c.Merge(ctx, connect.NewRequest(&audiov1.MergeRequest{}))
			return err
		}},
		{"split", func() error {
			_, err := c.Split(ctx, connect.NewRequest(&audiov1.SplitRequest{}))
			return err
		}},
		{"fade", func() error {
			_, err := c.Fade(ctx, connect.NewRequest(&audiov1.FadeRequest{}))
			return err
		}},
		{"volume", func() error {
			_, err := c.Volume(ctx, connect.NewRequest(&audiov1.VolumeRequest{}))
			return err
		}},
		{"normalize", func() error {
			_, err := c.Normalize(ctx, connect.NewRequest(&audiov1.NormalizeRequest{}))
			return err
		}},
		{"metadata", func() error {
			_, err := c.ExtractMetadata(ctx, connect.NewRequest(&audiov1.ExtractMetadataRequest{}))
			return err
		}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err)
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "got: %v", err)
		})
	}
}

func TestContentTypeFor_MapsKnownFormats(t *testing.T) {
	cases := map[string]string{
		"":     "audio/wav",
		"wav":  "audio/wav",
		"mp3":  "audio/mpeg",
		"flac": "audio/flac",
		"aac":  "audio/aac",
		"ogg":  "audio/ogg",
		"???":  "application/octet-stream",
	}
	for in, want := range cases {
		require.Equal(t, want, contentTypeFor(in), "format=%q", in)
	}
}
