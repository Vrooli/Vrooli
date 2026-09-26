package config

import (
	"testing"

	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

func strptr(s string) *string { return &s }

// TestResolveArtifactSettingsWithDefault_NilUsesConfiguredDefault proves that an
// execution with no per-execution artifact config resolves to the operator-
// configured default profile rather than a hardcoded "full".
func TestResolveArtifactSettingsWithDefault_NilUsesConfiguredDefault(t *testing.T) {
	got := ResolveArtifactSettingsWithDefault(nil, ProfileStandard)
	want := artifactProfiles[ProfileStandard]
	if got != want {
		t.Fatalf("nil config with standard default: got %+v, want %+v", got, want)
	}

	// DOM snapshots are the largest artifact category and must be OFF for standard.
	if got.CollectDOMSnapshots {
		t.Fatalf("standard default should not collect DOM snapshots")
	}
	if !got.CollectScreenshots {
		t.Fatalf("standard default should still collect screenshots")
	}
}

// TestResolveArtifactSettingsWithDefault_EmptyProfileUsesDefault proves that a
// config present but with an empty profile string still falls back to the
// configured default.
func TestResolveArtifactSettingsWithDefault_EmptyProfileUsesDefault(t *testing.T) {
	cfg := &basexecution.ArtifactCollectionConfig{Profile: strptr("")}
	got := ResolveArtifactSettingsWithDefault(cfg, ProfileMinimal)
	if got != artifactProfiles[ProfileMinimal] {
		t.Fatalf("empty profile should fall back to default profile minimal, got %+v", got)
	}
}

// TestResolveArtifactSettingsWithDefault_ExplicitProfileWins proves that an
// explicit per-execution profile always overrides the global default.
func TestResolveArtifactSettingsWithDefault_ExplicitProfileWins(t *testing.T) {
	cases := []struct {
		profile string
		want    ArtifactCollectionSettings
	}{
		{ProfileFull, artifactProfiles[ProfileFull]},
		{ProfileMinimal, artifactProfiles[ProfileMinimal]},
		{ProfileDebug, artifactProfiles[ProfileDebug]},
		{ProfileNone, artifactProfiles[ProfileNone]},
	}
	for _, tc := range cases {
		cfg := &basexecution.ArtifactCollectionConfig{Profile: strptr(tc.profile)}
		// Global default is standard, but the explicit profile must win.
		got := ResolveArtifactSettingsWithDefault(cfg, ProfileStandard)
		if got != tc.want {
			t.Fatalf("profile %q: got %+v, want %+v", tc.profile, got, tc.want)
		}
	}
}

// TestResolveArtifactSettingsWithDefault_CustomProfile proves custom toggles are
// honored when explicitly requested.
func TestResolveArtifactSettingsWithDefault_CustomProfile(t *testing.T) {
	collectDOM := true
	collectScreens := false
	cfg := &basexecution.ArtifactCollectionConfig{
		Profile:             strptr(ProfileCustom),
		CollectDomSnapshots: &collectDOM,
		CollectScreenshots:  &collectScreens,
	}
	got := ResolveArtifactSettingsWithDefault(cfg, ProfileStandard)
	if !got.CollectDOMSnapshots {
		t.Fatalf("custom profile should honor collect_dom_snapshots=true")
	}
	if got.CollectScreenshots {
		t.Fatalf("custom profile should honor collect_screenshots=false")
	}
}

// TestResolveArtifactSettingsWithDefault_SizeOverridesOnDefault proves size-limit
// overrides apply even when the profile is defaulted.
func TestResolveArtifactSettingsWithDefault_SizeOverridesOnDefault(t *testing.T) {
	var maxShot int32 = 1024
	cfg := &basexecution.ArtifactCollectionConfig{MaxScreenshotBytes: &maxShot}
	got := ResolveArtifactSettingsWithDefault(cfg, ProfileStandard)
	if got.MaxScreenshotBytes != 1024 {
		t.Fatalf("expected size override 1024, got %d", got.MaxScreenshotBytes)
	}
	// Profile toggles still come from the default (standard).
	if got.CollectDOMSnapshots {
		t.Fatalf("standard default should still disable DOM snapshots")
	}
}

// TestDefaultArtifactSettingsForProfile_UnknownFallsBackToStandard proves an
// unknown/empty profile name falls back to standard, not full.
func TestDefaultArtifactSettingsForProfile_UnknownFallsBackToStandard(t *testing.T) {
	for _, name := range []string{"", "bogus", "  "} {
		if got := DefaultArtifactSettingsForProfile(name); got != artifactProfiles[ProfileStandard] {
			t.Fatalf("profile %q should fall back to standard, got %+v", name, got)
		}
	}
}

// TestConfigDefaultProfileIsStandard proves the loaded config default profile is
// "standard" when BAS_ARTIFACT_DEFAULT_PROFILE is unset, and validates.
func TestConfigDefaultProfileIsStandard(t *testing.T) {
	t.Setenv("BAS_ARTIFACT_DEFAULT_PROFILE", "")
	cfg := loadFromEnv()
	if cfg.ArtifactLimits.DefaultProfile != ProfileStandard {
		t.Fatalf("expected default profile %q, got %q", ProfileStandard, cfg.ArtifactLimits.DefaultProfile)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("standard default profile should validate: %v", err)
	}
}

// TestConfigDefaultProfileHonorsEnv proves the operator override is honored.
func TestConfigDefaultProfileHonorsEnv(t *testing.T) {
	t.Setenv("BAS_ARTIFACT_DEFAULT_PROFILE", "full")
	cfg := loadFromEnv()
	if cfg.ArtifactLimits.DefaultProfile != ProfileFull {
		t.Fatalf("expected env override full, got %q", cfg.ArtifactLimits.DefaultProfile)
	}
	settings := ResolveArtifactSettingsWithDefault(nil, cfg.ArtifactLimits.DefaultProfile)
	if !settings.CollectDOMSnapshots {
		t.Fatalf("full default should collect DOM snapshots")
	}
}

func TestConfigExplicitExecutionTimeoutCapHonorsEnv(t *testing.T) {
	t.Setenv("BAS_EXECUTION_MAX_EXPLICIT_TIMEOUT_MS", "5400000")
	cfg := loadFromEnv()
	if got := cfg.Execution.ExplicitMaxTimeout.Milliseconds(); got != 5400000 {
		t.Fatalf("explicit timeout cap = %dms, want 5400000ms", got)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("explicit timeout cap should validate: %v", err)
	}
}
