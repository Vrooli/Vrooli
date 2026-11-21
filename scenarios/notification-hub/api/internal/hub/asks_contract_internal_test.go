package hub

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeAskSpecReservesOtherForRecipientAuthoredAlternative(t *testing.T) {
	spec := AskSpec{
		Recipient: "alice", Question: "Choose?", SensitivityLabel: "public",
		Options:     []AskOption{{Key: "approve", Label: "Approve"}, {Key: "other", Label: "Producer's other"}},
		Recommended: "approve", Deadline: time.Now().Add(time.Hour), IdempotencyKey: "reserved-other",
	}
	_, err := normalizeAskSpec(spec)
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("producer option 'other' must be refused: %v", err)
	}
}
