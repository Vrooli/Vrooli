package corpusgen

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseQuery_StripsLabelsQuotesPunctuation(t *testing.T) {
	cases := map[string]string{
		"restart the api service":           "restart the api service",
		"Query: restart the api service":    "restart the api service",
		"\"restart the api service\"":       "restart the api service",
		"restart the api service?":          "restart the api service",
		"\n\n  how do i list scenarios  \n": "how do i list scenarios",
		"A: find the logs":                  "find the logs",
	}
	for in, want := range cases {
		require.Equal(t, want, parseQuery(in), "input %q", in)
	}
}

func TestParseQuery_FirstNonEmptyLine(t *testing.T) {
	require.Equal(t, "the real query", parseQuery("\n  \nthe real query\nignored second line"))
	require.Equal(t, "", parseQuery("   \n  \n "))
}

// stubGen returns a fixed gateway-envelope reply, asserting the prompt shape.
func stubGen(reply string, seen *string) generateFn {
	return func(ctx context.Context, model, prompt string, maxTokens int) ([]byte, error) {
		if seen != nil {
			*seen = prompt
		}
		return []byte(fmt.Sprintf(`{"response":%q}`, reply)), nil
	}
}

func TestOllamaInverter_PositiveUnwrapsAndCleans(t *testing.T) {
	var prompt string
	inv := &OllamaInverter{role: "classify.routing", maxTokens: 64, generate: stubGen("<think></think>\nQuery: how to restart the api", &prompt)}
	q, err := inv.InvertPositive(context.Background(), Item{ID: "x", Title: "Restart API command", Type: "command"})
	require.NoError(t, err)
	require.Equal(t, "how to restart the api", q)
	require.Contains(t, prompt, "Restart API command", "the item text is in the prompt")
	require.Contains(t, prompt, "/no_think")
}

func TestOllamaInverter_NegativePromptDiffersFromPositive(t *testing.T) {
	var pos, neg string
	invP := &OllamaInverter{role: "classify.routing", maxTokens: 64, generate: stubGen("q", &pos)}
	invN := &OllamaInverter{role: "classify.routing", maxTokens: 64, generate: stubGen("q", &neg)}
	_, _ = invP.InvertPositive(context.Background(), Item{ID: "x", Title: "t", Type: "command"})
	_, _ = invN.InvertNegative(context.Background(), Item{ID: "x", Title: "t", Type: "command"})
	require.NotEqual(t, pos, neg)
	require.Contains(t, strings.ToLower(neg), "negative")
}

func TestOllamaInverter_EmptyReplyIsError(t *testing.T) {
	inv := &OllamaInverter{role: "classify.routing", maxTokens: 64, generate: stubGen("   ", nil)}
	_, err := inv.InvertPositive(context.Background(), Item{ID: "x", Title: "t"})
	require.Error(t, err, "a model that returns no usable query is an error, not a silent empty case")
}

func TestNewOllamaInverter_DefaultModel(t *testing.T) {
	t.Setenv("SEARCH_HUB_CORPUSGEN_ROLE", "")
	require.Equal(t, defaultInverterRole, NewOllamaInverter().role)
	t.Setenv("SEARCH_HUB_CORPUSGEN_ROLE", "custom.role")
	require.Equal(t, "custom.role", NewOllamaInverter().role)
}
