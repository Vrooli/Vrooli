package domains

import "strings"

// Glossary is a constant-time lookup of (domain, symbol) declared in the
// derived domain map. The symbol-glossary signal asks "is this symbol
// declared by domain D?" against the index rather than rescanning the map
// on every call. Built once per derived map via BuildGlossary.
type Glossary struct {
	byDomain map[string]map[string]struct{}
}

// BuildGlossary constructs the lookup index from a derived domain map.
func BuildGlossary(m DerivedDomainMap) Glossary {
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
// glossary. Case-insensitive; empty domain or symbol returns false.
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

// DomainsFor returns the domain names whose glossary contains the given
// symbol, in alphabetical order.
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
