// Pillar 2 (prose scan) validator. The static graph (Pillar 1) describes
// who SHOULD read/write what. The runtime attribution scanner (Pillar 3)
// surfaces what was actually written. This file is the second leg: it
// walks markdown files an agent reads on every heartbeat (RESPONSIBILITIES,
// HEARTBEAT, TEAM, agent identity templates, skill SKILL.md, and domain
// docs), detects topic-prefix references the agent will follow, and joins
// each reference back to the topics.json declarations the writer should
// have made.
//
// Authoritative contract: docs/agent-system/PROSE_SCAN_TARGETS.md (scan
// targets, regex set, owner-derivation rules, code-block exclusion,
// cross-reference matrix, severity guidance). The constants and matchers
// below are derived from that doc; changes there are decision-gated
// (changes here must keep parity).
//
// Scope of this file:
//
//  1. Public rule entry point: ruleProseTopicLeak — the ValidationOptions
//     gate, target discovery, scan, and join-against-declarations pass.
//
//  2. Target enumeration (discoverProseTargets) — derives the explicit
//     include-list from RepoRoot/ScanRoots per the doc's target table.
//
//  3. Pattern matchers (proseRegexes) — the five regexes from the doc's
//     pattern set. cli-knowledge-* patterns are at error severity;
//     backtick-topic-ref stays at warning permanently.
//
//  4. Code-block exclusion — line-by-line fenced-code-block tracker,
//     enabled only for files under docs/agent-system/ per the doc's
//     "target-conditional scanner setting" rule. Other targets are
//     scanned without exclusion.
//
//  5. Owner-derivation — file path → owner key per the doc's rules.
//
// Out of scope here (other code owns them):
//
//   - Writer-skill `writes_to[]` declaration: lives on skill.json
//     (the kind-conditional join in this file consults it via
//     buildProseSkillIndex; no logic in this file owns the field).
//
//   - Subsumption proof for the retired `non_portable_classifier` rule:
//     non_portable_classifier_subsumption_test.go is the permanent
//     regression record demonstrating that every realistic legacy
//     coupling pattern is caught here.
package memberflow

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// proseScanRule names this rule in Finding.Rule. Single source of truth
// so tests can assert the rule name without typo drift.
const proseScanRule = "prose_topic_leak"

// proseTargetKind enumerates the surface-types the scanner walks. The
// kind controls (a) which declaration set is consulted in the join pass
// and (b) whether code-block exclusion applies.
type proseTargetKind string

const (
	proseTargetMember proseTargetKind = "member" // teams/<t>/members/<m>/{RESPONSIBILITIES,HEARTBEAT}.md
	proseTargetTeam   proseTargetKind = "team"   // teams/<t>/shared/*.md and teams/<t>/*.md
	proseTargetAgent  proseTargetKind = "agent"  // agents/<id>/{SOUL,AGENTS,TOOLS}.md
	proseTargetSkill  proseTargetKind = "skill"  // skills/packs/<pack>/<id>/SKILL.md
	proseTargetDocs   proseTargetKind = "docs"   // docs/<domain>/**/*.md
)

// proseTarget is one walked file with the metadata needed to derive its
// owner key and select the right declaration set during the join pass.
type proseTarget struct {
	Path           string // absolute
	Kind           proseTargetKind
	OwnerKey       string // "team:<t>/<m>" | "team:<t>" | "agent:<id>" | "skill:<id>" | "docs:<domain>"
	TeamID         string // populated for member, team kinds
	MemberID       string // populated for member kind
	AgentID        string // populated for agent kind
	SkillID        string // populated for skill kind
	DocsDomain     string // populated for docs kind
	AllowCodeBlock bool   // when true, content inside fenced code blocks is skipped (true for docs/agent-system/ only)
}

// proseRegex is one scanner pattern. Severity here follows
// docs/agent-system/PROSE_SCAN_TARGETS.md § Severity guidance; cli-*
// patterns are at error severity, backtick-topic-ref at warning.
type proseRegex struct {
	Name     string
	Severity Severity
	Re       *regexp.Regexp
	IsCLI    bool // true for cli-knowledge-*; false for backtick-topic-ref
	IsWrite  bool // true when the pattern represents a write (knowledge-add / knowledge-update); false for reads
}

// proseRegexes is the locked pattern set from docs/agent-system/PROSE_SCAN_TARGETS.md
// § Pattern set. Adding a pattern requires a doc change.
//
// All five patterns capture the topic prefix in group 1; the captured
// string may include `<...>` placeholders (treated as wildcard segments
// by Overlap) and trailing `*` / `/`.
var proseRegexes = []proseRegex{
	{
		// Error severity: a `knowledge-add --topic` invocation in
		// prose that does not overlap a real declaration is concrete
		// drift the agent will execute on the next tick.
		Name:     "cli-knowledge-add-topic",
		Severity: SeverityError,
		Re:       regexp.MustCompile(`prompt-manager team knowledge-add\b[^\n]*?--topic[= ]"?([a-z][a-z0-9-]*(?:/[a-z0-9<>_*-]+)+)"?`),
		IsCLI:    true,
		IsWrite:  true,
	},
	{
		// Error severity.
		Name:     "cli-knowledge-list-topic",
		Severity: SeverityError,
		Re:       regexp.MustCompile(`prompt-manager team knowledge-list\b[^\n]*?--topic[= ]"?([a-z][a-z0-9-]*(?:/[a-z0-9<>_*-]+)+)"?`),
		IsCLI:    true,
		IsWrite:  false,
	},
	{
		// Error severity.
		Name:     "cli-knowledge-list-prefix",
		Severity: SeverityError,
		Re:       regexp.MustCompile(`prompt-manager team knowledge-list\b[^\n]*?--topic-prefix[= ]"?([a-z][a-z0-9-]*(?:/[a-z0-9<>_*-]+)*/?)"?`),
		IsCLI:    true,
		IsWrite:  false,
	},
	{
		// Error severity.
		Name:     "cli-knowledge-update-topic",
		Severity: SeverityError,
		Re:       regexp.MustCompile(`prompt-manager team knowledge-update\b[^\n]*?--topic[= ]"?([a-z][a-z0-9-]*(?:/[a-z0-9<>_*-]+)+)"?`),
		IsCLI:    true,
		IsWrite:  true,
	},
	{
		// Note: this regex is built from an interpreted string, not
		// a raw string, because a Go raw-string literal cannot
		// contain a backtick. The two literal backticks at start /
		// end of the body anchor the match to a backticked prose
		// reference; the captured group requires at least one '/'
		// to avoid matching bare identifiers like `audience-scan`.
		Name:     "backtick-topic-ref",
		Severity: SeverityWarning,
		Re:       regexp.MustCompile("`([a-z][a-z0-9-]*/[a-z0-9<>_*/-]+)`"),
		IsCLI:    false,
		IsWrite:  false,
	},
}

// proseMatch is one regex hit on one file.
type proseMatch struct {
	Target  proseTarget
	Pattern proseRegex
	Prefix  string // captured group 1, normalized (trimmed quotes, trailing slash stripped)
	Line    int
	RawLine string
}

// ruleProseTopicLeak is the Pillar 2 rule. Walks the scan targets enumerated
// in docs/agent-system/PROSE_SCAN_TARGETS.md, matches the regex set against
// every line outside excluded code blocks, and emits a `prose_topic_leak`
// finding when the captured prefix has no satisfying declaration in the
// matrix-determined declaration set.
//
// Silent when:
//   - opts.RepoRoot is empty AND opts.ScanRoots is empty (no tree to walk;
//     unit tests that pass synthetic members[] without a backing repo get
//     this);
//   - the discovered target list is empty (a fresh repo with no store /
//     docs subtrees);
//
// Per-file work is deterministic and side-effect-free: results depend only
// on (members, opts, on-disk file contents). No clock, no randomness.
func ruleProseTopicLeak(members []MemberTopics, opts ValidationOptions) []Finding {
	roots := opts.ScanRoots
	if len(roots) == 0 {
		if strings.TrimSpace(opts.RepoRoot) == "" {
			return nil
		}
		roots = []string{opts.RepoRoot}
	}

	var (
		targets        []proseTarget
		discoveryError []Finding
	)
	for _, root := range roots {
		discovered, err := discoverProseTargets(root)
		if err != nil {
			// One discovery failure should not silence the whole
			// validator; emit a warning naming the broken root
			// and continue with the other roots' targets (if any).
			discoveryError = append(discoveryError, Finding{
				Rule:     proseScanRule,
				Severity: SeverityWarning,
				Detail:   fmt.Sprintf("could not walk prose-scan root %s: %v", root, err),
			})
			continue
		}
		targets = append(targets, discovered...)
	}
	if len(targets) == 0 {
		return discoveryError
	}

	// Pre-build the join indexes once so per-file lookups stay O(1).
	idx := buildProseDeclarationIndex(members)

	// Pre-load skill kinds (writer-skill vs other) — needed for the
	// kind-conditional rule on skill SKILL.md targets. Empty when no
	// scan root contains a skills tree; that's fine because the join
	// pass falls back to the strict "no topic refs" rule for unknown
	// skills, which is the safe default per the doc.
	skillIndex := buildProseSkillIndex(roots)

	// Sort targets for stable output. The join pass walks in order.
	sort.Slice(targets, func(i, j int) bool { return targets[i].Path < targets[j].Path })

	findings := discoveryError
	for _, t := range targets {
		matches, err := scanProseFile(t)
		if err != nil {
			findings = append(findings, Finding{
				Rule:     proseScanRule,
				Severity: SeverityWarning,
				OwnerKey: t.OwnerKey,
				Detail:   fmt.Sprintf("could not read prose file %s: %v", t.Path, err),
			})
			continue
		}
		for _, m := range matches {
			if f, ok := joinProseMatch(m, idx, skillIndex); ok {
				findings = append(findings, f)
			}
		}
	}
	return findings
}

// -----------------------------------------------------------------------------
// Target discovery
// -----------------------------------------------------------------------------

// discoverProseTargets walks `root` and returns every file matching the
// explicit-include rules in docs/agent-system/PROSE_SCAN_TARGETS.md
// § Scan targets. Returns an empty list when neither the
// scenarios/prompt-manager/store/ nor docs/ subtree exists under root —
// callers (tests with non-repo roots) get an empty result, not an error.
func discoverProseTargets(root string) ([]proseTarget, error) {
	root = filepath.Clean(root)
	var out []proseTarget

	storeDir := filepath.Join(root, "scenarios", "prompt-manager", "store")
	if st, err := os.Stat(storeDir); err == nil && st.IsDir() {
		teamTargets, err := discoverTeamAndMemberTargets(storeDir)
		if err != nil {
			return nil, fmt.Errorf("prose-scan: walk teams under %s: %w", storeDir, err)
		}
		out = append(out, teamTargets...)

		agentTargets, err := discoverAgentTargets(storeDir)
		if err != nil {
			return nil, fmt.Errorf("prose-scan: walk agents under %s: %w", storeDir, err)
		}
		out = append(out, agentTargets...)

		skillTargets, err := discoverSkillTargets(storeDir)
		if err != nil {
			return nil, fmt.Errorf("prose-scan: walk skills under %s: %w", storeDir, err)
		}
		out = append(out, skillTargets...)
	}

	docsDir := filepath.Join(root, "docs")
	if st, err := os.Stat(docsDir); err == nil && st.IsDir() {
		docsTargets, err := discoverDocsTargets(docsDir)
		if err != nil {
			return nil, fmt.Errorf("prose-scan: walk docs under %s: %w", docsDir, err)
		}
		out = append(out, docsTargets...)
	}

	return out, nil
}

// discoverTeamAndMemberTargets walks <storeDir>/teams/<team>/ and emits
// targets for per-member RESPONSIBILITIES.md / HEARTBEAT.md plus per-team
// shared/*.md and team-root *.md files.
func discoverTeamAndMemberTargets(storeDir string) ([]proseTarget, error) {
	teamsDir := filepath.Join(storeDir, "teams")
	teams, err := os.ReadDir(teamsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []proseTarget
	for _, te := range teams {
		if !te.IsDir() || strings.HasPrefix(te.Name(), ".") {
			continue
		}
		teamID := te.Name()
		teamRoot := filepath.Join(teamsDir, teamID)

		// Per-member files.
		membersDir := filepath.Join(teamRoot, "members")
		if entries, err := os.ReadDir(membersDir); err == nil {
			for _, me := range entries {
				if !me.IsDir() || strings.HasPrefix(me.Name(), ".") {
					continue
				}
				memberID := me.Name()
				memberRoot := filepath.Join(membersDir, memberID)
				for _, name := range []string{"RESPONSIBILITIES.md", "HEARTBEAT.md"} {
					p := filepath.Join(memberRoot, name)
					if !fileExists(p) {
						continue
					}
					out = append(out, proseTarget{
						Path:     p,
						Kind:     proseTargetMember,
						OwnerKey: fmt.Sprintf("team:%s/%s", teamID, memberID),
						TeamID:   teamID,
						MemberID: memberID,
					})
				}
			}
		}

		// Per-team shared/*.md
		sharedDir := filepath.Join(teamRoot, "shared")
		if sharedEntries, err := os.ReadDir(sharedDir); err == nil {
			for _, se := range sharedEntries {
				if se.IsDir() {
					continue
				}
				if !strings.HasSuffix(se.Name(), ".md") {
					continue
				}
				out = append(out, proseTarget{
					Path:     filepath.Join(sharedDir, se.Name()),
					Kind:     proseTargetTeam,
					OwnerKey: "team:" + teamID,
					TeamID:   teamID,
				})
			}
		}

		// Per-team root *.md (defensive coverage; none exist today).
		if rootEntries, err := os.ReadDir(teamRoot); err == nil {
			for _, re := range rootEntries {
				if re.IsDir() {
					continue
				}
				if !strings.HasSuffix(re.Name(), ".md") {
					continue
				}
				out = append(out, proseTarget{
					Path:     filepath.Join(teamRoot, re.Name()),
					Kind:     proseTargetTeam,
					OwnerKey: "team:" + teamID,
					TeamID:   teamID,
				})
			}
		}
	}
	return out, nil
}

// discoverAgentTargets walks <storeDir>/agents/<agent-id>/ and emits a
// target for each of SOUL.md, AGENTS.md, TOOLS.md that exists.
func discoverAgentTargets(storeDir string) ([]proseTarget, error) {
	agentsDir := filepath.Join(storeDir, "agents")
	agents, err := os.ReadDir(agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []proseTarget
	for _, ae := range agents {
		if !ae.IsDir() || strings.HasPrefix(ae.Name(), ".") {
			continue
		}
		agentID := ae.Name()
		agentRoot := filepath.Join(agentsDir, agentID)
		for _, name := range []string{"SOUL.md", "AGENTS.md", "TOOLS.md"} {
			p := filepath.Join(agentRoot, name)
			if !fileExists(p) {
				continue
			}
			out = append(out, proseTarget{
				Path:     p,
				Kind:     proseTargetAgent,
				OwnerKey: "agent:" + agentID,
				AgentID:  agentID,
			})
		}
	}
	return out, nil
}

// discoverSkillTargets walks <storeDir>/skills/packs/<pack>/<skill>/ and
// emits a target for each SKILL.md.
func discoverSkillTargets(storeDir string) ([]proseTarget, error) {
	packsDir := filepath.Join(storeDir, "skills", "packs")
	packs, err := os.ReadDir(packsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []proseTarget
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
			skillID := s.Name()
			skillMD := filepath.Join(packsDir, p.Name(), skillID, "SKILL.md")
			if !fileExists(skillMD) {
				continue
			}
			out = append(out, proseTarget{
				Path:     skillMD,
				Kind:     proseTargetSkill,
				OwnerKey: "skill:" + skillID,
				SkillID:  skillID,
			})
		}
	}
	return out, nil
}

// discoverDocsTargets walks <docsDir>/<domain>/**/*.md and emits targets
// under owner key `docs:<domain>`. Drafts and the agent-system outline
// are excluded per the doc.
func discoverDocsTargets(docsDir string) ([]proseTarget, error) {
	domains, err := os.ReadDir(docsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []proseTarget
	for _, d := range domains {
		if !d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			continue
		}
		domain := d.Name()
		domainRoot := filepath.Join(docsDir, domain)
		err := filepath.WalkDir(domainRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				// Exclude docs/agent-system/drafts/.
				if domain == "agent-system" && entry.Name() == "drafts" {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(entry.Name(), ".md") {
				return nil
			}
			// Exclude docs/agent-system/_outline.md.
			if domain == "agent-system" && entry.Name() == "_outline.md" {
				return nil
			}
			// Defensive size cap (1 MiB) per the doc's exclusion table.
			if info, err := entry.Info(); err == nil && info.Size() > 1<<20 {
				return nil
			}
			out = append(out, proseTarget{
				Path:           path,
				Kind:           proseTargetDocs,
				OwnerKey:       "docs:" + domain,
				DocsDomain:     domain,
				AllowCodeBlock: domain == "agent-system",
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// fileExists is a light wrapper that treats every error (incl. permission
// denied) as "missing" — discovery should be tolerant; the scanner itself
// will fail loudly if it can read the directory listing but not the file.
func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// -----------------------------------------------------------------------------
// File scanner
// -----------------------------------------------------------------------------

// scanProseFile reads the file line-by-line, applies the regex set, and
// returns every match. When target.AllowCodeBlock is true (docs/agent-system/),
// content inside markdown fenced code blocks (delimited by lines whose
// stripped content starts with ``` ) is skipped. The fenced-block detection
// is intentionally simple: a line whose TrimSpace is exactly "```" or
// starts with "```" toggles the block; nested fences are not supported (and
// none appear in the corpus).
func scanProseFile(t proseTarget) ([]proseMatch, error) {
	f, err := os.Open(t.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []proseMatch
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	inCodeBlock := false
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if t.AllowCodeBlock && strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		// Skip HTML comments inline. Cheap heuristic: lines that
		// look entirely like a single-line comment. Multi-line
		// comments are rare in our corpus and acceptable noise.
		if strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->") {
			continue
		}

		for _, pr := range proseRegexes {
			for _, m := range pr.Re.FindAllStringSubmatch(line, -1) {
				if len(m) < 2 {
					continue
				}
				prefix := normalizeObservedPrefix(m[1])
				if prefix == "" {
					continue
				}
				out = append(out, proseMatch{
					Target:  t,
					Pattern: pr,
					Prefix:  prefix,
					Line:    lineNo,
					RawLine: line,
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// normalizeObservedPrefix trims surrounding quotes the regex may have
// allowed and strips a single trailing slash so prefixes like
// "audience-scan/" compare cleanly against declared "audience-scan/*"
// via Overlap.
func normalizeObservedPrefix(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, `"'`)
	p = strings.TrimSuffix(p, "/")
	return p
}

// -----------------------------------------------------------------------------
// Declaration index + join
// -----------------------------------------------------------------------------

// proseDeclarationIndex pre-computes the four declaration sets the join pass
// queries. Constructed once per Validate() call and reused across every
// matched line.
type proseDeclarationIndex struct {
	// memberOutputs[teamID][memberID] = list of declared output[].prefix
	memberOutputs map[string]map[string][]string

	// memberReads[teamID][memberID] = union of intake[]/required_read[]/evidence_consumed[]
	memberReads map[string]map[string][]string

	// teamPrefixes[teamID] = union of every prefix declared by any member
	// of the team across every read/write field
	teamPrefixes map[string][]string

	// agentTeams[agentID] = list of teamIDs that bind this agent (i.e.
	// have a member with the same id and a topics.json on disk)
	agentTeams map[string][]string

	// allPrefixes = global union, used for docs/<domain>/ scope
	allPrefixes []string
}

func buildProseDeclarationIndex(members []MemberTopics) proseDeclarationIndex {
	idx := proseDeclarationIndex{
		memberOutputs: make(map[string]map[string][]string),
		memberReads:   make(map[string]map[string][]string),
		teamPrefixes:  make(map[string][]string),
		agentTeams:    make(map[string][]string),
	}
	allSet := make(map[string]struct{})
	for _, m := range members {
		// agent → bound teams (member id is the agent id by
		// convention; a member's existence == an agent binding to
		// that team).
		idx.agentTeams[m.Ref.Member] = append(idx.agentTeams[m.Ref.Member], m.Ref.Team)

		if idx.memberOutputs[m.Ref.Team] == nil {
			idx.memberOutputs[m.Ref.Team] = make(map[string][]string)
		}
		if idx.memberReads[m.Ref.Team] == nil {
			idx.memberReads[m.Ref.Team] = make(map[string][]string)
		}

		var outs, reads []string
		for _, o := range m.Topics.Output {
			outs = append(outs, o.Prefix)
			idx.teamPrefixes[m.Ref.Team] = append(idx.teamPrefixes[m.Ref.Team], o.Prefix)
			allSet[o.Prefix] = struct{}{}
		}
		for _, in := range m.Topics.Intake {
			reads = append(reads, in.Prefix)
			idx.teamPrefixes[m.Ref.Team] = append(idx.teamPrefixes[m.Ref.Team], in.Prefix)
			allSet[in.Prefix] = struct{}{}
		}
		for _, r := range m.Topics.RequiredRead {
			reads = append(reads, r.Prefix)
			idx.teamPrefixes[m.Ref.Team] = append(idx.teamPrefixes[m.Ref.Team], r.Prefix)
			allSet[r.Prefix] = struct{}{}
		}
		for _, e := range m.Topics.EvidenceConsumed {
			reads = append(reads, e.Prefix)
			idx.teamPrefixes[m.Ref.Team] = append(idx.teamPrefixes[m.Ref.Team], e.Prefix)
			allSet[e.Prefix] = struct{}{}
		}
		idx.memberOutputs[m.Ref.Team][m.Ref.Member] = outs
		idx.memberReads[m.Ref.Team][m.Ref.Member] = reads
	}
	for p := range allSet {
		idx.allPrefixes = append(idx.allPrefixes, p)
	}
	sort.Strings(idx.allPrefixes)
	return idx
}

// joinProseMatch checks one match against the relevant declaration set
// per the doc's cross-reference matrix and returns (finding, true) when
// the match is drift, or (Finding{}, false) when the match is satisfied
// by some declaration.
func joinProseMatch(m proseMatch, idx proseDeclarationIndex, skills proseSkillIndex) (Finding, bool) {
	// candidate is the prose-original prefix used in finding details so
	// operators see the exact text from the source file. joinKey is the
	// placeholder-normalized form used for the actual overlap check.
	// Two normalizations apply:
	//
	//   1. `<...>` placeholder segments collapse to a trailing wildcard
	//      so prose like `friction-report/<scope>/<date>/<slug>`
	//      overlaps `friction-report/toolchain/*`. PROSE_SCAN_TARGETS.md
	//      guarantees this behavior; a parameterized CLI invocation
	//      shape is the canonical TOOLS.md / HEARTBEAT.md form.
	//
	//   2. `cli-knowledge-list-prefix` (the `--topic-prefix=foo/`
	//      pattern) is *prefix-match* in CLI semantics — it matches
	//      anything under `foo/` — so the captured prefix joins as if
	//      it ended in `/*`. The literal CLI flag implies the wildcard;
	//      Overlap is segment-based and needs the wildcard suffix on
	//      one side to consider a longer declared prefix as covered.
	//      Without this, `--topic-prefix=friction-report/` does not
	//      overlap `friction-report/toolchain/*` even though the CLI
	//      would clearly list entries written there.
	candidate := m.Prefix
	joinKey := normalizePlaceholderPrefix(candidate)
	if m.Pattern.Name == "cli-knowledge-list-prefix" && joinKey != "" && !strings.HasSuffix(joinKey, "/*") && joinKey != "*" {
		joinKey += "/*"
	}
	switch m.Target.Kind {
	case proseTargetMember:
		// Writes (knowledge-add / knowledge-update) join against the
		// owning member's own output[]. Reads (knowledge-list /
		// knowledge-list-prefix / backtick) join against intake +
		// required_read + evidence_consumed for that member, falling
		// back to the team-wide union (the doc treats backtick refs
		// as team-scoped already; for cli-list we keep the same
		// fallback because a member may legitimately list a peer's
		// queue).
		var declared []string
		if m.Pattern.IsWrite {
			declared = idx.memberOutputs[m.Target.TeamID][m.Target.MemberID]
			if anyOverlaps(declared, joinKey) {
				return Finding{}, false
			}
			// Fall back to team-wide outputs: a member document
			// may legitimately reference another member's write
			// (e.g., publisher prose telling the reader to invoke
			// `knowledge-add` against the campaign-draft prefix
			// the researcher writes). We still surface the
			// finding because the writer-side member should
			// declare it; for now this is not flagged at the
			// member level — but note we return drift only when
			// the prefix is undeclared anywhere on the team.
			if anyOverlaps(idx.teamPrefixes[m.Target.TeamID], joinKey) {
				return Finding{}, false
			}
		} else {
			declared = idx.memberReads[m.Target.TeamID][m.Target.MemberID]
			if anyOverlaps(declared, joinKey) {
				return Finding{}, false
			}
			if anyOverlaps(idx.teamPrefixes[m.Target.TeamID], joinKey) {
				return Finding{}, false
			}
		}
		return memberFinding(m, candidate), true

	case proseTargetTeam:
		if anyOverlaps(idx.teamPrefixes[m.Target.TeamID], joinKey) {
			return Finding{}, false
		}
		return Finding{
			Rule:     proseScanRule,
			Severity: m.Pattern.Severity,
			Member:   MemberRef{Team: m.Target.TeamID},
			Prefix:   candidate,
			OwnerKey: m.Target.OwnerKey,
			Detail:   fmt.Sprintf("%s: line %d references topic prefix %q via %s pattern, but no member of team %q declares an overlapping prefix in topics.json (intake/required_read/evidence_consumed/output)", m.Target.Path, m.Line, candidate, m.Pattern.Name, m.Target.TeamID),
		}, true

	case proseTargetAgent:
		// Agent identity prose: join against the union of all teams
		// that bind this agent. If the agent is bound to no team, the
		// reference is drift by default (a team-specific topic on a
		// template that has no concrete binding is the canonical
		// classifier-style coupling problem).
		teams := idx.agentTeams[m.Target.AgentID]
		for _, teamID := range teams {
			if anyOverlaps(idx.teamPrefixes[teamID], joinKey) {
				return Finding{}, false
			}
		}
		detail := fmt.Sprintf("%s: line %d references topic prefix %q via %s pattern, but no team binding agent %q declares an overlapping prefix", m.Target.Path, m.Line, candidate, m.Pattern.Name, m.Target.AgentID)
		if len(teams) == 0 {
			detail += " (agent has no team bindings — identity templates should not embed team-specific topic strings)"
		}
		return Finding{
			Rule:     proseScanRule,
			Severity: m.Pattern.Severity,
			Prefix:   candidate,
			OwnerKey: m.Target.OwnerKey,
			Detail:   detail,
		}, true

	case proseTargetSkill:
		// Kind-conditional rule per the doc:
		//
		//   - Writer skills (tagged `writer-skill`) are explicitly
		//     topic-aware. Their **write** patterns
		//     (`knowledge-add` / `knowledge-update`) must overlap
		//     `skill.json::writes_to[]` — that is the producer-side
		//     declaration the rest of the system trusts. Their
		//     **read** patterns (`knowledge-list` / -prefix /
		//     backtick) document where to look in the store; those
		//     references must resolve against the global declaration
		//     set (any team's topics.json), but are not constrained
		//     to writes_to[]. Without this read/write split, the
		//     rule punishes every legitimate read reference
		//     ("read the queue you're appending to," "consult the
		//     pool you're flipping status on"), which would force
		//     authors to drop CLI prose for legitimate reads.
		//
		//   - Classifier or generic skills are held to the strict
		//     portability rule: any topic reference (read or write)
		//     fires. They must not embed team-specific topic strings.
		entry := skills[m.Target.SkillID]
		if entry.IsWriter {
			if m.Pattern.IsWrite {
				if anyOverlaps(entry.WritesTo, joinKey) {
					return Finding{}, false
				}
				detail := fmt.Sprintf("%s: line %d references topic prefix %q via %s pattern, but writer skill %q does not declare it in skill.json::writes_to[]", m.Target.Path, m.Line, candidate, m.Pattern.Name, m.Target.SkillID)
				if len(entry.WritesTo) == 0 {
					detail += " — writes_to[] is missing or empty"
				}
				return Finding{
					Rule:     proseScanRule,
					Severity: m.Pattern.Severity,
					Prefix:   candidate,
					OwnerKey: m.Target.OwnerKey,
					Detail:   detail,
				}, true
			}
			// Read pattern on a writer skill: clean if either
			// (a) the prefix overlaps the skill's own
			// writes_to[] (the skill is allowed to read its
			// own past writes), or (b) the prefix overlaps
			// any team's declared prefix (the skill is
			// documenting the storage shape of a topic some
			// member already owns). Drift = neither.
			if anyOverlaps(entry.WritesTo, joinKey) {
				return Finding{}, false
			}
			if anyOverlaps(idx.allPrefixes, joinKey) {
				return Finding{}, false
			}
			return Finding{
				Rule:     proseScanRule,
				Severity: m.Pattern.Severity,
				Prefix:   candidate,
				OwnerKey: m.Target.OwnerKey,
				Detail:   fmt.Sprintf("%s: line %d references topic prefix %q via %s pattern (read), but no member on any team declares an overlapping prefix in topics.json", m.Target.Path, m.Line, candidate, m.Pattern.Name),
			}, true
		}
		// Classifier or generic skill: any topic reference fires.
		return Finding{
			Rule:     proseScanRule,
			Severity: m.Pattern.Severity,
			Prefix:   candidate,
			OwnerKey: m.Target.OwnerKey,
			Detail:   fmt.Sprintf("%s: line %d references topic prefix %q via %s pattern; classifier and generic skills must be portable across teams (no topic strings allowed)", m.Target.Path, m.Line, candidate, m.Pattern.Name),
		}, true

	case proseTargetDocs:
		// Domain docs: any-team-anywhere union.
		if anyOverlaps(idx.allPrefixes, joinKey) {
			return Finding{}, false
		}
		return Finding{
			Rule:     proseScanRule,
			Severity: m.Pattern.Severity,
			Prefix:   candidate,
			OwnerKey: m.Target.OwnerKey,
			Detail:   fmt.Sprintf("%s: line %d references topic prefix %q via %s pattern, but no member on any team declares an overlapping prefix in topics.json", m.Target.Path, m.Line, candidate, m.Pattern.Name),
		}, true
	}
	return Finding{}, false
}

func memberFinding(m proseMatch, candidate string) Finding {
	declarationSet := "intake/required_read/evidence_consumed"
	if m.Pattern.IsWrite {
		declarationSet = "output"
	}
	return Finding{
		Rule:     proseScanRule,
		Severity: m.Pattern.Severity,
		Member:   MemberRef{Team: m.Target.TeamID, Member: m.Target.MemberID},
		Prefix:   candidate,
		OwnerKey: m.Target.OwnerKey,
		Detail:   fmt.Sprintf("%s: line %d references topic prefix %q via %s pattern, but member %s/%s does not declare an overlapping prefix in topics.json::%s (or anywhere on team %q)", m.Target.Path, m.Line, candidate, m.Pattern.Name, m.Target.TeamID, m.Target.MemberID, declarationSet, m.Target.TeamID),
	}
}

// normalizePlaceholderPrefix collapses `<...>` placeholder segments into a
// trailing wildcard `*` so prose-captured prefixes join against
// declarations using the standard Overlap semantics.
//
// PROSE_SCAN_TARGETS.md guarantees this behavior:
//
//	"The scanner treats segments containing `<...>` placeholders or
//	trailing `*` as wildcards when joining against declarations
//	(e.g., `audience-scan/<date>/<slug>` joins against the declared
//	`audience-scan/*` output prefix)."
//
// Without this normalization, a prose reference like
// `friction-report/<scope>/<date>/<slug>` (the canonical placeholder form
// agents use in TOOLS.md and HEARTBEAT.md to document a parameterized CLI
// invocation) does not overlap a declaration like
// `friction-report/toolchain/*` because Overlap treats `<scope>` as a
// literal string segment, not a wildcard. The result was spurious
// prose_topic_leak findings on every parameterized prose reference.
//
// Strategy: segments through (but not including) the first `<...>` segment
// are preserved literally; the `<...>` segment and everything after it
// become a single trailing `/*`. Examples:
//
//   - `friction-report/<scope>/<date>/<slug>` → `friction-report/*`
//   - `audience-scan/<date>`                → `audience-scan/*`
//   - `audience-scan/2026-04-23/q2`         → unchanged (no placeholder)
//   - `<placeholder>`                        → `*`
//   - empty                                  → empty (callers handle empty)
//
// The truncation choice (rather than per-segment wildcard substitution)
// avoids producing multi-segment patterns like `foo/*/*/*` that Overlap
// does not handle. The semantic loss — that a reference like
// `audience-scan/<date>/q2-creators` matches `audience-scan/*` instead of
// requiring a `audience-scan/*/q2-creators` declaration — is intentional:
// the placeholder convention is for parameterized invocations, and the
// wildcard tail is the convention's natural mapping.
func normalizePlaceholderPrefix(prefix string) string {
	if prefix == "" {
		return prefix
	}
	segments := strings.Split(prefix, "/")
	for i, seg := range segments {
		if strings.ContainsAny(seg, "<>") {
			if i == 0 {
				return "*"
			}
			return strings.Join(segments[:i], "/") + "/*"
		}
	}
	return prefix
}

// anyOverlaps reports whether any declared prefix overlaps the observed
// prefix. Empty list returns false; empty observed string returns false.
func anyOverlaps(declared []string, observed string) bool {
	if observed == "" {
		return false
	}
	for _, d := range declared {
		if Overlap(d, observed) {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------------
// Skill index
// -----------------------------------------------------------------------------

// proseSkillEntry is the per-skill data the join pass needs.
type proseSkillEntry struct {
	IsWriter bool
	WritesTo []string // populated from skill.json::writes_to[]
}

// proseSkillIndex maps skill id -> proseSkillEntry. Empty when no scan
// root contains a skills tree; the join pass treats unknown skills as
// non-writer (strict no-topic rule), the safe default.
type proseSkillIndex map[string]proseSkillEntry

// buildProseSkillIndex walks each scan root's skills/packs/<pack>/<id>/skill.json
// and returns the index. A skill is `IsWriter=true` when its `tags` array
// contains the literal "writer-skill". `writes_to[]` is read straight off
// skill.json; when missing or empty, every CLI hit on a writer-skill
// SKILL.md fires (the writer must declare its writes).
func buildProseSkillIndex(roots []string) proseSkillIndex {
	idx := proseSkillIndex{}
	for _, root := range roots {
		packsDir := filepath.Join(root, "scenarios", "prompt-manager", "store", "skills", "packs")
		packs, err := os.ReadDir(packsDir)
		if err != nil {
			continue
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
				data, err := os.ReadFile(skillJSON)
				if err != nil {
					continue
				}
				var raw struct {
					Tags     []string `json:"tags"`
					WritesTo []string `json:"writes_to"`
				}
				if err := json.Unmarshal(data, &raw); err != nil {
					continue
				}
				entry := proseSkillEntry{
					WritesTo: raw.WritesTo,
				}
				for _, tag := range raw.Tags {
					if tag == "writer-skill" {
						entry.IsWriter = true
						break
					}
				}
				idx[s.Name()] = entry
			}
		}
	}
	return idx
}
