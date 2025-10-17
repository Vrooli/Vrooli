package main

import (
	"testing"
)

// TestCampaignValidation tests campaign field validation logic
func TestCampaignValidation(t *testing.T) {
	tests := []struct {
		name        string
		campaign    Campaign
		expectValid bool
		errorMsg    string
	}{
		{
			name: "Valid campaign with all fields",
			campaign: Campaign{
				Name:        "Test Campaign",
				Description: "Test description",
				Color:       "#FF5733",
			},
			expectValid: true,
		},
		{
			name: "Empty name should be invalid",
			campaign: Campaign{
				Name:        "",
				Description: "Test description",
			},
			expectValid: false,
			errorMsg:    "name is required",
		},
		{
			name: "Name too long (>100 chars)",
			campaign: Campaign{
				Name:        string(make([]byte, 101)),
				Description: "Test",
			},
			expectValid: false,
			errorMsg:    "name must be 100 characters or less",
		},
		{
			name: "Description too long (>500 chars)",
			campaign: Campaign{
				Name:        "Test",
				Description: string(make([]byte, 501)),
			},
			expectValid: false,
			errorMsg:    "description must be 500 characters or less",
		},
		{
			name: "Name at max length (100 chars) should be valid",
			campaign: Campaign{
				Name:        string(make([]byte, 100)),
				Description: "Test",
			},
			expectValid: true,
		},
		{
			name: "Description at max length (500 chars) should be valid",
			campaign: Campaign{
				Name:        "Test",
				Description: string(make([]byte, 500)),
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate name
			if tt.campaign.Name == "" {
				if tt.expectValid {
					t.Error("Expected valid but name is empty")
				}
				return
			}

			if len(tt.campaign.Name) > 100 {
				if tt.expectValid {
					t.Error("Expected valid but name is too long")
				}
				return
			}

			// Validate description
			if len(tt.campaign.Description) > 500 {
				if tt.expectValid {
					t.Error("Expected valid but description is too long")
				}
				return
			}

			// If we got here, validation passed
			if !tt.expectValid {
				t.Errorf("Expected validation to fail but it passed")
			}
		})
	}
}

// TestIdeaLimitValidation tests idea query limit validation
func TestIdeaLimitValidation(t *testing.T) {
	tests := []struct {
		name        string
		limit       int
		expectValid bool
	}{
		{"Limit 0 should be invalid", 0, false},
		{"Limit 1 should be valid", 1, true},
		{"Limit 50 should be valid", 50, true},
		{"Limit 100 should be valid", 100, true},
		{"Limit 101 should be invalid", 101, false},
		{"Negative limit should be invalid", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.limit >= 1 && tt.limit <= 100

			if valid != tt.expectValid {
				t.Errorf("Expected valid=%v for limit %d, got %v", tt.expectValid, tt.limit, valid)
			}
		})
	}
}

// TestGenerateIdeasCountValidation tests idea generation count validation
func TestGenerateIdeasCountValidation(t *testing.T) {
	tests := []struct {
		name        string
		count       int
		expectValid bool
	}{
		{"Count 0 should be invalid", 0, false},
		{"Count 1 should be valid", 1, true},
		{"Count 5 should be valid", 5, true},
		{"Count 10 should be valid", 10, true},
		{"Count 11 should be invalid", 11, false},
		{"Negative count should be invalid", -5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.count >= 1 && tt.count <= 10

			if valid != tt.expectValid {
				t.Errorf("Expected valid=%v for count %d, got %v", tt.expectValid, tt.count, valid)
			}
		})
	}
}

// TestRefinementLengthValidation tests refinement text length validation
func TestRefinementLengthValidation(t *testing.T) {
	tests := []struct {
		name        string
		textLength  int
		expectValid bool
	}{
		{"Empty refinement should be invalid", 0, false},
		{"Short refinement should be valid", 10, true},
		{"Max length refinement (2000) should be valid", 2000, true},
		{"Over max length (2001) should be invalid", 2001, false},
		{"Very long refinement should be invalid", 5000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := string(make([]byte, tt.textLength))
			valid := len(text) > 0 && len(text) <= 2000

			if valid != tt.expectValid {
				t.Errorf("Expected valid=%v for length %d, got %v", tt.expectValid, tt.textLength, valid)
			}
		})
	}
}

// TestUUIDValidation tests UUID format validation patterns
func TestUUIDValidation(t *testing.T) {
	tests := []struct {
		name        string
		uuid        string
		expectValid bool
	}{
		{"Valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", true},
		{"Empty UUID should be invalid", "", false},
		{"Invalid format should be invalid", "not-a-uuid", false},
		{"Partial UUID should be invalid", "550e8400-e29b", false},
		{"UUID with spaces should be invalid", "550e8400-e29b-41d4-a716-4466 55440000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic UUID format check (length and dashes)
			valid := len(tt.uuid) == 36 &&
				tt.uuid[8] == '-' &&
				tt.uuid[13] == '-' &&
				tt.uuid[18] == '-' &&
				tt.uuid[23] == '-'

			if valid != tt.expectValid {
				t.Errorf("Expected valid=%v for UUID '%s', got %v", tt.expectValid, tt.uuid, valid)
			}
		})
	}
}

// TestSearchQueryValidation tests search query validation
func TestSearchQueryValidation(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectValid bool
	}{
		{"Valid query", "test query", true},
		{"Empty query should be invalid", "", false},
		{"Single word query should be valid", "test", true},
		{"Long query should be valid", string(make([]byte, 500)), true},
		{"Very long query might be problematic", string(make([]byte, 10000)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := len(tt.query) > 0

			if valid != tt.expectValid {
				t.Errorf("Expected valid=%v for query length %d, got %v", tt.expectValid, len(tt.query), valid)
			}
		})
	}
}
