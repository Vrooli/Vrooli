package uismoke

import (
	"testing"
)

func TestParseArgs_ScenarioOnly(t *testing.T) {
	args := []string{"demo"}
	parsed, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if parsed.Scenario != "demo" {
		t.Errorf("Scenario = %q, want %q", parsed.Scenario, "demo")
	}
	if parsed.URL != "" {
		t.Errorf("URL = %q, want empty", parsed.URL)
	}
	if parsed.BrowserlessURL != "" {
		t.Errorf("BrowserlessURL = %q, want empty", parsed.BrowserlessURL)
	}
	if parsed.TimeoutMs != 0 {
		t.Errorf("TimeoutMs = %d, want 0", parsed.TimeoutMs)
	}
	if parsed.JSON {
		t.Error("JSON should be false by default")
	}
}

func TestParseArgs_WithURL(t *testing.T) {
	args := []string{"demo", "--url", "http://localhost:8080"}
	parsed, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if parsed.Scenario != "demo" {
		t.Errorf("Scenario = %q, want %q", parsed.Scenario, "demo")
	}
	if parsed.URL != "http://localhost:8080" {
		t.Errorf("URL = %q, want %q", parsed.URL, "http://localhost:8080")
	}
}

func TestParseArgs_WithBrowserlessURL(t *testing.T) {
	args := []string{"demo", "--browserless", "http://custom-browserless:7777"}
	parsed, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if parsed.BrowserlessURL != "http://custom-browserless:7777" {
		t.Errorf("BrowserlessURL = %q, want %q", parsed.BrowserlessURL, "http://custom-browserless:7777")
	}
}

func TestParseArgs_WithTimeout(t *testing.T) {
	args := []string{"demo", "--timeout", "120000"}
	parsed, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if parsed.TimeoutMs != 120000 {
		t.Errorf("TimeoutMs = %d, want %d", parsed.TimeoutMs, 120000)
	}
}

func TestParseArgs_WithJSON(t *testing.T) {
	args := []string{"demo", "--json"}
	parsed, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if !parsed.JSON {
		t.Error("JSON should be true")
	}
}

func TestParseArgs_AllOptions(t *testing.T) {
	args := []string{
		"my-scenario",
		"--url", "http://example.com:3000",
		"--browserless", "http://browserless:4110",
		"--timeout", "180000",
		"--json",
	}
	parsed, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if parsed.Scenario != "my-scenario" {
		t.Errorf("Scenario = %q, want %q", parsed.Scenario, "my-scenario")
	}
	if parsed.URL != "http://example.com:3000" {
		t.Errorf("URL = %q, want %q", parsed.URL, "http://example.com:3000")
	}
	if parsed.BrowserlessURL != "http://browserless:4110" {
		t.Errorf("BrowserlessURL = %q, want %q", parsed.BrowserlessURL, "http://browserless:4110")
	}
	if parsed.TimeoutMs != 180000 {
		t.Errorf("TimeoutMs = %d, want %d", parsed.TimeoutMs, 180000)
	}
	if !parsed.JSON {
		t.Error("JSON should be true")
	}
}

func TestParseArgs_NoArgs(t *testing.T) {
	args := []string{}
	_, err := ParseArgs(args)
	if err == nil {
		t.Error("ParseArgs() expected error for empty args")
	}
}

func TestParseArgs_InvalidFlag(t *testing.T) {
	args := []string{"demo", "--invalid-flag"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Error("ParseArgs() expected error for invalid flag")
	}
}

func TestParseArgs_InvalidTimeout(t *testing.T) {
	args := []string{"demo", "--timeout", "not-a-number"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Error("ParseArgs() expected error for invalid timeout")
	}
}

func TestParseArgs_ShortFormNotSupported(t *testing.T) {
	// Verify short forms aren't accidentally enabled
	args := []string{"demo", "-u", "http://localhost:8080"}
	_, err := ParseArgs(args)
	// This should error because -u is not defined
	if err == nil {
		t.Error("ParseArgs() expected error for short flag -u")
	}
}

func TestStatusEmoji(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"passed", "✅"},
		{"failed", "❌"},
		{"skipped", "⏭️"},
		{"blocked", "🚫"},
		{"unknown", "❓"},
		{"", "❓"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := statusEmoji(tt.status)
			if got != tt.want {
				t.Errorf("statusEmoji(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestUsageError(t *testing.T) {
	err := usageError("test message")
	if err == nil {
		t.Error("usageError() should return non-nil error")
	}
	if err.Error() != "test message" {
		t.Errorf("Error message = %q, want %q", err.Error(), "test message")
	}
}

func TestRequest_ZeroValues(t *testing.T) {
	// Verify zero values are what we expect
	req := Request{}
	if req.URL != "" {
		t.Errorf("URL zero value = %q, want empty", req.URL)
	}
	if req.BrowserlessURL != "" {
		t.Errorf("BrowserlessURL zero value = %q, want empty", req.BrowserlessURL)
	}
	if req.TimeoutMs != 0 {
		t.Errorf("TimeoutMs zero value = %d, want 0", req.TimeoutMs)
	}
}

func TestArgs_ZeroValues(t *testing.T) {
	// Verify zero values are what we expect
	args := Args{}
	if args.Scenario != "" {
		t.Errorf("Scenario zero value = %q, want empty", args.Scenario)
	}
	if args.URL != "" {
		t.Errorf("URL zero value = %q, want empty", args.URL)
	}
	if args.BrowserlessURL != "" {
		t.Errorf("BrowserlessURL zero value = %q, want empty", args.BrowserlessURL)
	}
	if args.TimeoutMs != 0 {
		t.Errorf("TimeoutMs zero value = %d, want 0", args.TimeoutMs)
	}
	if args.JSON {
		t.Error("JSON zero value should be false")
	}
}
