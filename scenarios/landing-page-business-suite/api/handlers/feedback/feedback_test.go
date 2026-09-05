package feedback

import "testing"

func TestFeedbackInputValidationEnumerations(t *testing.T) {
	for _, value := range []string{"refund", "bug", "feature", "general"} {
		if !validType(value) {
			t.Errorf("validType(%q) = false, want true", value)
		}
	}
	if validType("unknown") {
		t.Fatal("unknown feedback type must not be accepted")
	}
}
