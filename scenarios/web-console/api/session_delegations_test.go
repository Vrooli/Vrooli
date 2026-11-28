package main

import (
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func TestDefaultSessionEnvEmitsDelegationOnlyWhenPresent(t *testing.T) {
	const sid = "11111111-1111-1111-1111-111111111111"
	// Isolate the process-wide holder for this test.
	prev := activeSessionDelegations
	activeSessionDelegations = newSessionDelegations()
	t.Cleanup(func() { activeSessionDelegations = prev })

	// Absent: a plain / non-human session carries no delegation env var.
	if _, ok := defaultSessionEnv(sid)[cliutil.HumanOwnerDelegationEnv]; ok {
		t.Fatal("delegation env var emitted for a session with no delegation")
	}

	// Present + unexpired: the verified-human delegation is emitted.
	activeSessionDelegations.Put(sid, "delegation-token", time.Now().Add(time.Minute))
	if got := defaultSessionEnv(sid)[cliutil.HumanOwnerDelegationEnv]; got != "delegation-token" {
		t.Fatalf("delegation env var = %q, want %q", got, "delegation-token")
	}
	// A different session never sees another session's delegation.
	if _, ok := defaultSessionEnv("22222222-2222-2222-2222-222222222222")[cliutil.HumanOwnerDelegationEnv]; ok {
		t.Fatal("delegation leaked to a different session id")
	}

	// Expired: a stale delegation is never handed to a freshly launched agent.
	activeSessionDelegations.Put(sid, "stale-token", time.Now().Add(-time.Second))
	if _, ok := defaultSessionEnv(sid)[cliutil.HumanOwnerDelegationEnv]; ok {
		t.Fatal("expired delegation env var was emitted")
	}
}

func TestBearerTokenValueStripsScheme(t *testing.T) {
	if got := bearerTokenValue("Bearer abc.def.ghi"); got != "abc.def.ghi" {
		t.Fatalf("bearer strip = %q", got)
	}
	// A Cloudflare Access assertion has no scheme and passes through unchanged.
	if got := bearerTokenValue("cfaccess.jwt.assertion"); got != "cfaccess.jwt.assertion" {
		t.Fatalf("cf-access passthrough = %q", got)
	}
}
