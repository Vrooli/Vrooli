package research

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DefaultSynthesisRole is the Ollama role used for L2 document synthesis.
const DefaultSynthesisRole = "summarize.default"

// abstainNote is the explicit text emitted when synthesis abstains.
const abstainNote = "sources insufficient or disagree"

// defaultSynthesisTimeout bounds a single L2 synthesis round-trip.
const defaultSynthesisTimeout = 60 * time.Second

// maxDocChars is the historical per-document cap on what is sent to the model
// (a single long page must not blow the context window). The excerpting step
// in the L2 pipeline now owns this budget (DefaultExcerptChars aliases it and
// the WEB_SEARCH_SYNTH_EXCERPT_CHARS lever overrides it); buildSynthesisPrompt
// no longer truncates so a raised budget is honored end-to-end.
const maxDocChars = 6000

// Synthesizer is the L2 seam: a single-pass LLM synthesis over the FULL fetched
// page text (not just snippets). It grounds every claim with a Citation back to
// a Document, or abstains. Tests inject a fake; production wraps the Ollama
// gateway CLI.
type Synthesizer interface {
	// Synthesize summarizes docs into a cited answer for query. It returns an
	// abstaining Synthesis (Abstained=true) when the documents are insufficient
	// or in conflict rather than fabricating.
	Synthesize(ctx context.Context, query string, docs []Document) (Synthesis, error)
}

// OllamaSynthesizer is the production Synthesizer: it sends a constrained
// prompt through the shared Ollama gateway and parses a strict JSON reply into
// a cited (or abstaining) Synthesis. It carries the fetched document text and
// forbids the model from using outside knowledge.
type OllamaSynthesizer struct {
	Role    string
	Runner  func(ctx context.Context, args []string, stdin string) ([]byte, error)
	Timeout time.Duration
}

// NewOllamaSynthesizer constructs an Ollama-backed synthesizer. Empty role
// falls back to the default role.
func NewOllamaSynthesizer(role string) *OllamaSynthesizer {
	role = strings.TrimSpace(role)
	if role == "" {
		role = DefaultSynthesisRole
	}
	return &OllamaSynthesizer{
		Role:    role,
		Timeout: defaultSynthesisTimeout,
	}
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
		return AbstainWith(ReasonNoCandidates), nil
	}
	if s.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.Timeout)
		defer cancel()
	}

	role := strings.TrimSpace(s.Role)
	if role == "" {
		role = DefaultSynthesisRole
	}
	prompt := synthesisSystemPrompt + "\n\n" + buildSynthesisPrompt(query, docs)
	out, err := s.run(ctx, []string{"gateway", "generate", "--role", role, "--json", "--temperature", "0", "--prompt-stdin"}, prompt)
	if err != nil {
		return Synthesis{}, fmt.Errorf("research: synthesis request: %w", err)
	}

	var env struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(out))).Decode(&env); err != nil {
		return Synthesis{}, fmt.Errorf("research: decode synthesis response: %w", err)
	}
	return ParseSynthesisReply(env.Response, docs), nil
}

func (s *OllamaSynthesizer) run(ctx context.Context, args []string, stdin string) ([]byte, error) {
	if s.Runner != nil {
		return s.Runner(ctx, args, stdin)
	}
	cmd := exec.CommandContext(ctx, "resource-ollama", args...)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

// Abstain returns the canonical abstaining synthesis with no specific reason.
// Prefer AbstainWith at any site where the collapse cause is known.
func Abstain() Synthesis {
	return Synthesis{Text: abstainNote, Abstained: true}
}

// AbstainWith returns the canonical abstaining synthesis carrying the reason
// for the collapse (observable on the RunL2 response and CLI summary).
func AbstainWith(reason AbstainReason) Synthesis {
	return Synthesis{Text: abstainNote, Abstained: true, AbstainReason: reason}
}

const synthesisSystemPrompt = "You synthesize full web-page text into a short, factual answer. " +
	"Use ONLY the provided documents; never add outside knowledge. " +
	"Ground every claim in the documents you used by listing their indices. " +
	"If the documents are insufficient or disagree, abstain instead of guessing."

// buildSynthesisPrompt renders the document-grounded synthesis prompt. Each
// document is numbered so the model can cite by index. Documents arrive
// pre-excerpted by the pipeline's Excerpter (which owns the per-doc budget),
// so no truncation happens here.
func buildSynthesisPrompt(query string, docs []Document) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Question: %s\n\nDocuments:\n", query)
	for i, d := range docs {
		fmt.Fprintf(&b, "[%d] %s (%s)\n%s\n\n", i, d.Title, d.URL, d.Text)
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
		return AbstainWith(ReasonReplyUnparseable)
	}
	var reply synthesisReply
	if err := json.Unmarshal([]byte(obj), &reply); err != nil {
		return AbstainWith(ReasonReplyUnparseable)
	}
	if reply.Abstained || strings.TrimSpace(reply.Text) == "" {
		return AbstainWith(ReasonModelAbstained)
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
		return AbstainWith(ReasonCitationsInvalid)
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
