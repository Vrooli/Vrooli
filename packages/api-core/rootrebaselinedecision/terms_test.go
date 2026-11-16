package rootrebaselinedecision

import (
	"strings"
	"testing"
)

func TestDigestCanonicalV1(t *testing.T) {
	terms := Terms{RequesterRef: "r", Deadline: "2030-01-01T00:00:00Z", CandidateSHA: "c", ServingSHA: "s", WatchdogSHA: "w", CandidateContract: "cc", ServingContract: "sc", Changes: []string{"change"}}
	digest, err := Digest(terms)
	if err != nil || digest != "baa04f75d61696f99077158b023d1abdf76ca7ae05705198cc550a7b5e5e3cca" {
		t.Fatalf("digest %q: %v", digest, err)
	}
	terms.WatchdogSHA = "other"
	changed, err := Digest(terms)
	if err != nil || changed == digest {
		t.Fatalf("watchdog not bound: %q %v", changed, err)
	}
}

func TestValidateChangesUsesDisplayBudgetWithoutTruncation(t *testing.T) {
	many := make([]string, 30)
	for i := range many {
		many[i] = "changed owner"
	}
	if err := ValidateChanges(many); err != nil {
		t.Fatalf("many changes refused: %v", err)
	}
	if err := ValidateChanges([]string{strings.Repeat("x", 1000)}); err != nil {
		t.Fatalf("long explanation refused: %v", err)
	}
	if err := ValidateChanges([]string{strings.Repeat("x", 3501)}); err == nil {
		t.Fatal("oversize explanation accepted")
	}
	if err := ValidateChanges([]string{"bad\nline"}); err == nil {
		t.Fatal("control character accepted")
	}
}
