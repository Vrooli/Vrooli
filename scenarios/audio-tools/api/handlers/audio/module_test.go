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

func newLimitedClient(t *testing.T, maxRequestBytes int) audioconnect.AudioProcessingServiceClient {
	t.Helper()
	mod := ModuleWithLimit(&mocks.FakeLogger{}, maxRequestBytes)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return audioconnect.NewAudioProcessingServiceClient(http.DefaultClient, srv.URL)
}

// TestModuleRejectsOversizedRequest proves the transport ceiling in front of
// the whole transform surface. Without it Connect buffers an unbounded unary
// body in process memory before the handler can validate anything.
func TestModuleRejectsOversizedRequest(t *testing.T) {
	const limit = 256
	c := newLimitedClient(t, limit)

	// ~1 KiB of audio expands past the 256-byte wire ceiling once encoded.
	_, err := c.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{
		Audio:        make([]byte, 1024),
		OutputFormat: 1,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), "got: %v", err)
}

// TestModuleBelowLimitStillReachesHandler keeps the ceiling from silently
// becoming a functional regression: a small request must still reach the
// handler and fail on its own validation, not on transport size.
func TestModuleBelowLimitStillReachesHandler(t *testing.T) {
	c := newLimitedClient(t, 4096)

	_, err := c.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "got: %v", err)
}

func TestModuleWithLimitRequiresPositiveLimit(t *testing.T) {
	require.Panics(t, func() { ModuleWithLimit(&mocks.FakeLogger{}, 0) })
}
