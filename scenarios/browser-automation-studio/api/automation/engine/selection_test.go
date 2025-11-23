package engine

import "testing"

func TestFeatureEnabledDefaultsToOn(t *testing.T) {
	cfg := SelectionConfig{}
	if !cfg.FeatureEnabled() {
		t.Fatal("expected blank feature flag to enable automation executor by default")
	}
}

func TestFeatureEnabledExplicitOff(t *testing.T) {
	cfg := SelectionConfig{FeatureFlag: "off"}
	if cfg.FeatureEnabled() {
		t.Fatal("expected feature flag 'off' to disable automation executor")
	}
	cfg.FeatureFlag = "false"
	if cfg.FeatureEnabled() {
		t.Fatal("expected feature flag 'false' to disable automation executor")
	}
}

func TestFeatureEnabledOn(t *testing.T) {
	cfg := SelectionConfig{FeatureFlag: "on"}
	if !cfg.FeatureEnabled() {
		t.Fatal("expected feature flag 'on' to enable automation executor")
	}
}
