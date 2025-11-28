package main

import (
	"testing"
)

// [REQ:PCT-REQ-INTEGRATION] Ecosystem manager task matching utilities
func TestMatchesTaskTarget(t *testing.T) {
	tests := []struct {
		name        string
		item        ecosystemTaskItem
		targetLower string
		expected    bool
	}{
		{
			name: "exact match on Target field",
			item: ecosystemTaskItem{
				Target:  "scenario-alpha",
				Targets: []string{},
			},
			targetLower: "scenario-alpha",
			expected:    true,
		},
		{
			name: "case insensitive match on Target field",
			item: ecosystemTaskItem{
				Target:  "Scenario-Alpha",
				Targets: []string{},
			},
			targetLower: "scenario-alpha",
			expected:    true,
		},
		{
			name: "match in Targets array",
			item: ecosystemTaskItem{
				Target:  "",
				Targets: []string{"scenario-beta", "scenario-gamma"},
			},
			targetLower: "scenario-beta",
			expected:    true,
		},
		{
			name: "case insensitive match in Targets array",
			item: ecosystemTaskItem{
				Target:  "",
				Targets: []string{"Scenario-Beta", "Scenario-Gamma"},
			},
			targetLower: "scenario-gamma",
			expected:    true,
		},
		{
			name: "no match",
			item: ecosystemTaskItem{
				Target:  "scenario-alpha",
				Targets: []string{"scenario-beta"},
			},
			targetLower: "scenario-delta",
			expected:    false,
		},
		{
			name: "empty Target and Targets",
			item: ecosystemTaskItem{
				Target:  "",
				Targets: []string{},
			},
			targetLower: "scenario-alpha",
			expected:    false,
		},
		{
			name: "no match with whitespace (EqualFold doesn't trim)",
			item: ecosystemTaskItem{
				Target:  "  scenario-alpha  ",
				Targets: []string{},
			},
			targetLower: "scenario-alpha",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesTaskTarget(tt.item, tt.targetLower)
			if result != tt.expected {
				t.Errorf("matchesTaskTarget() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// [REQ:PCT-REQ-INTEGRATION] Ecosystem manager task title derivation
func TestDeriveTaskTitle(t *testing.T) {
	tests := []struct {
		name       string
		requested  string
		entityName string
		expected   string
	}{
		{
			name:       "requested title provided",
			requested:  "Implement user authentication",
			entityName: "auth-service",
			expected:   "Implement user authentication",
		},
		{
			name:       "empty requested title falls back",
			requested:  "",
			entityName: "auth-service",
			expected:   "Implement auth-service requirements",
		},
		{
			name:       "whitespace only requested title falls back",
			requested:  "   \t\n   ",
			entityName: "payment-gateway",
			expected:   "Implement payment-gateway requirements",
		},
		{
			name:       "requested title with whitespace is trimmed",
			requested:  "  Implement dashboard analytics  ",
			entityName: "analytics",
			expected:   "Implement dashboard analytics",
		},
		{
			name:       "both empty uses entity name",
			requested:  "",
			entityName: "notification-service",
			expected:   "Implement notification-service requirements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveTaskTitle(tt.requested, tt.entityName)
			if result != tt.expected {
				t.Errorf("deriveTaskTitle(%q, %q) = %q, want %q",
					tt.requested, tt.entityName, result, tt.expected)
			}
		})
	}
}

// [REQ:PCT-REQ-INTEGRATION] Default string utility for fallback values
func TestDefaultString(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		expected string
	}{
		{
			name:     "non-empty value returns value",
			value:    "custom value",
			fallback: "default",
			expected: "custom value",
		},
		{
			name:     "empty value returns fallback",
			value:    "",
			fallback: "default",
			expected: "default",
		},
		{
			name:     "whitespace only returns fallback",
			value:    "   \t\n   ",
			fallback: "default",
			expected: "default",
		},
		{
			name:     "value with whitespace is trimmed",
			value:    "  custom  ",
			fallback: "default",
			expected: "custom",
		},
		{
			name:     "both empty",
			value:    "",
			fallback: "",
			expected: "",
		},
		{
			name:     "empty value with empty fallback",
			value:    "",
			fallback: "",
			expected: "",
		},
		{
			name:     "fallback with whitespace",
			value:    "",
			fallback: "  fallback  ",
			expected: "  fallback  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultString(tt.value, tt.fallback)
			if result != tt.expected {
				t.Errorf("defaultString(%q, %q) = %q, want %q",
					tt.value, tt.fallback, result, tt.expected)
			}
		})
	}
}
