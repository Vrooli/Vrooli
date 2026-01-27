package dns

import "testing"

func TestBaseDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Apex domain handling
		{
			name:     "apex domain unchanged",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "apex with different TLD",
			input:    "example.org",
			expected: "example.org",
		},
		{
			name:     "apex with country code TLD",
			input:    "example.co.uk",
			expected: "example.co.uk",
		},

		// WWW prefix handling
		{
			name:     "www collapses to apex",
			input:    "www.example.com",
			expected: "example.com",
		},
		{
			name:     "www with different TLD",
			input:    "www.example.org",
			expected: "example.org",
		},
		{
			name:     "www with country code TLD",
			input:    "www.example.co.uk",
			expected: "example.co.uk",
		},

		// Subdomain preservation
		{
			name:     "app subdomain preserved",
			input:    "app.example.com",
			expected: "app.example.com",
		},
		{
			name:     "api subdomain preserved",
			input:    "api.example.com",
			expected: "api.example.com",
		},
		{
			name:     "multi-level subdomain preserved",
			input:    "dev.api.example.com",
			expected: "dev.api.example.com",
		},
		{
			name:     "staging subdomain preserved",
			input:    "staging.example.com",
			expected: "staging.example.com",
		},

		// Origin subdomain (common pattern)
		{
			name:     "do-origin subdomain preserved",
			input:    "do-origin.example.com",
			expected: "do-origin.example.com",
		},
		{
			name:     "origin subdomain preserved",
			input:    "origin.example.com",
			expected: "origin.example.com",
		},

		// Edge cases
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single label (localhost)",
			input:    "localhost",
			expected: "localhost",
		},
		{
			name:     "IP address passthrough",
			input:    "203.0.113.10",
			expected: "203.0.113.10",
		},

		// Case normalization (if applicable)
		{
			name:     "uppercase domain",
			input:    "EXAMPLE.COM",
			expected: "example.com",
		},
		{
			name:     "mixed case www",
			input:    "WWW.Example.COM",
			expected: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BaseDomain(tt.input)
			if got != tt.expected {
				t.Errorf("BaseDomain(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBaseDomainIdempotent(t *testing.T) {
	// Calling BaseDomain twice should produce the same result
	domains := []string{
		"example.com",
		"www.example.com",
		"app.example.com",
		"api.staging.example.com",
	}

	for _, domain := range domains {
		t.Run(domain, func(t *testing.T) {
			first := BaseDomain(domain)
			second := BaseDomain(first)
			if first != second {
				t.Errorf("BaseDomain not idempotent: BaseDomain(%q) = %q, BaseDomain(%q) = %q",
					domain, first, first, second)
			}
		})
	}
}

func TestBaseDomainCloudflarePatterns(t *testing.T) {
	// Common Cloudflare + VPS deployment patterns
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "proxied apex",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "proxied www",
			input:    "www.example.com",
			expected: "example.com",
		},
		{
			name:     "origin bypass subdomain",
			input:    "do-origin.example.com",
			expected: "do-origin.example.com",
		},
		{
			name:     "direct origin subdomain",
			input:    "direct.example.com",
			expected: "direct.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BaseDomain(tt.input)
			if got != tt.expected {
				t.Errorf("BaseDomain(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
