package corpusgen

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"search-hub/internal/aigateway"
	"search-hub/internal/ollama"
)

// Inverter turns a sampled index item into a candidate query. It is the LLM seam
// of corpusgen: InvertPositive produces a natural-language query that SHOULD
// retrieve the item (the heart of query inversion); InvertNegative produces a
// plausible-but-unanswerable query in the same domain (a hard negative). Tests
// inject a deterministic fake; production uses AI Gateway's local-only typed
// inference boundary.
type Inverter interface {
	InvertPositive(ctx context.Context, it Item) (query string, err error)
	InvertNegative(ctx context.Context, it Item) (query string, err error)
}

const defaultInverterRole = "classify.fast"

// inverterMaxTokens caps generation — one short query line needs little room.
const inverterMaxTokens = 128

// generateFn is the gateway seam (mirrors routing's): tests inject a runner
// instead of shelling the real model.
type generateFn = aigateway.TextGenerator

const corpusgenPrivacySource = "search_hub.corpus_generation.sampled_index_item_metadata"

// GatewayInverter is the production Inverter: it asks the local role, for a
// sampled item, "what would a user type to find THIS?" (positive) or "what
// plausible query in this area has no good answer here?" (negative). It reads
// only the item's own title/snippet/type, so it holds no provider-specific
// knowledge.
type GatewayInverter struct {
	role      string
	maxTokens int
	generate  generateFn
}

// NewGatewayInverter sends the confidential corpus sample through AI Gateway's
// local-only typed request path. Role/model selection remains policy-owned.
func NewGatewayInverter() *GatewayInverter {
	return newGatewayInverter(func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
	}, &http.Client{Timeout: 90 * time.Second})
}

func newGatewayInverter(resolve aigateway.URLResolver, client connect.HTTPClient) *GatewayInverter {
	return &GatewayInverter{
		role:      defaultInverterRole,
		maxTokens: inverterMaxTokens,
		generate:  aigateway.NewTextGenerator(resolve, client, corpusgenPrivacySource),
	}
}

// InvertPositive asks for one natural-language query that retrieves the item.
func (o *GatewayInverter) InvertPositive(ctx context.Context, it Item) (string, error) {
	return o.ask(ctx, buildPositivePrompt(it))
}

// InvertNegative asks for one plausible query in the same area that the item
// does NOT answer (a hard negative).
func (o *GatewayInverter) InvertNegative(ctx context.Context, it Item) (string, error) {
	return o.ask(ctx, buildNegativePrompt(it))
}

func (o *GatewayInverter) ask(ctx context.Context, prompt string) (string, error) {
	raw, err := o.generate(ctx, o.role, prompt, o.maxTokens)
	if err != nil {
		return "", fmt.Errorf("AI Gateway generate: %w", err)
	}
	q := parseQuery(ollama.StripThink(ollama.UnwrapResponse(raw)))
	if q == "" {
		return "", fmt.Errorf("model returned no usable query")
	}
	return q, nil
}

// parseQuery extracts a single clean query line from the model's reply: the
// first non-empty line, with surrounding quotes / a leading "Query:" label /
// trailing punctuation stripped. The inverter asks for exactly one line, but
// small models sometimes add a label or quotes — this normalizes those away.
func parseQuery(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Drop a leading "Query:"/"Q:"/"A:" label if present.
		if i := strings.Index(line, ":"); i >= 0 && i <= 6 {
			label := strings.ToLower(strings.TrimSpace(line[:i]))
			switch label {
			case "query", "q", "a", "answer", "search":
				line = strings.TrimSpace(line[i+1:])
			}
		}
		line = strings.Trim(line, "\"'`")
		line = strings.TrimRight(line, ".?! ")
		if line != "" {
			return line
		}
	}
	return ""
}

func buildPositivePrompt(it Item) string {
	var b strings.Builder
	b.WriteString("You are building a search-quality test set. Given one item from a search index, ")
	b.WriteString("write the single most natural search query a real user would type to find THIS item.\n\n")
	if t := strings.TrimSpace(it.Type); t != "" {
		fmt.Fprintf(&b, "Item type: %s\n", t)
	}
	fmt.Fprintf(&b, "Item: %s\n", it.text())
	if s := strings.TrimSpace(it.Snippet); s != "" && s != it.text() {
		fmt.Fprintf(&b, "Detail: %s\n", s)
	}
	b.WriteString("\nRules:\n")
	b.WriteString("- Output ONLY the query text, one line, no quotes, no label, no explanation.\n")
	b.WriteString("- Write it the way a user would search — natural language, not the item's exact title.\n")
	b.WriteString("- Keep it short (3–10 words).\n\n")
	b.WriteString("/no_think")
	return b.String()
}

func buildNegativePrompt(it Item) string {
	var b strings.Builder
	b.WriteString("You are building hard negatives for a search-quality test set. Given one item from a search index, ")
	b.WriteString("write a single plausible search query that belongs to the SAME general area but that this item does NOT answer ")
	b.WriteString("— so a good search engine should return no strong match.\n\n")
	if t := strings.TrimSpace(it.Type); t != "" {
		fmt.Fprintf(&b, "Item type: %s\n", t)
	}
	fmt.Fprintf(&b, "Item: %s\n", it.text())
	b.WriteString("\nRules:\n")
	b.WriteString("- Output ONLY the query text, one line, no quotes, no label, no explanation.\n")
	b.WriteString("- It must be realistic for the domain but NOT satisfied by the item above.\n")
	b.WriteString("- Keep it short (3–10 words).\n\n")
	b.WriteString("/no_think")
	return b.String()
}
