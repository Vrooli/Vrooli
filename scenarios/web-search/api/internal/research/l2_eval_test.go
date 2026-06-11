package research_test

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	aisearch "github.com/vrooli/ai-go/search"

	"web-search/internal/livesearch"
	"web-search/internal/research"
	"web-search/internal/research/fetch"
)

// l2EvalCase is one query → expected-claim-substring pair. The answer is
// correct when it contains ANY of the alternates (case-insensitive); the
// alternates are chosen to be stable facts so the eval doesn't rot.
type l2EvalCase struct {
	name     string
	query    string
	expected []string
}

// l2EvalCases mixes the shapes that historically exposed L2 weakness: a
// feature-list question, simple facts, an ambiguous term ("Rust" the game vs
// the language), and answers that tend to live deep in long pages.
var l2EvalCases = []l2EvalCase{
	{"feature-list", "What were the headline features of the Go 1.24 release?", []string{"generic type alias", "tool directive", "swiss", "go.mod"}},
	{"simple-fact-release-year", "What year was Rust programming language version 1.0 released?", []string{"2015"}},
	{"ambiguous-term-game", "When was the survival game Rust by Facepunch first released?", []string{"2013", "2018"}},
	{"deep-page-fact", "What is the default TCP port for PostgreSQL?", []string{"5432"}},
	{"person-fact", "Who created the Python programming language?", []string{"guido", "van rossum"}},
	{"geo-fact", "What is the capital city of Australia?", []string{"canberra"}},
	{"protocol-fact", "What does HTTP status code 418 mean?", []string{"teapot"}},
	{"physics-fact", "What is the speed of light in a vacuum in kilometers per second?", []string{"299,792", "299792", "300,000", "300000"}},
	{"person-fact-2", "Who created the Linux kernel?", []string{"torvalds"}},
	{"chemistry-fact", "What is the chemical symbol for gold?", []string{"au"}},
}

// TestL2AnswerQualityEval is the NON-hermetic L2 answer-quality eval harness
// (plan: web-search-hardening Phase 3 item 5). It drives the PRODUCTION L2
// pipeline — real SearXNG candidates, real HTTP fetches, real Ollama
// synthesis — over ~10 stable query/expected-substring pairs and reports
// answered-correct / answered-wrong / abstained(+reason) counts.
//
// It is NOT part of any required test phase (same attended pattern as
// scripts/live-validate.sh): run it deliberately before and after touching
// the synthesis model, prompt, or excerpting to measure — not vibe — the
// change:
//
//	WEB_SEARCH_L2_EVAL=1 go test ./internal/research/ -run TestL2AnswerQualityEval -v -timeout 30m
//	WEB_SEARCH_L2_EVAL=1 WEB_SEARCH_SYNTH_RELEVANT_EXCERPTS=off go test ...   # positional baseline
//
// The only hard assertion is that at least one case answers correctly (the
// stack works at all); quality movement is read from the logged report.
func TestL2AnswerQualityEval(t *testing.T) {
	if os.Getenv("WEB_SEARCH_L2_EVAL") == "" {
		t.Skip("L2 answer-quality eval is opt-in: set WEB_SEARCH_L2_EVAL=1 (needs network + searxng + ollama)")
	}
	if testing.Short() {
		t.Skip("eval skipped in -short mode")
	}
	if _, err := net.DefaultResolver.LookupHost(context.Background(), "example.com"); err != nil {
		t.Skipf("network unreachable: %v", err)
	}

	searxng := os.Getenv("SEARXNG_URL")
	live := livesearch.NewService(livesearch.Deps{
		Client:   livesearch.NewHTTPSearxngClient(searxng, nil),
		Cache:    livesearch.NewCache(livesearch.DefaultCacheTTL, evalClock{}),
		Governor: livesearch.NewGovernor(livesearch.DefaultGovernorCapacity, livesearch.DefaultGovernorWindow, evalClock{}),
	})

	// Excerpting mode mirrors the production lever: relevance-aware unless
	// WEB_SEARCH_SYNTH_RELEVANT_EXCERPTS=off.
	mode := "relevant"
	var excerpter research.Excerpter = research.RelevantExcerpter{Embedder: aisearch.NewEmbedder("")}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("WEB_SEARCH_SYNTH_RELEVANT_EXCERPTS"))) {
	case "off", "false", "0", "no", "disabled":
		mode = "positional"
		excerpter = research.PositionalExcerpter{}
	}

	svc := research.NewService(research.Deps{
		Searcher: research.LiveSearcher{Service: live},
		// HTTP-only fetch keeps the eval independent of browser-automation-studio.
		Fetcher:     fetch.NewHTTPFetcher(20*time.Second, 0),
		Synthesizer: research.NewOllamaSynthesizer(os.Getenv("OLLAMA_SYNTHESIS_ROLE")),
		Excerpter:   excerpter,
	})

	var correct, wrong, abstained int
	for _, tc := range l2EvalCases {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		out, err := svc.RunL2(ctx, tc.query, 5, false)
		cancel()
		switch {
		case err != nil:
			wrong++
			t.Logf("[ERROR]    %-25s %v", tc.name, err)
		case out.Abstained:
			abstained++
			t.Logf("[ABSTAIN]  %-25s reason=%s", tc.name, out.AbstainReason)
		default:
			answer := strings.ToLower(out.Brief.Summary)
			hit := false
			for _, want := range tc.expected {
				if strings.Contains(answer, strings.ToLower(want)) {
					hit = true
					break
				}
			}
			if hit {
				correct++
				t.Logf("[CORRECT]  %-25s %.120s", tc.name, out.Brief.Summary)
			} else {
				wrong++
				t.Logf("[WRONG]    %-25s wanted any of %v, got: %.200s", tc.name, tc.expected, out.Brief.Summary)
			}
		}
	}

	t.Logf("L2 eval (%s excerpting, role=%s): correct=%d wrong=%d abstained=%d of %d",
		mode, envOr("OLLAMA_SYNTHESIS_ROLE", research.DefaultSynthesisRole), correct, wrong, abstained, len(l2EvalCases))
	if correct == 0 {
		t.Fatalf("zero correct answers — the L2 stack is not functioning (see per-case log above)")
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// evalClock satisfies the livesearch clock seam with real time.
type evalClock struct{}

func (evalClock) Now() time.Time { return time.Now() }
