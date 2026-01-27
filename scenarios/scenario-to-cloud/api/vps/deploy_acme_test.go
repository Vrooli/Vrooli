package vps

import (
	"strings"
	"testing"
)

func TestCaddyACMEOriginUnreachableHintEmpty(t *testing.T) {
	tests := []struct {
		name            string
		logs            string
		dns01Configured bool
	}{
		{
			name:            "empty logs",
			logs:            "",
			dns01Configured: false,
		},
		{
			name:            "whitespace only",
			logs:            "   \t\n  ",
			dns01Configured: false,
		},
		{
			name:            "whitespace with dns01 configured",
			logs:            "   ",
			dns01Configured: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := caddyACMEOriginUnreachableHint(tt.logs, tt.dns01Configured)
			if hint != "" {
				t.Errorf("expected empty hint for empty/whitespace logs, got %q", hint)
			}
		})
	}
}

func TestCaddyACMEOriginUnreachableHintNoACMEVariants(t *testing.T) {
	// Logs that don't contain "acme" should return empty hint
	tests := []struct {
		name string
		logs string
	}{
		{
			name: "regular nginx log",
			logs: "2025/01/01 00:00:00 [error] upstream connection refused",
		},
		{
			name: "generic server error",
			logs: "Error: connection timeout to backend",
		},
		{
			name: "522 without acme context",
			logs: "HTTP/1.1 522 Connection Timed Out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := caddyACMEOriginUnreachableHint(tt.logs, false)
			if hint != "" {
				t.Errorf("expected empty hint for non-ACME logs, got %q", hint)
			}
		})
	}
}

func TestCaddyACMEOriginUnreachableHintDNS01Missing(t *testing.T) {
	tests := []struct {
		name string
		logs string
	}{
		{
			name: "basic dns-01 remaining",
			logs: "2025/01/01 00:00:00 acme: remaining=[dns-01] no solvers available",
		},
		{
			name: "dns-01 with additional context",
			logs: `2025/01/01 00:00:00 [INFO] Starting Caddy
2025/01/01 00:00:01 acme: challenge failed, remaining=[dns-01] no solvers available for this challenge type
2025/01/01 00:00:02 [ERROR] Certificate request failed`,
		},
		{
			name: "uppercase ACME",
			logs: "ACME: remaining=[dns-01] no solvers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := caddyACMEOriginUnreachableHint(tt.logs, false)
			if hint == "" {
				t.Fatal("expected hint for dns-01 missing")
			}
			if !strings.Contains(hint, "DNS-01") {
				t.Errorf("expected hint to mention DNS-01, got %q", hint)
			}
			if !strings.Contains(hint, "CLOUDFLARE_API_TOKEN") {
				t.Errorf("expected hint to mention CLOUDFLARE_API_TOKEN, got %q", hint)
			}
		})
	}
}

func TestCaddyACMEOriginUnreachableHintDNS01Configured(t *testing.T) {
	// When DNS-01 is already configured, should not suggest configuring it again
	logs := "2025/01/01 00:00:00 acme: remaining=[dns-01] no solvers available"
	hint := caddyACMEOriginUnreachableHint(logs, true)

	// With dns01Configured=true, this shouldn't trigger the DNS-01 hint
	if strings.Contains(hint, "CLOUDFLARE_API_TOKEN") {
		t.Errorf("should not suggest CLOUDFLARE_API_TOKEN when dns01 already configured, got %q", hint)
	}
}

func TestCaddyACMEOriginUnreachableHintOriginBlocked(t *testing.T) {
	tests := []struct {
		name            string
		logs            string
		dns01Configured bool
		wantKeywords    []string
	}{
		{
			name:            "522 origin unreachable",
			logs:            "acme: error: challenge failed: 522 origin unreachable",
			dns01Configured: true,
			wantKeywords:    []string{"origin unreachable", "Open inbound 80/443"},
		},
		{
			name:            "403 forbidden",
			logs:            "acme: challenge verification returned 403 Forbidden",
			dns01Configured: false,
			wantKeywords:    []string{"80/443"},
		},
		{
			name: "multiline with 522",
			logs: `2025/01/01 00:00:00 [INFO] Attempting ACME challenge
2025/01/01 00:00:05 acme: challenge failed with status 522
2025/01/01 00:00:05 [ERROR] Giving up`,
			dns01Configured: false,
			wantKeywords:    []string{"522"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := caddyACMEOriginUnreachableHint(tt.logs, tt.dns01Configured)
			if hint == "" {
				t.Fatal("expected hint for origin unreachable")
			}
			for _, keyword := range tt.wantKeywords {
				if !strings.Contains(hint, keyword) {
					t.Errorf("expected hint to contain %q, got %q", keyword, hint)
				}
			}
		})
	}
}

func TestCaddyACMEOriginUnreachableHintIncludesLogLines(t *testing.T) {
	// Verify that relevant log lines are included in the hint for debugging
	logs := `2025/01/01 00:00:00 [INFO] Caddy starting
2025/01/01 00:00:01 acme: challenge verification returned 522 origin unreachable
2025/01/01 00:00:02 [ERROR] Certificate request failed`

	hint := caddyACMEOriginUnreachableHint(logs, false)

	if !strings.Contains(hint, "Recent Caddy logs") {
		t.Errorf("expected hint to include recent Caddy logs reference, got %q", hint)
	}
	// The log line containing the error should be included
	if !strings.Contains(hint, "522") {
		t.Errorf("expected hint to include the 522 error context, got %q", hint)
	}
}

func TestCaddyACMEOriginUnreachableHintLimitsLogLines(t *testing.T) {
	// Verify that at most 3 log lines are included
	var logs strings.Builder
	for i := 0; i < 10; i++ {
		logs.WriteString("acme: error 522 line ")
		logs.WriteString(string(rune('A' + i)))
		logs.WriteString("\n")
	}

	hint := caddyACMEOriginUnreachableHint(logs.String(), false)

	// Count occurrences of separator
	separatorCount := strings.Count(hint, " | ")
	// With 3 log lines, there should be at most 2 separators
	if separatorCount > 2 {
		t.Errorf("expected at most 2 separators for 3 log lines, got %d", separatorCount)
	}
}

func containsAll(value string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}
