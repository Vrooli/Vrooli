package summarize_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	summH "audio-tools/handlers/summarize"
	"audio-tools/internal/ai/summarizechain"
	summocks "audio-tools/internal/ai/summarizechain/mocks"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/logx"
	intsumm "audio-tools/internal/summarize"

	"github.com/vrooli/api-core/schedule"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
)

func newServer(t *testing.T, chain *summarizechain.Chain) summconnect.SummarizeServiceClient {
	t.Helper()
	var mu sync.Mutex
	cfg := intsumm.DefaultSummarizeConfig()
	mod := summH.ModuleWithRoleResolver(chain, func() intsumm.SummarizeConfig {
		mu.Lock()
		defer mu.Unlock()
		return cfg
	}, func(c intsumm.SummarizeConfig) {
		mu.Lock()
		defer mu.Unlock()
		cfg = c
	}, func(context.Context) ([]intsumm.SummarizeModelInfo, error) {
		return intsumm.SummarizeRoleOptions([]intsumm.DeclaredSummaryRole{{ID: intsumm.DefaultSummarizeModel}, {ID: "summarize.audio_tools.quality"}}), nil
	}, func(_ context.Context, role string) (bool, error) {
		return role == intsumm.DefaultSummarizeModel || role == "summarize.audio_tools.quality", nil
	}, logx.Std{}, schedule.System(), nil)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return summconnect.NewSummarizeServiceClient(http.DefaultClient, srv.URL)
}

func TestSummarizeEscalatesLongTextToDeclaredQualityRole(t *testing.T) {
	var selected string
	local := intsumm.NewSummarizerWithRunner("test", func(_ context.Context, args []string, _ string) ([]byte, error) {
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "--role" || args[i] == "--model" {
				selected = args[i+1]
			}
		}
		return []byte(`{"model":"qwen3.5:4b","response":"facts","eval_count":3}`), nil
	})
	chain := summarizechain.NewChain(summarizechain.Options{EnableLocal: true, Local: summarizechain.NewLocalProvider(local, intsumm.DefaultSummarizeModel)})
	c := newServer(t, chain)
	text := strings.Repeat("fact ", 300)
	res, err := c.Summarize(context.Background(), connect.NewRequest(&summv1.SummarizeRequest{Text: text}))
	require.NoError(t, err)
	require.Equal(t, "summarize.audio_tools.quality", selected)
	require.Equal(t, "qwen3.5:4b", res.Msg.GetModelId())
}

func TestSummarizeKeepsCompactRouteForShortText(t *testing.T) {
	var selected string
	local := intsumm.NewSummarizerWithRunner("test", func(_ context.Context, args []string, _ string) ([]byte, error) {
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "--role" || args[i] == "--model" {
				selected = args[i+1]
			}
		}
		return []byte(`{"model":"qwen3:1.7b","response":"summary","eval_count":3}`), nil
	})
	chain := summarizechain.NewChain(summarizechain.Options{EnableLocal: true, Local: summarizechain.NewLocalProvider(local, intsumm.DefaultSummarizeModel)})
	c := newServer(t, chain)
	_, err := c.Summarize(context.Background(), connect.NewRequest(&summv1.SummarizeRequest{Text: "short report"}))
	require.NoError(t, err)
	require.Equal(t, intsumm.DefaultSummarizeModel, selected)
}

func TestSummarizeHandlerPreservesSourceAssignmentWithoutChangingResponseContract(t *testing.T) {
	local := intsumm.NewSummarizerWithRunner("test", func(_ context.Context, _ []string, _ string) ([]byte, error) {
		return []byte(`{"model":"qwen3:1.7b","response":"Priya manages caption follow-up.","eval_count":8}`), nil
	})
	chain := summarizechain.NewChain(summarizechain.Options{
		EnableLocal: true,
		Local:       summarizechain.NewLocalProvider(local, intsumm.DefaultSummarizeModel),
	})
	c := newServer(t, chain)
	res, err := c.Summarize(context.Background(), connect.NewRequest(&summv1.SummarizeRequest{
		Text: "Priya owns caption follow-up. Marcus owns legal approval.",
	}))
	require.NoError(t, err)
	require.Equal(t, "Priya manages caption follow-up.\n\nMarcus owns legal approval.", res.Msg.GetText())
	require.Equal(t, "qwen3:1.7b", res.Msg.GetModelId())
	require.Equal(t, commonv1.ProviderTier_PROVIDER_TIER_LOCAL, res.Msg.GetProviderTier())
}

func TestSummarize_ListModelsReturnsCatalog(t *testing.T) {
	c := newServer(t, nil)
	res, err := c.ListSummarizeModels(context.Background(), connect.NewRequest(&summv1.ListSummarizeModelsRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, res.Msg.GetModels())
	require.Equal(t, intsumm.DefaultSummarizeModel, res.Msg.GetModels()[0].GetId())
	require.False(t, res.Msg.GetModels()[0].GetInstalled(), "declared role does not imply that its model is installed")
}

func TestSummarize_HappyPathViaVrooli(t *testing.T) {
	chain := summarizechain.NewChain(summarizechain.Options{
		EnableVrooli: true,
		Vrooli: summarizechain.NewVrooliProvider(&summocks.FakeVrooliClient{
			Available: true,
			Result:    &summarizechain.Result{Text: "summary", PromptTokens: 10, OutputTokens: 5, ProviderID: "lpbs", ModelID: "summary-model", Latency: 1500 * time.Microsecond},
		}),
	})
	c := newServer(t, chain)
	req := connect.NewRequest(&summv1.SummarizeRequest{Text: "long text", Level: summv1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE})
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	res, err := c.Summarize(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "summary", res.Msg.GetText())
	require.Equal(t, int32(10), res.Msg.GetPromptTokens())
	require.Equal(t, commonv1.ProviderTier_PROVIDER_TIER_VROOLI, res.Msg.GetProviderTier())
	require.Equal(t, "lpbs", res.Msg.GetProviderId())
	require.Equal(t, "summary-model", res.Msg.GetModelId())
	require.InDelta(t, 1.5, res.Msg.GetLatencyMs(), 1e-9)
}

func TestSummarizeLocalRoleSelectorIsConsumedByLocalProvider(t *testing.T) {
	var selected string
	local := intsumm.NewSummarizerWithRunner("test", func(_ context.Context, args []string, _ string) ([]byte, error) {
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "--role" || args[i] == "--model" {
				selected = args[i+1]
			}
		}
		return []byte(`{"model":"qwen3.5:4b","response":"facts","eval_count":3}`), nil
	})
	chain := summarizechain.NewChain(summarizechain.Options{EnableLocal: true, Local: summarizechain.NewLocalProvider(local, intsumm.DefaultSummarizeModel)})
	c := newServer(t, chain)
	res, err := c.Summarize(context.Background(), connect.NewRequest(&summv1.SummarizeRequest{Text: "facts", LocalModelRole: "summarize.audio_tools.quality"}))
	require.NoError(t, err)
	require.Equal(t, "summarize.audio_tools.quality", selected)
	require.Equal(t, "qwen3.5:4b", res.Msg.GetModelId())
}

func TestSummarizeRejectsUndeclaredLocalRole(t *testing.T) {
	c := newServer(t, summarizechain.NewChain(summarizechain.Options{EnableLocal: true, Local: summarizechain.NewLocalProvider(nil, intsumm.DefaultSummarizeModel)}))
	_, err := c.Summarize(context.Background(), connect.NewRequest(&summv1.SummarizeRequest{Text: "facts", LocalModelRole: "summarize.audio_tools.unknown"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestSummarizePreservesSlashStyleBYOKModelOverride(t *testing.T) {
	var received string
	chain := summarizechain.NewChain(summarizechain.Options{
		EnableBYOK: true,
		BYOK: summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{
			"openrouter": &summocks.FakeBYOK{IDStr: "openrouter", Available: true, SummarizeFn: func(_ context.Context, _ string, req summarizechain.Request) (*summarizechain.Result, error) {
				received = req.Model
				return &summarizechain.Result{Text: "summary"}, nil
			}},
		}),
	})
	c := newServer(t, chain)
	req := connect.NewRequest(&summv1.SummarizeRequest{Text: "facts", Model: "anthropic/claude-3.7-sonnet"})
	req.Header().Set(envelope.HeaderProvider, "openrouter")
	req.Header().Set(envelope.HeaderBYOKKey, "test-key")
	_, err := c.Summarize(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "anthropic/claude-3.7-sonnet", received)
}

func TestSummarize_InsufficientCreditsMapsToResourceExhausted(t *testing.T) {
	chain := summarizechain.NewChain(summarizechain.Options{
		EnableVrooli: true,
		Vrooli: summarizechain.NewVrooliProvider(&summocks.FakeVrooliClient{
			Available: true,
			Err:       summarizechain.ErrInsufficientCredits,
		}),
	})
	c := newServer(t, chain)
	req := connect.NewRequest(&summv1.SummarizeRequest{Text: "x", Level: summv1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT})
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	_, err := c.Summarize(context.Background(), req)
	require.Error(t, err)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
}

func TestSummarize_MissingBYOKProviderMapsToInvalidArgument(t *testing.T) {
	chain := summarizechain.NewChain(summarizechain.Options{
		EnableBYOK: true,
		BYOK: summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{
			"openrouter": &summocks.FakeBYOK{IDStr: "openrouter", Available: true, Result: &summarizechain.Result{Text: "byok"}},
		}),
	})
	c := newServer(t, chain)
	req := connect.NewRequest(&summv1.SummarizeRequest{Text: "x", Level: summv1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT})
	// Set BYOK key but no provider — chain returns ErrMissingBYOKProvider.
	req.Header().Set(envelope.HeaderBYOKKey, "sk-1")
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
			Level:   summv1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY,
		},
	}))
	require.NoError(t, err)
	require.False(t, upd.Msg.GetConfig().GetEnabled())
	require.Equal(t, summv1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY, upd.Msg.GetConfig().GetLevel())

	got, err := c.GetSummarizeConfig(ctx, connect.NewRequest(&summv1.GetSummarizeConfigRequest{}))
	require.NoError(t, err)
	require.False(t, got.Msg.GetConfig().GetEnabled())
	require.Equal(t, summv1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY, got.Msg.GetConfig().GetLevel())
}
