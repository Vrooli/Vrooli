package operations

import "testing"

func TestParse(t *testing.T) {
	if got := Parse(""); got != ActionInvoke {
		t.Fatalf("Parse(\"\") = %q", got)
	}
	if got := Parse(" START "); got != ActionStart {
		t.Fatalf("Parse(\" START \") = %q", got)
	}
}

func TestIsStandard(t *testing.T) {
	if !IsStandard(ActionLogs) {
		t.Fatal("expected logs to be standard")
	}
	if IsStandard(Action("custom")) {
		t.Fatal("expected custom action to be non-standard")
	}
}
