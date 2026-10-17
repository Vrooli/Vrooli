package generation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// stubProvider is a minimal Provider for chain-ordering tests.
type stubProvider struct {
	name      string
	available bool
	text      string
	textErr   error
}

func (s stubProvider) Name() string                   { return s.name }
func (s stubProvider) Available(context.Context) bool { return s.available }
func (s stubProvider) GenerateText(context.Context, TextRequest) (TextResponse, error) {
	if s.textErr != nil {
		return TextResponse{}, s.textErr
	}
	return TextResponse{Text: s.text, Provider: s.name}, nil
}

func TestChain_SkipsUnavailableAndUsesFirstSuccess(t *testing.T) {
	chain := NewChain(
		stubProvider{name: "ollama", available: false, text: "from-ollama"},
		stubProvider{name: "openrouter", available: true, text: "from-openrouter"},
	)
	resp, err := chain.GenerateText(context.Background(), TextRequest{Prompt: "hi"})
	require.NoError(t, err)
	require.Equal(t, "from-openrouter", resp.Text)
	require.Equal(t, "openrouter", resp.Provider)
}

func TestChain_AllUnavailableIsError(t *testing.T) {
	chain := NewChain(stubProvider{name: "ollama", available: false})
	_, err := chain.GenerateText(context.Background(), TextRequest{Prompt: "hi"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "all providers failed")
}

func TestChain_AvailableReflectsAnyProvider(t *testing.T) {
	require.False(t, NewChain(stubProvider{available: false}).Available(context.Background()))
	require.True(t, NewChain(stubProvider{available: false}, stubProvider{available: true}).Available(context.Background()))
}

func TestChain_StatusesReportEachProvider(t *testing.T) {
	chain := NewChain(
		stubProvider{name: "ollama", available: true},
		stubProvider{name: "openrouter", available: false},
	)
	statuses := chain.Statuses(context.Background())
	require.Len(t, statuses, 2)
	require.Equal(t, ProviderStatus{Name: "ollama", Available: true}, statuses[0])
	require.Equal(t, ProviderStatus{Name: "openrouter", Available: false}, statuses[1])
}

func TestColorPrompt_IncludesContextAndJSONShape(t *testing.T) {
	p := colorPrompt("Acme", "a fintech", "playful")
	require.Contains(t, p, "Acme")
	require.Contains(t, p, "a fintech")
	require.Contains(t, p, "playful")
	require.Contains(t, p, `"primary"`)
	require.Contains(t, p, "WCAG AA")
}

func TestVoicePrompt_OmitsEmptyOptionalContext(t *testing.T) {
	p := voicePrompt("Acme", "", "")
	require.Contains(t, p, "Acme")
	require.NotContains(t, p, "Description:")
	require.NotContains(t, p, "Notes:")
}

func TestLogoPrompt_AppendsPrimaryColorWhenSet(t *testing.T) {
	require.Contains(t, logoPrompt("Acme", "", "#112233"), "#112233")
	require.NotContains(t, logoPrompt("Acme", "", ""), "Primary color:")
}

func TestParseGeneratedJSON_StripsMarkdownFence(t *testing.T) {
	out, err := parseGeneratedJSON("```json\n{\"primary\":\"#112233\"}\n```")
	require.NoError(t, err)
	require.Equal(t, "#112233", out["primary"])
}

func TestParseGeneratedJSON_NarrowsToObjectAmidProse(t *testing.T) {
	out, err := parseGeneratedJSON("Sure! Here it is: {\"tone\":\"bold\"} — hope that helps")
	require.NoError(t, err)
	require.Equal(t, "bold", out["tone"])
}

func TestParseGeneratedJSON_RejectsNonJSON(t *testing.T) {
	_, err := parseGeneratedJSON("I cannot help with that")
	require.Error(t, err)
}

func TestVoiceFromJSON_CollectsStringKeywordsOnly(t *testing.T) {
	v := voiceFromJSON(map[string]any{
		"tone":     "bold",
		"keywords": []any{"fast", "", "secure", 42},
	})
	require.Equal(t, "bold", v.Tone)
	require.Equal(t, []string{"fast", "secure"}, v.Keywords)
}
