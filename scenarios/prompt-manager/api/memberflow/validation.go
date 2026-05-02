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
// (skill registry, PoR file existence). Repo root is required for
// dangling_por_sink and missing_drain_skill checks.
type ValidationOptions struct {
	// RepoRoot is the absolute path to the repository root (the parent of
	// docs/, scenarios/, etc.). Used to verify por_file destination_path
	// targets exist. Empty disables those checks.
	RepoRoot string

	// SkillIDs is the set of skill IDs known to the skill registry.
	// Empty disables missing_drain_skill check.
	SkillIDs map[string]bool
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
	var findings []Finding

	findings = append(findings, ruleConflictingDrain(members)...)
	findings = append(findings, ruleOrphanOutput(members, opts)...)
	findings = append(findings, ruleOrphanInput(members)...)
	findings = append(findings, ruleMissingDrainSkill(members, opts)...)
	findings = append(findings, ruleDanglingPORSink(members, opts)...)

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
// Consumer types:
//   - another member's intake prefix overlaps the output prefix
//   - the output destination_kind is non-knowledge (decision, por_file,
//     capability_gap, skill_proposal, backlog) — those are sinks, not orphans
func ruleOrphanOutput(members []MemberTopics, opts ValidationOptions) []Finding {
	var out []Finding

	// Build set of intake prefixes for quick lookup.
	type intakeClaim struct {
		ref    MemberRef
		prefix string
	}
	var intakes []intakeClaim
	for _, m := range members {
		for _, e := range m.Topics.Intake {
			intakes = append(intakes, intakeClaim{m.Ref, e.Prefix})
		}
	}

	for _, m := range members {
		for _, o := range m.Topics.Output {
			// Non-knowledge destinations are sinks, never orphans.
			if o.DestinationKind != DestinationKnowledge {
				continue
			}
			// Same-team self-consumption counts as a consumer.
			matched := false
			for _, in := range intakes {
				if Overlap(in.prefix, o.Prefix) {
					matched = true
					break
				}
			}
			if !matched {
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
	}
	return out
}

// ruleOrphanInput — a member's intake prefix has no producer.
//
// Producer types:
//   - another member's output prefix overlaps the intake prefix
//   - the intake's source_team plus a member of that team writing the prefix
//   - any external_producer named on this member counts as a potential producer
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

// ruleMissingDrainSkill — the drained_by_skill on an intake entry references
// a skill ID that doesn't exist in the registry. Skipped when SkillIDs is
// nil/empty.
func ruleMissingDrainSkill(members []MemberTopics, opts ValidationOptions) []Finding {
	if len(opts.SkillIDs) == 0 {
		return nil
	}
	var out []Finding
	for _, m := range members {
		for _, in := range m.Topics.Intake {
			if !opts.SkillIDs[in.DrainedBySkill] {
				out = append(out, Finding{
					Rule:     "missing_drain_skill",
					Severity: SeverityError,
					Member:   m.Ref,
					Prefix:   in.Prefix,
					Detail:   fmt.Sprintf("drained_by_skill %q is not in the skill registry", in.DrainedBySkill),
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

// LoadSkillIDs reads the local skill registry and returns the set of known
// skill IDs. The registry layout matches the prompt-manager store:
//
//	<storeDir>/skills/packs/*/<skill-id>/skill.json
//
// Returns an empty map without error when the registry is missing — callers
// that need the missing_drain_skill check should treat that as "skip the
// check," not as a failure.
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
