package livesearch

import (
	"context"
	"net/http"
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

// panicDoer fails the test if any HTTP request is attempted.
type panicDoer struct{ t *testing.T }

func (p panicDoer) Do(*http.Request) (*http.Response, error) {
	p.t.Fatal("unexpected HTTP call: synthesis below the snippet threshold must abstain without calling the LLM")
	return nil, nil
}

// TestSynthesizeAbstainsOnZeroResults pins the thin-coverage abstention: below
// the minimum snippet threshold (no snippets at all) the synthesizer returns an
// explicit abstain — with the canonical insufficient-sources note — and never
// even reaches the LLM.
func TestSynthesizeAbstainsOnZeroResults(t *testing.T) {
	s := NewOllamaSynthesizer("", "", panicDoer{t})

	syn, err := s.Synthesize(context.Background(), "anything", nil)
	require.NoError(t, err)
	require.True(t, syn.Abstained, "thin coverage (zero snippets) must abstain")
	require.Equal(t, abstainNote, syn.Text)
	require.Empty(t, syn.Citations)
}

// blockingDoer parks until the request context is cancelled, simulating a hung
// LLM backend. It returns the context error so the deadline is observable.
type blockingDoer struct{}

func (blockingDoer) Do(req *http.Request) (*http.Response, error) {
	<-req.Context().Done()
	return nil, req.Context().Err()
}

// TestSynthesizeHonorsTimeout is the bounded-latency guard for L1: synthesis is
// deadline-bounded by the synthesizer's Timeout, so a hung LLM cannot stall the
// search path indefinitely (the service then returns raw L0 results unharmed —
// see TestServiceSynthesisFailureDoesNotBlockResults).
func TestSynthesizeHonorsTimeout(t *testing.T) {
	s := NewOllamaSynthesizer("http://synthesis.invalid", "test-model", blockingDoer{})
	s.Timeout = 50 * time.Millisecond

	start := time.Now()
	_, err := s.Synthesize(context.Background(), "q", synthResults())
	elapsed := time.Since(start)

	require.Error(t, err, "a hung backend must surface as an error, not block")
	require.Less(t, elapsed, 2*time.Second, "synthesis must return within its configured deadline, not hang")
}
