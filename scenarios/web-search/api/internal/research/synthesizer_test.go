package research_test

import (
	"context"
	"reflect"
	"testing"

	"web-search/internal/research"

	"github.com/stretchr/testify/require"
)

var synthDocs = []research.Document{
	{URL: "https://a.example", Title: "A", Text: "alpha"},
	{URL: "https://b.example", Title: "B", Text: "beta"},
}

func TestParseSynthesisReplyCited(t *testing.T) {
	raw := `{"abstained":false,"text":"alpha and beta","citations":[0,1]}`
	got := research.ParseSynthesisReply(raw, synthDocs)
	require.False(t, got.Abstained)
	require.Equal(t, "alpha and beta", got.Text)
	require.Len(t, got.Citations, 2)
	require.Equal(t, "https://a.example", got.Citations[0].URL)
}

func TestParseSynthesisReplyAbstains(t *testing.T) {
	got := research.ParseSynthesisReply(`{"abstained":true,"text":"","citations":[]}`, synthDocs)
	require.True(t, got.Abstained)
}

// TestParseSynthesisReplyUncitedAbstains pins the always-cited contract: a
// non-abstained reply with NO valid citation is treated as a fabrication and
// abstains rather than surfacing an ungrounded claim.
func TestParseSynthesisReplyUncitedAbstains(t *testing.T) {
	got := research.ParseSynthesisReply(`{"abstained":false,"text":"made up","citations":[]}`, synthDocs)
	require.True(t, got.Abstained)
}

// TestParseSynthesisReplyDropsOutOfRangeCitations pins that out-of-range / dup
// citation indices are dropped; if none remain valid it abstains.
func TestParseSynthesisReplyDropsOutOfRangeCitations(t *testing.T) {
	got := research.ParseSynthesisReply(`{"abstained":false,"text":"x","citations":[5,5]}`, synthDocs)
	require.True(t, got.Abstained, "no valid citation survives -> abstain")

	got = research.ParseSynthesisReply(`{"abstained":false,"text":"x","citations":[0,9,0]}`, synthDocs)
	require.False(t, got.Abstained)
	require.Len(t, got.Citations, 1, "dedup + drop out-of-range leaves one citation")
}

func TestParseSynthesisReplyUnparseableAbstains(t *testing.T) {
	require.True(t, research.ParseSynthesisReply("not json at all", synthDocs).Abstained)
}

func TestSynthesizeUsesGatewayRole(t *testing.T) {
	var gotArgs []string
	var gotStdin string
	s := research.NewOllamaSynthesizer("summarize.default")
	s.Runner = func(_ context.Context, args []string, stdin string) ([]byte, error) {
		gotArgs = append([]string(nil), args...)
		gotStdin = stdin
		return []byte(`{"response":"{\"abstained\":false,\"text\":\"x\",\"citations\":[0]}"}`), nil
	}

	syn, err := s.Synthesize(context.Background(), "q", synthDocs)
	require.NoError(t, err)
	require.False(t, syn.Abstained)
	require.True(t, reflect.DeepEqual(gotArgs, []string{"gateway", "generate", "--role", "summarize.default", "--json", "--temperature", "0", "--prompt-stdin"}))
	require.Contains(t, gotStdin, "Use ONLY the provided documents")
	require.Contains(t, gotStdin, "Question: q")
}
