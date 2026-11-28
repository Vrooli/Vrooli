package validation

import (
	"strings"
	"testing"
)

func TestQualificationSessionIDsAreUniqueAndDurableSafe(t *testing.T) {
	first, err := newQualificationSessionID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := newQualificationSessionID()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("qualification runs reused a durable stream identity")
	}
	for _, id := range []string{first, second} {
		if !strings.HasPrefix(id, "out-of-band-audio-qualification-") {
			t.Fatalf("unexpected session ID prefix: %q", id)
		}
	}
}

func TestQualificationSessionByteCapLeavesLedgerHeadroom(t *testing.T) {
	const ledgerRetentionCap = 64 * 1024 * 1024
	if qualificationSessionByteCap <= 0 || qualificationSessionByteCap >= ledgerRetentionCap {
		t.Fatalf("qualification session byte cap = %d, must stay positive and below ledger cap %d", qualificationSessionByteCap, ledgerRetentionCap)
	}
}
