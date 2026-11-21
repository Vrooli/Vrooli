package tts_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	ttsH "audio-tools/handlers/tts"
	"audio-tools/internal/logx"

	"github.com/vrooli/api-core/schedule"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

// TestTTSModuleRejectsOversizedMessage proves the transport ceiling in front of
// the TTS service. Text input has no proto max length, so an unbounded unary
// body would otherwise be buffered before the handler can validate anything.
func TestTTSModuleRejectsOversizedMessage(t *testing.T) {
	mod := ttsH.ModuleWithLimit(ttsH.Deps{Logger: logx.Std{}, Clock: schedule.System()}, 256)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	client := ttsconnect.NewTTSServiceClient(http.DefaultClient, srv.URL)

	_, err := client.Synthesize(context.Background(), connect.NewRequest(&ttsv1.SynthesizeRequest{
		Text: strings.Repeat("a", 1024),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), "got: %v", err)
}
