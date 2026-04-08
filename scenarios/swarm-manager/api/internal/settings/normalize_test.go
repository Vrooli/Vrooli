package settings

import (
	"testing"
)

func TestNormalizeSettingsDefaultsTheme(t *testing.T) {
	normalized := normalizeSettings(Settings{Theme: ""})
	if normalized.Theme != "dark" {
		t.Fatalf("expected default theme dark, got %q", normalized.Theme)
	}
}

func TestNormalizeIntFields(t *testing.T) {
	type intCase struct {
		name  string
		input int
		want  int
	}

	fields := []struct {
		field string
		cases []intCase
		build func(int) Settings
		get   func(Settings) int
	}{
		{
			field: "MaxConcurrentExecutions",
			cases: []intCase{
				{"zero clamped to 1", 0, 1},
				{"negative clamped to 1", -5, 1},
				{"min boundary", 1, 1},
				{"mid range", 10, 10},
				{"max boundary", 20, 20},
				{"over max clamped to 20", 50, 20},
			},
			build: func(v int) Settings { return Settings{MaxConcurrentExecutions: v} },
			get:   func(s Settings) int { return s.MaxConcurrentExecutions },
		},
		{
			field: "MaxQueueDepth",
			cases: []intCase{
				{"negative clamped to 0", -1, 0},
				{"zero allowed (unlimited)", 0, 0},
				{"mid range", 50, 50},
				{"max boundary", 100, 100},
				{"over max clamped to 100", 200, 100},
			},
			build: func(v int) Settings { return Settings{MaxQueueDepth: v} },
			get:   func(s Settings) int { return s.MaxQueueDepth },
		},
		{
			field: "CircuitBreakerThreshold",
			cases: []intCase{
				{"zero clamped to 1", 0, 1},
				{"min boundary", 1, 1},
				{"mid range", 5, 5},
				{"max boundary", 10, 10},
				{"over max clamped to 10", 15, 10},
			},
			build: func(v int) Settings { return Settings{CircuitBreakerThreshold: v} },
			get:   func(s Settings) int { return s.CircuitBreakerThreshold },
		},
		{
			field: "CircuitBreakerCooldownMinutes",
			cases: []intCase{
				{"zero clamped to 5", 0, 5},
				{"below min clamped to 5", 3, 5},
				{"min boundary", 5, 5},
				{"mid range", 60, 60},
				{"max boundary", 1440, 1440},
				{"over max clamped to 1440", 2000, 1440},
			},
			build: func(v int) Settings { return Settings{CircuitBreakerCooldownMinutes: v} },
			get:   func(s Settings) int { return s.CircuitBreakerCooldownMinutes },
		},
	}

	for _, f := range fields {
		t.Run(f.field, func(t *testing.T) {
			for _, tc := range f.cases {
				t.Run(tc.name, func(t *testing.T) {
					got := f.get(normalizeSettings(f.build(tc.input)))
					if got != tc.want {
						t.Fatalf("%s: got %d, want %d", f.field, got, tc.want)
					}
				})
			}
		})
	}
}

func TestNormalizeFloatFields(t *testing.T) {
	type floatCase struct {
		name  string
		input float64
		want  float64
	}

	fields := []struct {
		field string
		cases []floatCase
		build func(float64) Settings
		get   func(Settings) float64
	}{
		{
			field: "ExecutionCostCapPerRun",
			cases: []floatCase{
				{"negative clamped to 0", -5.0, 0.0},
				{"zero allowed (unlimited)", 0.0, 0.0},
				{"positive value preserved", 10.0, 10.0},
			},
			build: func(v float64) Settings { return Settings{ExecutionCostCapPerRun: v} },
			get:   func(s Settings) float64 { return s.ExecutionCostCapPerRun },
		},
		{
			field: "CostPerTurnEstimate",
			cases: []floatCase{
				{"negative clamped to 0", -1.0, 0.0},
				{"zero allowed", 0.0, 0.0},
				{"mid range", 0.10, 0.10},
				{"max boundary", 5.0, 5.0},
				{"over max clamped to 5", 10.0, 5.0},
			},
			build: func(v float64) Settings { return Settings{CostPerTurnEstimate: v} },
			get:   func(s Settings) float64 { return s.CostPerTurnEstimate },
		},
	}

	for _, f := range fields {
		t.Run(f.field, func(t *testing.T) {
			for _, tc := range f.cases {
				t.Run(tc.name, func(t *testing.T) {
					got := f.get(normalizeSettings(f.build(tc.input)))
					if got != tc.want {
						t.Fatalf("%s: got %f, want %f", f.field, got, tc.want)
					}
				})
			}
		})
	}
}
