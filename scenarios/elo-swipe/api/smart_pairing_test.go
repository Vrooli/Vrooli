package main

import (
	"encoding/json"
	"testing"
)

func TestGenerateFallbackPairs(t *testing.T) {
	sp := NewSmartPairing(nil)
	
	tests := []struct {
		name          string
		items         []PairItem
		expectedPairs int
	}{
		{
			name: "Two items",
			items: []PairItem{
				{ID: "1", Content: json.RawMessage(`{"name": "Item A"}`)},
				{ID: "2", Content: json.RawMessage(`{"name": "Item B"}`)},
			},
			expectedPairs: 1,
		},
		{
			name: "Three items",
			items: []PairItem{
				{ID: "1", Content: json.RawMessage(`{"name": "Item A"}`)},
				{ID: "2", Content: json.RawMessage(`{"name": "Item B"}`)},
				{ID: "3", Content: json.RawMessage(`{"name": "Item C"}`)},
			},
			expectedPairs: 3,
		},
		{
			name:          "No items",
			items:         []PairItem{},
			expectedPairs: 0,
		},
		{
			name: "One item",
			items: []PairItem{
				{ID: "1", Content: json.RawMessage(`{"name": "Item A"}`)},
			},
			expectedPairs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairs := sp.generateFallbackPairs(tt.items)
			if len(pairs) != tt.expectedPairs {
				t.Errorf("generateFallbackPairs() got %d pairs, expected %d", len(pairs), tt.expectedPairs)
			}
			
			// Verify no duplicate pairs
			pairMap := make(map[string]bool)
			for _, pair := range pairs {
				key := pair.ItemAID + "-" + pair.ItemBID
				if pairMap[key] {
					t.Errorf("Found duplicate pair: %s", key)
				}
				pairMap[key] = true
				
				// Verify items are different
				if pair.ItemAID == pair.ItemBID {
					t.Errorf("Pair contains same item: %s", pair.ItemAID)
				}
			}
		})
	}
}

func TestParseAIResponse(t *testing.T) {
	sp := NewSmartPairing(nil)
	
	tests := []struct {
		name        string
		response    string
		expectError bool
		expectPairs int
	}{
		{
			name: "Valid JSON response",
			response: `{
				"suggested_pairs": [
					{"item_a_id": "1", "item_b_id": "2", "priority": 80},
					{"item_a_id": "2", "item_b_id": "3", "priority": 60}
				]
			}`,
			expectError: false,
			expectPairs: 2,
		},
		{
			name: "JSON with markdown formatting",
			response: "```json\n" + `{
				"suggested_pairs": [
					{"item_a_id": "1", "item_b_id": "2", "priority": 90}
				]
			}` + "\n```",
			expectError: false,
			expectPairs: 1,
		},
		{
			name:        "Invalid JSON",
			response:    "not json at all",
			expectError: true,
			expectPairs: 0,
		},
		{
			name: "Same item in pair (should be filtered)",
			response: `{
				"suggested_pairs": [
					{"item_a_id": "1", "item_b_id": "1", "priority": 80}
				]
			}`,
			expectError: false,
			expectPairs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairs, err := sp.parseAIResponse(tt.response)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("parseAIResponse() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("parseAIResponse() unexpected error: %v", err)
				}
				if len(pairs) != tt.expectPairs {
					t.Errorf("parseAIResponse() got %d pairs, expected %d", len(pairs), tt.expectPairs)
				}
			}
		})
	}
}
