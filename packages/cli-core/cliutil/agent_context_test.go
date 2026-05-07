package cliutil

import "testing"

func TestIsAgentControlledContext(t *testing.T) {
	for _, key := range agentContextEnvKeys {
		t.Setenv(key, "")
	}
	if IsAgentControlledContext() {
		t.Fatal("expected false without agent env")
	}
	t.Setenv("VROOLI_SANDBOX_ID", "sbx-1")
	if !IsAgentControlledContext() {
		t.Fatal("expected true with sandbox env")
	}
}
