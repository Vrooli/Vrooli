package planmodel

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

// OperatorOnlyBoundaryPrefix is the authoring escape that records why a plan has
// no acceptance_allow (operator-only / no-code work) — the boundary analogue of
// the references NO_CODE_REFS escape.
const OperatorOnlyBoundaryPrefix = "OPERATOR_ONLY:"

// ParseBoundarySection parses an authored acceptance-boundary section into a
// ChangeBoundary. It accepts several shapes so a small agent can submit whichever
// is easiest:
//
//   - a JSON object: {"acceptance_allow":[...],"acceptance_deny":[...],
//     "operator_only_reason":"..."}
//   - keyed lists: an "acceptance_allow:" / "acceptance_deny:" header followed by
//     "- glob" bullets (or an inline comma list on the same line)
//   - an "OPERATOR_ONLY: <reason>" escape line
//   - a bare list of globs with no header (treated as acceptance_allow)
//
// The result is normalized. Unrecognized prose lines are ignored (the finalize
// gate separately rejects an empty/placeholder boundary).
func ParseBoundarySection(content string) ChangeBoundary {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ChangeBoundary{}
	}
	if b, ok := parseBoundaryJSON(trimmed); ok {
		return b.Normalized()
	}
	var b ChangeBoundary
	current := "allow" // a bare leading list defaults to allow
	for _, line := range strings.Split(trimmed, "\n") {
		raw := strings.TrimSpace(line)
		if raw == "" {
			continue
		}
		lower := strings.ToLower(raw)
		switch {
		case strings.HasPrefix(lower, "operator_only:"):
			b.OperatorOnlyReason = strings.TrimSpace(raw[len(OperatorOnlyBoundaryPrefix):])
			current = ""
			continue
		case strings.HasPrefix(lower, "operator-only:"):
			b.OperatorOnlyReason = strings.TrimSpace(raw[len("operator-only:"):])
			current = ""
			continue
		case strings.HasPrefix(lower, "acceptance_allow:") || strings.HasPrefix(lower, "allow:"):
			current = "allow"
			rest := raw[strings.IndexByte(raw, ':')+1:]
			b.AcceptanceAllow = append(b.AcceptanceAllow, splitInlineGlobs(rest)...)
			continue
		case strings.HasPrefix(lower, "acceptance_deny:") || strings.HasPrefix(lower, "deny:"):
			current = "deny"
			rest := raw[strings.IndexByte(raw, ':')+1:]
			b.AcceptanceDeny = append(b.AcceptanceDeny, splitInlineGlobs(rest)...)
			continue
		}
		entry := trimBoundaryEntry(raw)
		if entry == "" {
			continue
		}
		switch current {
		case "allow":
			b.AcceptanceAllow = append(b.AcceptanceAllow, entry)
		case "deny":
			b.AcceptanceDeny = append(b.AcceptanceDeny, entry)
		}
	}
	return b.Normalized()
}

func parseBoundaryJSON(s string) (ChangeBoundary, bool) {
	if !strings.HasPrefix(s, "{") {
		return ChangeBoundary{}, false
	}
	var doc struct {
		AcceptanceAllow    []string `json:"acceptance_allow"`
		AcceptanceDeny     []string `json:"acceptance_deny"`
		OperatorOnlyReason string   `json:"operator_only_reason"`
	}
	if err := json.Unmarshal([]byte(s), &doc); err != nil {
		return ChangeBoundary{}, false
	}
	return ChangeBoundary{
		AcceptanceAllow:    doc.AcceptanceAllow,
		AcceptanceDeny:     doc.AcceptanceDeny,
		OperatorOnlyReason: doc.OperatorOnlyReason,
	}, true
}

// trimBoundaryEntry strips a leading "- "/"* " bullet and surrounding backticks
// from one glob entry line.
func trimBoundaryEntry(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "- ")
	line = strings.TrimPrefix(line, "* ")
	return trimMarkdownValue(line)
}

func splitInlineGlobs(rest string) []string {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return nil
	}
	return splitCommaList(rest)
}

// ChangeBoundary is the first-class blast-radius contract for a plan (and,
// optionally, a per-phase refinement). It declares which repo-relative path
// globs the work is permitted to change (AcceptanceAllow) and which it must not
// (AcceptanceDeny), using the SAME vocabulary as Swarm Manager's backlog change
// boundary (acceptance_allow / acceptance_deny) so the two compose without
// translation.
//
// The boundary is the source of truth for a plan's blast radius: posture,
// regression-anchor intent, validation scope, and execution reminders are all
// DERIVED from it. It deliberately has NO `scope` field and no
// primary_scenario / affected_scenario identity — scenario names are derived
// from the allow globs and references, never authored as a top-level plan
// identity (see docs/concepts/PLAN-MODEL.md, Change Boundary). This mirrors
// swarm-manager/internal/pathutil; the two should converge on a shared package
// once a second consumer needs it.
type ChangeBoundary struct {
	// AcceptanceAllow is the set of repo-relative path globs the plan is
	// permitted to change (e.g. "scenarios/plan-manager/**", "packages/proto/**",
	// "docs/**"). Required for newly authored implementation plans unless an
	// explicit operator-only / no-code reason is recorded in OperatorOnlyReason.
	AcceptanceAllow []string
	// AcceptanceDeny is the optional set of path globs the plan must NOT change.
	// Deny entries are guardrails only: they never widen validation scope; they
	// flag forbidden edits and render/validate as pre-execution constraints.
	AcceptanceDeny []string
	// OperatorOnlyReason records why a plan legitimately carries no
	// acceptance_allow (operator-only / no-code work). When set, the
	// allow-required invariant is satisfied without an allow list. Mutually
	// understood as the boundary analogue of the references NO_CODE_REFS escape.
	OperatorOnlyReason string
}

// IsZero reports whether the boundary carries no authored data at all.
func (b ChangeBoundary) IsZero() bool {
	return len(b.AcceptanceAllow) == 0 && len(b.AcceptanceDeny) == 0 &&
		strings.TrimSpace(b.OperatorOnlyReason) == ""
}

// HasAllow reports whether at least one allow glob is present.
func (b ChangeBoundary) HasAllow() bool {
	for _, g := range b.AcceptanceAllow {
		if strings.TrimSpace(g) != "" {
			return true
		}
	}
	return false
}

// Normalized returns a copy of the boundary with each glob trimmed, slash-
// normalized, de-duplicated, and sorted, and the operator-only reason trimmed.
// Normalization is idempotent so render -> parse -> render is stable.
func (b ChangeBoundary) Normalized() ChangeBoundary {
	return ChangeBoundary{
		AcceptanceAllow:    normalizeGlobs(b.AcceptanceAllow),
		AcceptanceDeny:     normalizeGlobs(b.AcceptanceDeny),
		OperatorOnlyReason: strings.TrimSpace(b.OperatorOnlyReason),
	}
}

// AffectedScenarios returns the deduplicated, sorted scenario names the allow
// globs touch. A glob like "scenarios/<name>/..." yields <name>; every other
// pattern (packages/**, docs/**, root tooling) is a repo-level path and is
// excluded here. Deny globs never contribute affected scenarios — they only
// constrain.
func (b ChangeBoundary) AffectedScenarios() []string {
	return scenariosFromGlobs(b.AcceptanceAllow)
}

// RepoPaths returns the deduplicated, sorted allow globs that are NOT scoped to
// a single scenario (shared packages, docs, root tooling). These are the paths
// that have no scenario baseline oracle today and validate as informational
// repo/path diffs until a path-baseline substrate exists.
func (b ChangeBoundary) RepoPaths() []string {
	out := make([]string, 0, len(b.AcceptanceAllow))
	seen := map[string]struct{}{}
	for _, g := range b.AcceptanceAllow {
		trimmed := normalizeGlob(g)
		if trimmed == "" {
			continue
		}
		if _, ok := scenarioFromGlob(trimmed); ok {
			continue
		}
		if _, dup := seen[trimmed]; dup {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

// placeholderRe matches an unresolved authoring placeholder such as
// "<scenario>", "<path>", "<branch>", or "<allowed path>". These are invalid in
// finalized mandatory boundary/anchor data (see ValidateBoundary).
var placeholderRe = regexp.MustCompile(`<[^<>]+>`)

// ContainsUnresolvedPlaceholder reports whether s carries an unresolved
// authoring placeholder like "<scenario>". It is the single placeholder oracle
// shared by boundary, anchor, and context-target validation.
func ContainsUnresolvedPlaceholder(s string) bool {
	return placeholderRe.MatchString(s)
}

// UnresolvedPlaceholders returns every distinct unresolved placeholder token in
// s, in first-seen order (used to build precise validation messages).
func UnresolvedPlaceholders(s string) []string {
	matches := placeholderRe.FindAllString(s, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	return out
}

// ValidateBoundary checks the boundary invariants and returns a human-readable
// reason for each problem (empty slice == valid). requireAllow asks the caller
// (finalize) to enforce that an implementation plan declares an allow list
// unless an operator-only reason is recorded; a non-finalizing reader passes
// false to validate shape only.
//
// Invariants enforced:
//   - acceptance_allow is required when requireAllow and no operator-only reason.
//   - no allow/deny entry may carry an unresolved placeholder (<scenario>, ...).
//   - deny globs are guardrails; they may not duplicate an allow glob (a path
//     cannot be both permitted and forbidden).
func ValidateBoundary(b ChangeBoundary, requireAllow bool) []string {
	b = b.Normalized()
	var problems []string
	if requireAllow && !b.HasAllow() && b.OperatorOnlyReason == "" {
		problems = append(problems,
			"acceptance_allow is required (declare the paths this plan may change, or record an operator-only/no-code reason)")
	}
	for _, g := range b.AcceptanceAllow {
		if tokens := UnresolvedPlaceholders(g); len(tokens) > 0 {
			problems = append(problems,
				"acceptance_allow glob has unresolved placeholder(s) "+strings.Join(tokens, ", ")+": "+g)
		}
	}
	for _, g := range b.AcceptanceDeny {
		if tokens := UnresolvedPlaceholders(g); len(tokens) > 0 {
			problems = append(problems,
				"acceptance_deny glob has unresolved placeholder(s) "+strings.Join(tokens, ", ")+": "+g)
		}
	}
	if tokens := UnresolvedPlaceholders(b.OperatorOnlyReason); len(tokens) > 0 {
		problems = append(problems,
			"operator-only reason has unresolved placeholder(s) "+strings.Join(tokens, ", "))
	}
	allow := map[string]struct{}{}
	for _, g := range b.AcceptanceAllow {
		allow[g] = struct{}{}
	}
	for _, g := range b.AcceptanceDeny {
		if _, ok := allow[g]; ok {
			problems = append(problems, "acceptance_deny glob also appears in acceptance_allow: "+g)
		}
	}
	return problems
}

// NormalizeBoundaryGlob is the exported single-entry normalizer used by callers
// outside this package (execution-time boundary extension) so a glob submitted
// by an agent is compared against the stored boundary under identical rules.
func NormalizeBoundaryGlob(g string) string { return normalizeGlob(g) }

// DenyCovers reports whether any deny glob forbids the candidate path/glob, and
// returns the deny entry that does. It treats a trailing "**" as a prefix
// guard, so a deny of "scenarios/swarm-manager/**" covers a candidate of
// "scenarios/swarm-manager/api/**".
//
// This is deliberately conservative in the direction of refusing: a candidate
// that is broader than a deny glob (e.g. candidate "scenarios/**" against deny
// "scenarios/swarm-manager/**") is also refused, because granting it would
// silently swallow the forbidden subtree. Boundary extension is an
// execution-time convenience; it must never become a way to erase an authored
// prohibition.
func DenyCovers(denies []string, candidate string) (bool, string) {
	candidate = normalizeGlob(candidate)
	if candidate == "" {
		return false, ""
	}
	candidateBase := strings.TrimSuffix(candidate, "**")
	for _, deny := range denies {
		deny = normalizeGlob(deny)
		if deny == "" {
			continue
		}
		if candidate == deny {
			return true, deny
		}
		denyBase := strings.TrimSuffix(deny, "**")
		// The candidate sits under a forbidden subtree.
		if denyBase != deny && denyBase != "" && strings.HasPrefix(candidate, denyBase) {
			return true, deny
		}
		// The candidate is a wildcard that would swallow a forbidden subtree.
		if candidateBase != candidate && candidateBase != "" && strings.HasPrefix(deny, candidateBase) {
			return true, deny
		}
	}
	return false, ""
}

// --- pure path/glob helpers (local mirror of swarm-manager pathutil) ---

// normalizeGlobs trims, slash-normalizes, de-duplicates, and sorts a glob slice.
func normalizeGlobs(globs []string) []string {
	if len(globs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(globs))
	out := make([]string, 0, len(globs))
	for _, g := range globs {
		trimmed := normalizeGlob(g)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

// normalizeGlob trims surrounding whitespace and backslashes->slashes for one
// glob/path entry, and strips a leading "./".
func normalizeGlob(g string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(g, `\`, "/"))
	trimmed = strings.TrimPrefix(trimmed, "./")
	return trimmed
}

// scenariosFromGlobs extracts the deduplicated, sorted scenario names from a set
// of allow globs. Globs that start with "scenarios/<name>/..." (or exactly
// "scenarios/<name>") yield the scenario name; all other patterns are skipped.
func scenariosFromGlobs(globs []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(globs))
	for _, g := range globs {
		name, ok := scenarioFromGlob(normalizeGlob(g))
		if !ok {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

// BoundaryFromLegacyAnchor derives a ChangeBoundary from a legacy regression
// anchor during import, so a pre-cutover plan that predates the boundary model
// still gains an explicit blast radius:
//
//   - a scenario_baseline anchor (Scenario "foo") => allow "scenarios/foo/**".
//   - a head_sha_allowlist anchor's AllowlistPaths => allow entries verbatim
//     (a bare "scenarios/foo" is expanded to "scenarios/foo/**").
//
// An unstructured/legacy-prose anchor (no scenario, no allowlist) yields a zero
// boundary — there is nothing safe to derive, and validation stays honestly
// degraded rather than inventing a scope.
func BoundaryFromLegacyAnchor(a RegressionAnchor) ChangeBoundary {
	var allow []string
	if scenario := strings.TrimSpace(a.Scenario); scenario != "" && !ContainsUnresolvedPlaceholder(scenario) {
		allow = append(allow, "scenarios/"+scenario+"/**")
	}
	for _, p := range a.AllowlistPaths {
		entry := normalizeGlob(p)
		if entry == "" || ContainsUnresolvedPlaceholder(entry) {
			continue
		}
		// A bare scenario dir gets a recursive glob so it matches the scenario tree.
		if name, ok := scenarioFromGlob(entry); ok && entry == "scenarios/"+name {
			entry = "scenarios/" + name + "/**"
		}
		allow = append(allow, entry)
	}
	return ChangeBoundary{AcceptanceAllow: allow}.Normalized()
}

// BoundaryAnchorCommands derives the tiered check commands a boundary-native
// anchor implies, in a deterministic order:
//
//  1. One git-control-tower baseline snapshot-status + baseline diff pair per
//     affected scenario, but ONLY when baselineName is a single safe token (the
//     verified GCT CLI requires --scenario and --name). These are verdict
//     ORACLES.
//  2. One informational `git diff --stat [<headSha>] -- <repo paths>` for the
//     non-scenario allow globs, when any exist. This is NOT an oracle until a
//     path-baseline substrate exists (see validation isOracleCommand).
//
// It returns the commands and the subset that is informational (non-oracle) so
// callers can label tiers honestly without re-deriving the split.
func BoundaryAnchorCommands(b ChangeBoundary, baselineName, headSha string) (commands, informational []string) {
	name := strings.TrimSpace(baselineName)
	safeName := name != "" && !strings.ContainsAny(name, " \t\r\n")
	for _, scenario := range b.AffectedScenarios() {
		if !safeName {
			continue
		}
		commands = append(commands,
			"git-control-tower baseline snapshot status --scenario "+scenario+" --name "+name+" --wait --json",
			"git-control-tower baseline diff --scenario "+scenario+" --name "+name+" --wait",
		)
	}
	if repoPaths := b.RepoPaths(); len(repoPaths) > 0 {
		cmd := "git diff --stat"
		if sha := strings.TrimSpace(headSha); sha != "" && !ContainsUnresolvedPlaceholder(sha) {
			if strings.ToLower(sha) != "captured at execution start" {
				cmd += " " + sha
			}
		}
		cmd += " -- " + strings.Join(repoPaths, " ")
		commands = append(commands, cmd)
		informational = append(informational, cmd)
	}
	return commands, informational
}

// scenarioFromGlob maps a normalized glob/path to a scenario name when it is
// scoped under scenarios/<name>/... A bare "scenarios/" or "scenarios/**"
// (no concrete scenario segment) is NOT a single scenario and yields ok=false.
func scenarioFromGlob(glob string) (string, bool) {
	if !strings.HasPrefix(glob, "scenarios/") {
		return "", false
	}
	rest := strings.TrimPrefix(glob, "scenarios/")
	parts := strings.SplitN(rest, "/", 2)
	name := strings.TrimSpace(parts[0])
	if name == "" || name == "*" || name == "**" {
		return "", false
	}
	return name, true
}
