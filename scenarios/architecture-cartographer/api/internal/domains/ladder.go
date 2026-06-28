package domains

import (
	"context"
	"sort"
	"time"

	"architecture-cartographer/internal/archetype"
)

// RunLadder runs each extractor against the scenario directory in trust
// order and returns the extractions in that same order. An extractor that
// errors aborts the ladder (a malformed present source is a hard error);
// an absent source contributes an empty extraction.
func RunLadder(ctx context.Context, scenarioDir string, extractors []DomainSourceExtractor) ([]Extraction, error) {
	out := make([]Extraction, 0, len(extractors))
	scenario := scenarioNameFromDir(scenarioDir)
	for _, ex := range extractors {
		extraction, err := ex.Extract(ctx, scenarioDir)
		if err != nil {
			return nil, err
		}
		extraction.Source = ex.Source()
		extraction = rebaseExtractionToRepoRoot(scenario, extraction)
		out = append(out, extraction)
	}
	return out, nil
}

// Resolve collapses the ordered extractions into one canonical
// DerivedDomainMap. The first extraction (in trust order) that declares at
// least one domain becomes the authority and defines the expected domain
// set and each domain's paths/glossary/archetype. Lower rungs only
// contribute provenance and the per-source declaration list (the input to
// convergence reporting). SharedSubstrate / NonDomains come from whichever
// extraction supplies them (currently only the DOMAINS.md rung).
//
// Resolve is pure: it does not touch the filesystem or clock — callers
// pass derivedAt so output is deterministic in tests.
func Resolve(scenario string, extractions []Extraction, derivedAt time.Time) (DerivedDomainMap, error) {
	m := DerivedDomainMap{Scenario: scenario, DerivedAt: derivedAt}

	// authorityIdx is the first non-advisory extraction with any domain.
	// Advisory rungs (UI features) contribute provenance + convergence
	// input but never define the expected set.
	authorityIdx := -1
	for i, ex := range extractions {
		if ex.Source.Advisory() {
			continue
		}
		if len(ex.Domains) > 0 {
			authorityIdx = i
			break
		}
	}
	if authorityIdx < 0 {
		return DerivedDomainMap{}, ErrNoAuthority{Scenario: scenario}
	}
	authority := extractions[authorityIdx]
	m.Authority = authority.Source
	if authority.Source == SourceDomainsDoc || authority.Source == SourceAPIManifest {
		m.AuthorityConfidence = ConfidenceHigh
	} else {
		m.AuthorityConfidence = ConfidenceLow
	}

	// Index every source's domain names for provenance lookup.
	declaredBy := map[string][]Source{}
	for _, ex := range extractions {
		for _, d := range ex.Domains {
			declaredBy[d.Name] = append(declaredBy[d.Name], ex.Source)
		}
	}

	for _, d := range authority.Domains {
		prov := append([]Source(nil), declaredBy[d.Name]...)
		sortSources(prov)
		archetypes := cloneArchetypes(d.Archetypes)
		archetypes = appendInferredArchetypes(archetypes, archetype.Infer(archetype.Input{
			Name:  d.Name,
			Paths: d.Paths,
		})...)
		m.Domains = append(m.Domains, DerivedDomain{
			Name:            d.Name,
			Paths:           append([]string(nil), d.Paths...),
			Glossary:        append([]string(nil), d.Glossary...),
			Responsibility:  d.Responsibility,
			Purpose:         d.Purpose,
			OwnsData:        d.OwnsData,
			SecondaryTraits: append([]string(nil), d.SecondaryTraits...),
			Archetypes:      archetypes,
			Provenance:      prov,
		})
	}
	sort.Slice(m.Domains, func(i, j int) bool { return m.Domains[i].Name < m.Domains[j].Name })

	// Shared substrate / non-domains: prefer the authority's, else any rung
	// that supplies them.
	for _, ex := range extractions {
		if len(ex.SharedSubstrate) > 0 && m.SharedSubstrate == nil {
			m.SharedSubstrate = append([]string(nil), ex.SharedSubstrate...)
		}
		if len(ex.NonDomains) > 0 && m.NonDomains == nil {
			m.NonDomains = append([]string(nil), ex.NonDomains...)
		}
	}

	// Forward extractor warnings (e.g. DOMAINS.md row drops) so the
	// audit pipeline can surface them as parse-warning conflicts.
	for _, ex := range extractions {
		if len(ex.Warnings) > 0 {
			m.Warnings = append(m.Warnings, ex.Warnings...)
		}
	}

	// Per-source declarations, in trust order.
	for i, ex := range extractions {
		names := make([]string, 0, len(ex.Domains))
		for _, d := range ex.Domains {
			names = append(names, d.Name)
		}
		sort.Strings(names)
		m.Declarations = append(m.Declarations, DomainDeclaration{
			Source:        ex.Source,
			DomainNames:   names,
			Authoritative: i == authorityIdx,
		})
	}

	return m, nil
}

func cloneArchetypes(in []DomainArchetype) []DomainArchetype {
	if len(in) == 0 {
		return nil
	}
	out := make([]DomainArchetype, 0, len(in))
	for _, archetype := range in {
		archetype.Evidence = append([]string(nil), archetype.Evidence...)
		out = append(out, archetype)
	}
	return out
}

func appendInferredArchetypes(in []DomainArchetype, inferred ...archetype.Result) []DomainArchetype {
	seen := make(map[string]struct{}, len(in)+len(inferred))
	for _, existing := range in {
		seen[string(existing.Source)+"\x00"+existing.Name] = struct{}{}
	}
	out := append([]DomainArchetype(nil), in...)
	for _, result := range inferred {
		name := result.Name
		if name == "" {
			continue
		}
		key := string(ArchetypeSourceInferred) + "\x00" + name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, DomainArchetype{
			Name:       name,
			Source:     ArchetypeSourceInferred,
			Confidence: result.Confidence,
			Evidence:   append([]string(nil), result.Evidence...),
		})
	}
	return out
}

func sortSources(s []Source) {
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
}
