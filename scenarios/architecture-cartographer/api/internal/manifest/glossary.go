package manifest

import "strings"

// Glossary is a constant-time lookup of (domain, symbol) → declared
// in the manifest. Built once per validated manifest via
// BuildGlossary; the symbol-glossary signal asks "is this symbol
// declared by domain D?" against the index rather than re-scanning the
// manifest on every call.
type Glossary struct {
	// byDomain maps domain -> set of canonicalized symbol names.
	byDomain map[string]map[string]struct{}
}

// BuildGlossary constructs the lookup index from a validated manifest.
func BuildGlossary(m ManifestDefinition) Glossary {
	out := Glossary{byDomain: make(map[string]map[string]struct{}, len(m.Domains))}
	for _, d := range m.Domains {
		set := make(map[string]struct{}, len(d.Glossary))
		for _, term := range d.Glossary {
			set[canonicalizeSymbol(term)] = struct{}{}
		}
		out.byDomain[d.Name] = set
	}
	return out
}

// Match reports whether the symbol is declared in the named domain's
// glossary. The match is case-insensitive (symbols are typically
// type/function names whose canonical casing is preserved in the
// manifest, but signal-side detection may pass identifiers in either
// case). Empty domain or symbol returns false.
func (g Glossary) Match(domain, symbol string) bool {
	if domain == "" || symbol == "" {
		return false
	}
	set, ok := g.byDomain[domain]
	if !ok {
		return false
	}
	_, ok = set[canonicalizeSymbol(symbol)]
	return ok
}

// DomainsFor returns the domain names whose glossary contains the
// given symbol (alphabetical order). Used by the symbol-glossary
// signal when scoring placement.
func (g Glossary) DomainsFor(symbol string) []string {
	sym := canonicalizeSymbol(symbol)
	if sym == "" {
		return nil
	}
	var out []string
	for domain, set := range g.byDomain {
		if _, ok := set[sym]; ok {
			out = append(out, domain)
		}
	}
	// Simple insertion sort — domain lists are small.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

func canonicalizeSymbol(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
