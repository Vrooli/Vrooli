package jsonutil

import (
	"encoding/json"
	"testing"
)

func TestRepairTruncatedJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantNil     bool
		wantCount   int // expected number of items in the first array
		wantArrayAt string
	}{
		{
			name:      "valid JSON still produces output (callers gate on parse failure)",
			input:     `{"questions":[{"id":"q1"},{"id":"q2"}]}`,
			wantNil:   false,
			wantCount: 2,
		},
		{
			name:      "truncated mid-object recovers complete objects",
			input:     `{"questions":[{"id":"q1","text":"hello"},{"id":"q2","text":"wor`,
			wantNil:   false,
			wantCount: 1,
		},
		{
			name:      "truncated after comma recovers complete objects",
			input:     `{"questions":[{"id":"q1","text":"hello"},{"id":"q2","text":"world"},{"id":"q3`,
			wantNil:   false,
			wantCount: 2,
		},
		{
			name:      "truncated mid-string with escapes",
			input:     `{"questions":[{"id":"q1","text":"line1\nline2"},{"id":"q2","text":"trunc`,
			wantNil:   false,
			wantCount: 1,
		},
		{
			name:      "truncated mid-nested array",
			input:     `{"questions":[{"id":"q1","options":["a","b"]},{"id":"q2","options":["c`,
			wantNil:   false,
			wantCount: 1,
		},
		{
			name:    "no complete objects",
			input:   `{"questions":[{"id":"q1","tex`,
			wantNil: true,
		},
		{
			name:    "no array found",
			input:   `{"questions": "not an array"`,
			wantNil: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantNil: true,
		},
		{
			name:      "suggestions array key works too",
			input:     `{"suggestions":[{"id":"s1","text":"good"},{"id":"s2","te`,
			wantNil:   false,
			wantCount: 1,
		},
		{
			name:      "three complete one truncated",
			input:     `{"questions":[{"id":"q1"},{"id":"q2"},{"id":"q3"},{"id":"q4","context":"Currently 46 resources are configured in service.json. Guiding through all of them would be overwhelming. The description mentions starting`,
			wantNil:   false,
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RepairTruncatedJSON([]byte(tt.input))
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got: %s", string(result))
				}
				return
			}
			if result == nil {
				t.Fatal("expected repaired JSON, got nil")
			}
			if !json.Valid(result) {
				t.Errorf("repaired JSON is not valid: %s", string(result))
			}

			// Parse and verify item count
			var parsed map[string][]json.RawMessage
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("failed to unmarshal repaired JSON: %v", err)
			}
			// Find the first array
			var count int
			for _, arr := range parsed {
				count = len(arr)
				break
			}
			if count != tt.wantCount {
				t.Errorf("expected %d items, got %d in: %s", tt.wantCount, count, string(result))
			}
		})
	}
}
