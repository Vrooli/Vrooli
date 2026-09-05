package services

import (
	"testing"
	"time"
)

// TestSnapshotOperation_BackoffFromProto verifies backoff config extraction.
func TestSnapshotOperation_BackoffFromProto(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	// Operation with full backoff config
	svc.mu.Lock()
	svc.operations["tc-1"] = &AsyncOperation{
		ToolCallID: "tc-1",
		AsyncBehavior: &AsyncBehavior{
			StatusPolling: &StatusPolling{
				PollIntervalSeconds: 5,
				Backoff: &PollingBackoff{
					InitialIntervalSeconds: 2,
					MaxIntervalSeconds:     30,
					Multiplier:             1.5,
				},
			},
		},
	}
	svc.mu.Unlock()

	snap, ok := svc.snapshotOperation("tc-1")
	if !ok {
		t.Fatal("expected snapshot")
	}

	if snap.BackoffInitial != 2*time.Second {
		t.Errorf("expected initial=2s, got %v", snap.BackoffInitial)
	}
	if snap.BackoffMax != 30*time.Second {
		t.Errorf("expected max=30s, got %v", snap.BackoffMax)
	}
	if snap.BackoffMultiplier != 1.5 {
		t.Errorf("expected multiplier=1.5, got %f", snap.BackoffMultiplier)
	}
}

// TestSnapshotOperation_BackoffMinInterval verifies minimum interval enforcement.
func TestSnapshotOperation_BackoffMinInterval(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	// Operation with too-low interval (below MinPollInterval)
	svc.mu.Lock()
	svc.operations["tc-1"] = &AsyncOperation{
		ToolCallID: "tc-1",
		AsyncBehavior: &AsyncBehavior{
			StatusPolling: &StatusPolling{
				PollIntervalSeconds: 0, // Below minimum
				Backoff: &PollingBackoff{
					InitialIntervalSeconds: 0, // Also below minimum
					MaxIntervalSeconds:     60,
					Multiplier:             2.0,
				},
			},
		},
	}
	svc.mu.Unlock()

	snap, ok := svc.snapshotOperation("tc-1")
	if !ok {
		t.Fatal("expected snapshot")
	}

	// Should fall back to default, not go below minimum
	if snap.BackoffInitial < MinPollInterval {
		t.Errorf("expected initial >= %v (MinPollInterval), got %v", MinPollInterval, snap.BackoffInitial)
	}
}

// TestSnapshotOperation_BackoffInvalidMultiplier verifies multiplier validation.
func TestSnapshotOperation_BackoffInvalidMultiplier(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	// Operation with invalid multiplier (< 1.0)
	svc.mu.Lock()
	svc.operations["tc-1"] = &AsyncOperation{
		ToolCallID: "tc-1",
		AsyncBehavior: &AsyncBehavior{
			StatusPolling: &StatusPolling{
				PollIntervalSeconds: 5,
				Backoff: &PollingBackoff{
					InitialIntervalSeconds: 5,
					MaxIntervalSeconds:     30,
					Multiplier:             0.5, // Invalid - less than 1.0
				},
			},
		},
	}
	svc.mu.Unlock()

	snap, ok := svc.snapshotOperation("tc-1")
	if !ok {
		t.Fatal("expected snapshot")
	}

	// Invalid multiplier should result in no backoff (1.0)
	if snap.BackoffMultiplier != 1.0 {
		t.Errorf("expected multiplier=1.0 for invalid config, got %f", snap.BackoffMultiplier)
	}
}

// TestBackoffCalculation verifies the backoff interval growth calculation.
func TestBackoffCalculation(t *testing.T) {
	// This tests the math used in pollLoop for interval growth
	tests := []struct {
		name       string
		initial    time.Duration
		max        time.Duration
		multiplier float64
		iterations int
		expected   []time.Duration // Expected intervals after each iteration
	}{
		{
			name:       "standard backoff",
			initial:    2 * time.Second,
			max:        30 * time.Second,
			multiplier: 1.5,
			iterations: 5,
			expected: []time.Duration{
				2 * time.Second,          // Initial
				3 * time.Second,          // 2 * 1.5 = 3
				4500 * time.Millisecond,  // 3 * 1.5 = 4.5
				6750 * time.Millisecond,  // 4.5 * 1.5 = 6.75
				10125 * time.Millisecond, // 6.75 * 1.5 = 10.125
			},
		},
		{
			name:       "hits max",
			initial:    5 * time.Second,
			max:        10 * time.Second,
			multiplier: 2.0,
			iterations: 4,
			expected: []time.Duration{
				5 * time.Second,  // Initial
				10 * time.Second, // 5 * 2 = 10 (at max)
				10 * time.Second, // Capped at max
				10 * time.Second, // Capped at max
			},
		},
		{
			name:       "no backoff",
			initial:    5 * time.Second,
			max:        5 * time.Second,
			multiplier: 1.0,
			iterations: 3,
			expected: []time.Duration{
				5 * time.Second, // No change
				5 * time.Second, // No change
				5 * time.Second, // No change
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			currentInterval := tc.initial

			for i, expected := range tc.expected {
				// First iteration uses initial interval
				if i == 0 {
					if currentInterval != expected {
						t.Errorf("iteration %d: expected %v, got %v", i, expected, currentInterval)
					}
					continue
				}

				// Calculate next interval (same logic as pollLoop)
				if tc.multiplier > 1.0 {
					nextInterval := time.Duration(float64(currentInterval) * tc.multiplier)
					if nextInterval > tc.max {
						nextInterval = tc.max
					}
					currentInterval = nextInterval
				}

				// Allow 1ms tolerance for floating point
				diff := currentInterval - expected
				if diff < 0 {
					diff = -diff
				}
				if diff > time.Millisecond {
					t.Errorf("iteration %d: expected ~%v, got %v", i, expected, currentInterval)
				}
			}
		})
	}
}

// TestExtractOperationID verifies operation ID extraction.
func TestExtractOperationID(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	tests := []struct {
		name      string
		result    interface{}
		fieldPath string
		expected  string
		wantErr   bool
	}{
		{
			name:      "simple field",
			result:    map[string]interface{}{"run_id": "run-123"},
			fieldPath: "run_id",
			expected:  "run-123",
		},
		{
			name: "nested field",
			result: map[string]interface{}{
				"data": map[string]interface{}{
					"execution_id": "exec-456",
				},
			},
			fieldPath: "data.execution_id",
			expected:  "exec-456",
		},
		{
			name:      "missing field",
			result:    map[string]interface{}{"other": "value"},
			fieldPath: "run_id",
			wantErr:   true,
		},
		{
			name:      "not a map",
			result:    "just a string",
			fieldPath: "run_id",
			wantErr:   true,
		},
		{
			name:      "empty field value",
			result:    map[string]interface{}{"run_id": ""},
			fieldPath: "run_id",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.extractOperationID(tc.result, tc.fieldPath)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}
