package orchestration

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/packages/proto/contracts"
)

func TestResolveRunConfigAcceptsCompletionConditionAtSharedByteLimit(t *testing.T) {
	value := strings.Repeat("x", contracts.AgentManagerMaxGoalCompletionBytes)
	if err := validateCompletionCondition(value); err != nil {
		t.Fatalf("exactly %d UTF-8 bytes rejected: %v", len(value), err)
	}

	tooLarge := value + "x"
	if err := validateCompletionCondition(tooLarge); err == nil {
		t.Fatal("completion condition above the shared byte limit was accepted")
	}
}

func TestResolveRunConfigCountsUTF8BytesForCompletionCondition(t *testing.T) {
	value := strings.Repeat("界", contracts.AgentManagerMaxGoalCompletionBytes/3+1)
	if len(value) <= contracts.AgentManagerMaxGoalCompletionBytes {
		t.Fatal("fixture must exceed the byte limit while using fewer than that many runes")
	}
	if err := validateCompletionCondition(value); err == nil {
		t.Fatal("oversized UTF-8 completion condition was accepted")
	}
}
