package coverage

import (
	"context"
	"strings"

	"github.com/vrooli/api-core/spacedoc"
)

// NumeratorJoiner computes the live numerator for a projection: it joins the
// denominator cells against the owner's live registry (search-hub provider
// registry / test-genie self-health / prompt-manager graph health) and returns
// the effective per-cell status, whether the registry was reachable, and an
// honest reason when it was not. The numerator is computed live and never
// stored.
//
// The production joiner (numeratorclient.go) reads each owner over a typed
// Connect-RPC client resolved through api-core/discovery, bounded by a short
// per-owner deadline. This file holds the transport-independent join logic: the
// pure per-cell recompute functions and the token helpers they share, so the
// same matching rules are unit-testable without any owner reachable.
type NumeratorJoiner interface {
	Join(ctx context.Context, p Projection, cells []spacedoc.Cell) JoinResult
}

// JoinResult is the outcome of a live numerator join.
type JoinResult struct {
	// Statuses is the effective live status per cell id. A cell absent from the
	// map keeps its authored denominator status (the join could not speak to it).
	Statuses map[string]spacedoc.CellStatus
	// Available is false when the owner registry was unreachable; the projection
	// degrades (counts fall back to authored status) but never false-fails.
	Available bool
	// Reason is the honest explanation when Available is false.
	Reason string
	// DenominatorConfidence is supplied by owners that audit their own
	// denominator (currently Act/program-runtime). Empty means the owner did
	// not provide a confidence signal and the space document remains the source.
	DenominatorConfidence spacedoc.DenominatorConfidence
}

// guideHealthyScore is the prompt-manager graph health-score threshold at or
// above which a Guide skill node counts as "now" (healthy enough to answer a
// Guide question). It is the load-bearing cut that drives the headline Guide
// numerator; see docs/concepts/COVERAGE-MODEL.md. 0.5 means "more healthy than
// not" — a deliberately lenient bar, because a skill existing and scoring at
// least neutral is the signal that the Guide cell is served at all.
const guideHealthyScore = 0.5

// validateProviderStatus is one test-genie provider's distilled Validate signal.
type validateProviderStatus struct {
	failing        bool
	autofixPending bool
}

// recomputeAnswer re-derives each Answer cell's status from the live provider
// set. A cell is NOW iff its declared provider matches a live provider; a cell
// whose authored status is NOW but whose provider is not live degrades to
// IN_REACH (the substrate is declared but not serving); a cell with no
// resolvable provider keeps its authored status (the join cannot speak to it).
func recomputeAnswer(cells []spacedoc.Cell, live map[string]bool) map[string]spacedoc.CellStatus {
	out := make(map[string]spacedoc.CellStatus, len(cells))
	for _, c := range cells {
		toks := providerTokens(c.Owner)
		if len(toks) == 0 {
			continue // no provider to join against — keep authored status
		}
		matched := false
		for _, t := range toks {
			if matchesLive(t, live) {
				matched = true
				break
			}
		}
		if matched {
			out[c.ID] = spacedoc.StatusNow
		} else if c.Status == spacedoc.StatusNow {
			// Authored NOW but no live provider: honest downgrade.
			out[c.ID] = spacedoc.StatusInReach
		}
		// else: authored IN_REACH/MISSING with an unmatched declared provider —
		// keep authored status (no map entry).
	}
	return out
}

// recomputeValidate re-derives each Validate cell's status from the live
// test-genie provider index: a matched provider that is failing or has pending
// autofix work degrades to IN_REACH; a matched healthy provider is NOW; an
// unmatched cell keeps its authored status (no map entry).
func recomputeValidate(cells []spacedoc.Cell, index map[string]validateProviderStatus) map[string]spacedoc.CellStatus {
	out := make(map[string]spacedoc.CellStatus, len(cells))
	for _, c := range cells {
		for _, tok := range providerTokens(c.Owner) {
			status, ok := index[tok]
			if !ok {
				continue
			}
			if status.failing || status.autofixPending {
				out[c.ID] = spacedoc.StatusInReach
			} else {
				out[c.ID] = spacedoc.StatusNow
			}
			break
		}
	}
	return out
}

// recomputeGuide re-derives each Guide cell's status from the live
// prompt-manager graph score index: a cell is NOW iff every declared skill
// resolves to a score at or above guideHealthyScore; IN_REACH if at least one
// declared skill resolves (the capability exists but is not all-healthy); an
// unresolved cell keeps its authored status (no map entry).
func recomputeGuide(cells []spacedoc.Cell, scores map[string]float64) map[string]spacedoc.CellStatus {
	out := make(map[string]spacedoc.CellStatus, len(cells))
	for _, c := range cells {
		toks := skillTokens(c.Owner)
		if len(toks) == 0 {
			continue
		}
		resolved := 0
		healthy := 0
		for _, tok := range toks {
			score, ok := resolveGuideScore(tok, scores)
			if !ok {
				continue
			}
			resolved++
			if score >= guideHealthyScore {
				healthy++
			}
		}
		switch {
		case resolved == len(toks) && healthy == len(toks):
			out[c.ID] = spacedoc.StatusNow
		case resolved > 0:
			out[c.ID] = spacedoc.StatusInReach
		}
	}
	return out
}

// providerTokens extracts candidate provider identifiers from a cell's free-text
// owner ("ui-health.surfaces + cli-health.commands (API)" -> [ui-health.surfaces
// cli-health.commands]). A token is a dotted or hyphenated identifier; the
// "_(none)_" placeholder and prose yield nothing.
func providerTokens(owner string) []string {
	owner = strings.TrimSpace(owner)
	if owner == "" || strings.Contains(strings.ToLower(owner), "none") {
		return nil
	}
	var toks []string
	var b strings.Builder
	flush := func() {
		t := strings.Trim(b.String(), ".-")
		// A real provider token contains a hyphen or a dot (scenario-name or
		// scenario.leaf); bare words like "API" or "code" are skipped.
		if (strings.Contains(t, "-") || strings.Contains(t, ".")) && len(t) > 2 {
			toks = append(toks, strings.ToLower(t))
		}
		b.Reset()
	}
	for _, r := range owner {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '.' || r == '_':
			b.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return toks
}

// skillTokens extracts Guide skill ids from an owner cell. Prefer explicit
// backtick-delimited ids from the space doc; fall back to the same comma/plus
// shape used by ValidateBaseDocs tests where the parser has already stripped
// Markdown.
func skillTokens(owner string) []string {
	owner = strings.TrimSpace(owner)
	if owner == "" || strings.Contains(strings.ToLower(owner), "none") {
		return nil
	}
	var toks []string
	for {
		start := strings.IndexByte(owner, '`')
		if start < 0 {
			break
		}
		rest := owner[start+1:]
		end := strings.IndexByte(rest, '`')
		if end < 0 {
			break
		}
		if tok := normalizeSkillToken(rest[:end]); tok != "" {
			toks = append(toks, tok)
		}
		owner = rest[end+1:]
	}
	if len(toks) > 0 {
		return toks
	}
	for _, f := range strings.FieldsFunc(owner, func(r rune) bool { return r == ',' || r == '+' }) {
		if tok := normalizeSkillToken(f); tok != "" {
			toks = append(toks, tok)
		}
	}
	return toks
}

func normalizeSkillToken(s string) string {
	s = strings.TrimSpace(strings.Trim(s, "`()"))
	if s == "" {
		return ""
	}
	low := strings.ToLower(s)
	if strings.Contains(low, "none") ||
		strings.HasPrefix(low, "the ") ||
		strings.HasPrefix(low, "adjacent") ||
		strings.HasPrefix(low, "partial") {
		return ""
	}
	words := strings.Fields(low)
	if len(words) != 1 {
		return ""
	}
	return strings.Trim(words[0], ".,;:")
}

// matchesLive reports whether a denominator provider token corresponds to a live
// registry key. Match is on the leaf-or-scenario head: "ui-health.surfaces"
// matches a live "ui-health" or "ui-health.surfaces"; exact and head-prefix both
// count so a scenario that registers granular leaves still matches a coarse
// denominator entry and vice versa.
func matchesLive(token string, live map[string]bool) bool {
	if live[token] {
		return true
	}
	head := token
	if i := strings.IndexByte(token, '.'); i > 0 {
		head = token[:i]
	}
	if live[head] {
		return true
	}
	for k := range live {
		kHead := k
		if i := strings.IndexByte(k, '.'); i > 0 {
			kHead = k[:i]
		}
		if kHead == head {
			return true
		}
	}
	return false
}

func resolveGuideScore(token string, scores map[string]float64) (float64, bool) {
	if score, ok := scores[token]; ok {
		return score, true
	}
	if score, ok := scores["skill:"+token]; ok {
		return score, true
	}
	return 0, false
}
