package development

import (
	"reflect"
	"strings"
	"testing"
)

// [REQ:SWM-P0-004] [REQ:SWM-P0-017]
func TestGuidanceIsReviewedRoundTrippedAndCannotGrantScope(t *testing.T) {
	r, p := fixture(t)
	before, err := r.Preview(p)
	if err != nil {
		t.Fatal(err)
	}
	p.Guidance = Guidance{Effort: "thorough", StartingState: "fragile", Validation: "targeted", RepairRelatedCode: true, AdditionalInstructions: "Keep consumer interfaces stable.\nDo not hide uncertain evidence."}
	converted, err := ProposalFromProto(ProposalProto(p))
	if err != nil || !reflect.DeepEqual(converted, p) {
		t.Fatalf("guidance lost in wire conversion: %v %#v", err, converted)
	}
	after, err := r.Preview(converted)
	if err != nil {
		t.Fatal(err)
	}
	if before.ProposalDigest == after.ProposalDigest {
		t.Fatal("guidance change reused approval digest")
	}
	for _, want := range []string{"intent of the approved product contract", "Limit baselines and full suites", "inside the explicit allow paths", "This option adds no paths or effects", "Starting-state assessment: fragile", "cannot override protected targets", "Keep consumer interfaces stable", "NOT AUTHORIZATION TO RUN"} {
		if !strings.Contains(after.GoalMessage, want) {
			t.Errorf("missing execution guidance: %s", want)
		}
	}
	if len(after.LaunchBlockers) == 0 {
		t.Fatal("guidance presented as enforced runtime authority")
	}
	changed := p
	changed.Guidance.AdditionalInstructions += " Another instruction."
	next, _ := r.Preview(changed)
	if next.ProposalDigest == after.ProposalDigest {
		t.Fatal("custom instructions did not invalidate review")
	}
}

func TestGuidanceRejectsUnknownPoliciesAndOversizedInstructions(t *testing.T) {
	for _, g := range []Guidance{{Effort: "unlimited"}, {StartingState: "production-certified"}, {Validation: "skip"}, {AdditionalInstructions: strings.Repeat("a", 8193)}, {AdditionalInstructions: "bad\x00text"}} {
		if err := g.Validate(); err == nil {
			t.Fatalf("invalid guidance accepted: %#v", g)
		}
	}
}

func TestGuidanceProfilesPreserveEvidenceObligations(t *testing.T) {
	for _, mode := range []string{"targeted", "balanced", "certification"} {
		got := renderGuidance(Guidance{Validation: mode})
		if !strings.Contains(got, "required evidence") || !strings.Contains(got, "Related repairs: not delegated") {
			t.Fatal(got)
		}
	}
}
