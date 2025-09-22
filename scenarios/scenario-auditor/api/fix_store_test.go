package main

import "testing"

func TestAutomatedFixStoreEnableDefaults(t *testing.T) {
	store := &AutomatedFixStore{
		config: AutomatedFixConfig{
			Enabled:        false,
			ViolationTypes: []string{"security"},
			Severities:     []string{"critical", "high"},
			Strategy:       defaultAutomatedFixStrategy,
			LoopDelay:      defaultLoopDelaySeconds,
			TimeoutSeconds: defaultTimeoutSeconds,
			MaxFixes:       defaultMaxFixes,
		},
		history:    nil,
		maxHistory: 10,
	}

	cfg := store.Enable(AutomatedFixConfigInput{})
	if !cfg.Enabled {
		t.Fatalf("expected automation to be enabled")
	}
	if cfg.Strategy != defaultAutomatedFixStrategy {
		t.Errorf("expected default strategy %q, got %q", defaultAutomatedFixStrategy, cfg.Strategy)
	}
	if cfg.LoopDelay != defaultLoopDelaySeconds {
		t.Errorf("expected default loop delay %d, got %d", defaultLoopDelaySeconds, cfg.LoopDelay)
	}
	if cfg.TimeoutSeconds != defaultTimeoutSeconds {
		t.Errorf("expected default timeout %d, got %d", defaultTimeoutSeconds, cfg.TimeoutSeconds)
	}
	if cfg.MaxFixes != defaultMaxFixes {
		t.Errorf("expected default max fixes %d, got %d", defaultMaxFixes, cfg.MaxFixes)
	}
	if len(cfg.ViolationTypes) == 0 {
		t.Errorf("expected violation types to default to security")
	}
	if len(cfg.Severities) == 0 {
		t.Errorf("expected severities to default to critical/high")
	}
	if cfg.Model != openRouterModel {
		t.Errorf("expected default model %q, got %q", openRouterModel, cfg.Model)
	}
}

func TestAutomatedFixStoreEnableCustom(t *testing.T) {
	store := &AutomatedFixStore{
		config: AutomatedFixConfig{
			Enabled:        false,
			ViolationTypes: []string{"security"},
			Severities:     []string{"critical", "high"},
			Strategy:       defaultAutomatedFixStrategy,
			LoopDelay:      defaultLoopDelaySeconds,
			TimeoutSeconds: defaultTimeoutSeconds,
			MaxFixes:       defaultMaxFixes,
		},
		history:    nil,
		maxHistory: 10,
	}

	loopDelay := 45
	timeout := 900
	maxFixes := 25
	model := "openrouter/google/gemini-2.5-flash"
	cfg := store.Enable(AutomatedFixConfigInput{
		ViolationTypes: []string{"standards"},
		Severities:     []string{"low", "medium"},
		Strategy:       "low_first",
		LoopDelay:      &loopDelay,
		TimeoutSeconds: &timeout,
		MaxFixes:       &maxFixes,
		Model:          model,
	})

	if cfg.Strategy != "low_first" {
		t.Fatalf("expected low_first strategy, got %s", cfg.Strategy)
	}
	if cfg.LoopDelay != 45 {
		t.Fatalf("expected loop delay 45, got %d", cfg.LoopDelay)
	}
	if cfg.TimeoutSeconds != 900 {
		t.Fatalf("expected timeout 900, got %d", cfg.TimeoutSeconds)
	}
	if cfg.MaxFixes != 25 {
		t.Fatalf("expected max fixes 25, got %d", cfg.MaxFixes)
	}
	if len(cfg.ViolationTypes) != 1 || cfg.ViolationTypes[0] != "standards" {
		t.Fatalf("expected violation types to include standards only: %#v", cfg.ViolationTypes)
	}
	if len(cfg.Severities) != 2 {
		t.Fatalf("expected two severities, got %#v", cfg.Severities)
	}
	if cfg.Model != normalizeAgentModel(model) {
		t.Fatalf("expected model %s, got %s", normalizeAgentModel(model), cfg.Model)
	}
}
