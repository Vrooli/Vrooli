package adapter

import (
	"testing"
)

// [REQ:RM-001] Helper function tests

func TestExtractHostname(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://app.example.com", "app.example.com"},
		{"http://app.example.com/path", "app.example.com"},
		{"https://app.example.com/", "app.example.com"},
		{"app.example.com", "app.example.com"},
	}
	for _, tt := range tests {
		got := ExtractHostname(tt.input)
		if got != tt.want {
			t.Errorf("ExtractHostname(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
