package jobs

import "testing"

func TestFailedExecutionEvidenceKeepsSafetyEvidence(t *testing.T) {
	got := failedExecutionEvidence(map[string]string{
		"safety_review":        "refused",
		"safety_action":        "refuse",
		"safety_reason":        "sexual_unassessable",
		"safety_evidence":      `{"reviewer_version":"test"}`,
		"safety_refused_refs":  "out/a.png",
		"safety_review_detail": "no readable face",
		"nsfw_score":           "0.9700",
		"result_refs":          "out/a.png",
		"cost_usd":             "1.00",
	})
	for _, key := range []string{"safety_review", "safety_action", "safety_reason", "safety_evidence", "safety_refused_refs", "safety_review_detail", "nsfw_score"} {
		if got[key] == "" {
			t.Errorf("failed job dropped %q: %v", key, got)
		}
	}
	if _, ok := got["result_refs"]; ok {
		t.Error("failed job retained an output reference")
	}
	if _, ok := got["cost_usd"]; ok {
		t.Error("failed job retained success-only cost evidence")
	}
}

func TestSafetyUnavailableIsANonFaultFailureKind(t *testing.T) {
	kind := ParseFailureKind(string(FailureSafetyUnavailable))
	if kind != FailureSafetyUnavailable {
		t.Fatalf("kind = %q", kind)
	}
	if kind.CountsAgainstModel() {
		t.Fatal("a technical safety failure must not count against the model")
	}
}
