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

// ValidationOptions configures runtime checks that depend on filesystem state
// (taxonomy registry, team-contract registry, prose-scan roots, PoR file
// existence). Repo root is required for dangling_por_sink and
// unknown_taxonomy checks; StoreDir is required for the team-contract
// registry lazy-load.
type ValidationOptions struct {
	// RepoRoot is the absolute path to the repository root (the parent of
	// docs/, scenarios/, etc.). Used for por_file existence and as a
	// fallback location for taxonomies when Taxonomies is nil.
	RepoRoot string

	// StoreDir is the absolute path to the prompt-manager Config-class
	// store (scenarios/prompt-manager/store/). Used as the fallback
	// location for team contracts when TeamContracts is nil; ignored when
	// set explicitly.
	StoreDir string

	// RuntimeDataDir is the absolute path to the prompt-manager
	// RuntimeData root (api-core/storage ClassData). Retained for runtime
	// enrichments that read mutable execution state. Team corpus data is read
	// through the source-ledger adapter, not from this tree.
	RuntimeDataDir string

	// SkillIDs is the set of skill ids known to the skill registry.
	// Empty disables skill-existence cross-checks.
	SkillIDs map[string]bool

	// Taxonomies is the loaded taxonomy registry used by
	// ruleUnknownTaxonomy and ruleMissingDestinationSchema. When nil and
	// RepoRoot is set, Validate loads taxonomies on demand.
	Taxonomies TaxonomyRegistry

	// TeamContracts is the loaded team-contract registry used by
	// runtime attribution. When nil and StoreDir is set,
	// Validate loads contracts on demand. An explicit empty registry
	// disables the rule (lets tests assert behavior without contracts
	// without needing a store on disk).
	TeamContracts TeamContractRegistry

	// ScanRoots are absolute filesystem roots the Pillar 2 prose scanner
	// (ruleProseTopicLeak) walks. When empty AND RepoRoot is empty, the
	// rule is silent (unit tests that pass synthetic members[] without a
	// backing tree get this). When ScanRoots is empty AND RepoRoot is
	// set, the scanner derives its targets from RepoRoot's
	// `scenarios/prompt-manager/store/` and `docs/` subtrees per
	// docs/agent-system/PROSE_SCAN_TARGETS.md. Tests that need a focused
	// scan (e.g., a temp-dir fixture rooted somewhere other than the live
	// repo) supply ScanRoots explicitly.
	ScanRoots []string

	// WriterSkillProducers is the union of `writes_to[]` prefixes
	// declared by every skill tagged "writer-skill". Used by
	// ruleUnreadRequired to resolve required_read[] entries against
	// writer-skill producers — Contract Decision C7 (the writer-skill
	// registry is the producer-side declaration for skill-written
	// prefixes; classifier and generic skills must be portable). When
	// nil and RepoRoot is set, Validate lazy-loads via
	// LoadWriterSkillProducers. An explicit empty slice disables the
	// lookup; a nil with RepoRoot empty is also a silent no-op.
	WriterSkillProducers []string
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
	if opts.WriterSkillProducers == nil && strings.TrimSpace(opts.RepoRoot) != "" {
		if producers, err := LoadWriterSkillProducers(opts.RepoRoot); err == nil {
			opts.WriterSkillProducers = producers
		}
	}

	registry, err := DefaultRuleRegistry()
	if err != nil {
		// Default registration is static; preserving a deterministic result is
		// safer than silently dropping validation if a future edit breaks it.
		return ValidationResult{Findings: []Finding{{Rule: "rule_registry_invalid", Severity: SeverityError, Detail: err.Error()}}, Errors: 1}
	}
	ctx := RuleContext{Members: members, Options: opts}
	var findings []Finding
	for _, registered := range registry.RulesForPass(RulePassTopic) {
		rule, ok := registered.(findingRule)
		if !ok {
			// Registration rejects this, so reaching it means the registry was
			// built by another path. Report rather than skip silently.
			findings = append(findings, Finding{
				Rule:     "rule_registry_invalid",
				Severity: SeverityError,
				Detail:   fmt.Sprintf("topic-pass rule %q does not implement findingRule", registered.ID()),
			})
			continue
		}
		if !registered.AppliesTo(ctx) {
			continue
		}
		findings = append(findings, rule.CheckFindings(ctx)...)
	}

	// stalled_drain and piling_inbox depend on team-knowledge queue depth +
	// age; those are computed by the CLI layer (which has access to the
	// prompt-manager team knowledge-list output) and appended to the
	// findings before display. The pure-Go validation cannot synthesize
	// those without I/O; they are warnings, never errors, so omitting them
	// here only loses warnings — never spurious failures.

	sortFindings(findings)
	StampFindingKinds(findings)

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
		if f[i].Subject() != f[j].Subject() {
			return f[i].Subject() < f[j].Subject()
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
					Team:     claims[i].ref.Team, Member: claims[i].ref.Member,
					Prefix: claims[i].prefix,
					Detail: fmt.Sprintf("intake prefix %q overlaps with %s/%s drain on %q", claims[i].prefix, claims[j].ref.Team, claims[j].ref.Member, claims[j].prefix),
				},
				Finding{
					Rule:     "conflicting_drain",
					Severity: SeverityError,
					Team:     claims[j].ref.Team, Member: claims[j].ref.Member,
					Prefix: claims[j].prefix,
					Detail: fmt.Sprintf("intake prefix %q overlaps with %s/%s drain on %q", claims[j].prefix, claims[i].ref.Team, claims[i].ref.Member, claims[i].prefix),
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
// Non-knowledge destinations (por_file, work_item,
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
				Team:     m.Ref.Team, Member: m.Ref.Member,
				Prefix: o.Prefix,
				Detail: fmt.Sprintf("output prefix %q has no declared consumer (operator-only snapshot is acceptable; pair with an intake if this should be drained)", o.Prefix),
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
					Team:     m.Ref.Team, Member: m.Ref.Member,
					Prefix: in.Prefix,
					Detail: fmt.Sprintf("intake prefix %q has no producer (no member's output overlaps and no external_producer is declared)", in.Prefix),
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
// member's `output[]` and no writer-skill `writes_to[]` declares the prefix,
// so the agent will load an empty (or stale) section and the operator's
// mental model has drifted from the declared graph.
//
// Producer sources consulted, in order:
//
//  1. Any member's `output[]` whose prefix overlaps the required_read prefix
//     (cross-team allowed; producer-side ownership is enforced elsewhere).
//
//  2. Any writer-skill `writes_to[]` entry whose prefix overlaps the
//     required_read prefix. Writer-skill writes_to is the producer-side
//     declaration for skill-written prefixes — treating it as a
//     first-class declared write keeps the rule's intent ("declared
//     reads must overlap declared writes") coherent across both members
//     and skills. The skill registry is loaded via
//     opts.WriterSkillProducers; tests that pass synthetic members
//     without WriterSkillProducers see writer-skill consultation as a
//     no-op.
//
// Severity is error. The rule is a CI gate:
// any unmatched required_read fails `prompt-manager graph topics` and
// must be resolved by renaming the read to overlap a producer, adding
// a member output, or adding the prefix to a writer-skill writes_to[].
//
// Member-level `external_producers` is intentionally NOT a satisfying
// anchor here. Where ruleOrphanInput is a hard gate and uses
// external_producers as a coarse-grained "trust the operator's producer-side
// documentation" escape hatch, this rule's purpose is to surface drift
// between declared read prefixes and declared write prefixes. external_producers
// is a freeform list of names without a writes_to[] equivalent; honoring it
// would silently hide drift the writer-skill writes_to[] consultation
// (above) is designed to catch. The contrast: writer-skill writes_to[] is a
// concrete declared-write list checkable against the read; external_producers
// is a textual hint without that structure.
//
// Skipped per-entry conditions:
//   - source_team == "*" (universal-source read): any team may write, so
//     producer existence is not enforced. Mirrors ruleOrphanInput's
//     wildcard treatment; the corresponding misuse pattern (wildcard with
//     no documented external anchor) is caught by ruleWildcardSourceMisuse.
func ruleUnreadRequired(members []MemberTopics, opts ValidationOptions) []Finding {
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
			if !matched {
				for _, p := range opts.WriterSkillProducers {
					if Overlap(p, r.Prefix) {
						matched = true
						break
					}
				}
			}
			if matched {
				continue
			}
			out = append(out, Finding{
				Rule:     "unread_required",
				Severity: SeverityError,
				Team:     m.Ref.Team, Member: m.Ref.Member,
				Prefix: r.Prefix,
				Detail: fmt.Sprintf("required_read prefix %q has no producer (no member's output[] overlaps and no writer-skill declares it in skill.json::writes_to[]; rename to match a producer's declared prefix, add a member that writes it, or add it to a writer skill's writes_to[])", r.Prefix),
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
				Team:     m.Ref.Team, Member: m.Ref.Member,
				Prefix: in.Prefix,
				Detail: fmt.Sprintf("intake %q declares source_team=%q but external_producers is empty; document the producer-side anchor (e.g., the writer skill or external system that produces entries)", in.Prefix, SourceTeamWildcard),
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
					Team:     m.Ref.Team, Member: m.Ref.Member,
					Prefix: in.Prefix,
					Detail: fmt.Sprintf("intake %q has no taxonomy id; set intake[].taxonomy to a registered taxonomy", in.Prefix),
				})
				continue
			}
			if _, ok := registry[id]; !ok {
				out = append(out, Finding{
					Rule:     "unknown_taxonomy",
					Severity: SeverityError,
					Team:     m.Ref.Team, Member: m.Ref.Member,
					Prefix: in.Prefix,
					Detail: fmt.Sprintf("taxonomy %q does not resolve under docs/", id),
				})
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
					Team:     m.Ref.Team, Member: m.Ref.Member,
					Prefix: o.Prefix,
					Detail: fmt.Sprintf("output schema %q is not declared by any loaded taxonomy", id),
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
					Team:     m.Ref.Team, Member: m.Ref.Member,
					Prefix: o.Prefix,
					Detail: fmt.Sprintf("destination_path %q does not exist (resolved: %s)", *o.DestinationPath, full),
				})
			}
		}
	}
	return out
}

// LoadSkillIDs reads the local skill registry and returns the set of known
// skill IDs. The registry layout matches the prompt-manager store:
//
//	<configDir>/skills/packs/*/<skill-id>/skill.json
//
// Returns an empty map without error when the registry is missing — callers
// that need cross-checks should treat that as "skip the check," not as a
// failure.
func LoadSkillIDs(configDir string) (map[string]bool, error) {
	out := make(map[string]bool)
	packsDir := filepath.Join(configDir, "skills", "packs")
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

// ruleTeamRoleMemberDrift cross-checks `roles.json` role ids against the team's
// contract member ids. Both directions are drift:
//
//   - a role with no matching member describes someone who does not exist;
//   - a member with no matching role has no role description at all.
//
// Nothing consumed roles.json at runtime — every team currently sets
// `showOrgContext: false`, so the file never reaches a prompt — and nothing
// validated it either. A file that is neither read nor checked rots silently,
// and one had: three of four role entries on a live team named or omitted the
// wrong member. This rule is what makes the file's correctness observable
// regardless of whether a given team renders it.
//
// Teams with no roles.json are skipped rather than flagged. The file is
// optional, and demanding one from every team would be a different decision
// than checking the ones that exist.
func ruleTeamRoleMemberDrift(opts ValidationOptions) []Finding {
	if len(opts.TeamContracts) == 0 {
		return nil
	}
	var out []Finding
	for _, teamID := range opts.TeamContracts.IDs() {
		contract := opts.TeamContracts[teamID]
		if contract == nil || contract.Contract == nil || contract.RolesSourcePath == "" {
			continue
		}

		memberIDs := make(map[string]bool, len(contract.Contract.Members))
		for memberID := range contract.Contract.Members {
			memberIDs[memberID] = true
		}
		roleIDs := make(map[string]bool, len(contract.RoleIDs))
		for _, roleID := range contract.RoleIDs {
			roleIDs[roleID] = true
		}

		for _, roleID := range contract.RoleIDs {
			if memberIDs[roleID] {
				continue
			}
			out = append(out, Finding{
				Rule:     "team_role_member_drift",
				Severity: SeverityError,
				Team:     teamID, Member: roleID,
				Detail: fmt.Sprintf("roles.json on team %q declares role %q but team.json::operatingContract.members has no such member; remove the role or add the member",
					teamID, roleID),
			})
		}

		missing := make([]string, 0, len(memberIDs))
		for memberID := range memberIDs {
			if !roleIDs[memberID] {
				missing = append(missing, memberID)
			}
		}
		sort.Strings(missing)
		for _, memberID := range missing {
			out = append(out, Finding{
				Rule:     "team_role_member_drift",
				Severity: SeverityError,
				Team:     teamID, Member: memberID,
				Detail: fmt.Sprintf("member %q on team %q has no entry in roles.json; add the role or drop the member from the contract",
					memberID, teamID),
			})
		}

		out = append(out, checkRelationBindingDrift(opts.StoreDir, teamID, memberIDs)...)
	}
	return out
}

// checkRelationBindingDrift compares store/relations/team-member/ against the
// team contract's member set, in both directions.
//
// Membership is written on three surfaces — roles.json, the team contract, and
// the relation store — and the two loops above compare only the first two. The
// relation store was the unchecked one, which is how the 2026-07-28
// marketing-crew consolidation left four relation records binding agents that
// had been retired: roles.json and the contract agreed, so nothing fired, and
// the records survived twelve days. They were not inert. The prompt-matrix
// endpoint enumerates members from the relation store, so a stale record puts
// a member that cannot run into a team's rendered roster.
//
// Silent when StoreDir is unset or the relations directory is absent: unit
// tests that pass synthetic contracts with no backing tree get no findings
// rather than spurious drift.
func checkRelationBindingDrift(storeDir, teamID string, memberIDs map[string]bool) []Finding {
	storeDir = strings.TrimSpace(storeDir)
	if storeDir == "" {
		return nil
	}
	dir := filepath.Join(storeDir, "relations", "team-member")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	prefix := teamID + "__"
	bound := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".json") {
			continue
		}
		agentID := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".json")
		if agentID == "" {
			continue
		}
		bound[agentID] = true
	}

	var out []Finding
	orphaned := make([]string, 0, len(bound))
	for agentID := range bound {
		if !memberIDs[agentID] {
			orphaned = append(orphaned, agentID)
		}
	}
	sort.Strings(orphaned)
	for _, agentID := range orphaned {
		out = append(out, Finding{
			Rule:     "team_role_member_drift",
			Severity: SeverityError,
			Team:     teamID, Member: agentID,
			Detail: fmt.Sprintf("relations/team-member/%s%s.json binds agent %q to team %q but the team contract has no such member; delete the relation record, or add the member if the binding is intended",
				prefix, agentID, agentID, teamID),
		})
	}

	unbound := make([]string, 0, len(memberIDs))
	for memberID := range memberIDs {
		if !bound[memberID] {
			unbound = append(unbound, memberID)
		}
	}
	sort.Strings(unbound)
	for _, memberID := range unbound {
		out = append(out, Finding{
			Rule:     "team_role_member_drift",
			Severity: SeverityError,
			Team:     teamID, Member: memberID,
			Detail: fmt.Sprintf("member %q on team %q has no relations/team-member/%s%s.json record; create it, or drop the member from the contract",
				memberID, teamID, prefix, memberID),
		})
	}
	return out
}

// StampFindingKinds copies each finding's catalogued Kind onto the finding.
//
// A check knows what it found; only the catalog knows whether that answer came
// from a checked-in file or from live agent behavior. Stamping here means every
// consumer — the gating command, the reporting command, the prompt section —
// reads one field instead of re-deriving the split from a hardcoded rule list.
// An uncatalogued id is left empty rather than defaulted, so it cannot quietly
// join the gate.
func StampFindingKinds(findings []Finding) {
	catalog, err := DefaultRuleCatalog()
	if err != nil {
		return
	}
	for i := range findings {
		if entry, ok := catalog[findings[i].Rule]; ok {
			findings[i].Kind = entry.Kind
		}
	}
}
