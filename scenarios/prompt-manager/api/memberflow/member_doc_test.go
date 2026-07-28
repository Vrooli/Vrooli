package memberflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeMemberDoc lays out store/teams/<team>/members/<member>/<file>.
func writeMemberDoc(t *testing.T, storeDir, team, member, file, body string) {
	t.Helper()
	dir := filepath.Join(storeDir, "teams", team, "members", member)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, file), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", file, err)
	}
}

const conformingHeartbeat = `# Heartbeat: Example

## Task Loop
1. Do the thing.

## Handoff Shape
` + "```" + `
## HANDOFF

### Things done
` + "```" + `

## Stop Conditions
- Nothing changed.
`

const conformingResponsibilities = `# Responsibilities: Example

## Primary Duties
- Do the standing thing.
`

func memberDocFindings(t *testing.T, storeDir string, refs ...MemberRef) []Finding {
	t.Helper()
	members := make([]MemberTopics, 0, len(refs))
	for _, ref := range refs {
		members = append(members, MemberTopics{Ref: ref})
	}
	return ruleMemberDocSections(members, ValidationOptions{StoreDir: storeDir})
}

func rulesOf(findings []Finding) []string {
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		out = append(out, f.Rule)
	}
	return out
}

func countRule(findings []Finding, rule string) int {
	n := 0
	for _, f := range findings {
		if f.Rule == rule {
			n++
		}
	}
	return n
}

func TestMemberDocConformingFilesProduceNoFindings(t *testing.T) {
	store := t.TempDir()
	writeMemberDoc(t, store, "t", "m", "HEARTBEAT.md", conformingHeartbeat)
	writeMemberDoc(t, store, "t", "m", "RESPONSIBILITIES.md", conformingResponsibilities)

	if got := memberDocFindings(t, store, MemberRef{Team: "t", Member: "m"}); len(got) != 0 {
		t.Fatalf("want no findings, got %v", rulesOf(got))
	}
}

// The handoff template inside every HEARTBEAT.md opens with "## HANDOFF".
// Reading that as a section would report a bogus heading on every member, so
// fence skipping is a correctness requirement rather than a nicety.
func TestMemberDocIgnoresHeadingsInsideFencedBlocks(t *testing.T) {
	store := t.TempDir()
	// The fenced block contains a retired alias and a duplicate of a real
	// section. Neither may be reported.
	body := `# Heartbeat: Example

## Task Loop
1. Step.

## Handoff Shape
` + "```" + `
## HANDOFF
## Required Loop
## Task Loop
` + "```" + `

## Stop Conditions
- Stop.
`
	writeMemberDoc(t, store, "t", "m", "HEARTBEAT.md", body)
	writeMemberDoc(t, store, "t", "m", "RESPONSIBILITIES.md", conformingResponsibilities)

	if got := memberDocFindings(t, store, MemberRef{Team: "t", Member: "m"}); len(got) != 0 {
		t.Fatalf("fenced headings leaked into validation: %v", rulesOf(got))
	}
}

func TestMemberDocReportsRetiredAlias(t *testing.T) {
	store := t.TempDir()
	body := strings.Replace(conformingHeartbeat, "## Task Loop", "## Required Loop", 1)
	writeMemberDoc(t, store, "t", "m", "HEARTBEAT.md", body)
	writeMemberDoc(t, store, "t", "m", "RESPONSIBILITIES.md", conformingResponsibilities)

	got := memberDocFindings(t, store, MemberRef{Team: "t", Member: "m"})
	if countRule(got, "member_doc_section_alias") != 1 {
		t.Fatalf("want one alias finding, got %v", rulesOf(got))
	}
	// The alias satisfies its canonical slot, so renaming must not also
	// report the canonical section as missing — one defect, one finding.
	if n := countRule(got, "member_doc_section_missing"); n != 0 {
		t.Fatalf("alias should satisfy its canonical slot, got %d missing findings", n)
	}
	if !strings.Contains(got[0].Detail, "## Task Loop") {
		t.Fatalf("alias finding should name the canonical heading, got %q", got[0].Detail)
	}
}

// A file mid-rename carries both names. That is a duplicate, and reporting
// only the rename would leave the second copy silently in place.
func TestMemberDocReportsAliasAndCanonicalAsDuplicate(t *testing.T) {
	store := t.TempDir()
	body := conformingHeartbeat + "\n## Required Loop\n1. Second copy.\n"
	writeMemberDoc(t, store, "t", "m", "HEARTBEAT.md", body)
	writeMemberDoc(t, store, "t", "m", "RESPONSIBILITIES.md", conformingResponsibilities)

	got := memberDocFindings(t, store, MemberRef{Team: "t", Member: "m"})
	if countRule(got, "member_doc_section_alias") != 1 {
		t.Fatalf("want alias finding, got %v", rulesOf(got))
	}
	if countRule(got, "member_doc_section_duplicate") != 1 {
		t.Fatalf("want duplicate finding, got %v", rulesOf(got))
	}
}

func TestMemberDocReportsMissingRequiredSection(t *testing.T) {
	store := t.TempDir()
	body := `# Heartbeat: Example

## Task Loop
1. Step.
`
	writeMemberDoc(t, store, "t", "m", "HEARTBEAT.md", body)
	writeMemberDoc(t, store, "t", "m", "RESPONSIBILITIES.md", conformingResponsibilities)

	got := memberDocFindings(t, store, MemberRef{Team: "t", Member: "m"})
	if countRule(got, "member_doc_section_missing") != 1 {
		t.Fatalf("want one missing-section finding, got %v", rulesOf(got))
	}
	for _, f := range got {
		if f.Rule == "member_doc_section_missing" {
			if f.Severity != SeverityError {
				t.Fatalf("required section absence must be an error, got %s", f.Severity)
			}
			if !strings.Contains(f.Detail, "Handoff Shape") {
				t.Fatalf("want Handoff Shape named, got %q", f.Detail)
			}
		}
	}
}

// Recommended sections stay warnings: the content is team judgment, and an
// error would either block the sweep or push a validator into authoring prose.
func TestMemberDocRecommendedSectionIsWarningNotError(t *testing.T) {
	store := t.TempDir()
	body := strings.Replace(conformingHeartbeat, "## Stop Conditions\n- Nothing changed.\n", "", 1)
	writeMemberDoc(t, store, "t", "m", "HEARTBEAT.md", body)
	writeMemberDoc(t, store, "t", "m", "RESPONSIBILITIES.md", "# Responsibilities: Example\n")

	got := memberDocFindings(t, store, MemberRef{Team: "t", Member: "m"})
	if countRule(got, "member_doc_section_recommended") != 2 {
		t.Fatalf("want two recommended findings (Stop Conditions, Primary Duties), got %v", rulesOf(got))
	}
	for _, f := range got {
		if f.Severity == SeverityError {
			t.Fatalf("recommended-only gaps must not produce errors, got %+v", f)
		}
	}
}

func TestMemberDocFileAbsenceSeverityDiffersByFile(t *testing.T) {
	store := t.TempDir()
	// Member exists in the tree (RESPONSIBILITIES present) but has no
	// HEARTBEAT.md; that member would run the generic fallback task.
	writeMemberDoc(t, store, "t", "m", "RESPONSIBILITIES.md", conformingResponsibilities)

	got := memberDocFindings(t, store, MemberRef{Team: "t", Member: "m"})
	var heartbeat *Finding
	for i := range got {
		if got[i].Rule == "member_doc_file_missing" {
			heartbeat = &got[i]
		}
	}
	if heartbeat == nil {
		t.Fatalf("want member_doc_file_missing, got %v", rulesOf(got))
	}
	if heartbeat.Severity != SeverityError {
		t.Fatalf("missing HEARTBEAT.md must be an error, got %s", heartbeat.Severity)
	}

	// The mirror case: RESPONSIBILITIES.md absent is a warning, because the
	// member still receives its full generated contract.
	store2 := t.TempDir()
	writeMemberDoc(t, store2, "t", "m", "HEARTBEAT.md", conformingHeartbeat)
	got2 := memberDocFindings(t, store2, MemberRef{Team: "t", Member: "m"})
	for _, f := range got2 {
		if f.Rule == "member_doc_file_missing" && f.Severity != SeverityWarning {
			t.Fatalf("missing RESPONSIBILITIES.md must be a warning, got %s", f.Severity)
		}
	}
}

// Member-specific sections are the escape hatch the contract promises; a rule
// that flagged unknown headings would force every member into one shape.
func TestMemberDocAllowsMemberSpecificSections(t *testing.T) {
	store := t.TempDir()
	body := conformingHeartbeat + "\n## Ledger Entry Shape\n- member-specific.\n"
	writeMemberDoc(t, store, "t", "m", "HEARTBEAT.md", body)
	writeMemberDoc(t, store, "t", "m", "RESPONSIBILITIES.md", conformingResponsibilities)

	if got := memberDocFindings(t, store, MemberRef{Team: "t", Member: "m"}); len(got) != 0 {
		t.Fatalf("member-specific section should pass, got %v", rulesOf(got))
	}
}

func TestMemberDocSilentWithoutStoreDir(t *testing.T) {
	got := ruleMemberDocSections(
		[]MemberTopics{{Ref: MemberRef{Team: "t", Member: "m"}}},
		ValidationOptions{},
	)
	if len(got) != 0 {
		t.Fatalf("want silence without StoreDir, got %v", rulesOf(got))
	}
}

// Live-roster guard. The canon tables and the store must not drift apart: a
// new member added without the required sections, or a retired heading
// reintroduced, fails here rather than in a heartbeat.
func TestMemberDocLiveRosterHasNoErrors(t *testing.T) {
	storeDir := requirePromptManagerStoreDir(t)
	members, err := LoadAll(storeDir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(members) == 0 {
		t.Fatal("expected members in the live store")
	}

	findings := ruleMemberDocSections(members, ValidationOptions{StoreDir: storeDir})
	var errs []string
	for _, f := range findings {
		if f.Severity == SeverityError {
			errs = append(errs, f.Member.String()+": "+f.Detail)
		}
	}
	if len(errs) > 0 {
		t.Fatalf("live roster has %d member-doc errors:\n%s", len(errs), strings.Join(errs, "\n"))
	}
}
