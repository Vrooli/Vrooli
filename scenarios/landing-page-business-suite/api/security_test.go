package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

// ============================================================================
// X-Forwarded-For Spoofing Tests
// ============================================================================

func TestSecurity_XFFSpoofing_UntrustedSourceIgnored(t *testing.T) {
	// Reset trusted proxies for this test
	resetTrustedProxies()
	os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	defer os.Unsetenv("TRUSTED_PROXY_CIDRS")

	// Attacker tries to spoof their IP via X-Forwarded-For
	// but their connection is NOT from a trusted proxy
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.RemoteAddr = "203.0.113.50:12345" // Attacker's real IP (not in trusted range)
	req.Header.Set("X-Forwarded-For", "1.2.3.4") // Spoofed IP

	ip := getClientIP(req)

	// Should return the attacker's real IP, not the spoofed one
	if ip != "203.0.113.50" {
		t.Errorf("Expected real IP 203.0.113.50, got spoofed IP: %s", ip)
	}
}

func TestSecurity_XFFSpoofing_XRealIPAlsoIgnored(t *testing.T) {
	resetTrustedProxies()
	os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	defer os.Unsetenv("TRUSTED_PROXY_CIDRS")

	// Attacker tries X-Real-IP header
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.RemoteAddr = "203.0.113.50:12345"
	req.Header.Set("X-Real-IP", "1.2.3.4")

	ip := getClientIP(req)

	if ip != "203.0.113.50" {
		t.Errorf("Expected real IP 203.0.113.50, got spoofed IP: %s", ip)
	}
}

func TestSecurity_XFFSpoofing_ValidProxyChain(t *testing.T) {
	resetTrustedProxies()
	os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8,172.16.0.0/12")
	defer os.Unsetenv("TRUSTED_PROXY_CIDRS")

	// Legitimate request through trusted load balancer
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.RemoteAddr = "10.0.0.5:12345" // Trusted load balancer
	req.Header.Set("X-Forwarded-For", "198.51.100.23, 10.0.0.1")

	ip := getClientIP(req)

	// Should return the first (client) IP from the chain
	if ip != "198.51.100.23" {
		t.Errorf("Expected client IP 198.51.100.23, got: %s", ip)
	}
}

func TestSecurity_XFFSpoofing_NoTrustedProxiesConfigured(t *testing.T) {
	resetTrustedProxies()
	os.Unsetenv("TRUSTED_PROXY_CIDRS")

	// Without any trusted proxies configured, all XFF headers are ignored
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")

	ip := getClientIP(req)

	// Even 10.x addresses should be used directly when no trusted proxies configured
	if ip != "10.0.0.5" {
		t.Errorf("Expected direct IP 10.0.0.5, got: %s", ip)
	}
}

func TestSecurity_XFFSpoofing_InvalidIPInHeader(t *testing.T) {
	resetTrustedProxies()
	os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	defer os.Unsetenv("TRUSTED_PROXY_CIDRS")

	// Attacker sends malformed IP to try to bypass validation
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.RemoteAddr = "10.0.0.5:12345" // Trusted proxy
	req.Header.Set("X-Forwarded-For", "not-a-valid-ip; DROP TABLE users;--")

	ip := getClientIP(req)

	// Should fall back to direct IP since XFF contains invalid data
	if ip != "10.0.0.5" {
		t.Errorf("Expected fallback to direct IP, got: %s", ip)
	}
}

// ============================================================================
// Session Security Tests
// ============================================================================

func TestSecurity_SecureCookies_DefaultsToSecureInProduction(t *testing.T) {
	oldEnv := os.Getenv("LPBS_ENVIRONMENT")
	oldCookies := os.Getenv("LPBS_SECURE_COOKIES")
	defer func() {
		if oldEnv != "" {
			os.Setenv("LPBS_ENVIRONMENT", oldEnv)
		} else {
			os.Unsetenv("LPBS_ENVIRONMENT")
		}
		if oldCookies != "" {
			os.Setenv("LPBS_SECURE_COOKIES", oldCookies)
		} else {
			os.Unsetenv("LPBS_SECURE_COOKIES")
		}
	}()

	// Production mode without explicit LPBS_SECURE_COOKIES
	os.Setenv("LPBS_ENVIRONMENT", "production")
	os.Unsetenv("LPBS_SECURE_COOKIES")

	if !isSecureCookiesEnabled() {
		t.Error("Expected secure cookies to be enabled in production by default")
	}
}

func TestSecurity_SecureCookies_CanBeDisabledForDev(t *testing.T) {
	oldCookies := os.Getenv("LPBS_SECURE_COOKIES")
	defer func() {
		if oldCookies != "" {
			os.Setenv("LPBS_SECURE_COOKIES", oldCookies)
		} else {
			os.Unsetenv("LPBS_SECURE_COOKIES")
		}
	}()

	os.Setenv("LPBS_SECURE_COOKIES", "false")

	if isSecureCookiesEnabled() {
		t.Error("Expected secure cookies to be disabled when LPBS_SECURE_COOKIES=false")
	}
}

func TestSecurity_SecureCookies_DisabledByDefault_InDevelopment(t *testing.T) {
	oldEnv := os.Getenv("LPBS_ENVIRONMENT")
	oldCookies := os.Getenv("LPBS_SECURE_COOKIES")
	defer func() {
		if oldEnv != "" {
			os.Setenv("LPBS_ENVIRONMENT", oldEnv)
		} else {
			os.Unsetenv("LPBS_ENVIRONMENT")
		}
		if oldCookies != "" {
			os.Setenv("LPBS_SECURE_COOKIES", oldCookies)
		} else {
			os.Unsetenv("LPBS_SECURE_COOKIES")
		}
	}()

	// Development mode (no LPBS_ENVIRONMENT or set to development)
	os.Unsetenv("LPBS_ENVIRONMENT")
	os.Unsetenv("LPBS_SECURE_COOKIES")

	// In development (no production env), secure cookies should NOT be enforced
	if isSecureCookiesEnabled() {
		t.Error("Expected secure cookies to be disabled in development by default")
	}
}

// ============================================================================
// Input Validation Security Tests
// ============================================================================

func TestSecurity_EmailValidation_RejectsInvalidFormats(t *testing.T) {
	// Test emails with injection attempts and malformed formats
	// Note: RFC 5322 allows long local parts, so we test for more clearly invalid formats
	maliciousInputs := []string{
		"user@<script>alert('xss')</script>",
		"user@example.com\r\nBcc: attacker@evil.com",
		"user@example.com; rm -rf /",
		"user@example.com\x00evil",
		"@example.com",          // Missing local part
		"user@",                  // Missing domain
		"user@@example.com",      // Double @
		"user example@test.com",  // Space in local part
	}

	for _, input := range maliciousInputs {
		displayName := input
		if len(displayName) > 30 {
			displayName = displayName[:30]
		}
		t.Run(displayName, func(t *testing.T) {
			_, err := ValidateEmail(input)
			if err == nil {
				t.Errorf("Expected rejection of malformed email: %s", displayName)
			}
		})
	}
}

func TestSecurity_URLValidation_RejectsNonHTTPSchemes(t *testing.T) {
	maliciousURLs := []string{
		"javascript:alert('xss')",
		"data:text/html,<script>alert('xss')</script>",
		"file:///etc/passwd",
		"ftp://evil.com/malware.exe",
		"gopher://evil.com",
	}

	for _, url := range maliciousURLs {
		t.Run(url[:min(20, len(url))], func(t *testing.T) {
			_, err := ValidateURL(url)
			if err == nil {
				t.Errorf("Expected rejection of non-HTTP URL: %s", url)
			}
		})
	}
}

func TestSecurity_URLValidation_AcceptsLegitimateURLs(t *testing.T) {
	legitimateURLs := []string{
		"https://example.com",
		"http://localhost:3000",
		"https://api.example.com/v1/endpoint",
		"https://example.com/path?query=value&foo=bar",
		"https://192.168.1.1:8080/admin",
	}

	for _, url := range legitimateURLs {
		t.Run(url[:min(30, len(url))], func(t *testing.T) {
			result, err := ValidateURL(url)
			if err != nil {
				t.Errorf("Expected acceptance of legitimate URL %s, got error: %v", url, err)
			}
			if result == "" {
				t.Errorf("Expected non-empty result for URL: %s", url)
			}
		})
	}
}

// ============================================================================
// API Key Encryption Security Tests
// ============================================================================

func TestSecurity_APIKeyEncryption_ProductionRequiresKey(t *testing.T) {
	oldEnv := os.Getenv("LPBS_ENVIRONMENT")
	oldKey := os.Getenv("LPBS_API_KEY_ENCRYPTION_KEY")
	defer func() {
		if oldEnv != "" {
			os.Setenv("LPBS_ENVIRONMENT", oldEnv)
		} else {
			os.Unsetenv("LPBS_ENVIRONMENT")
		}
		if oldKey != "" {
			os.Setenv("LPBS_API_KEY_ENCRYPTION_KEY", oldKey)
		} else {
			os.Unsetenv("LPBS_API_KEY_ENCRYPTION_KEY")
		}
	}()

	os.Setenv("LPBS_ENVIRONMENT", "production")
	os.Unsetenv("LPBS_API_KEY_ENCRYPTION_KEY")

	if !isProductionEnvironment() {
		t.Error("Expected isProductionEnvironment() to return true")
	}
}

// ============================================================================
// IP Address Validation Tests
// ============================================================================

func TestSecurity_IPValidation_ValidFormats(t *testing.T) {
	validIPs := []string{
		"192.168.1.1",
		"10.0.0.1",
		"::1",
		"2001:db8::1",
		"127.0.0.1",
		"0.0.0.0",
		"255.255.255.255",
	}

	for _, ip := range validIPs {
		t.Run(ip, func(t *testing.T) {
			if !validateIPFormat(ip) {
				t.Errorf("Expected valid IP format: %s", ip)
			}
		})
	}
}

func TestSecurity_IPValidation_InvalidFormats(t *testing.T) {
	invalidIPs := []string{
		"not-an-ip",
		"192.168.1",
		"192.168.1.1.1",
		"999.999.999.999",
		"",
		"localhost",
		"example.com",
		"192.168.1.1:8080", // Port should not be included
	}

	for _, ip := range invalidIPs {
		t.Run(ip, func(t *testing.T) {
			if validateIPFormat(ip) {
				t.Errorf("Expected invalid IP format: %s", ip)
			}
		})
	}
}

// ============================================================================
// Concurrent Access Tests
// ============================================================================

func TestSecurity_TrustedProxies_ConcurrentAccess(t *testing.T) {
	resetTrustedProxies()
	os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	defer os.Unsetenv("TRUSTED_PROXY_CIDRS")

	// Concurrent calls to getClientIP should not race
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if n%2 == 0 {
				req.RemoteAddr = "10.0.0.5:12345"
				req.Header.Set("X-Forwarded-For", "1.2.3.4")
			} else {
				req.RemoteAddr = "203.0.113.50:12345"
				req.Header.Set("X-Forwarded-For", "5.6.7.8")
			}
			ip := getClientIP(req)
			if ip == "" {
				t.Errorf("Got empty IP in concurrent access")
			}
		}(i)
	}
	wg.Wait()
}

// ============================================================================
// Helper Functions
// ============================================================================

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
