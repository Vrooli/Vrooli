package summarize_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	summH "audio-tools/handlers/summarize"
	"audio-tools/internal/logx"
	intsumm "audio-tools/internal/summarize"

	"github.com/vrooli/api-core/schedule"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
)

// TestSummarizeModuleRejectsOversizedMessage proves the transport ceiling in
// front of the summarize service. The text field has no proto max length, so a
// single unary body could otherwise be buffered without bound.
func TestSummarizeModuleRejectsOversizedMessage(t *testing.T) {
	cfg := intsumm.DefaultSummarizeConfig()
	mod := summH.ModuleWithLimit(nil,
		func() intsumm.SummarizeConfig { return cfg },
		func(next intsumm.SummarizeConfig) { cfg = next },
		func(context.Context) ([]intsumm.SummarizeModelInfo, error) { return nil, nil },
		logx.Std{}, schedule.System(), nil, 256)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	client := summconnect.NewSummarizeServiceClient(http.DefaultClient, srv.URL)

	_, err := client.Summarize(context.Background(), connect.NewRequest(&summv1.SummarizeRequest{
		Text: strings.Repeat("a", 1024),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), "got: %v", err)
}
