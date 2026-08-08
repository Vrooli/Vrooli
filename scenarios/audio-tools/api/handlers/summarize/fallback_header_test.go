package summarize_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	summH "audio-tools/handlers/summarize"
	"audio-tools/internal/ai/summarizechain"
	summocks "audio-tools/internal/ai/summarizechain/mocks"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/clock"
	"audio-tools/internal/logx"
	intsumm "audio-tools/internal/summarize"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
)

// newServerWithChain mirrors handler_test.go's newServer.
func newServerForFallback(t *testing.T, chain *summarizechain.Chain) summconnect.SummarizeServiceClient {
	t.Helper()
	var mu sync.Mutex
	cfg := intsumm.DefaultSummarizeConfig()
	mod := summH.Module(chain, func() intsumm.SummarizeConfig {
		mu.Lock()
		defer mu.Unlock()
		return cfg
	}, func(c intsumm.SummarizeConfig) {
		mu.Lock()
		defer mu.Unlock()
		cfg = c
	}, func(context.Context) ([]intsumm.SummarizeModelInfo, error) {
		return intsumm.MergeSummarizeModels(nil), nil
	}, logx.Std{}, clock.System{}, nil)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return summconnect.NewSummarizeServiceClient(http.DefaultClient, srv.URL)
}

// TestSummarize_FallbackHeader_SetWhenLowerTierServes asserts that
// x-audio-tools-fallback is emitted when the first-priority tier (BYOK)
// fails and a lower tier (Vrooli) succeeds.
func TestSummarize_FallbackHeader_SetWhenLowerTierServes(t *testing.T) {
	chain := summarizechain.NewChain(summarizechain.Options{
		EnableBYOK:   true,
		EnableVrooli: true,
		BYOK: summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{
			"openrouter": &summocks.FakeBYOK{IDStr: "openrouter", Available: true, Err: errors.New("upstream timeout")},
		}),
		Vrooli: summarizechain.NewVrooliProvider(&summocks.FakeVrooliClient{
			Available: true,
			Result:    &summarizechain.Result{Text: "ok", PromptTokens: 1, OutputTokens: 1},
		}),
	})
	c := newServerForFallback(t, chain)
	req := connect.NewRequest(&summv1.SummarizeRequest{Text: "long text", Level: summv1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT})
	// Wire both BYOK and Vrooli creds so both tiers are eligible.
	req.Header().Set(envelope.HeaderBYOKKey, "sk-1")
	req.Header().Set(envelope.HeaderProvider, "openrouter")
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	res, err := c.Summarize(context.Background(), req)
	require.NoError(t, err)
	got := res.Header().Get("x-audio-tools-fallback")
	require.Contains(t, got, "from=byok")
	require.Contains(t, got, "to=vrooli")
	require.Contains(t, got, "reason=")
}

// TestSummarize_FallbackHeader_AbsentWhenFirstTierServes asserts the
// header is NOT set when the first-priority tier handles the request.
func TestSummarize_FallbackHeader_AbsentWhenFirstTierServes(t *testing.T) {
	chain := summarizechain.NewChain(summarizechain.Options{
		EnableVrooli: true,
		Vrooli: summarizechain.NewVrooliProvider(&summocks.FakeVrooliClient{
			Available: true,
			Result:    &summarizechain.Result{Text: "ok"},
		}),
	})
	c := newServerForFallback(t, chain)
	req := connect.NewRequest(&summv1.SummarizeRequest{Text: "x"})
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	res, err := c.Summarize(context.Background(), req)
	require.NoError(t, err)
	require.Empty(t, res.Header().Get("x-audio-tools-fallback"))
}
