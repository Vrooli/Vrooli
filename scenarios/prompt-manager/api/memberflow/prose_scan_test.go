// Unit tests for the Pillar 2 prose scanner. Each test uses a
// throwaway temp directory styled to match the real repo layout
// (scenarios/prompt-manager/store/, docs/) so the discovery + join
// passes exercise the same paths they walk in production.
//
// Comprehensive failure-mode coverage with hand-crafted store fixtures
// lives in prose_scan_golden_test.go (golden-file pattern). These tests
// focus on the mechanics: regex set, code-block exclusion, owner
// derivation, kind-conditional skill rule, declaration-set selection
// per target-kind.
package memberflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Placeholder normalization: PROSE_SCAN_TARGETS.md guarantees that prose
// references containing `<...>` segments are joined against declarations
// using a trailing-wildcard form. This isolates that contract; the join-side
// integration is exercised separately by the rule tests below.
// ---------------------------------------------------------------------------

func TestNormalizePlaceholderPrefix_TruncatesAtFirstPlaceholderSegment(t *testing.T) {
	cases := map[string]string{
		"":                                      "",
		"audience-scan/2026-04-23/q2":           "audience-scan/2026-04-23/q2",
		"audience-scan/<date>":                  "audience-scan/*",
		"audience-scan/<date>/<slug>":           "audience-scan/*",
		"friction-report/<scope>/<date>/<slug>": "friction-report/*",
		"<placeholder>":                         "*",
		"single-segment":                        "single-segment",
		"<scope>/concrete":                      "*",
		"prefix/<bracketed-mid>/tail":           "prefix/*",
		"prefix/contains>only-close":            "prefix/*", // ContainsAny <,> matches > too
		"prefix/contains<only-open":             "prefix/*",
	}
	for in, want := range cases {
		got := normalizePlaceholderPrefix(in)
		if got != want {
			t.Errorf("normalizePlaceholderPrefix(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestNormalizePlaceholderPrefix_EnablesOverlapWithDeclaration(t *testing.T) {
	// Real-store regression: friction-curator/TOOLS.md uses
	// `friction-report/<scope>/<date>/<slug>` to document a parameterized
	// CLI invocation. The friction-curator member declares concrete
	// scopes (e.g., `friction-report/toolchain/*`). Without
	// normalization the join misses; with normalization the prose
	// reference's `<scope>` segment collapses to a wildcard tail and
	// Overlap accepts it.
	prose := "friction-report/<scope>/<date>/<slug>"
	declaration := "friction-report/toolchain/*"

	if Overlap(declaration, prose) {
		t.Fatal("setup invariant broken: raw Overlap should NOT match before normalization (otherwise this regression case is gone and the test no longer guards anything)")
	}
	normalized := normalizePlaceholderPrefix(prose)
	if !Overlap(declaration, normalized) {
		t.Errorf("normalized prose %q should overlap declaration %q (got false)", normalized, declaration)
	}
}

// ---------------------------------------------------------------------------
// Regex set: each pattern matches only its intended invocation shape.
// ---------------------------------------------------------------------------

func TestProseRegex_KnowledgeAddTopic_Matches(t *testing.T) {
	line := `prompt-manager team knowledge-add marketing-crew --topic="audience-scan/2026-05-04/q2-creators" --content="hi"`
	matches := findFirstByName(t, "cli-knowledge-add-topic", line)
	if matches != "audience-scan/2026-05-04/q2-creators" {
		t.Fatalf("expected matching prefix, got %q", matches)
	}
}

func TestProseRegex_KnowledgeListTopic_Matches(t *testing.T) {
	line := `prompt-manager team knowledge-list marketing-crew --topic=campaign-draft/q2`
	got := findFirstByName(t, "cli-knowledge-list-topic", line)
	if got != "campaign-draft/q2" {
		t.Fatalf("got %q", got)
	}
}

func TestProseRegex_KnowledgeListPrefix_Matches(t *testing.T) {
	line := `prompt-manager team knowledge-list marketing-crew --topic-prefix=audience-scan/`
	got := findFirstByName(t, "cli-knowledge-list-prefix", line)
	if got != "audience-scan/" {
		t.Fatalf("got %q", got)
	}
}

func TestProseRegex_KnowledgeUpdateTopic_Matches(t *testing.T) {
	line := `prompt-manager team knowledge-update marketing-crew knw-abc --topic="audience-scan/keep"`
	got := findFirstByName(t, "cli-knowledge-update-topic", line)
	if got != "audience-scan/keep" {
		t.Fatalf("got %q", got)
	}
}

func TestProseScanner_InferredBacktickRef_Matches(t *testing.T) {
	line := "Drain entries on `audience-scan/<date>/<slug>` every tick."
	matches := scanProseLineInferredBacktickTopicRefs(proseTarget{}, line, 1)
	if len(matches) != 1 {
		t.Fatalf("expected one inferred topic ref, got %+v", matches)
	}
	if matches[0].Pattern.Name != "inferred-backtick-topic-ref" {
		t.Fatalf("pattern = %q", matches[0].Pattern.Name)
	}
	if matches[0].Prefix != "audience-scan/<date>/<slug>" {
		t.Fatalf("prefix = %q", matches[0].Prefix)
	}
}

func TestProseScanner_InferredBacktickRef_RejectsBareIdentifier(t *testing.T) {
	// Bare identifier without a slash must not fire — too generic to
	// attribute, per PROSE_SCAN_TARGETS.md § What the scanner does not
	// match.
	line := "The `audience-scan` taxonomy lives under docs/marketing/."
	matches := scanProseLineInferredBacktickTopicRefs(proseTarget{}, line, 1)
	if len(matches) != 0 {
		t.Fatalf("expected no match for bare id, got %+v", matches)
	}
}

func TestProseScanner_InferredBacktickRef_RequiresDeclaredMembership(t *testing.T) {
	idx := buildProseDeclarationIndex([]MemberTopics{{
		Topics: Topics{Output: []OutputEntry{{Prefix: "audience-scan/*"}}},
	}})
	if !idx.recognizesInferredPrefix("audience-scan/<date>/<slug>") {
		t.Fatal("declared topic prefix must be recognized")
	}
	if idx.recognizesInferredPrefix("internal/testutil") {
		t.Fatal("directory path must not be recognized as a topic prefix")
	}
	if idx.recognizesInferredPrefix("*") {
		t.Fatal("literal wildcard must not be recognized as a topic prefix")
	}
}

func TestProseScanner_DropsBareWildcardReference(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs", "example")
	mustWriteFile(t, filepath.Join(docsDir, "OPERATING_MODEL.md"), "Use `topic:*` only as a wildcard example.\n")
	findings := ruleProseTopicLeak(nil, ValidationOptions{RepoRoot: root})
	if len(findings) != 0 {
		t.Fatalf("bare wildcard produced findings: %+v", findings)
	}
}

func TestProseScanner_MarkedTopicRef_Matches(t *testing.T) {
	line := "Drain entries on `topic:audience-scan/<date>/<slug>` every tick."
	matches := scanProseLineMarkedTopicRefs(proseTarget{}, line, 1)
	if len(matches) != 1 {
		t.Fatalf("expected one marked topic ref, got %+v", matches)
	}
	if matches[0].Pattern.Name != "marked-topic-ref" {
		t.Fatalf("pattern = %q", matches[0].Pattern.Name)
	}
	if matches[0].Prefix != "audience-scan/<date>/<slug>" {
		t.Fatalf("prefix = %q", matches[0].Prefix)
	}
}

func TestProseScanner_MarkedTopicRef_QualifiedExamplesDoNotRequireExistence(t *testing.T) {
	for _, line := range []string{
		"Use `topic[example]:audience-scan/<date>/<slug>` in examples.",
		"Old docs may mention `topic[old]:retired-inbox/foo`.",
		"Literal text can be `topic[literal]:if/else`.",
	} {
		matches := scanProseLineMarkedTopicRefs(proseTarget{}, line, 1)
		if len(matches) != 0 {
			t.Fatalf("expected qualified marked topic not to validate for line %q, got %+v", line, matches)
		}
	}
}

func TestProseScanner_MarkedNonTopicRefsDoNotMatchTopics(t *testing.T) {
	line := "See `path:docs/agent-system/TOPICS.md`, `platform:darwin/arm64`, and `literal:if/else`."
	matches := scanProseLineMarkedTopicRefs(proseTarget{}, line, 1)
	if len(matches) != 0 {
		t.Fatalf("expected non-topic marked refs to be ignored, got %+v", matches)
	}
}

func TestProseRegex_DecisionAddTopic_DoesNotMatchKnowledgeRegexes(t *testing.T) {
	// `team decision-add --topic="..."` is a decision title, not a
	// knowledge prefix; the discriminator (`team knowledge-`) ensures
	// no cli-knowledge-* regex fires.
	line := `prompt-manager team decision-add marketing-crew --topic="What is being decided"`
	for _, name := range []string{
		"cli-knowledge-add-topic",
		"cli-knowledge-list-topic",
		"cli-knowledge-list-prefix",
		"cli-knowledge-update-topic",
	} {
		got := findFirstByName(t, name, line)
		if got != "" {
			t.Errorf("regex %s unexpectedly matched decision-add line: %q", name, got)
		}
	}
}

// findFirstByName returns the first capture group from the named regex
// against a single line, or "" when no regex with that name is found
// or no match.
func findFirstByName(t *testing.T, name, line string) string {
	t.Helper()
	for _, pr := range proseRegexes {
		if pr.Name != name {
			continue
		}
		m := pr.Re.FindStringSubmatch(line)
		if len(m) >= 2 {
			return m[1]
		}
		return ""
	}
	t.Fatalf("no regex named %q in proseRegexes", name)
	return ""
}

// ---------------------------------------------------------------------------
// File scanner: code-block exclusion is target-conditional.
// ---------------------------------------------------------------------------

func TestScanProseFile_DocsAgentSystem_SkipsFencedCodeBlocks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "TOPICS.md")
	mustWriteFile(t, path, strings.Join([]string{
		"# Example doc",
		"Outside the block, an undeclared `audience-scan/<date>/<slug>` ref appears once.",
		"",
		"```bash",
		"prompt-manager team knowledge-add marketing-crew --topic=\"audience-scan/example\"",
		"```",
		"",
		"And another `bug-inbox/regression/cli-flag` ref outside.",
	}, "\n"))

	target := proseTarget{
		Path:           path,
		Kind:           proseTargetDocs,
		OwnerKey:       "docs:agent-system",
		DocsDomain:     "agent-system",
		AllowCodeBlock: true,
	}
	matches, err := scanProseFile(target)
	if err != nil {
		t.Fatal(err)
	}
	// Both backtick refs outside the fence should match; the in-fence
	// `cli-knowledge-add` line must NOT.
	want := map[string]bool{
		"audience-scan/<date>/<slug>":   true,
		"bug-inbox/regression/cli-flag": true,
	}
	got := map[string]bool{}
	for _, m := range matches {
		got[m.Prefix] = true
		if m.Pattern.Name == "cli-knowledge-add-topic" {
			t.Errorf("cli pattern matched inside fenced block (line %d): %s", m.Line, m.RawLine)
		}
	}
	for prefix := range want {
		if !got[prefix] {
			t.Errorf("expected match for %q (got %v)", prefix, got)
		}
	}
}

func TestScanProseFile_NonDocs_DoesNotSkipCodeBlocks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "RESPONSIBILITIES.md")
	mustWriteFile(t, path, strings.Join([]string{
		"# Member responsibilities",
		"```bash",
		"prompt-manager team knowledge-add marketing-crew --topic=\"audience-scan/example\"",
		"```",
	}, "\n"))

	target := proseTarget{
		Path:           path,
		Kind:           proseTargetMember,
		OwnerKey:       "team:marketing-crew/researcher",
		TeamID:         "marketing-crew",
		MemberID:       "researcher",
		AllowCodeBlock: false,
	}
	matches, err := scanProseFile(target)
	if err != nil {
		t.Fatal(err)
	}
	// Member prose has no pedagogical-example use case; a CLI invocation
	// inside ```bash is still a real instruction the agent will follow.
	if len(matches) != 1 || matches[0].Pattern.Name != "cli-knowledge-add-topic" {
		t.Fatalf("expected one cli match in non-docs target, got %+v", matches)
	}
}

// ---------------------------------------------------------------------------
// Owner derivation: discoverProseTargets emits the canonical owner keys.
// ---------------------------------------------------------------------------

func TestDiscoverProseTargets_OwnerKeys(t *testing.T) {
	root := buildSyntheticRepo(t)
	targets, err := discoverProseTargets(root)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, tgt := range targets {
		keys[tgt.OwnerKey] = true
	}
	for _, want := range []string{
		"team:marketing-crew/researcher",
		"team:marketing-crew",
		"agent:researcher",
		"skill:report-friction",
		"docs:agent-system",
	} {
		if !keys[want] {
			t.Errorf("expected owner key %q in discovered targets, got %v", want, sortedKeys(keys))
		}
	}
}

func TestDiscoverProseTargets_ExcludesDraftsAndOutline(t *testing.T) {
	root := buildSyntheticRepo(t)

	// Add an excluded file inside docs/agent-system/drafts/.
	mustWriteFile(t, filepath.Join(root, "docs", "agent-system", "drafts", "wip.md"),
		"`audience-scan/<date>/<slug>` is just a draft note.\n")
	// And the outline.
	mustWriteFile(t, filepath.Join(root, "docs", "agent-system", "_outline.md"),
		"`audience-scan/<date>/<slug>` is just an outline.\n")

	targets, err := discoverProseTargets(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, tgt := range targets {
		if strings.Contains(tgt.Path, "/drafts/") {
			t.Errorf("drafts file unexpectedly included: %s", tgt.Path)
		}
		if strings.HasSuffix(tgt.Path, "_outline.md") {
			t.Errorf("_outline.md unexpectedly included: %s", tgt.Path)
		}
	}
}

// ---------------------------------------------------------------------------
// End-to-end: ruleProseTopicLeak emits findings against a synthetic repo.
// Coverage spans every target kind + the kind-conditional skill rule.
// ---------------------------------------------------------------------------

func TestRuleProseTopicLeak_MemberDriftFires(t *testing.T) {
	root := buildSyntheticRepo(t)
	// Member RESPONSIBILITIES.md: declares output is `campaign/*` but
	// the prose CLI invocation references `campaign-draft/<slug>` —
	// classic Pillar-1-blind drift.
	mustWriteFile(t,
		filepath.Join(root, "scenarios", "prompt-manager", "store", "teams",
			"marketing-crew", "members", "researcher", "RESPONSIBILITIES.md"),
		"Write campaign drafts via `prompt-manager team knowledge-add marketing-crew --topic=\"campaign-draft/<slug>\" --content=\"draft\"`.\n",
	)

	members := []MemberTopics{
		{
			Ref: MemberRef{Team: "marketing-crew", Member: "researcher"},
			Topics: Topics{
				Output: []OutputEntry{{Prefix: "campaign/*", DestinationKind: DestinationKnowledge}},
			},
		},
	}
	findings := ruleProseTopicLeak(members, ValidationOptions{ScanRoots: []string{root}})

	if !hasFindingFor(findings, "team:marketing-crew/researcher", "campaign-draft/<slug>") {
		t.Fatalf("expected prose_topic_leak for member drift, got %s", debugFindings(findings))
	}
}

func TestRuleProseTopicLeak_DeclaredMemberPrefixIsClean(t *testing.T) {
	root := buildSyntheticRepo(t)
	mustWriteFile(t,
		filepath.Join(root, "scenarios", "prompt-manager", "store", "teams",
			"marketing-crew", "members", "researcher", "RESPONSIBILITIES.md"),
		"Write via `prompt-manager team knowledge-add marketing-crew --topic=\"audience-scan/<date>/<slug>\"`.\n",
	)

	members := []MemberTopics{
		{
			Ref: MemberRef{Team: "marketing-crew", Member: "researcher"},
			Topics: Topics{
				Output: []OutputEntry{{Prefix: "audience-scan/*", DestinationKind: DestinationKnowledge}},
			},
		},
	}
	findings := ruleProseTopicLeak(members, ValidationOptions{ScanRoots: []string{root}})

	for _, f := range findings {
		if f.OwnerKey == "team:marketing-crew/researcher" {
			t.Errorf("unexpected finding for declared prefix: %+v", f)
		}
	}
}

func TestRuleProseTopicLeak_GenericSkillFiresOnAnyTopicRef(t *testing.T) {
	root := buildSyntheticRepo(t)
	// A generic (non-writer) skill — note `tags` does not include
	// "writer-skill" — referencing any topic prefix at all.
	skillDir := filepath.Join(root, "scenarios", "prompt-manager", "store",
		"skills", "packs", "core", "audience-scan-classifier")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(skillDir, "skill.json"),
		`{"id":"audience-scan-classifier","tags":["skill","classifier"]}`)
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"),
		"Cite findings under `prompt-manager team knowledge-list marketing-crew --topic-prefix=audience-scan/`.\n")

	findings := ruleProseTopicLeak(nil, ValidationOptions{ScanRoots: []string{root}})

	if !hasFindingFor(findings, "skill:audience-scan-classifier", "audience-scan") {
		t.Fatalf("expected prose_topic_leak for generic skill topic ref, got %s", debugFindings(findings))
	}
}

func TestRuleProseTopicLeak_WriterSkillWithoutWritesToFires(t *testing.T) {
	root := buildSyntheticRepo(t)
	// The synthetic repo writes a writer-skill (report-friction) with
	// `tags: ["writer-skill"]` and no writes_to[]. Its SKILL.md
	// references friction-inbox/* — every CLI hit fires per the
	// "writer skill that has no writes_to[] yet" special case in the
	// doc.
	skillSKILL := filepath.Join(root, "scenarios", "prompt-manager", "store",
		"skills", "packs", "core", "report-friction", "SKILL.md")
	mustWriteFile(t, skillSKILL,
		"`prompt-manager team knowledge-add meta-optimization --topic=\"friction-inbox/<scope>/<slug>\"`.\n")

	findings := ruleProseTopicLeak(nil, ValidationOptions{ScanRoots: []string{root}})

	var got Finding
	for _, f := range findings {
		if f.OwnerKey == "skill:report-friction" {
			got = f
			break
		}
	}
	if got.Rule == "" {
		t.Fatalf("expected prose_topic_leak for writer skill without writes_to[], got %s", debugFindings(findings))
	}
	if !strings.Contains(got.Detail, "writes_to[] is missing or empty") {
		t.Errorf("detail should point at writes_to[]: %q", got.Detail)
	}
}

func TestRuleProseTopicLeak_WriterSkill_WritePatternRequiresWritesTo(t *testing.T) {
	// A `knowledge-add` (write pattern) on a writer-skill SKILL.md must
	// land on a prefix declared in the skill's own writes_to[]. A
	// reference to a prefix declared by SOMEONE ELSE on a write pattern
	// is still drift — the skill is claiming to write where it does not
	// have producer-side authority.
	root := buildSyntheticRepo(t)
	skillJSON := filepath.Join(root, "scenarios", "prompt-manager", "store",
		"skills", "packs", "core", "report-friction", "skill.json")
	mustWriteFile(t, skillJSON,
		`{"id":"report-friction","tags":["writer-skill"],"writes_to":["friction-inbox/*"]}`)
	// SKILL.md does a knowledge-add on a different prefix that IS
	// globally declared (audience-scan/* via the synthetic researcher
	// member in buildSyntheticRepo). That global declaration must NOT
	// satisfy the write-pattern check.
	skillSKILL := filepath.Join(root, "scenarios", "prompt-manager", "store",
		"skills", "packs", "core", "report-friction", "SKILL.md")
	mustWriteFile(t, skillSKILL,
		"`prompt-manager team knowledge-add marketing-crew --topic=\"audience-scan/2026-05-04/x\"`.\n")

	findings := ruleProseTopicLeak(nil, ValidationOptions{ScanRoots: []string{root}})
	hits := 0
	for _, f := range findings {
		if f.OwnerKey == "skill:report-friction" && strings.Contains(f.Detail, "cli-knowledge-add-topic") {
			hits++
			if !strings.Contains(f.Detail, "writes_to[]") {
				t.Errorf("write-pattern detail should reference writes_to[]; got %q", f.Detail)
			}
		}
	}
	if hits != 1 {
		t.Errorf("write-pattern on undeclared writes_to prefix should fire; got %d findings (findings=%v)", hits, findings)
	}
}

func TestRuleProseTopicLeak_WriterSkill_ReadPatternAcceptsGlobalDeclaration(t *testing.T) {
	// A `knowledge-list-prefix` (read pattern) on a writer-skill
	// SKILL.md must NOT require the prefix to be in the skill's own
	// writes_to[]. Writer skills legitimately read other topics
	// (queue depth checks, source data lookup). The reference is clean
	// when the prefix is declared by some member somewhere; only refs
	// to entirely undeclared prefixes are drift.
	root := buildSyntheticRepo(t)
	// Researcher member declares audience-scan/* output so the read
	// pattern below has a global producer to resolve against.
	mustWriteFile(t,
		filepath.Join(root, "scenarios", "prompt-manager", "store",
			"teams", "marketing-crew", "members", "researcher", "topics.json"),
		`{"output":[{"prefix":"audience-scan/*","destination_kind":"knowledge"}]}`)
	skillJSON := filepath.Join(root, "scenarios", "prompt-manager", "store",
		"skills", "packs", "core", "report-friction", "skill.json")
	mustWriteFile(t, skillJSON,
		`{"id":"report-friction","tags":["writer-skill"],"writes_to":["friction-inbox/*"]}`)
	// SKILL.md reads from a prefix declared on team marketing-crew.
	// Since it's a list-prefix (read), the global declaration suffices.
	skillSKILL := filepath.Join(root, "scenarios", "prompt-manager", "store",
		"skills", "packs", "core", "report-friction", "SKILL.md")
	mustWriteFile(t, skillSKILL,
		"`prompt-manager team knowledge-list marketing-crew --topic-prefix=audience-scan/`.\n")

	storeDir := filepath.Join(root, "scenarios", "prompt-manager", "store")
	members, err := LoadAll(storeDir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	findings := ruleProseTopicLeak(members, ValidationOptions{ScanRoots: []string{root}})
	for _, f := range findings {
		if f.OwnerKey == "skill:report-friction" && strings.Contains(f.Detail, "cli-knowledge-list-prefix") {
			t.Errorf("read pattern on globally-declared prefix should be clean; got %+v", f)
		}
	}
}

func TestRuleProseTopicLeak_WriterSkill_ReadPatternRejectsUndeclared(t *testing.T) {
	// Read pattern on a prefix that no team declares anywhere — drift.
	// Guards against the read-side check accidentally being a no-op.
	root := buildSyntheticRepo(t)
	skillJSON := filepath.Join(root, "scenarios", "prompt-manager", "store",
		"skills", "packs", "core", "report-friction", "skill.json")
	mustWriteFile(t, skillJSON,
		`{"id":"report-friction","tags":["writer-skill"],"writes_to":["friction-inbox/*"]}`)
	skillSKILL := filepath.Join(root, "scenarios", "prompt-manager", "store",
		"skills", "packs", "core", "report-friction", "SKILL.md")
	mustWriteFile(t, skillSKILL,
		"`prompt-manager team knowledge-list marketing-crew --topic-prefix=does-not-exist/`.\n")

	findings := ruleProseTopicLeak(nil, ValidationOptions{ScanRoots: []string{root}})
	hits := 0
	for _, f := range findings {
		if f.OwnerKey == "skill:report-friction" && strings.Contains(f.Detail, "cli-knowledge-list-prefix") {
			hits++
			if !strings.Contains(f.Detail, "(read)") {
				t.Errorf("read-pattern detail should annotate read; got %q", f.Detail)
			}
		}
	}
	if hits != 1 {
		t.Errorf("read pattern on undeclared prefix should fire; got %d findings (findings=%v)", hits, findings)
	}
}

func TestRuleProseTopicLeak_WriterSkillWithDeclaredWritesToIsClean(t *testing.T) {
	root := buildSyntheticRepo(t)
	// Same skill, but skill.json now declares writes_to[].
	skillJSON := filepath.Join(root, "scenarios", "prompt-manager", "store",
		"skills", "packs", "core", "report-friction", "skill.json")
	mustWriteFile(t, skillJSON,
		`{"id":"report-friction","tags":["writer-skill"],"writes_to":["friction-inbox/*"]}`)
	skillSKILL := filepath.Join(root, "scenarios", "prompt-manager", "store",
		"skills", "packs", "core", "report-friction", "SKILL.md")
	mustWriteFile(t, skillSKILL,
		"`prompt-manager team knowledge-add meta-optimization --topic=\"friction-inbox/<scope>/<slug>\"`.\n")

	findings := ruleProseTopicLeak(nil, ValidationOptions{ScanRoots: []string{root}})
	for _, f := range findings {
		if f.OwnerKey == "skill:report-friction" {
			t.Errorf("writer skill with declared writes_to should be clean, got %+v", f)
		}
	}
}

func TestRuleProseTopicLeak_AgentTemplateJoinsAcrossBindingTeams(t *testing.T) {
	root := buildSyntheticRepo(t)
	// Agent SOUL.md references `audience-scan/*`. The synthetic repo
	// binds agent `researcher` to team `marketing-crew` (via the
	// members/researcher/ directory). With the team's researcher
	// declaring audience-scan/* output, the reference is clean.
	mustWriteFile(t,
		filepath.Join(root, "scenarios", "prompt-manager", "store", "agents", "researcher", "SOUL.md"),
		"I drain `audience-scan/<date>/<slug>` for the team I am bound to.\n",
	)

	members := []MemberTopics{
		{
			Ref: MemberRef{Team: "marketing-crew", Member: "researcher"},
			Topics: Topics{
				Output: []OutputEntry{{Prefix: "audience-scan/*", DestinationKind: DestinationKnowledge}},
			},
		},
	}
	findings := ruleProseTopicLeak(members, ValidationOptions{ScanRoots: []string{root}})
	for _, f := range findings {
		if f.OwnerKey == "agent:researcher" {
			t.Errorf("agent ref for prefix declared by binding team should be clean, got %+v", f)
		}
	}
}

func TestRuleProseTopicLeak_MarkedTopicRefFiresWhenUndeclared(t *testing.T) {
	root := buildSyntheticRepo(t)
	mustWriteFile(t,
		filepath.Join(root, "scenarios", "prompt-manager", "store",
			"teams", "marketing-crew", "members", "researcher", "RESPONSIBILITIES.md"),
		"Drain `topic:campaign-draft/<slug>` when campaign drafts arrive.\n",
	)

	findings := ruleProseTopicLeak(nil, ValidationOptions{ScanRoots: []string{root}})

	var got Finding
	for _, f := range findings {
		if f.OwnerKey == "team:marketing-crew/researcher" && f.Prefix == "campaign-draft/<slug>" {
			got = f
			break
		}
	}
	if got.Rule == "" {
		t.Fatalf("expected marked topic ref finding, got %s", debugFindings(findings))
	}
	if got.Severity != SeverityWarning || !strings.Contains(got.Detail, "marked-topic-ref") {
		t.Fatalf("unexpected marked topic finding: %+v", got)
	}
}

func TestRuleProseTopicLeak_MarkedNonTopicRefsStayClean(t *testing.T) {
	root := buildSyntheticRepo(t)
	mustWriteFile(t,
		filepath.Join(root, "docs", "agent-system", "PRIMITIVES.md"),
		"See `path:docs/agent-system/TOPICS.md`, `platform:darwin/arm64`, and `literal:if/else`.\n",
	)

	findings := ruleProseTopicLeak(nil, ValidationOptions{ScanRoots: []string{root}})
	for _, f := range findings {
		if strings.Contains(f.Detail, "marked-topic-ref") || strings.Contains(f.Detail, "inferred-backtick-topic-ref") {
			t.Fatalf("non-topic marked refs should stay clean, got %+v", f)
		}
	}
}

func TestRuleProseTopicLeak_SilentWhenNoScanRootsAndNoRepoRoot(t *testing.T) {
	findings := ruleProseTopicLeak(nil, ValidationOptions{})
	if len(findings) != 0 {
		t.Fatalf("expected silent (no roots), got %d findings", len(findings))
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// buildSyntheticRepo creates a minimal repo skeleton with the directories
// the discovery walker recognizes:
//
//	<root>/scenarios/prompt-manager/store/teams/marketing-crew/
//	  members/researcher/{RESPONSIBILITIES.md, HEARTBEAT.md, topics.json}
//	  shared/TEAM.md
//	<root>/scenarios/prompt-manager/store/agents/researcher/SOUL.md
//	<root>/scenarios/prompt-manager/store/skills/packs/core/report-friction/{skill.json, SKILL.md}
//	<root>/docs/agent-system/PRIMITIVES.md (a domain doc with code blocks)
//
// All files contain a placeholder body that won't trigger any
// prose-scan finding by default; tests overwrite the specific files they
// care about.
func buildSyntheticRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	mk := func(rel, body string) {
		mustWriteFile(t, filepath.Join(root, rel), body)
	}

	// Team marketing-crew with one member.
	mk("scenarios/prompt-manager/store/teams/marketing-crew/members/researcher/RESPONSIBILITIES.md",
		"# Researcher\nDrain audience signals.\n")
	mk("scenarios/prompt-manager/store/teams/marketing-crew/members/researcher/HEARTBEAT.md",
		"# Heartbeat\nDo your tick.\n")
	mk("scenarios/prompt-manager/store/teams/marketing-crew/members/researcher/topics.json",
		`{}`)
	mk("scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md",
		"# Marketing team\n")

	// Agent identity template.
	mk("scenarios/prompt-manager/store/agents/researcher/SOUL.md",
		"# Researcher\nI study people.\n")

	// Writer skill (default starts with no writes_to[]).
	mk("scenarios/prompt-manager/store/skills/packs/core/report-friction/skill.json",
		`{"id":"report-friction","tags":["skill","writer-skill"]}`)
	mk("scenarios/prompt-manager/store/skills/packs/core/report-friction/SKILL.md",
		"# Report Friction\nWrite friction reports.\n")

	// Domain doc.
	mk("docs/agent-system/PRIMITIVES.md",
		"# Primitives\nThe topic flow lives in `topics.json`.\n")

	return root
}

func mustWriteFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasFindingFor(findings []Finding, ownerKey, prefix string) bool {
	for _, f := range findings {
		if f.OwnerKey != ownerKey {
			continue
		}
		if f.Prefix == prefix {
			return true
		}
	}
	return false
}

func debugFindings(findings []Finding) string {
	var lines []string
	for _, f := range findings {
		lines = append(lines, f.OwnerKey+" prefix="+f.Prefix+" detail="+f.Detail)
	}
	return strings.Join(lines, "\n")
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// jsonRoundTrip is a paranoia helper: ensures the public Finding shape
// (including OwnerKey) marshals/unmarshals cleanly so the findings.json
// telemetry artifact keeps working.
func TestFinding_JSONRoundTrip_OwnerKey(t *testing.T) {
	in := Finding{
		Rule:     proseScanRule,
		Severity: SeverityWarning,
		Member:   MemberRef{Team: "x", Member: "y"},
		Prefix:   "some-prefix",
		OwnerKey: "team:x/y",
		Detail:   "test",
	}
	bs, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Finding
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch: %+v != %+v", in, out)
	}
	if !strings.Contains(string(bs), `"owner_key":"team:x/y"`) {
		t.Errorf("owner_key not present in serialized form: %s", bs)
	}
}

// TestProseScanSkipsNonDeclarationBearingDocsDomains proves the docs corpus is
// the set of surfaces whose topic references are answerable — team plans of
// record plus the agent-system canon — and not everything under docs/.
//
// Regression guard: scanning all of docs/ pulled in implementation plans,
// reference material, and concept essays. Those referenced topic prefixes
// descriptively, so the rule's warning count tracked how much prose existed
// rather than how much had drifted, and its deadband had to be pinned to the
// observed total to stay green.
func TestProseScanSkipsNonDeclarationBearingDocsDomains(t *testing.T) {
	repoRoot := t.TempDir()

	// A team plan of record: has a manifest declaring the PoR contract.
	writeRepoFile(t, repoRoot, "docs/team-a/manifest.json",
		`{"contract":{"kind":"team-plan-of-record","schema":"team-plan-of-record/v1","team":"team-a"},"version":"1.0.0","sections":[]}`)
	writeRepoFile(t, repoRoot, "docs/team-a/operating/OPERATING_MODEL.md", "# Model\n")

	// The framework canon: no manifest yet, included by name because it
	// defines the topic vocabulary.
	writeRepoFile(t, repoRoot, "docs/agent-system/TOPICS.md", "# Topics\n")

	// Not declaration-bearing: no manifest at all.
	writeRepoFile(t, repoRoot, "docs/plans/some-implementation-plan.md", "# Plan\n")
	writeRepoFile(t, repoRoot, "docs/reference/some-reference.md", "# Reference\n")

	// A manifest that is not a plan-of-record contract does not qualify.
	writeRepoFile(t, repoRoot, "docs/other/manifest.json", `{"contract":{"kind":"something-else"}}`)
	writeRepoFile(t, repoRoot, "docs/other/notes.md", "# Notes\n")

	targets, err := discoverProseTargets(repoRoot)
	if err != nil {
		t.Fatalf("discover prose targets: %v", err)
	}

	scanned := map[string]bool{}
	for _, target := range targets {
		if target.Kind == proseTargetDocs {
			scanned[target.DocsDomain] = true
		}
	}

	for _, want := range []string{"team-a", "agent-system"} {
		if !scanned[want] {
			t.Errorf("docs:%s should be scanned", want)
		}
	}
	for _, notWant := range []string{"plans", "reference", "other"} {
		if scanned[notWant] {
			t.Errorf("docs:%s is not declaration-bearing and should not be scanned", notWant)
		}
	}
}
