package livesearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"web-search/internal/httpc"
)

// DefaultOllamaURL is the Ollama base URL used when OLLAMA_URL is unset.
const DefaultOllamaURL = "http://localhost:11434"

// DefaultSynthesisModel is the chat model used for L1 snippet synthesis.
const DefaultSynthesisModel = "llama3.2:3b"

// abstainNote is the explicit text emitted when synthesis abstains.
const abstainNote = "sources insufficient or disagree"

// defaultSynthesisTimeout bounds a single L1 synthesis round-trip.
const defaultSynthesisTimeout = 30 * time.Second

// Synthesizer is the L1 seam: an optional LLM pass over the snippets already
// returned by L0. It NEVER fetches new content — it summarizes the snippets in
// hand and grounds every claim with a Citation, or abstains. Tests inject a
// fake; production wraps the Ollama chat client.
type Synthesizer interface {
	// Synthesize summarizes results into a cited answer for query. It returns
	// an abstaining Synthesis (Abstained=true) when the snippets are
	// insufficient or in conflict rather than fabricating.
	Synthesize(ctx context.Context, query string, results []Result) (*Synthesis, error)
}

// OllamaSynthesizer is the production Synthesizer: it POSTs a constrained chat
// prompt to {BaseURL}/api/chat and parses a strict JSON reply into a cited (or
// abstaining) Synthesis. It is snippet-only: the prompt carries the returned
// snippets and forbids the model from using outside knowledge.
type OllamaSynthesizer struct {
	BaseURL string
	Model   string
	Doer    httpc.Doer
	Timeout time.Duration
}

// NewOllamaSynthesizer constructs an Ollama-backed synthesizer. Empty baseURL/
// model fall back to the defaults; a nil doer falls back to a timeout-bounded
// *http.Client.
func NewOllamaSynthesizer(baseURL, model string, doer httpc.Doer) *OllamaSynthesizer {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = DefaultOllamaURL
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = DefaultSynthesisModel
	}
	if doer == nil {
		doer = &http.Client{Timeout: defaultSynthesisTimeout}
	}
	return &OllamaSynthesizer{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   model,
		Doer:    doer,
		Timeout: defaultSynthesisTimeout,
	}
}

// chatMessage is one Ollama /api/chat message.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest is the Ollama /api/chat request body. stream is false so the
// reply arrives as a single JSON object.
type chatRequest struct {
	Model    string             `json:"model"`
	Messages []chatMessage      `json:"messages"`
	Stream   bool               `json:"stream"`
	Format   string             `json:"format"`
	Options  map[string]float64 `json:"options"`
}

// chatResponse is the Ollama /api/chat response envelope (non-streamed).
type chatResponse struct {
	Message chatMessage `json:"message"`
}

// synthesisReply is the strict JSON shape the model is asked to emit.
type synthesisReply struct {
	// Abstained is true when the snippets do not support a grounded answer.
	Abstained bool `json:"abstained"`
	// Text is the synthesized answer (empty when abstained).
	Text string `json:"text"`
	// Citations lists the zero-based result indices each claim is grounded in.
	Citations []int `json:"citations"`
}

// Synthesize runs the L1 pass over the returned snippets.
func (s *OllamaSynthesizer) Synthesize(ctx context.Context, query string, results []Result) (*Synthesis, error) {
	if len(results) == 0 {
		return abstain(), nil
	}
	if s.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.Timeout)
		defer cancel()
	}

	body := chatRequest{
		Model: s.Model,
		Messages: []chatMessage{
			{Role: "system", Content: synthesisSystemPrompt},
			{Role: "user", Content: buildSynthesisPrompt(query, results)},
		},
		Stream:  false,
		Format:  "json",
		Options: map[string]float64{"temperature": 0},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("livesearch: marshal synthesis request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("livesearch: build synthesis request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("livesearch: synthesis request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("livesearch: synthesis status %d", resp.StatusCode)
	}

	var env chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("livesearch: decode synthesis response: %w", err)
	}
	return parseSynthesisReply(env.Message.Content, results), nil
}

// abstain returns the canonical abstaining synthesis.
func abstain() *Synthesis {
	return &Synthesis{Text: abstainNote, Abstained: true}
}

const synthesisSystemPrompt = "You summarize web-search snippets into a short, factual answer. " +
	"Use ONLY the provided snippets; never add outside knowledge. " +
	"Ground every claim in the snippets you used by listing their indices. " +
	"If the snippets are insufficient or disagree, abstain instead of guessing."

// buildSynthesisPrompt renders the snippet-only synthesis prompt. Each snippet
// is numbered so the model can cite by index.
func buildSynthesisPrompt(query string, results []Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Question: %s\n\nSnippets:\n", query)
	for i, r := range results {
		fmt.Fprintf(&b, "[%d] %s — %s\n", i, r.Title, r.Snippet)
	}
	b.WriteString("\nOutput ONLY one JSON object, no prose:\n")
	b.WriteString(`{"abstained":<bool>,"text":"<answer or empty>","citations":[<result indices used>]}` + "\n")
	b.WriteString("- Set abstained=true (and text empty) when the snippets are insufficient or disagree.\n")
	b.WriteString("- citations must reference only the snippet indices above.\n")
	return b.String()
}

// parseSynthesisReply decodes the model's JSON, enforces the always-cited /
// abstain contract, and maps citation indices back to results. An unparseable
// or uncited reply abstains rather than surfacing an ungrounded answer.
func parseSynthesisReply(raw string, results []Result) *Synthesis {
	obj := firstJSONObject(raw)
	if obj == "" {
		return abstain()
	}
	var reply synthesisReply
	if err := json.Unmarshal([]byte(obj), &reply); err != nil {
		return abstain()
	}
	if reply.Abstained || strings.TrimSpace(reply.Text) == "" {
		return abstain()
	}

	cites := make([]Citation, 0, len(reply.Citations))
	seen := make(map[int]bool, len(reply.Citations))
	for _, idx := range reply.Citations {
		if idx < 0 || idx >= len(results) || seen[idx] {
			continue
		}
		seen[idx] = true
		cites = append(cites, Citation{
			ResultIndex: idx,
			URL:         results[idx].URL,
			Title:       results[idx].Title,
		})
	}
	// Always-cited contract: a claim with no valid grounding is treated as a
	// fabrication and abstains.
	if len(cites) == 0 {
		return abstain()
	}
	return &Synthesis{Text: strings.TrimSpace(reply.Text), Citations: cites}
}

// firstJSONObject returns the first balanced {…} object in s, ignoring braces
// inside string literals. Empty string when none is found.
func firstJSONObject(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	depth, inStr, escaped := 0, false, false
	for i := start; i < len(s); i++ {
		c := s[i]
		switch {
		case escaped:
			escaped = false
		case c == '\\' && inStr:
			escaped = true
		case c == '"':
			inStr = !inStr
		case inStr:
			// inside a string literal: ignore structural chars
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// Compile-time guarantee that the production synthesizer satisfies the seam.
var _ Synthesizer = (*OllamaSynthesizer)(nil)
