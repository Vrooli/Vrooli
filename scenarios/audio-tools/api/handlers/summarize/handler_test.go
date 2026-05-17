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
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	summH "audio-tools/handlers/summarize"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/byok/envelope"
	intsumm "audio-tools/internal/summarize"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// stubVrooliClient is a summarizechain.VrooliClient under test control.
type stubVrooliClient struct {
	available bool
	res       *summarizechain.Result
	err       error
}

func (s *stubVrooliClient) IsAvailable(context.Context) bool { return s.available }
func (s *stubVrooliClient) Model() string                    { return "stub" }
func (s *stubVrooliClient) Summarize(context.Context, string, string, summarizechain.Request) (*summarizechain.Result, error) {
	return s.res, s.err
}

func newServer(t *testing.T, chain *summarizechain.Chain) summconnect.SummarizeServiceClient {
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
	}, nil, nil)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return summconnect.NewSummarizeServiceClient(http.DefaultClient, srv.URL)
}

func TestSummarize_HappyPathViaVrooli(t *testing.T) {
	chain := summarizechain.NewChain(summarizechain.Options{
		EnableVrooli: true,
		Vrooli: summarizechain.NewVrooliProvider(&stubVrooliClient{
			available: true,
			res:       &summarizechain.Result{Text: "summary", PromptTokens: 10, OutputTokens: 5},
		}),
	})
	c := newServer(t, chain)
	req := connect.NewRequest(&summv1.SummarizeRequest{Text: "long text", Level: ttsv1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE})
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	res, err := c.Summarize(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "summary", res.Msg.GetText())
	require.Equal(t, int32(10), res.Msg.GetPromptTokens())
	require.Equal(t, commonv1.ProviderTier_PROVIDER_TIER_VROOLI, res.Msg.GetProviderTier())
}

func TestSummarize_InsufficientCreditsMapsToResourceExhausted(t *testing.T) {
	chain := summarizechain.NewChain(summarizechain.Options{
		EnableVrooli: true,
		Vrooli: summarizechain.NewVrooliProvider(&stubVrooliClient{
			available: true,
			err:       summarizechain.ErrInsufficientCredits,
		}),
	})
	c := newServer(t, chain)
	req := connect.NewRequest(&summv1.SummarizeRequest{Text: "x", Level: ttsv1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT})
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	_, err := c.Summarize(context.Background(), req)
	require.Error(t, err)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
}

// stubBYOK is a registry-resident BYOKAdapter under test control.
type stubBYOK struct{ id string }

func (s *stubBYOK) ID() string                               { return s.id }
func (s *stubBYOK) Model() string                            { return "stub" }
func (s *stubBYOK) IsAvailable(context.Context, string) bool { return true }
func (s *stubBYOK) Summarize(context.Context, string, summarizechain.Request) (*summarizechain.Result, error) {
	return &summarizechain.Result{Text: "byok"}, nil
}

func TestSummarize_MissingBYOKProviderMapsToInvalidArgument(t *testing.T) {
	chain := summarizechain.NewChain(summarizechain.Options{
		EnableBYOK: true,
		BYOK: summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{
			"openrouter": &stubBYOK{id: "openrouter"},
		}),
	})
	c := newServer(t, chain)
	req := connect.NewRequest(&summv1.SummarizeRequest{Text: "x", Level: ttsv1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT})
	// Set BYOK key but no provider — chain returns ErrMissingBYOKProvider.
	req.Header().Set(envelope.HeaderKey, "sk-1")
	_, err := c.Summarize(context.Background(), req)
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	_ = errors.Is // referenced symbol kept for clarity in tests
}

func TestSummarize_NilChainReturnsUnavailable(t *testing.T) {
	c := newServer(t, nil)
	_, err := c.Summarize(context.Background(), connect.NewRequest(&summv1.SummarizeRequest{Text: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestSummarize_ConfigGetSetRoundTrip(t *testing.T) {
	c := newServer(t, nil)
	ctx := context.Background()
	res, err := c.GetSummarizeConfig(ctx, connect.NewRequest(&summv1.GetSummarizeConfigRequest{}))
	require.NoError(t, err)
	require.NotNil(t, res.Msg.GetConfig())

	upd, err := c.UpdateSummarizeConfig(ctx, connect.NewRequest(&summv1.UpdateSummarizeConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"enabled", "level"}},
		Config: &summv1.SummarizeConfig{
			Enabled: false,
			Level:   ttsv1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY,
		},
	}))
	require.NoError(t, err)
	require.False(t, upd.Msg.GetConfig().GetEnabled())
	require.Equal(t, ttsv1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY, upd.Msg.GetConfig().GetLevel())

	got, err := c.GetSummarizeConfig(ctx, connect.NewRequest(&summv1.GetSummarizeConfigRequest{}))
	require.NoError(t, err)
	require.False(t, got.Msg.GetConfig().GetEnabled())
	require.Equal(t, ttsv1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY, got.Msg.GetConfig().GetLevel())
}
