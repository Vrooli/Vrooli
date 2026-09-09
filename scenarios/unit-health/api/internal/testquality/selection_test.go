package testquality

import "testing"

func TestSelectDeterministicAndIncludesControls(t *testing.T) {
	rows := []Candidate{
		{Workspace: "api", File: "a_test.go", TestID: "TestBad", Framework: "go", TestKind: "handler", StaticStatus: Violation, Severity: Error},
		{Workspace: "api", File: "b_test.go", TestID: "TestUnknown", Framework: "go", TestKind: "handler", StaticStatus: Unknown, Severity: Warning},
		{Workspace: "api", File: "c_test.go", TestID: "TestClean", Framework: "go", TestKind: "handler", StaticStatus: CheckedClean, Severity: Info},
	}
	a, err := Select(rows, 3, "seed", "sha256:source", true)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Select(rows, 3, "seed", "sha256:source", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Selected) != 3 || a.Controls != 1 {
		t.Fatalf("selected=%d controls=%d", len(a.Selected), a.Controls)
	}
	for i := range a.Selected {
		if a.Selected[i].Identity() != b.Selected[i].Identity() {
			t.Fatalf("selection not deterministic")
		}
	}
}

func TestSelectRequiresIdentityAndPreservesUnavailable(t *testing.T) {
	if _, err := Select(nil, 1, "", "source", true); err == nil {
		t.Fatal("expected identity validation")
	}
	c, err := Select(nil, 1, "seed", "source", true)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Unavailable || c.Denominator != 0 || len(c.Selected) != 0 {
		t.Fatalf("unexpected empty cohort: %+v", c)
	}
}

func TestSelectBoundsOutput(t *testing.T) {
	rows := make([]Candidate, 20)
	for i := range rows {
		rows[i] = Candidate{Workspace: "api", File: "x", TestID: string(rune('a' + i)), Framework: "go", StaticStatus: CheckedClean}
	}
	c, err := Select(rows, 4, "seed", "source", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Selected) != 4 || !c.Truncated || len(c.Excluded) != 16 {
		t.Fatalf("unexpected cap: selected=%d excluded=%d truncated=%v", len(c.Selected), len(c.Excluded), c.Truncated)
	}
}
