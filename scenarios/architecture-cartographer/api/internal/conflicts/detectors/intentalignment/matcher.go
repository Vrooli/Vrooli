package intentalignment

import (
	"context"
	"log"
	"sort"
	"strings"
	"unicode"

	intent "intent-go"
)

const (
	MatchVocabDrift             = "vocab_drift"
	MatchSemanticCoverageGap    = "semantic_coverage_gap"
	MatchResponsibilityMismatch = "responsibility_mismatch"
)

// Matcher compares adjacent intent rungs and returns normalized mismatches.
type Matcher interface {
	Name() string
	Match(context.Context, MatchInput) ([]Match, error)
}

type MatchInput struct {
	Outcomes              []intent.CapabilityClaim
	RequirementsByOutcome map[string][]intent.CapabilityClaim
	RequirementsByDomain  map[string][]intent.CapabilityClaim
	Domains               []intent.CapabilityClaim
}

type Match struct {
	Type   string
	Domain intent.CapabilityClaim
	Tokens []string
}

// LexicalMatcher implements Tier 1 intent matching with curated domain
// glossary terms. It is deterministic and intentionally conservative:
// domains without requirements or glossary terms are covered by other
// invariants and do not emit vocabulary drift.
type LexicalMatcher struct{}

func (LexicalMatcher) Name() string { return "lexical" }

func (LexicalMatcher) Match(_ context.Context, in MatchInput) ([]Match, error) {
	outcomeByID := map[string]intent.CapabilityClaim{}
	for _, outcome := range in.Outcomes {
		outcomeByID[strings.ToUpper(strings.TrimSpace(outcome.ID))] = outcome
	}

	var out []Match
	for _, domain := range in.Domains {
		reqs := uniqueRequirements(in.RequirementsByDomain[domain.ID])
		if len(reqs) == 0 {
			continue
		}
		glossary := glossaryTokens(domain)
		if len(glossary) == 0 {
			continue
		}
		claimTokens := map[string]struct{}{}
		for _, req := range reqs {
			addTokens(claimTokens, req.Text)
			if outcome, ok := outcomeByID[intent.RequirementPRDRef(req)]; ok {
				addTokens(claimTokens, outcome.Text)
			}
		}
		missing := missingTokens(glossary, claimTokens)
		if len(missing) == 0 {
			continue
		}
		out = append(out, Match{Type: MatchVocabDrift, Domain: domain, Tokens: missing})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Domain.ID < out[j].Domain.ID })
	return out, nil
}

// EmbeddingMatcher is the Tier 2 seam. It is deliberately off by default until
// the ai-go/search thresholds and cache policy are tuned.
type EmbeddingMatcher struct{}

func (EmbeddingMatcher) Name() string { return "embedding" }

func (EmbeddingMatcher) Match(context.Context, MatchInput) ([]Match, error) {
	log.Printf("intent alignment embedding matcher is off by default")
	return nil, nil
}

// LLMMatcher is the Tier 3 seam. It is deliberately off by default until an
// explicit Ollama-backed responsibility judge is configured.
type LLMMatcher struct{}

func (LLMMatcher) Name() string { return "llm" }

func (LLMMatcher) Match(context.Context, MatchInput) ([]Match, error) {
	log.Printf("intent alignment LLM matcher is off by default")
	return nil, nil
}

func uniqueRequirements(in []intent.CapabilityClaim) []intent.CapabilityClaim {
	seen := map[string]struct{}{}
	out := make([]intent.CapabilityClaim, 0, len(in))
	for _, req := range in {
		key := strings.TrimSpace(req.ID) + "\x00" + strings.TrimSpace(req.Anchor)
		if key == "\x00" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, req)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func glossaryTokens(domain intent.CapabilityClaim) []string {
	parts := strings.Fields(domain.Text)
	if len(parts) <= 1 {
		return nil
	}
	out := map[string]struct{}{}
	for _, part := range parts[1:] {
		for _, token := range textTokens(part) {
			out[token] = struct{}{}
		}
	}
	return sortedKeys(out)
}

func addTokens(out map[string]struct{}, text string) {
	for _, token := range textTokens(text) {
		out[token] = struct{}{}
	}
}

func missingTokens(glossary []string, claimTokens map[string]struct{}) []string {
	var out []string
	for _, token := range glossary {
		if _, ok := claimTokens[token]; ok {
			continue
		}
		out = append(out, token)
	}
	return out
}

func textTokens(s string) []string {
	seen := map[string]struct{}{}
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		word := strings.ToLower(string(current))
		current = current[:0]
		if len(word) < 3 {
			return
		}
		seen[word] = struct{}{}
	}
	for i, r := range s {
		switch {
		case r == '_' || r == '-' || r == '/' || r == '.' || unicode.IsSpace(r):
			flush()
		case unicode.IsUpper(r) && i > 0 && len(current) > 0 && !unicode.IsUpper(current[len(current)-1]):
			flush()
			current = append(current, unicode.ToLower(r))
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			current = append(current, unicode.ToLower(r))
		default:
			flush()
		}
	}
	flush()
	return sortedKeys(seen)
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
