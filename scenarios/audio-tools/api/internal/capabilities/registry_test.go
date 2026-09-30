package capabilities_test

import (
	"context"
	"testing"
	"time"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/capabilities/mocks"

	"github.com/vrooli/api-core/schedule"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

func TestRegistry_SharedContractAndMappings(t *testing.T) {
	reg := capabilities.NewRegistryWithClock(
		[]capabilities.Def{{ID: "cap", Name: "Capability", Description: "test", DependencyKind: capabilities.DependencyResource, DependencySlug: "cap"}},
		map[string]capabilities.Checker{"cap": mocks.NewFakeChecker(capabilities.StatusAvailable, "ok")}, time.Minute, schedule.System(),
	)
	if _, err := reg.Describe(context.Background()); err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if reg.CacheTTL() != time.Minute {
		t.Fatalf("unexpected cache ttl: %s", reg.CacheTTL())
	}
	for _, id := range []string{"whisper-stt", "kokoro-tts", "speaker-verification", "ollama", "openrouter", "unknown"} {
		_ = capabilities.TierForProviderID(id)
		_, _ = capabilities.ResourceSlugForProviderID(id)
		_ = capabilities.IsLocalProvider(id)
	}
	for _, id := range []string{"openai-whisper", "deepgram", "openai-tts", "elevenlabs"} {
		if got := capabilities.TierForProviderID(id); got != commonv1.ProviderTier_PROVIDER_TIER_BYOK {
			t.Fatalf("TierForProviderID(%q) = %s, want BYOK", id, got)
		}
	}
	if got := capabilities.TierForProviderID("browser-stt"); got != commonv1.ProviderTier_PROVIDER_TIER_BROWSER {
		t.Fatalf("TierForProviderID(browser-stt) = %s, want BROWSER", got)
	}
	for _, feature := range []string{"voice-input", "voice-output", "ai-command-generation", "unknown"} {
		_, _ = capabilities.CapabilityForFeature(feature)
	}
}

func TestKnownCatalogCoversOperatorHealthCapabilities(t *testing.T) {
	seen := map[string]bool{}
	for _, def := range capabilities.Known {
		if seen[def.ID] {
			t.Fatalf("duplicate known capability %q", def.ID)
		}
		seen[def.ID] = true
	}
	for _, want := range []string{"whisper-stt", "kyutai-stt", "kokoro-tts", "openai-whisper", "openai-tts", "browser-stt", "browser-tts", "ollama", "audio-transcode"} {
		if !seen[want] {
			t.Errorf("Known catalog does not include %q", want)
		}
	}
	for _, feature := range []string{"voice-input", "voice-output", "ai-command-generation", "transcode"} {
		if _, ok := capabilities.CapabilityForFeature(feature); !ok {
			t.Errorf("feature %q is not mapped to an operator capability", feature)
		}
	}
}

func TestRegistry_Resolve(t *testing.T) {
	defs := []capabilities.Def{
		{ID: "cap-a", Name: "Cap A"},
		{ID: "cap-b", Name: "Cap B"},
		{ID: "cap-c", Name: "Cap C"},
	}

	checkerA := mocks.NewFakeChecker(capabilities.StatusAvailable, "ok")
	checkerB := mocks.NewFakeChecker(capabilities.StatusUnavailable, "down")

	checkers := map[string]capabilities.Checker{
		"cap-a": checkerA,
		"cap-b": checkerB,
	}

	tests := []struct {
		name       string
		capID      string
		wantStatus capabilities.Status
		wantMsg    string
	}{
		{"available checker", "cap-a", capabilities.StatusAvailable, "ok"},
		{"unavailable checker", "cap-b", capabilities.StatusUnavailable, "down"},
		{"no checker defaults to unknown", "cap-c", capabilities.StatusUnknown, ""},
	}

	reg := capabilities.NewRegistry(defs, checkers, time.Minute)
	states := reg.Resolve(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var found *capabilities.State
			for i := range states {
				if states[i].ID == tt.capID {
					found = &states[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("capability %q not found in results", tt.capID)
			}
			if found.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", found.Status, tt.wantStatus)
			}
			if found.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", found.Message, tt.wantMsg)
			}
		})
	}
}

func TestServiceabilityUsesAnyAvailableProviderAndReportsRequiredFailures(t *testing.T) {
	states := []capabilities.State{
		{Def: capabilities.Def{ID: "local-stt", Features: []string{"voice-input"}}, Status: capabilities.StatusUnavailable, Message: "whisper down"},
		{Def: capabilities.Def{ID: "cloud-stt", Features: []string{"voice-input"}}, Status: capabilities.StatusAvailable, Message: "BYOK ready"},
		{Def: capabilities.Def{ID: "transcode", Features: []string{"transcode"}}, Status: capabilities.StatusUnavailable, Message: "ffmpeg missing"},
		{Def: capabilities.Def{ID: "optional", Features: []string{"ai-command-generation"}}, Status: capabilities.StatusUnavailable, Message: "optional down"},
	}
	groups := capabilities.Serviceability(states)
	var stt, transcode capabilities.CapabilityServiceability
	for _, group := range groups {
		switch group.Capability.String() {
		case "CAPABILITY_STT":
			stt = group
		case "CAPABILITY_TRANSCODE":
			transcode = group
		}
	}
	if !stt.Serviceable || len(stt.UnavailableProviders) != 1 || stt.UnavailableProviders[0] != "local-stt" {
		t.Fatalf("STT serviceability = %+v", stt)
	}
	if transcode.Serviceable {
		t.Fatalf("transcode unexpectedly serviceable: %+v", transcode)
	}
	failures := capabilities.RequiredFailures(states)
	if len(failures) != 1 || failures[0] == "" {
		t.Fatalf("required failures = %#v", failures)
	}
}

func TestRegistry_Caching(t *testing.T) {
	checker := mocks.NewFakeChecker(capabilities.StatusAvailable, "ok")
	defs := []capabilities.Def{{ID: "cap-x", Name: "Cap X"}}
	checkers := map[string]capabilities.Checker{"cap-x": checker}

	reg := capabilities.NewRegistry(defs, checkers, time.Minute)

	reg.Resolve(context.Background())
	if got := checker.CallCount(); got != 1 {
		t.Fatalf("after first Resolve: calls = %d, want 1", got)
	}

	reg.Resolve(context.Background())
	if got := checker.CallCount(); got != 1 {
		t.Errorf("after second Resolve (cached): calls = %d, want 1", got)
	}
}

func TestRegistry_ResolveLiveness(t *testing.T) {
	fullChecker := mocks.NewFakeChecker(capabilities.StatusAvailable, "full check ok")
	livenessChecker := mocks.NewFakeChecker(capabilities.StatusAvailable, "liveness ok")
	defs := []capabilities.Def{{ID: "cap-x", Name: "Cap X"}}

	t.Run("does not read cached full-check results", func(t *testing.T) {
		reg := capabilities.NewRegistry(defs, map[string]capabilities.Checker{"cap-x": fullChecker}, time.Minute)
		reg.SetLivenessCheckers(map[string]capabilities.Checker{"cap-x": livenessChecker})

		fullChecker.ResetCalls()
		livenessChecker.ResetCalls()

		reg.Resolve(context.Background())
		if got := fullChecker.CallCount(); got != 1 {
			t.Fatalf("full checker should be called once, got %d", got)
		}

		states := reg.ResolveLiveness(context.Background())
		if got := livenessChecker.CallCount(); got != 1 {
			t.Errorf("liveness checker should be called once, got %d calls", got)
		}
		if len(states) != 1 || states[0].Message != "liveness ok" {
			t.Errorf("expected independent liveness result, got %+v", states)
		}
	})

	t.Run("uses liveness checker when cache is stale", func(t *testing.T) {
		reg := capabilities.NewRegistry(defs, map[string]capabilities.Checker{"cap-x": fullChecker}, 0)
		reg.SetLivenessCheckers(map[string]capabilities.Checker{"cap-x": livenessChecker})

		livenessChecker.ResetCalls()

		states := reg.ResolveLiveness(context.Background())
		if got := livenessChecker.CallCount(); got != 1 {
			t.Errorf("liveness checker should be called once, got %d", got)
		}
		if len(states) != 1 || states[0].Message != "liveness ok" {
			t.Errorf("expected liveness result, got %+v", states)
		}
	})

	t.Run("does not run full resolve when no liveness checker map is configured", func(t *testing.T) {
		reg := capabilities.NewRegistry(defs, map[string]capabilities.Checker{"cap-x": fullChecker}, 0)

		fullChecker.ResetCalls()

		states := reg.ResolveLiveness(context.Background())
		if got := fullChecker.CallCount(); got != 0 {
			t.Errorf("should not run full checker, got %d calls", got)
		}
		if states[0].Status != capabilities.StatusUnknown {
			t.Errorf("state = %+v, want unknown", states[0])
		}
	})

	t.Run("does not run full checker for missing liveness entries once liveness is configured", func(t *testing.T) {
		fullOnly := mocks.NewFakeChecker(capabilities.StatusAvailable, "full-only ok")
		reg := capabilities.NewRegistry(
			[]capabilities.Def{
				{ID: "cap-live", Name: "Cap Live"},
				{ID: "cap-full", Name: "Cap Full"},
			},
			map[string]capabilities.Checker{
				"cap-live": fullChecker,
				"cap-full": fullOnly,
			},
			0,
		)
		reg.SetLivenessCheckers(map[string]capabilities.Checker{"cap-live": livenessChecker})

		fullChecker.ResetCalls()
		fullOnly.ResetCalls()
		livenessChecker.ResetCalls()

		states := reg.ResolveLiveness(context.Background())
		if got := livenessChecker.CallCount(); got != 1 {
			t.Errorf("liveness checker should be called once, got %d", got)
		}
		if got := fullChecker.CallCount(); got != 0 {
			t.Errorf("full checker with liveness replacement should not be called, got %d", got)
		}
		if got := fullOnly.CallCount(); got != 0 {
			t.Errorf("full checker without liveness replacement should not be called, got %d", got)
		}
		if len(states) != 2 {
			t.Fatalf("states len = %d, want 2", len(states))
		}
		if states[0].Status != capabilities.StatusAvailable || states[0].Message != "liveness ok" {
			t.Errorf("state[0] = %+v, want liveness result", states[0])
		}
		if states[1].Status != capabilities.StatusUnknown || states[1].Message != "" {
			t.Errorf("state[1] = %+v, want unknown without full fallback", states[1])
		}
	})
}

func TestRegistry_ResolveForce(t *testing.T) {
	// A long TTL would normally cause Resolve to return cached values
	// after the first call. ResolveForce must bust the cache regardless.
	checker := mocks.NewFakeChecker(capabilities.StatusAvailable, "ok")
	defs := []capabilities.Def{{ID: "cap-y", Name: "Cap Y"}}
	checkers := map[string]capabilities.Checker{"cap-y": checker}

	reg := capabilities.NewRegistry(defs, checkers, time.Hour)

	reg.Resolve(context.Background())
	if got := checker.CallCount(); got != 1 {
		t.Fatalf("after first Resolve: calls = %d, want 1", got)
	}

	reg.Resolve(context.Background())
	if got := checker.CallCount(); got != 1 {
		t.Fatalf("after second Resolve (cached): calls = %d, want 1", got)
	}

	states := reg.ResolveForce(context.Background())
	if got := checker.CallCount(); got != 2 {
		t.Errorf("after ResolveForce: calls = %d, want 2 (cache should be busted)", got)
	}
	if len(states) != 1 || states[0].Status != capabilities.StatusAvailable {
		t.Errorf("unexpected ResolveForce result: %+v", states)
	}

	// And the freshly-forced value should now be cached for the next
	// (TTL-respecting) Resolve.
	reg.Resolve(context.Background())
	if got := checker.CallCount(); got != 2 {
		t.Errorf("Resolve after ResolveForce should hit cache: calls = %d, want 2", got)
	}
}

func TestRegistry_IsAvailable(t *testing.T) {
	defs := []capabilities.Def{
		{ID: "avail", Name: "Available"},
		{ID: "unavail", Name: "Unavailable"},
	}
	checkers := map[string]capabilities.Checker{
		"avail":   mocks.NewFakeChecker(capabilities.StatusAvailable, "ok"),
		"unavail": mocks.NewFakeChecker(capabilities.StatusUnavailable, "down"),
	}

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"available capability", "avail", true},
		{"unavailable capability", "unavail", false},
		{"unknown capability ID", "nonexistent", false},
	}

	reg := capabilities.NewRegistry(defs, checkers, time.Minute)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reg.IsAvailable(context.Background(), tt.id)
			if got != tt.want {
				t.Errorf("IsAvailable(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}
