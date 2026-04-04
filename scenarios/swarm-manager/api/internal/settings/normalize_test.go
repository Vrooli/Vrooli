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

func TestNormalizeMaxConcurrentExecutions(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"zero clamped to 1", 0, 1},
		{"negative clamped to 1", -5, 1},
		{"min boundary", 1, 1},
		{"mid range", 10, 10},
		{"max boundary", 20, 20},
		{"over max clamped to 20", 50, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := normalizeSettings(Settings{MaxConcurrentExecutions: tt.input})
			if s.MaxConcurrentExecutions != tt.want {
				t.Fatalf("MaxConcurrentExecutions: got %d, want %d", s.MaxConcurrentExecutions, tt.want)
			}
		})
	}
}

func TestNormalizeMaxQueueDepth(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"negative clamped to 0", -1, 0},
		{"zero allowed (unlimited)", 0, 0},
		{"mid range", 50, 50},
		{"max boundary", 100, 100},
		{"over max clamped to 100", 200, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := normalizeSettings(Settings{MaxQueueDepth: tt.input})
			if s.MaxQueueDepth != tt.want {
				t.Fatalf("MaxQueueDepth: got %d, want %d", s.MaxQueueDepth, tt.want)
			}
		})
	}
}

func TestNormalizeCircuitBreakerThreshold(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"zero clamped to 1", 0, 1},
		{"min boundary", 1, 1},
		{"mid range", 5, 5},
		{"max boundary", 10, 10},
		{"over max clamped to 10", 15, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := normalizeSettings(Settings{CircuitBreakerThreshold: tt.input})
			if s.CircuitBreakerThreshold != tt.want {
				t.Fatalf("CircuitBreakerThreshold: got %d, want %d", s.CircuitBreakerThreshold, tt.want)
			}
		})
	}
}

func TestNormalizeCircuitBreakerCooldownMinutes(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"zero clamped to 5", 0, 5},
		{"below min clamped to 5", 3, 5},
		{"min boundary", 5, 5},
		{"mid range", 60, 60},
		{"max boundary", 1440, 1440},
		{"over max clamped to 1440", 2000, 1440},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := normalizeSettings(Settings{CircuitBreakerCooldownMinutes: tt.input})
			if s.CircuitBreakerCooldownMinutes != tt.want {
				t.Fatalf("CircuitBreakerCooldownMinutes: got %d, want %d", s.CircuitBreakerCooldownMinutes, tt.want)
			}
		})
	}
}

func TestNormalizeExecutionCostCapPerRun(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  float64
	}{
		{"negative clamped to 0", -5.0, 0.0},
		{"zero allowed (unlimited)", 0.0, 0.0},
		{"positive value preserved", 10.0, 10.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := normalizeSettings(Settings{ExecutionCostCapPerRun: tt.input})
			if s.ExecutionCostCapPerRun != tt.want {
				t.Fatalf("ExecutionCostCapPerRun: got %f, want %f", s.ExecutionCostCapPerRun, tt.want)
			}
		})
	}
}

func TestNormalizeCostPerTurnEstimate(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  float64
	}{
		{"negative clamped to 0", -1.0, 0.0},
		{"zero allowed", 0.0, 0.0},
		{"mid range", 0.10, 0.10},
		{"max boundary", 5.0, 5.0},
		{"over max clamped to 5", 10.0, 5.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := normalizeSettings(Settings{CostPerTurnEstimate: tt.input})
			if s.CostPerTurnEstimate != tt.want {
				t.Fatalf("CostPerTurnEstimate: got %f, want %f", s.CostPerTurnEstimate, tt.want)
			}
		})
	}
}
