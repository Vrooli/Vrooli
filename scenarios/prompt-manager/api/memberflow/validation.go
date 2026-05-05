package memberflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Severity classifies a validation finding. Errors fail the
// `prompt-manager graph topics` exit code; warnings are informational.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Finding is one validation result. Multiple findings may apply to a single
// member or prefix.
type Finding struct {
	Rule     string    `json:"rule"`
	Severity Severity  `json:"severity"`
	Member   MemberRef `json:"member,omitempty"`
	Prefix   string    `json:"prefix,omitempty"`
	Detail   string    `json:"detail"`
}

// ValidationOptions configures runtime checks that depend on filesystem state
// (taxonomy registry, team-contract registry, classifier-skill files, PoR
// file existence). Repo root is required for dangling_por_sink,
// unknown_taxonomy, and non_portable_classifier checks; StoreDir is
// required for the team-contract registry lazy-load.
type ValidationOptions struct {
	// RepoRoot is the absolute path to the repository root (the parent of
	// docs/, scenarios/, etc.). Used for por_file existence,
	// classifier-skill content scans, and as a fallback location for
	// taxonomies when Taxonomies is nil.
	RepoRoot string

	// StoreDir is the absolute path to the prompt-manager store
	// (scenarios/prompt-manager/store/). Used as the fallback location
	// for team contracts when TeamContracts is nil; ignored when set
	// explicitly. When both StoreDir and TeamContracts are empty, the
	// dangling_evidence_decision rule is skipped.
	StoreDir string

	// SkillIDs is the set of skill ids known to the skill registry.
	// Empty disables skill-existence cross-checks.
	SkillIDs map[string]bool

	// SkillPaths maps skill id -> absolute path to its SKILL.md content,
	// used by ruleNonPortableClassifier to grep classifier skills for
	// forbidden coupling. Optional: empty disables the rule.
	SkillPaths map[string]string

	// Taxonomies is the loaded taxonomy registry used by
	// ruleUnknownTaxonomy and ruleMissingDestinationSchema. When nil and
	// RepoRoot is set, Validate loads taxonomies on demand.
	Taxonomies TaxonomyRegistry

	// TeamContracts is the loaded team-contract registry used by
	// ruleDanglingEvidenceDecision. When nil and StoreDir is set,
	// Validate loads contracts on demand. An explicit empty registry
	// disables the rule (lets tests assert behavior without contracts
	// without needing a store on disk).
	TeamContracts TeamContractRegistry
}

// ValidationResult bundles findings with summary counts.
type ValidationResult struct {
	Findings []Finding `json:"findings"`
	Errors   int       `json:"errors"`
	Warnings int       `json:"warnings"`
}

// ExitCode returns 1 if any error-severity finding is present, otherwise 0.
func (r ValidationResult) ExitCode() int {
	if r.Errors > 0 {
		return 1
	}
	return 0
}

// Validate runs every cross-graph validation rule against the supplied member
// declarations. Shape-level errors (caught by Topics.Validate) should already
// have prevented bad declarations from reaching here; if any have slipped
// through (e.g., the loader was bypassed), they are still reported.
func Validate(members []MemberTopics, opts ValidationOptions) ValidationResult {
	if opts.Taxonomies == nil && strings.TrimSpace(opts.RepoRoot) != "" {
		if reg, err := LoadAllTaxonomies(opts.RepoRoot); err == nil {
			opts.Taxonomies = reg
		}
	}
	if opts.TeamContracts == nil && strings.TrimSpace(opts.StoreDir) != "" {
		if reg, err := LoadAllTeamContracts(opts.StoreDir); err == nil {
			opts.TeamContracts = reg
		}
	}

	var findings []Finding

	findings = append(findings, ruleConflictingDrain(members)...)
	findings = append(findings, ruleOrphanOutput(members, opts)...)
	findings = append(findings, ruleOrphanInput(members)...)
	findings = append(findings, ruleUnreadRequired(members)...)
	findings = append(findings, ruleWildcardSourceMisuse(members)...)
	findings = append(findings, ruleUnknownTaxonomy(members, opts)...)
	findings = append(findings, ruleNonPortableClassifier(members, opts)...)
	findings = append(findings, ruleMissingDestinationSchema(members, opts)...)
	findings = append(findings, ruleDanglingPORSink(members, opts)...)
	findings = append(findings, ruleDanglingEvidenceDecision(members, opts)...)
	findings = append(findings, ruleActualWriterUndeclared(members, opts)...)

	// stalled_drain and piling_inbox depend on team-knowledge queue depth +
	// age; those are computed by the CLI layer (which has access to the
	// prompt-manager team knowledge-list output) and appended to the
	// findings before display. The pure-Go validation cannot synthesize
	// those without I/O; they are warnings, never errors, so omitting them
	// here only loses warnings — never spurious failures.

	sortFindings(findings)

	r := ValidationResult{Findings: findings}
	for _, f := range findings {
		switch f.Severity {
		case SeverityError:
			r.Errors++
		case SeverityWarning:
			r.Warnings++
		}
	}
	return r
}

func sortFindings(f []Finding) {
	sort.Slice(f, func(i, j int) bool {
		if f[i].Rule != f[j].Rule {
			return f[i].Rule < f[j].Rule
		}
		if f[i].Member.String() != f[j].Member.String() {
			return f[i].Member.String() < f[j].Member.String()
		}
		return f[i].Prefix < f[j].Prefix
	})
}

// ruleConflictingDrain — two members claim drain duty for overlapping intake
// prefixes, which would race the inbox-router-drain mechanism.
func ruleConflictingDrain(members []MemberTopics) []Finding {
	var out []Finding
	type claim struct {
		ref    MemberRef
		prefix string
	}
	var claims []claim
	for _, m := range members {
		for _, e := range m.Topics.Intake {
			claims = append(claims, claim{m.Ref, e.Prefix})
		}
	}
	for i := 0; i < len(claims); i++ {
		for j := i + 1; j < len(claims); j++ {
			if claims[i].ref == claims[j].ref {
				continue
			}
			if !Overlap(claims[i].prefix, claims[j].prefix) {
				continue
			}
			out = append(out,
				Finding{
					Rule:     "conflicting_drain",
					Severity: SeverityError,
					Member:   claims[i].ref,
					Prefix:   claims[i].prefix,
					Detail:   fmt.Sprintf("intake prefix %q overlaps with %s/%s drain on %q", claims[i].prefix, claims[j].ref.Team, claims[j].ref.Member, claims[j].prefix),
				},
				Finding{
					Rule:     "conflicting_drain",
					Severity: SeverityError,
					Member:   claims[j].ref,
					Prefix:   claims[j].prefix,
					Detail:   fmt.Sprintf("intake prefix %q overlaps with %s/%s drain on %q", claims[j].prefix, claims[i].ref.Team, claims[i].ref.Member, claims[i].prefix),
				},
			)
		}
	}
	return out
}

// ruleOrphanOutput — a member's output prefix has no consumer.
//
// "Consumer" is whatever consumerSet (consumer_set.go) recognizes as a
// reader of a topic prefix. The set aggregates every read-side
// declaration on every member's topics.json: `intake[]` (drain),
// `required_read[]` (heartbeat-prompt context), and `evidence_consumed[]`
// (decision rationale). This rule is intentionally agnostic about which
// kinds of consumer exist — that knowledge lives in the set, so adding a
// new consumer kind is a one-line change in buildConsumerSet.
//
// Non-knowledge destinations (decision, por_file, capability_gap,
// skill_proposal, backlog) are sinks, never orphans, and are skipped
// before consulting the consumer set.
func ruleOrphanOutput(members []MemberTopics, opts ValidationOptions) []Finding {
	_ = opts
	var out []Finding

	consumers := buildConsumerSet(members)

	for _, m := range members {
		for _, o := range m.Topics.Output {
			// Non-knowledge destinations are sinks, never orphans.
			if o.DestinationKind != DestinationKnowledge {
				continue
			}
			if consumers.Overlaps(o.Prefix) {
				continue
			}
			// Warning, not error: many knowledge outputs are
			// operator-read snapshots (audits, ledgers, run
			// lessons) that have no member-level drain. The
			// smell is "no peer member claims this prefix" —
			// operators decide whether that's intentional.
			out = append(out, Finding{
				Rule:     "orphan_output",
				Severity: SeverityWarning,
				Member:   m.Ref,
				Prefix:   o.Prefix,
				Detail:   fmt.Sprintf("output prefix %q has no peer-member consumer (operator-only snapshot is acceptable; pair with an intake if this should be drained)", o.Prefix),
			})
		}
	}
	return out
}

// ruleOrphanInput — a member's intake prefix has no producer.
//
// Producer types:
//   - another member's output prefix overlaps the intake prefix
//   - the intake's source_team plus a member of that team writing the prefix
//   - any external_producer named on this member counts as a potential producer
//   - source_team == "*" (universal-source intake) — any team may write,
//     so producer existence is not enforced; misuse is caught by
//     ruleWildcardSourceMisuse
func ruleOrphanInput(members []MemberTopics) []Finding {
	var out []Finding

	// Build set of output prefixes.
	type outputClaim struct {
		ref    MemberRef
		prefix string
	}
	var outputs []outputClaim
	for _, m := range members {
		for _, o := range m.Topics.Output {
			outputs = append(outputs, outputClaim{m.Ref, o.Prefix})
		}
	}

	for _, m := range members {
		hasExternal := len(m.Topics.ExternalProducers) > 0
		for _, in := range m.Topics.Intake {
			if hasExternal {
				continue // external producer is sufficient evidence
			}
			if in.SourceTeam != nil && *in.SourceTeam == SourceTeamWildcard {
				continue // universal-source intake: any team may write
			}
			matched := false
			for _, o := range outputs {
				if Overlap(o.prefix, in.Prefix) {
					matched = true
					break
				}
			}
			if !matched {
				out = append(out, Finding{
					Rule:     "orphan_input",
					Severity: SeverityError,
					Member:   m.Ref,
					Prefix:   in.Prefix,
					Detail:   fmt.Sprintf("intake prefix %q has no producer (no member's output overlaps and no external_producer is declared)", in.Prefix),
				})
			}
		}
	}
	return out
}

// ruleUnreadRequired — a member's required_read[] prefix has no producer.
//
// Mirrors ruleOrphanInput's producer-discovery walk but for `required_read[]`
// entries. A required_read prefix says "I need this prefix's recent state
// rendered into my heartbeat prompt every tick"; a finding here means no
// member's `output[]` declares the prefix, so the agent will load an empty
// (or stale) section and the operator's mental model has drifted from the
// declared graph.
//
// Two design points distinguish this rule from ruleOrphanInput:
//
//  1. Severity is warning, not error. A required_read with no peer producer
//     is information ("you've declared you read this prefix; document its
//     producer or rename to match a real one"), not a structural fault that
//     blocks routing. Phase 4.1 promotes the rule to error after the
//     ecosystem is reconciled.
//
//  2. Member-level `external_producers` is not treated as a satisfying
//     anchor here. Where ruleOrphanInput is a hard gate and uses
//     external_producers as a coarse-grained "trust the operator's
//     producer-side documentation" escape hatch, this rule's purpose is to
//     surface drift between declared read prefixes and declared write
//     prefixes — drift that ruleOrphanInput's coarse skip would mask. The
//     plan calls out specific drift cases the rule must catch
//     (vision-walk-prep, portfolio-snapshot, outcome-snapshot, deep-audit,
//     etc.) where the consuming members do declare external_producers.
//     Honoring the loose skip would silently hide them. Per-prefix anchors
//     are a future refinement; until then, the warning is the operator's
//     prompt to either rename to match a producer (Phase 1.7) or accept
//     the warning as documenting an external-only write.
//
// Skipped per-entry conditions:
//   - source_team == "*" (universal-source read): any team may write, so
//     producer existence is not enforced. Mirrors ruleOrphanInput's
//     wildcard treatment; the corresponding misuse pattern (wildcard with
//     no documented external anchor) is caught by ruleWildcardSourceMisuse.
func ruleUnreadRequired(members []MemberTopics) []Finding {
	var out []Finding

	type outputClaim struct {
		ref    MemberRef
		prefix string
	}
	var outputs []outputClaim
	for _, m := range members {
		for _, o := range m.Topics.Output {
			outputs = append(outputs, outputClaim{m.Ref, o.Prefix})
		}
	}

	for _, m := range members {
		for _, r := range m.Topics.RequiredRead {
			if r.SourceTeam != nil && *r.SourceTeam == SourceTeamWildcard {
				continue
			}
			matched := false
			for _, o := range outputs {
				if Overlap(o.prefix, r.Prefix) {
					matched = true
					break
				}
			}
			if matched {
				continue
			}
			out = append(out, Finding{
				Rule:     "unread_required",
				Severity: SeverityWarning,
				Member:   m.Ref,
				Prefix:   r.Prefix,
				Detail:   fmt.Sprintf("required_read prefix %q has no peer-member producer (no member's output[] overlaps; rename to match a producer's declared prefix or add a member that writes this prefix)", r.Prefix),
			})
		}
	}
	return out
}

// ruleWildcardSourceMisuse — a universal-source intake (source_team == "*")
// without any documented producer-side anchor. The wildcard is a real
// declaration of intent ("any team may write here") but it must point at a
// concrete writer-side anchor — typically a writer skill listed in
// ExternalProducers — so the topology is auditable. Empty
// ExternalProducers + wildcard source means "I made it universal but
// forgot to document who actually writes," which is the misuse.
func ruleWildcardSourceMisuse(members []MemberTopics) []Finding {
	var out []Finding
	for _, m := range members {
		if len(m.Topics.ExternalProducers) > 0 {
			continue
		}
		for _, in := range m.Topics.Intake {
			if in.SourceTeam == nil || *in.SourceTeam != SourceTeamWildcard {
				continue
			}
			out = append(out, Finding{
				Rule:     "wildcard_source_misuse",
				Severity: SeverityWarning,
				Member:   m.Ref,
				Prefix:   in.Prefix,
				Detail:   fmt.Sprintf("intake %q declares source_team=%q but external_producers is empty; document the producer-side anchor (e.g., the writer skill or external system that produces entries)", in.Prefix, SourceTeamWildcard),
			})
		}
	}
	return out
}

// ruleUnknownTaxonomy reports two related error conditions on every intake
// entry:
//
//   - missing_taxonomy (error): the intake declares no taxonomy id.
//   - unknown_taxonomy (error): the intake names a taxonomy that does not
//     resolve in the registry.
//
// Skipped entirely when Taxonomies is empty AND RepoRoot is empty (e.g. unit
// tests that don't load taxonomies).
func ruleUnknownTaxonomy(members []MemberTopics, opts ValidationOptions) []Finding {
	if opts.Taxonomies == nil && strings.TrimSpace(opts.RepoRoot) == "" {
		return nil
	}
	registry := opts.Taxonomies
	if registry == nil {
		registry = TaxonomyRegistry{}
	}
	var out []Finding
	for _, m := range members {
		for _, in := range m.Topics.Intake {
			id := strings.TrimSpace(in.Taxonomy)
			if id == "" {
				out = append(out, Finding{
					Rule:     "missing_taxonomy",
					Severity: SeverityError,
					Member:   m.Ref,
					Prefix:   in.Prefix,
					Detail:   fmt.Sprintf("intake %q has no taxonomy id; set intake[].taxonomy to a registered taxonomy", in.Prefix),
				})
				continue
			}
			if _, ok := registry[id]; !ok {
				out = append(out, Finding{
					Rule:     "unknown_taxonomy",
					Severity: SeverityError,
					Member:   m.Ref,
					Prefix:   in.Prefix,
					Detail:   fmt.Sprintf("taxonomy %q does not resolve under docs/", id),
				})
			}
		}
	}
	return out
}

// classifierForbiddenSubstrings are patterns a portable classifier skill
// must NOT contain. They mark coupling to a specific inbox prefix,
// drain-procedure command, or team store — content that belongs in the
// generated heartbeat section, not in a reusable judgment skill. Hits fire
// `non_portable_classifier`.
//
// Note: legitimate documentation paths under docs/<domain>/ (e.g.
// docs/monetization/CATALOG.md) are *not* coupling and are not flagged.
// The patterns target either the specific inbox topic-prefix conventions
// or the knowledge-CLI write/delete verbs that perform routing.
var classifierForbiddenSubstrings = []string{
	// Inbox topic-prefix coupling.
	"research-inbox/",
	"opportunity-inbox/",
	"validation-inbox/",
	// Knowledge CLI verbs that perform routing (read-only knowledge-list
	// is fine in member prose, but a portable judgment skill should not
	// invoke these).
	"prompt-manager team knowledge-update",
	"prompt-manager team knowledge-delete",
	"prompt-manager team knowledge-add",
	"team knowledge-update",
	"team knowledge-delete",
	"team knowledge-add",
	// Team-store filesystem coupling. The `teams/<id>/members/<id>`
	// pattern only appears when a skill is reaching into a specific
	// team's per-member directory — exactly the coupling we want to
	// forbid.
	"teams/marketing-crew/members",
	"teams/monetization/members",
}

// ruleNonPortableClassifier scans each intake's classifier skill for
// forbidden coupling content. Skipped when SkillPaths is empty.
func ruleNonPortableClassifier(members []MemberTopics, opts ValidationOptions) []Finding {
	if len(opts.SkillPaths) == 0 {
		return nil
	}
	var out []Finding
	scanned := map[string]bool{}
	for _, m := range members {
		for _, in := range m.Topics.Intake {
			id := strings.TrimSpace(in.ClassifierSkill)
			if id == "" {
				continue
			}
			if scanned[id] {
				continue
			}
			scanned[id] = true

			path, ok := opts.SkillPaths[id]
			if !ok {
				out = append(out, Finding{
					Rule:     "non_portable_classifier",
					Severity: SeverityError,
					Member:   m.Ref,
					Prefix:   in.Prefix,
					Detail:   fmt.Sprintf("classifier_skill %q is not in the skill registry", id),
				})
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				out = append(out, Finding{
					Rule:     "non_portable_classifier",
					Severity: SeverityWarning,
					Member:   m.Ref,
					Prefix:   in.Prefix,
					Detail:   fmt.Sprintf("classifier_skill %q SKILL.md not readable at %s: %v", id, path, err),
				})
				continue
			}
			content := string(data)
			for _, sub := range classifierForbiddenSubstrings {
				if strings.Contains(content, sub) {
					out = append(out, Finding{
						Rule:     "non_portable_classifier",
						Severity: SeverityError,
						Member:   m.Ref,
						Prefix:   in.Prefix,
						Detail:   fmt.Sprintf("classifier_skill %q SKILL.md contains forbidden pattern %q (judgment skills must be member-agnostic)", id, sub),
					})
					break
				}
			}
		}
	}
	return out
}

// ruleMissingDestinationSchema warns when an output entry names a schema
// that no taxonomy declares. Skipped when Taxonomies is empty.
func ruleMissingDestinationSchema(members []MemberTopics, opts ValidationOptions) []Finding {
	if opts.Taxonomies == nil {
		return nil
	}
	var out []Finding
	for _, m := range members {
		for _, o := range m.Topics.Output {
			id := strings.TrimSpace(o.Schema)
			if id == "" {
				continue
			}
			if !opts.Taxonomies.HasSchema(id) {
				out = append(out, Finding{
					Rule:     "missing_destination_schema",
					Severity: SeverityWarning,
					Member:   m.Ref,
					Prefix:   o.Prefix,
					Detail:   fmt.Sprintf("output schema %q is not declared by any loaded taxonomy", id),
				})
			}
		}
	}
	return out
}

// ruleDanglingPORSink — output entries with destination_kind=por_file must
// reference a destination_path that exists. Skipped when RepoRoot is empty.
func ruleDanglingPORSink(members []MemberTopics, opts ValidationOptions) []Finding {
	if opts.RepoRoot == "" {
		return nil
	}
	var out []Finding
	for _, m := range members {
		for _, o := range m.Topics.Output {
			if o.DestinationKind != DestinationPORFile {
				continue
			}
			if o.DestinationPath == nil || strings.TrimSpace(*o.DestinationPath) == "" {
				// shape-level error; covered by Topics.Validate. Skip.
				continue
			}
			full := filepath.Join(opts.RepoRoot, *o.DestinationPath)
			if _, err := os.Stat(full); err != nil {
				out = append(out, Finding{
					Rule:     "dangling_por_sink",
					Severity: SeverityError,
					Member:   m.Ref,
					Prefix:   o.Prefix,
					Detail:   fmt.Sprintf("destination_path %q does not exist (resolved: %s)", *o.DestinationPath, full),
				})
			}
		}
	}
	return out
}

// ruleDanglingEvidenceDecision — every `evidence_consumed[].for_decisions[]`
// id on every member's topics.json must resolve against some team's
// `team.json::operatingContract.decisionContexts`. A typo or a context that
// has been removed from team.json without scrubbing the consuming
// member's topics.json shows up here as an error finding.
//
// Skipped (no findings) when the team-contract registry is empty: this
// happens in unit tests that don't fixture team contracts and on
// scaffolds that haven't yet defined any team. The plan's intent is to
// reject silent dead references, not to demand contracts exist.
//
// Severity is error: a dangling decision-context id is a structural
// declaration drift the operator must reconcile (rename the id, remove the
// evidence entry, or add the missing decision-context to team.json).
func ruleDanglingEvidenceDecision(members []MemberTopics, opts ValidationOptions) []Finding {
	if len(opts.TeamContracts) == 0 {
		return nil
	}
	var out []Finding
	for _, m := range members {
		for _, ev := range m.Topics.EvidenceConsumed {
			seen := map[string]bool{}
			for _, decisionID := range ev.ForDecisions {
				id := strings.TrimSpace(decisionID)
				if id == "" {
					// shape-level error; covered by Topics.Validate.
					continue
				}
				if seen[id] {
					// One finding per (member, prefix, id).
					continue
				}
				seen[id] = true
				if opts.TeamContracts.HasDecisionContext(id) {
					continue
				}
				out = append(out, Finding{
					Rule:     "dangling_evidence_decision",
					Severity: SeverityError,
					Member:   m.Ref,
					Prefix:   ev.Prefix,
					Detail:   fmt.Sprintf("evidence_consumed[].for_decisions references decision-context %q which is not declared in any team.json::operatingContract.decisionContexts", id),
				})
			}
		}
	}
	return out
}

// LoadSkillIDs reads the local skill registry and returns the set of known
// skill IDs. The registry layout matches the prompt-manager store:
//
//	<storeDir>/skills/packs/*/<skill-id>/skill.json
//
// Returns an empty map without error when the registry is missing — callers
// that need cross-checks should treat that as "skip the check," not as a
// failure.
func LoadSkillIDs(storeDir string) (map[string]bool, error) {
	out := make(map[string]bool)
	packsDir := filepath.Join(storeDir, "skills", "packs")
	packs, err := os.ReadDir(packsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, fmt.Errorf("memberflow: read packs %q: %w", packsDir, err)
	}
	for _, p := range packs {
		if !p.IsDir() {
			continue
		}
		skillRoots, err := os.ReadDir(filepath.Join(packsDir, p.Name()))
		if err != nil {
			continue
		}
		for _, s := range skillRoots {
			if !s.IsDir() {
				continue
			}
			if _, err := os.Stat(filepath.Join(packsDir, p.Name(), s.Name(), "skill.json")); err == nil {
				out[s.Name()] = true
			}
		}
	}
	return out, nil
}

// LoadSkillPaths returns skill id -> absolute path to its SKILL.md file,
// for every skill in the registry that has both a skill.json and a SKILL.md.
// Used by ruleNonPortableClassifier.
func LoadSkillPaths(storeDir string) (map[string]string, error) {
	out := make(map[string]string)
	packsDir := filepath.Join(storeDir, "skills", "packs")
	packs, err := os.ReadDir(packsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, fmt.Errorf("memberflow: read packs %q: %w", packsDir, err)
	}
	for _, p := range packs {
		if !p.IsDir() {
			continue
		}
		skillRoots, err := os.ReadDir(filepath.Join(packsDir, p.Name()))
		if err != nil {
			continue
		}
		for _, s := range skillRoots {
			if !s.IsDir() {
				continue
			}
			skillJSON := filepath.Join(packsDir, p.Name(), s.Name(), "skill.json")
			skillMD := filepath.Join(packsDir, p.Name(), s.Name(), "SKILL.md")
			if _, err := os.Stat(skillJSON); err != nil {
				continue
			}
			if _, err := os.Stat(skillMD); err != nil {
				continue
			}
			out[s.Name()] = skillMD
		}
	}
	return out, nil
}
