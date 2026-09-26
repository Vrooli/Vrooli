package archetype

import "strings"

// Name is a canonical domain archetype from the fixed fleet vocabulary. It is
// the single SSOT for what a domain's architectural role may be; the proto
// enum architecture-cartographer.v1.domains.Archetype mirrors it and the
// handler maps between the two. Declared archetypes (DOMAINS.md) and inferred
// archetypes (code shape) are both normalized to this set so convergence can
// compare them; a declared label that does not map is preserved verbatim and
// reported as drift rather than silently coerced.
type Name string

const (
	Reporting      Name = "reporting"
	Service        Name = "service"
	Mutation       Name = "mutation"
	Classification Name = "classification"
	Orchestration  Name = "orchestration"
	Scoring        Name = "scoring"
	Query          Name = "query"
)

// canonical is the ordered fixed vocabulary plus accepted aliases that map onto
// it. Aliases keep historical DOMAINS.md wording mapping cleanly to the SSOT
// without widening the vocabulary.
var canonical = map[string]Name{
	"reporting":      Reporting,
	"report":         Reporting,
	"service":        Service,
	"mutation":       Mutation,
	"apply":          Mutation,
	"classification": Classification,
	"classify":       Classification,
	"orchestration":  Orchestration,
	"orchestrator":   Orchestration,
	"scoring":        Scoring,
	"score":          Scoring,
	"query":          Query,
}

// All returns the canonical vocabulary in declaration order.
func All() []Name {
	return []Name{Reporting, Service, Mutation, Classification, Orchestration, Scoring, Query}
}

// Normalize maps a free-text archetype label onto the canonical vocabulary.
// ok is false when the label has no canonical mapping (a drift signal).
func Normalize(label string) (Name, bool) {
	n, ok := canonical[strings.ToLower(strings.TrimSpace(label))]
	return n, ok
}

// IsCanonical reports whether a label maps onto the canonical vocabulary.
func IsCanonical(label string) bool {
	_, ok := Normalize(label)
	return ok
}
