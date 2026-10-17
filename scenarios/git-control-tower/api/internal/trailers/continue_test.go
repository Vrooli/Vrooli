package trailers

import (
	"strings"
	"testing"
)

func TestContinueKeepsWorkRelationshipsWithoutReusingOldAttribution(t *testing.T) {
	message := "Improve review P2\n\nOld narrative.\n\nVrooli-Effort: effort:example@4\nVrooli-Epoch: effort:example#E2\nVrooli-Run: 70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb\nVrooli-Proposal: gctp-0123456789ab\nVrooli-Plan: example-plan\nVrooli-Work: incident incident-1\nSigned-off-by: Previous Author <old@example.test>"
	result := Continue(message, "123456789abcdef")
	for _, expected := range []string{"Improve review P3", "Vrooli-Effort: effort:example@4", "Vrooli-Plan: example-plan", "Vrooli-Work: incident incident-1", "Vrooli-Continues: 123456789abcdef"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("missing %q in %q", expected, result)
		}
	}
	for _, forbidden := range []string{"Vrooli-Epoch", "Vrooli-Run", "Vrooli-Proposal", "Signed-off-by", "Old narrative"} {
		if strings.Contains(result, forbidden) {
			t.Fatalf("reused old attribution %q", result)
		}
	}
	if Continue("Subject", "invalid") != "" {
		t.Fatal("invalid predecessor accepted")
	}
	if result := Continue("No legacy suffix", "1234567"); !strings.Contains(result, "Vrooli-Continues: 1234567") {
		t.Fatal("structured continuation requires no legacy suffix")
	}
}
