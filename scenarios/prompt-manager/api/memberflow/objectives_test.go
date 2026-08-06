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

func testObjectiveRegistry(t *testing.T) ObjectiveRegistry {
	t.Helper()
	reg, err := LoadObjectives(writeObjectivesDoc(t, objectivesFixture))
	if err != nil {
		t.Fatalf("LoadObjectives: %v", err)
	}
	return reg
}

func findingRules(result OperatingGraphValidationResult) []string {
	out := make([]string, 0, len(result.Findings))
	for _, f := range result.Findings {
		out = append(out, f.Rule)
	}
	return out
}

func hasObjectiveRule(result OperatingGraphValidationResult, rule string) bool {
	for _, f := range result.Findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func TestValidateObjectivesAcceptsAJoinThatMatchesTheTable(t *testing.T) {
	result := ValidateObjectives(ObjectiveValidationInput{
		Registry: testObjectiveRegistry(t),
		Declared: map[string][]TeamObjectiveDeclaration{
			"monetization":   {{ID: "T1", Role: ObjectiveRolePrimary}},
			"marketing-crew": {{ID: "T1", Role: ObjectiveRoleSupporting}, {ID: "T3", Coverage: ObjectiveCoveragePartial}},
			"director-swarm": {{ID: "I1"}},
			"scenario-qa":    {{ID: "I1"}},
		},
	})
	if result.Errors != 0 {
		t.Fatalf("errors = %d, rules = %v", result.Errors, findingRules(result))
	}
	// T2 is unserved but declared, so it raises no unserved finding — only the
	// unmeasurable one, because it names no evidence source either.
	if hasObjectiveRule(result, "objective_unserved") {
		t.Fatalf("declared hole raised objective_unserved: %v", findingRules(result))
	}
	if !hasObjectiveRule(result, "objective_unmeasurable") {
		t.Fatalf("T2 has no evidence source and should be unmeasurable: %v", findingRules(result))
	}
}

func TestValidateObjectivesChecksBothCoverageDirections(t *testing.T) {
	cases := []struct {
		name     string
		declared map[string][]TeamObjectiveDeclaration
		wantRule string
	}{
		{
			// Upward: the team claims an objective the table does not give it.
			name:     "team claims an objective the table does not",
			declared: map[string][]TeamObjectiveDeclaration{"scenario-qa": {{ID: "T1"}}},
			wantRule: "objective_link_missing_upward",
		},
		{
			// Downward: the table names a team that does not declare it back.
			name:     "table names a team that does not declare back",
			declared: map[string][]TeamObjectiveDeclaration{"monetization": {{ID: "I1"}}},
			wantRule: "objective_link_missing_downward",
		},
		{
			name:     "unknown objective id",
			declared: map[string][]TeamObjectiveDeclaration{"monetization": {{ID: "Z9"}}},
			wantRule: "objective_unknown_id",
		},
		{
			name:     "invalid role vocabulary",
			declared: map[string][]TeamObjectiveDeclaration{"monetization": {{ID: "T1", Role: "owner"}}},
			wantRule: "objective_role_invalid",
		},
		{
			name:     "duplicate declaration",
			declared: map[string][]TeamObjectiveDeclaration{"monetization": {{ID: "T1", Role: ObjectiveRolePrimary}, {ID: "T1"}}},
			wantRule: "objective_duplicate_declaration",
		},
		{
			name:     "role disagrees with the table",
			declared: map[string][]TeamObjectiveDeclaration{"monetization": {{ID: "T1", Role: ObjectiveRoleSupporting}}},
			wantRule: "objective_role_drift",
		},
		{
			name:     "team declares nothing",
			declared: map[string][]TeamObjectiveDeclaration{"monetization": {}},
			wantRule: "objective_team_unattached",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateObjectives(ObjectiveValidationInput{
				Registry: testObjectiveRegistry(t),
				Declared: tc.declared,
			})
			if !hasObjectiveRule(result, tc.wantRule) {
				t.Fatalf("want %s, got %v", tc.wantRule, findingRules(result))
			}
		})
	}
}

func TestValidateObjectivesReportsAnUndeclaredHole(t *testing.T) {
	// Same table, but T2's gap marker removed. An unserved objective is a
	// legitimate operator choice; an unserved objective nobody wrote down is
	// exactly what the coverage rule exists to catch.
	body := objectivesFixture
	const withMarker = "| `T2` | **Personal agency.** Less operator attention. | terminal | *none* (`pending-capability`) | *none* (`pending-capability`) |"
	const withoutMarker = "| `T2` | **Personal agency.** Less operator attention. | terminal | *none* | *none* |"
	replaced := replaceOnce(body, withMarker, withoutMarker)
	if replaced == body {
		t.Fatal("fixture row not found; the test would pass vacuously")
	}
	reg, err := LoadObjectives(writeObjectivesDoc(t, replaced))
	if err != nil {
		t.Fatalf("LoadObjectives: %v", err)
	}
	result := ValidateObjectives(ObjectiveValidationInput{Registry: reg})
	if !hasObjectiveRule(result, "objective_unserved") {
		t.Fatalf("want objective_unserved, got %v", findingRules(result))
	}
}

func TestValidateObjectivesFlagsProseDrift(t *testing.T) {
	model := OperatingModelDocument{
		ID:     "monetization-operating-model",
		Team:   "monetization",
		Source: OperatingModelSource{Path: "docs/monetization/operating/OPERATING_MODEL.md"},
		Sections: OperatingModelSections{
			Mission: OperatingMarkdownSection{
				Present: true,
				Body:    []string{"Some mission prose.", "**Objective served.** `I1` — capability compounding."},
			},
		},
	}
	result := ValidateObjectives(ObjectiveValidationInput{
		Registry: testObjectiveRegistry(t),
		Declared: map[string][]TeamObjectiveDeclaration{"monetization": {{ID: "T1", Role: ObjectiveRolePrimary}}},
		Models:   []OperatingModelDocument{model},
	})
	if !hasObjectiveRule(result, "objective_prose_drift") {
		t.Fatalf("want objective_prose_drift, got %v", findingRules(result))
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

func TestObjectiveRuleCatalogIsWellFormed(t *testing.T) {
	catalog, err := ObjectiveRuleCatalog()
	if err != nil {
		t.Fatalf("ObjectiveRuleCatalog: %v", err)
	}
	// Every rule the validator can emit must be describable to an operator,
	// with an actuator naming where the fix lands.
	for _, rule := range []string{
		"objective_unknown_id", "objective_role_invalid", "objective_duplicate_declaration",
		"objective_link_missing_upward", "objective_link_missing_downward", "objective_role_drift",
		"objective_prose_drift", "objective_team_unattached", "objective_unserved", "objective_unmeasurable",
	} {
		entry, ok := catalog[rule]
		if !ok {
			t.Fatalf("catalog omits %s", rule)
		}
		if entry.Group != OperatingRuleGroupObjective || entry.Actuator == "" {
			t.Fatalf("%s = %+v", rule, entry)
		}
	}
}

func replaceOnce(s, old, new string) string {
	idx := indexOf(s, old)
	if idx < 0 {
		return s
	}
	return s[:idx] + new + s[idx+len(old):]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
