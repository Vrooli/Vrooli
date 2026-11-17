package rootrebaselinedecision

import (
	"strings"
	"testing"
)

func TestDigestCanonicalV2(t *testing.T) {
	terms := Terms{Version: 2, Operation: "contract_change", RequesterRef: "r", Deadline: "2030-01-01T00:00:00Z", CandidateSHA: "c", ServingSHA: "s", WatchdogSHA: "w", CandidateContract: "cc", ServingContract: "sc", Changes: []string{"change"}}
	digest, err := Digest(terms)
	if err != nil || digest != "aaf074026b7af48fdbd507fceb9e9ee2755745beb4d85439f5f75ecc41334272" {
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
