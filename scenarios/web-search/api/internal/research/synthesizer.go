package research

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

// DefaultSynthesisModel is the chat model used for L2 document synthesis.
const DefaultSynthesisModel = "llama3.2:3b"

// abstainNote is the explicit text emitted when synthesis abstains.
const abstainNote = "sources insufficient or disagree"

// defaultSynthesisTimeout bounds a single L2 synthesis round-trip.
const defaultSynthesisTimeout = 60 * time.Second

// maxDocChars bounds how much of each fetched document is sent to the model so
// a single long page can't blow the context window.
const maxDocChars = 6000

// Synthesizer is the L2 seam: a single-pass LLM synthesis over the FULL fetched
// page text (not just snippets). It grounds every claim with a Citation back to
// a Document, or abstains. Tests inject a fake; production wraps the Ollama chat
// client.
type Synthesizer interface {
	// Synthesize summarizes docs into a cited answer for query. It returns an
	// abstaining Synthesis (Abstained=true) when the documents are insufficient
	// or in conflict rather than fabricating.
	Synthesize(ctx context.Context, query string, docs []Document) (Synthesis, error)
}

// OllamaSynthesizer is the production Synthesizer: it POSTs a constrained chat
// prompt to {BaseURL}/api/chat and parses a strict JSON reply into a cited (or
// abstaining) Synthesis. It carries the fetched document text and forbids the
// model from using outside knowledge.
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

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string             `json:"model"`
	Messages []chatMessage      `json:"messages"`
	Stream   bool               `json:"stream"`
	Format   string             `json:"format"`
	Options  map[string]float64 `json:"options"`
}

type chatResponse struct {
	Message chatMessage `json:"message"`
}

// synthesisReply is the strict JSON shape the model is asked to emit.
type synthesisReply struct {
	Abstained bool   `json:"abstained"`
	Text      string `json:"text"`
	Citations []int  `json:"citations"`
}

// Synthesize runs the single-pass L2 synthesis over the fetched documents.
func (s *OllamaSynthesizer) Synthesize(ctx context.Context, query string, docs []Document) (Synthesis, error) {
	if len(docs) == 0 {
		return Abstain(), nil
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
			{Role: "user", Content: buildSynthesisPrompt(query, docs)},
		},
		Stream:  false,
		Format:  "json",
		Options: map[string]float64{"temperature": 0},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Synthesis{}, fmt.Errorf("research: marshal synthesis request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return Synthesis{}, fmt.Errorf("research: build synthesis request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Doer.Do(req)
	if err != nil {
		return Synthesis{}, fmt.Errorf("research: synthesis request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Synthesis{}, fmt.Errorf("research: synthesis status %d", resp.StatusCode)
	}

	var env chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return Synthesis{}, fmt.Errorf("research: decode synthesis response: %w", err)
	}
	return ParseSynthesisReply(env.Message.Content, docs), nil
}

// Abstain returns the canonical abstaining synthesis.
func Abstain() Synthesis {
	return Synthesis{Text: abstainNote, Abstained: true}
}

const synthesisSystemPrompt = "You synthesize full web-page text into a short, factual answer. " +
	"Use ONLY the provided documents; never add outside knowledge. " +
	"Ground every claim in the documents you used by listing their indices. " +
	"If the documents are insufficient or disagree, abstain instead of guessing."

// buildSynthesisPrompt renders the document-grounded synthesis prompt. Each
// document is numbered so the model can cite by index; long bodies are
// truncated to maxDocChars.
func buildSynthesisPrompt(query string, docs []Document) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Question: %s\n\nDocuments:\n", query)
	for i, d := range docs {
		text := d.Text
		if len(text) > maxDocChars {
			text = text[:maxDocChars]
		}
		fmt.Fprintf(&b, "[%d] %s (%s)\n%s\n\n", i, d.Title, d.URL, text)
	}
	b.WriteString("Output ONLY one JSON object, no prose:\n")
	b.WriteString(`{"abstained":<bool>,"text":"<answer or empty>","citations":[<document indices used>]}` + "\n")
	b.WriteString("- Set abstained=true (and text empty) when the documents are insufficient or disagree.\n")
	b.WriteString("- citations must reference only the document indices above.\n")
	return b.String()
}

// ParseSynthesisReply decodes the model's JSON, enforces the always-cited /
// abstain contract, and maps citation indices back to documents. An unparseable
// or uncited reply abstains rather than surfacing an ungrounded answer.
func ParseSynthesisReply(raw string, docs []Document) Synthesis {
	obj := firstJSONObject(raw)
	if obj == "" {
		return Abstain()
	}
	var reply synthesisReply
	if err := json.Unmarshal([]byte(obj), &reply); err != nil {
		return Abstain()
	}
	if reply.Abstained || strings.TrimSpace(reply.Text) == "" {
		return Abstain()
	}

	cites := make([]Citation, 0, len(reply.Citations))
	seen := make(map[int]bool, len(reply.Citations))
	for _, idx := range reply.Citations {
		if idx < 0 || idx >= len(docs) || seen[idx] {
			continue
		}
		seen[idx] = true
		cites = append(cites, Citation{
			ResultIndex: idx,
			URL:         docs[idx].URL,
			Title:       docs[idx].Title,
		})
	}
	// Always-cited contract: a claim with no valid grounding is treated as a
	// fabrication and abstains.
	if len(cites) == 0 {
		return Abstain()
	}
	return Synthesis{Text: strings.TrimSpace(reply.Text), Citations: cites}
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
