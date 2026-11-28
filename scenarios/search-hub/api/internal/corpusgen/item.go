package corpusgen

import "strings"

// Item is one sampled index document the generator inverts into a query. It is a
// transport-free projection of whatever the provider's search/status contract
// surfaces — id + a little text + the coverage facets the sampler could read —
// so corpusgen holds no provider-specific knowledge. The production sampler
// (handlers/eval) fills these from the unified SearchHit; a fake fills them in
// tests.
type Item struct {
	// The provider id_field value — what a generated positive case expects.
	ID string
	// Human-readable title/snippet, the only signal the inverter reasons over.
	Title   string
	Snippet string
	// Coverage facets, when the contract exposes them. Type is the provider's
	// --type token (e.g. "command"); Group is the provider_group / origin. They
	// form the Stratum used to stratify the sample and tag generated cases so
	// adequacy can measure index coverage.
	Type  string
	Group string
}

// Stratum is the coverage bucket the item falls in: "type:<type>" plus, when
// known, "/<group>". A leaf with one type and no group collapses to a single
// stratum (honest — the search contract simply exposes no finer facet there).
func (it Item) Stratum() string {
	t := strings.TrimSpace(it.Type)
	if t == "" {
		t = "unknown"
	}
	s := "type:" + t
	if g := strings.TrimSpace(it.Group); g != "" {
		s += "/" + g
	}
	return s
}

// text is the inversion signal: title, falling back to the snippet, falling back
// to the id. Empty only when the item is empty.
func (it Item) text() string {
	if t := strings.TrimSpace(it.Title); t != "" {
		return t
	}
	if s := strings.TrimSpace(it.Snippet); s != "" {
		return s
	}
	return strings.TrimSpace(it.ID)
}
