package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSpeakerConfigMissingUsesSafeDefaults(t *testing.T) {
	cfg, err := LoadSpeakerConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("LoadSpeakerConfig() error = %v", err)
	}
	if cfg.Enabled || cfg.FallbackWithoutVerification {
		t.Fatalf("missing config must keep verification disabled and fallback off: %+v", cfg)
	}
	if cfg.Mode != "filter" || cfg.RejectBehavior != "drop" || cfg.Threshold != 0.5 {
		t.Fatalf("unexpected policy defaults: %+v", cfg)
	}
	if cfg.MinDecisionSeconds != DefaultMinDecisionSeconds || cfg.ScoreSmoothing != DefaultScoreSmoothing {
		t.Fatalf("unexpected session defaults: %+v", cfg)
	}
}

func TestLoadSpeakerConfigAppliesZeroValueDefaultsAndPreservesFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "speaker.json")
	if err := os.WriteFile(path, []byte(`{"enabled":true,"profileIds":["voice-a"],"fallbackWithoutVerification":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadSpeakerConfig(path)
	if err != nil {
		t.Fatalf("LoadSpeakerConfig() error = %v", err)
	}
	if !cfg.Enabled || !cfg.FallbackWithoutVerification {
		t.Fatalf("explicit enable/fallback policy was not preserved: %+v", cfg)
	}
	if cfg.Mode != "filter" || cfg.RejectBehavior != "drop" || cfg.Threshold != 0.5 || cfg.MinDecisionSeconds != DefaultMinDecisionSeconds || cfg.ScoreSmoothing != DefaultScoreSmoothing {
		t.Fatalf("zero-valued fields did not resolve to defaults: %+v", cfg)
	}
}

func TestLoadSpeakerConfigFailsClosedOnInvalidEnabledPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "speaker.json")
	if err := os.WriteFile(path, []byte(`{"enabled":true,"fallbackWithoutVerification":true,"profileIds":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadSpeakerConfig(path)
	if err == nil {
		t.Fatal("invalid enabled policy must report an error")
	}
	if cfg.Enabled || cfg.FallbackWithoutVerification {
		t.Fatalf("invalid policy must not be returned as active or permissive: %+v", cfg)
	}
}

func TestSpeakerConfigPatchCanExplicitlyDisableFallback(t *testing.T) {
	base := DefaultSpeakerConfig()
	base.Enabled = true
	base.ProfileIDs = []string{"voice-a"}
	base.FallbackWithoutVerification = true
	fallback := false
	got := (SpeakerConfigPatch{FallbackWithoutVerification: &fallback}).Apply(base)
	if got.FallbackWithoutVerification {
		t.Fatal("explicit false patch must disable fallback")
	}
	if !got.Enabled || len(got.ProfileIDs) != 1 {
		t.Fatalf("patch changed unrelated policy fields: %+v", got)
	}
}
