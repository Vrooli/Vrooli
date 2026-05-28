// Pillar 3 (runtime attribution) validator. The static graph (Pillar 1)
// describes who SHOULD read/write what. The prose scan (Pillar 2)
// catches stale topic references in markdown. This file is the third
// leg: it joins what was actually written to what is declared, and
// surfaces drift the static graph structurally cannot see.
//
// Authoritative contract: docs/agent-system/RUNTIME_ATTRIBUTION.md
// (§ Validator behavior at the boundary, § kind enum, § Per-team
// attributionValidFrom).
//
// Scope of this file:
//
//  1. ruleActualWriterUndeclared — emits errors for `agent-member`
//     drift (an entry whose declaring member's topics.json does not
//     declare an output[] prefix overlapping the entry's topic; or an
//     entry claiming a member id that does not exist on the team).
//     An agent-member writing where it has no producer-side declaration
//     is concrete drift between observed runtime behavior and the
//     topics.json contract — the kind of thing only Pillar 3 can catch.
//
//     Emits warnings (not errors) for `external` entries that exceed
//     the team's per-week threshold. The threshold is operator-tunable
//     (`policy.flagExternalWritesPerWeek` on team.json) and the finding
//     is an alert, not a structural fault — keeping it at warning
//     leaves the CI gate scoped to true drift while preserving the
//     operator-facing signal for tuning.
//
//     Emits a defensive `attribution_malformed` error when a
//     post-cutoff entry's attribution shape is structurally broken
//     (the API rejects this at write time, but the validator still
//     surfaces it as a defense-in-depth check).
//
//  2. The two helpers loadKnowledgeJSONL (resilient line-by-line parser
//     that skips blanks and tolerates trailing junk) and isoWeekKey
//     (deterministic ISO year-week derivation from a timestamp string).
//
// Out of scope here (owned elsewhere):
//
//   - writer-skill kind: governed by writes_to[] on skill.json (loaded
//     via WriterSkillProducers). We track but never flag writer-skill
//     writes here.
//   - operator-direct + flagOperatorWritesPerWeek: documented in
//     RUNTIME_ATTRIBUTION.md as a future opt-in; the threshold field is
//     not wired today to keep this rule's scope tight. Operator-direct
//     entries are skipped entirely.
//   - investigation kind: read-only relative to topic-flow declarations
//     by spec; skipped.
//   - findings.json telemetry artifact: cli/graph/findings_artifact.go.
//
// Severity policy. The agent-member subcase fires at error (a CI gate —
// concrete drift between observed runtime behavior and declared
// topics.json). The external-threshold subcase fires at warning (an
// operator-tunable alert, not a structural fault). The defensive
// `attribution_malformed` check is at error severity because the API
// enforces the same rule at write time, so a finding here means the
// on-disk file was hand-edited or a migration broke.
package memberflow

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"prompt-manager/store"
	"sort"
	"strings"
	"time"
)

// ruleActualWriterUndeclared is the Pillar 3 runtime ground-truth rule.
// It walks every team's knowledge.jsonl from the team's
// attributionValidFrom cutoff forward, joins each entry's structured
// attribution against the same store's topics.json declarations, and
// surfaces undeclared writer/topic combinations.
//
// The rule is silent when:
//   - opts.StoreDir is empty (no on-disk store to scan; unit tests that
//     pass synthetic members[] without a backing store get this);
//   - the team has no team.json or empty AttributionValidFrom (team has
//     not yet adopted Pillar 3);
//   - the team's knowledge.jsonl does not exist (fresh-team baseline).
//
// Per-team work is deterministic and side-effect-free: results depend
// only on (members, opts, on-disk file contents). No clock, no randomness.
func ruleActualWriterUndeclared(members []MemberTopics, opts ValidationOptions) []Finding {
	runtimeDir := strings.TrimSpace(opts.RuntimeDataDir)
	if runtimeDir == "" {
		// Tests that pre-runtime-class-split passed only StoreDir treated
		// it as both config and runtime; preserve that single-tree fallback
		// so test fixtures that put knowledge.jsonl alongside team.json
		// continue to validate.
		runtimeDir = strings.TrimSpace(opts.StoreDir)
	}
	if runtimeDir == "" {
		return nil
	}
	if len(opts.TeamContracts) == 0 {
		return nil
	}

	// Index members by (team, member-id) once; the scan body asks
	// "does <team>/<member> declare a matching output[]?" per entry, and
	// O(1) lookup is the only sensible shape given thousands-of-entries
	// knowledge.jsonl files in mature stores.
	memberIndex := make(map[string]map[string]MemberTopics, len(members))
	for _, m := range members {
		byMember := memberIndex[m.Ref.Team]
		if byMember == nil {
			byMember = make(map[string]MemberTopics)
			memberIndex[m.Ref.Team] = byMember
		}
		byMember[m.Ref.Member] = m
	}

	var findings []Finding
	for _, teamID := range opts.TeamContracts.IDs() {
		entry := opts.TeamContracts[teamID]
		if entry == nil {
			continue
		}
		findings = append(findings, scanTeamForUndeclaredWriters(teamID, entry, memberIndex[teamID], runtimeDir)...)
	}
	return findings
}

// scanTeamForUndeclaredWriters runs the rule body for one team. Split
// out to keep ruleActualWriterUndeclared readable and to give the test
// suite a per-team entry point that doesn't require a TeamContractRegistry
// fixture.
func scanTeamForUndeclaredWriters(teamID string, contract *LoadedTeamContract, teamMembers map[string]MemberTopics, configDir string) []Finding {
	cutoff := strings.TrimSpace(contract.AttributionValidFrom)
	if cutoff == "" {
		return nil
	}

	knowledgePath := filepath.Join(configDir, "teams", teamID, "shared", "knowledge.jsonl")
	entries, err := loadKnowledgeJSONL(knowledgePath)
	if err != nil {
		// Fresh teams have no knowledge.jsonl; that is not drift.
		// Other read errors are reported as a single warning so the
		// operator notices the broken file without the rule failing
		// the whole validation run.
		if os.IsNotExist(err) {
			return nil
		}
		return []Finding{{
			Rule:     "actual_writer_undeclared",
			Severity: SeverityWarning,
			Member:   MemberRef{Team: teamID},
			Detail:   fmt.Sprintf("could not read team knowledge.jsonl at %s: %v", knowledgePath, err),
		}}
	}

	threshold := contract.FlagExternalWritesPerWeek
	externalByWeek := make(map[string][]knowledgeEntryRow)

	var findings []Finding
	for _, e := range entries {
		// Skip pre-cutoff entries entirely. The migration tool stamps
		// these as kind="legacy" so they do not collide with live
		// validators; we date-compare here to be robust against
		// hand-authored entries that were not migrated through the
		// tool (defense-in-depth, mirroring the doc's § Validator
		// behavior at the boundary first bullet).
		if entryDateBeforeCutoff(e.At, cutoff) {
			continue
		}

		switch e.Attribution.Kind {
		case store.KnowledgeKindAgentMember:
			if f, ok := checkAgentMemberDeclaration(teamID, teamMembers, e); ok {
				findings = append(findings, f)
			}

		case store.KnowledgeKindExternal:
			if threshold > 0 {
				key, kerr := isoWeekKey(e.At)
				if kerr != nil {
					// Malformed timestamp on a post-cutoff
					// entry — surface as defensive
					// finding rather than silently
					// dropping the entry from the count.
					findings = append(findings, Finding{
						Rule:     "attribution_malformed",
						Severity: SeverityError,
						Member:   MemberRef{Team: teamID},
						Detail:   fmt.Sprintf("entry %q has unparseable `at` field %q: %v", e.ID, e.At, kerr),
					})
					continue
				}
				externalByWeek[key] = append(externalByWeek[key], e)
			}

		case store.KnowledgeKindLegacy:
			// `legacy` is the migration's marker for pre-cutoff
			// entries. A `legacy` entry written *after* the cutoff
			// is a contract violation (the migration only stamps
			// legacy on pre-existing rows); fire defensive error.
			findings = append(findings, Finding{
				Rule:     "attribution_malformed",
				Severity: SeverityError,
				Member:   MemberRef{Team: teamID},
				Prefix:   e.Topic,
				Detail:   fmt.Sprintf("post-cutoff entry %q carries kind=\"legacy\"; legacy is reserved for pre-cutoff migration output", e.ID),
			})

		case store.KnowledgeKindWriterSkill,
			store.KnowledgeKindOperatorDirect,
			store.KnowledgeKindInvestigation:
			// Out of scope for this rule — see file header.

		case "":
			findings = append(findings, Finding{
				Rule:     "attribution_malformed",
				Severity: SeverityError,
				Member:   MemberRef{Team: teamID},
				Prefix:   e.Topic,
				Detail:   fmt.Sprintf("entry %q has no attribution.kind; the API enforces this at write time", e.ID),
			})

		default:
			// Unknown kind: the API rejects this at write time.
			// If we see one on disk, the file was hand-edited or
			// a migration broke; fire defensive error.
			findings = append(findings, Finding{
				Rule:     "attribution_malformed",
				Severity: SeverityError,
				Member:   MemberRef{Team: teamID},
				Prefix:   e.Topic,
				Detail:   fmt.Sprintf("entry %q has unknown attribution.kind %q (known kinds: %s)", e.ID, e.Attribution.Kind, strings.Join(store.KnowledgeKinds, ", ")),
			})
		}
	}

	// External-threshold pass. Findings are emitted in stable order
	// (week ascending, then entry index within the week) so list
	// output is diff-friendly and findings.json renders deterministically.
	if threshold > 0 && len(externalByWeek) > 0 {
		findings = append(findings, externalThresholdFindings(teamID, externalByWeek, threshold)...)
	}

	return findings
}

// checkAgentMemberDeclaration asks: did the writer's topics.json declare
// an output[] prefix overlapping the entry's topic? Returns (finding,
// true) when drift is detected; (Finding{}, false) when the write is
// declared (no finding emitted).
//
// `teamMembers` is the per-team slice of memberIndex from
// ruleActualWriterUndeclared — nil/empty when the team has no member
// declarations (still walked, with the "unknown member" branch firing
// for every agent-member entry, which is the correct behavior: a
// declared agent-member writer with zero topics.json files is drift).
func checkAgentMemberDeclaration(teamID string, teamMembers map[string]MemberTopics, e knowledgeEntryRow) (Finding, bool) {
	if e.Attribution.MemberID == nil || strings.TrimSpace(*e.Attribution.MemberID) == "" {
		// API rejects this at write time; defensive check.
		return Finding{
			Rule:     "attribution_malformed",
			Severity: SeverityError,
			Member:   MemberRef{Team: teamID},
			Prefix:   e.Topic,
			Detail:   fmt.Sprintf("entry %q has kind=agent-member but no member_id", e.ID),
		}, true
	}
	memberID := strings.TrimSpace(*e.Attribution.MemberID)

	mt, ok := teamMembers[memberID]
	if !ok {
		return Finding{
			Rule:     "actual_writer_undeclared",
			Severity: SeverityError,
			Member:   MemberRef{Team: teamID, Member: memberID},
			Prefix:   e.Topic,
			Detail:   fmt.Sprintf("entry %q claims kind=agent-member writer %q but no team member of that id exists in the store", e.ID, memberID),
		}, true
	}

	for _, o := range mt.Topics.Output {
		if Overlap(o.Prefix, e.Topic) {
			return Finding{}, false
		}
	}

	return Finding{
		Rule:     "actual_writer_undeclared",
		Severity: SeverityError,
		Member:   MemberRef{Team: teamID, Member: memberID},
		Prefix:   e.Topic,
		Detail:   fmt.Sprintf("agent-member %s/%s wrote topic %q (entry %q) but member's output[] declares no overlapping prefix; either declare it on topics.json or correct the writer", teamID, memberID, e.Topic, e.ID),
	}, true
}

// externalThresholdFindings groups external-kind entries by ISO week and
// emits a warning for each entry past the team's threshold. Findings are
// per-entry rather than per-week so diff-against-previous-run telemetry
// can show exactly which entries crossed.
//
// Within a week, entries are sorted by `at` ascending. Findings fire on
// entries with index >= threshold (i.e., the first `threshold` entries
// in a week are below the cap; everything after is above).
func externalThresholdFindings(teamID string, externalByWeek map[string][]knowledgeEntryRow, threshold int) []Finding {
	weeks := make([]string, 0, len(externalByWeek))
	for w := range externalByWeek {
		weeks = append(weeks, w)
	}
	sort.Strings(weeks)

	var out []Finding
	for _, week := range weeks {
		entries := externalByWeek[week]
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].At < entries[j].At })
		if len(entries) <= threshold {
			continue
		}
		for _, e := range entries[threshold:] {
			out = append(out, Finding{
				Rule:     "actual_writer_undeclared",
				Severity: SeverityWarning,
				Member:   MemberRef{Team: teamID},
				Prefix:   e.Topic,
				Detail:   fmt.Sprintf("kind=external entry %q in ISO week %s pushes the team's external-writes count over the configured threshold (policy.flagExternalWritesPerWeek=%d)", e.ID, week, threshold),
			})
		}
	}
	return out
}

// knowledgeEntryRow is the validator-side projection of store.KnowledgeEntry
// — a narrow view that decodes only the fields the rule reads. Keeping it
// separate from store.KnowledgeEntry lets the validator be tolerant of
// schema drift on fields it does not consume (e.g., a future ContextRef)
// without coupling memberflow to the full store.KnowledgeEntry shape.
//
// AttributionInfo is reused verbatim from store; if the canonical kind /
// spawn-origin enum changes, this rule picks up the new constants
// immediately via the imported store.KnowledgeKind* names.
type knowledgeEntryRow struct {
	ID          string                `json:"id"`
	At          string                `json:"at"`
	Topic       string                `json:"topic"`
	Attribution store.AttributionInfo `json:"attribution"`
}

// loadKnowledgeJSONL reads <path>, parsing one entry per non-blank line.
// JSONL files in this codebase tolerate blank/whitespace-only lines; a
// malformed line is reported as an error (we choose strict here: a single
// bad line should not silently disappear from the validator's view of
// reality).
//
// The returned slice preserves on-disk order, which is the chronological
// append order the API uses; the rule's external-threshold pass relies
// on stable order for deterministic findings.
func loadKnowledgeJSONL(path string) ([]knowledgeEntryRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []knowledgeEntryRow
	scanner := bufio.NewScanner(f)
	// Mature stores can carry multi-kilobyte content blocks; bump the
	// scanner's max line size so we don't truncate the API-validated
	// payload that the rule was specifically designed to inspect.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var row knowledgeEntryRow
		if err := json.Unmarshal([]byte(raw), &row); err != nil {
			return nil, fmt.Errorf("parse %s line %d: %w", path, lineNum, err)
		}
		out = append(out, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return out, nil
}

// entryDateBeforeCutoff returns true when the entry's `at` falls strictly
// before the team's cutoff date. Both inputs are interpreted as ISO-8601:
// `at` is RFC3339-style (e.g. "2026-05-04T15:32:11Z") on production
// entries; cutoff is the YYYY-MM-DD form. We compare on the date portion
// of `at`, lexically — the canonical RFC3339 date prefix is exactly the
// 10 characters that lex-sort identically to the calendar order, so no
// time-zone parsing is required for the boundary check.
//
// Empty or short `at` values are treated as "not before cutoff" (i.e. NOT
// skipped) so the rule still surfaces them through the kind-switch's
// malformed paths instead of silently dropping them.
func entryDateBeforeCutoff(at, cutoff string) bool {
	at = strings.TrimSpace(at)
	cutoff = strings.TrimSpace(cutoff)
	if cutoff == "" || len(at) < 10 {
		return false
	}
	return at[:10] < cutoff
}

// isoWeekKey derives the ISO-8601 year-week key (e.g. "2026-W18") for an
// entry's `at` timestamp. The deterministic weekday/week math comes from
// time.Time.ISOWeek; this helper is a thin parser around it that accepts
// the two on-disk timestamp forms the migration tool and the API
// produce: full RFC3339 ("2026-05-04T15:32:11Z") and date-only
// ("2026-05-04").
func isoWeekKey(at string) (string, error) {
	at = strings.TrimSpace(at)
	if at == "" {
		return "", fmt.Errorf("empty timestamp")
	}
	layouts := []string{time.RFC3339, "2006-01-02"}
	var lastErr error
	for _, layout := range layouts {
		t, err := time.Parse(layout, at)
		if err == nil {
			y, w := t.ISOWeek()
			return fmt.Sprintf("%04d-W%02d", y, w), nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("unparseable timestamp %q: %w", at, lastErr)
}
