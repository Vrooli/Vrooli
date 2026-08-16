package coverage

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	// Available is false when any required owner signal was unreachable. The
	// projection may retain authored cell statuses for explanation, but callers
	// must treat its ratio as unavailable.
	Available bool
	// Reason is the honest explanation when Available is false.
	Reason string
	// DenominatorConfidence is supplied by owners that audit their own
	// denominator (currently Act/program-runtime). Empty means the owner did
	// not provide a confidence signal and the space document remains the source.
	DenominatorConfidence spacedoc.DenominatorConfidence
	// Evidence is keyed by denominator cell id. It is populated for Answer
	// cells even when a signal is unavailable, so ExplainCell can distinguish
	// missing evidence from a negative verdict.
	Evidence   map[string][]SignalEvidence
	Conditions map[string]ConditionVerdict
	// OwnerResolved records whether denominator owner tokens resolved to exact
	// registered leaf IDs. It separates an authoring error from reachability.
	OwnerResolved                 map[string]bool
	AnswerCorpusCapableNowCount   int
	AnswerCorpusCapableTotalCells int
	AnswerEndToEndNowCount        int
	AnswerEndToEndTotalCells      int
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
	condition      ConditionVerdict
	sustained      bool
}

type answerProviderEvidence struct {
	ProviderID           string
	Active               bool
	Reachable            bool
	FreshEval            bool
	CorpusCapable        bool
	EvalAvailable        bool
	Condition            ConditionVerdict
	ReachabilityEvidence string
	EvalEvidence         string
	CorpusEvalEvidence   string
	RoutingPrecision     float64
	RetrievalRecall      float64
	RatesAvailable       bool
}

const (
	answerEvalMinimumRoutingPrecision = 0.85
	answerEvalMinimumRetrievalRecall  = 0.80
)

// recomputeAnswer re-derives each Answer cell from the three independent
// runtime signals required by the readiness contract: an ACTIVE declaration,
// current federation reachability, and a non-degraded eval run inside the
// freshness window. Registration alone can never promote a cell to NOW.
func recomputeAnswer(cells []spacedoc.Cell, providers []answerProviderEvidence) (map[string]spacedoc.CellStatus, map[string][]SignalEvidence) {
	out := make(map[string]spacedoc.CellStatus, len(cells))
	evidence := make(map[string][]SignalEvidence, len(cells))
	for _, c := range cells {
		toks := providerTokens(c.Owner)
		if len(toks) == 0 {
			continue // no provider to join against — keep authored status
		}
		matched := matchAnswerProviders(toks, providers)
		if len(matched) != len(toks) {
			// The declared provider did not resolve to an ACTIVE descriptor. A
			// capability gap or unresolved owner must never be promoted and an
			// authored status remains the only honest answer.
			evidence[c.ID] = []SignalEvidence{
				{Signal: "active", Verdict: "did_not_hold", Evidence: "declared provider is not in the ACTIVE registry (unresolved or capability gap)"},
				{Signal: "reachable", Verdict: "not_evaluated", Evidence: "provider is not ACTIVE"},
				{Signal: "corpus_eval_fresh", Verdict: "not_evaluated", Evidence: "provider is not ACTIVE"},
				{Signal: "eval_fresh", Verdict: "not_evaluated", Evidence: "provider is not ACTIVE"},
			}
			if c.Status == spacedoc.StatusNow {
				// NOW is a runtime claim. Even though unresolved non-NOW cells
				// retain their authored status, an authored NOW with no declared
				// active owner must be downgraded so NOW remains a three-signal
				// verdict everywhere.
				out[c.ID] = spacedoc.StatusInReach
			}
			continue
		}
		evalVerdict := verdict(allAnswerProviders(matched, func(p answerProviderEvidence) bool { return p.FreshEval }))
		if !allAnswerProviders(matched, func(p answerProviderEvidence) bool { return p.EvalAvailable }) {
			evalVerdict = "unavailable"
		}
		evidence[c.ID] = []SignalEvidence{
			{Signal: "active", Verdict: verdict(allAnswerProviders(matched, func(p answerProviderEvidence) bool { return p.Active })), Evidence: answerProviderEvidenceText(matched, "active")},
			{Signal: "reachable", Verdict: verdict(allAnswerProviders(matched, func(p answerProviderEvidence) bool { return p.Reachable })), Evidence: answerProviderEvidenceText(matched, "reachable")},
			{Signal: "corpus_eval_fresh", Verdict: verdict(allAnswerProviders(matched, func(p answerProviderEvidence) bool { return p.CorpusCapable })), Evidence: answerProviderEvidenceText(matched, "corpus")},
			{Signal: "eval_fresh", Verdict: evalVerdict, Evidence: answerProviderEvidenceText(matched, "eval")},
		}
		if allAnswerProviders(matched, func(p answerProviderEvidence) bool { return p.RatesAvailable }) {
			evidence[c.ID] = append(evidence[c.ID],
				SignalEvidence{Signal: "routing_precision", Verdict: verdict(allAnswerProviders(matched, func(p answerProviderEvidence) bool { return p.RoutingPrecision >= answerEvalMinimumRoutingPrecision })), Evidence: answerRateEvidence(matched, "routing")},
				SignalEvidence{Signal: "retrieval_recall", Verdict: verdict(allAnswerProviders(matched, func(p answerProviderEvidence) bool { return p.RetrievalRecall >= answerEvalMinimumRetrievalRecall })), Evidence: answerRateEvidence(matched, "recall")},
			)
		}
		if allAnswerProviders(matched, func(p answerProviderEvidence) bool { return p.Active && p.Reachable && p.FreshEval }) {
			out[c.ID] = spacedoc.StatusNow
		} else if c.Status == spacedoc.StatusNow {
			// Authored NOW but one or more runtime signals is absent: honest
			// downgrade while preserving the distinction in ExplainCell notes.
			out[c.ID] = spacedoc.StatusInReach
		}
		// Authored IN_REACH/MISSING with an unresolved or incomplete provider
		// keeps its authored status (no overlay entry).
	}
	return out, evidence
}

func matchAnswerProviders(tokens []string, providers []answerProviderEvidence) []answerProviderEvidence {
	out := make([]answerProviderEvidence, 0, len(tokens))
	for _, token := range tokens {
		for _, provider := range providers {
			if matchesProviderToken(token, provider.ProviderID) {
				out = append(out, provider)
				break
			}
		}
	}
	return out
}

func allAnswerProviders(providers []answerProviderEvidence, predicate func(answerProviderEvidence) bool) bool {
	if len(providers) == 0 {
		return false
	}
	for _, provider := range providers {
		if !predicate(provider) {
			return false
		}
	}
	return true
}

func answerProviderEvidenceText(providers []answerProviderEvidence, kind string) string {
	parts := make([]string, 0, len(providers))
	for _, provider := range providers {
		var value string
		switch kind {
		case "active":
			value = provider.ProviderID + " is " + boolWord(provider.Active, "ACTIVE", "not ACTIVE")
		case "reachable":
			value = provider.ReachabilityEvidence
		case "corpus":
			value = provider.CorpusEvalEvidence
		default:
			value = provider.EvalEvidence
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, "; ")
}

func answerRateEvidence(providers []answerProviderEvidence, kind string) string {
	parts := make([]string, 0, len(providers))
	for _, provider := range providers {
		value := provider.RoutingPrecision
		minimum := answerEvalMinimumRoutingPrecision
		if kind == "recall" {
			value = provider.RetrievalRecall
			minimum = answerEvalMinimumRetrievalRecall
		}
		parts = append(parts, fmt.Sprintf("%s=%.3f (minimum %.2f)", provider.ProviderID, value, minimum))
	}
	return strings.Join(parts, "; ")
}

func answerCoverageCounts(cells []spacedoc.Cell, providers []answerProviderEvidence) (int, int) {
	corpus, endToEnd := 0, 0
	for _, cell := range cells {
		matched := matchAnswerProviders(providerTokens(cell.Owner), providers)
		if allAnswerProviders(matched, func(p answerProviderEvidence) bool { return p.Active && p.Reachable && p.CorpusCapable }) {
			corpus++
		}
		if allAnswerProviders(matched, func(p answerProviderEvidence) bool { return p.Active && p.Reachable && p.FreshEval }) {
			endToEnd++
		}
	}
	return corpus, endToEnd
}

func matchesProviderToken(token, providerID string) bool {
	token = strings.ToLower(strings.TrimSpace(token))
	providerID = strings.ToLower(strings.TrimSpace(providerID))
	if token == "" || providerID == "" {
		return false
	}
	return token == providerID
}

func verdict(ok bool) string {
	if ok {
		return "held"
	}
	return "did_not_hold"
}

func boolWord(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}

// sustainedDegradationWindow is intentionally longer than a restart or
// rollout recovery opportunity. Revisit when the fleet has enough timestamped
// phase history to calibrate a distribution rather than a single operator
// sweep; only the age of the current contiguous failure streak promotes a
// coverage downgrade.
const sustainedDegradationWindow = 7 * 24 * time.Hour

// recomputeValidate keeps existence coverage separate from provider condition.
// A failed or autofix-pending provider remains covered and is reported on the
// condition axis instead of being silently converted to IN_REACH.
func recomputeValidate(cells []spacedoc.Cell, index map[string]validateProviderStatus) map[string]spacedoc.CellStatus {
	out, _ := recomputeValidateWithConditions(cells, index)
	return out
}

func recomputeValidateWithConditions(cells []spacedoc.Cell, index map[string]validateProviderStatus) (map[string]spacedoc.CellStatus, map[string]ConditionVerdict) {
	out := make(map[string]spacedoc.CellStatus, len(cells))
	conditions := make(map[string]ConditionVerdict, len(cells))
	for _, c := range cells {
		for _, tok := range providerTokens(c.Owner) {
			status, ok := index[tok]
			if !ok {
				continue
			}
			out[c.ID] = spacedoc.StatusNow
			if status.sustained {
				out[c.ID] = spacedoc.StatusInReach
			}
			condition := status.condition
			if condition == "" {
				condition = ConditionOK
				if status.failing || status.autofixPending {
					condition = ConditionDegraded
				}
			}
			conditions[c.ID] = condition
			break
		}
	}
	return out, conditions
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

// matchesLive reports whether a denominator provider token corresponds to a
// live registry key. Only exact leaf IDs match; scenario heads and RPC names
// are not provider identities and cannot satisfy a cell by prefix.
func matchesLive(token string, live map[string]bool) bool {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return false
	}
	return live[token]
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
