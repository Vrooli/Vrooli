package claude

import "testing"

func TestAnswerVerifiedOnlyForCapturedVersions(t *testing.T) {
	// [REQ:P0-017i] Answering types into the agent, so only versions whose
	// prompt boxes are captured in testdata/prompts and whose answers were
	// verified live are allowed.
	if !AnswerVerified("2.1.268") {
		t.Fatal("2.1.268 is captured and verified; want it answerable")
	}
	for _, version := range []string{"", "2.1.267", "2.2.0", "unknown"} {
		if AnswerVerified(version) {
			t.Fatalf("version %q is not verified; want false", version)
		}
	}
}
