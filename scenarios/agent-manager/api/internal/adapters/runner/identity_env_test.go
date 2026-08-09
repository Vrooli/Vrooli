package runner

import (
	"os"
	"testing"
)

func TestScrubInheritedIdentityRemovesParentToken(t *testing.T) {
	got := scrubInheritedIdentity([]string{
		"PATH=/bin",
		"VROOLI_AGENT_IDENTITY_TOKEN=parent-secret",
		"VROOLI_AGENT_IDENTITY_TOKEN_CHILD=not-the-channel",
	})
	for _, entry := range got {
		if entry == "VROOLI_AGENT_IDENTITY_TOKEN=parent-secret" {
			t.Fatal("parent identity token was inherited")
		}
	}
	if len(got) != 2 {
		t.Fatalf("scrubbed env = %#v", got)
	}
}

func TestScrubInheritedIdentityUsesProcessEnvironmentWhenUnset(t *testing.T) {
	old, had := os.LookupEnv("VROOLI_AGENT_IDENTITY_TOKEN")
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", old)
		} else {
			_ = os.Unsetenv("VROOLI_AGENT_IDENTITY_TOKEN")
		}
	})
	_ = os.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", "inherited")
	for _, entry := range scrubInheritedIdentity(nil) {
		if entry == "VROOLI_AGENT_IDENTITY_TOKEN=inherited" {
			t.Fatal("process identity token was inherited")
		}
	}
}

func TestMergeRequestedEnvReplacesInheritedParentTokenWithChild(t *testing.T) {
	got := mergeRequestedEnv(
		[]string{"PATH=/bin", "VROOLI_AGENT_IDENTITY_TOKEN=parent"},
		[]string{"VROOLI_AGENT_IDENTITY_TOKEN=child", "VROOLI_RUN_ID=child-run"},
	)
	for _, entry := range got {
		if entry == "VROOLI_AGENT_IDENTITY_TOKEN=parent" {
			t.Fatal("parent token remained in child environment")
		}
	}
	found := false
	for _, entry := range got {
		if entry == "VROOLI_AGENT_IDENTITY_TOKEN=child" {
			found = true
		}
	}
	if !found {
		t.Fatalf("child token missing from environment: %#v", got)
	}
}
