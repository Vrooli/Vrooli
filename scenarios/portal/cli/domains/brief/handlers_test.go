package brief

import (
	"testing"

	briefv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/brief"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/shared"
)

func TestListConsumerDefaultsToUnspecified(t *testing.T) {
	if got := listConsumer(""); got != briefv1.BriefConsumer_BRIEF_CONSUMER_UNSPECIFIED {
		t.Fatalf("empty list consumer = %v", got)
	}
	if got := listConsumer("portal-agent"); got != briefv1.BriefConsumer_BRIEF_CONSUMER_PORTAL_AGENT {
		t.Fatalf("agent list consumer = %v", got)
	}
}

func TestHookRuntimeMappingUsesNativeScopesAndHarnesses(t *testing.T) {
	for _, runtime := range []string{"codex", "grok", "antigravity", "opencode"} {
		if got := hookScope(runtime); got != "user" {
			t.Fatalf("%s scope = %q, want user", runtime, got)
		}
	}
	if got := hookScope("claude-code"); got != "global" {
		t.Fatalf("claude scope = %q, want global", got)
	}
	if got := harnessForRuntime("antigravity"); got != sharedv1.AgentHarness_AGENT_HARNESS_ANTIGRAVITY {
		t.Fatalf("antigravity harness = %v", got)
	}
}
