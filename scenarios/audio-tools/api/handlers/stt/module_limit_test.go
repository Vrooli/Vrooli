package stt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/logx"

	"github.com/vrooli/api-core/schedule"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

// TestSTTModuleRejectsOversizedMessage proves the transport ceiling in front of
// both STT services. Without it a single oversized unary body is buffered
// before the handler's MaxAudioSize check can run.
func TestSTTModuleRejectsOversizedMessage(t *testing.T) {
	mod := moduleWithReadLimit(Deps{Logger: logx.Std{}, Clock: schedule.System()}, 256)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	client := sttconnect.NewSTTServiceClient(http.DefaultClient, srv.URL)

	_, err := client.Transcribe(context.Background(), connect.NewRequest(&sttv1.TranscribeRequest{
		Audio:  make([]byte, 1024),
		Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), "got: %v", err)
}
