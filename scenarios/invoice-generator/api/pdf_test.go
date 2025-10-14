package main

import (
	"testing"
)

// TestFormatAddress tests address formatting logic
func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name         string
		client       *Client
		wantContains []string
	}{
		{
			name: "full address",
			client: &Client{
				Name:          "Test Client",
				AddressLine1:  "123 Main St",
				AddressLine2:  "Suite 100",
				City:          "San Francisco",
				StateProvince: "CA",
				PostalCode:    "94102",
				Country:       "USA",
			},
			wantContains: []string{"123 Main St", "San Francisco"},
		},
		{
			name: "minimal address",
			client: &Client{
				Name:          "Minimal Client",
				AddressLine1:  "456 Oak Ave",
				City:          "Portland",
				StateProvince: "OR",
			},
			wantContains: []string{"456 Oak Ave", "Portland"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAddress(tt.client)

			if result == "" && len(tt.wantContains) > 0 {
				t.Error("formatAddress() returned empty string")
			}

			for _, want := range tt.wantContains {
				if !containsSubstring(result, want) {
					t.Errorf("formatAddress() missing expected content: %v in result: %v", want, result)
				}
			}
		})
	}
}

// TestFormatCompanyAddress tests company address formatting
func TestFormatCompanyAddress(t *testing.T) {
	tests := []struct {
		name         string
		company      *Company
		wantContains []string
	}{
		{
			name: "complete company info",
			company: &Company{
				Name:          "Test Company Inc",
				AddressLine1:  "789 Business Blvd",
				City:          "New York",
				StateProvince: "NY",
				PostalCode:    "10001",
			},
			wantContains: []string{"789 Business Blvd", "New York"},
		},
		{
			name: "minimal company info",
			company: &Company{
				Name: "Minimal Corp",
			},
			wantContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCompanyAddress(tt.company)

			for _, want := range tt.wantContains {
				if !containsSubstring(result, want) {
					t.Errorf("formatCompanyAddress() missing expected content: %v", want)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
