package ollama

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnwrapResponse(t *testing.T) {
	// The gateway envelope: the text lives under "response".
	require.Equal(t, "hello", UnwrapResponse([]byte(`{"response":"hello","eval_count":7}`)))
	// A non-envelope (plain text or empty response) is returned as-is, trimmed.
	require.Equal(t, "plain text", UnwrapResponse([]byte("  plain text  ")))
	require.Equal(t, `{"response":""}`, UnwrapResponse([]byte(`{"response":""}`)))
}

func TestStripThink(t *testing.T) {
	require.Equal(t, "\n\nhello", StripThink("<think>reasoning here</think>\n\nhello"))
	require.Equal(t, "abc", StripThink("abc"))
	require.Equal(t, "", StripThink("<think>never closed"))
}

func TestExtractJSONObject_IgnoresBracesInStrings(t *testing.T) {
	require.Equal(t, `{"a":"x}y"}`, ExtractJSONObject(`noise {"a":"x}y"} trailing`))
	require.Equal(t, "", ExtractJSONObject("no object here"))
}
