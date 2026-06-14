package facts

import (
	"strings"
	"testing"
)

func TestSplitCSVValuesMergesRepeatedFlags(t *testing.T) {
	got := splitCSVValues([]string{"health, notes_attach", "health", "validation_validate_scenario"})
	want := "health,notes_attach,validation_validate_scenario"
	if strings.Join(got, ",") != want {
		t.Fatalf("splitCSVValues() = %v, want %s", got, want)
	}
}
