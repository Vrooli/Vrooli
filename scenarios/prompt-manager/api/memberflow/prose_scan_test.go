// P2.1 unit tests for the Pillar 2 prose scanner. Each test uses a
// throwaway temp directory styled to match the real repo layout
// (scenarios/prompt-manager/store/, docs/) so the discovery + join
// passes exercise the same paths they walk in production.
//
// Comprehensive failure-mode coverage with hand-crafted store fixtures
// is P2.3 (separate phase, golden-file pattern). These tests focus on
// the mechanics: regex set, code-block exclusion, owner derivation,
// kind-conditional skill rule, declaration-set selection per
// target-kind.
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

func TestProseRegex_BacktickRef_Matches(t *testing.T) {
	line := "Drain entries on `audience-scan/<date>/<slug>` every tick."
	got := findFirstByName(t, "backtick-topic-ref", line)
	if got != "audience-scan/<date>/<slug>" {
		t.Fatalf("got %q", got)
	}
}

func TestProseRegex_BacktickRef_RejectsBareIdentifier(t *testing.T) {
	// Bare identifier without a slash must not fire — too generic to
	// attribute, per PROSE_SCAN_TARGETS.md § What the scanner does not
	// match.
	line := "The `audience-scan` taxonomy lives under docs/marketing/."
	got := findFirstByName(t, "backtick-topic-ref", line)
	if got != "" {
		t.Fatalf("expected no match for bare id, got %q", got)
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
		t.Errorf("detail should point at P2.2: %q", got.Detail)
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
// (with OwnerKey added in P2.1) marshals/unmarshals cleanly so the
// findings.json telemetry artifact (P3.7) keeps working.
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
