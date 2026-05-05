// Pillar 2 (prose scanner) golden-fixture coverage. Where prose_scan_test.go
// exercises the scanner's mechanics (regex matching, code-block exclusion,
// owner derivation, kind-conditional rules) with synthetic in-memory member
// declarations, this file exercises the rule end-to-end against on-disk
// fixture trees that mirror the real repo layout.
//
// Each fixture under testdata/prose_scan/<name>/ is a self-contained
// repo-shaped subtree:
//
//	testdata/prose_scan/<name>/
//	  scenarios/prompt-manager/store/teams/<team>/members/<member>/topics.json
//	  scenarios/prompt-manager/store/teams/<team>/members/<member>/RESPONSIBILITIES.md
//	  scenarios/prompt-manager/store/agents/<id>/SOUL.md          (optional)
//	  scenarios/prompt-manager/store/skills/packs/<pack>/<id>/{skill.json, SKILL.md} (optional)
//	  docs/<domain>/*.md                                          (optional)
//	  golden.json    -- expected normalized findings, sorted
//	  README.md      -- prose explanation of the failure mode (humans only)
//
// The harness loads members from the fixture's own topics.json files via
// LoadAll, runs ruleProseTopicLeak with ScanRoots pointing at the fixture
// root, normalizes the resulting Findings into a stable golden shape (path-
// relative, no absolute Detail strings), and diffs against golden.json.
//
// To regenerate goldens after a deliberate behavior change, run:
//
//	go test ./api/memberflow/ -run TestProseScan_GoldenFixtures -update
//
// The -update flag is intentionally test-local (not flag.Bool at package
// scope) so it never collides with other -update flags in the suite.
package memberflow

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// updateGoldens regenerates the golden.json files in testdata/prose_scan/
// when set. Off by default: the test asserts the live findings match the
// committed goldens.
var updateGoldens = flag.Bool("update", false, "rewrite testdata/prose_scan/<fixture>/golden.json with current rule output")

// goldenFinding is the stable on-disk shape we diff against. Deliberately
// excludes the absolute path and line number from Detail (which would make
// goldens brittle), but lifts the matched pattern name out of Detail so a
// regression that breaks one regex but not another is visible. The
// (Rule, Severity, OwnerKey, Pattern, Prefix, Team, Member) tuple is the
// minimal stable identifier for any drift the scanner can surface.
type goldenFinding struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	OwnerKey string `json:"owner_key"`
	Pattern  string `json:"pattern,omitempty"`
	Team     string `json:"team,omitempty"`
	Member   string `json:"member,omitempty"`
	Prefix   string `json:"prefix"`
}

// patternFromDetailRe extracts the matched-pattern name from the Finding's
// Detail string, which is formatted as `... via <pattern> pattern, ...` by
// joinProseMatch and friends. The regex is anchored to the literal " via "
// preamble so it never matches incidental occurrences of the word
// "pattern" elsewhere in the detail prose.
var patternFromDetailRe = regexp.MustCompile(` via ([a-z][a-z0-9-]*) pattern`)

// extractPatternName returns the matched-pattern name embedded in the
// finding's Detail, or "" when no pattern reference is present (rare:
// discovery-error and read-error findings have no source pattern).
func extractPatternName(detail string) string {
	m := patternFromDetailRe.FindStringSubmatch(detail)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// TestProseScan_GoldenFixtures walks every subdirectory of
// testdata/prose_scan/ and asserts the prose-scan rule output against the
// fixture's golden.json. A new fixture is added by creating a new directory
// under testdata/prose_scan/, populating its repo subtree, and running the
// suite once with -update to seed golden.json. Manual review of the seeded
// golden is required before committing.
func TestProseScan_GoldenFixtures(t *testing.T) {
	const fixtureRoot = "testdata/prose_scan"

	entries, err := os.ReadDir(fixtureRoot)
	if err != nil {
		t.Fatalf("read fixture root %q: %v", fixtureRoot, err)
	}

	var fixtures []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		fixtures = append(fixtures, e.Name())
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures found under testdata/prose_scan/ — at least one is required")
	}
	sort.Strings(fixtures)

	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			runProseGoldenFixture(t, filepath.Join(fixtureRoot, name))
		})
	}
}

// runProseGoldenFixture executes the rule against one fixture and either
// compares the output to golden.json or rewrites it (-update).
func runProseGoldenFixture(t *testing.T, fixtureDir string) {
	t.Helper()

	// Convert fixtureDir to absolute so ScanRoots and LoadAll see the
	// same on-disk paths the production code would.
	absRoot, err := filepath.Abs(fixtureDir)
	if err != nil {
		t.Fatalf("abs(%q): %v", fixtureDir, err)
	}

	// Members come from the fixture's own topics.json files. Fixtures
	// without a teams/ subtree are valid (e.g. an agent-only fixture);
	// LoadAll surfaces that as ENOENT, which we translate to "no
	// members".
	storeDir := filepath.Join(absRoot, "scenarios", "prompt-manager", "store")
	members, err := loadFixtureMembers(storeDir)
	if err != nil {
		t.Fatalf("load fixture members from %q: %v", storeDir, err)
	}

	findings := ruleProseTopicLeak(members, ValidationOptions{ScanRoots: []string{absRoot}})
	got := normalizeProseFindings(findings)

	goldenPath := filepath.Join(fixtureDir, "golden.json")

	if *updateGoldens {
		writeGoldenJSON(t, goldenPath, got)
		t.Logf("rewrote %s (%d findings)", goldenPath, len(got))
		return
	}

	want := readGoldenJSON(t, goldenPath)
	if !sameGoldenSlices(got, want) {
		t.Errorf("prose-scan findings drift in fixture %s\n  got  (%d):\n%s\n  want (%d):\n%s\n  to accept: go test ./api/memberflow/ -run TestProseScan_GoldenFixtures -update",
			filepath.Base(fixtureDir),
			len(got), debugGolden(got),
			len(want), debugGolden(want),
		)
	}
}

// loadFixtureMembers wraps LoadAll with ENOENT tolerance: a fixture without
// any teams/ subtree is a legitimate shape (agent-only or skill-only
// scenarios), and we want the harness to treat it as "no declared members"
// rather than fail.
func loadFixtureMembers(storeDir string) ([]MemberTopics, error) {
	teamsDir := filepath.Join(storeDir, "teams")
	if _, err := os.Stat(teamsDir); os.IsNotExist(err) {
		return nil, nil
	}
	return LoadAll(storeDir)
}

// normalizeProseFindings projects []Finding into the stable on-disk shape
// and sorts so equality is order-independent.
func normalizeProseFindings(findings []Finding) []goldenFinding {
	out := make([]goldenFinding, 0, len(findings))
	for _, f := range findings {
		out = append(out, goldenFinding{
			Rule:     f.Rule,
			Severity: string(f.Severity),
			OwnerKey: f.OwnerKey,
			Pattern:  extractPatternName(f.Detail),
			Team:     f.Member.Team,
			Member:   f.Member.Member,
			Prefix:   f.Prefix,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OwnerKey != out[j].OwnerKey {
			return out[i].OwnerKey < out[j].OwnerKey
		}
		if out[i].Prefix != out[j].Prefix {
			return out[i].Prefix < out[j].Prefix
		}
		if out[i].Pattern != out[j].Pattern {
			return out[i].Pattern < out[j].Pattern
		}
		if out[i].Severity != out[j].Severity {
			return out[i].Severity < out[j].Severity
		}
		return out[i].Rule < out[j].Rule
	})
	return out
}

func writeGoldenJSON(t *testing.T, path string, findings []goldenFinding) {
	t.Helper()
	// Use an explicit empty slice (not nil) so the encoder emits `[]`
	// instead of `null` — easier to diff and read on PR review.
	if findings == nil {
		findings = []goldenFinding{}
	}
	// Disable HTML escaping so prefix placeholders like
	// `campaign-draft/<slug>` round-trip as-is (otherwise `<` is
	// emitted as `<` and the goldens become hard to read in PR
	// review).
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(findings); err != nil {
		t.Fatalf("marshal goldens: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func readGoldenJSON(t *testing.T, path string) []goldenFinding {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %q: %v (run with -update to seed)", path, err)
	}
	var out []goldenFinding
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("parse golden %q: %v", path, err)
	}
	return out
}

func sameGoldenSlices(a, b []goldenFinding) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func debugGolden(findings []goldenFinding) string {
	if len(findings) == 0 {
		return "    (none)"
	}
	var sb strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&sb, "    - rule=%s severity=%s owner=%s pattern=%s prefix=%s",
			f.Rule, f.Severity, f.OwnerKey, f.Pattern, f.Prefix)
		if f.Team != "" || f.Member != "" {
			fmt.Fprintf(&sb, " team=%s member=%s", f.Team, f.Member)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
