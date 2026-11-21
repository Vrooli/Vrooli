package sessionstore

import (
	"context"
	"testing"
	"time"
)

func TestLaunchDescriptorNativeControlsValidateAndRetainLegacy(t *testing.T) {
	for _, raw := range []string{
		`{"agent":"codex","launchMode":"codex_app_server"}`,
		`{"agent":"codex","launchMode":"codex_app_server","controlsVersion":1,"model":"gpt-6-luna","reasoningEffort":"high","contextLimitTokens":12000,"approvalPolicy":"never","sandbox":"workspace-write","networkAccessDisabled":true}`,
	} {
		if _, err := ParseLaunchDescriptor(raw); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{
		`{"agent":"codex","launchMode":"codex_app_server","controlsVersion":2}`,
		`{"agent":"codex","launchMode":"codex_app_server","controlsVersion":1,"sandbox":"danger-full-access","approvalPolicy":"never","unsupportedControl":true}`,
		`{"agent":"codex","launchMode":"codex_app_server","reasoningEffort":"max"}`,
		`{"agent":"codex","launchMode":"codex_app_server","contextLimitTokens":-1}`,
		`{"agent":"codex","launchMode":"codex_app_server","networkAccessDisabled":true,"sandbox":"danger-full-access"}`,
		`{"agent":"codex","launchMode":"codex_app_server","networkAccessDisabled":true,"sandbox":"workspace-write","approvalPolicy":"on-request"}`,
		`{"agent":"codex","launchMode":"codex_app_server","networkAccessDisabled":true,"sandbox":"workspace-write","approvalPolicy":"never","profile":"unsafe"}`,
		`{"agent":"codex","launchMode":"codex_app_server","controlsVersion":1,"sandbox":"workspace-write","approvalPolicy":"never"}`,
	} {
		if _, err := ParseLaunchDescriptor(raw); err == nil {
			t.Fatalf("accepted conflicting controls: %s", raw)
		}
	}
}

func TestNativeOwnerInvalidationClearsClaimPreservingRecoveryEvidence(t *testing.T) {
	for name, store := range map[string]Store{"memory": NewInMemory(), "sqlite": newSQLStore(t)} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			want := Metadata{ID: "native", Created: time.Now(), NativeOwner: "stale-generation", NativeThreadID: "original-thread", LaunchDescriptorJSON: `{"model":"gpt-6-luna"}`, ControlMode: ControlModeNativeCapable}
			if err := store.Save(ctx, want); err != nil {
				t.Fatal(err)
			}
			if err := store.UpdateAgentInfo(ctx, want.ID, AgentInfo{ControlMode: ControlModeRecoveryOnly, ClearNativeOwner: true}); err != nil {
				t.Fatal(err)
			}
			got, err := store.Get(ctx, want.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.NativeOwner != "" || got.ControlMode != ControlModeRecoveryOnly || got.NativeThreadID != want.NativeThreadID || got.LaunchDescriptorJSON != want.LaunchDescriptorJSON {
				t.Fatalf("invalidation lost evidence or retained live claim: %+v", got)
			}
		})
	}
}
