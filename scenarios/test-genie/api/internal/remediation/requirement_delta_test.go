package remediation

import (
	"reflect"
	"testing"
)

func TestCompareRequirementsRequiresFreshRequirementEvidence(t *testing.T) {
	source := Plan{Requirements: []RequirementEvidence{{ID: "REQ-pass"}, {ID: "REQ-missing"}, {ID: "REQ-skip"}, {ID: "REQ-fail"}}}
	delta := CompareRequirements(source, []string{"REQ-pass", "REQ-missing", "REQ-skip", "REQ-fail"}, []RequirementEvidence{{ID: "REQ-pass", LiveStatus: "passed"}, {ID: "REQ-skip", LiveStatus: "not_applicable"}, {ID: "REQ-fail", LiveStatus: "failed"}}, true)
	if !reflect.DeepEqual(delta.Resolved, []string{"REQ-pass"}) || !reflect.DeepEqual(delta.Remaining, []string{"REQ-fail"}) || !reflect.DeepEqual(delta.Skipped, []string{"REQ-skip"}) || !reflect.DeepEqual(delta.Unverifiable, []string{"REQ-missing"}) {
		t.Fatalf("delta = %+v", delta)
	}
	if unavailable := CompareRequirements(source, []string{"REQ-pass"}, nil, false); !reflect.DeepEqual(unavailable.Unverifiable, []string{"REQ-pass"}) {
		t.Fatalf("unavailable delta = %+v", unavailable)
	}
}
