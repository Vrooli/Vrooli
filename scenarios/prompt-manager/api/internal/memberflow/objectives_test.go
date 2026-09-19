package memberflow

import (
	"os"
	"path/filepath"
	"testing"
)

// objectivesFixture mirrors the real table's shape, including the two rows that
// make the parser interesting: an unserved objective carrying a gap marker, and
// a served objective whose contribution the table qualifies as partial.
const objectivesFixture = "# Objectives\n" +
	"\n" +
	"## Vocabulary\n" +
	"\n" +
	"| not | a | table | of | objectives |\n" +
	"|---|---|---|---|---|\n" +
	"| `T9` | decoy row outside the section | terminal | `team:decoy` | none |\n" +
	"\n" +
	"## The objectives\n" +
	"\n" +
	"| # | Objective | Class | Served by | Evidence source |\n" +
	"|---|---|---|---|---|\n" +
	"| `T1` | **Income.** Vrooli sustains itself. | terminal | `team:monetization` (primary), `team:marketing-crew` (supporting) | Command Center `ledger` |\n" +
	"| `T2` | **Personal agency.** Less operator attention. | terminal | *none* (`pending-capability`) | *none* (`pending-capability`) |\n" +
	"| `T3` | **Contribution.** Others can run this. | terminal | `team:marketing-crew` (partial — OSS surface only) | Command Center `broadcast` |\n" +
	"| `I1` | **Capability compounding.** | instrumental | `team:director-swarm`, `team:scenario-qa` | Command Center `hive` |\n" +
	"\n" +
	"## Something else\n"

func writeObjectivesDoc(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, ObjectivesDocPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return root
}

func TestLoadObjectivesParsesOnlyTheObjectivesSection(t *testing.T) {
	reg, err := LoadObjectives(writeObjectivesDoc(t, objectivesFixture))
	if err != nil {
		t.Fatalf("LoadObjectives: %v", err)
	}
	if len(reg.Objectives) != 4 {
		t.Fatalf("parsed %d objectives, want 4: %+v", len(reg.Objectives), reg.Objectives)
	}
	// The decoy row sits under a different heading. Anchoring the parse to the
	// heading rather than to "the first table" is what keeps a narrative table
	// from silently entering the objective set.
	if _, ok := reg.Get("T9"); ok {
		t.Fatal("parsed a row from outside the objectives section")
	}

	t1, _ := reg.Get("T1")
	if t1.Title != "Income" || t1.Class != ObjectiveClassTerminal {
		t.Fatalf("T1 = %+v", t1)
	}
	if len(t1.ServedBy) != 2 {
		t.Fatalf("T1 servedBy = %+v", t1.ServedBy)
	}
	if t1.ServedBy[0].TeamID != "monetization" || t1.ServedBy[0].Role != ObjectiveRolePrimary {
		t.Fatalf("T1 primary = %+v", t1.ServedBy[0])
	}
	if t1.ServedBy[1].Role != ObjectiveRoleSupporting {
		t.Fatalf("T1 supporting = %+v", t1.ServedBy[1])
	}
	if !t1.HasEvidence {
		t.Fatal("T1 should carry an evidence source")
	}
}

func TestLoadObjectivesSeparatesDeclaredHolesFromCoverage(t *testing.T) {
	reg, err := LoadObjectives(writeObjectivesDoc(t, objectivesFixture))
	if err != nil {
		t.Fatalf("LoadObjectives: %v", err)
	}
	t2, _ := reg.Get("T2")
	if !t2.Unserved() {
		t.Fatal("T2 must parse as unserved")
	}
	// The marker is what separates a declared hole from an undeclared one. An
	// unserved objective with a marker is reported every cycle; one without is
	// a validation finding.
	if t2.GapMarker != "pending-capability" {
		t.Fatalf("T2 gapMarker = %q", t2.GapMarker)
	}
	if t2.HasEvidence {
		t.Fatal("T2 declares no evidence source")
	}

	t3, _ := reg.Get("T3")
	if len(t3.ServedBy) != 1 || t3.ServedBy[0].Coverage != ObjectiveCoveragePartial {
		t.Fatalf("T3 servedBy = %+v", t3.ServedBy)
	}
	// The table qualifies T3 as partial without naming a role. An unqualified
	// role must stay empty rather than defaulting to primary: inventing one
	// would assert something the operator did not write.
	if t3.ServedBy[0].Role != "" {
		t.Fatalf("T3 role = %q, want empty", t3.ServedBy[0].Role)
	}

	i1, _ := reg.Get("I1")
	if len(i1.ServedBy) != 2 || i1.ServedBy[0].Role != "" || i1.ServedBy[1].Role != "" {
		t.Fatalf("I1 servedBy = %+v", i1.ServedBy)
	}
}

func TestLoadObjectivesTreatsMissingDocumentAsEmpty(t *testing.T) {
	reg, err := LoadObjectives(t.TempDir())
	if err != nil {
		t.Fatalf("LoadObjectives: %v", err)
	}
	if len(reg.Objectives) != 0 {
		t.Fatalf("objectives = %+v, want none", reg.Objectives)
	}
}

func TestProseObjectiveIDsDistinguishesAbsentFromEmpty(t *testing.T) {
	if _, found := ProseObjectiveIDs([]string{"no declaration here"}); found {
		t.Fatal("absent paragraph reported as found")
	}
	ids, found := ProseObjectiveIDs([]string{"**Objective served.** `T1` — income (primary) and `T3` — contribution."})
	if !found || len(ids) != 2 || ids[0] != "T1" || ids[1] != "T3" {
		t.Fatalf("ids = %v found = %v", ids, found)
	}
}

// TestObjectiveRevisionTracksMeaningNotLayout pins what the digest is allowed
// to churn on. A reflowed line or a moved row must not read as a changed
// objective, or teams learn to re-acknowledge without reading.
func TestObjectiveRevisionTracksMeaningNotLayout(t *testing.T) {
	base := Objective{
		ID: "T1", Title: "Income.", Class: "terminal",
		ServedBy:       []ObjectiveTeamRef{{TeamID: "monetization", Role: ObjectiveRolePrimary, Coverage: ObjectiveCoverageFull}},
		EvidenceSource: "Command Center `ledger`", Line: 25,
	}
	reflowed := base
	reflowed.Title = "Income."
	reflowed.EvidenceSource = "Command  Center   `ledger`"
	reflowed.Line = 91
	if objectiveRevision(base) != objectiveRevision(reflowed) {
		t.Fatal("whitespace or row position changed the revision")
	}

	restated := base
	restated.Title = "Income. Vrooli sustains itself and its operator financially."
	if objectiveRevision(base) == objectiveRevision(restated) {
		t.Fatal("a restated objective kept its revision")
	}

	reassigned := base
	reassigned.ServedBy = []ObjectiveTeamRef{{TeamID: "marketing-crew", Role: ObjectiveRolePrimary, Coverage: ObjectiveCoverageFull}}
	if objectiveRevision(base) == objectiveRevision(reassigned) {
		t.Fatal("a reassigned objective kept its revision")
	}
}
