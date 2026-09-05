// Pins the contract that swarm-manager spawn helpers populate
// CreateRunRequest.ConversationId explicitly per Decision D7 of the
// agent-sandbox auditability contract. Spawn surfaces should never rely on
// agent-manager's mint-on-empty fallback.

package agentmanager

import (
	"testing"
)

// Two consecutive spawns should produce two distinct ConversationIds; each
// top-level spawn from swarm-manager is conceptually a fresh conversation.
func TestFreshConversationID_Unique(t *testing.T) {
	a := freshConversationID()
	b := freshConversationID()
	if a == nil || b == nil {
		t.Fatal("freshConversationID returned nil")
	}
	if *a == *b {
		t.Fatalf("expected distinct ConversationIds, got %q twice", *a)
	}
}
