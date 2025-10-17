package main

import (
	"testing"
	"time"
)

// TestCalculateNextDate tests the date calculation logic for recurring invoices
func TestCalculateNextDate(t *testing.T) {
	tests := []struct {
		name          string
		startDate     string
		frequency     string
		intervalCount int
		expected      string
	}{
		{
			name:          "monthly interval",
			startDate:     "2025-01-15",
			frequency:     "monthly",
			intervalCount: 1,
			expected:      "2025-02-15",
		},
		{
			name:          "quarterly interval",
			startDate:     "2025-01-15",
			frequency:     "quarterly",
			intervalCount: 1,
			expected:      "2025-04-15",
		},
		{
			name:          "yearly interval",
			startDate:     "2025-01-15",
			frequency:     "yearly",
			intervalCount: 1,
			expected:      "2026-01-15",
		},
		{
			name:          "weekly interval",
			startDate:     "2025-01-15",
			frequency:     "weekly",
			intervalCount: 2,
			expected:      "2025-01-29",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateNextDate(tt.startDate, tt.frequency, tt.intervalCount)
			if result != tt.expected {
				t.Errorf("calculateNextDate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDateCalculation tests basic date arithmetic
func TestDateCalculation(t *testing.T) {
	tests := []struct {
		name      string
		startDate time.Time
		addDays   int
		wantDay   int
		wantMonth time.Month
	}{
		{
			name:      "add 7 days",
			startDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			addDays:   7,
			wantDay:   22,
			wantMonth: time.January,
		},
		{
			name:      "add 30 days",
			startDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			addDays:   30,
			wantDay:   14,
			wantMonth: time.February,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.startDate.AddDate(0, 0, tt.addDays)
			if result.Day() != tt.wantDay || result.Month() != tt.wantMonth {
				t.Errorf("Date calculation = %v/%v, want %v/%v",
					result.Month(), result.Day(), tt.wantMonth, tt.wantDay)
			}
		})
	}
}
