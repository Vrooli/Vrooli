package focus

import (
	"strings"
	"testing"

	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/focus"
)

func TestFormatFocusItemIncludesMaturityRepairCommand(t *testing.T) {
	got := formatFocusItem(&focusv1.FocusItem{
		PriorityScore: 1,
		Gap: &focusv1.Gap{
			Id:    "condition/maturity/demo",
			Title: "search maturity has blocking findings",
			MaturityFindings: []*focusv1.MaturityFinding{{
				RepairCommand: "search-hub maturity fix demo --apply",
			}},
		},
	})
	if !strings.Contains(got, "repair: search-hub maturity fix demo --apply") {
		t.Fatalf("formatted focus item = %q, want repair command", got)
	}
}
