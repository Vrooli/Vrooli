package optimization

import (
	"testing"
)

func TestRecordSummary(t *testing.T) {
	stage, revision := recordSummary(`{"stage":"evaluate","revision":4}`)
	if stage != "evaluate" || revision != 4 {
		t.Fatalf("recordSummary()=(%q,%d), want (evaluate,4)", stage, revision)
	}
	stage, revision = recordSummary("not-json")
	if stage != "unknown" || revision != 0 {
		t.Fatalf("malformed recordSummary()=(%q,%d), want (unknown,0)", stage, revision)
	}
}
