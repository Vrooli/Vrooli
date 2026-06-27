package coverage

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/vrooli/api-core/spacedoc"
)

// NumeratorJoiner computes the live numerator for a projection: it joins the
// denominator cells against the owner's live registry (search-hub providers
// list / test-genie health / prompt-manager graph health) and returns the
// effective per-cell status, whether the registry was reachable, and an honest
// reason when it was not. The numerator is computed live and never stored.
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
}

// execNumeratorJoiner is the production joiner. It reads the owner's live
// registry over its CLI (behind the shared CommandRunner seam), then delegates
// per-cell status derivation to the projection's matcher.
type execNumeratorJoiner struct {
	run CommandRunner
}

// seam: cellMatcher is the per-projection numerator join strategy. Production
// wires one matcher per projection; tests exercise the same matchers with
// captured registry JSON so a cell absent from the returned map keeps its
// authored status ("can't resolve" is not fabricated as missing).
type cellMatcher interface {
	registryArgs() []string
	recompute(cells []spacedoc.Cell, raw []byte) map[string]spacedoc.CellStatus
}

type (
	answerMatcher   struct{}
	validateMatcher struct{}
	guideMatcher    struct{}
)

var (
	_ cellMatcher = answerMatcher{}
	_ cellMatcher = validateMatcher{}
	_ cellMatcher = guideMatcher{}
)

const guideHealthyScore = 0.5

// NewNumeratorJoiner returns the production NumeratorJoiner.
func NewNumeratorJoiner() NumeratorJoiner { return &execNumeratorJoiner{run: execRunner} }

// NewNumeratorJoinerWithRunner returns a joiner using the given runner (tests).
func NewNumeratorJoinerWithRunner(run CommandRunner) NumeratorJoiner {
	return &execNumeratorJoiner{run: run}
}

func (j *execNumeratorJoiner) Join(ctx context.Context, p Projection, cells []spacedoc.Cell) JoinResult {
	owner := OwnerFor(p)
	matcher := matcherFor(p)
	if owner == "" || matcher == nil {
		return JoinResult{Available: false, Reason: "unknown coverage projection: " + string(p)}
	}
	out, err := j.run(ctx, owner, matcher.registryArgs()...)
	if err != nil {
		return JoinResult{Available: false, Reason: owner + " registry unreachable: " + err.Error()}
	}
	return JoinResult{Available: true, Statuses: matcher.recompute(cells, out)}
}

func matcherFor(p Projection) cellMatcher {
	switch p {
	case ProjectionAnswer:
		return answerMatcher{}
	case ProjectionValidate:
		return validateMatcher{}
	case ProjectionGuide:
		return guideMatcher{}
	default:
		return nil
	}
}

func (answerMatcher) registryArgs() []string { return []string{"providers", "list", "--json"} }

func (answerMatcher) recompute(cells []spacedoc.Cell, raw []byte) map[string]spacedoc.CellStatus {
	live := collectStringValues(raw, "provider_id", "id", "name", "type")
	return recomputeAnswer(cells, live)
}

func (validateMatcher) registryArgs() []string { return []string{"health", "--json"} }

func (validateMatcher) recompute(cells []spacedoc.Cell, raw []byte) map[string]spacedoc.CellStatus {
	index := validateStatusIndex(raw)
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

func (guideMatcher) registryArgs() []string { return []string{"graph", "health", "--json"} }

func (guideMatcher) recompute(cells []spacedoc.Cell, raw []byte) map[string]spacedoc.CellStatus {
	scores := guideScoreIndex(raw)
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

type validateProviderStatus struct {
	failing        bool
	autofixPending bool
}

func validateStatusIndex(raw []byte) map[string]validateProviderStatus {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return map[string]validateProviderStatus{}
	}
	health := objectValue(root, "selfHealth")
	if health == nil {
		health = root
	}
	out := map[string]validateProviderStatus{}
	for _, phase := range arrayValue(objectValue(health, "catalog"), "phases") {
		provider := strings.ToLower(stringValue(phase, "provider"))
		if provider == "" {
			continue
		}
		out[provider] = validateProviderStatus{}
	}
	for _, phase := range arrayValue(objectValue(health, "ledger"), "phases") {
		provider := strings.ToLower(stringValue(phase, "provider"))
		if provider == "" {
			continue
		}
		status := out[provider]
		if numberValue(phase, "failureRate") > 0 || numberValue(phase, "failure_rate") > 0 {
			status.failing = true
		}
		out[provider] = status
	}
	for _, conformance := range arrayValue(health, "conformance") {
		provider := strings.ToLower(stringValue(conformance, "provider"))
		if provider == "" {
			continue
		}
		status := out[provider]
		autofix := objectValue(conformance, "autofix")
		if numberValue(autofix, "pending") > 0 {
			status.autofixPending = true
		}
		out[provider] = status
	}
	return out
}

func guideScoreIndex(raw []byte) map[string]float64 {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return map[string]float64{}
	}
	out := map[string]float64{}
	var walk func(any)
	walk = func(node any) {
		switch v := node.(type) {
		case []any:
			for _, e := range v {
				walk(e)
			}
		case map[string]any:
			id := firstString(v, "nodeId", "node_id", "id")
			if id != "" {
				if score, ok := firstNumber(v, "score", "healthScore", "health_score"); ok {
					out[strings.ToLower(id)] = score
				}
			}
			for _, e := range v {
				walk(e)
			}
		}
	}
	walk(root)
	return out
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

func objectValue(node any, key string) map[string]any {
	obj, ok := node.(map[string]any)
	if !ok {
		return nil
	}
	child, ok := obj[key].(map[string]any)
	if !ok {
		return nil
	}
	return child
}

func arrayValue(node any, key string) []any {
	obj, ok := node.(map[string]any)
	if !ok {
		return nil
	}
	arr, ok := obj[key].([]any)
	if !ok {
		return nil
	}
	return arr
}

func stringValue(node any, key string) string {
	obj, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	val, _ := obj[key].(string)
	return val
}

func numberValue(node any, key string) float64 {
	obj, ok := node.(map[string]any)
	if !ok {
		return 0
	}
	n, _ := obj[key].(float64)
	return n
}

func firstString(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		if s, ok := obj[key].(string); ok {
			return s
		}
	}
	return ""
}

func firstNumber(obj map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if n, ok := obj[key].(float64); ok {
			return n, true
		}
	}
	return 0, false
}

// collectStringValues walks arbitrary decoded JSON and collects the lower-cased
// string values stored under any of the given keys, into a set. Tolerant of the
// exact registry response shape (objects, nested arrays) so a shape tweak in the
// owner's `providers list --json` does not silently zero the numerator.
func collectStringValues(raw []byte, keys ...string) map[string]bool {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return map[string]bool{}
	}
	want := make(map[string]bool, len(keys))
	for _, k := range keys {
		want[k] = true
	}
	out := map[string]bool{}
	var walk func(node any)
	walk = func(node any) {
		switch v := node.(type) {
		case map[string]any:
			for k, val := range v {
				if want[k] {
					if s, ok := val.(string); ok && s != "" {
						out[strings.ToLower(s)] = true
					}
				}
				walk(val)
			}
		case []any:
			for _, e := range v {
				walk(e)
			}
		}
	}
	walk(root)
	return out
}
