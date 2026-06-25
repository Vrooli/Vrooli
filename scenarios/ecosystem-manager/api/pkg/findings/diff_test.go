package findings

import (
	"testing"

	"github.com/vrooli/maturity-go/dimensions"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func f(id string, dim dimensions.Dimension) Finding {
	return Finding{ID: id, Dimension: dim, Severity: architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR}
}

func TestDiffStates_ClosedIntroducedByDimension(t *testing.T) { // [REQ:EM-CTRL-002]
	before := BuildState([]Finding{
		f("a", "standards"),
		f("b", "standards"),
		f("c", "tests"),
	})
	after := BuildState([]Finding{
		f("a", "standards"), // unchanged
		f("d", "tests"),     // introduced
		f("e", "security"),  // introduced
	})
	// closed: b (standards), c (tests). introduced: d (tests), e (security).

	d := DiffStates(before, after)

	if d.ClosedByDimension["standards"] != 1 {
		t.Fatalf("expected 1 standards closed, got %d", d.ClosedByDimension["standards"])
	}
	if d.ClosedByDimension["tests"] != 1 {
		t.Fatalf("expected 1 tests closed, got %d", d.ClosedByDimension["tests"])
	}
	if d.IntroducedByDimension["tests"] != 1 {
		t.Fatalf("expected 1 tests introduced, got %d", d.IntroducedByDimension["tests"])
	}
	if d.IntroducedByDimension["security"] != 1 {
		t.Fatalf("expected 1 security introduced, got %d", d.IntroducedByDimension["security"])
	}
	if got := d.NetClosed(); got != 0 {
		t.Fatalf("expected net 0 (2 closed - 2 introduced), got %d", got)
	}
}

func TestDiffStates_NoChange(t *testing.T) {
	s := BuildState([]Finding{f("a", "standards"), f("b", "tests")})
	d := DiffStates(s, s)
	if len(d.ClosedByDimension) != 0 || len(d.IntroducedByDimension) != 0 {
		t.Fatalf("expected no churn, got closed=%v introduced=%v", d.ClosedByDimension, d.IntroducedByDimension)
	}
	if d.NetClosed() != 0 {
		t.Fatalf("expected net 0, got %d", d.NetClosed())
	}
}

func TestDiffStates_AllClosed(t *testing.T) {
	before := BuildState([]Finding{f("a", "standards"), f("b", "standards")})
	after := BuildState(nil)
	d := DiffStates(before, after)
	if d.ClosedByDimension["standards"] != 2 {
		t.Fatalf("expected 2 standards closed, got %d", d.ClosedByDimension["standards"])
	}
	if d.NetClosed() != 2 {
		t.Fatalf("expected net 2, got %d", d.NetClosed())
	}
}
