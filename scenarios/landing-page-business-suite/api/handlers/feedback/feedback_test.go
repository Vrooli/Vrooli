package feedback

import "testing"

func TestFeedbackInputValidationEnumerations(t *testing.T) {
	for _, value := range []string{"refund", "bug", "feature", "general"} {
		if !validType(value) {
			t.Errorf("validType(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"pending", "in_progress", "resolved", "rejected"} {
		if !validStatus(value) {
			t.Errorf("validStatus(%q) = false, want true", value)
		}
	}
	if validType("unknown") || validStatus("unknown") {
		t.Fatal("unknown feedback values must not be accepted")
	}
}
