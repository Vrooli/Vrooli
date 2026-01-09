package agents

import "testing"

func TestAgentConstants(t *testing.T) {
	// Test that agent ID constant is defined and non-empty
	if UnifiedResolverID == "" {
		t.Error("UnifiedResolverID should not be empty")
	}

	// Test expected value
	expectedID := "unified-resolver"
	if UnifiedResolverID != expectedID {
		t.Errorf("UnifiedResolverID = %q, want %q", UnifiedResolverID, expectedID)
	}
}
