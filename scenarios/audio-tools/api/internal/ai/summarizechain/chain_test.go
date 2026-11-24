package summarizechain_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/summarizechain"
	summocks "audio-tools/internal/ai/summarizechain/mocks"
	"audio-tools/internal/summarize"
)

func TestChain_WiresCredentialProviders(t *testing.T) {
	chain := summarizechain.NewChain(summarizechain.Options{
		EnableBYOK: true, EnableVrooli: true,
		BYOK:   summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{"openrouter": &summocks.FakeBYOK{IDStr: "openrouter", Available: true}}),
		Vrooli: summarizechain.NewVrooliProvider(&summocks.FakeVrooliClient{Available: true}),
	})

	byok, err := chain.Execute(context.Background(), summarizechain.Request{Text: "hi", BYOKProvider: "openrouter", BYOKKey: "sk", LPBSToken: "tok"})
	require.NoError(t, err)
	require.Equal(t, "byok-summary", byok.Text)

	vrooli, err := chain.Execute(context.Background(), summarizechain.Request{Text: "hi", LPBSToken: "tok"})
	require.NoError(t, err)
	require.Equal(t, "vrooli-summary", vrooli.Text)
}

func TestChain_ExplicitLocalRoleOverridesPresentRemoteCredentials(t *testing.T) {
	byokCalls, lpbsCalls := 0, 0
	local := summarize.NewSummarizerWithRunner("test", func(context.Context, []string, string) ([]byte, error) {
		return json.Marshal(map[string]string{"model": "qwen3.5:4b", "response": "local quality"})
	})
	chain := summarizechain.NewChain(summarizechain.Options{
		EnableLocal: true, EnableBYOK: true, EnableVrooli: true,
		Local: summarizechain.NewLocalProvider(local, summarize.DefaultSummarizeModel),
		BYOK: summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{"openrouter": &summocks.FakeBYOK{IDStr: "openrouter", Available: true, SummarizeFn: func(context.Context, string, summarizechain.Request) (*summarizechain.Result, error) {
			byokCalls++
			return &summarizechain.Result{Text: "byok"}, nil
		}}}),
		Vrooli: summarizechain.NewVrooliProvider(&summocks.FakeVrooliClient{Available: true, SummarizeFn: func(context.Context, string, string, summarizechain.Request) (*summarizechain.Result, error) {
			lpbsCalls++
			return &summarizechain.Result{Text: "lpbs"}, nil
		}}),
	})
	result, err := chain.Execute(context.Background(), summarizechain.Request{
		Text: "facts", LocalModelRole: "summarize.audio_tools.quality",
		BYOKProvider: "openrouter", BYOKKey: "key", LPBSToken: "token",
	})
	require.NoError(t, err)
	require.Equal(t, summarizechain.TierLocal, result.Tier)
	require.Equal(t, "qwen3.5:4b", result.ModelID)
	require.Equal(t, 0, byokCalls)
	require.Equal(t, 0, lpbsCalls)
}

func TestChain_Reconfigure_InvalidatesAvailabilityCache(t *testing.T) {
	c := summarizechain.NewChain(summarizechain.Options{
		EnableBYOK: true,
		BYOK:       summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{"x": &summocks.FakeBYOK{IDStr: "x", Available: true}}),
	})
	// Prime the cache by running availFor through Execute.
	_, _ = c.Execute(context.Background(), summarizechain.Request{Text: "hi", BYOKProvider: "x", BYOKKey: "k"})

	// Disable BYOK + reset TTLs. After Reconfigure, BYOK tier is off.
	c.Reconfigure(false, false, false, time.Minute, time.Second)
	res, err := c.Execute(context.Background(), summarizechain.Request{Text: "hi", BYOKProvider: "x", BYOKKey: "k"})
	require.Nil(t, res)
	require.Error(t, err)
	require.ErrorIs(t, err, summarizechain.ErrAllProvidersFailed)
}
