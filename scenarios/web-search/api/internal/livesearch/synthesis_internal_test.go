package livesearch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func synthResults() []Result {
	return []Result{
		{URL: "https://a.com", Title: "A", Snippet: "Claude is made by Anthropic."},
		{URL: "https://b.com", Title: "B", Snippet: "Claude is made by OpenAI."},
	}
}

func TestParseSynthesisReplyCited(t *testing.T) {
	raw := `{"abstained":false,"text":"Anthropic makes Claude.","citations":[0]}`
	syn := parseSynthesisReply(raw, synthResults())
	require.False(t, syn.Abstained)
	require.Equal(t, "Anthropic makes Claude.", syn.Text)
	require.Len(t, syn.Citations, 1)
	require.Equal(t, 0, syn.Citations[0].ResultIndex)
	require.Equal(t, "https://a.com", syn.Citations[0].URL)
	require.Equal(t, "A", syn.Citations[0].Title)
}

func TestParseSynthesisReplyAbstainsOnConflict(t *testing.T) {
	// Model reports the sources disagree → abstain, never fabricate.
	raw := `{"abstained":true,"text":"","citations":[]}`
	syn := parseSynthesisReply(raw, synthResults())
	require.True(t, syn.Abstained)
	require.Equal(t, abstainNote, syn.Text)
	require.Empty(t, syn.Citations)
}

func TestParseSynthesisReplyAbstainsWhenUncited(t *testing.T) {
	// A non-abstained answer with no VALID citation is an ungrounded claim and
	// is downgraded to an abstention (always-cited contract).
	raw := `{"abstained":false,"text":"Some claim.","citations":[99]}`
	syn := parseSynthesisReply(raw, synthResults())
	require.True(t, syn.Abstained)
	require.Empty(t, syn.Citations)
}

func TestParseSynthesisReplyAbstainsOnGarbage(t *testing.T) {
	syn := parseSynthesisReply("not json at all", synthResults())
	require.True(t, syn.Abstained)
}

func TestParseSynthesisReplyDeduplicatesCitations(t *testing.T) {
	raw := `{"abstained":false,"text":"x","citations":[0,0,1,1]}`
	syn := parseSynthesisReply(raw, synthResults())
	require.False(t, syn.Abstained)
	require.Len(t, syn.Citations, 2)
}
