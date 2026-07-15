package cliutil

import (
	"testing"
)

func TestInvocationHeadersForwardIdentityAndBoundedInvocationMetadata(t *testing.T) {
	t.Setenv(EnvIdentityToken, "opaque-agent-token")
	headersForRequest := InvocationHeaders("swarm-manager", "backlog create")
	first := headersForRequest()
	second := headersForRequest()

	if got := first[HeaderAgentIdentityToken]; got != "opaque-agent-token" {
		t.Fatalf("identity header = %q", got)
	}
	if got := first[HeaderInvocationScenario]; got != "swarm-manager" {
		t.Fatalf("scenario header = %q", got)
	}
	if got := first[HeaderInvocationCommand]; got != "backlog create" {
		t.Fatalf("command header = %q", got)
	}
	if first[HeaderInvocationID] == "" || first[HeaderInvocationID] != second[HeaderInvocationID] {
		t.Fatalf("invocation id must be non-empty and stable: first=%q second=%q", first[HeaderInvocationID], second[HeaderInvocationID])
	}
}

func TestInvocationHeadersOmitAbsentIdentity(t *testing.T) {
	t.Setenv(EnvIdentityToken, "")
	headers := InvocationHeaders("plan-manager", "plans create")()
	if _, ok := headers[HeaderAgentIdentityToken]; ok {
		t.Fatalf("identity header must be absent when no token is available")
	}
}
