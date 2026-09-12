package livesearch

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

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

// TestSynthesizeAbstainsOnZeroResults pins the thin-coverage abstention: below
// the minimum snippet threshold (no snippets at all) the synthesizer returns an
// explicit abstain — with the canonical insufficient-sources note — and never
// even reaches the LLM.
func TestSynthesizeAbstainsOnZeroResults(t *testing.T) {
	s := NewOllamaSynthesizer("")
	s.Runner = func(context.Context, []string, string) ([]byte, error) {
		t.Fatal("unexpected gateway call: synthesis below the snippet threshold must abstain without calling the LLM")
		return nil, nil
	}

	syn, err := s.Synthesize(context.Background(), "anything", nil)
	require.NoError(t, err)
	require.True(t, syn.Abstained, "thin coverage (zero snippets) must abstain")
	require.Equal(t, abstainNote, syn.Text)
	require.Empty(t, syn.Citations)
}

// blockingRunner parks until the request context is cancelled, simulating a hung
// LLM backend. It returns the context error so the deadline is observable.
// TestSynthesizeHonorsTimeout is the bounded-latency guard for L1: synthesis is
// deadline-bounded by the synthesizer's Timeout, so a hung LLM cannot stall the
// search path indefinitely (the service then returns raw L0 results unharmed —
// see TestServiceSynthesisFailureDoesNotBlockResults).
func TestSynthesizeHonorsTimeout(t *testing.T) {
	s := NewOllamaSynthesizer("summarize.default")
	s.Runner = func(ctx context.Context, _ []string, _ string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	s.Timeout = 50 * time.Millisecond

	start := time.Now()
	_, err := s.Synthesize(context.Background(), "q", synthResults())
	elapsed := time.Since(start)

	require.Error(t, err, "a hung backend must surface as an error, not block")
	require.Less(t, elapsed, 2*time.Second, "synthesis must return within its configured deadline, not hang")
}

func TestSynthesizeUsesGatewayRole(t *testing.T) {
	var gotArgs []string
	var gotStdin string
	s := NewOllamaSynthesizer("summarize.default")
	s.Runner = func(_ context.Context, args []string, stdin string) ([]byte, error) {
		gotArgs = append([]string(nil), args...)
		gotStdin = stdin
		return []byte(`{"response":"{\"abstained\":false,\"text\":\"x\",\"citations\":[0]}"}`), nil
	}

	syn, err := s.Synthesize(context.Background(), "q", synthResults())
	require.NoError(t, err)
	require.False(t, syn.Abstained)
	require.True(t, reflect.DeepEqual(gotArgs, []string{"gateway", "generate", "--role", "summarize.default", "--json", "--temperature", "0", "--prompt-stdin"}), fmt.Sprintf("args = %v", gotArgs))
	require.Contains(t, gotStdin, "Use ONLY the provided snippets")
	require.Contains(t, gotStdin, "Question: q")
}

func TestDefaultSynthesisRole(t *testing.T) {
	require.Equal(t, "summarize.default", DefaultSynthesisRole)
}
